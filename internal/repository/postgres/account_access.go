package postgres

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sfedu-crm/internal/domain"
	"github.com/sfedu-crm/internal/repository"
)

type accountAccessRepo struct{ db *pgxpool.Pool }

func NewAccountAccessRepository(db *pgxpool.Pool) repository.AccountAccessRepository {
	return &accountAccessRepo{db: db}
}

func (r *accountAccessRepo) CreatePendingUser(ctx context.Context, input domain.CreateUserInput) (*domain.User, error) {
	q := `INSERT INTO users (full_name, phone, email, password_hash, role, gender, is_active, password_setup_required)
		VALUES ($1,$2,$3,$4,$5,$6,false,true) RETURNING ` + userColumns
	u, err := scanUserRow(queryer(ctx, r.db).QueryRow(ctx, q,
		input.FullName, input.Phone, strings.ToLower(input.Email), input.Password, input.Role, input.Gender,
	))
	if err != nil {
		return nil, fmt.Errorf("accountAccessRepo.CreatePendingUser: %w", err)
	}
	return u, nil
}

func (r *accountAccessRepo) CreateToken(ctx context.Context, userID int64, purpose domain.AccountTokenPurpose, tokenHash string, expiresAt time.Time) error {
	_, err := queryer(ctx, r.db).Exec(ctx, `INSERT INTO account_tokens(user_id,purpose,token_hash,expires_at) VALUES($1,$2,$3,$4)`,
		userID, purpose, tokenHash, expiresAt)
	if err != nil {
		return fmt.Errorf("accountAccessRepo.CreateToken: %w", mapDBError(err))
	}
	return nil
}

func (r *accountAccessRepo) GetTokenForUpdate(ctx context.Context, tokenHash string, purpose domain.AccountTokenPurpose) (*domain.AccountToken, error) {
	// Serialize token consumption by account before locking individual tokens.
	// This also prevents deadlocks when two invitations are used concurrently.
	var userID int64
	if err := queryer(ctx, r.db).QueryRow(ctx, `SELECT u.id FROM users u JOIN account_tokens t ON t.user_id=u.id WHERE t.token_hash=$1 AND t.purpose=$2 FOR UPDATE OF u`, tokenHash, purpose).Scan(&userID); err != nil {
		return nil, mapDBError(err)
	}
	t := &domain.AccountToken{}
	err := queryer(ctx, r.db).QueryRow(ctx, `SELECT id,user_id,purpose,expires_at,used_at,created_at
		FROM account_tokens WHERE token_hash=$1 AND purpose=$2 FOR UPDATE`, tokenHash, purpose).
		Scan(&t.ID, &t.UserID, &t.Purpose, &t.ExpiresAt, &t.UsedAt, &t.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("accountAccessRepo.GetTokenForUpdate: %w", mapDBError(err))
	}
	return t, nil
}

func (r *accountAccessRepo) MarkTokenUsed(ctx context.Context, tokenID int64) error {
	tag, err := queryer(ctx, r.db).Exec(ctx, `UPDATE account_tokens SET used_at=NOW() WHERE id=$1 AND used_at IS NULL`, tokenID)
	if err != nil {
		return fmt.Errorf("accountAccessRepo.MarkTokenUsed: %w", mapDBError(err))
	}
	return requireAffected(tag)
}

func (r *accountAccessRepo) InvalidateTokens(ctx context.Context, userID int64, purpose domain.AccountTokenPurpose) error {
	_, err := queryer(ctx, r.db).Exec(ctx, `UPDATE account_tokens SET used_at=COALESCE(used_at,NOW()) WHERE user_id=$1 AND purpose=$2 AND used_at IS NULL`, userID, purpose)
	if err != nil {
		return fmt.Errorf("accountAccessRepo.InvalidateTokens: %w", mapDBError(err))
	}
	return nil
}

func (r *accountAccessRepo) ActivateUser(ctx context.Context, userID int64, passwordHash string) error {
	tag, err := queryer(ctx, r.db).Exec(ctx, `UPDATE users
		SET password_hash=$1,is_active=true,password_setup_required=false,token_version=token_version+1,updated_at=NOW()
		WHERE id=$2 AND password_setup_required=true`, passwordHash, userID)
	if err != nil {
		return fmt.Errorf("accountAccessRepo.ActivateUser: %w", mapDBError(err))
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("%w: account is already activated or unavailable", domain.ErrConflict)
	}
	return nil
}

func (r *accountAccessRepo) ResetPassword(ctx context.Context, userID int64, passwordHash string) error {
	tag, err := queryer(ctx, r.db).Exec(ctx, `UPDATE users
		SET password_hash=$1,token_version=token_version+1,updated_at=NOW()
		WHERE id=$2 AND password_setup_required=false`, passwordHash, userID)
	if err != nil {
		return fmt.Errorf("accountAccessRepo.ResetPassword: %w", mapDBError(err))
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("%w: account is not ready for password reset", domain.ErrConflict)
	}
	return nil
}

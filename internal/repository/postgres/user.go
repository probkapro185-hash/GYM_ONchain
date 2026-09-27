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

type userRepo struct{ db *pgxpool.Pool }

func NewUserRepository(db *pgxpool.Pool) repository.UserRepository { return &userRepo{db: db} }

const userColumns = `id, full_name, phone, email, password_hash, token_version, role, gender,
	balance, visits, is_active, password_setup_required, created_at, updated_at, last_visit_at`

func scanUserRow(row interface{ Scan(...any) error }) (*domain.User, error) {
	u := &domain.User{}
	err := row.Scan(
		&u.ID, &u.FullName, &u.Phone, &u.Email, &u.PasswordHash, &u.TokenVersion,
		&u.Role, &u.Gender, &u.Balance, &u.Visits, &u.IsActive, &u.PasswordSetupRequired,
		&u.CreatedAt, &u.UpdatedAt, &u.LastVisitAt,
	)
	if err != nil {
		return nil, mapDBError(err)
	}
	return u, nil
}

func (r *userRepo) Create(ctx context.Context, input domain.CreateUserInput) (*domain.User, error) {
	q := `INSERT INTO users (full_name, phone, email, password_hash, role, gender)
		VALUES ($1,$2,$3,$4,$5,$6) RETURNING ` + userColumns
	u, err := scanUserRow(queryer(ctx, r.db).QueryRow(ctx, q,
		input.FullName, input.Phone, strings.ToLower(input.Email), input.Password, input.Role, input.Gender,
	))
	if err != nil {
		return nil, fmt.Errorf("userRepo.Create: %w", err)
	}
	return u, nil
}

func (r *userRepo) GetByID(ctx context.Context, id int64) (*domain.User, error) {
	u, err := scanUserRow(queryer(ctx, r.db).QueryRow(ctx, `SELECT `+userColumns+` FROM users WHERE id=$1`, id))
	if err != nil {
		return nil, fmt.Errorf("userRepo.GetByID: %w", err)
	}
	return u, nil
}

func (r *userRepo) GetByIDForUpdate(ctx context.Context, id int64) (*domain.User, error) {
	u, err := scanUserRow(queryer(ctx, r.db).QueryRow(ctx, `SELECT `+userColumns+` FROM users WHERE id=$1 FOR UPDATE`, id))
	if err != nil {
		return nil, fmt.Errorf("userRepo.GetByIDForUpdate: %w", err)
	}
	return u, nil
}

func (r *userRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	u, err := scanUserRow(queryer(ctx, r.db).QueryRow(ctx,
		`SELECT `+userColumns+` FROM users WHERE lower(email)=lower($1)`, strings.TrimSpace(email)))
	if err != nil {
		return nil, fmt.Errorf("userRepo.GetByEmail: %w", err)
	}
	return u, nil
}

func (r *userRepo) GetByPhone(ctx context.Context, phone string) (*domain.User, error) {
	u, err := scanUserRow(queryer(ctx, r.db).QueryRow(ctx,
		`SELECT `+userColumns+` FROM users WHERE phone=$1`, strings.TrimSpace(phone)))
	if err != nil {
		return nil, fmt.Errorf("userRepo.GetByPhone: %w", err)
	}
	return u, nil
}

func (r *userRepo) List(ctx context.Context, filter repository.UserFilter) ([]*domain.User, error) {
	var conditions []string
	var args []any
	add := func(condition string, value any) {
		args = append(args, value)
		conditions = append(conditions, fmt.Sprintf(condition, len(args)))
	}
	if filter.Role != nil {
		add("u.role=$%d", *filter.Role)
	}
	if filter.IsActive != nil {
		add("u.is_active=$%d", *filter.IsActive)
	}
	if strings.TrimSpace(filter.Search) != "" {
		args = append(args, "%"+strings.TrimSpace(filter.Search)+"%")
		n := len(args)
		conditions = append(conditions, fmt.Sprintf("(u.full_name ILIKE $%d OR u.phone ILIKE $%d OR u.email ILIKE $%d)", n, n, n))
	}

	q := `SELECT u.id,u.full_name,u.phone,u.email,u.role,u.gender,
		u.balance,u.visits,u.is_active,u.password_setup_required,u.created_at,u.updated_at,u.last_visit_at,
		s.sessions_left
		FROM users u
		LEFT JOIN LATERAL (
			SELECT sessions_left
			FROM client_subscriptions
			WHERE client_id=u.id AND is_active=true AND frozen_at IS NULL AND start_date<=NOW() AND end_date>NOW()
			  AND (sessions_left IS NULL OR sessions_left>0)
			ORDER BY end_date ASC,id ASC LIMIT 1
		) s ON true`
	if len(conditions) > 0 {
		q += " WHERE " + strings.Join(conditions, " AND ")
	}
	q += " ORDER BY u.created_at DESC"

	rows, err := queryer(ctx, r.db).Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("userRepo.List: %w", err)
	}
	defer rows.Close()
	users := make([]*domain.User, 0)
	for rows.Next() {
		u := &domain.User{}
		if err := rows.Scan(
			&u.ID, &u.FullName, &u.Phone, &u.Email, &u.Role, &u.Gender,
			&u.Balance, &u.Visits, &u.IsActive, &u.PasswordSetupRequired, &u.CreatedAt, &u.UpdatedAt,
			&u.LastVisitAt, &u.SessionsLeft,
		); err != nil {
			return nil, fmt.Errorf("userRepo.List scan: %w", err)
		}
		users = append(users, u)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("userRepo.List rows: %w", err)
	}
	return users, nil
}

func (r *userRepo) Update(ctx context.Context, id int64, input domain.UpdateUserInput) (*domain.User, error) {
	q := `UPDATE users SET full_name=$1,phone=$2,email=$3,gender=$4,updated_at=NOW()
		WHERE id=$5 RETURNING ` + userColumns
	u, err := scanUserRow(queryer(ctx, r.db).QueryRow(ctx, q,
		input.FullName, input.Phone, strings.ToLower(input.Email), input.Gender, id,
	))
	if err != nil {
		return nil, fmt.Errorf("userRepo.Update: %w", err)
	}
	return u, nil
}

func (r *userRepo) UpdatePassword(ctx context.Context, id int64, passwordHash string) error {
	tag, err := queryer(ctx, r.db).Exec(ctx,
		`UPDATE users SET password_hash=$1,token_version=token_version+1,updated_at=NOW() WHERE id=$2`, passwordHash, id)
	if err != nil {
		return fmt.Errorf("userRepo.UpdatePassword: %w", mapDBError(err))
	}
	return requireAffected(tag)
}

func (r *userRepo) AddBalance(ctx context.Context, id int64, amountCents int64) error {
	tag, err := queryer(ctx, r.db).Exec(ctx,
		`UPDATE users SET balance=balance+($1::bigint/100.0),updated_at=NOW() WHERE id=$2`, amountCents, id)
	if err != nil {
		return fmt.Errorf("userRepo.AddBalance: %w", mapDBError(err))
	}
	return requireAffected(tag)
}

func (r *userRepo) DebitBalance(ctx context.Context, id int64, amountCents int64) error {
	tag, err := queryer(ctx, r.db).Exec(ctx,
		`UPDATE users SET balance=balance-($1::bigint/100.0),updated_at=NOW()
		 WHERE id=$2 AND balance >= ($1::bigint/100.0)`, amountCents, id)
	if err != nil {
		return fmt.Errorf("userRepo.DebitBalance: %w", mapDBError(err))
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrInsufficientFunds
	}
	return nil
}

func (r *userRepo) IncrementVisits(ctx context.Context, id int64, visitedAt time.Time) error {
	tag, err := queryer(ctx, r.db).Exec(ctx,
		`UPDATE users SET visits=visits+1,last_visit_at=GREATEST(last_visit_at,$2::timestamptz),updated_at=NOW() WHERE id=$1`, id, visitedAt)
	if err != nil {
		return fmt.Errorf("userRepo.IncrementVisits: %w", mapDBError(err))
	}
	return requireAffected(tag)
}

func (r *userRepo) SetActive(ctx context.Context, id int64, active bool) error {
	tag, err := queryer(ctx, r.db).Exec(ctx,
		`UPDATE users SET is_active=$1,token_version=token_version+1,updated_at=NOW() WHERE id=$2`, active, id)
	if err != nil {
		return fmt.Errorf("userRepo.SetActive: %w", mapDBError(err))
	}
	return requireAffected(tag)
}

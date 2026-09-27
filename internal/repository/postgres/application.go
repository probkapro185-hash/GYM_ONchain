package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sfedu-crm/internal/domain"
	"github.com/sfedu-crm/internal/repository"
)

type applicationRepo struct{ db *pgxpool.Pool }

func NewApplicationRepository(db *pgxpool.Pool) repository.ApplicationRepository {
	return &applicationRepo{db: db}
}

func scanApplication(row interface{ Scan(...any) error }) (*domain.ApplicationRequest, error) {
	a := &domain.ApplicationRequest{}
	if err := row.Scan(&a.ID, &a.FullName, &a.Phone, &a.Email, &a.Status, &a.CreatedAt, &a.Gender); err != nil {
		return nil, mapDBError(err)
	}
	return a, nil
}

func (r *applicationRepo) Create(ctx context.Context, input domain.CreateApplicationInput) (*domain.ApplicationRequest, error) {
	const q = `INSERT INTO application_requests(full_name,phone,email,status,gender)
		VALUES($1,$2,$3,'pending',$4)
		RETURNING id,full_name,phone,email,status,created_at,COALESCE(gender::text,'')`
	a, err := scanApplication(queryer(ctx, r.db).QueryRow(ctx, q,
		input.FullName, input.Phone, strings.ToLower(input.Email), input.Gender))
	if err != nil {
		return nil, fmt.Errorf("applicationRepo.Create: %w", err)
	}
	return a, nil
}

func (r *applicationRepo) GetByID(ctx context.Context, id int64) (*domain.ApplicationRequest, error) {
	a, err := scanApplication(queryer(ctx, r.db).QueryRow(ctx,
		`SELECT id,full_name,phone,email,status,created_at,COALESCE(gender::text,'') FROM application_requests WHERE id=$1`, id))
	if err != nil {
		return nil, fmt.Errorf("applicationRepo.GetByID: %w", err)
	}
	return a, nil
}

func (r *applicationRepo) GetByIDForUpdate(ctx context.Context, id int64) (*domain.ApplicationRequest, error) {
	a, err := scanApplication(queryer(ctx, r.db).QueryRow(ctx,
		`SELECT id,full_name,phone,email,status,created_at,COALESCE(gender::text,'') FROM application_requests WHERE id=$1 FOR UPDATE`, id))
	if err != nil {
		return nil, fmt.Errorf("applicationRepo.GetByIDForUpdate: %w", err)
	}
	return a, nil
}

func (r *applicationRepo) List(ctx context.Context, status domain.ApplicationStatus) ([]*domain.ApplicationRequest, error) {
	q := `SELECT id,full_name,phone,email,status,created_at,COALESCE(gender::text,'') FROM application_requests`
	var args []any
	if status != "" {
		q += ` WHERE status=$1`
		args = append(args, status)
	}
	q += ` ORDER BY created_at DESC`
	rows, err := queryer(ctx, r.db).Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("applicationRepo.List: %w", err)
	}
	defer rows.Close()
	result := make([]*domain.ApplicationRequest, 0)
	for rows.Next() {
		a := &domain.ApplicationRequest{}
		if err := rows.Scan(&a.ID, &a.FullName, &a.Phone, &a.Email, &a.Status, &a.CreatedAt, &a.Gender); err != nil {
			return nil, fmt.Errorf("applicationRepo.List scan: %w", err)
		}
		result = append(result, a)
	}
	return result, rows.Err()
}

func (r *applicationRepo) UpdateStatus(ctx context.Context, id int64, status domain.ApplicationStatus) error {
	tag, err := queryer(ctx, r.db).Exec(ctx, `UPDATE application_requests SET status=$1 WHERE id=$2`, status, id)
	if err != nil {
		return fmt.Errorf("applicationRepo.UpdateStatus: %w", mapDBError(err))
	}
	return requireAffected(tag)
}

func (r *applicationRepo) Delete(ctx context.Context, id int64) error {
	tag, err := queryer(ctx, r.db).Exec(ctx, `DELETE FROM application_requests WHERE id=$1`, id)
	if err != nil {
		return fmt.Errorf("applicationRepo.Delete: %w", mapDBError(err))
	}
	return requireAffected(tag)
}

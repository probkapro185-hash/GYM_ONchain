package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sfedu-crm/internal/domain"
	"github.com/sfedu-crm/internal/repository"
)

type trainerRepo struct{ db *pgxpool.Pool }

func NewTrainerRepository(db *pgxpool.Pool) repository.TrainerRepository {
	return &trainerRepo{db: db}
}

const trainerSelect = `SELECT t.id,t.user_id,u.full_name,t.specialization,
	COALESCE(t.bio,''),COALESCE(t.photo_url,''),t.experience_years,(t.is_active AND u.is_active),t.created_at
	FROM trainers t JOIN users u ON u.id=t.user_id`

func scanTrainer(row interface{ Scan(...any) error }) (*domain.Trainer, error) {
	t := &domain.Trainer{}
	if err := row.Scan(&t.ID, &t.UserID, &t.FullName, &t.Specialization, &t.Bio,
		&t.PhotoURL, &t.ExperienceYears, &t.IsActive, &t.CreatedAt); err != nil {
		return nil, mapDBError(err)
	}
	return t, nil
}

func (r *trainerRepo) Create(ctx context.Context, input domain.CreateTrainerInput) (*domain.Trainer, error) {
	const q = `INSERT INTO trainers(user_id,specialization,bio,photo_url,experience_years)
		VALUES($1,$2,$3,$4,$5) RETURNING id`
	var id int64
	if err := queryer(ctx, r.db).QueryRow(ctx, q, input.UserID, input.Specialization,
		input.Bio, input.PhotoURL, input.ExperienceYears).Scan(&id); err != nil {
		return nil, fmt.Errorf("trainerRepo.Create: %w", mapDBError(err))
	}
	return r.GetByID(ctx, id)
}

func (r *trainerRepo) GetByID(ctx context.Context, id int64) (*domain.Trainer, error) {
	t, err := scanTrainer(queryer(ctx, r.db).QueryRow(ctx, trainerSelect+` WHERE t.id=$1`, id))
	if err != nil {
		return nil, fmt.Errorf("trainerRepo.GetByID: %w", err)
	}
	return t, nil
}

func (r *trainerRepo) GetByIDForUpdate(ctx context.Context, id int64) (*domain.Trainer, error) {
	t, err := scanTrainer(queryer(ctx, r.db).QueryRow(ctx, trainerSelect+` WHERE t.id=$1 FOR UPDATE OF t, u`, id))
	if err != nil {
		return nil, fmt.Errorf("trainerRepo.GetByIDForUpdate: %w", err)
	}
	return t, nil
}

func (r *trainerRepo) GetByUserID(ctx context.Context, userID int64) (*domain.Trainer, error) {
	t, err := scanTrainer(queryer(ctx, r.db).QueryRow(ctx, trainerSelect+` WHERE t.user_id=$1`, userID))
	if err != nil {
		return nil, fmt.Errorf("trainerRepo.GetByUserID: %w", err)
	}
	return t, nil
}

func (r *trainerRepo) List(ctx context.Context, filter repository.TrainerFilter) ([]*domain.Trainer, error) {
	var conditions []string
	var args []any
	if filter.Specialization != nil {
		args = append(args, *filter.Specialization)
		conditions = append(conditions, fmt.Sprintf("t.specialization=$%d", len(args)))
	}
	if filter.IsActive != nil {
		args = append(args, *filter.IsActive)
		conditions = append(conditions, fmt.Sprintf("(t.is_active AND u.is_active)=$%d", len(args)))
	}
	if strings.TrimSpace(filter.Search) != "" {
		args = append(args, "%"+strings.TrimSpace(filter.Search)+"%")
		conditions = append(conditions, fmt.Sprintf("u.full_name ILIKE $%d", len(args)))
	}
	q := trainerSelect
	if len(conditions) > 0 {
		q += " WHERE " + strings.Join(conditions, " AND ")
	}
	q += " ORDER BY u.full_name"
	rows, err := queryer(ctx, r.db).Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("trainerRepo.List: %w", err)
	}
	defer rows.Close()
	result := make([]*domain.Trainer, 0)
	for rows.Next() {
		t := &domain.Trainer{}
		if err := rows.Scan(&t.ID, &t.UserID, &t.FullName, &t.Specialization, &t.Bio,
			&t.PhotoURL, &t.ExperienceYears, &t.IsActive, &t.CreatedAt); err != nil {
			return nil, fmt.Errorf("trainerRepo.List scan: %w", err)
		}
		result = append(result, t)
	}
	return result, rows.Err()
}

func (r *trainerRepo) Update(ctx context.Context, id int64, input domain.UpdateTrainerInput) (*domain.Trainer, error) {
	tag, err := queryer(ctx, r.db).Exec(ctx, `UPDATE trainers SET specialization=$1,bio=$2,photo_url=$3,
		experience_years=$4,is_active=$5 WHERE id=$6`, input.Specialization, input.Bio, input.PhotoURL,
		input.ExperienceYears, input.IsActive, id)
	if err != nil {
		return nil, fmt.Errorf("trainerRepo.Update: %w", mapDBError(err))
	}
	if err := requireAffected(tag); err != nil {
		return nil, err
	}
	return r.GetByID(ctx, id)
}

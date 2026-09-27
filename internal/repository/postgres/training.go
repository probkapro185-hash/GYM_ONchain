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

type trainingRepo struct{ db *pgxpool.Pool }
type trainingRequestRepo struct{ db *pgxpool.Pool }

func NewTrainingRepository(db *pgxpool.Pool) repository.TrainingRepository {
	return &trainingRepo{db: db}
}
func NewTrainingRequestRepository(db *pgxpool.Pool) repository.TrainingRequestRepository {
	return &trainingRequestRepo{db: db}
}

const trainingSelect = `SELECT tr.id,tr.client_id,tr.subscription_id,tr.trainer_id,tr.title,COALESCE(tr.description,''),
	tr.start_time,tr.end_time,tr.status,tr.created_at,tr.updated_at,
	c.full_name,COALESCE(tu.full_name,'')
	FROM trainings tr
	JOIN users c ON c.id=tr.client_id
	LEFT JOIN trainers tp ON tp.id=tr.trainer_id
	LEFT JOIN users tu ON tu.id=tp.user_id`

func scanTraining(row interface{ Scan(...any) error }) (*domain.Training, error) {
	t := &domain.Training{}
	if err := row.Scan(&t.ID, &t.ClientID, &t.SubscriptionID, &t.TrainerID, &t.Title, &t.Description,
		&t.StartTime, &t.EndTime, &t.Status, &t.CreatedAt, &t.UpdatedAt,
		&t.ClientName, &t.TrainerName); err != nil {
		return nil, mapDBError(err)
	}
	return t, nil
}

func (r *trainingRepo) Create(ctx context.Context, input domain.CreateTrainingInput) (*domain.Training, error) {
	const q = `INSERT INTO trainings(client_id,subscription_id,trainer_id,title,description,start_time,end_time,status)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8) RETURNING id`
	var id int64
	if err := queryer(ctx, r.db).QueryRow(ctx, q, input.ClientID, input.SubscriptionID, input.TrainerID,
		input.Title, input.Description, input.StartTime, input.EndTime, input.Status).Scan(&id); err != nil {
		return nil, fmt.Errorf("trainingRepo.Create: %w", mapDBError(err))
	}
	return r.GetByID(ctx, id)
}

func (r *trainingRepo) GetByID(ctx context.Context, id int64) (*domain.Training, error) {
	t, err := scanTraining(queryer(ctx, r.db).QueryRow(ctx, trainingSelect+` WHERE tr.id=$1`, id))
	if err != nil {
		return nil, fmt.Errorf("trainingRepo.GetByID: %w", err)
	}
	return t, nil
}

func (r *trainingRepo) GetByIDForUpdate(ctx context.Context, id int64) (*domain.Training, error) {
	// Lock only the training row; PostgreSQL does not allow FOR UPDATE on nullable side of LEFT JOIN.
	const q = `SELECT id,client_id,subscription_id,trainer_id,title,COALESCE(description,''),start_time,end_time,status,created_at,updated_at
		FROM trainings WHERE id=$1 FOR UPDATE`
	t := &domain.Training{}
	if err := queryer(ctx, r.db).QueryRow(ctx, q, id).Scan(&t.ID, &t.ClientID, &t.SubscriptionID, &t.TrainerID,
		&t.Title, &t.Description, &t.StartTime, &t.EndTime, &t.Status, &t.CreatedAt, &t.UpdatedAt); err != nil {
		return nil, fmt.Errorf("trainingRepo.GetByIDForUpdate: %w", mapDBError(err))
	}
	return t, nil
}

func (r *trainingRepo) List(ctx context.Context, filter domain.ScheduleFilter) ([]*domain.Training, error) {
	var conditions []string
	var args []any
	add := func(col string, value any) {
		args = append(args, value)
		conditions = append(conditions, fmt.Sprintf("%s=$%d", col, len(args)))
	}
	if filter.ClientID != nil {
		add("tr.client_id", *filter.ClientID)
	}
	if filter.TrainerID != nil {
		add("tr.trainer_id", *filter.TrainerID)
	}
	if filter.DateFrom != nil {
		args = append(args, *filter.DateFrom)
		conditions = append(conditions, fmt.Sprintf("tr.start_time >= $%d", len(args)))
	}
	if filter.DateTo != nil {
		args = append(args, *filter.DateTo)
		conditions = append(conditions, fmt.Sprintf("tr.start_time < $%d", len(args)))
	}
	if filter.Status != "" {
		add("tr.status", filter.Status)
	}
	q := trainingSelect
	if len(conditions) > 0 {
		q += " WHERE " + strings.Join(conditions, " AND ")
	}
	q += " ORDER BY tr.start_time"
	rows, err := queryer(ctx, r.db).Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("trainingRepo.List: %w", err)
	}
	defer rows.Close()
	result := make([]*domain.Training, 0)
	for rows.Next() {
		t := &domain.Training{}
		if err := rows.Scan(&t.ID, &t.ClientID, &t.SubscriptionID, &t.TrainerID, &t.Title, &t.Description,
			&t.StartTime, &t.EndTime, &t.Status, &t.CreatedAt, &t.UpdatedAt,
			&t.ClientName, &t.TrainerName); err != nil {
			return nil, fmt.Errorf("trainingRepo.List scan: %w", err)
		}
		result = append(result, t)
	}
	return result, rows.Err()
}

func (r *trainingRepo) Update(ctx context.Context, id int64, input domain.UpdateTrainingInput) (*domain.Training, error) {
	tag, err := queryer(ctx, r.db).Exec(ctx, `UPDATE trainings SET subscription_id=$1,trainer_id=$2,title=$3,description=$4,
		start_time=$5,end_time=$6,status=$7,updated_at=NOW() WHERE id=$8`,
		input.SubscriptionID, input.TrainerID, input.Title, input.Description, input.StartTime, input.EndTime, input.Status, id)
	if err != nil {
		return nil, fmt.Errorf("trainingRepo.Update: %w", mapDBError(err))
	}
	if err := requireAffected(tag); err != nil {
		return nil, err
	}
	return r.GetByID(ctx, id)
}

func (r *trainingRepo) Delete(ctx context.Context, id int64) error {
	tag, err := queryer(ctx, r.db).Exec(ctx, `DELETE FROM trainings WHERE id=$1`, id)
	if err != nil {
		return fmt.Errorf("trainingRepo.Delete: %w", mapDBError(err))
	}
	return requireAffected(tag)
}

func (r *trainingRepo) HasScheduleConflict(ctx context.Context, excludeID, clientID int64, trainerID *int64, start, end time.Time) (bool, error) {
	const q = `SELECT EXISTS (
		SELECT 1 FROM trainings
		WHERE id<>$1 AND status='scheduled'
		  AND start_time < $5 AND end_time > $4
		  AND (client_id=$2 OR ($3::bigint IS NOT NULL AND trainer_id=$3))
	)`
	var exists bool
	if err := queryer(ctx, r.db).QueryRow(ctx, q, excludeID, clientID, trainerID, start, end).Scan(&exists); err != nil {
		return false, fmt.Errorf("trainingRepo.HasScheduleConflict: %w", mapDBError(err))
	}
	return exists, nil
}

func (r *trainingRepo) CountScheduledBySubscription(ctx context.Context, subscriptionID, excludeTrainingID int64) (int, error) {
	const q = `SELECT COUNT(*) FROM trainings
		WHERE subscription_id=$1 AND id<>$2 AND status='scheduled'`
	var count int
	if err := queryer(ctx, r.db).QueryRow(ctx, q, subscriptionID, excludeTrainingID).Scan(&count); err != nil {
		return 0, fmt.Errorf("trainingRepo.CountScheduledBySubscription: %w", mapDBError(err))
	}
	return count, nil
}

func scanTrainingRequest(row interface{ Scan(...any) error }) (*domain.TrainingRequest, error) {
	req := &domain.TrainingRequest{}
	if err := row.Scan(&req.ID, &req.ClientID, &req.PreferredAt, &req.Comment,
		&req.Status, &req.CreatedAt, &req.ClientName, &req.TrainerID, &req.TrainerName); err != nil {
		return nil, mapDBError(err)
	}
	return req, nil
}

const requestSelect = `SELECT r.id,r.client_id,r.preferred_at,COALESCE(r.comment,''),r.status,r.created_at,u.full_name,r.trainer_id,COALESCE(tu.full_name,'')
	FROM training_requests r JOIN users u ON u.id=r.client_id
	LEFT JOIN trainers tr ON tr.id=r.trainer_id LEFT JOIN users tu ON tu.id=tr.user_id`

func (r *trainingRequestRepo) Create(ctx context.Context, clientID int64, input domain.CreateTrainingRequestInput) (*domain.TrainingRequest, error) {
	const q = `INSERT INTO training_requests(client_id,preferred_at,comment,status,trainer_id)
		VALUES($1,$2,$3,'pending',$4) RETURNING id`
	var id int64
	if err := queryer(ctx, r.db).QueryRow(ctx, q, clientID, input.PreferredAt, input.Comment, input.TrainerID).Scan(&id); err != nil {
		return nil, fmt.Errorf("trainingRequestRepo.Create: %w", mapDBError(err))
	}
	return r.GetByID(ctx, id)
}

func (r *trainingRequestRepo) GetByID(ctx context.Context, id int64) (*domain.TrainingRequest, error) {
	req, err := scanTrainingRequest(queryer(ctx, r.db).QueryRow(ctx, requestSelect+` WHERE r.id=$1`, id))
	if err != nil {
		return nil, fmt.Errorf("trainingRequestRepo.GetByID: %w", err)
	}
	return req, nil
}

func (r *trainingRequestRepo) GetByIDForUpdate(ctx context.Context, id int64) (*domain.TrainingRequest, error) {
	const q = `SELECT id,client_id,preferred_at,COALESCE(comment,''),status,created_at,trainer_id
		FROM training_requests WHERE id=$1 FOR UPDATE`
	req := &domain.TrainingRequest{}
	if err := queryer(ctx, r.db).QueryRow(ctx, q, id).Scan(&req.ID, &req.ClientID, &req.PreferredAt,
		&req.Comment, &req.Status, &req.CreatedAt, &req.TrainerID); err != nil {
		return nil, fmt.Errorf("trainingRequestRepo.GetByIDForUpdate: %w", mapDBError(err))
	}
	return req, nil
}

func (r *trainingRequestRepo) ListByClient(ctx context.Context, clientID int64) ([]*domain.TrainingRequest, error) {
	return r.list(ctx, requestSelect+` WHERE r.client_id=$1 ORDER BY r.created_at DESC`, clientID)
}

func (r *trainingRequestRepo) ListPending(ctx context.Context) ([]*domain.TrainingRequest, error) {
	return r.list(ctx, requestSelect+` WHERE r.status='pending' ORDER BY r.created_at`)
}

func (r *trainingRequestRepo) list(ctx context.Context, q string, args ...any) ([]*domain.TrainingRequest, error) {
	rows, err := queryer(ctx, r.db).Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("trainingRequestRepo.list: %w", err)
	}
	defer rows.Close()
	result := make([]*domain.TrainingRequest, 0)
	for rows.Next() {
		req := &domain.TrainingRequest{}
		if err := rows.Scan(&req.ID, &req.ClientID, &req.PreferredAt, &req.Comment,
			&req.Status, &req.CreatedAt, &req.ClientName, &req.TrainerID, &req.TrainerName); err != nil {
			return nil, fmt.Errorf("trainingRequestRepo.list scan: %w", err)
		}
		result = append(result, req)
	}
	return result, rows.Err()
}

func (r *trainingRequestRepo) UpdateStatus(ctx context.Context, id int64, status domain.TrainingRequestStatus) error {
	tag, err := queryer(ctx, r.db).Exec(ctx, `UPDATE training_requests SET status=$1 WHERE id=$2`, status, id)
	if err != nil {
		return fmt.Errorf("trainingRequestRepo.UpdateStatus: %w", mapDBError(err))
	}
	return requireAffected(tag)
}

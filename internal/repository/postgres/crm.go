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

type crmRepo struct{ db *pgxpool.Pool }

func NewCRMRepository(db *pgxpool.Pool) repository.CRMRepository { return &crmRepo{db: db} }

func (r *crmRepo) AddProgress(ctx context.Context, clientID, actorID int64, input domain.CreateProgressInput) (*domain.ClientProgress, error) {
	const q = `INSERT INTO client_progress(client_id,recorded_by,weight,body_fat,waist,chest,hips,comment)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8)
		RETURNING id,client_id,recorded_by,weight,body_fat,waist,chest,hips,COALESCE(comment,''),recorded_at`
	p := &domain.ClientProgress{}
	if err := queryer(ctx, r.db).QueryRow(ctx, q, clientID, actorID, input.Weight, input.BodyFat, input.Waist, input.Chest, input.Hips, input.Comment).
		Scan(&p.ID, &p.ClientID, &p.RecordedBy, &p.Weight, &p.BodyFat, &p.Waist, &p.Chest, &p.Hips, &p.Comment, &p.RecordedAt); err != nil {
		return nil, fmt.Errorf("crmRepo.AddProgress: %w", mapDBError(err))
	}
	return p, nil
}

func (r *crmRepo) ListProgress(ctx context.Context, clientID int64, limit int) ([]*domain.ClientProgress, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	const q = `SELECT p.id,p.client_id,p.recorded_by,p.weight,p.body_fat,p.waist,p.chest,p.hips,
		COALESCE(p.comment,''),p.recorded_at,u.full_name
		FROM client_progress p JOIN users u ON u.id=p.recorded_by
		WHERE p.client_id=$1 ORDER BY p.recorded_at DESC,p.id DESC LIMIT $2`
	rows, err := queryer(ctx, r.db).Query(ctx, q, clientID, limit)
	if err != nil {
		return nil, fmt.Errorf("crmRepo.ListProgress: %w", err)
	}
	defer rows.Close()
	out := make([]*domain.ClientProgress, 0)
	for rows.Next() {
		p := &domain.ClientProgress{}
		if err := rows.Scan(&p.ID, &p.ClientID, &p.RecordedBy, &p.Weight, &p.BodyFat, &p.Waist, &p.Chest, &p.Hips, &p.Comment, &p.RecordedAt, &p.AuthorName); err != nil {
			return nil, fmt.Errorf("crmRepo.ListProgress scan: %w", err)
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *crmRepo) DeleteProgress(ctx context.Context, id int64) error {
	tag, err := queryer(ctx, r.db).Exec(ctx, `DELETE FROM client_progress WHERE id=$1`, id)
	if err != nil {
		return fmt.Errorf("crmRepo.DeleteProgress: %w", mapDBError(err))
	}
	return requireAffected(tag)
}

func (r *crmRepo) AddNote(ctx context.Context, clientID, actorID int64, note string) (*domain.ClientNote, error) {
	const q = `INSERT INTO client_notes(client_id,author_id,note) VALUES($1,$2,$3)
		RETURNING id,client_id,author_id,note,created_at`
	n := &domain.ClientNote{}
	if err := queryer(ctx, r.db).QueryRow(ctx, q, clientID, actorID, note).Scan(&n.ID, &n.ClientID, &n.AuthorID, &n.Note, &n.CreatedAt); err != nil {
		return nil, fmt.Errorf("crmRepo.AddNote: %w", mapDBError(err))
	}
	return n, nil
}

func (r *crmRepo) ListNotes(ctx context.Context, clientID int64, limit int) ([]*domain.ClientNote, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	const q = `SELECT n.id,n.client_id,n.author_id,n.note,n.created_at,u.full_name
		FROM client_notes n JOIN users u ON u.id=n.author_id
		WHERE n.client_id=$1 ORDER BY n.created_at DESC,n.id DESC LIMIT $2`
	rows, err := queryer(ctx, r.db).Query(ctx, q, clientID, limit)
	if err != nil {
		return nil, fmt.Errorf("crmRepo.ListNotes: %w", err)
	}
	defer rows.Close()
	out := make([]*domain.ClientNote, 0)
	for rows.Next() {
		n := &domain.ClientNote{}
		if err := rows.Scan(&n.ID, &n.ClientID, &n.AuthorID, &n.Note, &n.CreatedAt, &n.AuthorName); err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

func (r *crmRepo) DeleteNote(ctx context.Context, id int64) error {
	tag, err := queryer(ctx, r.db).Exec(ctx, `DELETE FROM client_notes WHERE id=$1`, id)
	if err != nil {
		return fmt.Errorf("crmRepo.DeleteNote: %w", mapDBError(err))
	}
	return requireAffected(tag)
}

func scanTask(row interface{ Scan(...any) error }) (*domain.StaffTask, error) {
	t := &domain.StaffTask{}
	if err := row.Scan(&t.ID, &t.ClientID, &t.AssigneeID, &t.CreatedBy, &t.Title, &t.Description, &t.DueAt, &t.Status, &t.CompletedAt, &t.CreatedAt, &t.UpdatedAt, &t.ClientName, &t.Assignee); err != nil {
		return nil, mapDBError(err)
	}
	t.IsOverdue = t.Status == "open" && t.DueAt != nil && t.DueAt.Before(time.Now())
	return t, nil
}

const taskSelect = `SELECT t.id,t.client_id,t.assignee_id,t.created_by,t.title,COALESCE(t.description,''),t.due_at,t.status,t.completed_at,t.created_at,t.updated_at,
	COALESCE(c.full_name,''),a.full_name FROM staff_tasks t LEFT JOIN users c ON c.id=t.client_id JOIN users a ON a.id=t.assignee_id`

func (r *crmRepo) CreateTask(ctx context.Context, actorID int64, input domain.CreateTaskInput) (*domain.StaffTask, error) {
	const q = `INSERT INTO staff_tasks(client_id,assignee_id,created_by,title,description,due_at) VALUES($1,$2,$3,$4,$5,$6) RETURNING id`
	var id int64
	if err := queryer(ctx, r.db).QueryRow(ctx, q, input.ClientID, input.AssigneeID, actorID, input.Title, input.Description, input.DueAt).Scan(&id); err != nil {
		return nil, fmt.Errorf("crmRepo.CreateTask: %w", mapDBError(err))
	}
	return scanTask(queryer(ctx, r.db).QueryRow(ctx, taskSelect+` WHERE t.id=$1`, id))
}

func (r *crmRepo) ListTasks(ctx context.Context, assigneeID *int64, clientID *int64, includeDone bool, limit int) ([]*domain.StaffTask, error) {
	if limit <= 0 || limit > 1000 {
		limit = 200
	}
	where := []string{"1=1"}
	args := []any{}
	if assigneeID != nil {
		args = append(args, *assigneeID)
		where = append(where, fmt.Sprintf("t.assignee_id=$%d", len(args)))
	}
	if clientID != nil {
		args = append(args, *clientID)
		where = append(where, fmt.Sprintf("t.client_id=$%d", len(args)))
	}
	if !includeDone {
		where = append(where, "t.status='open'")
	}
	args = append(args, limit)
	q := taskSelect + ` WHERE ` + strings.Join(where, " AND ") + fmt.Sprintf(` ORDER BY CASE WHEN t.status='open' THEN 0 ELSE 1 END,t.due_at NULLS LAST,t.created_at DESC LIMIT $%d`, len(args))
	rows, err := queryer(ctx, r.db).Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("crmRepo.ListTasks: %w", err)
	}
	defer rows.Close()
	out := make([]*domain.StaffTask, 0)
	for rows.Next() {
		t := &domain.StaffTask{}
		if err := rows.Scan(&t.ID, &t.ClientID, &t.AssigneeID, &t.CreatedBy, &t.Title, &t.Description, &t.DueAt, &t.Status, &t.CompletedAt, &t.CreatedAt, &t.UpdatedAt, &t.ClientName, &t.Assignee); err != nil {
			return nil, err
		}
		t.IsOverdue = t.Status == "open" && t.DueAt != nil && t.DueAt.Before(time.Now())
		out = append(out, t)
	}
	return out, rows.Err()
}

func (r *crmRepo) UpdateTaskStatus(ctx context.Context, id int64, status string, assigneeID *int64) (*domain.StaffTask, error) {
	const q = `UPDATE staff_tasks SET status=$1::text,completed_at=CASE WHEN $1::text='done' THEN NOW() ELSE NULL END,updated_at=NOW() WHERE id=$2 AND ($3::bigint IS NULL OR assignee_id=$3) RETURNING id`
	var got int64
	if err := queryer(ctx, r.db).QueryRow(ctx, q, status, id, assigneeID).Scan(&got); err != nil {
		return nil, fmt.Errorf("crmRepo.UpdateTaskStatus: %w", mapDBError(err))
	}
	return scanTask(queryer(ctx, r.db).QueryRow(ctx, taskSelect+` WHERE t.id=$1`, got))
}

func (r *crmRepo) CreateNotification(ctx context.Context, input domain.CreateNotificationInput) (*domain.Notification, error) {
	const q = `INSERT INTO notifications(user_id,title,message) VALUES($1,$2,$3) RETURNING id,user_id,title,message,is_read,created_at`
	n := &domain.Notification{}
	if err := queryer(ctx, r.db).QueryRow(ctx, q, input.UserID, input.Title, input.Message).Scan(&n.ID, &n.UserID, &n.Title, &n.Message, &n.IsRead, &n.CreatedAt); err != nil {
		return nil, mapDBError(err)
	}
	return n, nil
}
func (r *crmRepo) ListNotifications(ctx context.Context, userID int64, limit int) ([]*domain.Notification, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := queryer(ctx, r.db).Query(ctx, `SELECT id,user_id,title,message,is_read,created_at FROM notifications WHERE user_id=$1 ORDER BY created_at DESC LIMIT $2`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]*domain.Notification, 0)
	for rows.Next() {
		n := &domain.Notification{}
		if err := rows.Scan(&n.ID, &n.UserID, &n.Title, &n.Message, &n.IsRead, &n.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, rows.Err()
}
func (r *crmRepo) MarkNotificationRead(ctx context.Context, id, userID int64) error {
	tag, err := queryer(ctx, r.db).Exec(ctx, `UPDATE notifications SET is_read=true WHERE id=$1 AND user_id=$2`, id, userID)
	if err != nil {
		return mapDBError(err)
	}
	return requireAffected(tag)
}

func (r *crmRepo) Dashboard(ctx context.Context) (*domain.DashboardSummary, error) {
	d := &domain.DashboardSummary{}
	q := queryer(ctx, r.db)
	if err := q.QueryRow(ctx, `SELECT COUNT(*) FROM users WHERE role='client' AND is_active=true`).Scan(&d.ActiveClients); err != nil {
		return nil, err
	}
	if err := q.QueryRow(ctx, `SELECT COUNT(*) FROM client_subscriptions WHERE is_active=true AND frozen_at IS NULL AND start_date<=NOW() AND end_date>NOW() AND (sessions_left IS NULL OR sessions_left>0)`).Scan(&d.ActiveSubscriptions); err != nil {
		return nil, err
	}
	if err := q.QueryRow(ctx, `SELECT COUNT(*) FROM client_subscriptions WHERE is_active=true AND frozen_at IS NOT NULL`).Scan(&d.FrozenSubscriptions); err != nil {
		return nil, err
	}
	if err := q.QueryRow(ctx, `SELECT COUNT(*) FROM trainings WHERE start_time>=date_trunc('day',NOW()) AND start_time<date_trunc('day',NOW())+interval '1 day' AND status='scheduled'`).Scan(&d.TrainingsToday); err != nil {
		return nil, err
	}
	if err := q.QueryRow(ctx, `SELECT COUNT(*) FROM application_requests WHERE status='pending'`).Scan(&d.PendingApplications); err != nil {
		return nil, err
	}
	if err := q.QueryRow(ctx, `SELECT COUNT(*) FROM staff_tasks WHERE status='open'`).Scan(&d.OpenTasks); err != nil {
		return nil, err
	}
	if err := q.QueryRow(ctx, `SELECT COUNT(*) FROM users WHERE role='client' AND is_active=true AND (last_visit_at IS NULL OR last_visit_at < NOW()-interval '14 days')`).Scan(&d.AtRiskClients); err != nil {
		return nil, err
	}
	if err := q.QueryRow(ctx, `SELECT COUNT(*) FROM client_subscriptions WHERE is_active=true AND frozen_at IS NULL AND (sessions_left IS NULL OR sessions_left>0) AND end_date>NOW() AND end_date<=NOW()+interval '7 days'`).Scan(&d.ExpiringSubscriptions); err != nil {
		return nil, err
	}
	if err := q.QueryRow(ctx, `SELECT COALESCE(SUM(amount) FILTER (WHERE operation_type='income'),0) FROM payments WHERE created_at>=NOW()-interval '30 days'`).Scan(&d.Revenue30Days); err != nil {
		return nil, err
	}
	if err := q.QueryRow(ctx, `SELECT COUNT(*) FROM trainings WHERE status='completed' AND start_time>=NOW()-interval '30 days'`).Scan(&d.CompletedTrainings30d); err != nil {
		return nil, err
	}
	rows, err := q.Query(ctx, `SELECT tr.id,u.full_name,
		COUNT(t.id) FILTER (WHERE t.start_time>=NOW()-interval '30 days' AND t.status IN ('scheduled','completed')),
		COUNT(t.id) FILTER (WHERE t.start_time>=NOW()-interval '30 days' AND t.status='completed')
		FROM trainers tr JOIN users u ON u.id=tr.user_id LEFT JOIN trainings t ON t.trainer_id=tr.id
		WHERE tr.is_active=true GROUP BY tr.id,u.full_name ORDER BY u.full_name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	d.TrainerUtilization = make([]domain.TrainerUtilization, 0)
	for rows.Next() {
		var v domain.TrainerUtilization
		if err := rows.Scan(&v.TrainerID, &v.TrainerName, &v.Scheduled30d, &v.Completed30d); err != nil {
			return nil, err
		}
		if v.Scheduled30d > 0 {
			v.CompletionRate = float64(v.Completed30d) * 100 / float64(v.Scheduled30d)
		}
		d.TrainerUtilization = append(d.TrainerUtilization, v)
	}
	return d, rows.Err()
}

func (r *crmRepo) RecordAudit(ctx context.Context, actorID *int64, actorRole, requestID, method, path string, statusCode int) error {
	_, err := queryer(ctx, r.db).Exec(ctx, `INSERT INTO audit_log(actor_id,actor_role,request_id,method,path,status_code) VALUES($1,$2,$3,$4,$5,$6)`, actorID, actorRole, requestID, method, path, statusCode)
	if err != nil {
		return fmt.Errorf("crmRepo.RecordAudit: %w", mapDBError(err))
	}
	return nil
}
func (r *crmRepo) ListAudit(ctx context.Context, limit int) ([]*domain.AuditRecord, error) {
	if limit <= 0 || limit > 1000 {
		limit = 200
	}
	rows, err := queryer(ctx, r.db).Query(ctx, `SELECT a.id,a.actor_id,COALESCE(a.actor_role,''),a.request_id,a.method,a.path,a.status_code,a.created_at,COALESCE(u.full_name,'') FROM audit_log a LEFT JOIN users u ON u.id=a.actor_id ORDER BY a.created_at DESC LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]*domain.AuditRecord, 0)
	for rows.Next() {
		a := &domain.AuditRecord{}
		if err := rows.Scan(&a.ID, &a.ActorID, &a.ActorRole, &a.RequestID, &a.Method, &a.Path, &a.StatusCode, &a.CreatedAt, &a.ActorName); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (r *crmRepo) HasFutureTrainingForSubscription(ctx context.Context, subscriptionID int64) (bool, error) {
	var exists bool
	err := queryer(ctx, r.db).QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM trainings WHERE subscription_id=$1 AND status='scheduled' AND start_time>NOW())`, subscriptionID).Scan(&exists)
	return exists, err
}
func (r *crmRepo) FreezeSubscription(ctx context.Context, id int64) (*domain.ClientSubscription, error) {
	tag, err := queryer(ctx, r.db).Exec(ctx, `UPDATE client_subscriptions SET frozen_at=NOW() WHERE id=$1 AND is_active=true AND frozen_at IS NULL AND end_date>NOW()`, id)
	if err != nil {
		return nil, mapDBError(err)
	}
	if err := requireAffected(tag); err != nil {
		return nil, err
	}
	return scanSubscription(queryer(ctx, r.db).QueryRow(ctx, subscriptionSelect+` WHERE s.id=$1`, id))
}
func (r *crmRepo) UnfreezeSubscription(ctx context.Context, id int64) (*domain.ClientSubscription, error) {
	tag, err := queryer(ctx, r.db).Exec(ctx, `UPDATE client_subscriptions SET end_date=end_date+(NOW()-frozen_at),freeze_days=freeze_days+GREATEST(1,CEIL(EXTRACT(EPOCH FROM (NOW()-frozen_at))/86400.0)::int),frozen_at=NULL WHERE id=$1 AND is_active=true AND frozen_at IS NOT NULL`, id)
	if err != nil {
		return nil, mapDBError(err)
	}
	if err := requireAffected(tag); err != nil {
		return nil, err
	}
	return scanSubscription(queryer(ctx, r.db).QueryRow(ctx, subscriptionSelect+` WHERE s.id=$1`, id))
}
func (r *crmRepo) ExtendSubscription(ctx context.Context, id int64, days int) (*domain.ClientSubscription, error) {
	tag, err := queryer(ctx, r.db).Exec(ctx, `UPDATE client_subscriptions SET end_date=end_date+($2::text||' days')::interval WHERE id=$1 AND is_active=true`, id, days)
	if err != nil {
		return nil, mapDBError(err)
	}
	if err := requireAffected(tag); err != nil {
		return nil, err
	}
	return scanSubscription(queryer(ctx, r.db).QueryRow(ctx, subscriptionSelect+` WHERE s.id=$1`, id))
}

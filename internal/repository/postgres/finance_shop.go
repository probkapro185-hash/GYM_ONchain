package postgres

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sfedu-crm/internal/domain"
	"github.com/sfedu-crm/internal/repository"
)

type productRepo struct{ db *pgxpool.Pool }
type subscriptionRepo struct{ db *pgxpool.Pool }
type paymentRepo struct{ db *pgxpool.Pool }
type orderRepo struct{ db *pgxpool.Pool }

func NewProductRepository(db *pgxpool.Pool) repository.ProductRepository {
	return &productRepo{db: db}
}
func NewSubscriptionRepository(db *pgxpool.Pool) repository.SubscriptionRepository {
	return &subscriptionRepo{db: db}
}
func NewPaymentRepository(db *pgxpool.Pool) repository.PaymentRepository {
	return &paymentRepo{db: db}
}
func NewOrderRepository(db *pgxpool.Pool) repository.OrderRepository {
	return &orderRepo{db: db}
}

const productColumns = `id,name,COALESCE(description,''),price,category,sub_type,duration_days,sessions_count,is_active,created_at,photo_url`

func scanProduct(row interface{ Scan(...any) error }) (*domain.Product, error) {
	p := &domain.Product{}
	if err := row.Scan(&p.ID, &p.Name, &p.Description, &p.Price, &p.Category,
		&p.SubType, &p.DurationDays, &p.SessionsCount, &p.IsActive, &p.CreatedAt, &p.PhotoURL); err != nil {
		return nil, mapDBError(err)
	}
	return p, nil
}

func (r *productRepo) Create(ctx context.Context, input domain.CreateProductInput) (*domain.Product, error) {
	q := `INSERT INTO products(name,description,price,category,sub_type,duration_days,sessions_count)
		VALUES($1,$2,($3::bigint/100.0),$4,$5,$6,$7) RETURNING ` + productColumns
	priceCents := int64(math.Round(input.Price * 100))
	p, err := scanProduct(queryer(ctx, r.db).QueryRow(ctx, q, input.Name, input.Description, priceCents,
		input.Category, input.SubType, input.DurationDays, input.SessionsCount))
	if err != nil {
		return nil, fmt.Errorf("productRepo.Create: %w", err)
	}
	return p, nil
}

func (r *productRepo) GetByID(ctx context.Context, id int64) (*domain.Product, error) {
	p, err := scanProduct(queryer(ctx, r.db).QueryRow(ctx, `SELECT `+productColumns+` FROM products WHERE id=$1`, id))
	if err != nil {
		return nil, fmt.Errorf("productRepo.GetByID: %w", err)
	}
	return p, nil
}

func (r *productRepo) GetByIDForUpdate(ctx context.Context, id int64) (*domain.Product, error) {
	p, err := scanProduct(queryer(ctx, r.db).QueryRow(ctx,
		`SELECT `+productColumns+` FROM products WHERE id=$1 FOR UPDATE`, id))
	if err != nil {
		return nil, fmt.Errorf("productRepo.GetByIDForUpdate: %w", err)
	}
	return p, nil
}

func (r *productRepo) List(ctx context.Context, filter repository.ProductFilter) ([]*domain.Product, error) {
	var conditions []string
	var args []any
	if filter.Category != nil {
		args = append(args, *filter.Category)
		conditions = append(conditions, fmt.Sprintf("category=$%d", len(args)))
	}
	if filter.IsActive != nil {
		args = append(args, *filter.IsActive)
		conditions = append(conditions, fmt.Sprintf("is_active=$%d", len(args)))
	}
	q := `SELECT ` + productColumns + ` FROM products`
	if len(conditions) > 0 {
		q += " WHERE " + strings.Join(conditions, " AND ")
	}
	q += " ORDER BY created_at DESC"
	rows, err := queryer(ctx, r.db).Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("productRepo.List: %w", err)
	}
	defer rows.Close()
	result := make([]*domain.Product, 0)
	for rows.Next() {
		p := &domain.Product{}
		if err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.Price, &p.Category,
			&p.SubType, &p.DurationDays, &p.SessionsCount, &p.IsActive, &p.CreatedAt, &p.PhotoURL); err != nil {
			return nil, fmt.Errorf("productRepo.List scan: %w", err)
		}
		result = append(result, p)
	}
	return result, rows.Err()
}

func (r *productRepo) Update(ctx context.Context, id int64, input domain.CreateProductInput) (*domain.Product, error) {
	q := `UPDATE products SET name=$1,description=$2,price=($3::bigint/100.0),category=$4,sub_type=$5,duration_days=$6,
		sessions_count=$7 WHERE id=$8 RETURNING ` + productColumns
	priceCents := int64(math.Round(input.Price * 100))
	p, err := scanProduct(queryer(ctx, r.db).QueryRow(ctx, q, input.Name, input.Description, priceCents,
		input.Category, input.SubType, input.DurationDays, input.SessionsCount, id))
	if err != nil {
		return nil, fmt.Errorf("productRepo.Update: %w", err)
	}
	return p, nil
}

func (r *productRepo) SetActive(ctx context.Context, id int64, active bool) error {
	tag, err := queryer(ctx, r.db).Exec(ctx, `UPDATE products SET is_active=$1 WHERE id=$2`, active, id)
	if err != nil {
		return fmt.Errorf("productRepo.SetActive: %w", mapDBError(err))
	}
	return requireAffected(tag)
}

func (r *productRepo) SetPhoto(ctx context.Context, id int64, photoURL string) (*domain.Product, error) {
	p, err := scanProduct(queryer(ctx, r.db).QueryRow(ctx, `UPDATE products SET photo_url=$1 WHERE id=$2 RETURNING `+productColumns, photoURL, id))
	if err != nil {
		return nil, fmt.Errorf("productRepo.SetPhoto: %w", err)
	}
	return p, nil
}

const subscriptionSelect = `SELECT s.id,s.client_id,s.product_id,s.start_date,s.end_date,s.sessions_left,
	s.is_active,s.frozen_at,s.freeze_days,s.created_at,s.product_name,s.purchase_price
	FROM client_subscriptions s`

func scanSubscription(row interface{ Scan(...any) error }) (*domain.ClientSubscription, error) {
	s := &domain.ClientSubscription{}
	if err := row.Scan(&s.ID, &s.ClientID, &s.ProductID, &s.StartDate, &s.EndDate,
		&s.SessionsLeft, &s.IsActive, &s.FrozenAt, &s.FreezeDays, &s.CreatedAt, &s.ProductName, &s.Price); err != nil {
		return nil, mapDBError(err)
	}
	return s, nil
}

func (r *subscriptionRepo) Create(ctx context.Context, clientID int64, product *domain.Product) (*domain.ClientSubscription, error) {
	if product.DurationDays == nil || *product.DurationDays <= 0 {
		return nil, fmt.Errorf("%w: subscription product has no valid duration", domain.ErrInvalidInput)
	}
	end := time.Now().AddDate(0, 0, *product.DurationDays)
	const q = `INSERT INTO client_subscriptions(client_id,product_id,end_date,sessions_left,product_name,purchase_price)
		VALUES($1,$2,$3,$4,$5,($6::bigint/100.0)) RETURNING id`
	var id int64
	priceCents := int64(math.Round(product.Price * 100))
	if err := queryer(ctx, r.db).QueryRow(ctx, q, clientID, product.ID, end, product.SessionsCount, product.Name, priceCents).Scan(&id); err != nil {
		return nil, fmt.Errorf("subscriptionRepo.Create: %w", mapDBError(err))
	}
	return r.GetByID(ctx, id)
}

func (r *subscriptionRepo) GetByID(ctx context.Context, id int64) (*domain.ClientSubscription, error) {
	s, err := scanSubscription(queryer(ctx, r.db).QueryRow(ctx, subscriptionSelect+` WHERE s.id=$1`, id))
	if err != nil {
		return nil, fmt.Errorf("subscriptionRepo.GetByID: %w", err)
	}
	return s, nil
}

func (r *subscriptionRepo) GetByIDForUpdate(ctx context.Context, id int64) (*domain.ClientSubscription, error) {
	const q = `SELECT s.id,s.client_id,s.product_id,s.start_date,s.end_date,s.sessions_left,
		s.is_active,s.frozen_at,s.freeze_days,s.created_at,s.product_name,s.purchase_price
		FROM client_subscriptions s
		WHERE s.id=$1 FOR UPDATE OF s`
	s, err := scanSubscription(queryer(ctx, r.db).QueryRow(ctx, q, id))
	if err != nil {
		return nil, fmt.Errorf("subscriptionRepo.GetByIDForUpdate: %w", err)
	}
	return s, nil
}

func (r *subscriptionRepo) ListByClient(ctx context.Context, clientID int64) ([]*domain.ClientSubscription, error) {
	rows, err := queryer(ctx, r.db).Query(ctx, subscriptionSelect+` WHERE s.client_id=$1 ORDER BY s.created_at DESC`, clientID)
	if err != nil {
		return nil, fmt.Errorf("subscriptionRepo.ListByClient: %w", err)
	}
	defer rows.Close()
	result := make([]*domain.ClientSubscription, 0)
	for rows.Next() {
		s := &domain.ClientSubscription{}
		if err := rows.Scan(&s.ID, &s.ClientID, &s.ProductID, &s.StartDate, &s.EndDate,
			&s.SessionsLeft, &s.IsActive, &s.FrozenAt, &s.FreezeDays, &s.CreatedAt, &s.ProductName, &s.Price); err != nil {
			return nil, fmt.Errorf("subscriptionRepo.ListByClient scan: %w", err)
		}
		result = append(result, s)
	}
	return result, rows.Err()
}

func (r *subscriptionRepo) GetActiveByClient(ctx context.Context, clientID int64) (*domain.ClientSubscription, error) {
	q := subscriptionSelect + ` WHERE s.client_id=$1 AND s.is_active=true
		AND s.frozen_at IS NULL AND s.start_date<=NOW() AND s.end_date>NOW()
		AND (s.sessions_left IS NULL OR s.sessions_left>0) ORDER BY s.end_date ASC,s.id ASC LIMIT 1`
	s, err := scanSubscription(queryer(ctx, r.db).QueryRow(ctx, q, clientID))
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.ErrNoActiveSubscription
		}
		return nil, fmt.Errorf("subscriptionRepo.GetActiveByClient: %w", err)
	}
	return s, nil
}

func (r *subscriptionRepo) GetCoveringByClient(ctx context.Context, clientID int64, start, end time.Time) (*domain.ClientSubscription, error) {
	q := subscriptionSelect + ` WHERE s.client_id=$1 AND s.is_active=true
		AND s.frozen_at IS NULL AND s.start_date<=$2 AND s.end_date>=$3
		AND (s.sessions_left IS NULL OR s.sessions_left>0)
		ORDER BY s.end_date ASC,s.id ASC LIMIT 1`
	s, err := scanSubscription(queryer(ctx, r.db).QueryRow(ctx, q, clientID, start, end))
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.ErrNoActiveSubscription
		}
		return nil, fmt.Errorf("subscriptionRepo.GetCoveringByClient: %w", err)
	}
	return s, nil
}

func (r *subscriptionRepo) GetCoveringByClientForUpdate(ctx context.Context, clientID int64, start, end time.Time, excludeTrainingID int64) (*domain.ClientSubscription, error) {
	const q = `SELECT s.id,s.client_id,s.product_id,s.start_date,s.end_date,s.sessions_left,
		s.is_active,s.frozen_at,s.freeze_days,s.created_at,s.product_name,s.purchase_price
		FROM client_subscriptions s
		WHERE s.client_id=$1 AND s.is_active=true
		  AND s.frozen_at IS NULL AND s.start_date<=$2 AND s.end_date>=$3
		  AND (s.sessions_left IS NULL OR s.sessions_left > (
		      SELECT COUNT(*) FROM trainings t
		      WHERE t.subscription_id=s.id AND t.status='scheduled' AND t.id<>$4
		  ))
		ORDER BY s.end_date ASC,s.id ASC LIMIT 1 FOR UPDATE OF s`
	s, err := scanSubscription(queryer(ctx, r.db).QueryRow(ctx, q, clientID, start, end, excludeTrainingID))
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.ErrNoActiveSubscription
		}
		return nil, fmt.Errorf("subscriptionRepo.GetCoveringByClientForUpdate: %w", err)
	}
	return s, nil
}

func (r *subscriptionRepo) GetActiveByClientForUpdate(ctx context.Context, clientID int64) (*domain.ClientSubscription, error) {
	const q = `SELECT s.id,s.client_id,s.product_id,s.start_date,s.end_date,s.sessions_left,
		s.is_active,s.frozen_at,s.freeze_days,s.created_at,s.product_name,s.purchase_price
		FROM client_subscriptions s
		WHERE s.client_id=$1 AND s.is_active=true AND s.frozen_at IS NULL AND s.start_date<=NOW() AND s.end_date>NOW()
		  AND (s.sessions_left IS NULL OR s.sessions_left>0)
		ORDER BY s.end_date ASC,s.id ASC LIMIT 1 FOR UPDATE OF s`
	s, err := scanSubscription(queryer(ctx, r.db).QueryRow(ctx, q, clientID))
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.ErrNoActiveSubscription
		}
		return nil, fmt.Errorf("subscriptionRepo.GetActiveByClientForUpdate: %w", err)
	}
	return s, nil
}

func (r *subscriptionRepo) DecrementSessions(ctx context.Context, id int64) error {
	const q = `UPDATE client_subscriptions
		SET sessions_left=CASE WHEN sessions_left IS NULL THEN NULL ELSE sessions_left-1 END,
		    is_active=CASE WHEN sessions_left IS NOT NULL AND sessions_left-1<=0 THEN false ELSE is_active END
		WHERE id=$1 AND is_active=true AND (sessions_left IS NULL OR sessions_left>0)`
	tag, err := queryer(ctx, r.db).Exec(ctx, q, id)
	if err != nil {
		return fmt.Errorf("subscriptionRepo.DecrementSessions: %w", mapDBError(err))
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNoActiveSubscription
	}
	return nil
}

func (r *subscriptionRepo) Deactivate(ctx context.Context, id int64) error {
	tag, err := queryer(ctx, r.db).Exec(ctx, `UPDATE client_subscriptions SET is_active=false WHERE id=$1`, id)
	if err != nil {
		return fmt.Errorf("subscriptionRepo.Deactivate: %w", mapDBError(err))
	}
	return requireAffected(tag)
}

func (r *paymentRepo) Create(ctx context.Context, input domain.CreatePaymentInput) (*domain.Payment, error) {
	const q = `INSERT INTO payments(client_id,amount,operation_type,service_type,description)
		VALUES($1,($2::bigint/100.0),$3,$4,$5)
		RETURNING id,client_id,amount,operation_type,service_type,COALESCE(description,''),created_at`
	p := &domain.Payment{}
	if err := queryer(ctx, r.db).QueryRow(ctx, q, input.ClientID, input.AmountCents, input.OperationType,
		input.ServiceType, input.Description).Scan(&p.ID, &p.ClientID, &p.Amount, &p.OperationType,
		&p.ServiceType, &p.Description, &p.CreatedAt); err != nil {
		return nil, fmt.Errorf("paymentRepo.Create: %w", mapDBError(err))
	}
	return p, nil
}

func (r *paymentRepo) GetByID(ctx context.Context, id int64) (*domain.Payment, error) {
	const q = `SELECT p.id,p.client_id,p.amount,p.operation_type,p.service_type,COALESCE(p.description,''),
		p.created_at,u.full_name FROM payments p JOIN users u ON u.id=p.client_id WHERE p.id=$1`
	p := &domain.Payment{}
	if err := queryer(ctx, r.db).QueryRow(ctx, q, id).Scan(&p.ID, &p.ClientID, &p.Amount,
		&p.OperationType, &p.ServiceType, &p.Description, &p.CreatedAt, &p.ClientName); err != nil {
		return nil, fmt.Errorf("paymentRepo.GetByID: %w", mapDBError(err))
	}
	return p, nil
}

func paymentWhere(filter domain.PaymentFilter) (string, []any) {
	var conditions []string
	var args []any
	add := func(condition string, value any) {
		args = append(args, value)
		conditions = append(conditions, fmt.Sprintf(condition, len(args)))
	}
	if filter.ClientID != nil {
		add("p.client_id=$%d", *filter.ClientID)
	}
	if filter.OperationType != "" {
		add("p.operation_type=$%d", filter.OperationType)
	}
	if filter.ServiceType != "" {
		add("p.service_type=$%d", filter.ServiceType)
	}
	if filter.DateFrom != nil {
		args = append(args, *filter.DateFrom)
		conditions = append(conditions, fmt.Sprintf("p.created_at >= $%d", len(args)))
	}
	if filter.DateTo != nil {
		args = append(args, *filter.DateTo)
		conditions = append(conditions, fmt.Sprintf("p.created_at < $%d", len(args)))
	}
	if len(conditions) == 0 {
		return "", args
	}
	return " WHERE " + strings.Join(conditions, " AND "), args
}

func (r *paymentRepo) List(ctx context.Context, filter domain.PaymentFilter) ([]*domain.Payment, error) {
	where, args := paymentWhere(filter)
	q := `SELECT p.id,p.client_id,p.amount,p.operation_type,p.service_type,COALESCE(p.description,''),
		p.created_at,u.full_name FROM payments p JOIN users u ON u.id=p.client_id` + where + ` ORDER BY p.created_at DESC`
	rows, err := queryer(ctx, r.db).Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("paymentRepo.List: %w", err)
	}
	defer rows.Close()
	result := make([]*domain.Payment, 0)
	for rows.Next() {
		p := &domain.Payment{}
		if err := rows.Scan(&p.ID, &p.ClientID, &p.Amount, &p.OperationType, &p.ServiceType,
			&p.Description, &p.CreatedAt, &p.ClientName); err != nil {
			return nil, fmt.Errorf("paymentRepo.List scan: %w", err)
		}
		result = append(result, p)
	}
	return result, rows.Err()
}

func (r *paymentRepo) GetSummary(ctx context.Context, filter domain.PaymentFilter) (*domain.FinanceSummary, error) {
	where, args := paymentWhere(filter)
	q := `SELECT
		COALESCE(SUM(amount) FILTER (WHERE operation_type='income'),0),
		COALESCE(SUM(amount) FILTER (WHERE operation_type='expense'),0),
		COALESCE(SUM(amount) FILTER (WHERE operation_type='refund'),0)
		FROM payments p` + where
	s := &domain.FinanceSummary{}
	if err := queryer(ctx, r.db).QueryRow(ctx, q, args...).Scan(&s.TotalIncome, &s.TotalExpense, &s.TotalRefund); err != nil {
		return nil, fmt.Errorf("paymentRepo.GetSummary: %w", err)
	}
	s.NetBalance = s.TotalIncome - s.TotalExpense - s.TotalRefund
	return s, nil
}

func (r *orderRepo) Create(ctx context.Context, clientID int64, product *domain.Product) (*domain.Order, error) {
	const q = `INSERT INTO orders(client_id,product_id,amount,status,product_name)
		VALUES($1,$2,($3::bigint/100.0),'paid',$4)
		RETURNING id,client_id,product_id,amount,status,created_at,product_name`
	o := &domain.Order{}
	if err := queryer(ctx, r.db).QueryRow(ctx, q, clientID, product.ID, int64(math.Round(product.Price*100)), product.Name).Scan(
		&o.ID, &o.ClientID, &o.ProductID, &o.Amount, &o.Status, &o.CreatedAt, &o.ProductName); err != nil {
		return nil, fmt.Errorf("orderRepo.Create: %w", mapDBError(err))
	}
	return o, nil
}

func (r *orderRepo) ListByClient(ctx context.Context, clientID int64) ([]*domain.Order, error) {
	const q = `SELECT o.id,o.client_id,o.product_id,o.amount,o.status,o.created_at,u.full_name,o.product_name
		FROM orders o JOIN users u ON u.id=o.client_id
		WHERE o.client_id=$1 ORDER BY o.created_at DESC`
	rows, err := queryer(ctx, r.db).Query(ctx, q, clientID)
	if err != nil {
		return nil, fmt.Errorf("orderRepo.ListByClient: %w", err)
	}
	defer rows.Close()
	result := make([]*domain.Order, 0)
	for rows.Next() {
		o := &domain.Order{}
		if err := rows.Scan(&o.ID, &o.ClientID, &o.ProductID, &o.Amount, &o.Status,
			&o.CreatedAt, &o.ClientName, &o.ProductName); err != nil {
			return nil, fmt.Errorf("orderRepo.ListByClient scan: %w", err)
		}
		result = append(result, o)
	}
	return result, rows.Err()
}

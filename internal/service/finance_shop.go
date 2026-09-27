package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/sfedu-crm/internal/domain"
	"github.com/sfedu-crm/internal/repository"
	"github.com/sfedu-crm/internal/validator"
	pkgcache "github.com/sfedu-crm/pkg/cache"
)

type FinanceService struct {
	paymentRepo repository.PaymentRepository
	userRepo    repository.UserRepository
	tx          repository.TransactionManager
}

func NewFinanceService(paymentRepo repository.PaymentRepository, userRepo repository.UserRepository,
	tx repository.TransactionManager) *FinanceService {
	return &FinanceService{paymentRepo: paymentRepo, userRepo: userRepo, tx: tx}
}

func (s *FinanceService) TopUpBalance(ctx context.Context, clientID int64, amount float64) (*domain.Payment, error) {
	amountCents, err := validator.MoneyToCents(amount)
	if err != nil {
		return nil, err
	}
	var payment *domain.Payment
	err = s.tx.WithinTransaction(ctx, func(txCtx context.Context) error {
		client, err := s.userRepo.GetByIDForUpdate(txCtx, clientID)
		if err != nil {
			return err
		}
		if client.Role != domain.RoleClient || !client.IsActive {
			return domain.ErrForbidden
		}
		if err := s.userRepo.AddBalance(txCtx, clientID, amountCents); err != nil {
			return err
		}
		payment, err = s.paymentRepo.Create(txCtx, domain.CreatePaymentInput{
			ClientID: clientID, AmountCents: amountCents, OperationType: domain.OperationIncome,
			ServiceType: domain.ServiceDeposit, Description: "Пополнение баланса",
		})
		return err
	})
	return payment, err
}

func (s *FinanceService) GetMyPayments(ctx context.Context, clientID int64) ([]*domain.Payment, error) {
	return s.paymentRepo.List(ctx, domain.PaymentFilter{ClientID: &clientID})
}

func (s *FinanceService) ListPayments(ctx context.Context, filter domain.PaymentFilter) ([]*domain.Payment, error) {
	return s.paymentRepo.List(ctx, filter)
}

func (s *FinanceService) GetSummary(ctx context.Context, filter domain.PaymentFilter) (*domain.FinanceSummary, error) {
	return s.paymentRepo.GetSummary(ctx, filter)
}

type ShopService struct {
	productRepo      repository.ProductRepository
	subscriptionRepo repository.SubscriptionRepository
	paymentRepo      repository.PaymentRepository
	orderRepo        repository.OrderRepository
	userRepo         repository.UserRepository
	tx               repository.TransactionManager
	cache            *pkgcache.RedisCache
}

const (
	productsCacheKey = "products:active"
	productsTTL      = 3 * time.Minute
)

func NewShopService(productRepo repository.ProductRepository, subscriptionRepo repository.SubscriptionRepository,
	paymentRepo repository.PaymentRepository, orderRepo repository.OrderRepository,
	userRepo repository.UserRepository, tx repository.TransactionManager, cache *pkgcache.RedisCache) *ShopService {
	return &ShopService{
		productRepo: productRepo, subscriptionRepo: subscriptionRepo, paymentRepo: paymentRepo,
		orderRepo: orderRepo, userRepo: userRepo, tx: tx, cache: cache,
	}
}

func (s *ShopService) ListProducts(ctx context.Context, actorRole domain.Role, filter repository.ProductFilter) ([]*domain.Product, error) {
	if filter.Category != nil && *filter.Category != domain.CategorySubscription && *filter.Category != domain.CategorySports {
		return nil, fmt.Errorf("%w: invalid category", domain.ErrInvalidInput)
	}
	if actorRole == domain.RoleClient {
		active := true
		filter.IsActive = &active
	}
	useCache := s.cache != nil && actorRole == domain.RoleClient && filter.Category == nil
	if useCache {
		var cached []*domain.Product
		if err := s.cache.Get(ctx, productsCacheKey, &cached); err == nil {
			return cached, nil
		}
	}
	products, err := s.productRepo.List(ctx, filter)
	if err == nil && useCache {
		_ = s.cache.Set(ctx, productsCacheKey, products, productsTTL)
	}
	return products, err
}

func (s *ShopService) GetProduct(ctx context.Context, actorRole domain.Role, id int64) (*domain.Product, error) {
	product, err := s.productRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if actorRole == domain.RoleClient && !product.IsActive {
		return nil, domain.ErrNotFound
	}
	return product, nil
}

func (s *ShopService) PurchaseProduct(ctx context.Context, clientID int64, productID int64) (*domain.PurchaseResult, error) {
	var result domain.PurchaseResult
	err := s.tx.WithinTransaction(ctx, func(txCtx context.Context) error {
		product, err := s.productRepo.GetByIDForUpdate(txCtx, productID)
		if err != nil {
			return err
		}
		if !product.IsActive {
			return fmt.Errorf("%w: product is inactive", domain.ErrNotFound)
		}
		client, err := s.userRepo.GetByIDForUpdate(txCtx, clientID)
		if err != nil {
			return err
		}
		if client.Role != domain.RoleClient || !client.IsActive {
			return domain.ErrForbidden
		}
		amountCents, err := validator.MoneyToCents(product.Price)
		if err != nil {
			return err
		}
		if err := s.userRepo.DebitBalance(txCtx, clientID, amountCents); err != nil {
			return err
		}

		serviceType := domain.ServiceProduct
		if product.Category == domain.CategorySubscription {
			serviceType = domain.ServiceSubscription
		}
		result.Payment, err = s.paymentRepo.Create(txCtx, domain.CreatePaymentInput{
			ClientID: clientID, AmountCents: amountCents, OperationType: domain.OperationExpense,
			ServiceType: serviceType, Description: "Покупка: " + product.Name,
		})
		if err != nil {
			return err
		}

		switch product.Category {
		case domain.CategorySubscription:
			result.Subscription, err = s.subscriptionRepo.Create(txCtx, clientID, product)
		case domain.CategorySports:
			result.Order, err = s.orderRepo.Create(txCtx, clientID, product)
		default:
			return fmt.Errorf("%w: unsupported product category", domain.ErrInvalidInput)
		}
		return err
	})
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (s *ShopService) GetMySubscriptions(ctx context.Context, clientID int64) ([]*domain.ClientSubscription, error) {
	return s.subscriptionRepo.ListByClient(ctx, clientID)
}

func (s *ShopService) GetMyOrders(ctx context.Context, clientID int64) ([]*domain.Order, error) {
	return s.orderRepo.ListByClient(ctx, clientID)
}

func (s *ShopService) CreateProduct(ctx context.Context, input domain.CreateProductInput) (*domain.Product, error) {
	if err := normalizeProductInput(&input); err != nil {
		return nil, err
	}
	product, err := s.productRepo.Create(ctx, input)
	if err == nil {
		s.invalidateProducts(ctx)
	}
	return product, err
}

func (s *ShopService) UpdateProduct(ctx context.Context, id int64, input domain.CreateProductInput) (*domain.Product, error) {
	if err := normalizeProductInput(&input); err != nil {
		return nil, err
	}
	product, err := s.productRepo.Update(ctx, id, input)
	if err == nil {
		s.invalidateProducts(ctx)
	}
	return product, err
}

func (s *ShopService) DeleteProduct(ctx context.Context, id int64) error {
	// Historical subscriptions/orders/payments must stay valid, so "delete" is a soft deactivate.
	if err := s.productRepo.SetActive(ctx, id, false); err != nil {
		return err
	}
	s.invalidateProducts(ctx)
	return nil
}

// Only the dedicated upload handler supplies a generated photo URL. Product edits
// never rewrite this field, so price/name edits and uploads do not overwrite each other.
func (s *ShopService) SetProductPhoto(ctx context.Context, id int64, photoURL string) (*domain.Product, string, error) {
	var updated *domain.Product
	var oldURL string
	err := s.tx.WithinTransaction(ctx, func(ctx context.Context) error {
		current, err := s.productRepo.GetByIDForUpdate(ctx, id)
		if err != nil {
			return err
		}
		oldURL = current.PhotoURL
		updated, err = s.productRepo.SetPhoto(ctx, id, photoURL)
		return err
	})
	if err == nil {
		s.invalidateProducts(ctx)
	}
	return updated, oldURL, err
}

func normalizeProductInput(input *domain.CreateProductInput) error {
	input.Name = strings.TrimSpace(input.Name)
	input.Description = strings.TrimSpace(input.Description)
	if input.Name == "" || len(input.Name) > 255 {
		return fmt.Errorf("%w: product name is required and must fit 255 chars", domain.ErrInvalidInput)
	}
	price, err := validator.ValidateMoney(input.Price)
	if err != nil {
		return err
	}
	input.Price = price
	switch input.Category {
	case domain.CategorySubscription:
		if input.SubType == nil || input.DurationDays == nil || *input.DurationDays <= 0 || *input.DurationDays > 3650 {
			return fmt.Errorf("%w: subscription requires sub_type and positive duration_days", domain.ErrInvalidInput)
		}
		switch *input.SubType {
		case domain.SubTypeMonthly, domain.SubTypeQuarterly, domain.SubTypeAnnual, domain.SubTypeSingle:
		default:
			return fmt.Errorf("%w: invalid subscription type", domain.ErrInvalidInput)
		}
		if input.SessionsCount != nil && *input.SessionsCount <= 0 {
			return fmt.Errorf("%w: sessions_count must be positive or omitted", domain.ErrInvalidInput)
		}
	case domain.CategorySports:
		input.SubType = nil
		input.DurationDays = nil
		input.SessionsCount = nil
	default:
		return fmt.Errorf("%w: invalid product category", domain.ErrInvalidInput)
	}
	return nil
}

func (s *ShopService) invalidateProducts(ctx context.Context) {
	if s.cache != nil {
		_ = s.cache.Delete(ctx, productsCacheKey)
	}
}

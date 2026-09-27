package service

import (
	"context"
	"errors"
	"testing"

	"github.com/sfedu-crm/internal/domain"
	"github.com/sfedu-crm/internal/repository"
)

func TestFinanceTopUpBalanceTransactionalHappyPath(t *testing.T) {
	var added int64
	var paymentIn domain.CreatePaymentInput
	users := &fakeUserRepo{
		getByIDForUpdateFn: func(context.Context, int64) (*domain.User, error) {
			return &domain.User{ID: 1, Role: domain.RoleClient, IsActive: true}, nil
		},
		addBalanceFn: func(_ context.Context, id, amount int64) error {
			if id != 1 {
				t.Fatalf("bad id")
			}
			added = amount
			return nil
		},
	}
	pays := &fakePaymentRepo{createFn: func(_ context.Context, in domain.CreatePaymentInput) (*domain.Payment, error) {
		paymentIn = in
		return &domain.Payment{ID: 3, ClientID: in.ClientID, Amount: 12.34}, nil
	}}
	tx := &fakeTx{}
	svc := NewFinanceService(pays, users, tx)
	p, err := svc.TopUpBalance(context.Background(), 1, 12.34)
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if p.ID != 3 || added != 1234 || paymentIn.AmountCents != 1234 || paymentIn.OperationType != domain.OperationIncome || paymentIn.ServiceType != domain.ServiceDeposit {
		t.Fatalf("bad topup: p=%+v added=%d in=%+v", p, added, paymentIn)
	}
	if tx.calls != 1 {
		t.Fatalf("expected transaction")
	}
}

func TestFinanceTopUpBalanceRejectsInvalidClientAndMoney(t *testing.T) {
	svc := NewFinanceService(&fakePaymentRepo{}, &fakeUserRepo{getByIDForUpdateFn: func(context.Context, int64) (*domain.User, error) {
		return &domain.User{Role: domain.RoleManager, IsActive: true}, nil
	}}, &fakeTx{})
	if _, err := svc.TopUpBalance(context.Background(), 1, 0); !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("expected invalid money: %v", err)
	}
	if _, err := svc.TopUpBalance(context.Background(), 1, 10); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected forbidden: %v", err)
	}
}

func TestShopClientProductVisibility(t *testing.T) {
	active := false
	var got repository.ProductFilter
	repo := &fakeProductRepo{
		listFn: func(_ context.Context, f repository.ProductFilter) ([]*domain.Product, error) {
			got = f
			return nil, nil
		},
		getByIDFn: func(context.Context, int64) (*domain.Product, error) {
			return &domain.Product{ID: 2, IsActive: false}, nil
		},
	}
	svc := NewShopService(repo, &fakeSubscriptionRepo{}, &fakePaymentRepo{}, &fakeOrderRepo{}, &fakeUserRepo{}, &fakeTx{}, nil)
	_, err := svc.ListProducts(context.Background(), domain.RoleClient, repository.ProductFilter{IsActive: &active})
	if err != nil {
		t.Fatal(err)
	}
	if got.IsActive == nil || !*got.IsActive {
		t.Fatal("client must only see active products")
	}
	if _, err := svc.GetProduct(context.Background(), domain.RoleClient, 2); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("inactive product must be hidden: %v", err)
	}
	if _, err := svc.GetProduct(context.Background(), domain.RoleAdmin, 2); err != nil {
		t.Fatalf("admin may inspect inactive product: %v", err)
	}
}

func TestShopListRejectsInvalidCategory(t *testing.T) {
	cat := domain.ProductCategory("invalid")
	svc := NewShopService(&fakeProductRepo{}, &fakeSubscriptionRepo{}, &fakePaymentRepo{}, &fakeOrderRepo{}, &fakeUserRepo{}, &fakeTx{}, nil)
	if _, err := svc.ListProducts(context.Background(), domain.RoleAdmin, repository.ProductFilter{Category: &cat}); !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("expected invalid input: %v", err)
	}
}

func TestShopPurchaseSubscription(t *testing.T) {
	days, sessions := 30, 8
	subType := domain.SubTypeMonthly
	product := &domain.Product{ID: 9, Name: "8 занятий", Price: 5000, Category: domain.CategorySubscription, SubType: &subType, DurationDays: &days, SessionsCount: &sessions, IsActive: true}
	var debit int64
	var payIn domain.CreatePaymentInput
	var subCreated bool
	svc := NewShopService(
		&fakeProductRepo{getByIDForUpdateFn: func(context.Context, int64) (*domain.Product, error) { return product, nil }},
		&fakeSubscriptionRepo{createFn: func(_ context.Context, id int64, p *domain.Product) (*domain.ClientSubscription, error) {
			subCreated = true
			return &domain.ClientSubscription{ID: 2, ClientID: id, ProductID: p.ID}, nil
		}},
		&fakePaymentRepo{createFn: func(_ context.Context, in domain.CreatePaymentInput) (*domain.Payment, error) {
			payIn = in
			return &domain.Payment{ID: 5}, nil
		}},
		&fakeOrderRepo{},
		&fakeUserRepo{getByIDForUpdateFn: func(context.Context, int64) (*domain.User, error) {
			return &domain.User{ID: 1, Role: domain.RoleClient, IsActive: true}, nil
		}, debitBalanceFn: func(_ context.Context, _ int64, a int64) error { debit = a; return nil }},
		&fakeTx{}, nil,
	)
	out, err := svc.PurchaseProduct(context.Background(), 1, 9)
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if debit != 500000 || payIn.AmountCents != 500000 || payIn.ServiceType != domain.ServiceSubscription || !subCreated || out.Subscription == nil || out.Order != nil {
		t.Fatalf("bad purchase: debit=%d pay=%+v out=%+v", debit, payIn, out)
	}
}

func TestShopPurchaseSportsProduct(t *testing.T) {
	product := &domain.Product{ID: 4, Name: "Перчатки", Price: 1000, Category: domain.CategorySports, IsActive: true}
	orderCreated := false
	svc := NewShopService(
		&fakeProductRepo{getByIDForUpdateFn: func(context.Context, int64) (*domain.Product, error) { return product, nil }},
		&fakeSubscriptionRepo{},
		&fakePaymentRepo{createFn: func(context.Context, domain.CreatePaymentInput) (*domain.Payment, error) {
			return &domain.Payment{ID: 1}, nil
		}},
		&fakeOrderRepo{createFn: func(_ context.Context, id int64, p *domain.Product) (*domain.Order, error) {
			orderCreated = true
			return &domain.Order{ID: 8, ClientID: id, ProductID: p.ID}, nil
		}},
		&fakeUserRepo{getByIDForUpdateFn: func(context.Context, int64) (*domain.User, error) {
			return &domain.User{Role: domain.RoleClient, IsActive: true}, nil
		}},
		&fakeTx{}, nil,
	)
	out, err := svc.PurchaseProduct(context.Background(), 2, 4)
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if !orderCreated || out.Order == nil || out.Subscription != nil {
		t.Fatalf("sports purchase must create order: %+v", out)
	}
}

func TestShopPurchaseRejectsInactiveProductAndClient(t *testing.T) {
	svcInactiveProduct := NewShopService(&fakeProductRepo{getByIDForUpdateFn: func(context.Context, int64) (*domain.Product, error) { return &domain.Product{IsActive: false}, nil }}, &fakeSubscriptionRepo{}, &fakePaymentRepo{}, &fakeOrderRepo{}, &fakeUserRepo{}, &fakeTx{}, nil)
	if _, err := svcInactiveProduct.PurchaseProduct(context.Background(), 1, 1); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected not found: %v", err)
	}

	svcInactiveClient := NewShopService(&fakeProductRepo{getByIDForUpdateFn: func(context.Context, int64) (*domain.Product, error) {
		return &domain.Product{IsActive: true, Price: 100, Category: domain.CategorySports}, nil
	}}, &fakeSubscriptionRepo{}, &fakePaymentRepo{}, &fakeOrderRepo{}, &fakeUserRepo{getByIDForUpdateFn: func(context.Context, int64) (*domain.User, error) {
		return &domain.User{Role: domain.RoleClient, IsActive: false}, nil
	}}, &fakeTx{}, nil)
	if _, err := svcInactiveClient.PurchaseProduct(context.Background(), 1, 1); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected forbidden: %v", err)
	}
}

func TestNormalizeProductInputMatrix(t *testing.T) {
	days, sessions := 30, 8
	monthly := domain.SubTypeMonthly
	validSub := domain.CreateProductInput{Name: " Plan ", Description: " x ", Price: 100.25, Category: domain.CategorySubscription, SubType: &monthly, DurationDays: &days, SessionsCount: &sessions}
	if err := normalizeProductInput(&validSub); err != nil {
		t.Fatalf("valid subscription rejected: %v", err)
	}
	if validSub.Name != "Plan" || validSub.Description != "x" {
		t.Fatalf("not trimmed: %+v", validSub)
	}

	sports := domain.CreateProductInput{Name: "Bottle", Price: 100, Category: domain.CategorySports, SubType: &monthly, DurationDays: &days, SessionsCount: &sessions}
	if err := normalizeProductInput(&sports); err != nil {
		t.Fatal(err)
	}
	if sports.SubType != nil || sports.DurationDays != nil || sports.SessionsCount != nil {
		t.Fatal("sports product must clear subscription fields")
	}

	badCases := []domain.CreateProductInput{
		{Name: "", Price: 10, Category: domain.CategorySports},
		{Name: "x", Price: -1, Category: domain.CategorySports},
		{Name: "x", Price: 10.001, Category: domain.CategorySports},
		{Name: "x", Price: 10, Category: domain.CategorySubscription},
		{Name: "x", Price: 10, Category: domain.ProductCategory("bad")},
	}
	for i, in := range badCases {
		if err := normalizeProductInput(&in); !errors.Is(err, domain.ErrInvalidInput) {
			t.Fatalf("case %d expected invalid input: %v", i, err)
		}
	}
}

func TestShopDeleteIsSoftDeactivate(t *testing.T) {
	var gotID int64
	var gotActive bool = true
	svc := NewShopService(&fakeProductRepo{setActiveFn: func(_ context.Context, id int64, a bool) error { gotID = id; gotActive = a; return nil }}, &fakeSubscriptionRepo{}, &fakePaymentRepo{}, &fakeOrderRepo{}, &fakeUserRepo{}, &fakeTx{}, nil)
	if err := svc.DeleteProduct(context.Background(), 9); err != nil {
		t.Fatal(err)
	}
	if gotID != 9 || gotActive {
		t.Fatalf("delete must deactivate, id=%d active=%v", gotID, gotActive)
	}
}

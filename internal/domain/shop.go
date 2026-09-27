package domain

import "time"

type SubscriptionType string

const (
	SubTypeMonthly   SubscriptionType = "monthly"
	SubTypeQuarterly SubscriptionType = "quarterly"
	SubTypeAnnual    SubscriptionType = "annual"
	SubTypeSingle    SubscriptionType = "single"
)

type ProductCategory string

const (
	CategorySubscription ProductCategory = "subscription"
	CategorySports       ProductCategory = "sports"
)

type Product struct {
	PhotoURL      string            `json:"photo_url"`
	ID            int64             `json:"id"`
	Name          string            `json:"name"`
	Description   string            `json:"description"`
	Price         float64           `json:"price"`
	Category      ProductCategory   `json:"category"`
	SubType       *SubscriptionType `json:"sub_type,omitempty"`
	DurationDays  *int              `json:"duration_days,omitempty"`
	SessionsCount *int              `json:"sessions_count,omitempty"`
	IsActive      bool              `json:"is_active"`
	CreatedAt     time.Time         `json:"created_at"`
}

type ClientSubscription struct {
	ID           int64      `json:"id"`
	ClientID     int64      `json:"client_id"`
	ProductID    int64      `json:"product_id"`
	StartDate    time.Time  `json:"start_date"`
	EndDate      time.Time  `json:"end_date"`
	SessionsLeft *int       `json:"sessions_left,omitempty"`
	IsActive     bool       `json:"is_active"`
	FrozenAt     *time.Time `json:"frozen_at,omitempty"`
	FreezeDays   int        `json:"freeze_days"`
	CreatedAt    time.Time  `json:"created_at"`
	ProductName  string     `json:"product_name,omitempty"`
	Price        float64    `json:"price,omitempty"`
}

type Order struct {
	ID          int64     `json:"id"`
	ClientID    int64     `json:"client_id"`
	ProductID   int64     `json:"product_id"`
	Amount      float64   `json:"amount"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	ClientName  string    `json:"client_name,omitempty"`
	ProductName string    `json:"product_name,omitempty"`
}

type CreateProductInput struct {
	Name          string            `json:"name"`
	Description   string            `json:"description"`
	Price         float64           `json:"price"`
	Category      ProductCategory   `json:"category"`
	SubType       *SubscriptionType `json:"sub_type,omitempty"`
	DurationDays  *int              `json:"duration_days,omitempty"`
	SessionsCount *int              `json:"sessions_count,omitempty"`
}

type PurchaseProductInput struct {
	ProductID int64 `json:"product_id"`
}

type PurchaseResult struct {
	Payment      *Payment            `json:"payment"`
	Subscription *ClientSubscription `json:"subscription,omitempty"`
	Order        *Order              `json:"order,omitempty"`
}

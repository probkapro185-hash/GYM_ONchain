package domain

import "time"

type OperationType string

const (
	OperationIncome  OperationType = "income"
	OperationExpense OperationType = "expense"
	OperationRefund  OperationType = "refund"
)

type ServiceType string

const (
	ServiceSubscription ServiceType = "subscription"
	ServiceTraining     ServiceType = "training"
	ServiceProduct      ServiceType = "product"
	ServiceDeposit      ServiceType = "deposit"
)

type Payment struct {
	ID            int64         `json:"id"`
	ClientID      int64         `json:"client_id"`
	Amount        float64       `json:"amount"`
	OperationType OperationType `json:"operation_type"`
	ServiceType   ServiceType   `json:"service_type"`
	Description   string        `json:"description"`
	CreatedAt     time.Time     `json:"created_at"`
	ClientName    string        `json:"client_name,omitempty"`
}

type PaymentFilter struct {
	ClientID      *int64
	OperationType OperationType
	ServiceType   ServiceType
	DateFrom      *time.Time
	DateTo        *time.Time
}

type FinanceSummary struct {
	TotalIncome  float64 `json:"total_income"`
	TotalExpense float64 `json:"total_expense"`
	TotalRefund  float64 `json:"total_refund"`
	NetBalance   float64 `json:"net_balance"`
}

type CreatePaymentInput struct {
	ClientID      int64         `json:"client_id"`
	AmountCents   int64         `json:"-"`
	OperationType OperationType `json:"operation_type"`
	ServiceType   ServiceType   `json:"service_type"`
	Description   string        `json:"description"`
}

type TopUpBalanceInput struct {
	ClientID int64   `json:"client_id"`
	Amount   float64 `json:"amount"`
}

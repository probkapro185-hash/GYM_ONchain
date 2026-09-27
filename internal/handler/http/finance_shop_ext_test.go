package httphandler

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/sfedu-crm/internal/domain"
)

func TestPaymentFilterFromRequestEmpty(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/finance/payments", nil)
	got, err := paymentFilterFromRequest(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ClientID != nil || got.OperationType != "" || got.ServiceType != "" || got.DateFrom != nil || got.DateTo != nil {
		t.Fatalf("expected empty filter, got %+v", got)
	}
}

func TestPaymentFilterFromRequestAllFields(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/finance/payments?client_id=42&operation_type=income&service_type=subscription&date_from=2026-09-01&date_to=2026-09-15", nil)
	got, err := paymentFilterFromRequest(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ClientID == nil || *got.ClientID != 42 {
		t.Fatalf("client id = %#v", got.ClientID)
	}
	if got.OperationType != domain.OperationIncome {
		t.Fatalf("operation = %q", got.OperationType)
	}
	if got.ServiceType != domain.ServiceSubscription {
		t.Fatalf("service = %q", got.ServiceType)
	}
	wantFrom := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	wantToExclusive := time.Date(2026, 9, 16, 0, 0, 0, 0, time.UTC)
	if got.DateFrom == nil || !got.DateFrom.Equal(wantFrom) {
		t.Fatalf("date from = %v", got.DateFrom)
	}
	if got.DateTo == nil || !got.DateTo.Equal(wantToExclusive) {
		t.Fatalf("date to = %v", got.DateTo)
	}
}

func TestPaymentFilterFromRequestRejectsInvalidValues(t *testing.T) {
	cases := []string{
		"?client_id=0",
		"?client_id=nope",
		"?operation_type=chargeback",
		"?service_type=unknown",
		"?date_from=01.09.2026",
		"?date_to=tomorrow",
		"?date_from=2026-09-20&date_to=2026-09-10",
	}
	for _, query := range cases {
		t.Run(query, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/v1/finance/payments"+query, nil)
			if _, err := paymentFilterFromRequest(req); err == nil {
				t.Fatalf("expected validation error for %s", query)
			}
		})
	}
}

func TestPaymentFilterFromRequestAcceptsEveryEnum(t *testing.T) {
	operations := []domain.OperationType{domain.OperationIncome, domain.OperationExpense, domain.OperationRefund}
	for _, op := range operations {
		req := httptest.NewRequest("GET", "/?operation_type="+string(op), nil)
		got, err := paymentFilterFromRequest(req)
		if err != nil || got.OperationType != op {
			t.Fatalf("op %q: got %+v err=%v", op, got, err)
		}
	}
	services := []domain.ServiceType{domain.ServiceSubscription, domain.ServiceTraining, domain.ServiceProduct, domain.ServiceDeposit}
	for _, svc := range services {
		req := httptest.NewRequest("GET", "/?service_type="+string(svc), nil)
		got, err := paymentFilterFromRequest(req)
		if err != nil || got.ServiceType != svc {
			t.Fatalf("svc %q: got %+v err=%v", svc, got, err)
		}
	}
}

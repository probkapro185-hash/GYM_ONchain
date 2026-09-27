package postgres

import (
	"strings"
	"testing"
	"time"

	"github.com/sfedu-crm/internal/domain"
)

func TestPaymentWhereEmpty(t *testing.T) {
	where, args := paymentWhere(domain.PaymentFilter{})
	if where != "" || len(args) != 0 {
		t.Fatalf("where=%q args=%#v", where, args)
	}
}

func TestPaymentWhereBuildsStablePlaceholders(t *testing.T) {
	clientID := int64(9)
	from := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	where, args := paymentWhere(domain.PaymentFilter{
		ClientID:      &clientID,
		OperationType: domain.OperationIncome,
		ServiceType:   domain.ServiceSubscription,
		DateFrom:      &from,
		DateTo:        &to,
	})
	wants := []string{"p.client_id=$1", "p.operation_type=$2", "p.service_type=$3", "p.created_at >= $4", "p.created_at < $5"}
	for _, want := range wants {
		if !strings.Contains(where, want) {
			t.Fatalf("where %q missing %q", where, want)
		}
	}
	if len(args) != 5 {
		t.Fatalf("args len=%d: %#v", len(args), args)
	}
	if args[0] != clientID || args[1] != domain.OperationIncome || args[2] != domain.ServiceSubscription || args[3] != from || args[4] != to {
		t.Fatalf("unexpected args: %#v", args)
	}
}

package validator

import (
	"testing"

	"github.com/sfedu-crm/internal/domain"
)

func TestValidateMoney(t *testing.T) {
	tests := []struct {
		name    string
		value   float64
		want    float64
		wantErr bool
	}{
		{"integer", 100, 100, false},
		{"two decimals", 99.99, 99.99, false},
		{"zero", 0, 0, true},
		{"negative", -1, 0, true},
		{"too many decimals", 1.001, 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ValidateMoney(tt.value)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("ValidateMoney(%v) expected error", tt.value)
				}
				return
			}
			if err != nil || got != tt.want {
				t.Fatalf("ValidateMoney(%v)=(%v,%v), want (%v,nil)", tt.value, got, err, tt.want)
			}
		})
	}
}

func TestMoneyToCents(t *testing.T) {
	cents, err := MoneyToCents(1234.56)
	if err != nil {
		t.Fatal(err)
	}
	if cents != 123456 {
		t.Fatalf("MoneyToCents=%d, want 123456", cents)
	}
}

func TestValidators(t *testing.T) {
	if err := ValidateEmail("user@gmail.com"); err != nil {
		t.Fatal(err)
	}
	if err := ValidateEmail("user@example.com"); err == nil {
		t.Fatal("expected unsupported email domain error")
	}
	if err := ValidatePhone("+79871234567"); err != nil {
		t.Fatal(err)
	}
	if err := ValidatePhone("123"); err == nil {
		t.Fatal("expected invalid phone error")
	}
	for _, input := range []string{"+7 (987) 123-45-67", "8 987 123 45 67", "+79871234567"} {
		got, err := NormalizePhone(input)
		if err != nil {
			t.Fatalf("NormalizePhone(%q): %v", input, err)
		}
		if got != "+79871234567" {
			t.Fatalf("NormalizePhone(%q)=%q", input, got)
		}
	}
	if _, err := NormalizePhone("+89871234567"); err == nil {
		t.Fatal("+8 prefix must be rejected")
	}
	if err := ValidateFullName("Иван Иванов"); err != nil {
		t.Fatal(err)
	}
	if err := ValidateFullName("Иван"); err == nil {
		t.Fatal("expected full name error")
	}
	if err := ValidatePassword("StrongPass123"); err != nil {
		t.Fatal(err)
	}
	if err := ValidatePassword("short"); err == nil {
		t.Fatal("expected short password error")
	}
	if err := ValidateRole(domain.RoleClient); err != nil {
		t.Fatal(err)
	}
	if err := ValidateRole(domain.Role("owner")); err == nil {
		t.Fatal("expected invalid role")
	}
}

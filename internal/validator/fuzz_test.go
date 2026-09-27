package validator

import (
	"math"
	"strings"
	"testing"
)

func FuzzNormalizePhoneNeverPanics(f *testing.F) {
	for _, seed := range []string{"+7(999)123-45-67", "89991234567", "", "abc", "+79999999999", "+8 999"} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, s string) {
		out, err := NormalizePhone(s)
		if err == nil {
			if len(out) != 12 || !strings.HasPrefix(out, "+7") {
				t.Fatalf("successful normalization broke invariant: %q -> %q", s, out)
			}
			if err := ValidatePhone(out); err != nil {
				t.Fatalf("normalized value must validate: %q: %v", out, err)
			}
		}
	})
}

func FuzzMoneyToCentsRoundTrip(f *testing.F) {
	for _, seed := range []float64{0, 1, 1.01, 999.99, -1, math.MaxFloat64} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, v float64) {
		cents, err := MoneyToCents(v)
		if err == nil {
			if cents <= 0 {
				t.Fatalf("valid money must be positive cents: %v -> %d", v, cents)
			}
			back := float64(cents) / 100
			if math.Abs(back-v) > 0.0000001 {
				t.Fatalf("round trip mismatch: %v -> %d -> %v", v, cents, back)
			}
		}
	})
}

func FuzzValidateFullNameNeverPanics(f *testing.F) {
	for _, seed := range []string{"Иван Иванов", "John Smith", "A", "Иван123 Иванов", ""} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, s string) { _ = ValidateFullName(s) })
}

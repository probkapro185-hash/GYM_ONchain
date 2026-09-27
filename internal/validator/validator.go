package validator

import (
	"fmt"
	"math"
	"net/mail"
	"net/url"
	"strings"
	"unicode/utf8"

	"github.com/sfedu-crm/internal/domain"
)

func NormalizePhone(phone string) (string, error) {
	raw := strings.TrimSpace(phone)
	if raw == "" {
		return "", fmt.Errorf("%w: must be a valid Russian phone number", domain.ErrInvalidPhone)
	}
	var digits strings.Builder
	digits.Grow(11)
	for i, r := range raw {
		switch {
		case r >= '0' && r <= '9':
			digits.WriteRune(r)
		case r == '+':
			if i != 0 {
				return "", fmt.Errorf("%w: must be a valid Russian phone number", domain.ErrInvalidPhone)
			}
		case r == ' ' || r == '-' || r == '(' || r == ')':
			// Accepted formatting characters are removed during canonicalization.
		default:
			return "", fmt.Errorf("%w: must be a valid Russian phone number", domain.ErrInvalidPhone)
		}
	}
	value := digits.String()
	if len(value) != 11 || (value[0] != '7' && value[0] != '8') || (strings.HasPrefix(raw, "+") && value[0] != '7') {
		return "", fmt.Errorf("%w: must be a valid Russian phone number", domain.ErrInvalidPhone)
	}
	if value[0] == '8' {
		value = "7" + value[1:]
	}
	return "+" + value, nil
}

func ValidatePhone(phone string) error {
	_, err := NormalizePhone(phone)
	return err
}

func ValidateEmail(email string) error {
	email = strings.TrimSpace(email)
	addr, err := mail.ParseAddress(email)
	if err != nil || !strings.EqualFold(addr.Address, email) {
		return fmt.Errorf("%w: malformed email", domain.ErrInvalidEmail)
	}
	parts := strings.Split(addr.Address, "@")
	if len(parts) != 2 {
		return fmt.Errorf("%w: malformed email", domain.ErrInvalidEmail)
	}
	domainName := strings.ToLower(parts[1])
	if domainName != "gmail.com" && domainName != "mail.ru" {
		return fmt.Errorf("%w: only @gmail.com and @mail.ru are allowed", domain.ErrInvalidEmail)
	}
	return nil
}

func ValidateFullName(name string) error {
	name = strings.TrimSpace(name)
	parts := strings.Fields(name)
	if len(parts) < 2 || utf8.RuneCountInString(name) > 255 {
		return fmt.Errorf("%w: full name must contain first and last name and fit 255 chars", domain.ErrInvalidInput)
	}
	for _, r := range name {
		if !(r == ' ' || r == '-' || r == '\'' || r == '’' ||
			(r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') ||
			(r >= 'А' && r <= 'я') || r == 'Ё' || r == 'ё') {
			return fmt.Errorf("%w: unsupported character in full name", domain.ErrInvalidInput)
		}
	}
	return nil
}

func ValidatePassword(password string) error {
	if len(password) < 8 || len(password) > 72 {
		return fmt.Errorf("%w: password length must be from 8 to 72 bytes", domain.ErrInvalidInput)
	}
	return nil
}

func ValidateRole(role domain.Role) error {
	switch role {
	case domain.RoleClient, domain.RoleManager, domain.RoleAdmin:
		return nil
	default:
		return fmt.Errorf("%w: invalid role", domain.ErrInvalidInput)
	}
}

func ValidateGender(gender domain.Gender) error {
	switch gender {
	case domain.GenderMale, domain.GenderFemale:
		return nil
	default:
		return fmt.Errorf("%w: invalid gender", domain.ErrInvalidInput)
	}
}

func ValidateSpecialization(v domain.TrainerSpecialization) error {
	switch v {
	case domain.SpecBodyRelief, domain.SpecWeightLoss, domain.SpecMassGain:
		return nil
	default:
		return fmt.Errorf("%w: invalid trainer specialization", domain.ErrInvalidInput)
	}
}

func ValidateTrainingStatus(v domain.TrainingStatus) error {
	switch v {
	case domain.TrainingStatusScheduled, domain.TrainingStatusCompleted, domain.TrainingStatusCancelled:
		return nil
	default:
		return fmt.Errorf("%w: invalid training status", domain.ErrInvalidInput)
	}
}

func ValidateMoney(amount float64) (float64, error) {
	if math.IsNaN(amount) || math.IsInf(amount, 0) || amount <= 0 || amount > 9999999999.99 {
		return 0, fmt.Errorf("%w: amount must be positive and finite", domain.ErrInvalidInput)
	}
	normalized := math.Round(amount*100) / 100
	if math.Abs(amount-normalized) > 0.0000001 {
		return 0, fmt.Errorf("%w: amount supports at most 2 decimal places", domain.ErrInvalidInput)
	}
	return normalized, nil
}

// MoneyToCents validates a public decimal amount and converts it to exact integer cents
// before any balance/payment write. This avoids binary-float arithmetic in financial updates.
func MoneyToCents(amount float64) (int64, error) {
	normalized, err := ValidateMoney(amount)
	if err != nil {
		return 0, err
	}
	return int64(math.Round(normalized * 100)), nil
}

func ValidateHTTPURL(value string) error {
	u, err := url.ParseRequestURI(strings.TrimSpace(value))
	if err != nil || u.Host == "" || u.User != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return fmt.Errorf("%w: URL must be an absolute http/https URL", domain.ErrInvalidInput)
	}
	return nil
}

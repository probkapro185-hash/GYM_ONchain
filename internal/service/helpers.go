package service

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/sfedu-crm/internal/domain"
	"github.com/sfedu-crm/internal/validator"
)

func isNotFound(err error) bool { return errors.Is(err, domain.ErrNotFound) }

func normalizeUserInput(input *domain.CreateUserInput) error {
	input.FullName = strings.TrimSpace(input.FullName)
	normalizedPhone, err := validator.NormalizePhone(input.Phone)
	if err != nil {
		return err
	}
	input.Phone = normalizedPhone
	input.Email = strings.ToLower(strings.TrimSpace(input.Email))
	if input.Role == "" {
		input.Role = domain.RoleClient
	}
	if input.Gender == "" {
		input.Gender = domain.GenderMale
	}
	if err := validator.ValidateFullName(input.FullName); err != nil {
		return err
	}
	if err := validator.ValidatePhone(input.Phone); err != nil {
		return err
	}
	if err := validator.ValidateEmail(input.Email); err != nil {
		return err
	}
	if err := validator.ValidatePassword(input.Password); err != nil {
		return err
	}
	if err := validator.ValidateRole(input.Role); err != nil {
		return err
	}
	return validator.ValidateGender(input.Gender)
}

func normalizeUpdateUserInput(input *domain.UpdateUserInput) error {
	input.FullName = strings.TrimSpace(input.FullName)
	normalizedPhone, err := validator.NormalizePhone(input.Phone)
	if err != nil {
		return err
	}
	input.Phone = normalizedPhone
	input.Email = strings.ToLower(strings.TrimSpace(input.Email))
	if err := validator.ValidateFullName(input.FullName); err != nil {
		return err
	}
	if err := validator.ValidatePhone(input.Phone); err != nil {
		return err
	}
	if err := validator.ValidateEmail(input.Email); err != nil {
		return err
	}
	return validator.ValidateGender(input.Gender)
}

func validateTrainingTimes(start, end time.Time) error {
	if start.IsZero() || end.IsZero() || !end.After(start) {
		return fmt.Errorf("%w: end_time must be after start_time", domain.ErrInvalidInput)
	}
	if end.Sub(start) > 8*time.Hour {
		return fmt.Errorf("%w: training duration is unreasonably long", domain.ErrInvalidInput)
	}
	return nil
}

func canTransitionTraining(from, to domain.TrainingStatus) bool {
	if from == domain.TrainingStatusScheduled {
		return to == domain.TrainingStatusScheduled ||
			to == domain.TrainingStatusCompleted ||
			to == domain.TrainingStatusCancelled
	}
	return false
}

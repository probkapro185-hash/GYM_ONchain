package service

import (
	"context"
	"errors"
	"testing"

	"github.com/sfedu-crm/internal/domain"
)

func TestManagerCannotCreateAdmin(t *testing.T) {
	s := &UserService{}
	_, err := s.CreateUser(context.Background(), domain.RoleManager, domain.CreateUserInput{
		FullName: "Иван Иванов",
		Phone:    "+79871234567",
		Email:    "manager-test@gmail.com",
		Password: "StrongPassword123",
		Role:     domain.RoleAdmin,
		Gender:   domain.GenderMale,
	})
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("error=%v, want forbidden", err)
	}
}

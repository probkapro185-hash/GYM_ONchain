package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/sfedu-crm/internal/domain"
	"github.com/sfedu-crm/pkg/hash"
	jwtpkg "github.com/sfedu-crm/pkg/jwt"
)

func TestAuthSubmitApplicationNormalizesAndCreates(t *testing.T) {
	var got domain.CreateApplicationInput
	users := &fakeUserRepo{
		getByEmailFn: func(context.Context, string) (*domain.User, error) { return nil, domain.ErrNotFound },
		getByPhoneFn: func(context.Context, string) (*domain.User, error) { return nil, domain.ErrNotFound },
	}
	apps := &fakeApplicationRepo{createFn: func(_ context.Context, in domain.CreateApplicationInput) (*domain.ApplicationRequest, error) {
		got = in
		return &domain.ApplicationRequest{ID: 1, FullName: in.FullName, Phone: in.Phone, Email: in.Email, Status: domain.ApplicationPending}, nil
	}}
	svc := NewAuthService(users, apps, jwtpkg.NewManager("12345678901234567890123456789012"), hash.NewBcrypt(4), time.Hour)
	out, err := svc.SubmitApplication(context.Background(), domain.CreateApplicationInput{Gender: domain.GenderMale, FullName: "  Иван Иванов  ", Phone: "8 (999) 123-45-67", Email: "  USER@GMAIL.COM  "})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.ID != 1 {
		t.Fatalf("unexpected id %d", out.ID)
	}
	if got.FullName != "Иван Иванов" {
		t.Fatalf("full name not trimmed: %q", got.FullName)
	}
	if got.Email != "user@gmail.com" {
		t.Fatalf("email not normalized: %q", got.Email)
	}
	if got.Phone != "+79991234567" {
		t.Fatalf("phone not normalized: %q", got.Phone)
	}
}

func TestAuthSubmitApplicationDuplicateEmail(t *testing.T) {
	users := &fakeUserRepo{getByEmailFn: func(context.Context, string) (*domain.User, error) { return &domain.User{ID: 1}, nil }}
	svc := NewAuthService(users, &fakeApplicationRepo{}, jwtpkg.NewManager("12345678901234567890123456789012"), hash.NewBcrypt(4), time.Hour)
	_, err := svc.SubmitApplication(context.Background(), domain.CreateApplicationInput{Gender: domain.GenderMale, FullName: "Иван Иванов", Phone: "+79991234567", Email: "u@gmail.com"})
	if !errors.Is(err, domain.ErrAlreadyExists) {
		t.Fatalf("expected already exists, got %v", err)
	}
}

func TestAuthSubmitApplicationDuplicatePhone(t *testing.T) {
	users := &fakeUserRepo{
		getByEmailFn: func(context.Context, string) (*domain.User, error) { return nil, domain.ErrNotFound },
		getByPhoneFn: func(context.Context, string) (*domain.User, error) { return &domain.User{ID: 2}, nil },
	}
	svc := NewAuthService(users, &fakeApplicationRepo{}, jwtpkg.NewManager("12345678901234567890123456789012"), hash.NewBcrypt(4), time.Hour)
	_, err := svc.SubmitApplication(context.Background(), domain.CreateApplicationInput{Gender: domain.GenderMale, FullName: "Иван Иванов", Phone: "+79991234567", Email: "u@gmail.com"})
	if !errors.Is(err, domain.ErrAlreadyExists) {
		t.Fatalf("expected already exists, got %v", err)
	}
}

func TestAuthSubmitApplicationRejectsRepositoryFailure(t *testing.T) {
	sentinel := errors.New("db down")
	users := &fakeUserRepo{getByEmailFn: func(context.Context, string) (*domain.User, error) { return nil, sentinel }}
	svc := NewAuthService(users, &fakeApplicationRepo{}, jwtpkg.NewManager("12345678901234567890123456789012"), hash.NewBcrypt(4), time.Hour)
	_, err := svc.SubmitApplication(context.Background(), domain.CreateApplicationInput{Gender: domain.GenderMale, FullName: "Иван Иванов", Phone: "+79991234567", Email: "u@gmail.com"})
	if !errors.Is(err, sentinel) {
		t.Fatalf("expected repository error, got %v", err)
	}
}

func TestAuthLoginSuccessAndClaims(t *testing.T) {
	h := hash.NewBcrypt(4)
	ph, err := h.Hash("GoodPass123!")
	if err != nil {
		t.Fatal(err)
	}
	users := &fakeUserRepo{getByEmailFn: func(_ context.Context, email string) (*domain.User, error) {
		if email != "user@gmail.com" {
			t.Fatalf("email not normalized: %q", email)
		}
		return &domain.User{ID: 42, PasswordHash: ph, Role: domain.RoleClient, TokenVersion: 7, IsActive: true}, nil
	}}
	tm := jwtpkg.NewManager("12345678901234567890123456789012")
	svc := NewAuthService(users, &fakeApplicationRepo{}, tm, h, time.Hour)
	token, user, err := svc.Login(context.Background(), domain.LoginInput{Email: " USER@GMAIL.COM ", Password: "GoodPass123!"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user.ID != 42 || token == "" {
		t.Fatalf("bad login result")
	}
	claims, err := tm.Parse(token)
	if err != nil {
		t.Fatalf("parse token: %v", err)
	}
	if claims.UserID != 42 || claims.Role != "client" || claims.TokenVersion != 7 {
		t.Fatalf("unexpected claims: %+v", claims)
	}
}

func TestAuthLoginRejectsBadInputs(t *testing.T) {
	h := hash.NewBcrypt(4)
	ph, _ := h.Hash("GoodPass123!")
	cases := []struct {
		name    string
		in      domain.LoginInput
		user    *domain.User
		repoErr error
	}{
		{"empty email", domain.LoginInput{Password: "x"}, nil, nil},
		{"empty password", domain.LoginInput{Email: "x@gmail.com"}, nil, nil},
		{"missing user", domain.LoginInput{Email: "x@gmail.com", Password: "x"}, nil, domain.ErrNotFound},
		{"bad password", domain.LoginInput{Email: "x@gmail.com", Password: "wrong"}, &domain.User{ID: 1, PasswordHash: ph, IsActive: true}, nil},
		{"inactive", domain.LoginInput{Email: "x@gmail.com", Password: "GoodPass123!"}, &domain.User{ID: 1, PasswordHash: ph, IsActive: false}, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := &fakeUserRepo{getByEmailFn: func(context.Context, string) (*domain.User, error) { return tc.user, tc.repoErr }}
			svc := NewAuthService(repo, &fakeApplicationRepo{}, jwtpkg.NewManager("12345678901234567890123456789012"), h, time.Hour)
			token, user, err := svc.Login(context.Background(), tc.in)
			if !errors.Is(err, domain.ErrUnauthorized) {
				t.Fatalf("expected unauthorized, got %v", err)
			}
			if token != "" || user != nil {
				t.Fatalf("unexpected successful login")
			}
		})
	}
}

package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/sfedu-crm/internal/domain"
	"github.com/sfedu-crm/internal/repository"
	"github.com/sfedu-crm/internal/validator"
	"github.com/sfedu-crm/pkg/hash"
	"github.com/sfedu-crm/pkg/jwt"
)

type AuthService struct {
	userRepo repository.UserRepository
	appRepo  repository.ApplicationRepository
	tokenMgr *jwt.Manager
	hasher   *hash.Bcrypt
	tokenTTL time.Duration
}

func NewAuthService(userRepo repository.UserRepository, appRepo repository.ApplicationRepository,
	tokenMgr *jwt.Manager, hasher *hash.Bcrypt, tokenTTL time.Duration) *AuthService {
	return &AuthService{userRepo: userRepo, appRepo: appRepo, tokenMgr: tokenMgr, hasher: hasher, tokenTTL: tokenTTL}
}

func (s *AuthService) SubmitApplication(ctx context.Context, input domain.CreateApplicationInput) (*domain.ApplicationRequest, error) {
	if err := validator.ValidateGender(input.Gender); err != nil {
		return nil, err
	}
	input.FullName = strings.TrimSpace(input.FullName)
	normalizedPhone, err := validator.NormalizePhone(input.Phone)
	if err != nil {
		return nil, err
	}
	input.Phone = normalizedPhone
	input.Email = strings.ToLower(strings.TrimSpace(input.Email))
	if err := validator.ValidateFullName(input.FullName); err != nil {
		return nil, err
	}
	if err := validator.ValidatePhone(input.Phone); err != nil {
		return nil, err
	}
	if err := validator.ValidateEmail(input.Email); err != nil {
		return nil, err
	}

	if _, err := s.userRepo.GetByEmail(ctx, input.Email); err == nil {
		return nil, fmt.Errorf("%w: user with this email already exists", domain.ErrAlreadyExists)
	} else if !isNotFound(err) {
		return nil, err
	}
	if _, err := s.userRepo.GetByPhone(ctx, input.Phone); err == nil {
		return nil, fmt.Errorf("%w: user with this phone already exists", domain.ErrAlreadyExists)
	} else if !isNotFound(err) {
		return nil, err
	}
	return s.appRepo.Create(ctx, input)
}

func (s *AuthService) Login(ctx context.Context, input domain.LoginInput) (string, *domain.User, error) {
	email := strings.ToLower(strings.TrimSpace(input.Email))
	if email == "" || input.Password == "" {
		return "", nil, domain.ErrUnauthorized
	}
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return "", nil, domain.ErrUnauthorized
	}
	if !s.hasher.Compare(user.PasswordHash, input.Password) {
		return "", nil, domain.ErrUnauthorized
	}
	if !user.IsActive {
		return "", nil, domain.ErrUnauthorized
	}
	token, err := s.tokenMgr.Generate(jwt.Claims{UserID: user.ID, Role: string(user.Role), TokenVersion: user.TokenVersion}, s.tokenTTL)
	if err != nil {
		return "", nil, fmt.Errorf("generate token: %w", err)
	}
	return token, user, nil
}

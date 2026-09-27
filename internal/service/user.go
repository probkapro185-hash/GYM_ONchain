package service

import (
	"context"
	"fmt"

	"github.com/sfedu-crm/internal/domain"
	"github.com/sfedu-crm/internal/repository"
	"github.com/sfedu-crm/internal/validator"
	pkgcache "github.com/sfedu-crm/pkg/cache"
	"github.com/sfedu-crm/pkg/hash"
)

const (
	usersCacheKey = "users:list"
)

type UserService struct {
	userRepo repository.UserRepository
	appRepo  repository.ApplicationRepository
	tx       repository.TransactionManager
	hasher   *hash.Bcrypt
	cache    *pkgcache.RedisCache
}

func NewUserService(userRepo repository.UserRepository, appRepo repository.ApplicationRepository,
	tx repository.TransactionManager, hasher *hash.Bcrypt, cache *pkgcache.RedisCache) *UserService {
	return &UserService{userRepo: userRepo, appRepo: appRepo, tx: tx, hasher: hasher, cache: cache}
}

func (s *UserService) GetByID(ctx context.Context, id int64) (*domain.User, error) {
	return s.userRepo.GetByID(ctx, id)
}

func (s *UserService) GetForActor(ctx context.Context, actorRole domain.Role, targetID int64) (*domain.User, error) {
	u, err := s.userRepo.GetByID(ctx, targetID)
	if err != nil {
		return nil, err
	}
	if actorRole == domain.RoleManager && u.Role != domain.RoleClient {
		return nil, domain.ErrForbidden
	}
	return u, nil
}

func (s *UserService) List(ctx context.Context, actorRole domain.Role, filter repository.UserFilter) ([]*domain.User, error) {
	if actorRole == domain.RoleManager {
		if filter.Role != nil && *filter.Role != domain.RoleClient {
			return nil, domain.ErrForbidden
		}
		client := domain.RoleClient
		filter.Role = &client
	}

	return s.userRepo.List(ctx, filter)
}

func (s *UserService) UpdateProfile(ctx context.Context, id int64, input domain.UpdateUserInput) (*domain.User, error) {
	if err := normalizeUpdateUserInput(&input); err != nil {
		return nil, err
	}
	user, err := s.userRepo.Update(ctx, id, input)
	if err == nil {
		s.invalidateUsers(ctx)
	}
	return user, err
}

func (s *UserService) UpdateForActor(ctx context.Context, actorRole domain.Role, targetID int64, input domain.UpdateUserInput) (*domain.User, error) {
	if _, err := s.GetForActor(ctx, actorRole, targetID); err != nil {
		return nil, err
	}
	return s.UpdateProfile(ctx, targetID, input)
}

func (s *UserService) ChangePassword(ctx context.Context, id int64, input domain.ChangePasswordInput) error {
	if err := validator.ValidatePassword(input.NewPassword); err != nil {
		return err
	}
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if !s.hasher.Compare(user.PasswordHash, input.OldPassword) {
		return domain.ErrInvalidPassword
	}
	newHash, err := s.hasher.Hash(input.NewPassword)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}
	return s.userRepo.UpdatePassword(ctx, id, newHash)
}

func (s *UserService) AdminResetPassword(ctx context.Context, targetUserID int64, newPassword string) error {
	if err := validator.ValidatePassword(newPassword); err != nil {
		return err
	}
	user, err := s.userRepo.GetByID(ctx, targetUserID)
	if err != nil {
		return err
	}
	if user.PasswordSetupRequired {
		return fmt.Errorf("%w: client must finish account activation first", domain.ErrConflict)
	}
	newHash, err := s.hasher.Hash(newPassword)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}
	return s.userRepo.UpdatePassword(ctx, targetUserID, newHash)
}

func (s *UserService) CreateUser(ctx context.Context, actorRole domain.Role, input domain.CreateUserInput) (*domain.User, error) {
	if err := normalizeUserInput(&input); err != nil {
		return nil, err
	}
	if input.Role == domain.RoleClient {
		return nil, fmt.Errorf("%w: clients must be created through /users/invite", domain.ErrForbidden)
	}
	if actorRole != domain.RoleAdmin {
		return nil, domain.ErrForbidden
	}

	passwordHash, err := s.hasher.Hash(input.Password)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}
	input.Password = passwordHash
	user, err := s.userRepo.Create(ctx, input)
	if err == nil {
		s.invalidateUsers(ctx)
	}
	return user, err
}

func (s *UserService) RejectApplication(ctx context.Context, appID int64) error {
	return s.tx.WithinTransaction(ctx, func(txCtx context.Context) error {
		app, err := s.appRepo.GetByIDForUpdate(txCtx, appID)
		if err != nil {
			return err
		}
		if app.Status != domain.ApplicationPending {
			return fmt.Errorf("%w: application is already processed", domain.ErrConflict)
		}
		return s.appRepo.UpdateStatus(txCtx, appID, domain.ApplicationRejected)
	})
}

func (s *UserService) ListApplications(ctx context.Context, status domain.ApplicationStatus) ([]*domain.ApplicationRequest, error) {
	if status != "" && status != domain.ApplicationPending && status != domain.ApplicationApproved && status != domain.ApplicationRejected {
		return nil, fmt.Errorf("%w: invalid application status", domain.ErrInvalidInput)
	}
	return s.appRepo.List(ctx, status)
}

func (s *UserService) DeleteUser(ctx context.Context, actorID, targetID int64) error {
	if actorID == targetID {
		return fmt.Errorf("%w: cannot deactivate your own account", domain.ErrForbidden)
	}
	// Keep training/payment/subscription history intact. DELETE is implemented as account deactivation.
	if err := s.userRepo.SetActive(ctx, targetID, false); err != nil {
		return err
	}
	s.invalidateUsers(ctx)
	return nil
}

func (s *UserService) SetActive(ctx context.Context, actorID, targetID int64, active bool) error {
	if actorID == targetID && !active {
		return fmt.Errorf("%w: cannot deactivate your own account", domain.ErrForbidden)
	}
	if active {
		user, err := s.userRepo.GetByID(ctx, targetID)
		if err != nil {
			return err
		}
		if user.PasswordSetupRequired {
			return fmt.Errorf("%w: client must finish account activation first", domain.ErrConflict)
		}
	}
	if err := s.userRepo.SetActive(ctx, targetID, active); err != nil {
		return err
	}
	s.invalidateUsers(ctx)
	return nil
}

func (s *UserService) invalidateUsers(ctx context.Context) {
	if s.cache != nil {
		_ = s.cache.Delete(ctx, usersCacheKey)
		// Trainer list contains user full_name/is_active, so user mutations invalidate it too.
		_ = s.cache.Delete(ctx, trainersCacheKey)
	}
}

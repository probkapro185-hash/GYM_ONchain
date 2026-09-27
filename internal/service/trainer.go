package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/sfedu-crm/internal/domain"
	"github.com/sfedu-crm/internal/repository"
	"github.com/sfedu-crm/internal/validator"
	pkgcache "github.com/sfedu-crm/pkg/cache"
)

const (
	trainersCacheKey = "trainers:list"
	trainersTTL      = 5 * time.Minute
)

type TrainerService struct {
	trainerRepo repository.TrainerRepository
	userRepo    repository.UserRepository
	cache       *pkgcache.RedisCache
}

func NewTrainerService(trainerRepo repository.TrainerRepository, userRepo repository.UserRepository,
	cache *pkgcache.RedisCache) *TrainerService {
	return &TrainerService{trainerRepo: trainerRepo, userRepo: userRepo, cache: cache}
}

func (s *TrainerService) List(ctx context.Context, filter repository.TrainerFilter) ([]*domain.Trainer, error) {
	if filter.Specialization != nil {
		if err := validator.ValidateSpecialization(*filter.Specialization); err != nil {
			return nil, err
		}
	}
	useCache := s.cache != nil && filter.Specialization == nil && filter.IsActive == nil && strings.TrimSpace(filter.Search) == ""
	if useCache {
		var cached []*domain.Trainer
		if err := s.cache.Get(ctx, trainersCacheKey, &cached); err == nil {
			return cached, nil
		}
	}
	trainers, err := s.trainerRepo.List(ctx, filter)
	if err == nil && useCache {
		_ = s.cache.Set(ctx, trainersCacheKey, trainers, trainersTTL)
	}
	return trainers, err
}

func (s *TrainerService) GetByID(ctx context.Context, id int64) (*domain.Trainer, error) {
	return s.trainerRepo.GetByID(ctx, id)
}

func (s *TrainerService) Create(ctx context.Context, input domain.CreateTrainerInput) (*domain.Trainer, error) {
	if input.UserID <= 0 {
		return nil, fmt.Errorf("%w: user_id is required", domain.ErrInvalidInput)
	}
	input.Bio = strings.TrimSpace(input.Bio)
	input.PhotoURL = strings.TrimSpace(input.PhotoURL)
	if err := validateTrainerFields(input.Specialization, input.Bio, input.PhotoURL, input.ExperienceYears); err != nil {
		return nil, err
	}
	user, err := s.userRepo.GetByID(ctx, input.UserID)
	if err != nil {
		return nil, err
	}
	if !user.IsActive {
		return nil, domain.ErrForbidden
	}
	trainer, err := s.trainerRepo.Create(ctx, input)
	if err == nil {
		s.invalidate(ctx)
	}
	return trainer, err
}

func (s *TrainerService) Update(ctx context.Context, id int64, input domain.UpdateTrainerInput) (*domain.Trainer, error) {
	input.Bio = strings.TrimSpace(input.Bio)
	input.PhotoURL = strings.TrimSpace(input.PhotoURL)
	if err := validateTrainerFields(input.Specialization, input.Bio, input.PhotoURL, input.ExperienceYears); err != nil {
		return nil, err
	}
	trainer, err := s.trainerRepo.Update(ctx, id, input)
	if err == nil {
		s.invalidate(ctx)
	}
	return trainer, err
}

func (s *TrainerService) Delete(ctx context.Context, id int64) error {
	current, err := s.trainerRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	_, err = s.trainerRepo.Update(ctx, id, domain.UpdateTrainerInput{
		Specialization:  current.Specialization,
		Bio:             current.Bio,
		PhotoURL:        current.PhotoURL,
		ExperienceYears: current.ExperienceYears,
		IsActive:        false,
	})
	if err != nil {
		return err
	}
	s.invalidate(ctx)
	return nil
}

func validateTrainerFields(spec domain.TrainerSpecialization, bio, photoURL string, experience int) error {
	if experience < 0 || experience > 80 {
		return fmt.Errorf("%w: invalid experience_years", domain.ErrInvalidInput)
	}
	if len(bio) > 5000 {
		return fmt.Errorf("%w: bio is too long", domain.ErrInvalidInput)
	}
	if len(photoURL) > 500 {
		return fmt.Errorf("%w: photo_url is too long", domain.ErrInvalidInput)
	}
	if err := validator.ValidateSpecialization(spec); err != nil {
		return err
	}
	if photoURL != "" && !validLocalTrainerPhotoURL(photoURL) {
		if err := validator.ValidateHTTPURL(photoURL); err != nil {
			return err
		}
	}
	return nil
}

func validLocalTrainerPhotoURL(value string) bool {
	const prefix = "/uploads/trainers/"
	if !strings.HasPrefix(value, prefix) {
		return false
	}
	name := strings.TrimPrefix(value, prefix)
	if strings.Contains(name, "/") || strings.Contains(name, "\\") || strings.Contains(name, "..") {
		return false
	}
	dot := strings.LastIndexByte(name, '.')
	if dot != 24 {
		return false
	}
	ext := strings.ToLower(name[dot:])
	if ext != ".webp" && ext != ".jpg" {
		return false
	}
	for _, c := range name[:dot] {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			return false
		}
	}
	return true
}

func (s *TrainerService) invalidate(ctx context.Context) {
	if s.cache != nil {
		_ = s.cache.Delete(ctx, trainersCacheKey)
	}
}

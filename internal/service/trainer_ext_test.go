package service

import (
	"context"
	"errors"
	"testing"

	"github.com/sfedu-crm/internal/domain"
	"github.com/sfedu-crm/internal/repository"
)

func TestTrainerListValidatesSpecialization(t *testing.T) {
	bad := domain.TrainerSpecialization("crossfit")
	svc := NewTrainerService(&fakeTrainerRepo{}, &fakeUserRepo{}, nil)
	if _, err := svc.List(context.Background(), repository.TrainerFilter{Specialization: &bad}); !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("expected invalid input, got %v", err)
	}
}

func TestTrainerCreateValidatesAndNormalizes(t *testing.T) {
	var got domain.CreateTrainerInput
	tr := &fakeTrainerRepo{createFn: func(_ context.Context, i domain.CreateTrainerInput) (*domain.Trainer, error) {
		got = i
		return &domain.Trainer{ID: 3, UserID: i.UserID, Specialization: i.Specialization, IsActive: true}, nil
	}}
	users := &fakeUserRepo{getByIDFn: func(context.Context, int64) (*domain.User, error) { return &domain.User{ID: 4, IsActive: true}, nil }}
	svc := NewTrainerService(tr, users, nil)
	out, err := svc.Create(context.Background(), domain.CreateTrainerInput{UserID: 4, Specialization: domain.SpecMassGain, Bio: "  опытный тренер  ", PhotoURL: " https://example.com/a.jpg ", ExperienceYears: 5})
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if out.ID != 3 || got.Bio != "опытный тренер" || got.PhotoURL != "https://example.com/a.jpg" {
		t.Fatalf("normalization failed: %+v", got)
	}
}

func TestTrainerCreateRejectsInactiveUserAndBadFields(t *testing.T) {
	users := &fakeUserRepo{getByIDFn: func(context.Context, int64) (*domain.User, error) { return &domain.User{ID: 2, IsActive: false}, nil }}
	svc := NewTrainerService(&fakeTrainerRepo{}, users, nil)
	valid := domain.CreateTrainerInput{UserID: 2, Specialization: domain.SpecBodyRelief, ExperienceYears: 1}
	if _, err := svc.Create(context.Background(), valid); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected forbidden: %v", err)
	}
	valid.UserID = 0
	if _, err := svc.Create(context.Background(), valid); !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("expected invalid user id: %v", err)
	}
	valid.UserID = 2
	valid.ExperienceYears = 81
	if _, err := svc.Create(context.Background(), valid); !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("expected invalid experience: %v", err)
	}
	valid.ExperienceYears = 1
	valid.PhotoURL = "javascript:alert(1)"
	if _, err := svc.Create(context.Background(), valid); !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("expected invalid URL: %v", err)
	}
}

func TestTrainerUpdateAndSoftDelete(t *testing.T) {
	current := &domain.Trainer{ID: 7, Specialization: domain.SpecWeightLoss, Bio: "bio", PhotoURL: "https://example.com/x.jpg", ExperienceYears: 4, IsActive: true}
	var updates []domain.UpdateTrainerInput
	repo := &fakeTrainerRepo{
		getByIDFn: func(context.Context, int64) (*domain.Trainer, error) { cp := *current; return &cp, nil },
		updateFn: func(_ context.Context, id int64, in domain.UpdateTrainerInput) (*domain.Trainer, error) {
			updates = append(updates, in)
			return &domain.Trainer{ID: id, Specialization: in.Specialization, IsActive: in.IsActive}, nil
		},
	}
	svc := NewTrainerService(repo, &fakeUserRepo{}, nil)
	_, err := svc.Update(context.Background(), 7, domain.UpdateTrainerInput{Specialization: domain.SpecMassGain, Bio: " new ", ExperienceYears: 6, IsActive: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(updates) != 1 || updates[0].Bio != "new" {
		t.Fatalf("bad update: %+v", updates)
	}
	if err := svc.Delete(context.Background(), 7); err != nil {
		t.Fatal(err)
	}
	if len(updates) != 2 || updates[1].IsActive {
		t.Fatalf("delete must soft deactivate: %+v", updates)
	}
	if updates[1].Specialization != current.Specialization || updates[1].Bio != current.Bio {
		t.Fatal("soft delete must preserve fields")
	}
}

func TestTrainerUpdateAcceptsManagedLocalPhotoURL(t *testing.T) {
	var got domain.UpdateTrainerInput
	repo := &fakeTrainerRepo{updateFn: func(_ context.Context, id int64, in domain.UpdateTrainerInput) (*domain.Trainer, error) {
		got = in
		return &domain.Trainer{ID: id, PhotoURL: in.PhotoURL, Specialization: in.Specialization, IsActive: in.IsActive}, nil
	}}
	svc := NewTrainerService(repo, &fakeUserRepo{}, nil)
	photoURL := "/uploads/trainers/0123456789abcdef01234567.webp"
	_, err := svc.Update(context.Background(), 9, domain.UpdateTrainerInput{
		Specialization: domain.SpecMassGain, PhotoURL: photoURL, ExperienceYears: 4, IsActive: true,
	})
	if err != nil {
		t.Fatalf("managed photo URL should be accepted: %v", err)
	}
	if got.PhotoURL != photoURL {
		t.Fatalf("photo URL changed: %q", got.PhotoURL)
	}
}

func TestTrainerUpdateRejectsUnsafeLocalPhotoURL(t *testing.T) {
	svc := NewTrainerService(&fakeTrainerRepo{}, &fakeUserRepo{}, nil)
	for _, photoURL := range []string{
		"/uploads/trainers/../../secret.jpg",
		"/uploads/trainers/not-random.webp",
		"/uploads/trainers/0123456789abcdef01234567.png",
	} {
		_, err := svc.Update(context.Background(), 9, domain.UpdateTrainerInput{
			Specialization: domain.SpecMassGain, PhotoURL: photoURL, ExperienceYears: 4, IsActive: true,
		})
		if !errors.Is(err, domain.ErrInvalidInput) {
			t.Fatalf("expected invalid input for %q, got %v", photoURL, err)
		}
	}
}

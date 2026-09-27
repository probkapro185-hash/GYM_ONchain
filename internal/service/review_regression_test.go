package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/sfedu-crm/internal/domain"
	"github.com/sfedu-crm/pkg/hash"
)

func TestDirectUserCreationCannotBypassClientActivation(t *testing.T) {
	repo := &fakeUserRepo{createFn: func(context.Context, domain.CreateUserInput) (*domain.User, error) {
		t.Fatal("must use invitation flow")
		return nil, nil
	}}
	svc := NewUserService(repo, &fakeApplicationRepo{}, &fakeTx{}, hash.NewBcrypt(4), nil)
	for _, actor := range []domain.Role{domain.RoleAdmin, domain.RoleManager, domain.RoleClient} {
		_, err := svc.CreateUser(context.Background(), actor, domain.CreateUserInput{FullName: "Иван Иванов", Phone: "+79991112233", Email: "client@gmail.com", Password: "password1", Role: domain.RoleClient, Gender: domain.GenderMale})
		if !errors.Is(err, domain.ErrForbidden) {
			t.Fatalf("actor %s: %v", actor, err)
		}
	}
}

func TestTaskUpdateScopesManagerToAssignee(t *testing.T) {
	var scope *int64
	called := false
	repo := &fakeCRMRepo{updateTaskStatusFn: func(_ context.Context, id int64, status string, assigneeID *int64) (*domain.StaffTask, error) {
		called = true
		scope = assigneeID
		return &domain.StaffTask{ID: id}, nil
	}}
	svc := newCRMService(repo, nil, nil, nil)
	if _, err := svc.UpdateTaskStatus(context.Background(), 42, domain.RoleManager, 9, "done"); err != nil {
		t.Fatal(err)
	}
	if scope == nil || *scope != 42 {
		t.Fatal("manager update lacks assignee restriction")
	}
	if _, err := svc.UpdateTaskStatus(context.Background(), 1, domain.RoleAdmin, 9, "done"); err != nil {
		t.Fatal(err)
	}
	if scope != nil {
		t.Fatal("admin must be able to update any task")
	}
	called = false
	if _, err := svc.UpdateTaskStatus(context.Background(), 3, domain.RoleClient, 9, "done"); !errors.Is(err, domain.ErrForbidden) || called {
		t.Fatalf("client update: %v", err)
	}
}

func TestApprovalUsesRequestedTrainerUnlessExplicitlyChanged(t *testing.T) {
	preferred := time.Now().Add(24 * time.Hour)
	requested, replacement := int64(9), int64(10)
	for _, tc := range []struct {
		name     string
		override *int64
		clear    bool
		want     *int64
	}{
		{"preserve", nil, false, &requested}, {"replace", &replacement, false, &replacement}, {"remove", nil, true, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := &fakeTrainingRequestRepo{getByIDForUpdateFn: func(context.Context, int64) (*domain.TrainingRequest, error) {
				return &domain.TrainingRequest{ClientID: 1, TrainerID: &requested, PreferredAt: preferred, Status: domain.TrainingRequestPending}, nil
			}}
			trainers := &fakeTrainerRepo{getByIDFn: func(_ context.Context, id int64) (*domain.Trainer, error) {
				return &domain.Trainer{ID: id, IsActive: true}, nil
			}}
			subs := &fakeSubscriptionRepo{getCoveringByClientFn: func(context.Context, int64, time.Time, time.Time) (*domain.ClientSubscription, error) {
				return &domain.ClientSubscription{ID: 2, ClientID: 1, IsActive: true}, nil
			}}
			tr := &fakeTrainingRepo{createFn: func(_ context.Context, in domain.CreateTrainingInput) (*domain.Training, error) {
				if (in.TrainerID == nil) != (tc.want == nil) || (tc.want != nil && *in.TrainerID != *tc.want) {
					t.Fatalf("wrong trainer: %+v", in.TrainerID)
				}
				return &domain.Training{ID: 1}, nil
			}}
			svc := newScheduleSvc(tr, req, activeClientRepo(), trainers, subs, nil)
			if _, err := svc.ApproveRequest(context.Background(), 1, domain.CreateTrainingInput{Title: "Тренировка", TrainerID: tc.override, ClearTrainer: tc.clear}); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestLegacyApplicationRequiresExplicitGender(t *testing.T) {
	apps := &fakeApplicationRepo{getByIDForUpdateFn: func(context.Context, int64) (*domain.ApplicationRequest, error) {
		return &domain.ApplicationRequest{ID: 1, FullName: "Анна Иванова", Email: "anna@gmail.com", Phone: "+79991112233", Status: domain.ApplicationPending}, nil
	}}
	access := &fakeAccountAccessRepo{createPendingFn: func(_ context.Context, in domain.CreateUserInput) (*domain.User, error) {
		if in.Gender != domain.GenderFemale {
			t.Fatalf("gender not preserved: %s", in.Gender)
		}
		return &domain.User{ID: 1, Email: in.Email}, nil
	}}
	svc := newAccountSvc(&fakeUserRepo{}, apps, access, &captureMailer{})
	if _, _, err := svc.ApproveApplication(context.Background(), 1); !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("missing gender: %v", err)
	}
	if _, _, err := svc.ApproveApplication(context.Background(), 1, domain.GenderFemale); err != nil {
		t.Fatal(err)
	}
}

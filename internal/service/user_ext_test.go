package service

import (
	"context"
	"errors"
	"testing"

	"github.com/sfedu-crm/internal/domain"
	"github.com/sfedu-crm/internal/repository"
	"github.com/sfedu-crm/pkg/hash"
)

func TestUserGetForActorManagerRestrictions(t *testing.T) {
	repo := &fakeUserRepo{getByIDFn: func(_ context.Context, id int64) (*domain.User, error) {
		if id == 1 {
			return &domain.User{ID: 1, Role: domain.RoleClient}, nil
		}
		return &domain.User{ID: id, Role: domain.RoleManager}, nil
	}}
	svc := NewUserService(repo, &fakeApplicationRepo{}, &fakeTx{}, hash.NewBcrypt(4), nil)
	if _, err := svc.GetForActor(context.Background(), domain.RoleManager, 1); err != nil {
		t.Fatalf("manager should read client: %v", err)
	}
	if _, err := svc.GetForActor(context.Background(), domain.RoleManager, 2); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected forbidden, got %v", err)
	}
	if _, err := svc.GetForActor(context.Background(), domain.RoleAdmin, 2); err != nil {
		t.Fatalf("admin should read manager: %v", err)
	}
}

func TestUserListManagerForcedToClients(t *testing.T) {
	var got repository.UserFilter
	repo := &fakeUserRepo{listFn: func(_ context.Context, f repository.UserFilter) ([]*domain.User, error) {
		got = f
		return []*domain.User{{ID: 1, Role: domain.RoleClient}}, nil
	}}
	svc := NewUserService(repo, &fakeApplicationRepo{}, &fakeTx{}, hash.NewBcrypt(4), nil)
	_, err := svc.List(context.Background(), domain.RoleManager, repository.UserFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if got.Role == nil || *got.Role != domain.RoleClient {
		t.Fatalf("manager filter not forced to client: %+v", got)
	}
	manager := domain.RoleManager
	_, err = svc.List(context.Background(), domain.RoleManager, repository.UserFilter{Role: &manager})
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected forbidden, got %v", err)
	}
}

func TestUserCreateUserRoleBoundariesAndNormalization(t *testing.T) {
	var got domain.CreateUserInput
	repo := &fakeUserRepo{createFn: func(_ context.Context, i domain.CreateUserInput) (*domain.User, error) {
		got = i
		return &domain.User{ID: 1, FullName: i.FullName, Phone: i.Phone, Email: i.Email, Role: i.Role, Gender: i.Gender}, nil
	}}
	svc := NewUserService(repo, &fakeApplicationRepo{}, &fakeTx{}, hash.NewBcrypt(4), nil)
	input := domain.CreateUserInput{FullName: " Иван Иванов ", Phone: "8 999 111-22-33", Email: " TEST@GMAIL.COM ", Password: "password1", Role: domain.RoleManager, Gender: domain.GenderMale}
	out, err := svc.CreateUser(context.Background(), domain.RoleAdmin, input)
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if out.ID != 1 || got.FullName != "Иван Иванов" || got.Phone != "+79991112233" || got.Email != "test@gmail.com" {
		t.Fatalf("not normalized: %+v", got)
	}
	if got.Password == "password1" {
		t.Fatal("password must be hashed")
	}

	input.Role = domain.RoleAdmin
	if _, err := svc.CreateUser(context.Background(), domain.RoleManager, input); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("manager must not create admin: %v", err)
	}
	input.Role = domain.RoleClient
	if _, err := svc.CreateUser(context.Background(), domain.RoleClient, input); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("client must not create user: %v", err)
	}
}

func TestUserChangePassword(t *testing.T) {
	h := hash.NewBcrypt(4)
	oldHash, _ := h.Hash("oldpass12")
	var stored string
	repo := &fakeUserRepo{
		getByIDFn: func(context.Context, int64) (*domain.User, error) {
			return &domain.User{ID: 5, PasswordHash: oldHash}, nil
		},
		updatePasswordFn: func(_ context.Context, id int64, s string) error {
			if id != 5 {
				t.Fatalf("bad id")
			}
			stored = s
			return nil
		},
	}
	svc := NewUserService(repo, &fakeApplicationRepo{}, &fakeTx{}, h, nil)
	if err := svc.ChangePassword(context.Background(), 5, domain.ChangePasswordInput{OldPassword: "wrong", NewPassword: "newpass12"}); !errors.Is(err, domain.ErrInvalidPassword) {
		t.Fatalf("expected invalid password: %v", err)
	}
	if err := svc.ChangePassword(context.Background(), 5, domain.ChangePasswordInput{OldPassword: "oldpass12", NewPassword: "short"}); !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("expected invalid input: %v", err)
	}
	if err := svc.ChangePassword(context.Background(), 5, domain.ChangePasswordInput{OldPassword: "oldpass12", NewPassword: "newpass12"}); err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if stored == "" || !h.Compare(stored, "newpass12") {
		t.Fatal("new password hash not stored")
	}
}

func TestUserRejectApplicationTransitions(t *testing.T) {
	app := &domain.ApplicationRequest{ID: 7, Status: domain.ApplicationPending}
	var status domain.ApplicationStatus
	apps := &fakeApplicationRepo{
		getByIDForUpdateFn: func(context.Context, int64) (*domain.ApplicationRequest, error) { cp := *app; return &cp, nil },
		updateStatusFn:     func(_ context.Context, _ int64, s domain.ApplicationStatus) error { status = s; return nil },
	}
	svc := NewUserService(&fakeUserRepo{}, apps, &fakeTx{}, hash.NewBcrypt(4), nil)
	if err := svc.RejectApplication(context.Background(), 7); err != nil {
		t.Fatal(err)
	}
	if status != domain.ApplicationRejected {
		t.Fatalf("bad reject status: %s", status)
	}
	app.Status = domain.ApplicationApproved
	if err := svc.RejectApplication(context.Background(), 7); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("expected conflict: %v", err)
	}
}

func TestUserApplicationStatusValidation(t *testing.T) {
	svc := NewUserService(&fakeUserRepo{}, &fakeApplicationRepo{}, &fakeTx{}, hash.NewBcrypt(4), nil)
	if _, err := svc.ListApplications(context.Background(), domain.ApplicationStatus("wat")); !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("expected invalid input: %v", err)
	}
}

func TestUserSelfDeactivationIsForbidden(t *testing.T) {
	calls := 0
	repo := &fakeUserRepo{setActiveFn: func(context.Context, int64, bool) error { calls++; return nil }}
	svc := NewUserService(repo, &fakeApplicationRepo{}, &fakeTx{}, hash.NewBcrypt(4), nil)
	if err := svc.DeleteUser(context.Background(), 9, 9); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected forbidden: %v", err)
	}
	if err := svc.SetActive(context.Background(), 9, 9, false); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected forbidden: %v", err)
	}
	if calls != 0 {
		t.Fatalf("repo must not be called, calls=%d", calls)
	}
	if err := svc.SetActive(context.Background(), 9, 10, false); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("expected one repo call")
	}
}

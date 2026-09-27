package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/sfedu-crm/internal/domain"
)

func newCRMService(crm *fakeCRMRepo, users *fakeUserRepo, subs *fakeSubscriptionRepo, tx *fakeTx) *CRMService {
	if crm == nil {
		crm = &fakeCRMRepo{}
	}
	if users == nil {
		users = &fakeUserRepo{}
	}
	if subs == nil {
		subs = &fakeSubscriptionRepo{}
	}
	if tx == nil {
		tx = &fakeTx{}
	}
	return NewCRMService(crm, users, subs, tx)
}

func activeClientRepo() *fakeUserRepo {
	return &fakeUserRepo{getByIDFn: func(context.Context, int64) (*domain.User, error) {
		return &domain.User{ID: 1, Role: domain.RoleClient, IsActive: true}, nil
	}}
}

func TestCRMAddProgressValidation(t *testing.T) {
	svc := newCRMService(nil, activeClientRepo(), nil, nil)
	if _, err := svc.AddProgress(context.Background(), 1, 1, domain.CreateProgressInput{}); !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("empty progress must fail: %v", err)
	}
	bad := 19.9
	if _, err := svc.AddProgress(context.Background(), 1, 1, domain.CreateProgressInput{Weight: &bad}); !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("bad weight must fail: %v", err)
	}
	huge := 101.0
	if _, err := svc.AddProgress(context.Background(), 1, 1, domain.CreateProgressInput{BodyFat: &huge}); !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("bad body fat must fail: %v", err)
	}
}

func TestCRMAddProgressTrimsAndStores(t *testing.T) {
	weight := 80.5
	var got domain.CreateProgressInput
	crm := &fakeCRMRepo{addProgressFn: func(_ context.Context, c, a int64, in domain.CreateProgressInput) (*domain.ClientProgress, error) {
		got = in
		return &domain.ClientProgress{ID: 2, ClientID: c, RecordedBy: a, Comment: in.Comment}, nil
	}}
	svc := newCRMService(crm, activeClientRepo(), nil, nil)
	out, err := svc.AddProgress(context.Background(), 1, 9, domain.CreateProgressInput{Weight: &weight, Comment: "  хорошо  "})
	if err != nil {
		t.Fatal(err)
	}
	if out.ID != 2 || got.Comment != "хорошо" {
		t.Fatalf("bad progress: %+v %+v", out, got)
	}
}

func TestCRMListProgressClientCannotReadAnotherClient(t *testing.T) {
	var clientID int64
	crm := &fakeCRMRepo{listProgressFn: func(_ context.Context, id int64, _ int) ([]*domain.ClientProgress, error) {
		clientID = id
		return nil, nil
	}}
	users := &fakeUserRepo{getByIDFn: func(_ context.Context, id int64) (*domain.User, error) {
		return &domain.User{ID: id, Role: domain.RoleClient}, nil
	}}
	svc := newCRMService(crm, users, nil, nil)
	_, err := svc.ListProgress(context.Background(), 5, domain.RoleClient, 999)
	if err != nil {
		t.Fatal(err)
	}
	if clientID != 5 {
		t.Fatalf("client should be forced to self, got %d", clientID)
	}
}

func TestCRMNotesValidation(t *testing.T) {
	svc := newCRMService(nil, activeClientRepo(), nil, nil)
	if _, err := svc.AddNote(context.Background(), 1, 9, "   "); !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("empty note: %v", err)
	}
	long := make([]byte, 4001)
	for i := range long {
		long[i] = 'x'
	}
	if _, err := svc.AddNote(context.Background(), 1, 9, string(long)); !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("long note: %v", err)
	}
}

func TestCRMCreateTaskAssigneeAndClientValidation(t *testing.T) {
	users := &fakeUserRepo{getByIDFn: func(_ context.Context, id int64) (*domain.User, error) {
		switch id {
		case 2:
			return &domain.User{ID: 2, Role: domain.RoleManager, IsActive: true}, nil
		case 3:
			return &domain.User{ID: 3, Role: domain.RoleClient, IsActive: true}, nil
		case 4:
			return &domain.User{ID: 4, Role: domain.RoleClient, IsActive: false}, nil
		}
		return nil, domain.ErrNotFound
	}}
	crm := &fakeCRMRepo{createTaskFn: func(_ context.Context, a int64, in domain.CreateTaskInput) (*domain.StaffTask, error) {
		return &domain.StaffTask{ID: 7, CreatedBy: a, AssigneeID: in.AssigneeID, Title: in.Title}, nil
	}}
	svc := newCRMService(crm, users, nil, nil)
	clientID := int64(3)
	out, err := svc.CreateTask(context.Background(), 1, domain.CreateTaskInput{ClientID: &clientID, AssigneeID: 2, Title: "  Позвонить  "})
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if out.Title != "Позвонить" {
		t.Fatalf("title not trimmed: %q", out.Title)
	}
	if _, err := svc.CreateTask(context.Background(), 1, domain.CreateTaskInput{AssigneeID: 3, Title: "x"}); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("client cannot be assignee: %v", err)
	}
	inactiveClient := int64(4)
	// ensureClient currently checks role, not active state: task is still allowed for historical/inactive clients.
	if _, err := svc.CreateTask(context.Background(), 1, domain.CreateTaskInput{ClientID: &inactiveClient, AssigneeID: 2, Title: "x"}); err != nil {
		t.Fatalf("inactive client task should remain possible: %v", err)
	}
}

func TestCRMListTasksManagerForcedToSelf(t *testing.T) {
	var assignee *int64
	crm := &fakeCRMRepo{listTasksFn: func(_ context.Context, a, b *int64, _ bool, _ int) ([]*domain.StaffTask, error) {
		assignee = a
		return nil, nil
	}}
	svc := newCRMService(crm, nil, nil, nil)
	if _, err := svc.ListTasks(context.Background(), 12, domain.RoleManager, nil, false); err != nil {
		t.Fatal(err)
	}
	if assignee == nil || *assignee != 12 {
		t.Fatalf("manager should only see own tasks: %v", assignee)
	}
	if _, err := svc.ListTasks(context.Background(), 12, domain.RoleAdmin, nil, false); err != nil {
		t.Fatal(err)
	}
	if assignee != nil {
		t.Fatalf("admin should not be force-filtered")
	}
}

func TestCRMTaskStatusValidation(t *testing.T) {
	svc := newCRMService(nil, nil, nil, nil)
	for _, status := range []string{"open", "done", "cancelled"} {
		svc.crm = &fakeCRMRepo{updateTaskStatusFn: func(_ context.Context, _ int64, s string, _ *int64) (*domain.StaffTask, error) {
			return &domain.StaffTask{Status: s}, nil
		}}
		out, err := svc.UpdateTaskStatus(context.Background(), 5, domain.RoleAdmin, 1, status)
		if err != nil || out.Status != status {
			t.Fatalf("status %s failed: %v", status, err)
		}
	}
	if _, err := svc.UpdateTaskStatus(context.Background(), 5, domain.RoleAdmin, 1, "unknown"); !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("expected invalid status: %v", err)
	}
}

func TestCRMNotificationValidationAndUserExistence(t *testing.T) {
	users := &fakeUserRepo{getByIDFn: func(context.Context, int64) (*domain.User, error) { return &domain.User{ID: 3}, nil }}
	var got domain.CreateNotificationInput
	crm := &fakeCRMRepo{createNotificationFn: func(_ context.Context, in domain.CreateNotificationInput) (*domain.Notification, error) {
		got = in
		return &domain.Notification{ID: 1, UserID: in.UserID, Title: in.Title, Message: in.Message}, nil
	}}
	svc := newCRMService(crm, users, nil, nil)
	out, err := svc.CreateNotification(context.Background(), domain.CreateNotificationInput{UserID: 3, Title: "  Напоминание ", Message: "  Тренировка завтра  "})
	if err != nil {
		t.Fatal(err)
	}
	if out.ID != 1 || got.Title != "Напоминание" || got.Message != "Тренировка завтра" {
		t.Fatalf("bad notification: %+v", got)
	}
	if _, err := svc.CreateNotification(context.Background(), domain.CreateNotificationInput{UserID: 3, Title: "", Message: "x"}); !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("expected invalid input: %v", err)
	}
}

func TestCRMFreezeSubscriptionRules(t *testing.T) {
	future := time.Now().Add(10 * 24 * time.Hour)
	valid := &domain.ClientSubscription{ID: 9, ClientID: 1, IsActive: true, EndDate: future}
	subs := &fakeSubscriptionRepo{getByIDForUpdateFn: func(context.Context, int64) (*domain.ClientSubscription, error) { cp := *valid; return &cp, nil }}
	crm := &fakeCRMRepo{freezeFn: func(context.Context, int64) (*domain.ClientSubscription, error) {
		return &domain.ClientSubscription{ID: 9, IsActive: true}, nil
	}}
	svc := newCRMService(crm, nil, subs, &fakeTx{})
	if _, err := svc.FreezeSubscription(context.Background(), 9); err != nil {
		t.Fatalf("valid freeze failed: %v", err)
	}

	crm.hasFutureFn = func(context.Context, int64) (bool, error) { return true, nil }
	if _, err := svc.FreezeSubscription(context.Background(), 9); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("future training should block freeze: %v", err)
	}
	crm.hasFutureFn = nil
	frozenAt := time.Now()
	valid.FrozenAt = &frozenAt
	if _, err := svc.FreezeSubscription(context.Background(), 9); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("already frozen should conflict: %v", err)
	}
}

func TestCRMExtendSubscriptionBounds(t *testing.T) {
	svc := newCRMService(&fakeCRMRepo{extendFn: func(context.Context, int64, int) (*domain.ClientSubscription, error) {
		return &domain.ClientSubscription{ID: 1}, nil
	}}, nil, nil, nil)
	for _, days := range []int{0, -1, 366} {
		if _, err := svc.ExtendSubscription(context.Background(), 1, days); !errors.Is(err, domain.ErrInvalidInput) {
			t.Fatalf("days=%d expected invalid: %v", days, err)
		}
	}
	if _, err := svc.ExtendSubscription(context.Background(), 1, 365); err != nil {
		t.Fatalf("365 days should be valid: %v", err)
	}
}

func TestCRMClientCardAggregatesDataAndDaysInactive(t *testing.T) {
	last := time.Now().Add(-72 * time.Hour)
	users := &fakeUserRepo{getByIDFn: func(context.Context, int64) (*domain.User, error) {
		return &domain.User{ID: 1, Role: domain.RoleClient, LastVisitAt: &last}, nil
	}}
	subs := &fakeSubscriptionRepo{listByClientFn: func(context.Context, int64) ([]*domain.ClientSubscription, error) {
		return []*domain.ClientSubscription{{ID: 2}}, nil
	}}
	crm := &fakeCRMRepo{
		listProgressFn: func(context.Context, int64, int) ([]*domain.ClientProgress, error) {
			return []*domain.ClientProgress{{ID: 3}}, nil
		},
		listNotesFn: func(context.Context, int64, int) ([]*domain.ClientNote, error) {
			return []*domain.ClientNote{{ID: 4}}, nil
		},
		listTasksFn: func(context.Context, *int64, *int64, bool, int) ([]*domain.StaffTask, error) {
			return []*domain.StaffTask{{ID: 5}}, nil
		},
	}
	svc := newCRMService(crm, users, subs, nil)
	card, err := svc.ClientCard(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(card.Subscriptions) != 1 || len(card.Progress) != 1 || len(card.Notes) != 1 || len(card.Tasks) != 1 || card.DaysInactive == nil {
		t.Fatalf("incomplete card: %+v", card)
	}
	if *card.DaysInactive < 2 || *card.DaysInactive > 4 {
		t.Fatalf("unexpected inactive days: %d", *card.DaysInactive)
	}
}

func TestCRMClientCardRejectsNonClient(t *testing.T) {
	svc := newCRMService(nil, &fakeUserRepo{getByIDFn: func(context.Context, int64) (*domain.User, error) { return &domain.User{Role: domain.RoleManager}, nil }}, nil, nil)
	if _, err := svc.ClientCard(context.Background(), 2); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected forbidden: %v", err)
	}
}

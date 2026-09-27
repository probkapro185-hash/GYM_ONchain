package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/sfedu-crm/internal/domain"
)

func newScheduleSvc(tr *fakeTrainingRepo, req *fakeTrainingRequestRepo, users *fakeUserRepo, trainers *fakeTrainerRepo, subs *fakeSubscriptionRepo, tx *fakeTx) *ScheduleService {
	if tr == nil {
		tr = &fakeTrainingRepo{}
	}
	if req == nil {
		req = &fakeTrainingRequestRepo{}
	}
	if users == nil {
		users = &fakeUserRepo{}
	}
	if trainers == nil {
		trainers = &fakeTrainerRepo{}
	}
	if subs == nil {
		subs = &fakeSubscriptionRepo{}
	}
	if tx == nil {
		tx = &fakeTx{}
	}
	return NewScheduleService(tr, req, users, trainers, subs, tx)
}

func TestScheduleClientIsolation(t *testing.T) {
	var got domain.ScheduleFilter
	tr := &fakeTrainingRepo{listFn: func(_ context.Context, f domain.ScheduleFilter) ([]*domain.Training, error) { got = f; return nil, nil }, getByIDFn: func(context.Context, int64) (*domain.Training, error) {
		return &domain.Training{ID: 5, ClientID: 99}, nil
	}}
	svc := newScheduleSvc(tr, nil, nil, nil, nil, nil)
	_, err := svc.GetSchedule(context.Background(), 42, domain.RoleClient, domain.ScheduleFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if got.ClientID == nil || *got.ClientID != 42 {
		t.Fatalf("client filter not enforced: %+v", got)
	}
	if _, err := svc.GetTraining(context.Background(), 42, domain.RoleClient, 5); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected IDOR protection, got %v", err)
	}
	if _, err := svc.GetTraining(context.Background(), 42, domain.RoleAdmin, 5); err != nil {
		t.Fatalf("admin should access: %v", err)
	}
}

func TestScheduleRejectsInvalidStatusFilter(t *testing.T) {
	svc := newScheduleSvc(nil, nil, nil, nil, nil, nil)
	if _, err := svc.GetSchedule(context.Background(), 1, domain.RoleAdmin, domain.ScheduleFilter{Status: domain.TrainingStatus("weird")}); !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("expected invalid input: %v", err)
	}
}

func TestSubmitTrainingRequestRequiresFutureActiveClientAndCoveringSubscription(t *testing.T) {
	future := time.Now().Add(24 * time.Hour)
	users := &fakeUserRepo{getByIDFn: func(context.Context, int64) (*domain.User, error) {
		return &domain.User{ID: 1, Role: domain.RoleClient, IsActive: true}, nil
	}}
	subs := &fakeSubscriptionRepo{getCoveringByClientFn: func(_ context.Context, id int64, s, e time.Time) (*domain.ClientSubscription, error) {
		return &domain.ClientSubscription{ID: 2, ClientID: id, IsActive: true}, nil
	}}
	var got domain.CreateTrainingRequestInput
	req := &fakeTrainingRequestRepo{createFn: func(_ context.Context, id int64, in domain.CreateTrainingRequestInput) (*domain.TrainingRequest, error) {
		got = in
		return &domain.TrainingRequest{ID: 8, ClientID: id, PreferredAt: in.PreferredAt, Comment: in.Comment, Status: domain.TrainingRequestPending}, nil
	}}
	svc := newScheduleSvc(nil, req, users, nil, subs, nil)
	out, err := svc.SubmitTrainingRequest(context.Background(), 1, domain.CreateTrainingRequestInput{PreferredAt: future, Comment: "  хочу утром  "})
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if out.ID != 8 || got.Comment != "хочу утром" {
		t.Fatalf("bad request: %+v / %+v", out, got)
	}

	if _, err := svc.SubmitTrainingRequest(context.Background(), 1, domain.CreateTrainingRequestInput{PreferredAt: time.Now().Add(-time.Minute)}); !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("past request must fail: %v", err)
	}
	tooLong := make([]byte, 2001)
	for i := range tooLong {
		tooLong[i] = 'x'
	}
	if _, err := svc.SubmitTrainingRequest(context.Background(), 1, domain.CreateTrainingRequestInput{PreferredAt: future, Comment: string(tooLong)}); !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("long comment must fail: %v", err)
	}
}

func TestSubmitTrainingRequestRejectsInactiveOrNonClient(t *testing.T) {
	future := time.Now().Add(time.Hour)
	cases := []*domain.User{{Role: domain.RoleClient, IsActive: false}, {Role: domain.RoleManager, IsActive: true}}
	for i, u := range cases {
		svc := newScheduleSvc(nil, nil, &fakeUserRepo{getByIDFn: func(context.Context, int64) (*domain.User, error) { return u, nil }}, nil, &fakeSubscriptionRepo{}, nil)
		if _, err := svc.SubmitTrainingRequest(context.Background(), 1, domain.CreateTrainingRequestInput{PreferredAt: future}); !errors.Is(err, domain.ErrForbidden) {
			t.Fatalf("case %d expected forbidden: %v", i, err)
		}
	}
}

func TestApproveRequestUsesRequestOwnerAndDefaultsTimes(t *testing.T) {
	pref := time.Now().Add(24 * time.Hour).Truncate(time.Second)
	reqRepo := &fakeTrainingRequestRepo{
		getByIDForUpdateFn: func(context.Context, int64) (*domain.TrainingRequest, error) {
			return &domain.TrainingRequest{ID: 3, ClientID: 77, PreferredAt: pref, Status: domain.TrainingRequestPending}, nil
		},
		updateStatusFn: func(_ context.Context, _ int64, s domain.TrainingRequestStatus) error {
			if s != domain.TrainingRequestScheduled {
				t.Fatalf("bad status %s", s)
			}
			return nil
		},
	}
	users := &fakeUserRepo{getByIDForUpdateFn: func(_ context.Context, id int64) (*domain.User, error) {
		if id != 77 {
			t.Fatalf("must use request client id, got %d", id)
		}
		return &domain.User{ID: id, Role: domain.RoleClient, IsActive: true}, nil
	}}
	subs := &fakeSubscriptionRepo{getCoveringByClientForUpdateFn: func(_ context.Context, id int64, s, e time.Time) (*domain.ClientSubscription, error) {
		if id != 77 {
			t.Fatalf("wrong client")
		}
		if !s.Equal(pref) || !e.Equal(pref.Add(time.Hour)) {
			t.Fatalf("bad defaults: %v %v", s, e)
		}
		return &domain.ClientSubscription{ID: 9, ClientID: id, IsActive: true}, nil
	}}
	var created domain.CreateTrainingInput
	tr := &fakeTrainingRepo{createFn: func(_ context.Context, in domain.CreateTrainingInput) (*domain.Training, error) {
		created = in
		return &domain.Training{ID: 10, ClientID: in.ClientID, SubscriptionID: in.SubscriptionID, StartTime: in.StartTime, EndTime: in.EndTime, Status: in.Status}, nil
	}}
	svc := newScheduleSvc(tr, reqRepo, users, nil, subs, &fakeTx{})
	out, err := svc.ApproveRequest(context.Background(), 3, domain.CreateTrainingInput{ClientID: 999, Title: " Test "})
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if out.ID != 10 || created.ClientID != 77 || created.Status != domain.TrainingStatusScheduled || created.SubscriptionID == nil || *created.SubscriptionID != 9 {
		t.Fatalf("approval ownership/reservation failed: %+v", created)
	}
}

func TestApproveAndRejectAlreadyProcessedRequestConflict(t *testing.T) {
	req := &fakeTrainingRequestRepo{getByIDForUpdateFn: func(context.Context, int64) (*domain.TrainingRequest, error) {
		return &domain.TrainingRequest{Status: domain.TrainingRequestScheduled}, nil
	}}
	svc := newScheduleSvc(nil, req, nil, nil, nil, &fakeTx{})
	if _, err := svc.ApproveRequest(context.Background(), 1, domain.CreateTrainingInput{}); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("expected conflict: %v", err)
	}
	if err := svc.RejectRequest(context.Background(), 1); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("expected conflict: %v", err)
	}
}

func TestCreateTrainingRejectsOverlapAndExhaustedReservation(t *testing.T) {
	start := time.Now().Add(24 * time.Hour)
	end := start.Add(time.Hour)
	left := 1
	users := &fakeUserRepo{getByIDForUpdateFn: func(context.Context, int64) (*domain.User, error) {
		return &domain.User{Role: domain.RoleClient, IsActive: true}, nil
	}}
	subs := &fakeSubscriptionRepo{getCoveringByClientForUpdateFn: func(context.Context, int64, time.Time, time.Time) (*domain.ClientSubscription, error) {
		return &domain.ClientSubscription{ID: 2, ClientID: 1, IsActive: true, SessionsLeft: &left}, nil
	}}
	conflict := &fakeTrainingRepo{conflictFn: func(context.Context, int64, int64, *int64, time.Time, time.Time) (bool, error) { return true, nil }}
	svc := newScheduleSvc(conflict, nil, users, nil, subs, &fakeTx{})
	if _, err := svc.CreateTraining(context.Background(), domain.CreateTrainingInput{ClientID: 1, Title: "A", StartTime: start, EndTime: end}); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("overlap should conflict: %v", err)
	}

	exhausted := &fakeTrainingRepo{countFn: func(context.Context, int64, int64) (int, error) { return 1, nil }}
	svc = newScheduleSvc(exhausted, nil, users, nil, subs, &fakeTx{})
	if _, err := svc.CreateTraining(context.Background(), domain.CreateTrainingInput{ClientID: 1, Title: "A", StartTime: start, EndTime: end}); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("reserved sessions should conflict: %v", err)
	}
}

func TestCompleteTrainingConsumesReservedSubscriptionAndVisit(t *testing.T) {
	start := time.Now().Add(-2 * time.Hour)
	end := start.Add(time.Hour)
	subID := int64(11)
	current := &domain.Training{ID: 4, ClientID: 6, SubscriptionID: &subID, Title: "PT", StartTime: start, EndTime: end, Status: domain.TrainingStatusScheduled}
	users := &fakeUserRepo{getByIDForUpdateFn: func(context.Context, int64) (*domain.User, error) {
		return &domain.User{ID: 6, Role: domain.RoleClient, IsActive: true}, nil
	}, incrementVisitsFn: func(_ context.Context, id int64, visitedAt time.Time) error {
		if !visitedAt.Equal(start) {
			t.Fatalf("visit date must be the training date, got %v", visitedAt)
		}
		if id != 6 {
			t.Fatalf("bad client")
		}
		return nil
	}}
	dec := 0
	subs := &fakeSubscriptionRepo{getByIDForUpdateFn: func(context.Context, int64) (*domain.ClientSubscription, error) {
		return &domain.ClientSubscription{ID: subID, ClientID: 6, StartDate: start.Add(-24 * time.Hour), EndDate: end.Add(24 * time.Hour), IsActive: true}, nil
	}, decrementSessionsFn: func(_ context.Context, id int64) error {
		if id != subID {
			t.Fatalf("bad sub")
		}
		dec++
		return nil
	}}
	var update domain.UpdateTrainingInput
	tr := &fakeTrainingRepo{getByIDForUpdateFn: func(context.Context, int64) (*domain.Training, error) { return current, nil }, updateFn: func(_ context.Context, id int64, in domain.UpdateTrainingInput) (*domain.Training, error) {
		update = in
		return &domain.Training{ID: id, Status: in.Status}, nil
	}}
	svc := newScheduleSvc(tr, nil, users, nil, subs, &fakeTx{})
	_, err := svc.UpdateTraining(context.Background(), 4, domain.UpdateTrainingInput{Title: "PT", StartTime: start, EndTime: end, Status: domain.TrainingStatusCompleted})
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if dec != 1 || update.SubscriptionID == nil || *update.SubscriptionID != subID || update.Status != domain.TrainingStatusCompleted {
		t.Fatalf("completion side effects missing: dec=%d update=%+v", dec, update)
	}
}

func TestCompleteFutureTrainingRejected(t *testing.T) {
	start := time.Now().Add(time.Hour)
	end := start.Add(time.Hour)
	tr := &fakeTrainingRepo{getByIDForUpdateFn: func(context.Context, int64) (*domain.Training, error) {
		return &domain.Training{ID: 1, ClientID: 1, Title: "x", StartTime: start, EndTime: end, Status: domain.TrainingStatusScheduled}, nil
	}}
	svc := newScheduleSvc(tr, nil, nil, nil, nil, &fakeTx{})
	_, err := svc.UpdateTraining(context.Background(), 1, domain.UpdateTrainingInput{Title: "x", StartTime: start, EndTime: end, Status: domain.TrainingStatusCompleted})
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("expected invalid input: %v", err)
	}
}

func TestCancelMyTrainingOwnershipAndCutoff(t *testing.T) {
	base := &domain.Training{ID: 5, ClientID: 7, Title: "x", StartTime: time.Now().Add(3 * time.Hour), EndTime: time.Now().Add(4 * time.Hour), Status: domain.TrainingStatusScheduled}
	updated := 0
	tr := &fakeTrainingRepo{getByIDForUpdateFn: func(context.Context, int64) (*domain.Training, error) { cp := *base; return &cp, nil }, updateFn: func(_ context.Context, _ int64, in domain.UpdateTrainingInput) (*domain.Training, error) {
		updated++
		if in.Status != domain.TrainingStatusCancelled {
			t.Fatalf("bad status")
		}
		return &domain.Training{Status: in.Status}, nil
	}}
	svc := newScheduleSvc(tr, nil, nil, nil, nil, &fakeTx{})
	if err := svc.CancelMyTraining(context.Background(), 8, 5); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected forbidden: %v", err)
	}
	base.StartTime = time.Now().Add(90 * time.Minute)
	base.EndTime = base.StartTime.Add(time.Hour)
	if err := svc.CancelMyTraining(context.Background(), 7, 5); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("expected cutoff conflict: %v", err)
	}
	base.StartTime = time.Now().Add(3 * time.Hour)
	base.EndTime = base.StartTime.Add(time.Hour)
	if err := svc.CancelMyTraining(context.Background(), 7, 5); err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if updated != 1 {
		t.Fatalf("expected one update, got %d", updated)
	}
}

func TestDeleteCompletedTrainingRejected(t *testing.T) {
	deleted := 0
	tr := &fakeTrainingRepo{getByIDFn: func(context.Context, int64) (*domain.Training, error) {
		return &domain.Training{Status: domain.TrainingStatusCompleted}, nil
	}, deleteFn: func(context.Context, int64) error { deleted++; return nil }}
	svc := newScheduleSvc(tr, nil, nil, nil, nil, nil)
	if err := svc.DeleteTraining(context.Background(), 1); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("expected conflict: %v", err)
	}
	if deleted != 0 {
		t.Fatal("completed training must not be deleted")
	}
}

//go:build integration

package integration

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/sfedu-crm/internal/domain"
	"github.com/sfedu-crm/internal/repository"
	"github.com/sfedu-crm/internal/repository/postgres"
	"github.com/sfedu-crm/internal/service"
)

// Each scenario runs in a transaction and rolls back all test data.
func TestReviewDatabaseRegressions(t *testing.T) {
	pool := testPool(t)
	users := postgres.NewUserRepository(pool)
	trainers := postgres.NewTrainerRepository(pool)
	trainings := postgres.NewTrainingRepository(pool)
	requests := postgres.NewTrainingRequestRepository(pool)
	subs := postgres.NewSubscriptionRepository(pool)
	products := postgres.NewProductRepository(pool)
	crm := postgres.NewCRMRepository(pool)
	apps := postgres.NewApplicationRepository(pool)
	txm := postgres.NewTransactionManager(pool)
	schedule := service.NewScheduleService(trainings, requests, users, trainers, subs, txm)
	crmService := service.NewCRMService(crm, users, subs, txm)
	rollback := errors.New("rollback test data")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	err := txm.WithinTransaction(ctx, func(ctx context.Context) error {
		t.Run("operations", func(t *testing.T) {
			counter := 0
			newUser := func(role domain.Role) *domain.User {
				counter++
				u, err := users.Create(ctx, domain.CreateUserInput{FullName: "Тест Проверки", Phone: fmt.Sprintf("+7987000%04d", counter), Email: fmt.Sprintf("review-%d@gmail.com", counter), Password: "unused-test-hash", Role: role, Gender: domain.GenderFemale})
				if err != nil {
					t.Fatal(err)
				}
				return u
			}
			client, manager, otherManager, trainerUser := newUser(domain.RoleClient), newUser(domain.RoleManager), newUser(domain.RoleManager), newUser(domain.RoleClient)
			trainer, err := trainers.Create(ctx, domain.CreateTrainerInput{UserID: trainerUser.ID, Specialization: domain.SpecBodyRelief})
			if err != nil {
				t.Fatal(err)
			}
			days, count := 30, 1
			subtype := domain.SubscriptionType("monthly")
			product, err := products.Create(ctx, domain.CreateProductInput{Name: "Проверка", Price: 100, Category: domain.ProductCategory("subscription"), SubType: &subtype, DurationDays: &days, SessionsCount: &count})
			if err != nil {
				t.Fatal(err)
			}
			first, err := subs.Create(ctx, client.ID, product)
			if err != nil {
				t.Fatal(err)
			}
			second, err := subs.Create(ctx, client.ID, product)
			if err != nil {
				t.Fatal(err)
			}
			start := time.Now().Add(24 * time.Hour).Truncate(time.Second)
			one, err := schedule.CreateTraining(ctx, domain.CreateTrainingInput{ClientID: client.ID, Title: "Первая", StartTime: start, EndTime: start.Add(time.Hour)})
			if err != nil {
				t.Fatal(err)
			}
			if one.SubscriptionID == nil || *one.SubscriptionID != first.ID {
				t.Fatal("first reservation used wrong subscription")
			}
			two, err := schedule.CreateTraining(ctx, domain.CreateTrainingInput{ClientID: client.ID, Title: "Вторая", StartTime: start.Add(2 * time.Hour), EndTime: start.Add(3 * time.Hour)})
			if err != nil {
				t.Fatalf("second subscription has capacity but scheduling failed: %v", err)
			}
			if two.SubscriptionID == nil || *two.SubscriptionID != second.ID {
				t.Fatal("full first subscription must be skipped")
			}
			moved, err := schedule.UpdateTraining(ctx, one.ID, domain.UpdateTrainingInput{Title: "Перенос", StartTime: start.Add(4 * time.Hour), EndTime: start.Add(5 * time.Hour), Status: domain.TrainingStatusScheduled})
			if err != nil || moved == nil || *moved.SubscriptionID != first.ID {
				t.Fatalf("rescheduling counted its own reservation: %v", err)
			}
			if err := schedule.DeleteTraining(ctx, two.ID); err != nil {
				t.Fatal(err)
			}
			req, err := schedule.SubmitTrainingRequest(ctx, client.ID, domain.CreateTrainingRequestInput{PreferredAt: start, TrainerID: &trainer.ID})
			if err != nil {
				t.Fatal(err)
			}
			pending, err := requests.ListPending(ctx)
			if err != nil {
				t.Fatal(err)
			}
			found := false
			for _, r := range pending {
				if r.ID == req.ID {
					found = r.TrainerID != nil && *r.TrainerID == trainer.ID && r.TrainerName == trainerUser.FullName
				}
			}
			if !found {
				t.Fatal("stored request lost trainer or name")
			}
			approved, err := schedule.ApproveRequest(ctx, req.ID, domain.CreateTrainingInput{Title: "С тренером"})
			if err != nil {
				t.Fatal(err)
			}
			if approved.TrainerID == nil || *approved.TrainerID != trainer.ID {
				t.Fatal("approved training lost requested trainer")
			}
			if _, err := trainers.Update(ctx, trainer.ID, domain.UpdateTrainerInput{Specialization: domain.SpecBodyRelief, IsActive: false}); err != nil {
				t.Fatal(err)
			}
			active := true
			public, err := trainers.List(ctx, repository.TrainerFilter{IsActive: &active})
			if err != nil {
				t.Fatal(err)
			}
			for _, tr := range public {
				if tr.ID == trainer.ID {
					t.Fatal("inactive trainer is public")
				}
			}
			all, err := trainers.List(ctx, repository.TrainerFilter{})
			if err != nil {
				t.Fatal(err)
			}
			found = false
			for _, tr := range all {
				if tr.ID == trainer.ID {
					found = true
				}
			}
			if !found {
				t.Fatal("admin cannot find inactive trainer")
			}
			restored, err := trainers.Update(ctx, trainer.ID, domain.UpdateTrainerInput{Specialization: domain.SpecBodyRelief, IsActive: true})
			if err != nil || !restored.IsActive {
				t.Fatalf("trainer restoration: %v", err)
			}

			task, err := crm.CreateTask(ctx, manager.ID, domain.CreateTaskInput{AssigneeID: manager.ID, Title: "Связаться с клиентом"})
			if err != nil {
				t.Fatal(err)
			}
			if _, err := crmService.UpdateTaskStatus(ctx, otherManager.ID, domain.RoleManager, task.ID, "done"); !errors.Is(err, domain.ErrNotFound) {
				t.Fatalf("foreign task must be unavailable: %v", err)
			}
			own, err := crm.ListTasks(ctx, &manager.ID, nil, true, 100)
			if err != nil {
				t.Fatal(err)
			}
			if len(own) != 1 || own[0].Status != "open" {
				t.Fatal("foreign manager changed the task")
			}
			if _, err := crmService.UpdateTaskStatus(ctx, manager.ID, domain.RoleManager, task.ID, "done"); err != nil {
				t.Fatal(err)
			}
			if _, err := crmService.UpdateTaskStatus(ctx, otherManager.ID, domain.RoleAdmin, task.ID, "open"); err != nil {
				t.Fatal(err)
			}

			past := time.Now().Add(-24 * time.Hour).Truncate(time.Second)
			if err := users.IncrementVisits(ctx, client.ID, past); err != nil {
				t.Fatal(err)
			}
			if err := users.IncrementVisits(ctx, client.ID, past.Add(-24*time.Hour)); err != nil {
				t.Fatal(err)
			}
			visited, err := users.GetByID(ctx, client.ID)
			if err != nil {
				t.Fatal(err)
			}
			if visited.Visits != 2 || visited.LastVisitAt == nil || !visited.LastVisitAt.Equal(past) {
				t.Fatalf("incorrect historical visit: %+v", visited)
			}

			application, err := apps.Create(ctx, domain.CreateApplicationInput{FullName: "Анна Иванова", Phone: "+79871112233", Email: "anna-review@gmail.com", Gender: domain.GenderFemale})
			if err != nil {
				t.Fatal(err)
			}
			loaded, err := apps.GetByID(ctx, application.ID)
			if err != nil || loaded.Gender != domain.GenderFemale {
				t.Fatalf("application gender round trip: %v", err)
			}
			listed, err := apps.List(ctx, domain.ApplicationPending)
			if err != nil {
				t.Fatal(err)
			}
			found = false
			for _, a := range listed {
				if a.ID == application.ID {
					found = a.Gender == domain.GenderFemale
				}
			}
			if !found {
				t.Fatal("application list lost gender")
			}
			frozenClient := newUser(domain.RoleClient)
			frozenSub, err := subs.Create(ctx, frozenClient.ID, product)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := crm.FreezeSubscription(ctx, frozenSub.ID); err != nil {
				t.Fatal(err)
			}
			listedUsers, err := users.List(ctx, repository.UserFilter{Search: frozenClient.Email})
			if err != nil {
				t.Fatal(err)
			}
			if len(listedUsers) != 1 || listedUsers[0].SessionsLeft != nil {
				t.Fatal("frozen subscription counted as active in clients")
			}
			personal, err := postgres.NewAIRepository(pool).GetPersonalContext(ctx, frozenClient.ID, domain.RoleClient)
			if err != nil {
				t.Fatal(err)
			}
			if personal.SubscriptionUntil != nil || personal.SubscriptionName != "" {
				t.Fatal("frozen subscription leaked into active AI context")
			}
			if _, err := subs.GetCoveringByClientForUpdate(ctx, frozenClient.ID, start, start.Add(time.Hour), 0); !errors.Is(err, domain.ErrNoActiveSubscription) {
				t.Fatalf("frozen subscription available for booking: %v", err)
			}
			if _, err := crm.UnfreezeSubscription(ctx, frozenSub.ID); err != nil {
				t.Fatal(err)
			}
			if _, err := subs.GetCoveringByClientForUpdate(ctx, frozenClient.ID, start, start.Add(time.Hour), 0); err != nil {
				t.Fatal(err)
			}
		})
		return rollback
	})
	if !errors.Is(err, rollback) {
		t.Fatal(err)
	}
}

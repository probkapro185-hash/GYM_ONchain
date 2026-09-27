package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/sfedu-crm/internal/domain"
	"github.com/sfedu-crm/internal/repository"
)

type CRMService struct {
	crm   repository.CRMRepository
	users repository.UserRepository
	subs  repository.SubscriptionRepository
	tx    repository.TransactionManager
}

func NewCRMService(crm repository.CRMRepository, users repository.UserRepository, subs repository.SubscriptionRepository, tx repository.TransactionManager) *CRMService {
	return &CRMService{crm: crm, users: users, subs: subs, tx: tx}
}

func (s *CRMService) AddProgress(ctx context.Context, clientID, actorID int64, input domain.CreateProgressInput) (*domain.ClientProgress, error) {
	if err := s.ensureClient(ctx, clientID); err != nil {
		return nil, err
	}
	input.Comment = strings.TrimSpace(input.Comment)
	if len(input.Comment) > 2000 {
		return nil, fmt.Errorf("%w: comment is too long", domain.ErrInvalidInput)
	}
	if input.Weight == nil && input.BodyFat == nil && input.Waist == nil && input.Chest == nil && input.Hips == nil && input.Comment == "" {
		return nil, fmt.Errorf("%w: progress entry is empty", domain.ErrInvalidInput)
	}
	if !validRange(input.Weight, 20, 500) || !validRange(input.BodyFat, 0, 100) || !validRange(input.Waist, 20, 400) || !validRange(input.Chest, 20, 400) || !validRange(input.Hips, 20, 400) {
		return nil, fmt.Errorf("%w: invalid progress measurement", domain.ErrInvalidInput)
	}
	return s.crm.AddProgress(ctx, clientID, actorID, input)
}

func validRange(v *float64, min, max float64) bool { return v == nil || (*v >= min && *v <= max) }

func (s *CRMService) ListProgress(ctx context.Context, actorID int64, actorRole domain.Role, clientID int64) ([]*domain.ClientProgress, error) {
	if actorRole == domain.RoleClient {
		clientID = actorID
	}
	if err := s.ensureClient(ctx, clientID); err != nil {
		return nil, err
	}
	return s.crm.ListProgress(ctx, clientID, 200)
}
func (s *CRMService) DeleteProgress(ctx context.Context, id int64) error {
	return s.crm.DeleteProgress(ctx, id)
}

func (s *CRMService) AddNote(ctx context.Context, clientID, actorID int64, note string) (*domain.ClientNote, error) {
	if err := s.ensureClient(ctx, clientID); err != nil {
		return nil, err
	}
	note = strings.TrimSpace(note)
	if note == "" || len(note) > 4000 {
		return nil, fmt.Errorf("%w: note must contain 1-4000 chars", domain.ErrInvalidInput)
	}
	return s.crm.AddNote(ctx, clientID, actorID, note)
}
func (s *CRMService) ListNotes(ctx context.Context, clientID int64) ([]*domain.ClientNote, error) {
	if err := s.ensureClient(ctx, clientID); err != nil {
		return nil, err
	}
	return s.crm.ListNotes(ctx, clientID, 200)
}
func (s *CRMService) DeleteNote(ctx context.Context, id int64) error {
	return s.crm.DeleteNote(ctx, id)
}

func (s *CRMService) CreateTask(ctx context.Context, actorID int64, input domain.CreateTaskInput) (*domain.StaffTask, error) {
	input.Title = strings.TrimSpace(input.Title)
	input.Description = strings.TrimSpace(input.Description)
	if input.Title == "" || len(input.Title) > 255 || len(input.Description) > 4000 {
		return nil, fmt.Errorf("%w: invalid task", domain.ErrInvalidInput)
	}
	assignee, err := s.users.GetByID(ctx, input.AssigneeID)
	if err != nil {
		return nil, err
	}
	if !assignee.IsActive || (assignee.Role != domain.RoleManager && assignee.Role != domain.RoleAdmin) {
		return nil, domain.ErrForbidden
	}
	if input.ClientID != nil {
		if err := s.ensureClient(ctx, *input.ClientID); err != nil {
			return nil, err
		}
	}
	return s.crm.CreateTask(ctx, actorID, input)
}
func (s *CRMService) ListTasks(ctx context.Context, actorID int64, actorRole domain.Role, clientID *int64, includeDone bool) ([]*domain.StaffTask, error) {
	var assignee *int64
	if actorRole == domain.RoleManager {
		assignee = &actorID
	}
	return s.crm.ListTasks(ctx, assignee, clientID, includeDone, 500)
}
func (s *CRMService) UpdateTaskStatus(ctx context.Context, actorID int64, actorRole domain.Role, id int64, status string) (*domain.StaffTask, error) {
	status = strings.TrimSpace(status)
	if status != "open" && status != "done" && status != "cancelled" {
		return nil, fmt.Errorf("%w: invalid task status", domain.ErrInvalidInput)
	}
	if actorID <= 0 || (actorRole != domain.RoleAdmin && actorRole != domain.RoleManager) {
		return nil, domain.ErrForbidden
	}
	var assigneeID *int64
	if actorRole == domain.RoleManager {
		assigneeID = &actorID
	}
	return s.crm.UpdateTaskStatus(ctx, id, status, assigneeID)
}

func (s *CRMService) CreateNotification(ctx context.Context, input domain.CreateNotificationInput) (*domain.Notification, error) {
	input.Title = strings.TrimSpace(input.Title)
	input.Message = strings.TrimSpace(input.Message)
	if input.Title == "" || input.Message == "" || len(input.Title) > 255 || len(input.Message) > 4000 {
		return nil, fmt.Errorf("%w: invalid notification", domain.ErrInvalidInput)
	}
	if _, err := s.users.GetByID(ctx, input.UserID); err != nil {
		return nil, err
	}
	return s.crm.CreateNotification(ctx, input)
}
func (s *CRMService) ListNotifications(ctx context.Context, userID int64) ([]*domain.Notification, error) {
	return s.crm.ListNotifications(ctx, userID, 100)
}
func (s *CRMService) MarkNotificationRead(ctx context.Context, id, userID int64) error {
	return s.crm.MarkNotificationRead(ctx, id, userID)
}

func (s *CRMService) FreezeSubscription(ctx context.Context, id int64) (*domain.ClientSubscription, error) {
	var out *domain.ClientSubscription
	err := s.tx.WithinTransaction(ctx, func(txCtx context.Context) error {
		sub, err := s.subs.GetByIDForUpdate(txCtx, id)
		if err != nil {
			return err
		}
		if !sub.IsActive || sub.FrozenAt != nil || !sub.EndDate.After(time.Now()) {
			return fmt.Errorf("%w: subscription cannot be frozen", domain.ErrConflict)
		}
		future, err := s.crm.HasFutureTrainingForSubscription(txCtx, id)
		if err != nil {
			return err
		}
		if future {
			return fmt.Errorf("%w: move or cancel future trainings before freezing", domain.ErrConflict)
		}
		out, err = s.crm.FreezeSubscription(txCtx, id)
		return err
	})
	return out, err
}
func (s *CRMService) UnfreezeSubscription(ctx context.Context, id int64) (*domain.ClientSubscription, error) {
	return s.crm.UnfreezeSubscription(ctx, id)
}
func (s *CRMService) ExtendSubscription(ctx context.Context, id int64, days int) (*domain.ClientSubscription, error) {
	if days <= 0 || days > 365 {
		return nil, fmt.Errorf("%w: days must be between 1 and 365", domain.ErrInvalidInput)
	}
	return s.crm.ExtendSubscription(ctx, id, days)
}

func (s *CRMService) Dashboard(ctx context.Context) (*domain.DashboardSummary, error) {
	return s.crm.Dashboard(ctx)
}
func (s *CRMService) ListAudit(ctx context.Context) ([]*domain.AuditRecord, error) {
	return s.crm.ListAudit(ctx, 500)
}

func (s *CRMService) ClientCard(ctx context.Context, clientID int64) (*domain.ClientCRMCard, error) {
	u, err := s.users.GetByID(ctx, clientID)
	if err != nil {
		return nil, err
	}
	if u.Role != domain.RoleClient {
		return nil, domain.ErrForbidden
	}
	subs, err := s.subs.ListByClient(ctx, clientID)
	if err != nil {
		return nil, err
	}
	progress, err := s.crm.ListProgress(ctx, clientID, 200)
	if err != nil {
		return nil, err
	}
	notes, err := s.crm.ListNotes(ctx, clientID, 200)
	if err != nil {
		return nil, err
	}
	tasks, err := s.crm.ListTasks(ctx, nil, &clientID, true, 200)
	if err != nil {
		return nil, err
	}
	card := &domain.ClientCRMCard{User: u, Subscriptions: subs, Progress: progress, Notes: notes, Tasks: tasks, LastVisitAt: u.LastVisitAt}
	if u.LastVisitAt != nil {
		days := int(time.Since(*u.LastVisitAt).Hours() / 24)
		if days < 0 {
			days = 0
		}
		card.DaysInactive = &days
	}
	return card, nil
}

func (s *CRMService) ensureClient(ctx context.Context, id int64) error {
	u, err := s.users.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if u.Role != domain.RoleClient {
		return domain.ErrForbidden
	}
	return nil
}

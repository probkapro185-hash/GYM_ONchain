package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/sfedu-crm/internal/domain"
	"github.com/sfedu-crm/internal/repository"
	"github.com/sfedu-crm/internal/validator"
)

type ScheduleService struct {
	trainingRepo     repository.TrainingRepository
	requestRepo      repository.TrainingRequestRepository
	userRepo         repository.UserRepository
	trainerRepo      repository.TrainerRepository
	subscriptionRepo repository.SubscriptionRepository
	tx               repository.TransactionManager
}

func NewScheduleService(trainingRepo repository.TrainingRepository,
	requestRepo repository.TrainingRequestRepository,
	userRepo repository.UserRepository,
	trainerRepo repository.TrainerRepository,
	subscriptionRepo repository.SubscriptionRepository,
	tx repository.TransactionManager,
) *ScheduleService {
	return &ScheduleService{
		trainingRepo: trainingRepo, requestRepo: requestRepo, userRepo: userRepo,
		trainerRepo: trainerRepo, subscriptionRepo: subscriptionRepo, tx: tx,
	}
}

func (s *ScheduleService) GetSchedule(ctx context.Context, actorID int64, actorRole domain.Role,
	filter domain.ScheduleFilter) ([]*domain.Training, error) {
	if actorRole == domain.RoleClient {
		filter.ClientID = &actorID
	}
	if filter.Status != "" {
		if err := validator.ValidateTrainingStatus(filter.Status); err != nil {
			return nil, err
		}
	}
	return s.trainingRepo.List(ctx, filter)
}

func (s *ScheduleService) GetTraining(ctx context.Context, actorID int64, actorRole domain.Role, id int64) (*domain.Training, error) {
	training, err := s.trainingRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if actorRole == domain.RoleClient && training.ClientID != actorID {
		return nil, domain.ErrForbidden
	}
	return training, nil
}

func (s *ScheduleService) SubmitTrainingRequest(ctx context.Context, clientID int64,
	input domain.CreateTrainingRequestInput) (*domain.TrainingRequest, error) {
	input.Comment = strings.TrimSpace(input.Comment)
	if input.PreferredAt.IsZero() || !input.PreferredAt.After(time.Now()) {
		return nil, fmt.Errorf("%w: preferred_at must be in the future", domain.ErrInvalidInput)
	}
	if len(input.Comment) > 2000 {
		return nil, fmt.Errorf("%w: comment is too long", domain.ErrInvalidInput)
	}
	if err := s.ensureActiveClient(ctx, clientID); err != nil {
		return nil, err
	}
	if input.TrainerID != nil {
		if *input.TrainerID <= 0 {
			return nil, domain.ErrInvalidInput
		}
		trainer, err := s.trainerRepo.GetByID(ctx, *input.TrainerID)
		if err != nil {
			return nil, err
		}
		if !trainer.IsActive {
			return nil, domain.ErrForbidden
		}
	}
	// A request represents a normal one-hour slot. Require a subscription that actually
	// covers the requested time instead of accepting any unrelated active subscription.
	if _, err := s.subscriptionRepo.GetCoveringByClient(ctx, clientID, input.PreferredAt, input.PreferredAt.Add(time.Hour)); err != nil {
		return nil, err
	}
	return s.requestRepo.Create(ctx, clientID, input)
}

func (s *ScheduleService) ListMyRequests(ctx context.Context, clientID int64) ([]*domain.TrainingRequest, error) {
	return s.requestRepo.ListByClient(ctx, clientID)
}

func (s *ScheduleService) ListPendingRequests(ctx context.Context) ([]*domain.TrainingRequest, error) {
	return s.requestRepo.ListPending(ctx)
}

func (s *ScheduleService) ApproveRequest(ctx context.Context, requestID int64,
	input domain.CreateTrainingInput) (*domain.Training, error) {
	var created *domain.Training
	err := s.tx.WithinTransaction(ctx, func(txCtx context.Context) error {
		req, err := s.requestRepo.GetByIDForUpdate(txCtx, requestID)
		if err != nil {
			return err
		}
		if req.Status != domain.TrainingRequestPending {
			return fmt.Errorf("%w: training request is already processed", domain.ErrConflict)
		}

		input.ClientID = req.ClientID // ownership always comes from the request, never from JSON.
		if input.TrainerID == nil && !input.ClearTrainer {
			input.TrainerID = req.TrainerID
		}
		input.Status = domain.TrainingStatusScheduled
		input.Title = strings.TrimSpace(input.Title)
		input.Description = strings.TrimSpace(input.Description)
		if input.StartTime.IsZero() {
			input.StartTime = req.PreferredAt
		}
		if input.EndTime.IsZero() && !input.StartTime.IsZero() {
			input.EndTime = input.StartTime.Add(time.Hour)
		}
		if err := s.validateNewTraining(txCtx, input); err != nil {
			return err
		}
		sub, err := s.subscriptionRepo.GetCoveringByClientForUpdate(txCtx, req.ClientID, input.StartTime, input.EndTime, 0)
		if err != nil {
			return err
		}
		input.SubscriptionID = &sub.ID
		if err := s.ensureSubscriptionCapacity(txCtx, input.ClientID, sub, 0); err != nil {
			return err
		}
		created, err = s.trainingRepo.Create(txCtx, input)
		if err != nil {
			return err
		}
		return s.requestRepo.UpdateStatus(txCtx, requestID, domain.TrainingRequestScheduled)
	})
	return created, err
}

func (s *ScheduleService) RejectRequest(ctx context.Context, requestID int64) error {
	return s.tx.WithinTransaction(ctx, func(txCtx context.Context) error {
		req, err := s.requestRepo.GetByIDForUpdate(txCtx, requestID)
		if err != nil {
			return err
		}
		if req.Status != domain.TrainingRequestPending {
			return fmt.Errorf("%w: training request is already processed", domain.ErrConflict)
		}
		return s.requestRepo.UpdateStatus(txCtx, requestID, domain.TrainingRequestRejected)
	})
}

func (s *ScheduleService) CreateTraining(ctx context.Context, input domain.CreateTrainingInput) (*domain.Training, error) {
	input.Status = domain.TrainingStatusScheduled
	input.Title = strings.TrimSpace(input.Title)
	input.Description = strings.TrimSpace(input.Description)
	var created *domain.Training
	err := s.tx.WithinTransaction(ctx, func(txCtx context.Context) error {
		if err := s.validateNewTraining(txCtx, input); err != nil {
			return err
		}
		sub, err := s.subscriptionRepo.GetCoveringByClientForUpdate(txCtx, input.ClientID, input.StartTime, input.EndTime, 0)
		if err != nil {
			return err
		}
		input.SubscriptionID = &sub.ID
		if err := s.ensureSubscriptionCapacity(txCtx, input.ClientID, sub, 0); err != nil {
			return err
		}
		created, err = s.trainingRepo.Create(txCtx, input)
		return err
	})
	return created, err
}

func (s *ScheduleService) UpdateTraining(ctx context.Context, id int64, input domain.UpdateTrainingInput) (*domain.Training, error) {
	var updated *domain.Training
	err := s.tx.WithinTransaction(ctx, func(txCtx context.Context) error {
		current, err := s.trainingRepo.GetByIDForUpdate(txCtx, id)
		if err != nil {
			return err
		}
		if current.Status != domain.TrainingStatusScheduled {
			return fmt.Errorf("%w: completed/cancelled training is immutable", domain.ErrInvalidTransition)
		}

		if input.Status == "" {
			input.Status = current.Status
		}
		input.Title = strings.TrimSpace(input.Title)
		input.Description = strings.TrimSpace(input.Description)
		if input.Title == "" || len(input.Title) > 255 {
			return fmt.Errorf("%w: title is required and must fit 255 chars", domain.ErrInvalidInput)
		}
		if err := validateTrainingTimes(input.StartTime, input.EndTime); err != nil {
			return err
		}
		if err := validator.ValidateTrainingStatus(input.Status); err != nil {
			return err
		}
		if current.Status != input.Status && !canTransitionTraining(current.Status, input.Status) {
			return fmt.Errorf("%w: %s -> %s", domain.ErrInvalidTransition, current.Status, input.Status)
		}

		switch input.Status {
		case domain.TrainingStatusScheduled:
			if input.StartTime.Before(time.Now().Add(-time.Minute)) {
				return fmt.Errorf("%w: scheduled training cannot start in the past", domain.ErrInvalidInput)
			}
			if err := s.ensureActiveClientForUpdate(txCtx, current.ClientID); err != nil {
				return err
			}
			if input.TrainerID != nil {
				if err := s.ensureActiveTrainerForUpdate(txCtx, *input.TrainerID); err != nil {
					return err
				}
			}
			conflict, err := s.trainingRepo.HasScheduleConflict(txCtx, id, current.ClientID, input.TrainerID, input.StartTime, input.EndTime)
			if err != nil {
				return err
			}
			if conflict {
				return fmt.Errorf("%w: client or trainer already has an overlapping training", domain.ErrConflict)
			}
			sub, err := s.subscriptionRepo.GetCoveringByClientForUpdate(txCtx, current.ClientID, input.StartTime, input.EndTime, id)
			if err != nil {
				return err
			}
			input.SubscriptionID = &sub.ID
			if err := s.ensureSubscriptionCapacity(txCtx, current.ClientID, sub, id); err != nil {
				return err
			}

		case domain.TrainingStatusCompleted:
			if input.StartTime.After(time.Now()) {
				return fmt.Errorf("%w: cannot complete a future training", domain.ErrInvalidInput)
			}
			// Lock in the same user -> subscription order used by scheduling to avoid deadlocks.
			client, err := s.userRepo.GetByIDForUpdate(txCtx, current.ClientID)
			if err != nil {
				return err
			}
			if client.Role != domain.RoleClient {
				return domain.ErrForbidden
			}
			var sub *domain.ClientSubscription
			if current.SubscriptionID != nil {
				sub, err = s.subscriptionRepo.GetByIDForUpdate(txCtx, *current.SubscriptionID)
				if err != nil {
					return err
				}
				if sub.ClientID != current.ClientID || input.StartTime.Before(sub.StartDate) || input.EndTime.After(sub.EndDate) {
					return fmt.Errorf("%w: training is outside its reserved subscription", domain.ErrNoActiveSubscription)
				}
			} else {
				// Compatibility path for trainings created before subscription_id existed.
				sub, err = s.subscriptionRepo.GetCoveringByClientForUpdate(txCtx, current.ClientID, input.StartTime, input.EndTime, id)
				if err != nil {
					return err
				}
			}
			input.SubscriptionID = &sub.ID
			if err := s.subscriptionRepo.DecrementSessions(txCtx, sub.ID); err != nil {
				return err
			}
			if err := s.userRepo.IncrementVisits(txCtx, current.ClientID, input.StartTime); err != nil {
				return err
			}

		case domain.TrainingStatusCancelled:
			input.SubscriptionID = current.SubscriptionID
		}

		updated, err = s.trainingRepo.Update(txCtx, id, input)
		return err
	})
	return updated, err
}

func (s *ScheduleService) DeleteTraining(ctx context.Context, id int64) error {
	training, err := s.trainingRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if training.Status == domain.TrainingStatusCompleted {
		return fmt.Errorf("%w: completed training cannot be deleted", domain.ErrConflict)
	}
	return s.trainingRepo.Delete(ctx, id)
}

func (s *ScheduleService) validateNewTraining(ctx context.Context, input domain.CreateTrainingInput) error {
	input.Title = strings.TrimSpace(input.Title)
	if input.ClientID <= 0 || input.Title == "" || len(input.Title) > 255 {
		return fmt.Errorf("%w: client_id and title are required", domain.ErrInvalidInput)
	}
	if err := validateTrainingTimes(input.StartTime, input.EndTime); err != nil {
		return err
	}
	if input.StartTime.Before(time.Now().Add(-time.Minute)) {
		return fmt.Errorf("%w: new training cannot start in the past", domain.ErrInvalidInput)
	}
	if input.Status != domain.TrainingStatusScheduled {
		return fmt.Errorf("%w: new training must be scheduled", domain.ErrInvalidInput)
	}
	if err := s.ensureActiveClientForUpdate(ctx, input.ClientID); err != nil {
		return err
	}
	if input.TrainerID != nil {
		if err := s.ensureActiveTrainerForUpdate(ctx, *input.TrainerID); err != nil {
			return err
		}
	}
	conflict, err := s.trainingRepo.HasScheduleConflict(ctx, 0, input.ClientID, input.TrainerID, input.StartTime, input.EndTime)
	if err != nil {
		return err
	}
	if conflict {
		return fmt.Errorf("%w: client or trainer already has an overlapping training", domain.ErrConflict)
	}
	return nil
}

func (s *ScheduleService) ensureActiveClient(ctx context.Context, id int64) error {
	u, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if u.Role != domain.RoleClient || !u.IsActive {
		return domain.ErrForbidden
	}
	return nil
}

func (s *ScheduleService) ensureActiveClientForUpdate(ctx context.Context, id int64) error {
	u, err := s.userRepo.GetByIDForUpdate(ctx, id)
	if err != nil {
		return err
	}
	if u.Role != domain.RoleClient || !u.IsActive {
		return domain.ErrForbidden
	}
	return nil
}

func (s *ScheduleService) ensureActiveTrainerForUpdate(ctx context.Context, trainerID int64) error {
	t, err := s.trainerRepo.GetByIDForUpdate(ctx, trainerID)
	if err != nil {
		return err
	}
	if !t.IsActive {
		return fmt.Errorf("%w: trainer is inactive", domain.ErrForbidden)
	}
	return nil
}

func (s *ScheduleService) ensureSubscriptionCapacity(ctx context.Context, clientID int64, sub *domain.ClientSubscription, excludeTrainingID int64) error {
	if sub == nil || sub.ClientID != clientID || !sub.IsActive || sub.SessionsLeft != nil && *sub.SessionsLeft <= 0 {
		return domain.ErrNoActiveSubscription
	}
	if sub.SessionsLeft == nil {
		return nil
	}
	scheduled, err := s.trainingRepo.CountScheduledBySubscription(ctx, sub.ID, excludeTrainingID)
	if err != nil {
		return err
	}
	if scheduled >= *sub.SessionsLeft {
		return fmt.Errorf("%w: all remaining sessions are already scheduled", domain.ErrConflict)
	}
	return nil
}

// CancelMyTraining lets a client cancel their own scheduled training at least two hours before start.
// Sessions are only decremented on completion, so cancellation simply releases the reservation.
func (s *ScheduleService) CancelMyTraining(ctx context.Context, clientID, trainingID int64) error {
	return s.tx.WithinTransaction(ctx, func(txCtx context.Context) error {
		current, err := s.trainingRepo.GetByIDForUpdate(txCtx, trainingID)
		if err != nil {
			return err
		}
		if current.ClientID != clientID {
			return domain.ErrForbidden
		}
		if current.Status != domain.TrainingStatusScheduled {
			return fmt.Errorf("%w: training is not scheduled", domain.ErrInvalidTransition)
		}
		if time.Until(current.StartTime) < 2*time.Hour {
			return fmt.Errorf("%w: cancellation is allowed at least 2 hours before start", domain.ErrConflict)
		}
		_, err = s.trainingRepo.Update(txCtx, trainingID, domain.UpdateTrainingInput{
			SubscriptionID: current.SubscriptionID,
			TrainerID:      current.TrainerID,
			Title:          current.Title,
			Description:    current.Description,
			StartTime:      current.StartTime,
			EndTime:        current.EndTime,
			Status:         domain.TrainingStatusCancelled,
		})
		return err
	})
}

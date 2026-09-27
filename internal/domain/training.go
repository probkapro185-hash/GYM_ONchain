package domain

import "time"

type TrainingStatus string

const (
	TrainingStatusScheduled TrainingStatus = "scheduled"
	TrainingStatusCompleted TrainingStatus = "completed"
	TrainingStatusCancelled TrainingStatus = "cancelled"
	TrainingStatusPending   TrainingStatus = "pending"
)

type TrainingRequestStatus string

const (
	TrainingRequestPending   TrainingRequestStatus = "pending"
	TrainingRequestScheduled TrainingRequestStatus = "scheduled"
	TrainingRequestRejected  TrainingRequestStatus = "rejected"
)

type Training struct {
	ID             int64          `json:"id"`
	ClientID       int64          `json:"client_id"`
	SubscriptionID *int64         `json:"subscription_id,omitempty"`
	TrainerID      *int64         `json:"trainer_id,omitempty"`
	Title          string         `json:"title"`
	Description    string         `json:"description"`
	StartTime      time.Time      `json:"start_time"`
	EndTime        time.Time      `json:"end_time"`
	Status         TrainingStatus `json:"status"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	ClientName     string         `json:"client_name,omitempty"`
	TrainerName    string         `json:"trainer_name,omitempty"`
}

type TrainingRequest struct {
	TrainerID   *int64                `json:"trainer_id,omitempty"`
	TrainerName string                `json:"trainer_name,omitempty"`
	ID          int64                 `json:"id"`
	ClientID    int64                 `json:"client_id"`
	PreferredAt time.Time             `json:"preferred_at"`
	Comment     string                `json:"comment"`
	Status      TrainingRequestStatus `json:"status"`
	CreatedAt   time.Time             `json:"created_at"`
	ClientName  string                `json:"client_name,omitempty"`
}

type CreateTrainingRequestInput struct {
	TrainerID   *int64    `json:"trainer_id,omitempty"`
	PreferredAt time.Time `json:"preferred_at"`
	Comment     string    `json:"comment"`
}

type CreateTrainingInput struct {
	ClearTrainer   bool           `json:"clear_trainer,omitempty"`
	ClientID       int64          `json:"client_id"`
	SubscriptionID *int64         `json:"-"`
	TrainerID      *int64         `json:"trainer_id,omitempty"`
	Title          string         `json:"title"`
	Description    string         `json:"description"`
	StartTime      time.Time      `json:"start_time"`
	EndTime        time.Time      `json:"end_time"`
	Status         TrainingStatus `json:"status"`
}

type UpdateTrainingInput struct {
	SubscriptionID *int64         `json:"-"`
	TrainerID      *int64         `json:"trainer_id,omitempty"`
	Title          string         `json:"title"`
	Description    string         `json:"description"`
	StartTime      time.Time      `json:"start_time"`
	EndTime        time.Time      `json:"end_time"`
	Status         TrainingStatus `json:"status"`
}

type ScheduleFilter struct {
	ClientID  *int64
	TrainerID *int64
	DateFrom  *time.Time
	DateTo    *time.Time
	Status    TrainingStatus
}

package domain

import "time"

type ClientProgress struct {
	ID         int64     `json:"id"`
	ClientID   int64     `json:"client_id"`
	RecordedBy int64     `json:"recorded_by"`
	Weight     *float64  `json:"weight,omitempty"`
	BodyFat    *float64  `json:"body_fat,omitempty"`
	Waist      *float64  `json:"waist,omitempty"`
	Chest      *float64  `json:"chest,omitempty"`
	Hips       *float64  `json:"hips,omitempty"`
	Comment    string    `json:"comment"`
	RecordedAt time.Time `json:"recorded_at"`
	AuthorName string    `json:"author_name,omitempty"`
}

type CreateProgressInput struct {
	Weight  *float64 `json:"weight,omitempty"`
	BodyFat *float64 `json:"body_fat,omitempty"`
	Waist   *float64 `json:"waist,omitempty"`
	Chest   *float64 `json:"chest,omitempty"`
	Hips    *float64 `json:"hips,omitempty"`
	Comment string   `json:"comment"`
}

type ClientNote struct {
	ID         int64     `json:"id"`
	ClientID   int64     `json:"client_id"`
	AuthorID   int64     `json:"author_id"`
	Note       string    `json:"note"`
	CreatedAt  time.Time `json:"created_at"`
	AuthorName string    `json:"author_name,omitempty"`
}

type CreateNoteInput struct {
	Note string `json:"note"`
}

type StaffTask struct {
	ID          int64      `json:"id"`
	ClientID    *int64     `json:"client_id,omitempty"`
	AssigneeID  int64      `json:"assignee_id"`
	CreatedBy   int64      `json:"created_by"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	DueAt       *time.Time `json:"due_at,omitempty"`
	Status      string     `json:"status"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	ClientName  string     `json:"client_name,omitempty"`
	Assignee    string     `json:"assignee_name,omitempty"`
	IsOverdue   bool       `json:"is_overdue"`
}

type CreateTaskInput struct {
	ClientID    *int64     `json:"client_id,omitempty"`
	AssigneeID  int64      `json:"assignee_id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	DueAt       *time.Time `json:"due_at,omitempty"`
}

type UpdateTaskStatusInput struct {
	Status string `json:"status"`
}

type Notification struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	Title     string    `json:"title"`
	Message   string    `json:"message"`
	IsRead    bool      `json:"is_read"`
	CreatedAt time.Time `json:"created_at"`
}

type CreateNotificationInput struct {
	UserID  int64  `json:"user_id"`
	Title   string `json:"title"`
	Message string `json:"message"`
}

type DashboardSummary struct {
	ActiveClients         int64                `json:"active_clients"`
	ActiveSubscriptions   int64                `json:"active_subscriptions"`
	FrozenSubscriptions   int64                `json:"frozen_subscriptions"`
	TrainingsToday        int64                `json:"trainings_today"`
	PendingApplications   int64                `json:"pending_applications"`
	OpenTasks             int64                `json:"open_tasks"`
	AtRiskClients         int64                `json:"at_risk_clients"`
	ExpiringSubscriptions int64                `json:"expiring_subscriptions"`
	Revenue30Days         float64              `json:"revenue_30_days"`
	CompletedTrainings30d int64                `json:"completed_trainings_30d"`
	TrainerUtilization    []TrainerUtilization `json:"trainer_utilization"`
}

type TrainerUtilization struct {
	TrainerID      int64   `json:"trainer_id"`
	TrainerName    string  `json:"trainer_name"`
	Scheduled30d   int64   `json:"scheduled_30d"`
	Completed30d   int64   `json:"completed_30d"`
	CompletionRate float64 `json:"completion_rate"`
}

type AuditRecord struct {
	ID         int64     `json:"id"`
	ActorID    *int64    `json:"actor_id,omitempty"`
	ActorRole  string    `json:"actor_role"`
	RequestID  string    `json:"request_id"`
	Method     string    `json:"method"`
	Path       string    `json:"path"`
	StatusCode int       `json:"status_code"`
	CreatedAt  time.Time `json:"created_at"`
	ActorName  string    `json:"actor_name,omitempty"`
}

type ClientCRMCard struct {
	User          *User                 `json:"user"`
	Subscriptions []*ClientSubscription `json:"subscriptions"`
	Progress      []*ClientProgress     `json:"progress"`
	Notes         []*ClientNote         `json:"notes"`
	Tasks         []*StaffTask          `json:"tasks"`
	LastVisitAt   *time.Time            `json:"last_visit_at,omitempty"`
	DaysInactive  *int                  `json:"days_inactive,omitempty"`
}

type ExtendSubscriptionInput struct {
	Days int `json:"days"`
}

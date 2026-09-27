package repository

import (
	"context"
	"time"

	"github.com/sfedu-crm/internal/domain"
)

type TransactionManager interface {
	WithinTransaction(ctx context.Context, fn func(context.Context) error) error
}

type UserRepository interface {
	Create(ctx context.Context, input domain.CreateUserInput) (*domain.User, error)
	GetByID(ctx context.Context, id int64) (*domain.User, error)
	GetByIDForUpdate(ctx context.Context, id int64) (*domain.User, error)
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	GetByPhone(ctx context.Context, phone string) (*domain.User, error)
	List(ctx context.Context, filter UserFilter) ([]*domain.User, error)
	Update(ctx context.Context, id int64, input domain.UpdateUserInput) (*domain.User, error)
	UpdatePassword(ctx context.Context, id int64, passwordHash string) error
	AddBalance(ctx context.Context, id int64, amountCents int64) error
	DebitBalance(ctx context.Context, id int64, amountCents int64) error
	IncrementVisits(ctx context.Context, id int64, visitedAt time.Time) error
	SetActive(ctx context.Context, id int64, active bool) error
}

type UserFilter struct {
	Role     *domain.Role
	IsActive *bool
	Search   string
}

type ApplicationRepository interface {
	Create(ctx context.Context, input domain.CreateApplicationInput) (*domain.ApplicationRequest, error)
	GetByID(ctx context.Context, id int64) (*domain.ApplicationRequest, error)
	GetByIDForUpdate(ctx context.Context, id int64) (*domain.ApplicationRequest, error)
	List(ctx context.Context, status domain.ApplicationStatus) ([]*domain.ApplicationRequest, error)
	UpdateStatus(ctx context.Context, id int64, status domain.ApplicationStatus) error
	Delete(ctx context.Context, id int64) error
}

type TrainerRepository interface {
	Create(ctx context.Context, input domain.CreateTrainerInput) (*domain.Trainer, error)
	GetByID(ctx context.Context, id int64) (*domain.Trainer, error)
	GetByIDForUpdate(ctx context.Context, id int64) (*domain.Trainer, error)
	GetByUserID(ctx context.Context, userID int64) (*domain.Trainer, error)
	List(ctx context.Context, filter TrainerFilter) ([]*domain.Trainer, error)
	Update(ctx context.Context, id int64, input domain.UpdateTrainerInput) (*domain.Trainer, error)
}

type TrainerFilter struct {
	Specialization *domain.TrainerSpecialization
	IsActive       *bool
	Search         string
}

type TrainingRepository interface {
	Create(ctx context.Context, input domain.CreateTrainingInput) (*domain.Training, error)
	GetByID(ctx context.Context, id int64) (*domain.Training, error)
	GetByIDForUpdate(ctx context.Context, id int64) (*domain.Training, error)
	List(ctx context.Context, filter domain.ScheduleFilter) ([]*domain.Training, error)
	Update(ctx context.Context, id int64, input domain.UpdateTrainingInput) (*domain.Training, error)
	Delete(ctx context.Context, id int64) error
	HasScheduleConflict(ctx context.Context, excludeID, clientID int64, trainerID *int64, start, end time.Time) (bool, error)
	CountScheduledBySubscription(ctx context.Context, subscriptionID, excludeTrainingID int64) (int, error)
}

type TrainingRequestRepository interface {
	Create(ctx context.Context, clientID int64, input domain.CreateTrainingRequestInput) (*domain.TrainingRequest, error)
	GetByID(ctx context.Context, id int64) (*domain.TrainingRequest, error)
	GetByIDForUpdate(ctx context.Context, id int64) (*domain.TrainingRequest, error)
	ListByClient(ctx context.Context, clientID int64) ([]*domain.TrainingRequest, error)
	ListPending(ctx context.Context) ([]*domain.TrainingRequest, error)
	UpdateStatus(ctx context.Context, id int64, status domain.TrainingRequestStatus) error
}

type ProductRepository interface {
	SetPhoto(ctx context.Context, id int64, photoURL string) (*domain.Product, error)
	Create(ctx context.Context, input domain.CreateProductInput) (*domain.Product, error)
	GetByID(ctx context.Context, id int64) (*domain.Product, error)
	GetByIDForUpdate(ctx context.Context, id int64) (*domain.Product, error)
	List(ctx context.Context, filter ProductFilter) ([]*domain.Product, error)
	Update(ctx context.Context, id int64, input domain.CreateProductInput) (*domain.Product, error)
	SetActive(ctx context.Context, id int64, active bool) error
}

type ProductFilter struct {
	Category *domain.ProductCategory
	IsActive *bool
}

type SubscriptionRepository interface {
	Create(ctx context.Context, clientID int64, product *domain.Product) (*domain.ClientSubscription, error)
	GetByID(ctx context.Context, id int64) (*domain.ClientSubscription, error)
	GetByIDForUpdate(ctx context.Context, id int64) (*domain.ClientSubscription, error)
	ListByClient(ctx context.Context, clientID int64) ([]*domain.ClientSubscription, error)
	GetActiveByClient(ctx context.Context, clientID int64) (*domain.ClientSubscription, error)
	GetCoveringByClient(ctx context.Context, clientID int64, start, end time.Time) (*domain.ClientSubscription, error)
	GetCoveringByClientForUpdate(ctx context.Context, clientID int64, start, end time.Time, excludeTrainingID int64) (*domain.ClientSubscription, error)
	DecrementSessions(ctx context.Context, id int64) error
	Deactivate(ctx context.Context, id int64) error
}

type PaymentRepository interface {
	Create(ctx context.Context, input domain.CreatePaymentInput) (*domain.Payment, error)
	GetByID(ctx context.Context, id int64) (*domain.Payment, error)
	List(ctx context.Context, filter domain.PaymentFilter) ([]*domain.Payment, error)
	GetSummary(ctx context.Context, filter domain.PaymentFilter) (*domain.FinanceSummary, error)
}

type OrderRepository interface {
	Create(ctx context.Context, clientID int64, product *domain.Product) (*domain.Order, error)
	ListByClient(ctx context.Context, clientID int64) ([]*domain.Order, error)
}

type CRMRepository interface {
	AddProgress(ctx context.Context, clientID, actorID int64, input domain.CreateProgressInput) (*domain.ClientProgress, error)
	ListProgress(ctx context.Context, clientID int64, limit int) ([]*domain.ClientProgress, error)
	DeleteProgress(ctx context.Context, id int64) error
	AddNote(ctx context.Context, clientID, actorID int64, note string) (*domain.ClientNote, error)
	ListNotes(ctx context.Context, clientID int64, limit int) ([]*domain.ClientNote, error)
	DeleteNote(ctx context.Context, id int64) error
	CreateTask(ctx context.Context, actorID int64, input domain.CreateTaskInput) (*domain.StaffTask, error)
	ListTasks(ctx context.Context, assigneeID *int64, clientID *int64, includeDone bool, limit int) ([]*domain.StaffTask, error)
	UpdateTaskStatus(ctx context.Context, id int64, status string, assigneeID *int64) (*domain.StaffTask, error)
	CreateNotification(ctx context.Context, input domain.CreateNotificationInput) (*domain.Notification, error)
	ListNotifications(ctx context.Context, userID int64, limit int) ([]*domain.Notification, error)
	MarkNotificationRead(ctx context.Context, id, userID int64) error
	Dashboard(ctx context.Context) (*domain.DashboardSummary, error)
	ListAudit(ctx context.Context, limit int) ([]*domain.AuditRecord, error)
	RecordAudit(ctx context.Context, actorID *int64, actorRole, requestID, method, path string, statusCode int) error
	FreezeSubscription(ctx context.Context, id int64) (*domain.ClientSubscription, error)
	UnfreezeSubscription(ctx context.Context, id int64) (*domain.ClientSubscription, error)
	ExtendSubscription(ctx context.Context, id int64, days int) (*domain.ClientSubscription, error)
	HasFutureTrainingForSubscription(ctx context.Context, subscriptionID int64) (bool, error)
}

type AIRepository interface {
	CreateConversation(ctx context.Context, userID int64, title string) (*domain.AIConversation, error)
	GetConversation(ctx context.Context, conversationID, userID int64) (*domain.AIConversation, error)
	ListConversations(ctx context.Context, userID int64, limit int) ([]*domain.AIConversation, error)
	DeleteConversation(ctx context.Context, conversationID, userID int64) error
	AddMessage(ctx context.Context, conversationID int64, role, content string, sources []domain.AISource) (*domain.AIMessage, error)
	ListMessages(ctx context.Context, conversationID, userID int64, limit int) ([]*domain.AIMessage, error)

	UpsertDocument(ctx context.Context, title, source, checksum string) (documentID int64, changed bool, err error)
	ReplaceDocumentChunks(ctx context.Context, documentID int64, chunks []domain.AIChunkInput) error
	SearchChunks(ctx context.Context, embedding []float64, limit int) ([]domain.AIRetrievedChunk, error)
	ListDocuments(ctx context.Context) ([]*domain.AIDocument, error)
	DeleteDocument(ctx context.Context, documentID int64) error
	GetPersonalContext(ctx context.Context, userID int64, role domain.Role) (*domain.AIPersonalContext, error)
}

type AccountAccessRepository interface {
	CreatePendingUser(ctx context.Context, input domain.CreateUserInput) (*domain.User, error)
	CreateToken(ctx context.Context, userID int64, purpose domain.AccountTokenPurpose, tokenHash string, expiresAt time.Time) error
	GetTokenForUpdate(ctx context.Context, tokenHash string, purpose domain.AccountTokenPurpose) (*domain.AccountToken, error)
	MarkTokenUsed(ctx context.Context, tokenID int64) error
	InvalidateTokens(ctx context.Context, userID int64, purpose domain.AccountTokenPurpose) error
	ActivateUser(ctx context.Context, userID int64, passwordHash string) error
	ResetPassword(ctx context.Context, userID int64, passwordHash string) error
}

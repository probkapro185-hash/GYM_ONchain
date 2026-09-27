package service

import (
	"context"
	"time"

	"github.com/sfedu-crm/internal/domain"
	"github.com/sfedu-crm/internal/repository"
)

type fakeTx struct {
	err   error
	calls int
}

func (f *fakeTx) WithinTransaction(ctx context.Context, fn func(context.Context) error) error {
	f.calls++
	if f.err != nil {
		return f.err
	}
	return fn(ctx)
}

type fakeUserRepo struct {
	createFn           func(context.Context, domain.CreateUserInput) (*domain.User, error)
	getByIDFn          func(context.Context, int64) (*domain.User, error)
	getByIDForUpdateFn func(context.Context, int64) (*domain.User, error)
	getByEmailFn       func(context.Context, string) (*domain.User, error)
	getByPhoneFn       func(context.Context, string) (*domain.User, error)
	listFn             func(context.Context, repository.UserFilter) ([]*domain.User, error)
	updateFn           func(context.Context, int64, domain.UpdateUserInput) (*domain.User, error)
	updatePasswordFn   func(context.Context, int64, string) error
	addBalanceFn       func(context.Context, int64, int64) error
	debitBalanceFn     func(context.Context, int64, int64) error
	incrementVisitsFn  func(context.Context, int64, time.Time) error
	setActiveFn        func(context.Context, int64, bool) error
}

func (f *fakeUserRepo) Create(c context.Context, i domain.CreateUserInput) (*domain.User, error) {
	if f.createFn != nil {
		return f.createFn(c, i)
	}
	return nil, domain.ErrNotFound
}
func (f *fakeUserRepo) GetByID(c context.Context, id int64) (*domain.User, error) {
	if f.getByIDFn != nil {
		return f.getByIDFn(c, id)
	}
	return nil, domain.ErrNotFound
}
func (f *fakeUserRepo) GetByIDForUpdate(c context.Context, id int64) (*domain.User, error) {
	if f.getByIDForUpdateFn != nil {
		return f.getByIDForUpdateFn(c, id)
	}
	return f.GetByID(c, id)
}
func (f *fakeUserRepo) GetByEmail(c context.Context, s string) (*domain.User, error) {
	if f.getByEmailFn != nil {
		return f.getByEmailFn(c, s)
	}
	return nil, domain.ErrNotFound
}
func (f *fakeUserRepo) GetByPhone(c context.Context, s string) (*domain.User, error) {
	if f.getByPhoneFn != nil {
		return f.getByPhoneFn(c, s)
	}
	return nil, domain.ErrNotFound
}
func (f *fakeUserRepo) List(c context.Context, fl repository.UserFilter) ([]*domain.User, error) {
	if f.listFn != nil {
		return f.listFn(c, fl)
	}
	return nil, nil
}
func (f *fakeUserRepo) Update(c context.Context, id int64, i domain.UpdateUserInput) (*domain.User, error) {
	if f.updateFn != nil {
		return f.updateFn(c, id, i)
	}
	return nil, domain.ErrNotFound
}
func (f *fakeUserRepo) UpdatePassword(c context.Context, id int64, s string) error {
	if f.updatePasswordFn != nil {
		return f.updatePasswordFn(c, id, s)
	}
	return nil
}
func (f *fakeUserRepo) AddBalance(c context.Context, id int64, a int64) error {
	if f.addBalanceFn != nil {
		return f.addBalanceFn(c, id, a)
	}
	return nil
}
func (f *fakeUserRepo) DebitBalance(c context.Context, id int64, a int64) error {
	if f.debitBalanceFn != nil {
		return f.debitBalanceFn(c, id, a)
	}
	return nil
}
func (f *fakeUserRepo) IncrementVisits(c context.Context, id int64, visitedAt time.Time) error {
	if f.incrementVisitsFn != nil {
		return f.incrementVisitsFn(c, id, visitedAt)
	}
	return nil
}
func (f *fakeUserRepo) SetActive(c context.Context, id int64, a bool) error {
	if f.setActiveFn != nil {
		return f.setActiveFn(c, id, a)
	}
	return nil
}

type fakeApplicationRepo struct {
	createFn           func(context.Context, domain.CreateApplicationInput) (*domain.ApplicationRequest, error)
	getByIDFn          func(context.Context, int64) (*domain.ApplicationRequest, error)
	getByIDForUpdateFn func(context.Context, int64) (*domain.ApplicationRequest, error)
	listFn             func(context.Context, domain.ApplicationStatus) ([]*domain.ApplicationRequest, error)
	updateStatusFn     func(context.Context, int64, domain.ApplicationStatus) error
	deleteFn           func(context.Context, int64) error
}

func (f *fakeApplicationRepo) Create(c context.Context, i domain.CreateApplicationInput) (*domain.ApplicationRequest, error) {
	if f.createFn != nil {
		return f.createFn(c, i)
	}
	return nil, domain.ErrNotFound
}
func (f *fakeApplicationRepo) GetByID(c context.Context, id int64) (*domain.ApplicationRequest, error) {
	if f.getByIDFn != nil {
		return f.getByIDFn(c, id)
	}
	return nil, domain.ErrNotFound
}
func (f *fakeApplicationRepo) GetByIDForUpdate(c context.Context, id int64) (*domain.ApplicationRequest, error) {
	if f.getByIDForUpdateFn != nil {
		return f.getByIDForUpdateFn(c, id)
	}
	return f.GetByID(c, id)
}
func (f *fakeApplicationRepo) List(c context.Context, s domain.ApplicationStatus) ([]*domain.ApplicationRequest, error) {
	if f.listFn != nil {
		return f.listFn(c, s)
	}
	return nil, nil
}
func (f *fakeApplicationRepo) UpdateStatus(c context.Context, id int64, s domain.ApplicationStatus) error {
	if f.updateStatusFn != nil {
		return f.updateStatusFn(c, id, s)
	}
	return nil
}
func (f *fakeApplicationRepo) Delete(c context.Context, id int64) error {
	if f.deleteFn != nil {
		return f.deleteFn(c, id)
	}
	return nil
}

type fakeTrainerRepo struct {
	createFn           func(context.Context, domain.CreateTrainerInput) (*domain.Trainer, error)
	getByIDFn          func(context.Context, int64) (*domain.Trainer, error)
	getByIDForUpdateFn func(context.Context, int64) (*domain.Trainer, error)
	getByUserIDFn      func(context.Context, int64) (*domain.Trainer, error)
	listFn             func(context.Context, repository.TrainerFilter) ([]*domain.Trainer, error)
	updateFn           func(context.Context, int64, domain.UpdateTrainerInput) (*domain.Trainer, error)
}

func (f *fakeTrainerRepo) Create(c context.Context, i domain.CreateTrainerInput) (*domain.Trainer, error) {
	if f.createFn != nil {
		return f.createFn(c, i)
	}
	return nil, domain.ErrNotFound
}
func (f *fakeTrainerRepo) GetByID(c context.Context, id int64) (*domain.Trainer, error) {
	if f.getByIDFn != nil {
		return f.getByIDFn(c, id)
	}
	return nil, domain.ErrNotFound
}
func (f *fakeTrainerRepo) GetByIDForUpdate(c context.Context, id int64) (*domain.Trainer, error) {
	if f.getByIDForUpdateFn != nil {
		return f.getByIDForUpdateFn(c, id)
	}
	return f.GetByID(c, id)
}
func (f *fakeTrainerRepo) GetByUserID(c context.Context, id int64) (*domain.Trainer, error) {
	if f.getByUserIDFn != nil {
		return f.getByUserIDFn(c, id)
	}
	return nil, domain.ErrNotFound
}
func (f *fakeTrainerRepo) List(c context.Context, fl repository.TrainerFilter) ([]*domain.Trainer, error) {
	if f.listFn != nil {
		return f.listFn(c, fl)
	}
	return nil, nil
}
func (f *fakeTrainerRepo) Update(c context.Context, id int64, i domain.UpdateTrainerInput) (*domain.Trainer, error) {
	if f.updateFn != nil {
		return f.updateFn(c, id, i)
	}
	return nil, domain.ErrNotFound
}

type fakeTrainingRepo struct {
	createFn           func(context.Context, domain.CreateTrainingInput) (*domain.Training, error)
	getByIDFn          func(context.Context, int64) (*domain.Training, error)
	getByIDForUpdateFn func(context.Context, int64) (*domain.Training, error)
	listFn             func(context.Context, domain.ScheduleFilter) ([]*domain.Training, error)
	updateFn           func(context.Context, int64, domain.UpdateTrainingInput) (*domain.Training, error)
	deleteFn           func(context.Context, int64) error
	conflictFn         func(context.Context, int64, int64, *int64, time.Time, time.Time) (bool, error)
	countFn            func(context.Context, int64, int64) (int, error)
}

func (f *fakeTrainingRepo) Create(c context.Context, i domain.CreateTrainingInput) (*domain.Training, error) {
	if f.createFn != nil {
		return f.createFn(c, i)
	}
	return nil, domain.ErrNotFound
}
func (f *fakeTrainingRepo) GetByID(c context.Context, id int64) (*domain.Training, error) {
	if f.getByIDFn != nil {
		return f.getByIDFn(c, id)
	}
	return nil, domain.ErrNotFound
}
func (f *fakeTrainingRepo) GetByIDForUpdate(c context.Context, id int64) (*domain.Training, error) {
	if f.getByIDForUpdateFn != nil {
		return f.getByIDForUpdateFn(c, id)
	}
	return f.GetByID(c, id)
}
func (f *fakeTrainingRepo) List(c context.Context, fl domain.ScheduleFilter) ([]*domain.Training, error) {
	if f.listFn != nil {
		return f.listFn(c, fl)
	}
	return nil, nil
}
func (f *fakeTrainingRepo) Update(c context.Context, id int64, i domain.UpdateTrainingInput) (*domain.Training, error) {
	if f.updateFn != nil {
		return f.updateFn(c, id, i)
	}
	return nil, domain.ErrNotFound
}
func (f *fakeTrainingRepo) Delete(c context.Context, id int64) error {
	if f.deleteFn != nil {
		return f.deleteFn(c, id)
	}
	return nil
}
func (f *fakeTrainingRepo) HasScheduleConflict(c context.Context, e, cl int64, tr *int64, s, en time.Time) (bool, error) {
	if f.conflictFn != nil {
		return f.conflictFn(c, e, cl, tr, s, en)
	}
	return false, nil
}
func (f *fakeTrainingRepo) CountScheduledBySubscription(c context.Context, s, e int64) (int, error) {
	if f.countFn != nil {
		return f.countFn(c, s, e)
	}
	return 0, nil
}

type fakeTrainingRequestRepo struct {
	createFn           func(context.Context, int64, domain.CreateTrainingRequestInput) (*domain.TrainingRequest, error)
	getByIDFn          func(context.Context, int64) (*domain.TrainingRequest, error)
	getByIDForUpdateFn func(context.Context, int64) (*domain.TrainingRequest, error)
	listByClientFn     func(context.Context, int64) ([]*domain.TrainingRequest, error)
	listPendingFn      func(context.Context) ([]*domain.TrainingRequest, error)
	updateStatusFn     func(context.Context, int64, domain.TrainingRequestStatus) error
}

func (f *fakeTrainingRequestRepo) Create(c context.Context, id int64, i domain.CreateTrainingRequestInput) (*domain.TrainingRequest, error) {
	if f.createFn != nil {
		return f.createFn(c, id, i)
	}
	return nil, domain.ErrNotFound
}
func (f *fakeTrainingRequestRepo) GetByID(c context.Context, id int64) (*domain.TrainingRequest, error) {
	if f.getByIDFn != nil {
		return f.getByIDFn(c, id)
	}
	return nil, domain.ErrNotFound
}
func (f *fakeTrainingRequestRepo) GetByIDForUpdate(c context.Context, id int64) (*domain.TrainingRequest, error) {
	if f.getByIDForUpdateFn != nil {
		return f.getByIDForUpdateFn(c, id)
	}
	return f.GetByID(c, id)
}
func (f *fakeTrainingRequestRepo) ListByClient(c context.Context, id int64) ([]*domain.TrainingRequest, error) {
	if f.listByClientFn != nil {
		return f.listByClientFn(c, id)
	}
	return nil, nil
}
func (f *fakeTrainingRequestRepo) ListPending(c context.Context) ([]*domain.TrainingRequest, error) {
	if f.listPendingFn != nil {
		return f.listPendingFn(c)
	}
	return nil, nil
}
func (f *fakeTrainingRequestRepo) UpdateStatus(c context.Context, id int64, s domain.TrainingRequestStatus) error {
	if f.updateStatusFn != nil {
		return f.updateStatusFn(c, id, s)
	}
	return nil
}

type fakeProductRepo struct {
	setPhotoFn         func(context.Context, int64, string) (*domain.Product, error)
	createFn           func(context.Context, domain.CreateProductInput) (*domain.Product, error)
	getByIDFn          func(context.Context, int64) (*domain.Product, error)
	getByIDForUpdateFn func(context.Context, int64) (*domain.Product, error)
	listFn             func(context.Context, repository.ProductFilter) ([]*domain.Product, error)
	updateFn           func(context.Context, int64, domain.CreateProductInput) (*domain.Product, error)
	setActiveFn        func(context.Context, int64, bool) error
}

func (f *fakeProductRepo) SetPhoto(c context.Context, id int64, url string) (*domain.Product, error) {
	if f.setPhotoFn != nil {
		return f.setPhotoFn(c, id, url)
	}
	return nil, domain.ErrNotFound
}

func (f *fakeProductRepo) Create(c context.Context, i domain.CreateProductInput) (*domain.Product, error) {
	if f.createFn != nil {
		return f.createFn(c, i)
	}
	return nil, domain.ErrNotFound
}
func (f *fakeProductRepo) GetByID(c context.Context, id int64) (*domain.Product, error) {
	if f.getByIDFn != nil {
		return f.getByIDFn(c, id)
	}
	return nil, domain.ErrNotFound
}
func (f *fakeProductRepo) GetByIDForUpdate(c context.Context, id int64) (*domain.Product, error) {
	if f.getByIDForUpdateFn != nil {
		return f.getByIDForUpdateFn(c, id)
	}
	return f.GetByID(c, id)
}
func (f *fakeProductRepo) List(c context.Context, fl repository.ProductFilter) ([]*domain.Product, error) {
	if f.listFn != nil {
		return f.listFn(c, fl)
	}
	return nil, nil
}
func (f *fakeProductRepo) Update(c context.Context, id int64, i domain.CreateProductInput) (*domain.Product, error) {
	if f.updateFn != nil {
		return f.updateFn(c, id, i)
	}
	return nil, domain.ErrNotFound
}
func (f *fakeProductRepo) SetActive(c context.Context, id int64, a bool) error {
	if f.setActiveFn != nil {
		return f.setActiveFn(c, id, a)
	}
	return nil
}

type fakeSubscriptionRepo struct {
	createFn                       func(context.Context, int64, *domain.Product) (*domain.ClientSubscription, error)
	getByIDFn                      func(context.Context, int64) (*domain.ClientSubscription, error)
	getByIDForUpdateFn             func(context.Context, int64) (*domain.ClientSubscription, error)
	listByClientFn                 func(context.Context, int64) ([]*domain.ClientSubscription, error)
	getActiveByClientFn            func(context.Context, int64) (*domain.ClientSubscription, error)
	getCoveringByClientFn          func(context.Context, int64, time.Time, time.Time) (*domain.ClientSubscription, error)
	getCoveringByClientForUpdateFn func(context.Context, int64, time.Time, time.Time) (*domain.ClientSubscription, error)
	decrementSessionsFn            func(context.Context, int64) error
	deactivateFn                   func(context.Context, int64) error
}

func (f *fakeSubscriptionRepo) Create(c context.Context, id int64, p *domain.Product) (*domain.ClientSubscription, error) {
	if f.createFn != nil {
		return f.createFn(c, id, p)
	}
	return nil, domain.ErrNotFound
}
func (f *fakeSubscriptionRepo) GetByID(c context.Context, id int64) (*domain.ClientSubscription, error) {
	if f.getByIDFn != nil {
		return f.getByIDFn(c, id)
	}
	return nil, domain.ErrNotFound
}
func (f *fakeSubscriptionRepo) GetByIDForUpdate(c context.Context, id int64) (*domain.ClientSubscription, error) {
	if f.getByIDForUpdateFn != nil {
		return f.getByIDForUpdateFn(c, id)
	}
	return f.GetByID(c, id)
}
func (f *fakeSubscriptionRepo) ListByClient(c context.Context, id int64) ([]*domain.ClientSubscription, error) {
	if f.listByClientFn != nil {
		return f.listByClientFn(c, id)
	}
	return nil, nil
}
func (f *fakeSubscriptionRepo) GetActiveByClient(c context.Context, id int64) (*domain.ClientSubscription, error) {
	if f.getActiveByClientFn != nil {
		return f.getActiveByClientFn(c, id)
	}
	return nil, domain.ErrNotFound
}
func (f *fakeSubscriptionRepo) GetCoveringByClient(c context.Context, id int64, s, e time.Time) (*domain.ClientSubscription, error) {
	if f.getCoveringByClientFn != nil {
		return f.getCoveringByClientFn(c, id, s, e)
	}
	return nil, domain.ErrNoActiveSubscription
}
func (f *fakeSubscriptionRepo) GetCoveringByClientForUpdate(c context.Context, id int64, s, e time.Time, excludeTrainingID int64) (*domain.ClientSubscription, error) {
	if f.getCoveringByClientForUpdateFn != nil {
		return f.getCoveringByClientForUpdateFn(c, id, s, e)
	}
	return f.GetCoveringByClient(c, id, s, e)
}
func (f *fakeSubscriptionRepo) DecrementSessions(c context.Context, id int64) error {
	if f.decrementSessionsFn != nil {
		return f.decrementSessionsFn(c, id)
	}
	return nil
}
func (f *fakeSubscriptionRepo) Deactivate(c context.Context, id int64) error {
	if f.deactivateFn != nil {
		return f.deactivateFn(c, id)
	}
	return nil
}

type fakePaymentRepo struct {
	createFn  func(context.Context, domain.CreatePaymentInput) (*domain.Payment, error)
	getByIDFn func(context.Context, int64) (*domain.Payment, error)
	listFn    func(context.Context, domain.PaymentFilter) ([]*domain.Payment, error)
	summaryFn func(context.Context, domain.PaymentFilter) (*domain.FinanceSummary, error)
}

func (f *fakePaymentRepo) Create(c context.Context, i domain.CreatePaymentInput) (*domain.Payment, error) {
	if f.createFn != nil {
		return f.createFn(c, i)
	}
	return nil, domain.ErrNotFound
}
func (f *fakePaymentRepo) GetByID(c context.Context, id int64) (*domain.Payment, error) {
	if f.getByIDFn != nil {
		return f.getByIDFn(c, id)
	}
	return nil, domain.ErrNotFound
}
func (f *fakePaymentRepo) List(c context.Context, fl domain.PaymentFilter) ([]*domain.Payment, error) {
	if f.listFn != nil {
		return f.listFn(c, fl)
	}
	return nil, nil
}
func (f *fakePaymentRepo) GetSummary(c context.Context, fl domain.PaymentFilter) (*domain.FinanceSummary, error) {
	if f.summaryFn != nil {
		return f.summaryFn(c, fl)
	}
	return &domain.FinanceSummary{}, nil
}

type fakeOrderRepo struct {
	createFn func(context.Context, int64, *domain.Product) (*domain.Order, error)
	listFn   func(context.Context, int64) ([]*domain.Order, error)
}

func (f *fakeOrderRepo) Create(c context.Context, id int64, p *domain.Product) (*domain.Order, error) {
	if f.createFn != nil {
		return f.createFn(c, id, p)
	}
	return nil, domain.ErrNotFound
}
func (f *fakeOrderRepo) ListByClient(c context.Context, id int64) ([]*domain.Order, error) {
	if f.listFn != nil {
		return f.listFn(c, id)
	}
	return nil, nil
}

type fakeCRMRepo struct {
	addProgressFn          func(context.Context, int64, int64, domain.CreateProgressInput) (*domain.ClientProgress, error)
	listProgressFn         func(context.Context, int64, int) ([]*domain.ClientProgress, error)
	deleteProgressFn       func(context.Context, int64) error
	addNoteFn              func(context.Context, int64, int64, string) (*domain.ClientNote, error)
	listNotesFn            func(context.Context, int64, int) ([]*domain.ClientNote, error)
	deleteNoteFn           func(context.Context, int64) error
	createTaskFn           func(context.Context, int64, domain.CreateTaskInput) (*domain.StaffTask, error)
	listTasksFn            func(context.Context, *int64, *int64, bool, int) ([]*domain.StaffTask, error)
	updateTaskStatusFn     func(context.Context, int64, string, *int64) (*domain.StaffTask, error)
	createNotificationFn   func(context.Context, domain.CreateNotificationInput) (*domain.Notification, error)
	listNotificationsFn    func(context.Context, int64, int) ([]*domain.Notification, error)
	markNotificationReadFn func(context.Context, int64, int64) error
	dashboardFn            func(context.Context) (*domain.DashboardSummary, error)
	listAuditFn            func(context.Context, int) ([]*domain.AuditRecord, error)
	recordAuditFn          func(context.Context, *int64, string, string, string, string, int) error
	freezeFn               func(context.Context, int64) (*domain.ClientSubscription, error)
	unfreezeFn             func(context.Context, int64) (*domain.ClientSubscription, error)
	extendFn               func(context.Context, int64, int) (*domain.ClientSubscription, error)
	hasFutureFn            func(context.Context, int64) (bool, error)
}

func (f *fakeCRMRepo) AddProgress(c context.Context, a, b int64, i domain.CreateProgressInput) (*domain.ClientProgress, error) {
	if f.addProgressFn != nil {
		return f.addProgressFn(c, a, b, i)
	}
	return nil, domain.ErrNotFound
}
func (f *fakeCRMRepo) ListProgress(c context.Context, id int64, l int) ([]*domain.ClientProgress, error) {
	if f.listProgressFn != nil {
		return f.listProgressFn(c, id, l)
	}
	return nil, nil
}
func (f *fakeCRMRepo) DeleteProgress(c context.Context, id int64) error {
	if f.deleteProgressFn != nil {
		return f.deleteProgressFn(c, id)
	}
	return nil
}
func (f *fakeCRMRepo) AddNote(c context.Context, a, b int64, n string) (*domain.ClientNote, error) {
	if f.addNoteFn != nil {
		return f.addNoteFn(c, a, b, n)
	}
	return nil, domain.ErrNotFound
}
func (f *fakeCRMRepo) ListNotes(c context.Context, id int64, l int) ([]*domain.ClientNote, error) {
	if f.listNotesFn != nil {
		return f.listNotesFn(c, id, l)
	}
	return nil, nil
}
func (f *fakeCRMRepo) DeleteNote(c context.Context, id int64) error {
	if f.deleteNoteFn != nil {
		return f.deleteNoteFn(c, id)
	}
	return nil
}
func (f *fakeCRMRepo) CreateTask(c context.Context, a int64, i domain.CreateTaskInput) (*domain.StaffTask, error) {
	if f.createTaskFn != nil {
		return f.createTaskFn(c, a, i)
	}
	return nil, domain.ErrNotFound
}
func (f *fakeCRMRepo) ListTasks(c context.Context, a, b *int64, d bool, l int) ([]*domain.StaffTask, error) {
	if f.listTasksFn != nil {
		return f.listTasksFn(c, a, b, d, l)
	}
	return nil, nil
}
func (f *fakeCRMRepo) UpdateTaskStatus(c context.Context, id int64, s string, assigneeID *int64) (*domain.StaffTask, error) {
	if f.updateTaskStatusFn != nil {
		return f.updateTaskStatusFn(c, id, s, assigneeID)
	}
	return nil, domain.ErrNotFound
}
func (f *fakeCRMRepo) CreateNotification(c context.Context, i domain.CreateNotificationInput) (*domain.Notification, error) {
	if f.createNotificationFn != nil {
		return f.createNotificationFn(c, i)
	}
	return nil, domain.ErrNotFound
}
func (f *fakeCRMRepo) ListNotifications(c context.Context, id int64, l int) ([]*domain.Notification, error) {
	if f.listNotificationsFn != nil {
		return f.listNotificationsFn(c, id, l)
	}
	return nil, nil
}
func (f *fakeCRMRepo) MarkNotificationRead(c context.Context, id, u int64) error {
	if f.markNotificationReadFn != nil {
		return f.markNotificationReadFn(c, id, u)
	}
	return nil
}
func (f *fakeCRMRepo) Dashboard(c context.Context) (*domain.DashboardSummary, error) {
	if f.dashboardFn != nil {
		return f.dashboardFn(c)
	}
	return &domain.DashboardSummary{}, nil
}
func (f *fakeCRMRepo) ListAudit(c context.Context, l int) ([]*domain.AuditRecord, error) {
	if f.listAuditFn != nil {
		return f.listAuditFn(c, l)
	}
	return nil, nil
}
func (f *fakeCRMRepo) RecordAudit(c context.Context, a *int64, r, req, m, p string, s int) error {
	if f.recordAuditFn != nil {
		return f.recordAuditFn(c, a, r, req, m, p, s)
	}
	return nil
}
func (f *fakeCRMRepo) FreezeSubscription(c context.Context, id int64) (*domain.ClientSubscription, error) {
	if f.freezeFn != nil {
		return f.freezeFn(c, id)
	}
	return nil, domain.ErrNotFound
}
func (f *fakeCRMRepo) UnfreezeSubscription(c context.Context, id int64) (*domain.ClientSubscription, error) {
	if f.unfreezeFn != nil {
		return f.unfreezeFn(c, id)
	}
	return nil, domain.ErrNotFound
}
func (f *fakeCRMRepo) ExtendSubscription(c context.Context, id int64, d int) (*domain.ClientSubscription, error) {
	if f.extendFn != nil {
		return f.extendFn(c, id, d)
	}
	return nil, domain.ErrNotFound
}
func (f *fakeCRMRepo) HasFutureTrainingForSubscription(c context.Context, id int64) (bool, error) {
	if f.hasFutureFn != nil {
		return f.hasFutureFn(c, id)
	}
	return false, nil
}

var _ repository.UserRepository = (*fakeUserRepo)(nil)
var _ repository.ApplicationRepository = (*fakeApplicationRepo)(nil)
var _ repository.TrainerRepository = (*fakeTrainerRepo)(nil)
var _ repository.TrainingRepository = (*fakeTrainingRepo)(nil)
var _ repository.TrainingRequestRepository = (*fakeTrainingRequestRepo)(nil)
var _ repository.ProductRepository = (*fakeProductRepo)(nil)
var _ repository.SubscriptionRepository = (*fakeSubscriptionRepo)(nil)
var _ repository.PaymentRepository = (*fakePaymentRepo)(nil)
var _ repository.OrderRepository = (*fakeOrderRepo)(nil)
var _ repository.CRMRepository = (*fakeCRMRepo)(nil)

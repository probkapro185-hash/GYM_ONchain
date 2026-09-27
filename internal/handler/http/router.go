package httphandler

import (
	"context"
	"net/http"
	"time"

	"github.com/sfedu-crm/internal/domain"
	"github.com/sfedu-crm/internal/middleware"
	"github.com/sfedu-crm/internal/repository"
	"github.com/sfedu-crm/pkg/jwt"
)

type Handlers struct {
	Auth     *AuthHandler
	Account  *AccountHandler
	User     *UserHandler
	Schedule *ScheduleHandler
	Finance  *FinanceHandler
	Shop     *ShopHandler
	Trainer  *TrainerHandler
	AI       *AIHandler
	CRM      *CRMHandler
}

func NewRouter(h *Handlers, tokenMgr *jwt.Manager, users repository.UserRepository, crmRepo repository.CRMRepository, ready func(context.Context) error) http.Handler {
	mux := http.NewServeMux()

	// Full SFEDU web interface. API routes below are more specific and therefore
	// continue to win in Go 1.22+ ServeMux matching.
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		http.ServeFile(w, r, "frontend/index.html")
	})

	mux.Handle("GET /assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir("frontend/assets"))))
	mux.HandleFunc("GET /uploads/trainers/{filename}", h.Trainer.ServeTrainerPhoto)
	mux.HandleFunc("GET /uploads/products/{filename}", h.Shop.ServeProductPhoto)
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		respond(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, r *http.Request) {
		if ready != nil {
			ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
			defer cancel()
			if err := ready(ctx); err != nil {
				respondError(w, http.StatusServiceUnavailable, "not ready")
				return
			}
		}
		respond(w, http.StatusOK, map[string]string{"status": "ready"})
	})
	mux.HandleFunc("GET /metrics", middleware.MetricsHandler)
	mux.Handle("POST /api/v1/auth/apply", middleware.IPRateLimit(5, time.Hour)(http.HandlerFunc(h.Auth.SubmitApplication)))
	mux.Handle("POST /api/v1/auth/login", middleware.IPRateLimit(10, time.Minute)(http.HandlerFunc(h.Auth.Login)))
	mux.Handle("POST /api/v1/auth/forgot-password", middleware.IPRateLimit(5, time.Hour)(http.HandlerFunc(h.Account.ForgotPassword)))
	mux.Handle("POST /api/v1/auth/activate", middleware.IPRateLimit(10, 15*time.Minute)(http.HandlerFunc(h.Account.Activate)))
	mux.Handle("POST /api/v1/auth/reset-password", middleware.IPRateLimit(10, 15*time.Minute)(http.HandlerFunc(h.Account.ResetPassword)))
	mux.HandleFunc("GET /api/v1/trainers", h.Trainer.ListTrainers)
	mux.HandleFunc("GET /api/v1/trainers/{id}", h.Trainer.GetTrainer)

	authMw := middleware.Auth(tokenMgr, users)
	adminOnly := chain(authMw, middleware.RequireRole(domain.RoleAdmin))
	managerAndAdmin := chain(authMw, middleware.RequireRole(domain.RoleManager, domain.RoleAdmin))
	clientOnly := chain(authMw, middleware.RequireRole(domain.RoleClient))
	allRoles := chain(authMw, middleware.RequireRole(domain.RoleClient, domain.RoleManager, domain.RoleAdmin))
	aiChat := chain(authMw, middleware.RequireRole(domain.RoleClient, domain.RoleManager, domain.RoleAdmin), middleware.AIRateLimit(20, time.Minute))
	auditChanges := middleware.AuditChanges(crmRepo)
	adminOnlyAudit := chain(authMw, middleware.RequireRole(domain.RoleAdmin), auditChanges)
	managerAndAdminAudit := chain(authMw, middleware.RequireRole(domain.RoleManager, domain.RoleAdmin), auditChanges)
	clientOnlyAudit := chain(authMw, middleware.RequireRole(domain.RoleClient), auditChanges)
	allRolesAudit := chain(authMw, middleware.RequireRole(domain.RoleClient, domain.RoleManager, domain.RoleAdmin), auditChanges)

	mux.Handle("GET /api/v1/users/me", allRoles(http.HandlerFunc(h.User.GetMe)))
	mux.Handle("PUT /api/v1/users/me", allRolesAudit(http.HandlerFunc(h.User.UpdateMe)))
	mux.Handle("PUT /api/v1/users/me/password", allRolesAudit(http.HandlerFunc(h.User.ChangeMyPassword)))

	mux.Handle("GET /api/v1/users", managerAndAdmin(http.HandlerFunc(h.User.ListUsers)))
	mux.Handle("POST /api/v1/users", adminOnlyAudit(http.HandlerFunc(h.User.CreateUser)))
	mux.Handle("POST /api/v1/users/invite", managerAndAdminAudit(http.HandlerFunc(h.Account.InviteClient)))
	mux.Handle("GET /api/v1/users/{id}", managerAndAdmin(http.HandlerFunc(h.User.GetUser)))
	mux.Handle("PUT /api/v1/users/{id}", managerAndAdminAudit(http.HandlerFunc(h.User.UpdateUser)))
	mux.Handle("PUT /api/v1/users/{id}/password", adminOnlyAudit(http.HandlerFunc(h.User.AdminResetPassword)))
	mux.Handle("DELETE /api/v1/users/{id}", adminOnlyAudit(http.HandlerFunc(h.User.DeleteUser)))
	mux.Handle("PATCH /api/v1/users/{id}/activate", adminOnlyAudit(h.User.SetActive(true)))
	mux.Handle("PATCH /api/v1/users/{id}/deactivate", adminOnlyAudit(h.User.SetActive(false)))

	mux.Handle("GET /api/v1/applications", managerAndAdmin(http.HandlerFunc(h.User.ListApplications)))
	mux.Handle("POST /api/v1/applications/{id}/approve", managerAndAdminAudit(http.HandlerFunc(h.Account.ApproveApplication)))
	mux.Handle("POST /api/v1/applications/{id}/reject", managerAndAdminAudit(http.HandlerFunc(h.User.RejectApplication)))
	mux.Handle("POST /api/v1/users/{id}/resend-activation", managerAndAdminAudit(http.HandlerFunc(h.Account.ResendActivation)))

	mux.Handle("GET /api/v1/schedule", allRoles(http.HandlerFunc(h.Schedule.GetSchedule)))
	mux.Handle("GET /api/v1/schedule/{id}", allRoles(http.HandlerFunc(h.Schedule.GetTraining)))
	mux.Handle("POST /api/v1/schedule/requests", clientOnlyAudit(http.HandlerFunc(h.Schedule.SubmitTrainingRequest)))
	mux.Handle("GET /api/v1/schedule/requests/my", clientOnly(http.HandlerFunc(h.Schedule.ListMyRequests)))
	mux.Handle("GET /api/v1/schedule/requests", managerAndAdmin(http.HandlerFunc(h.Schedule.ListPendingRequests)))
	mux.Handle("POST /api/v1/schedule/requests/{id}/approve", managerAndAdminAudit(http.HandlerFunc(h.Schedule.ApproveRequest)))
	mux.Handle("POST /api/v1/schedule/requests/{id}/reject", managerAndAdminAudit(http.HandlerFunc(h.Schedule.RejectRequest)))
	mux.Handle("POST /api/v1/schedule", managerAndAdminAudit(http.HandlerFunc(h.Schedule.CreateTraining)))
	mux.Handle("PUT /api/v1/schedule/{id}", managerAndAdminAudit(http.HandlerFunc(h.Schedule.UpdateTraining)))
	mux.Handle("DELETE /api/v1/schedule/{id}", adminOnlyAudit(http.HandlerFunc(h.Schedule.DeleteTraining)))
	mux.Handle("POST /api/v1/schedule/{id}/cancel", clientOnlyAudit(http.HandlerFunc(h.Schedule.CancelMyTraining)))

	// Self-crediting balance endpoint was intentionally removed. Balance changes are trusted staff actions
	// until a real payment provider with signed callbacks is integrated.
	mux.Handle("POST /api/v1/finance/topup", managerAndAdminAudit(http.HandlerFunc(h.Finance.TopUpBalance)))
	mux.Handle("GET /api/v1/finance/me/payments", clientOnly(http.HandlerFunc(h.Finance.GetMyPayments)))
	mux.Handle("GET /api/v1/finance/payments", adminOnly(http.HandlerFunc(h.Finance.ListPayments)))
	mux.Handle("GET /api/v1/finance/summary", adminOnly(http.HandlerFunc(h.Finance.GetSummary)))

	mux.Handle("GET /api/v1/shop/products", allRoles(http.HandlerFunc(h.Shop.ListProducts)))
	mux.Handle("GET /api/v1/shop/products/{id}", allRoles(http.HandlerFunc(h.Shop.GetProduct)))
	mux.Handle("POST /api/v1/shop/purchase", clientOnlyAudit(http.HandlerFunc(h.Shop.PurchaseProduct)))
	mux.Handle("GET /api/v1/shop/my-subscriptions", clientOnly(http.HandlerFunc(h.Shop.GetMySubscriptions)))
	mux.Handle("GET /api/v1/shop/my-orders", clientOnly(http.HandlerFunc(h.Shop.GetMyOrders)))
	mux.Handle("POST /api/v1/shop/products", managerAndAdminAudit(http.HandlerFunc(h.Shop.CreateProduct)))
	mux.Handle("PUT /api/v1/shop/products/{id}", managerAndAdminAudit(http.HandlerFunc(h.Shop.UpdateProduct)))
	mux.Handle("POST /api/v1/shop/products/{id}/photo", managerAndAdminAudit(http.HandlerFunc(h.Shop.UploadProductPhoto)))
	mux.Handle("DELETE /api/v1/shop/products/{id}/photo", managerAndAdminAudit(http.HandlerFunc(h.Shop.DeleteProductPhoto)))
	mux.Handle("DELETE /api/v1/shop/products/{id}", adminOnlyAudit(http.HandlerFunc(h.Shop.DeleteProduct)))

	mux.Handle("GET /api/v1/admin/trainers", adminOnly(http.HandlerFunc(h.Trainer.ListAllTrainers)))
	mux.Handle("POST /api/v1/trainers", adminOnlyAudit(http.HandlerFunc(h.Trainer.CreateTrainer)))
	mux.Handle("POST /api/v1/trainers/{id}/photo", adminOnlyAudit(http.HandlerFunc(h.Trainer.UploadTrainerPhoto)))
	mux.Handle("DELETE /api/v1/trainers/{id}/photo", adminOnlyAudit(http.HandlerFunc(h.Trainer.DeleteTrainerPhoto)))
	mux.Handle("PUT /api/v1/trainers/{id}", adminOnlyAudit(http.HandlerFunc(h.Trainer.UpdateTrainer)))
	mux.Handle("DELETE /api/v1/trainers/{id}", adminOnlyAudit(http.HandlerFunc(h.Trainer.DeleteTrainer)))

	// CRM+ modules. AI/RAG routes below are intentionally unchanged.
	mux.Handle("GET /api/v1/crm/progress", clientOnly(http.HandlerFunc(h.CRM.MyProgress)))
	mux.Handle("POST /api/v1/crm/progress", clientOnlyAudit(http.HandlerFunc(h.CRM.AddMyProgress)))
	mux.Handle("GET /api/v1/crm/clients/{id}/progress", managerAndAdmin(http.HandlerFunc(h.CRM.ClientProgress)))
	mux.Handle("POST /api/v1/crm/clients/{id}/progress", managerAndAdminAudit(http.HandlerFunc(h.CRM.AddClientProgress)))
	mux.Handle("DELETE /api/v1/crm/progress/{id}", managerAndAdminAudit(http.HandlerFunc(h.CRM.DeleteProgress)))
	mux.Handle("GET /api/v1/crm/clients/{id}/notes", managerAndAdmin(http.HandlerFunc(h.CRM.ClientNotes)))
	mux.Handle("POST /api/v1/crm/clients/{id}/notes", managerAndAdminAudit(http.HandlerFunc(h.CRM.AddClientNote)))
	mux.Handle("DELETE /api/v1/crm/notes/{id}", managerAndAdminAudit(http.HandlerFunc(h.CRM.DeleteNote)))
	mux.Handle("GET /api/v1/crm/tasks", managerAndAdmin(http.HandlerFunc(h.CRM.ListTasks)))
	mux.Handle("POST /api/v1/crm/tasks", managerAndAdminAudit(http.HandlerFunc(h.CRM.CreateTask)))
	mux.Handle("PATCH /api/v1/crm/tasks/{id}", managerAndAdminAudit(http.HandlerFunc(h.CRM.UpdateTaskStatus)))
	mux.Handle("GET /api/v1/crm/notifications", allRoles(http.HandlerFunc(h.CRM.MyNotifications)))
	mux.Handle("PATCH /api/v1/crm/notifications/{id}/read", allRolesAudit(http.HandlerFunc(h.CRM.MarkNotificationRead)))
	mux.Handle("POST /api/v1/crm/notifications", managerAndAdminAudit(http.HandlerFunc(h.CRM.CreateNotification)))
	mux.Handle("GET /api/v1/crm/clients/{id}", managerAndAdmin(http.HandlerFunc(h.CRM.ClientCard)))
	mux.Handle("POST /api/v1/crm/subscriptions/{id}/freeze", managerAndAdminAudit(http.HandlerFunc(h.CRM.FreezeSubscription)))
	mux.Handle("POST /api/v1/crm/subscriptions/{id}/unfreeze", managerAndAdminAudit(http.HandlerFunc(h.CRM.UnfreezeSubscription)))
	mux.Handle("POST /api/v1/crm/subscriptions/{id}/extend", managerAndAdminAudit(http.HandlerFunc(h.CRM.ExtendSubscription)))
	mux.Handle("POST /api/v1/shop/staff-purchase", managerAndAdminAudit(http.HandlerFunc(h.CRM.StaffPurchase)))
	mux.Handle("GET /api/v1/crm/dashboard", adminOnly(http.HandlerFunc(h.CRM.Dashboard)))
	mux.Handle("GET /api/v1/crm/audit", adminOnly(http.HandlerFunc(h.CRM.Audit)))
	mux.Handle("GET /api/v1/exports/clients.csv", adminOnly(http.HandlerFunc(h.CRM.ExportClients)))
	mux.Handle("GET /api/v1/exports/finance.csv", adminOnly(http.HandlerFunc(h.CRM.ExportFinance)))
	mux.Handle("GET /api/v1/crm/clients/{id}/export.csv", managerAndAdmin(http.HandlerFunc(h.CRM.ExportClientCard)))

	mux.Handle("POST /api/v1/ai/chat", aiChat(http.HandlerFunc(h.AI.Chat)))
	mux.Handle("GET /api/v1/ai/conversations", allRoles(http.HandlerFunc(h.AI.ListConversations)))
	mux.Handle("GET /api/v1/ai/conversations/{id}", allRoles(http.HandlerFunc(h.AI.GetConversationMessages)))
	mux.Handle("DELETE /api/v1/ai/conversations/{id}", allRoles(http.HandlerFunc(h.AI.DeleteConversation)))
	mux.Handle("GET /api/v1/ai/knowledge", adminOnly(http.HandlerFunc(h.AI.ListKnowledge)))
	mux.Handle("POST /api/v1/ai/reindex", adminOnly(http.HandlerFunc(h.AI.Reindex)))
	return mux
}

func chain(middlewares ...func(http.Handler) http.Handler) func(http.Handler) http.Handler {
	return func(final http.Handler) http.Handler {
		for i := len(middlewares) - 1; i >= 0; i-- {
			final = middlewares[i](final)
		}
		return final
	}
}

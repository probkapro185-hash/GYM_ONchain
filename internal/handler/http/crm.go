package httphandler

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/sfedu-crm/internal/domain"
	"github.com/sfedu-crm/internal/repository"
	"github.com/sfedu-crm/internal/service"
)

type CRMHandler struct {
	crm     *service.CRMService
	users   *service.UserService
	finance *service.FinanceService
	shop    *service.ShopService
}

func NewCRMHandler(crm *service.CRMService, users *service.UserService, finance *service.FinanceService, shop *service.ShopService) *CRMHandler {
	return &CRMHandler{crm: crm, users: users, finance: finance, shop: shop}
}

func (h *CRMHandler) MyProgress(w http.ResponseWriter, r *http.Request) {
	id, role, ok := actor(r)
	if !ok {
		respondError(w, 401, "unauthorized")
		return
	}
	items, err := h.crm.ListProgress(r.Context(), id, role, id)
	if err != nil {
		handleError(w, err)
		return
	}
	respond(w, 200, items)
}
func (h *CRMHandler) AddMyProgress(w http.ResponseWriter, r *http.Request) {
	id, _, ok := actor(r)
	if !ok {
		respondError(w, 401, "unauthorized")
		return
	}
	var in domain.CreateProgressInput
	if err := decode(w, r, &in); err != nil {
		respondError(w, 400, "invalid request body")
		return
	}
	item, err := h.crm.AddProgress(r.Context(), id, id, in)
	if err != nil {
		handleError(w, err)
		return
	}
	respond(w, 201, item)
}
func (h *CRMHandler) ClientProgress(w http.ResponseWriter, r *http.Request) {
	clientID, err := parseIDFromPath(r)
	if err != nil {
		respondError(w, 400, "invalid client id")
		return
	}
	actorID, role, ok := actor(r)
	if !ok {
		respondError(w, 401, "unauthorized")
		return
	}
	items, err := h.crm.ListProgress(r.Context(), actorID, role, clientID)
	if err != nil {
		handleError(w, err)
		return
	}
	respond(w, 200, items)
}
func (h *CRMHandler) AddClientProgress(w http.ResponseWriter, r *http.Request) {
	clientID, err := parseIDFromPath(r)
	if err != nil {
		respondError(w, 400, "invalid client id")
		return
	}
	actorID, _, ok := actor(r)
	if !ok {
		respondError(w, 401, "unauthorized")
		return
	}
	var in domain.CreateProgressInput
	if err := decode(w, r, &in); err != nil {
		respondError(w, 400, "invalid request body")
		return
	}
	item, err := h.crm.AddProgress(r.Context(), clientID, actorID, in)
	if err != nil {
		handleError(w, err)
		return
	}
	respond(w, 201, item)
}
func (h *CRMHandler) DeleteProgress(w http.ResponseWriter, r *http.Request) {
	id, err := parseIDFromPath(r)
	if err != nil {
		respondError(w, 400, "invalid progress id")
		return
	}
	if err := h.crm.DeleteProgress(r.Context(), id); err != nil {
		handleError(w, err)
		return
	}
	respond(w, 204, nil)
}

func (h *CRMHandler) ClientNotes(w http.ResponseWriter, r *http.Request) {
	clientID, err := parseIDFromPath(r)
	if err != nil {
		respondError(w, 400, "invalid client id")
		return
	}
	items, err := h.crm.ListNotes(r.Context(), clientID)
	if err != nil {
		handleError(w, err)
		return
	}
	respond(w, 200, items)
}
func (h *CRMHandler) AddClientNote(w http.ResponseWriter, r *http.Request) {
	clientID, err := parseIDFromPath(r)
	if err != nil {
		respondError(w, 400, "invalid client id")
		return
	}
	actorID, _, ok := actor(r)
	if !ok {
		respondError(w, 401, "unauthorized")
		return
	}
	var in domain.CreateNoteInput
	if err := decode(w, r, &in); err != nil {
		respondError(w, 400, "invalid request body")
		return
	}
	item, err := h.crm.AddNote(r.Context(), clientID, actorID, in.Note)
	if err != nil {
		handleError(w, err)
		return
	}
	respond(w, 201, item)
}
func (h *CRMHandler) DeleteNote(w http.ResponseWriter, r *http.Request) {
	id, err := parseIDFromPath(r)
	if err != nil {
		respondError(w, 400, "invalid note id")
		return
	}
	if err := h.crm.DeleteNote(r.Context(), id); err != nil {
		handleError(w, err)
		return
	}
	respond(w, 204, nil)
}

func (h *CRMHandler) ListTasks(w http.ResponseWriter, r *http.Request) {
	actorID, role, ok := actor(r)
	if !ok {
		respondError(w, 401, "unauthorized")
		return
	}
	var clientID *int64
	if v := strings.TrimSpace(r.URL.Query().Get("client_id")); v != "" {
		id, err := parsePositiveInt64(v)
		if err != nil {
			respondError(w, 400, "invalid client_id")
			return
		}
		clientID = &id
	}
	includeDone, _ := strconv.ParseBool(r.URL.Query().Get("include_done"))
	items, err := h.crm.ListTasks(r.Context(), actorID, role, clientID, includeDone)
	if err != nil {
		handleError(w, err)
		return
	}
	respond(w, 200, items)
}
func (h *CRMHandler) CreateTask(w http.ResponseWriter, r *http.Request) {
	actorID, _, ok := actor(r)
	if !ok {
		respondError(w, 401, "unauthorized")
		return
	}
	var in domain.CreateTaskInput
	if err := decode(w, r, &in); err != nil {
		respondError(w, 400, "invalid request body")
		return
	}
	item, err := h.crm.CreateTask(r.Context(), actorID, in)
	if err != nil {
		handleError(w, err)
		return
	}
	respond(w, 201, item)
}
func (h *CRMHandler) UpdateTaskStatus(w http.ResponseWriter, r *http.Request) {
	id, err := parseIDFromPath(r)
	if err != nil {
		respondError(w, 400, "invalid task id")
		return
	}
	var in domain.UpdateTaskStatusInput
	if err := decode(w, r, &in); err != nil {
		respondError(w, 400, "invalid request body")
		return
	}
	actorID, role, ok := actor(r)
	if !ok {
		respondError(w, 401, "unauthorized")
		return
	}
	item, err := h.crm.UpdateTaskStatus(r.Context(), actorID, role, id, in.Status)
	if err != nil {
		handleError(w, err)
		return
	}
	respond(w, 200, item)
}

func (h *CRMHandler) MyNotifications(w http.ResponseWriter, r *http.Request) {
	id, _, ok := actor(r)
	if !ok {
		respondError(w, 401, "unauthorized")
		return
	}
	items, err := h.crm.ListNotifications(r.Context(), id)
	if err != nil {
		handleError(w, err)
		return
	}
	respond(w, 200, items)
}
func (h *CRMHandler) MarkNotificationRead(w http.ResponseWriter, r *http.Request) {
	id, err := parseIDFromPath(r)
	if err != nil {
		respondError(w, 400, "invalid notification id")
		return
	}
	userID, _, ok := actor(r)
	if !ok {
		respondError(w, 401, "unauthorized")
		return
	}
	if err := h.crm.MarkNotificationRead(r.Context(), id, userID); err != nil {
		handleError(w, err)
		return
	}
	respond(w, 200, map[string]string{"status": "ok"})
}
func (h *CRMHandler) CreateNotification(w http.ResponseWriter, r *http.Request) {
	var in domain.CreateNotificationInput
	if err := decode(w, r, &in); err != nil {
		respondError(w, 400, "invalid request body")
		return
	}
	item, err := h.crm.CreateNotification(r.Context(), in)
	if err != nil {
		handleError(w, err)
		return
	}
	respond(w, 201, item)
}

func (h *CRMHandler) Dashboard(w http.ResponseWriter, r *http.Request) {
	d, err := h.crm.Dashboard(r.Context())
	if err != nil {
		handleError(w, err)
		return
	}
	respond(w, 200, d)
}
func (h *CRMHandler) Audit(w http.ResponseWriter, r *http.Request) {
	items, err := h.crm.ListAudit(r.Context())
	if err != nil {
		handleError(w, err)
		return
	}
	respond(w, 200, items)
}
func (h *CRMHandler) ClientCard(w http.ResponseWriter, r *http.Request) {
	id, err := parseIDFromPath(r)
	if err != nil {
		respondError(w, 400, "invalid client id")
		return
	}
	card, err := h.crm.ClientCard(r.Context(), id)
	if err != nil {
		handleError(w, err)
		return
	}
	respond(w, 200, card)
}

func (h *CRMHandler) FreezeSubscription(w http.ResponseWriter, r *http.Request) {
	id, err := parseIDFromPath(r)
	if err != nil {
		respondError(w, 400, "invalid subscription id")
		return
	}
	s, err := h.crm.FreezeSubscription(r.Context(), id)
	if err != nil {
		handleError(w, err)
		return
	}
	respond(w, 200, s)
}
func (h *CRMHandler) UnfreezeSubscription(w http.ResponseWriter, r *http.Request) {
	id, err := parseIDFromPath(r)
	if err != nil {
		respondError(w, 400, "invalid subscription id")
		return
	}
	s, err := h.crm.UnfreezeSubscription(r.Context(), id)
	if err != nil {
		handleError(w, err)
		return
	}
	respond(w, 200, s)
}
func (h *CRMHandler) ExtendSubscription(w http.ResponseWriter, r *http.Request) {
	id, err := parseIDFromPath(r)
	if err != nil {
		respondError(w, 400, "invalid subscription id")
		return
	}
	var in domain.ExtendSubscriptionInput
	if err := decode(w, r, &in); err != nil {
		respondError(w, 400, "invalid request body")
		return
	}
	s, err := h.crm.ExtendSubscription(r.Context(), id, in.Days)
	if err != nil {
		handleError(w, err)
		return
	}
	respond(w, 200, s)
}

func (h *CRMHandler) StaffPurchase(w http.ResponseWriter, r *http.Request) {
	var in struct {
		ClientID  int64 `json:"client_id"`
		ProductID int64 `json:"product_id"`
	}
	if err := decode(w, r, &in); err != nil || in.ClientID <= 0 || in.ProductID <= 0 {
		respondError(w, 400, "invalid request body")
		return
	}
	result, err := h.shop.PurchaseProduct(r.Context(), in.ClientID, in.ProductID)
	if err != nil {
		handleError(w, err)
		return
	}
	respond(w, 201, result)
}

func (h *CRMHandler) ExportClients(w http.ResponseWriter, r *http.Request) {
	_, role, ok := actor(r)
	if !ok {
		respondError(w, 401, "unauthorized")
		return
	}
	clientRole := domain.RoleClient
	items, err := h.users.List(r.Context(), role, repository.UserFilter{Role: &clientRole})
	if err != nil {
		handleError(w, err)
		return
	}
	startCSV(w, "clients.csv")
	cw := csv.NewWriter(w)
	_ = cw.Write([]string{"ID", "ФИО", "Телефон", "Email", "Баланс", "Посещения", "Последний визит", "Активен"})
	for _, u := range items {
		last := ""
		if u.LastVisitAt != nil {
			last = u.LastVisitAt.Format(time.RFC3339)
		}
		_ = cw.Write([]string{strconv.FormatInt(u.ID, 10), u.FullName, u.Phone, u.Email, fmt.Sprintf("%.2f", u.Balance), strconv.Itoa(u.Visits), last, strconv.FormatBool(u.IsActive)})
	}
	cw.Flush()
}
func (h *CRMHandler) ExportFinance(w http.ResponseWriter, r *http.Request) {
	items, err := h.finance.ListPayments(r.Context(), domain.PaymentFilter{})
	if err != nil {
		handleError(w, err)
		return
	}
	startCSV(w, "finance.csv")
	cw := csv.NewWriter(w)
	_ = cw.Write([]string{"ID", "Клиент", "Сумма", "Операция", "Тип", "Описание", "Дата"})
	for _, p := range items {
		_ = cw.Write([]string{strconv.FormatInt(p.ID, 10), p.ClientName, fmt.Sprintf("%.2f", p.Amount), string(p.OperationType), string(p.ServiceType), p.Description, p.CreatedAt.Format(time.RFC3339)})
	}
	cw.Flush()
}
func (h *CRMHandler) ExportClientCard(w http.ResponseWriter, r *http.Request) {
	id, err := parseIDFromPath(r)
	if err != nil {
		respondError(w, 400, "invalid client id")
		return
	}
	card, err := h.crm.ClientCard(r.Context(), id)
	if err != nil {
		handleError(w, err)
		return
	}
	startCSV(w, fmt.Sprintf("client-%d-crm.csv", id))
	cw := csv.NewWriter(w)
	_ = cw.Write([]string{"Раздел", "Поле", "Значение"})
	_ = cw.Write([]string{"Клиент", "ФИО", card.User.FullName})
	_ = cw.Write([]string{"Клиент", "Email", card.User.Email})
	_ = cw.Write([]string{"Клиент", "Телефон", card.User.Phone})
	for _, s := range card.Subscriptions {
		_ = cw.Write([]string{"Абонемент", s.ProductName, fmt.Sprintf("до %s; занятий %v; frozen=%v", s.EndDate.Format("2006-01-02"), csvSessions(s.SessionsLeft), s.FrozenAt != nil)})
	}
	for _, p := range card.Progress {
		_ = cw.Write([]string{"Прогресс", p.RecordedAt.Format("2006-01-02"), fmt.Sprintf("вес=%v; жир=%v; талия=%v; комментарий=%s", csvMeasurement(p.Weight), csvMeasurement(p.BodyFat), csvMeasurement(p.Waist), p.Comment)})
	}
	for _, n := range card.Notes {
		_ = cw.Write([]string{"Заметка", n.AuthorName, n.Note})
	}
	for _, t := range card.Tasks {
		_ = cw.Write([]string{"Задача", t.Status, t.Title})
	}
	cw.Flush()
}
func startCSV(w http.ResponseWriter, name string) {
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="`+name+`"`)
	_, _ = w.Write([]byte{0xEF, 0xBB, 0xBF})
}

func csvSessions(value *int) string {
	if value == nil {
		return "Безлимит"
	}
	return strconv.Itoa(*value)
}
func csvMeasurement(value *float64) string {
	if value == nil {
		return "—"
	}
	return strconv.FormatFloat(*value, 'f', -1, 64)
}

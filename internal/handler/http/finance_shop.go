package httphandler

import (
	"fmt"
	"net/http"

	"github.com/sfedu-crm/internal/domain"
	"github.com/sfedu-crm/internal/repository"
	"github.com/sfedu-crm/internal/service"
)

type FinanceHandler struct{ financeSvc *service.FinanceService }

func NewFinanceHandler(financeSvc *service.FinanceService) *FinanceHandler {
	return &FinanceHandler{financeSvc: financeSvc}
}

func (h *FinanceHandler) TopUpBalance(w http.ResponseWriter, r *http.Request) {
	var input domain.TopUpBalanceInput
	if err := decode(w, r, &input); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if input.ClientID <= 0 {
		respondError(w, http.StatusBadRequest, "client_id is required")
		return
	}
	payment, err := h.financeSvc.TopUpBalance(r.Context(), input.ClientID, input.Amount)
	if err != nil {
		handleError(w, err)
		return
	}
	respond(w, http.StatusCreated, payment)
}

func (h *FinanceHandler) GetMyPayments(w http.ResponseWriter, r *http.Request) {
	userID, _, ok := actor(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	payments, err := h.financeSvc.GetMyPayments(r.Context(), userID)
	if err != nil {
		handleError(w, err)
		return
	}
	respond(w, http.StatusOK, payments)
}

func (h *FinanceHandler) ListPayments(w http.ResponseWriter, r *http.Request) {
	filter, err := paymentFilterFromRequest(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	payments, err := h.financeSvc.ListPayments(r.Context(), filter)
	if err != nil {
		handleError(w, err)
		return
	}
	respond(w, http.StatusOK, payments)
}

func (h *FinanceHandler) GetSummary(w http.ResponseWriter, r *http.Request) {
	filter, err := paymentFilterFromRequest(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	summary, err := h.financeSvc.GetSummary(r.Context(), filter)
	if err != nil {
		handleError(w, err)
		return
	}
	respond(w, http.StatusOK, summary)
}

func paymentFilterFromRequest(r *http.Request) (domain.PaymentFilter, error) {
	q := r.URL.Query()
	filter := domain.PaymentFilter{}
	if value := q.Get("client_id"); value != "" {
		id, err := parsePositiveInt64(value)
		if err != nil {
			return filter, fmt.Errorf("invalid client_id")
		}
		filter.ClientID = &id
	}
	if value := q.Get("operation_type"); value != "" {
		op := domain.OperationType(value)
		switch op {
		case domain.OperationIncome, domain.OperationExpense, domain.OperationRefund:
			filter.OperationType = op
		default:
			return filter, fmt.Errorf("invalid operation_type")
		}
	}
	if value := q.Get("service_type"); value != "" {
		svc := domain.ServiceType(value)
		switch svc {
		case domain.ServiceSubscription, domain.ServiceTraining, domain.ServiceProduct, domain.ServiceDeposit:
			filter.ServiceType = svc
		default:
			return filter, fmt.Errorf("invalid service_type")
		}
	}
	if value := q.Get("date_from"); value != "" {
		t, err := parseDate(value)
		if err != nil {
			return filter, fmt.Errorf("invalid date_from format (YYYY-MM-DD)")
		}
		filter.DateFrom = &t
	}
	if value := q.Get("date_to"); value != "" {
		t, err := parseDate(value)
		if err != nil {
			return filter, fmt.Errorf("invalid date_to format (YYYY-MM-DD)")
		}
		t = t.AddDate(0, 0, 1)
		filter.DateTo = &t
	}
	if filter.DateFrom != nil && filter.DateTo != nil && !filter.DateTo.After(*filter.DateFrom) {
		return filter, fmt.Errorf("date_to must be on or after date_from")
	}
	return filter, nil
}

type ShopHandler struct {
	shopSvc  *service.ShopService
	photoDir string
}

func NewShopHandler(shopSvc *service.ShopService) *ShopHandler {
	return &ShopHandler{shopSvc: shopSvc, photoDir: "data/uploads/products"}
}

func (h *ShopHandler) ListProducts(w http.ResponseWriter, r *http.Request) {
	_, role, ok := actor(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	filter := repository.ProductFilter{}
	if cat := r.URL.Query().Get("category"); cat != "" {
		value := domain.ProductCategory(cat)
		filter.Category = &value
	}
	// Admin/manager may explicitly inspect inactive products.
	if role != domain.RoleClient {
		if active := r.URL.Query().Get("is_active"); active != "" {
			switch active {
			case "true":
				v := true
				filter.IsActive = &v
			case "false":
				v := false
				filter.IsActive = &v
			default:
				respondError(w, http.StatusBadRequest, "invalid is_active")
				return
			}
		}
	}
	products, err := h.shopSvc.ListProducts(r.Context(), role, filter)
	if err != nil {
		handleError(w, err)
		return
	}
	respond(w, http.StatusOK, products)
}

func (h *ShopHandler) GetProduct(w http.ResponseWriter, r *http.Request) {
	id, err := parseIDFromPath(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid product id")
		return
	}
	_, role, ok := actor(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	product, err := h.shopSvc.GetProduct(r.Context(), role, id)
	if err != nil {
		handleError(w, err)
		return
	}
	respond(w, http.StatusOK, product)
}

func (h *ShopHandler) PurchaseProduct(w http.ResponseWriter, r *http.Request) {
	userID, _, ok := actor(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var input domain.PurchaseProductInput
	if err := decode(w, r, &input); err != nil || input.ProductID <= 0 {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	result, err := h.shopSvc.PurchaseProduct(r.Context(), userID, input.ProductID)
	if err != nil {
		handleError(w, err)
		return
	}
	respond(w, http.StatusCreated, result)
}

func (h *ShopHandler) GetMySubscriptions(w http.ResponseWriter, r *http.Request) {
	userID, _, ok := actor(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	items, err := h.shopSvc.GetMySubscriptions(r.Context(), userID)
	if err != nil {
		handleError(w, err)
		return
	}
	respond(w, http.StatusOK, items)
}

func (h *ShopHandler) GetMyOrders(w http.ResponseWriter, r *http.Request) {
	userID, _, ok := actor(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	items, err := h.shopSvc.GetMyOrders(r.Context(), userID)
	if err != nil {
		handleError(w, err)
		return
	}
	respond(w, http.StatusOK, items)
}

func (h *ShopHandler) CreateProduct(w http.ResponseWriter, r *http.Request) {
	var input domain.CreateProductInput
	if err := decode(w, r, &input); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	product, err := h.shopSvc.CreateProduct(r.Context(), input)
	if err != nil {
		handleError(w, err)
		return
	}
	respond(w, http.StatusCreated, product)
}

func (h *ShopHandler) UpdateProduct(w http.ResponseWriter, r *http.Request) {
	id, err := parseIDFromPath(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid product id")
		return
	}
	var input domain.CreateProductInput
	if err := decode(w, r, &input); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	product, err := h.shopSvc.UpdateProduct(r.Context(), id, input)
	if err != nil {
		handleError(w, err)
		return
	}
	respond(w, http.StatusOK, product)
}

func (h *ShopHandler) DeleteProduct(w http.ResponseWriter, r *http.Request) {
	id, err := parseIDFromPath(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid product id")
		return
	}
	if err := h.shopSvc.DeleteProduct(r.Context(), id); err != nil {
		handleError(w, err)
		return
	}
	respond(w, http.StatusNoContent, nil)
}

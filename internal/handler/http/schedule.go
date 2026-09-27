package httphandler

import (
	"net/http"
	"strings"
	"time"

	"github.com/sfedu-crm/internal/domain"
	"github.com/sfedu-crm/internal/service"
)

type ScheduleHandler struct{ scheduleSvc *service.ScheduleService }

func NewScheduleHandler(scheduleSvc *service.ScheduleService) *ScheduleHandler {
	return &ScheduleHandler{scheduleSvc: scheduleSvc}
}

func (h *ScheduleHandler) GetSchedule(w http.ResponseWriter, r *http.Request) {
	userID, role, ok := actor(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	q := r.URL.Query()
	filter := domain.ScheduleFilter{}

	if value := q.Get("date_from"); value != "" {
		t, err := parseDate(value)
		if err != nil {
			respondError(w, http.StatusBadRequest, "invalid date_from format (YYYY-MM-DD)")
			return
		}
		filter.DateFrom = &t
	}
	if value := q.Get("date_to"); value != "" {
		t, err := parseDate(value)
		if err != nil {
			respondError(w, http.StatusBadRequest, "invalid date_to format (YYYY-MM-DD)")
			return
		}
		t = t.AddDate(0, 0, 1) // date_to is inclusive for API callers.
		filter.DateTo = &t
	}
	if filter.DateFrom == nil && filter.DateTo == nil {
		now := time.Now().UTC()
		from := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
		to := from.AddDate(0, 1, 0)
		filter.DateFrom, filter.DateTo = &from, &to
	}
	if filter.DateFrom != nil && filter.DateTo != nil && !filter.DateTo.After(*filter.DateFrom) {
		respondError(w, http.StatusBadRequest, "date_to must be on or after date_from")
		return
	}

	if role != domain.RoleClient {
		if value := q.Get("client_id"); value != "" {
			id, err := parsePositiveInt64(value)
			if err != nil {
				respondError(w, http.StatusBadRequest, "invalid client_id")
				return
			}
			filter.ClientID = &id
		}
	}
	if value := q.Get("trainer_id"); value != "" {
		id, err := parsePositiveInt64(value)
		if err != nil {
			respondError(w, http.StatusBadRequest, "invalid trainer_id")
			return
		}
		filter.TrainerID = &id
	}
	if value := strings.TrimSpace(q.Get("status")); value != "" {
		filter.Status = domain.TrainingStatus(value)
	}

	trainings, err := h.scheduleSvc.GetSchedule(r.Context(), userID, role, filter)
	if err != nil {
		handleError(w, err)
		return
	}
	respond(w, http.StatusOK, trainings)
}

func (h *ScheduleHandler) GetTraining(w http.ResponseWriter, r *http.Request) {
	id, err := parseIDFromPath(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid training id")
		return
	}
	userID, role, ok := actor(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	training, err := h.scheduleSvc.GetTraining(r.Context(), userID, role, id)
	if err != nil {
		handleError(w, err)
		return
	}
	respond(w, http.StatusOK, training)
}

func (h *ScheduleHandler) SubmitTrainingRequest(w http.ResponseWriter, r *http.Request) {
	userID, _, ok := actor(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var input domain.CreateTrainingRequestInput
	if err := decode(w, r, &input); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	req, err := h.scheduleSvc.SubmitTrainingRequest(r.Context(), userID, input)
	if err != nil {
		handleError(w, err)
		return
	}
	respond(w, http.StatusCreated, req)
}

func (h *ScheduleHandler) ListMyRequests(w http.ResponseWriter, r *http.Request) {
	userID, _, ok := actor(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	list, err := h.scheduleSvc.ListMyRequests(r.Context(), userID)
	if err != nil {
		handleError(w, err)
		return
	}
	respond(w, http.StatusOK, list)
}

func (h *ScheduleHandler) ListPendingRequests(w http.ResponseWriter, r *http.Request) {
	list, err := h.scheduleSvc.ListPendingRequests(r.Context())
	if err != nil {
		handleError(w, err)
		return
	}
	respond(w, http.StatusOK, list)
}

func (h *ScheduleHandler) ApproveRequest(w http.ResponseWriter, r *http.Request) {
	id, err := parseIDFromPath(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid request id")
		return
	}
	var input domain.CreateTrainingInput
	if err := decode(w, r, &input); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	training, err := h.scheduleSvc.ApproveRequest(r.Context(), id, input)
	if err != nil {
		handleError(w, err)
		return
	}
	respond(w, http.StatusCreated, training)
}

func (h *ScheduleHandler) RejectRequest(w http.ResponseWriter, r *http.Request) {
	id, err := parseIDFromPath(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid request id")
		return
	}
	if err := h.scheduleSvc.RejectRequest(r.Context(), id); err != nil {
		handleError(w, err)
		return
	}
	respond(w, http.StatusOK, map[string]string{"status": "rejected"})
}

func (h *ScheduleHandler) CreateTraining(w http.ResponseWriter, r *http.Request) {
	var input domain.CreateTrainingInput
	if err := decode(w, r, &input); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	training, err := h.scheduleSvc.CreateTraining(r.Context(), input)
	if err != nil {
		handleError(w, err)
		return
	}
	respond(w, http.StatusCreated, training)
}

func (h *ScheduleHandler) UpdateTraining(w http.ResponseWriter, r *http.Request) {
	id, err := parseIDFromPath(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid training id")
		return
	}
	var input domain.UpdateTrainingInput
	if err := decode(w, r, &input); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	training, err := h.scheduleSvc.UpdateTraining(r.Context(), id, input)
	if err != nil {
		handleError(w, err)
		return
	}
	respond(w, http.StatusOK, training)
}

func (h *ScheduleHandler) DeleteTraining(w http.ResponseWriter, r *http.Request) {
	id, err := parseIDFromPath(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid training id")
		return
	}
	if err := h.scheduleSvc.DeleteTraining(r.Context(), id); err != nil {
		handleError(w, err)
		return
	}
	respond(w, http.StatusNoContent, nil)
}

func (h *ScheduleHandler) CancelMyTraining(w http.ResponseWriter, r *http.Request) {
	id, err := parseIDFromPath(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid training id")
		return
	}
	clientID, _, ok := actor(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if err := h.scheduleSvc.CancelMyTraining(r.Context(), clientID, id); err != nil {
		handleError(w, err)
		return
	}
	respond(w, http.StatusOK, map[string]string{"status": "cancelled"})
}

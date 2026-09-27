package httphandler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/sfedu-crm/internal/domain"
	"github.com/sfedu-crm/internal/repository"
	"github.com/sfedu-crm/internal/service"
)

type UserHandler struct{ userSvc *service.UserService }

func NewUserHandler(userSvc *service.UserService) *UserHandler { return &UserHandler{userSvc: userSvc} }

func (h *UserHandler) GetMe(w http.ResponseWriter, r *http.Request) {
	userID, _, ok := actor(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	user, err := h.userSvc.GetByID(r.Context(), userID)
	if err != nil {
		handleError(w, err)
		return
	}
	respond(w, http.StatusOK, user)
}

func (h *UserHandler) UpdateMe(w http.ResponseWriter, r *http.Request) {
	userID, _, ok := actor(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var input domain.UpdateUserInput
	if err := decode(w, r, &input); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	user, err := h.userSvc.UpdateProfile(r.Context(), userID, input)
	if err != nil {
		handleError(w, err)
		return
	}
	respond(w, http.StatusOK, user)
}

func (h *UserHandler) ChangeMyPassword(w http.ResponseWriter, r *http.Request) {
	userID, _, ok := actor(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var input domain.ChangePasswordInput
	if err := decode(w, r, &input); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := h.userSvc.ChangePassword(r.Context(), userID, input); err != nil {
		handleError(w, err)
		return
	}
	respond(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *UserHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	_, role, ok := actor(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	q := r.URL.Query()
	filter := repository.UserFilter{Search: strings.TrimSpace(q.Get("search"))}
	if roleValue := q.Get("role"); roleValue != "" {
		v := domain.Role(roleValue)
		filter.Role = &v
	}
	if activeValue := q.Get("is_active"); activeValue != "" {
		v, err := strconv.ParseBool(activeValue)
		if err != nil {
			respondError(w, http.StatusBadRequest, "invalid is_active")
			return
		}
		filter.IsActive = &v
	}
	users, err := h.userSvc.List(r.Context(), role, filter)
	if err != nil {
		handleError(w, err)
		return
	}
	respond(w, http.StatusOK, users)
}

func (h *UserHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	_, role, ok := actor(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var input domain.CreateUserInput
	if err := decode(w, r, &input); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	user, err := h.userSvc.CreateUser(r.Context(), role, input)
	if err != nil {
		handleError(w, err)
		return
	}
	respond(w, http.StatusCreated, user)
}

func (h *UserHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	id, err := parseIDFromPath(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid user id")
		return
	}
	_, role, ok := actor(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	user, err := h.userSvc.GetForActor(r.Context(), role, id)
	if err != nil {
		handleError(w, err)
		return
	}
	respond(w, http.StatusOK, user)
}

func (h *UserHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	id, err := parseIDFromPath(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid user id")
		return
	}
	_, role, ok := actor(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var input domain.UpdateUserInput
	if err := decode(w, r, &input); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	user, err := h.userSvc.UpdateForActor(r.Context(), role, id, input)
	if err != nil {
		handleError(w, err)
		return
	}
	respond(w, http.StatusOK, user)
}

func (h *UserHandler) AdminResetPassword(w http.ResponseWriter, r *http.Request) {
	id, err := parseIDFromPath(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid user id")
		return
	}
	var input struct {
		NewPassword string `json:"new_password"`
	}
	if err := decode(w, r, &input); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := h.userSvc.AdminResetPassword(r.Context(), id, input.NewPassword); err != nil {
		handleError(w, err)
		return
	}
	respond(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *UserHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	targetID, err := parseIDFromPath(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid user id")
		return
	}
	actorID, _, ok := actor(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if err := h.userSvc.DeleteUser(r.Context(), actorID, targetID); err != nil {
		handleError(w, err)
		return
	}
	respond(w, http.StatusNoContent, nil)
}

func (h *UserHandler) SetActive(active bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		targetID, err := parseIDFromPath(r)
		if err != nil {
			respondError(w, http.StatusBadRequest, "invalid user id")
			return
		}
		actorID, _, ok := actor(r)
		if !ok {
			respondError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		if err := h.userSvc.SetActive(r.Context(), actorID, targetID, active); err != nil {
			handleError(w, err)
			return
		}
		respond(w, http.StatusOK, map[string]string{"status": "ok"})
	}
}

func (h *UserHandler) ListApplications(w http.ResponseWriter, r *http.Request) {
	status := domain.ApplicationStatus(strings.TrimSpace(r.URL.Query().Get("status")))
	apps, err := h.userSvc.ListApplications(r.Context(), status)
	if err != nil {
		handleError(w, err)
		return
	}
	respond(w, http.StatusOK, apps)
}

func (h *UserHandler) RejectApplication(w http.ResponseWriter, r *http.Request) {
	id, err := parseIDFromPath(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid application id")
		return
	}
	if err := h.userSvc.RejectApplication(r.Context(), id); err != nil {
		handleError(w, err)
		return
	}
	respond(w, http.StatusOK, map[string]string{"status": "rejected"})
}

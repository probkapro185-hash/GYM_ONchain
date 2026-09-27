package httphandler

import (
	"context"
	"net/http"

	"github.com/sfedu-crm/internal/domain"
)

type accountAccess interface {
	ApproveApplication(context.Context, int64, ...domain.Gender) (*domain.User, bool, error)
	InviteClient(context.Context, domain.InviteClientInput) (*domain.User, bool, error)
	ResendActivation(context.Context, int64) (bool, error)
	Activate(context.Context, domain.SetPasswordInput) error
	RequestPasswordReset(context.Context, string)
	ResetPassword(context.Context, domain.SetPasswordInput) error
}

type AccountHandler struct{ access accountAccess }

func NewAccountHandler(access accountAccess) *AccountHandler {
	return &AccountHandler{access: access}
}

func (h *AccountHandler) ApproveApplication(w http.ResponseWriter, r *http.Request) {
	id, err := parseIDFromPath(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid application id")
		return
	}
	var body struct {
		Gender domain.Gender `json:"gender"`
	}
	if err := decode(w, r, &body); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	user, emailSent, err := h.access.ApproveApplication(r.Context(), id, body.Gender)
	if err != nil {
		handleError(w, err)
		return
	}
	respond(w, http.StatusCreated, map[string]any{"user": user, "email_sent": emailSent})
}

func (h *AccountHandler) InviteClient(w http.ResponseWriter, r *http.Request) {
	var input domain.InviteClientInput
	if err := decode(w, r, &input); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	user, emailSent, err := h.access.InviteClient(r.Context(), input)
	if err != nil {
		handleError(w, err)
		return
	}
	respond(w, http.StatusCreated, map[string]any{"user": user, "email_sent": emailSent})
}

func (h *AccountHandler) ResendActivation(w http.ResponseWriter, r *http.Request) {
	id, err := parseIDFromPath(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid user id")
		return
	}
	sent, err := h.access.ResendActivation(r.Context(), id)
	if err != nil {
		handleError(w, err)
		return
	}
	respond(w, http.StatusOK, map[string]any{"status": "ok", "email_sent": sent})
}

func (h *AccountHandler) Activate(w http.ResponseWriter, r *http.Request) {
	var input domain.SetPasswordInput
	if err := decode(w, r, &input); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := h.access.Activate(r.Context(), input); err != nil {
		handleError(w, err)
		return
	}
	respond(w, http.StatusOK, map[string]string{"status": "activated"})
}

func (h *AccountHandler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	var input domain.ForgotPasswordInput
	if err := decode(w, r, &input); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	h.access.RequestPasswordReset(r.Context(), input.Email)
	respond(w, http.StatusAccepted, map[string]string{"status": "accepted"})
}

func (h *AccountHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var input domain.SetPasswordInput
	if err := decode(w, r, &input); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := h.access.ResetPassword(r.Context(), input); err != nil {
		handleError(w, err)
		return
	}
	respond(w, http.StatusOK, map[string]string{"status": "password_reset"})
}

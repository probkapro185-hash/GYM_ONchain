package httphandler

import (
	"net/http"

	"github.com/sfedu-crm/internal/domain"
	"github.com/sfedu-crm/internal/service"
)

type AuthHandler struct{ authSvc *service.AuthService }

func NewAuthHandler(authSvc *service.AuthService) *AuthHandler { return &AuthHandler{authSvc: authSvc} }

func (h *AuthHandler) SubmitApplication(w http.ResponseWriter, r *http.Request) {
	var input domain.CreateApplicationInput
	if err := decode(w, r, &input); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	app, err := h.authSvc.SubmitApplication(r.Context(), input)
	if err != nil {
		handleError(w, err)
		return
	}
	respond(w, http.StatusCreated, app)
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var input domain.LoginInput
	if err := decode(w, r, &input); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	token, user, err := h.authSvc.Login(r.Context(), input)
	if err != nil {
		handleError(w, err)
		return
	}
	respond(w, http.StatusOK, map[string]any{"token": token, "user": user})
}

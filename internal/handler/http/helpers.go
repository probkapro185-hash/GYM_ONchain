package httphandler

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/sfedu-crm/internal/domain"
	"github.com/sfedu-crm/internal/middleware"
)

const maxRequestBody = 1 << 20 // 1 MiB

func respond(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if data != nil {
		if err := json.NewEncoder(w).Encode(data); err != nil {
			slog.Error("encode response", "error", err)
		}
	}
}

func respondError(w http.ResponseWriter, status int, message string) {
	respond(w, status, map[string]string{"error": message})
}

func decode(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBody)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return err
	}
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("request body must contain one JSON object")
		}
		return err
	}
	return nil
}

func handleError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrInvalidAccountLink):
		respond(w, http.StatusBadRequest, map[string]string{"code": "invalid_account_link", "error": "Ссылка недействительна, уже использована или срок её действия истёк."})
	case errors.Is(err, domain.ErrNotFound), errors.Is(err, domain.ErrNoActiveSubscription):
		respondError(w, http.StatusNotFound, publicMessage(err))
	case errors.Is(err, domain.ErrAlreadyExists), errors.Is(err, domain.ErrConflict), errors.Is(err, domain.ErrInvalidTransition):
		respondError(w, http.StatusConflict, publicMessage(err))
	case errors.Is(err, domain.ErrInvalidInput), errors.Is(err, domain.ErrInvalidPhone), errors.Is(err, domain.ErrInvalidEmail):
		respondError(w, http.StatusBadRequest, publicMessage(err))
	case errors.Is(err, domain.ErrUnauthorized), errors.Is(err, domain.ErrInvalidPassword):
		respondError(w, http.StatusUnauthorized, "unauthorized")
	case errors.Is(err, domain.ErrForbidden):
		respondError(w, http.StatusForbidden, "forbidden")
	case errors.Is(err, domain.ErrInsufficientFunds):
		respondError(w, http.StatusPaymentRequired, "insufficient funds")
	case errors.Is(err, domain.ErrServiceUnavailable):
		respondError(w, http.StatusServiceUnavailable, "AI service unavailable")
	default:
		slog.Error("unhandled request error", "error", err)
		respondError(w, http.StatusInternalServerError, "internal server error")
	}
}

func publicMessage(err error) string {
	switch {
	case errors.Is(err, domain.ErrNoActiveSubscription):
		return "no active subscription"
	case errors.Is(err, domain.ErrNotFound):
		return "not found"
	case errors.Is(err, domain.ErrAlreadyExists):
		return "already exists"
	case errors.Is(err, domain.ErrConflict):
		return "conflict"
	case errors.Is(err, domain.ErrInvalidTransition):
		return "invalid status transition"
	default:
		return err.Error()
	}
}

func parseIDFromPath(r *http.Request) (int64, error) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		return 0, domain.ErrInvalidInput
	}
	return id, nil
}

func actor(r *http.Request) (int64, domain.Role, bool) {
	id, okID := middleware.GetUserID(r.Context())
	role, okRole := middleware.GetRole(r.Context())
	return id, role, okID && okRole
}

func parsePositiveInt64(value string) (int64, error) {
	id, err := strconv.ParseInt(value, 10, 64)
	if err != nil || id <= 0 {
		return 0, domain.ErrInvalidInput
	}
	return id, nil
}

func parseDate(value string) (time.Time, error) {
	return time.ParseInLocation(time.DateOnly, value, time.UTC)
}

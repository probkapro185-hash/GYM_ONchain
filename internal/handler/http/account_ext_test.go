package httphandler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sfedu-crm/internal/domain"
)

type fakeAccountHandlerService struct {
	approveFn  func(context.Context, int64) (*domain.User, bool, error)
	inviteFn   func(context.Context, domain.InviteClientInput) (*domain.User, bool, error)
	resendFn   func(context.Context, int64) (bool, error)
	activateFn func(context.Context, domain.SetPasswordInput) error
	forgotFn   func(context.Context, string)
	resetFn    func(context.Context, domain.SetPasswordInput) error
}

func (f *fakeAccountHandlerService) ApproveApplication(c context.Context, id int64, gender ...domain.Gender) (*domain.User, bool, error) {
	if f.approveFn != nil {
		return f.approveFn(c, id)
	}
	return nil, false, domain.ErrNotFound
}
func (f *fakeAccountHandlerService) InviteClient(c context.Context, in domain.InviteClientInput) (*domain.User, bool, error) {
	if f.inviteFn != nil {
		return f.inviteFn(c, in)
	}
	return nil, false, domain.ErrNotFound
}
func (f *fakeAccountHandlerService) ResendActivation(c context.Context, id int64) (bool, error) {
	if f.resendFn != nil {
		return f.resendFn(c, id)
	}
	return false, domain.ErrNotFound
}
func (f *fakeAccountHandlerService) Activate(c context.Context, in domain.SetPasswordInput) error {
	if f.activateFn != nil {
		return f.activateFn(c, in)
	}
	return nil
}
func (f *fakeAccountHandlerService) RequestPasswordReset(c context.Context, email string) {
	if f.forgotFn != nil {
		f.forgotFn(c, email)
	}
}
func (f *fakeAccountHandlerService) ResetPassword(c context.Context, in domain.SetPasswordInput) error {
	if f.resetFn != nil {
		return f.resetFn(c, in)
	}
	return nil
}

func TestAccountHandlerApproveRejectsStaffPasswordField(t *testing.T) {
	called := false
	h := NewAccountHandler(&fakeAccountHandlerService{approveFn: func(_ context.Context, id int64) (*domain.User, bool, error) {
		called = true
		return &domain.User{ID: 9}, true, nil
	}})
	r := httptest.NewRequest(http.MethodPost, "/api/v1/applications/7/approve", bytes.NewBufferString(`{"password":"staff-secret"}`))
	r.SetPathValue("id", "7")
	w := httptest.NewRecorder()
	h.ApproveApplication(w, r)
	if w.Code != http.StatusBadRequest || called {
		t.Fatalf("password field must be rejected: code=%d called=%v body=%s", w.Code, called, w.Body.String())
	}
}

func TestAccountHandlerApproveSuccess(t *testing.T) {
	called := false
	h := NewAccountHandler(&fakeAccountHandlerService{approveFn: func(_ context.Context, id int64) (*domain.User, bool, error) {
		called = id == 7
		return &domain.User{ID: 9, Email: "client@gmail.com", PasswordSetupRequired: true}, true, nil
	}})
	r := httptest.NewRequest(http.MethodPost, "/api/v1/applications/7/approve", bytes.NewBufferString(`{}`))
	r.SetPathValue("id", "7")
	w := httptest.NewRecorder()
	h.ApproveApplication(w, r)
	if w.Code != http.StatusCreated || !called {
		t.Fatalf("code=%d body=%s", w.Code, w.Body.String())
	}
	var out map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if sent, ok := out["email_sent"].(bool); !ok || !sent {
		t.Fatalf("unexpected response: %#v", out)
	}
}

func TestAccountHandlerForgotPasswordAlwaysAccepted(t *testing.T) {
	var got string
	h := NewAccountHandler(&fakeAccountHandlerService{forgotFn: func(_ context.Context, email string) { got = email }})
	r := httptest.NewRequest(http.MethodPost, "/api/v1/auth/forgot-password", bytes.NewBufferString(`{"email":"u@gmail.com"}`))
	w := httptest.NewRecorder()
	h.ForgotPassword(w, r)
	if w.Code != http.StatusAccepted || got != "u@gmail.com" {
		t.Fatalf("code=%d got=%q", w.Code, got)
	}
}

func TestAccountHandlerActivateValidationError(t *testing.T) {
	h := NewAccountHandler(&fakeAccountHandlerService{activateFn: func(context.Context, domain.SetPasswordInput) error { return fmtInvalidLink() }})
	r := httptest.NewRequest(http.MethodPost, "/api/v1/auth/activate", bytes.NewBufferString(`{"token":"bad","new_password":"password1"}`))
	w := httptest.NewRecorder()
	h.Activate(w, r)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("code=%d body=%s", w.Code, w.Body.String())
	}
}

func TestAccountHandlerResetRejectsMalformedJSON(t *testing.T) {
	h := NewAccountHandler(&fakeAccountHandlerService{})
	r := httptest.NewRequest(http.MethodPost, "/api/v1/auth/reset-password", bytes.NewBufferString(`{"token":`))
	w := httptest.NewRecorder()
	h.ResetPassword(w, r)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("code=%d", w.Code)
	}
}

func TestAccountHandlerInviteClient(t *testing.T) {
	var got domain.InviteClientInput
	h := NewAccountHandler(&fakeAccountHandlerService{inviteFn: func(_ context.Context, in domain.InviteClientInput) (*domain.User, bool, error) {
		got = in
		return &domain.User{ID: 4}, false, nil
	}})
	r := httptest.NewRequest(http.MethodPost, "/api/v1/users/invite", bytes.NewBufferString(`{"full_name":"Иван Иванов","phone":"+79991234567","email":"u@gmail.com","gender":"male"}`))
	w := httptest.NewRecorder()
	h.InviteClient(w, r)
	if w.Code != http.StatusCreated || got.Email != "u@gmail.com" {
		t.Fatalf("code=%d got=%+v", w.Code, got)
	}
}

func TestAccountHandlerResendMapsConflict(t *testing.T) {
	h := NewAccountHandler(&fakeAccountHandlerService{resendFn: func(context.Context, int64) (bool, error) { return false, domain.ErrConflict }})
	r := httptest.NewRequest(http.MethodPost, "/api/v1/users/5/resend-activation", nil)
	r.SetPathValue("id", "5")
	w := httptest.NewRecorder()
	h.ResendActivation(w, r)
	if w.Code != http.StatusConflict {
		t.Fatalf("code=%d", w.Code)
	}
}

func fmtInvalidLink() error {
	return errors.Join(domain.ErrInvalidInput, errors.New("invalid or expired link"))
}

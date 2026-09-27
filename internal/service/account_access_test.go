package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/sfedu-crm/internal/domain"
	"github.com/sfedu-crm/pkg/hash"
	"github.com/sfedu-crm/pkg/mailer"
)

type fakeAccountAccessRepo struct {
	createPendingFn func(context.Context, domain.CreateUserInput) (*domain.User, error)
	createTokenFn   func(context.Context, int64, domain.AccountTokenPurpose, string, time.Time) error
	getTokenFn      func(context.Context, string, domain.AccountTokenPurpose) (*domain.AccountToken, error)
	markUsedFn      func(context.Context, int64) error
	invalidateFn    func(context.Context, int64, domain.AccountTokenPurpose) error
	activateUserFn  func(context.Context, int64, string) error
	resetPasswordFn func(context.Context, int64, string) error
}

func (f *fakeAccountAccessRepo) CreatePendingUser(c context.Context, i domain.CreateUserInput) (*domain.User, error) {
	if f.createPendingFn != nil {
		return f.createPendingFn(c, i)
	}
	return nil, domain.ErrNotFound
}
func (f *fakeAccountAccessRepo) CreateToken(c context.Context, id int64, p domain.AccountTokenPurpose, h string, e time.Time) error {
	if f.createTokenFn != nil {
		return f.createTokenFn(c, id, p, h, e)
	}
	return nil
}
func (f *fakeAccountAccessRepo) GetTokenForUpdate(c context.Context, h string, p domain.AccountTokenPurpose) (*domain.AccountToken, error) {
	if f.getTokenFn != nil {
		return f.getTokenFn(c, h, p)
	}
	return nil, domain.ErrNotFound
}
func (f *fakeAccountAccessRepo) MarkTokenUsed(c context.Context, id int64) error {
	if f.markUsedFn != nil {
		return f.markUsedFn(c, id)
	}
	return nil
}
func (f *fakeAccountAccessRepo) InvalidateTokens(c context.Context, id int64, p domain.AccountTokenPurpose) error {
	if f.invalidateFn != nil {
		return f.invalidateFn(c, id, p)
	}
	return nil
}
func (f *fakeAccountAccessRepo) ActivateUser(c context.Context, id int64, h string) error {
	if f.activateUserFn != nil {
		return f.activateUserFn(c, id, h)
	}
	return nil
}
func (f *fakeAccountAccessRepo) ResetPassword(c context.Context, id int64, h string) error {
	if f.resetPasswordFn != nil {
		return f.resetPasswordFn(c, id, h)
	}
	return nil
}

type captureMailer struct {
	messages []mailer.Message
	err      error
}

func (m *captureMailer) Send(_ context.Context, msg mailer.Message) error {
	m.messages = append(m.messages, msg)
	return m.err
}

func newAccountSvc(users *fakeUserRepo, apps *fakeApplicationRepo, access *fakeAccountAccessRepo, mail *captureMailer) *AccountAccessService {
	return NewAccountAccessService(users, apps, access, &fakeTx{}, hash.NewBcrypt(4), mail,
		"http://localhost:8080", 24*time.Hour, time.Hour, nil)
}

func TestAccountApproveCreatesPendingUserAndSendsActivation(t *testing.T) {
	apps := &fakeApplicationRepo{
		getByIDForUpdateFn: func(context.Context, int64) (*domain.ApplicationRequest, error) {
			return &domain.ApplicationRequest{Gender: domain.GenderFemale, ID: 7, FullName: "Иван Иванов", Phone: "+79991234567", Email: "client@gmail.com", Status: domain.ApplicationPending}, nil
		},
	}
	var status domain.ApplicationStatus
	apps.updateStatusFn = func(_ context.Context, _ int64, s domain.ApplicationStatus) error { status = s; return nil }
	access := &fakeAccountAccessRepo{}
	access.createPendingFn = func(_ context.Context, in domain.CreateUserInput) (*domain.User, error) {
		if in.Gender != domain.GenderFemale {
			t.Fatalf("application gender was not preserved: %s", in.Gender)
		}
		if in.Password == "" || in.Password == "temporary" {
			t.Fatal("temporary password must be hashed")
		}
		return &domain.User{ID: 11, FullName: in.FullName, Email: in.Email, Role: domain.RoleClient, PasswordSetupRequired: true}, nil
	}
	var purpose domain.AccountTokenPurpose
	var tokenHash string
	access.createTokenFn = func(_ context.Context, id int64, p domain.AccountTokenPurpose, h string, exp time.Time) error {
		if id != 11 || !exp.After(time.Now()) {
			t.Fatalf("bad token persistence")
		}
		purpose, tokenHash = p, h
		return nil
	}
	mail := &captureMailer{}
	svc := newAccountSvc(&fakeUserRepo{}, apps, access, mail)
	user, sent, err := svc.ApproveApplication(context.Background(), 7)
	if err != nil {
		t.Fatal(err)
	}
	if user.ID != 11 || !sent || status != domain.ApplicationApproved {
		t.Fatalf("bad approval result: user=%+v sent=%v status=%s", user, sent, status)
	}
	if purpose != domain.AccountTokenActivation || len(tokenHash) != 64 {
		t.Fatalf("bad activation token: %s %q", purpose, tokenHash)
	}
	if len(mail.messages) != 1 || !strings.Contains(mail.messages[0].TextBody, "action=activate&token=") || !strings.Contains(mail.messages[0].ActionURL, "action=activate&token=") {
		t.Fatalf("activation email missing: %+v", mail.messages)
	}
}

func TestAccountApproveMailFailureDoesNotUndoAccount(t *testing.T) {
	apps := &fakeApplicationRepo{
		getByIDForUpdateFn: func(context.Context, int64) (*domain.ApplicationRequest, error) {
			return &domain.ApplicationRequest{Gender: domain.GenderFemale, ID: 1, FullName: "Иван Иванов", Phone: "+79991234567", Email: "client@gmail.com", Status: domain.ApplicationPending}, nil
		},
		updateStatusFn: func(context.Context, int64, domain.ApplicationStatus) error { return nil },
	}
	access := &fakeAccountAccessRepo{createPendingFn: func(_ context.Context, in domain.CreateUserInput) (*domain.User, error) {
		return &domain.User{ID: 2, FullName: in.FullName, Email: in.Email, Role: domain.RoleClient, PasswordSetupRequired: true}, nil
	}}
	mail := &captureMailer{err: errors.New("smtp down")}
	svc := newAccountSvc(&fakeUserRepo{}, apps, access, mail)
	user, sent, err := svc.ApproveApplication(context.Background(), 1)
	if err != nil || user == nil {
		t.Fatalf("account creation should survive mail failure: user=%+v err=%v", user, err)
	}
	if sent {
		t.Fatal("email_sent must be false")
	}
}

func TestAccountActivateConsumesValidToken(t *testing.T) {
	plain := "activation-token"
	token := &domain.AccountToken{ID: 3, UserID: 9, Purpose: domain.AccountTokenActivation, ExpiresAt: time.Now().Add(time.Hour)}
	access := &fakeAccountAccessRepo{}
	access.getTokenFn = func(_ context.Context, got string, purpose domain.AccountTokenPurpose) (*domain.AccountToken, error) {
		if got != hashToken(plain) || purpose != domain.AccountTokenActivation {
			t.Fatalf("wrong token lookup")
		}
		return token, nil
	}
	h := hash.NewBcrypt(4)
	activated := false
	access.activateUserFn = func(_ context.Context, id int64, pwHash string) error {
		activated = id == 9 && h.Compare(pwHash, "newpass123")
		return nil
	}
	used := false
	access.markUsedFn = func(_ context.Context, id int64) error { used = id == 3; return nil }
	svc := NewAccountAccessService(&fakeUserRepo{}, &fakeApplicationRepo{}, access, &fakeTx{}, h, &captureMailer{}, "http://localhost:8080", time.Hour, time.Hour, nil)
	if err := svc.Activate(context.Background(), domain.SetPasswordInput{Token: plain, NewPassword: "newpass123"}); err != nil {
		t.Fatal(err)
	}
	if !activated || !used {
		t.Fatalf("activation was not committed: activated=%v used=%v", activated, used)
	}
}

func TestAccountActivateRejectsExpiredToken(t *testing.T) {
	access := &fakeAccountAccessRepo{getTokenFn: func(context.Context, string, domain.AccountTokenPurpose) (*domain.AccountToken, error) {
		return &domain.AccountToken{ID: 1, UserID: 2, ExpiresAt: time.Now().Add(-time.Minute)}, nil
	}}
	activated := false
	access.activateUserFn = func(context.Context, int64, string) error { activated = true; return nil }
	svc := newAccountSvc(&fakeUserRepo{}, &fakeApplicationRepo{}, access, &captureMailer{})
	err := svc.Activate(context.Background(), domain.SetPasswordInput{Token: "expired", NewPassword: "newpass123"})
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("want invalid input, got %v", err)
	}
	if activated {
		t.Fatal("expired token must not activate account")
	}
}

func TestAccountForgotPasswordDoesNotEnumerateUnknownEmail(t *testing.T) {
	mail := &captureMailer{}
	users := &fakeUserRepo{getByEmailFn: func(context.Context, string) (*domain.User, error) { return nil, domain.ErrNotFound }}
	svc := newAccountSvc(users, &fakeApplicationRepo{}, &fakeAccountAccessRepo{}, mail)
	svc.RequestPasswordReset(context.Background(), "unknown@gmail.com")
	if len(mail.messages) != 0 {
		t.Fatal("unknown account must not receive mail")
	}
}

func TestAccountPasswordResetSendsAndConsumesResetToken(t *testing.T) {
	user := &domain.User{ID: 5, FullName: "Иван Иванов", Email: "client@gmail.com", IsActive: true}
	users := &fakeUserRepo{getByEmailFn: func(context.Context, string) (*domain.User, error) { return user, nil }, getByIDForUpdateFn: func(context.Context, int64) (*domain.User, error) { return user, nil }}
	access := &fakeAccountAccessRepo{}
	var persistedHash string
	access.createTokenFn = func(_ context.Context, id int64, p domain.AccountTokenPurpose, h string, _ time.Time) error {
		if id != 5 || p != domain.AccountTokenPasswordReset {
			t.Fatal("wrong reset token")
		}
		persistedHash = h
		return nil
	}
	mail := &captureMailer{}
	h := hash.NewBcrypt(4)
	svc := NewAccountAccessService(users, &fakeApplicationRepo{}, access, &fakeTx{}, h, mail, "http://localhost:8080", time.Hour, time.Hour, nil)
	svc.RequestPasswordReset(context.Background(), "CLIENT@GMAIL.COM")
	if len(mail.messages) != 1 || !strings.Contains(mail.messages[0].TextBody, "action=reset&token=") || persistedHash == "" || !strings.Contains(mail.messages[0].ActionURL, "action=reset&token=") {
		t.Fatalf("reset mail/token missing")
	}
	// Extract the plain token from the generated link and consume it.
	body := mail.messages[0].TextBody
	idx := strings.Index(body, "token=")
	if idx < 0 {
		t.Fatal("token missing from mail")
	}
	plain := strings.Fields(body[idx+len("token="):])[0]
	access.getTokenFn = func(_ context.Context, got string, p domain.AccountTokenPurpose) (*domain.AccountToken, error) {
		if got != hashToken(plain) || p != domain.AccountTokenPasswordReset {
			t.Fatal("wrong reset lookup")
		}
		return &domain.AccountToken{ID: 8, UserID: 5, Purpose: p, ExpiresAt: time.Now().Add(time.Hour)}, nil
	}
	reset := false
	access.resetPasswordFn = func(_ context.Context, id int64, pwHash string) error {
		reset = id == 5 && h.Compare(pwHash, "changed123")
		return nil
	}
	if err := svc.ResetPassword(context.Background(), domain.SetPasswordInput{Token: plain, NewPassword: "changed123"}); err != nil {
		t.Fatal(err)
	}
	if !reset {
		t.Fatal("password was not reset")
	}
}

func TestAccountInviteClientUsesPendingActivation(t *testing.T) {
	access := &fakeAccountAccessRepo{}
	access.createPendingFn = func(_ context.Context, in domain.CreateUserInput) (*domain.User, error) {
		if in.Role != domain.RoleClient || in.Email != "client@gmail.com" || in.Phone != "+79991234567" {
			t.Fatalf("unexpected create input: %+v", in)
		}
		return &domain.User{ID: 15, FullName: in.FullName, Email: in.Email, Role: in.Role, PasswordSetupRequired: true}, nil
	}
	createdToken := false
	access.createTokenFn = func(_ context.Context, id int64, p domain.AccountTokenPurpose, h string, _ time.Time) error {
		createdToken = id == 15 && p == domain.AccountTokenActivation && len(h) == 64
		return nil
	}
	mail := &captureMailer{}
	svc := newAccountSvc(&fakeUserRepo{}, &fakeApplicationRepo{}, access, mail)
	user, sent, err := svc.InviteClient(context.Background(), domain.InviteClientInput{FullName: "Иван Иванов", Phone: "8 (999) 123-45-67", Email: "CLIENT@GMAIL.COM", Gender: domain.GenderMale})
	if err != nil {
		t.Fatal(err)
	}
	if user.ID != 15 || !sent || !createdToken || len(mail.messages) != 1 {
		t.Fatalf("bad invite result: user=%+v sent=%v token=%v mails=%d", user, sent, createdToken, len(mail.messages))
	}
}

func TestAccountResendActivationRequiresPendingClient(t *testing.T) {
	users := &fakeUserRepo{getByIDFn: func(context.Context, int64) (*domain.User, error) {
		return &domain.User{ID: 1, Role: domain.RoleClient, PasswordSetupRequired: false}, nil
	}}
	svc := newAccountSvc(users, &fakeApplicationRepo{}, &fakeAccountAccessRepo{}, &captureMailer{})
	if _, err := svc.ResendActivation(context.Background(), 1); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("expected conflict, got %v", err)
	}
}

func TestResendPreservesExistingActivationLinksWhenMailFails(t *testing.T) {
	for _, deliveryErr := range []error{nil, mailer.ErrLogOnly, errors.New("smtp unavailable")} {
		access := &fakeAccountAccessRepo{}
		created, invalidated := 0, 0
		access.createTokenFn = func(context.Context, int64, domain.AccountTokenPurpose, string, time.Time) error {
			created++
			return nil
		}
		access.invalidateFn = func(context.Context, int64, domain.AccountTokenPurpose) error { invalidated++; return nil }
		users := &fakeUserRepo{getByIDForUpdateFn: func(context.Context, int64) (*domain.User, error) {
			return &domain.User{ID: 1, Role: domain.RoleClient, PasswordSetupRequired: true}, nil
		}}
		mail := &captureMailer{err: deliveryErr}
		sent, err := newAccountSvc(users, &fakeApplicationRepo{}, access, mail).ResendActivation(context.Background(), 1)
		if err != nil || sent != (deliveryErr == nil) || created != 1 || invalidated != 0 {
			t.Fatalf("sent=%v err=%v created=%d invalidated=%d", sent, err, created, invalidated)
		}
	}
}
func TestInvalidLinkHasDedicatedError(t *testing.T) {
	svc := newAccountSvc(&fakeUserRepo{}, &fakeApplicationRepo{}, &fakeAccountAccessRepo{}, &captureMailer{})
	for _, token := range []string{"", "missing"} {
		err := svc.Activate(context.Background(), domain.SetPasswordInput{Token: token, NewPassword: "validpass123"})
		if !errors.Is(err, domain.ErrInvalidAccountLink) || !errors.Is(err, domain.ErrInvalidInput) {
			t.Fatalf("unexpected error: %v", err)
		}
	}
}

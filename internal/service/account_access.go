package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"html"
	"log/slog"
	"net/url"
	"strings"
	"time"

	"github.com/sfedu-crm/internal/domain"
	"github.com/sfedu-crm/internal/repository"
	"github.com/sfedu-crm/internal/validator"
	"github.com/sfedu-crm/pkg/hash"
	"github.com/sfedu-crm/pkg/mailer"
)

type AccountAccessService struct {
	users         repository.UserRepository
	apps          repository.ApplicationRepository
	access        repository.AccountAccessRepository
	tx            repository.TransactionManager
	hasher        *hash.Bcrypt
	mail          mailer.Sender
	baseURL       string
	activationTTL time.Duration
	resetTTL      time.Duration
	log           *slog.Logger
}

func NewAccountAccessService(
	users repository.UserRepository,
	apps repository.ApplicationRepository,
	access repository.AccountAccessRepository,
	tx repository.TransactionManager,
	hasher *hash.Bcrypt,
	mail mailer.Sender,
	baseURL string,
	activationTTL, resetTTL time.Duration,
	log *slog.Logger,
) *AccountAccessService {
	if log == nil {
		log = slog.Default()
	}
	return &AccountAccessService{
		users: users, apps: apps, access: access, tx: tx, hasher: hasher, mail: mail,
		baseURL: strings.TrimRight(baseURL, "/"), activationTTL: activationTTL, resetTTL: resetTTL, log: log,
	}
}

// ApproveApplication creates a locked client account and a one-time activation link.
// Email delivery is deliberately outside the database transaction. A delivery failure
// does not roll back the approved application; staff can resend the activation link.
func (s *AccountAccessService) ApproveApplication(ctx context.Context, appID int64, gender ...domain.Gender) (*domain.User, bool, error) {
	plainToken, tokenHash, err := newOneTimeToken()
	if err != nil {
		return nil, false, fmt.Errorf("generate activation token: %w", err)
	}
	temporaryPassword, _, err := newOneTimeToken()
	if err != nil {
		return nil, false, fmt.Errorf("generate temporary credential: %w", err)
	}

	var created *domain.User
	err = s.tx.WithinTransaction(ctx, func(txCtx context.Context) error {
		app, err := s.apps.GetByIDForUpdate(txCtx, appID)
		if err != nil {
			return err
		}
		if app.Status != domain.ApplicationPending {
			return fmt.Errorf("%w: application is already processed", domain.ErrConflict)
		}

		input := domain.CreateUserInput{
			FullName: app.FullName,
			Phone:    app.Phone,
			Email:    app.Email,
			Password: temporaryPassword,
			Role:     domain.RoleClient,
			Gender:   app.Gender,
		}
		if len(gender) > 0 && gender[0] != "" {
			input.Gender = gender[0]
		}
		if err := validator.ValidateGender(input.Gender); err != nil {
			return err
		}
		if err := normalizeUserInput(&input); err != nil {
			return err
		}
		passwordHash, err := s.hasher.Hash(input.Password)
		if err != nil {
			return fmt.Errorf("hash temporary credential: %w", err)
		}
		input.Password = passwordHash
		created, err = s.access.CreatePendingUser(txCtx, input)
		if err != nil {
			return err
		}
		if err := s.access.CreateToken(txCtx, created.ID, domain.AccountTokenActivation, tokenHash, time.Now().Add(s.activationTTL)); err != nil {
			return err
		}
		return s.apps.UpdateStatus(txCtx, appID, domain.ApplicationApproved)
	})
	if err != nil {
		return nil, false, err
	}

	sent := s.sendActivationEmail(ctx, created, plainToken) == nil
	return created, sent, nil
}

func (s *AccountAccessService) InviteClient(ctx context.Context, input domain.InviteClientInput) (*domain.User, bool, error) {
	plainToken, tokenHash, err := newOneTimeToken()
	if err != nil {
		return nil, false, fmt.Errorf("generate activation token: %w", err)
	}
	temporaryPassword, _, err := newOneTimeToken()
	if err != nil {
		return nil, false, fmt.Errorf("generate temporary credential: %w", err)
	}
	create := domain.CreateUserInput{
		FullName: input.FullName,
		Phone:    input.Phone,
		Email:    input.Email,
		Password: temporaryPassword,
		Role:     domain.RoleClient,
		Gender:   input.Gender,
	}
	if err := normalizeUserInput(&create); err != nil {
		return nil, false, err
	}
	passwordHash, err := s.hasher.Hash(create.Password)
	if err != nil {
		return nil, false, fmt.Errorf("hash temporary credential: %w", err)
	}
	create.Password = passwordHash

	var created *domain.User
	if err := s.tx.WithinTransaction(ctx, func(txCtx context.Context) error {
		var err error
		created, err = s.access.CreatePendingUser(txCtx, create)
		if err != nil {
			return err
		}
		return s.access.CreateToken(txCtx, created.ID, domain.AccountTokenActivation, tokenHash, time.Now().Add(s.activationTTL))
	}); err != nil {
		return nil, false, err
	}
	return created, s.sendActivationEmail(ctx, created, plainToken) == nil, nil
}

func (s *AccountAccessService) ResendActivation(ctx context.Context, userID int64) (bool, error) {
	plainToken, tokenHash, err := newOneTimeToken()
	if err != nil {
		return false, fmt.Errorf("generate activation token: %w", err)
	}
	var user *domain.User
	if err := s.tx.WithinTransaction(ctx, func(txCtx context.Context) error {
		var err error
		user, err = s.users.GetByIDForUpdate(txCtx, userID)
		if err != nil {
			return err
		}
		if user.Role != domain.RoleClient {
			return domain.ErrForbidden
		}
		if !user.PasswordSetupRequired {
			return fmt.Errorf("%w: account is already activated", domain.ErrConflict)
		}
		// Keep unexpired invitations usable when delivery fails or messages arrive out of order.
		// Activating the account still invalidates every activation token atomically.
		return s.access.CreateToken(txCtx, userID, domain.AccountTokenActivation, tokenHash, time.Now().Add(s.activationTTL))
	}); err != nil {
		return false, err
	}
	return s.sendActivationEmail(ctx, user, plainToken) == nil, nil
}

func (s *AccountAccessService) Activate(ctx context.Context, input domain.SetPasswordInput) error {
	return s.consumePasswordToken(ctx, input, domain.AccountTokenActivation, true)
}

// RequestPasswordReset always returns nil for unknown/inactive/pending accounts so the
// public endpoint does not reveal whether an email is registered.
func (s *AccountAccessService) RequestPasswordReset(ctx context.Context, email string) {
	email = strings.ToLower(strings.TrimSpace(email))
	if validator.ValidateEmail(email) != nil {
		return
	}
	user, err := s.users.GetByEmail(ctx, email)
	if err != nil || !user.IsActive || user.PasswordSetupRequired {
		return
	}

	plainToken, tokenHash, err := newOneTimeToken()
	if err != nil {
		s.log.Error("password reset token generation failed", "error", err)
		return
	}
	if err := s.tx.WithinTransaction(ctx, func(txCtx context.Context) error {
		locked, err := s.users.GetByIDForUpdate(txCtx, user.ID)
		if err != nil {
			return err
		}
		if !locked.IsActive || locked.PasswordSetupRequired {
			return domain.ErrForbidden
		}
		if err := s.access.InvalidateTokens(txCtx, user.ID, domain.AccountTokenPasswordReset); err != nil {
			return err
		}
		return s.access.CreateToken(txCtx, user.ID, domain.AccountTokenPasswordReset, tokenHash, time.Now().Add(s.resetTTL))
	}); err != nil {
		s.log.Error("password reset token persistence failed", "user_id", user.ID, "error", err)
		return
	}
	if err := s.sendPasswordResetEmail(ctx, user, plainToken); err != nil && !errors.Is(err, mailer.ErrLogOnly) {
		s.log.Error("password reset email failed", "user_id", user.ID, "error", err)
	}
}

func (s *AccountAccessService) ResetPassword(ctx context.Context, input domain.SetPasswordInput) error {
	return s.consumePasswordToken(ctx, input, domain.AccountTokenPasswordReset, false)
}

func (s *AccountAccessService) consumePasswordToken(ctx context.Context, input domain.SetPasswordInput, purpose domain.AccountTokenPurpose, activate bool) error {
	input.Token = strings.TrimSpace(input.Token)
	if input.Token == "" || len(input.Token) > 256 {
		return fmt.Errorf("%w: %w", domain.ErrInvalidInput, domain.ErrInvalidAccountLink)
	}
	if err := validator.ValidatePassword(input.NewPassword); err != nil {
		return err
	}
	tokenHash := hashToken(input.Token)
	passwordHash, err := s.hasher.Hash(input.NewPassword)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	return s.tx.WithinTransaction(ctx, func(txCtx context.Context) error {
		token, err := s.access.GetTokenForUpdate(txCtx, tokenHash, purpose)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				return fmt.Errorf("%w: %w", domain.ErrInvalidInput, domain.ErrInvalidAccountLink)
			}
			return err
		}
		if token.UsedAt != nil || !token.ExpiresAt.After(time.Now()) {
			return fmt.Errorf("%w: %w", domain.ErrInvalidInput, domain.ErrInvalidAccountLink)
		}
		if activate {
			if err := s.access.ActivateUser(txCtx, token.UserID, passwordHash); err != nil {
				return err
			}
		} else {
			if err := s.access.ResetPassword(txCtx, token.UserID, passwordHash); err != nil {
				return err
			}
		}
		if err := s.access.MarkTokenUsed(txCtx, token.ID); err != nil {
			return err
		}
		return s.access.InvalidateTokens(txCtx, token.UserID, purpose)
	})
}

func (s *AccountAccessService) sendActivationEmail(ctx context.Context, user *domain.User, token string) error {
	link := s.link("activate", token)
	name := html.EscapeString(user.FullName)
	msg := mailer.Message{
		To:        user.Email,
		ActionURL: link,
		Subject:   "SFEDU Gym — создайте пароль для входа",
		TextBody:  fmt.Sprintf("Здравствуйте, %s! Ваша заявка в SFEDU Gym одобрена. Создайте пароль по ссылке: %s\n\nСсылка действует ограниченное время и может быть использована только один раз.", user.FullName, link),
		HTMLBody:  fmt.Sprintf(`<p>Здравствуйте, <strong>%s</strong>!</p><p>Ваша заявка в SFEDU Gym одобрена.</p><p><a href="%s">Создать пароль для входа</a></p><p>Ссылка действует ограниченное время и может быть использована только один раз.</p>`, name, html.EscapeString(link)),
	}
	if err := s.mail.Send(ctx, msg); err != nil {
		if errors.Is(err, mailer.ErrLogOnly) {
			s.log.Info("activation link generated in MAIL_MODE=log", "user_id", user.ID)
			return err
		}
		s.log.Error("activation email failed", "user_id", user.ID, "error", err)
		return err
	}
	return nil
}

func (s *AccountAccessService) sendPasswordResetEmail(ctx context.Context, user *domain.User, token string) error {
	link := s.link("reset", token)
	name := html.EscapeString(user.FullName)
	return s.mail.Send(ctx, mailer.Message{
		To:        user.Email,
		ActionURL: link,
		Subject:   "SFEDU Gym — восстановление пароля",
		TextBody:  fmt.Sprintf("Здравствуйте, %s! Для создания нового пароля откройте ссылку: %s\n\nЕсли вы не запрашивали восстановление, просто проигнорируйте это письмо.", user.FullName, link),
		HTMLBody:  fmt.Sprintf(`<p>Здравствуйте, <strong>%s</strong>!</p><p>Для создания нового пароля откройте ссылку:</p><p><a href="%s">Создать новый пароль</a></p><p>Если вы не запрашивали восстановление, просто проигнорируйте это письмо.</p>`, name, html.EscapeString(link)),
	})
}

func (s *AccountAccessService) link(action, token string) string {
	return s.baseURL + "/?action=" + url.QueryEscape(action) + "&token=" + url.QueryEscape(token)
}

func newOneTimeToken() (plain, hash string, err error) {
	buf := make([]byte, 32)
	if _, err = rand.Read(buf); err != nil {
		return "", "", err
	}
	plain = base64.RawURLEncoding.EncodeToString(buf)
	return plain, hashToken(plain), nil
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

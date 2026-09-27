package config

import "testing"

func TestLoadRejectsPlaceholders(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://u:p@localhost/db")
	t.Setenv("JWT_SECRET", "REPLACE_WITH_AT_LEAST_32_RANDOM_CHARACTERS")
	t.Setenv("ADMIN_PASSWORD", "StrongPassword123")
	t.Setenv("OPENAI_API_KEY", "")
	if _, err := Load(); err == nil {
		t.Fatal("expected placeholder JWT secret to be rejected")
	}
}

func TestLoadValid(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://u:p@localhost/db")
	t.Setenv("JWT_SECRET", "01234567890123456789012345678901")
	t.Setenv("ADMIN_PASSWORD", "StrongPassword123")
	t.Setenv("OPENAI_API_KEY", "")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.BcryptCost != 12 || len(cfg.CORSOrigins) == 0 {
		t.Fatalf("unexpected config: %+v", cfg)
	}
}

func TestLoadRejectsUnsafeOpenAIBaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://u:p@localhost/db")
	t.Setenv("JWT_SECRET", "01234567890123456789012345678901")
	t.Setenv("ADMIN_PASSWORD", "StrongPassword123")
	t.Setenv("OPENAI_API_KEY", "test-key")
	t.Setenv("OPENAI_BASE_URL", "https://user:pass@example.com/v1")

	if _, err := Load(); err == nil {
		t.Fatal("expected OpenAI base URL with credentials to be rejected")
	}
}

func TestLoadAcceptsOpenAIBaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://u:p@localhost/db")
	t.Setenv("JWT_SECRET", "01234567890123456789012345678901")
	t.Setenv("ADMIN_PASSWORD", "StrongPassword123")
	t.Setenv("OPENAI_API_KEY", "test-key")
	t.Setenv("OPENAI_BASE_URL", "https://api.openai.com/v1")

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.OpenAIBaseURL != "https://api.openai.com/v1" {
		t.Fatalf("unexpected OpenAI base URL: %q", cfg.OpenAIBaseURL)
	}
}

func TestLoadRejectsLogMailerInProduction(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://u:p@localhost/db")
	t.Setenv("JWT_SECRET", "01234567890123456789012345678901")
	t.Setenv("ADMIN_PASSWORD", "StrongPassword123")
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("APP_ENV", "production")
	t.Setenv("MAIL_MODE", "log")
	if _, err := Load(); err == nil {
		t.Fatal("expected production to require SMTP")
	}
}

func TestLoadAcceptsSMTPMailer(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://u:p@localhost/db")
	t.Setenv("JWT_SECRET", "01234567890123456789012345678901")
	t.Setenv("ADMIN_PASSWORD", "StrongPassword123")
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("APP_ENV", "development")
	t.Setenv("MAIL_MODE", "smtp")
	t.Setenv("SMTP_HOST", "smtp.gmail.com")
	t.Setenv("SMTP_PORT", "587")
	t.Setenv("SMTP_USERNAME", "sender@gmail.com")
	t.Setenv("SMTP_PASSWORD", "app-password")
	t.Setenv("SMTP_FROM", "sender@gmail.com")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.MailMode != "smtp" || cfg.SMTPPort != 587 || cfg.AppBaseURL == "" {
		t.Fatalf("unexpected mail config: %+v", cfg)
	}
}

package config

import (
	"fmt"
	"net/mail"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Env                  string
	HTTPPort             string
	DatabaseURL          string
	JWTSecret            string
	JWTTokenTTL          time.Duration
	BcryptCost           int
	CORSOrigins          []string
	AdminEmail           string
	AdminPassword        string
	RedisAddr            string
	OpenAIAPIKey         string
	OpenAIBaseURL        string
	AIChatModel          string
	AIEmbeddingModel     string
	AIKnowledgeDir       string
	AppBaseURL           string
	MailMode             string
	SMTPHost             string
	SMTPPort             int
	SMTPUsername         string
	SMTPPassword         string
	SMTPFrom             string
	SMTPFromName         string
	AccountActivationTTL time.Duration
	PasswordResetTTL     time.Duration
}

func Load() (*Config, error) {
	ttlHours, err := getEnvInt("JWT_TOKEN_TTL_HOURS", 24)
	if err != nil || ttlHours < 1 || ttlHours > 24*30 {
		return nil, fmt.Errorf("JWT_TOKEN_TTL_HOURS must be an integer from 1 to 720")
	}
	cost, err := getEnvInt("BCRYPT_COST", 12)
	if err != nil || cost < 10 || cost > 14 {
		return nil, fmt.Errorf("BCRYPT_COST must be an integer from 10 to 14")
	}
	activationHours, err := getEnvInt("ACCOUNT_ACTIVATION_TTL_HOURS", 24)
	if err != nil || activationHours < 1 || activationHours > 24*7 {
		return nil, fmt.Errorf("ACCOUNT_ACTIVATION_TTL_HOURS must be an integer from 1 to 168")
	}
	resetMinutes, err := getEnvInt("PASSWORD_RESET_TTL_MINUTES", 60)
	if err != nil || resetMinutes < 10 || resetMinutes > 24*60 {
		return nil, fmt.Errorf("PASSWORD_RESET_TTL_MINUTES must be an integer from 10 to 1440")
	}
	smtpPort, err := getEnvInt("SMTP_PORT", 587)
	if err != nil || smtpPort < 1 || smtpPort > 65535 {
		return nil, fmt.Errorf("SMTP_PORT must be an integer from 1 to 65535")
	}

	cfg := &Config{
		Env:                  getEnv("APP_ENV", "development"),
		HTTPPort:             getEnv("HTTP_PORT", "8080"),
		DatabaseURL:          strings.TrimSpace(os.Getenv("DATABASE_URL")),
		JWTSecret:            os.Getenv("JWT_SECRET"),
		JWTTokenTTL:          time.Duration(ttlHours) * time.Hour,
		BcryptCost:           cost,
		CORSOrigins:          splitCSV(getEnv("CORS_ALLOWED_ORIGIN", "http://localhost:8080,http://localhost:3000")),
		AdminEmail:           strings.ToLower(strings.TrimSpace(getEnv("ADMIN_EMAIL", "admin@gmail.com"))),
		AdminPassword:        os.Getenv("ADMIN_PASSWORD"),
		RedisAddr:            strings.TrimSpace(getEnv("REDIS_ADDR", "localhost:6379")),
		OpenAIAPIKey:         strings.TrimSpace(os.Getenv("OPENAI_API_KEY")),
		OpenAIBaseURL:        strings.TrimSpace(getEnv("OPENAI_BASE_URL", "https://api.openai.com/v1")),
		AIChatModel:          strings.TrimSpace(getEnv("AI_CHAT_MODEL", "gpt-5.6")),
		AIEmbeddingModel:     strings.TrimSpace(getEnv("AI_EMBEDDING_MODEL", "text-embedding-3-small")),
		AIKnowledgeDir:       strings.TrimSpace(getEnv("AI_KNOWLEDGE_DIR", "knowledge")),
		AppBaseURL:           strings.TrimRight(strings.TrimSpace(getEnv("APP_BASE_URL", "http://localhost:8080")), "/"),
		MailMode:             strings.ToLower(strings.TrimSpace(getEnv("MAIL_MODE", "log"))),
		SMTPHost:             strings.TrimSpace(os.Getenv("SMTP_HOST")),
		SMTPPort:             smtpPort,
		SMTPUsername:         strings.TrimSpace(os.Getenv("SMTP_USERNAME")),
		SMTPPassword:         os.Getenv("SMTP_PASSWORD"),
		SMTPFrom:             strings.TrimSpace(os.Getenv("SMTP_FROM")),
		SMTPFromName:         strings.TrimSpace(getEnv("SMTP_FROM_NAME", "SFEDU Gym")),
		AccountActivationTTL: time.Duration(activationHours) * time.Hour,
		PasswordResetTTL:     time.Duration(resetMinutes) * time.Minute,
	}
	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}
	if len(cfg.JWTSecret) < 32 {
		return nil, fmt.Errorf("JWT_SECRET must contain at least 32 characters")
	}
	lowerSecret := strings.ToLower(cfg.JWTSecret)
	if strings.Contains(lowerSecret, "change_me") || strings.Contains(lowerSecret, "example") || strings.Contains(lowerSecret, "replace_") {
		return nil, fmt.Errorf("JWT_SECRET still contains an insecure placeholder")
	}
	if len(cfg.AdminPassword) < 8 || len(cfg.AdminPassword) > 72 || strings.Contains(strings.ToLower(cfg.AdminPassword), "replace_") || strings.Contains(strings.ToLower(cfg.AdminPassword), "change_me") {
		return nil, fmt.Errorf("ADMIN_PASSWORD must be a real password from 8 to 72 bytes")
	}
	if !validAdminEmail(cfg.AdminEmail) {
		return nil, fmt.Errorf("ADMIN_EMAIL must be a valid @gmail.com or @mail.ru address")
	}
	if cfg.OpenAIAPIKey != "" {
		lowerKey := strings.ToLower(cfg.OpenAIAPIKey)
		if strings.Contains(lowerKey, "replace_") || strings.Contains(lowerKey, "change_me") {
			return nil, fmt.Errorf("OPENAI_API_KEY still contains a placeholder")
		}
		u, err := url.ParseRequestURI(cfg.OpenAIBaseURL)
		if err != nil || u.Host == "" || u.User != nil || (u.Scheme != "http" && u.Scheme != "https") || u.RawQuery != "" || u.Fragment != "" {
			return nil, fmt.Errorf("OPENAI_BASE_URL must be an absolute http/https URL without credentials, query or fragment")
		}
	}
	if cfg.AIChatModel == "" || cfg.AIEmbeddingModel == "" || cfg.AIKnowledgeDir == "" {
		return nil, fmt.Errorf("AI model names and AI_KNOWLEDGE_DIR must not be empty")
	}
	baseURL, err := url.ParseRequestURI(cfg.AppBaseURL)
	if err != nil || baseURL.Host == "" || baseURL.User != nil || (baseURL.Scheme != "http" && baseURL.Scheme != "https") || baseURL.RawQuery != "" || baseURL.Fragment != "" {
		return nil, fmt.Errorf("APP_BASE_URL must be an absolute http/https URL without credentials, query or fragment")
	}
	if cfg.MailMode != "log" && cfg.MailMode != "smtp" {
		return nil, fmt.Errorf("MAIL_MODE must be log or smtp")
	}
	if cfg.Env == "production" && cfg.MailMode != "smtp" {
		return nil, fmt.Errorf("MAIL_MODE must be smtp in production")
	}
	if cfg.MailMode == "smtp" {
		if cfg.SMTPHost == "" || cfg.SMTPUsername == "" || cfg.SMTPPassword == "" {
			return nil, fmt.Errorf("SMTP_HOST, SMTP_USERNAME and SMTP_PASSWORD are required in smtp mode")
		}
		if cfg.SMTPFrom == "" {
			cfg.SMTPFrom = cfg.SMTPUsername
		}
		if _, err := mail.ParseAddress(cfg.SMTPFrom); err != nil {
			return nil, fmt.Errorf("SMTP_FROM must be a valid email address")
		}
	}
	port, err := strconv.Atoi(cfg.HTTPPort)
	if err != nil || port < 1 || port > 65535 {
		return nil, fmt.Errorf("HTTP_PORT must be an integer from 1 to 65535")
	}
	return cfg, nil
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func getEnvInt(key string, fallback int) (int, error) {
	value := os.Getenv(key)
	if value == "" {
		return fallback, nil
	}
	return strconv.Atoi(value)
}

func splitCSV(value string) []string {
	var result []string
	for _, item := range strings.Split(value, ",") {
		item = strings.TrimSpace(item)
		if item != "" {
			result = append(result, item)
		}
	}
	return result
}

func validAdminEmail(value string) bool {
	addr, err := mail.ParseAddress(value)
	if err != nil || !strings.EqualFold(addr.Address, value) {
		return false
	}
	parts := strings.Split(strings.ToLower(addr.Address), "@")
	return len(parts) == 2 && (parts[1] == "gmail.com" || parts[1] == "mail.ru")
}

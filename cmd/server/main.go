package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/sfedu-crm/internal/config"
	httphandler "github.com/sfedu-crm/internal/handler/http"
	"github.com/sfedu-crm/internal/middleware"
	pgrepo "github.com/sfedu-crm/internal/repository/postgres"
	"github.com/sfedu-crm/internal/service"
	rediscache "github.com/sfedu-crm/pkg/cache"
	"github.com/sfedu-crm/pkg/hash"
	"github.com/sfedu-crm/pkg/jwt"
	"github.com/sfedu-crm/pkg/logger"
	"github.com/sfedu-crm/pkg/mailer"
	openaiapi "github.com/sfedu-crm/pkg/openai"
)

const bootstrapPasswordMarker = "BOOTSTRAP_REQUIRED"

func main() {
	// .env is a local-development convenience only. Docker/production should provide environment variables directly.
	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		log.Printf("warning: could not load .env: %v", err)
	}

	appLog := logger.New(os.Getenv("APP_ENV"))
	cfg, err := config.Load()
	if err != nil {
		appLog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	dbStartupCtx, dbStartupCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer dbStartupCancel()

	poolConfig, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		appLog.Error("invalid DATABASE_URL", "error", err)
		os.Exit(1)
	}
	poolConfig.MaxConns = 20
	poolConfig.MinConns = 1
	poolConfig.MaxConnLifetime = time.Hour
	poolConfig.MaxConnIdleTime = 15 * time.Minute

	pool, err := pgxpool.NewWithConfig(dbStartupCtx, poolConfig)
	if err != nil {
		appLog.Error("failed to create db pool", "error", err)
		os.Exit(1)
	}
	defer pool.Close()
	if err := pool.Ping(dbStartupCtx); err != nil {
		appLog.Error("db ping failed", "error", err)
		os.Exit(1)
	}

	dbStartupCancel()
	if err := applyMigrations(cfg.DatabaseURL); err != nil {
		appLog.Error("migration failed", "error", err)
		os.Exit(1)
	}

	hasher := hash.NewBcrypt(cfg.BcryptCost)
	bootstrapCtx, bootstrapCancel := context.WithTimeout(context.Background(), 15*time.Second)
	if err := bootstrapAdmin(bootstrapCtx, pool, hasher, cfg.AdminEmail, cfg.AdminPassword); err != nil {
		bootstrapCancel()
		appLog.Error("admin bootstrap failed", "error", err)
		os.Exit(1)
	}
	bootstrapCancel()

	var cache *rediscache.RedisCache
	redisCandidate := rediscache.NewRedisCache(cfg.RedisAddr)
	if err := redisCandidate.Ping(context.Background()); err != nil {
		appLog.Warn("redis unavailable; cache disabled", "error", err)
		_ = redisCandidate.Close()
	} else {
		cache = redisCandidate
		defer cache.Close()
		appLog.Info("redis connected")
	}

	tokenMgr := jwt.NewManager(cfg.JWTSecret)
	txManager := pgrepo.NewTransactionManager(pool)

	userRepo := pgrepo.NewUserRepository(pool)
	appRepo := pgrepo.NewApplicationRepository(pool)
	trainerRepo := pgrepo.NewTrainerRepository(pool)
	trainingRepo := pgrepo.NewTrainingRepository(pool)
	trainingReqRepo := pgrepo.NewTrainingRequestRepository(pool)
	productRepo := pgrepo.NewProductRepository(pool)
	subscriptionRepo := pgrepo.NewSubscriptionRepository(pool)
	paymentRepo := pgrepo.NewPaymentRepository(pool)
	orderRepo := pgrepo.NewOrderRepository(pool)
	aiRepo := pgrepo.NewAIRepository(pool)
	crmRepo := pgrepo.NewCRMRepository(pool)
	accountRepo := pgrepo.NewAccountAccessRepository(pool)

	authSvc := service.NewAuthService(userRepo, appRepo, tokenMgr, hasher, cfg.JWTTokenTTL)
	userSvc := service.NewUserService(userRepo, appRepo, txManager, hasher, cache)
	var mailSender mailer.Sender = &mailer.LogSender{Log: appLog}
	if cfg.MailMode == "smtp" {
		smtpSender, err := mailer.NewSMTPSender(mailer.SMTPConfig{
			Host: cfg.SMTPHost, Port: cfg.SMTPPort, Username: cfg.SMTPUsername, Password: cfg.SMTPPassword,
			From: cfg.SMTPFrom, FromName: cfg.SMTPFromName, Timeout: 10 * time.Second,
		})
		if err != nil {
			appLog.Error("failed to configure SMTP mailer", "error", err)
			os.Exit(1)
		}
		mailSender = smtpSender
	} else {
		appLog.Warn("MAIL_MODE=log: activation/reset links are written to server logs and are not emailed")
	}
	accountSvc := service.NewAccountAccessService(userRepo, appRepo, accountRepo, txManager, hasher, mailSender,
		cfg.AppBaseURL, cfg.AccountActivationTTL, cfg.PasswordResetTTL, appLog)
	trainerSvc := service.NewTrainerService(trainerRepo, userRepo, cache)
	scheduleSvc := service.NewScheduleService(trainingRepo, trainingReqRepo, userRepo, trainerRepo, subscriptionRepo, txManager)
	financeSvc := service.NewFinanceService(paymentRepo, userRepo, txManager)
	shopSvc := service.NewShopService(productRepo, subscriptionRepo, paymentRepo, orderRepo, userRepo, txManager, cache)
	crmSvc := service.NewCRMService(crmRepo, userRepo, subscriptionRepo, txManager)
	aiClient := openaiapi.NewClient(cfg.OpenAIAPIKey, cfg.OpenAIBaseURL, cfg.AIChatModel, cfg.AIEmbeddingModel, 45*time.Second)
	aiSvc := service.NewAIAssistantService(aiRepo, txManager, aiClient, cfg.AIKnowledgeDir)
	if aiSvc.Enabled() {
		indexCtx, indexCancel := context.WithTimeout(context.Background(), 90*time.Second)
		indexed, indexErr := aiSvc.IndexKnowledge(indexCtx)
		indexCancel()
		if indexErr != nil {
			appLog.Warn("AI knowledge indexing failed; chat may have incomplete RAG context", "error", indexErr)
		} else {
			appLog.Info("AI knowledge ready", "documents_indexed", indexed, "chat_model", cfg.AIChatModel, "embedding_model", cfg.AIEmbeddingModel)
		}
	} else {
		appLog.Warn("OPENAI_API_KEY is not configured; AI chat is disabled")
	}

	handlers := &httphandler.Handlers{
		Auth:     httphandler.NewAuthHandler(authSvc),
		Account:  httphandler.NewAccountHandler(accountSvc),
		User:     httphandler.NewUserHandler(userSvc),
		Schedule: httphandler.NewScheduleHandler(scheduleSvc),
		Finance:  httphandler.NewFinanceHandler(financeSvc),
		Shop:     httphandler.NewShopHandler(shopSvc),
		Trainer:  httphandler.NewTrainerHandler(trainerSvc),
		AI:       httphandler.NewAIHandler(aiSvc),
		CRM:      httphandler.NewCRMHandler(crmSvc, userSvc, financeSvc, shopSvc),
	}

	router := httphandler.NewRouter(handlers, tokenMgr, userRepo, crmRepo, pool.Ping)
	var handler http.Handler = router
	handler = middleware.SecurityHeaders(handler)
	handler = middleware.CORS(cfg.CORSOrigins)(handler)
	handler = middleware.Metrics(handler)
	handler = middleware.Logging(appLog)(handler)
	handler = middleware.RequestID(handler)

	srv := &http.Server{
		Addr:              fmt.Sprintf(":%s", cfg.HTTPPort),
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      120 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}

	serverErr := make(chan error, 1)
	go func() {
		appLog.Info("server starting", "port", cfg.HTTPPort, "env", cfg.Env)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
		close(serverErr)
	}()

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(signals)

	select {
	case sig := <-signals:
		appLog.Info("shutdown signal received", "signal", sig.String())
	case err := <-serverErr:
		if err != nil {
			appLog.Error("server stopped unexpectedly", "error", err)
			return
		}
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		appLog.Error("graceful shutdown failed", "error", err)
	}
	appLog.Info("server stopped")
}

func applyMigrations(databaseURL string) error {
	m, err := migrate.New("file://migrations", databaseURL)
	if err != nil {
		return err
	}
	defer func() {
		_, _ = m.Close()
	}()
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}
	return nil
}

func bootstrapAdmin(ctx context.Context, pool *pgxpool.Pool, hasher *hash.Bcrypt, email, password string) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(context.Background())
	// Initial creation is serialized across replicas.
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(731928461)`); err != nil {
		return err
	}
	if err := bootstrapAdminRecord(ctx, tx, hasher, email, password); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

type bootstrapDB interface {
	QueryRow(context.Context, string, ...any) pgx.Row
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

func bootstrapAdminRecord(ctx context.Context, db bootstrapDB, hasher *hash.Bcrypt, email, password string) error {
	var id int64
	var currentHash, currentRole string
	err := db.QueryRow(ctx, `SELECT id,password_hash,role FROM users WHERE lower(email)=lower($1) FOR UPDATE`, email).
		Scan(&id, &currentHash, &currentRole)
	if err == nil {
		if currentRole != "admin" {
			return fmt.Errorf("ADMIN_EMAIL belongs to a non-admin account")
		}
		if currentHash != bootstrapPasswordMarker {
			return nil
		}
	} else if errors.Is(err, pgx.ErrNoRows) {
		// Reuse the migration placeholder; its reserved phone is already unique.
		err = db.QueryRow(ctx, `SELECT id FROM users WHERE role='admin' AND password_hash=$1 ORDER BY id LIMIT 1 FOR UPDATE`, bootstrapPasswordMarker).Scan(&id)
		if errors.Is(err, pgx.ErrNoRows) {
			var exists bool
			if err := db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE role='admin')`).Scan(&exists); err != nil {
				return err
			}
			// Bootstrap settings only apply once. An administrator may have changed their email.
			if exists {
				return nil
			}
		} else if err != nil {
			return fmt.Errorf("read bootstrap placeholder: %w", err)
		}
	} else {
		return fmt.Errorf("read bootstrap admin: %w", err)
	}
	passwordHash, err := hasher.Hash(password)
	if err != nil {
		return fmt.Errorf("hash admin password: %w", err)
	}
	if id != 0 {
		_, err = db.Exec(ctx, `UPDATE users SET email=lower($1),password_hash=$2,is_active=true,token_version=token_version+1,updated_at=NOW() WHERE id=$3`, email, passwordHash, id)
	} else {
		_, err = db.Exec(ctx, `INSERT INTO users(full_name,phone,email,password_hash,role,gender,is_active)
			VALUES('Администратор SFEDU','+70000000000',lower($1),$2,'admin','male',true)`, email, passwordHash)
	}
	if err != nil {
		return fmt.Errorf("initialize bootstrap admin: %w", err)
	}
	return nil
}

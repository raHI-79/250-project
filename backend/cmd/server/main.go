package main

import (
	"context"
	"errors"
	"github.com/raHI-79/250-project/backend/internal/auth"
	"github.com/raHI-79/250-project/backend/internal/config"
	"github.com/raHI-79/250-project/backend/internal/db"
	"github.com/raHI-79/250-project/backend/internal/filestore"
	"github.com/raHI-79/250-project/backend/internal/handler"
	"github.com/raHI-79/250-project/backend/internal/jobs"
	"github.com/raHI-79/250-project/backend/internal/mailer"
	"github.com/raHI-79/250-project/backend/internal/notify"
	"github.com/raHI-79/250-project/backend/internal/push"
	"github.com/raHI-79/250-project/backend/internal/server"
	"github.com/raHI-79/250-project/backend/internal/store"
	"github.com/raHI-79/250-project/backend/internal/upload"
	"log"
	"net"
	"net/http"
	"os/signal"
	"syscall"
	"time"
)

const (
	readTimeout     = 10 * time.Second
	writeTimeout    = 10 * time.Second
	idleTimeout     = 60 * time.Second
	shutdownTimeout = 10 * time.Second
	startupTimeout  = 30 * time.Second
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("server: %v", err)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	notifyContext, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	startupContext, cancelStartup := context.WithTimeout(notifyContext, startupTimeout)
	defer cancelStartup()
	pool, err := db.Connect(startupContext, cfg.DatabaseURL, cfg.DBMaxConns, db.ConnectWindow)
	if err != nil {
		return err
	}
	defer pool.Close()
	log.Println("db: connected")
	if err := db.Migrate(startupContext, pool); err != nil {
		return err
	}
	tokens, err := auth.NewTokens(cfg.JWTSecret, cfg.JWTTTL)
	if err != nil {
		return err
	}
	proofs, err := auth.NewProofs(cfg.JWTSecret, auth.ProofTTL)
	if err != nil {
		return err
	}
	userStore := store.New(pool)
	fileStore := upload.Store(upload.NewDisk(cfg.UploadDir))
	if cfg.FilesInDatabase() {
		fileStore = filestore.New(pool)
		log.Println("upload: files are kept in the database")
	}
	alerts := notify.New(userStore)
	var pushWorker *push.Worker
	var pushTests handler.PushTester
	if cfg.PushEnabled() {
		sender := push.NewSender(cfg.VAPIDPublicKey, cfg.VAPIDPrivateKey, cfg.VAPIDSubject)
		pushWorker = push.NewWorker(userStore, sender, push.QueueSize)
		pushTests = pushWorker
		alerts = alerts.WithPush(pushWorker, cfg.FrontendOrigin)
		log.Println("push: web push notifications are on")
	} else {
		log.Println("push: VAPID keys are missing, web push notifications stay off")
	}
	if cfg.HasAdminSeed() {
		if err := seedAdmin(startupContext, userStore, cfg); err != nil {
			return err
		}
	}
	cancelStartup()
	httpServer := &http.Server{
		Addr: net.JoinHostPort("", cfg.Port),
		Handler: server.NewRouter(server.Dependencies{
			Config:    cfg,
			Database:  pool,
			Tokens:    tokens,
			Proofs:    proofs,
			Photos:    upload.NewPhotosIn(fileStore),
			Accounts:  userStore,
			Users:     userStore,
			Auth:      userStore,
			Numbers:   userStore,
			Devices:   userStore,
			PushTests: pushTests,
			People:    userStore,
			Catalog:   userStore,
			Lookups:   userStore,
			Periods:   userStore,
			Alerts:    alerts,
			Classes:   userStore,
			Items:     userStore,
			Notifier:  alerts,
			Teachers:  userStore,
			Students:  userStore,
			Alarms:    alerts,
			Notices:   userStore,
			Drives:    userStore,
			Contacts:  userStore,
			Chats:     userStore,
			Shared:    upload.NewAttachmentsIn(fileStore, cfg.MaxUploadMB),
			Votes:     userStore,
			Badges:    userStore,
			Files:     upload.NewSaverIn(fileStore, cfg.MaxUploadMB),
			Mailer: mailer.New(cfg.MailProvider, mailer.Options{
				Host:     cfg.SMTPHost,
				Port:     cfg.SMTPPort,
				Username: cfg.SMTPUsername,
				Password: cfg.SMTPPassword,
				From:     cfg.MailFrom,
				FromName: cfg.MailFromName,
				APIKey:   cfg.BrevoAPIKey,
				Endpoint: cfg.AppsScriptURL,
				Secret:   cfg.AppsScriptSecret,
			}),
		}),
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
		IdleTimeout:  idleTimeout,
	}
	jobContext, stopJobs := context.WithCancel(context.Background())
	defer stopJobs()
	go jobs.NewScheduler(cfg, userStore, alerts).Run(jobContext, jobs.Interval)
	if pushWorker != nil {
		pushWorker.Run(jobContext, push.Workers)
	}
	serverErrors := make(chan error, 1)
	go func() {
		log.Printf("server: listening on http://localhost:%s (env=%s, timezone=%s)", cfg.Port, cfg.AppEnv, cfg.Timezone)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErrors <- err
			return
		}
		serverErrors <- nil
	}()
	select {
	case err := <-serverErrors:
		return err
	case <-notifyContext.Done():
		log.Println("server: shutdown signal received, finishing active requests")
		stop()
		stopJobs()
		shutdownContext, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		if err := httpServer.Shutdown(shutdownContext); err != nil {
			return err
		}
		log.Println("server: stopped cleanly")
		return nil
	}
}

func seedAdmin(ctx context.Context, userStore *store.Store, cfg *config.Config) error {
	hash, err := auth.HashPassword(cfg.AdminPassword)
	if err != nil {
		return err
	}
	if _, err := userStore.EnsureAdmin(ctx, cfg.AdminIdentifier, hash, cfg.AdminEmail, cfg.AdminPhone); err != nil {
		return err
	}
	log.Println("auth: admin account ready")
	return nil
}

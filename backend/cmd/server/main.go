package main

import (
	"context"
	"errors"
	"github.com/raHI-79/250-project/backend/internal/auth"
	"github.com/raHI-79/250-project/backend/internal/config"
	"github.com/raHI-79/250-project/backend/internal/db"
	"github.com/raHI-79/250-project/backend/internal/server"
	"github.com/raHI-79/250-project/backend/internal/store"
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
	pool, err := db.Open(startupContext, cfg.DatabaseURL)
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
	userStore := store.New(pool)
	if cfg.HasAdminSeed() {
		if err := seedAdmin(startupContext, userStore, cfg); err != nil {
			return err
		}
	}
	cancelStartup()
	httpServer := &http.Server{
		Addr: net.JoinHostPort("", cfg.Port),
		Handler: server.NewRouter(server.Dependencies{
			Config:   cfg,
			Database: pool,
			Tokens:   tokens,
			Users:    userStore,
			Auth:     userStore,
		}),
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
		IdleTimeout:  idleTimeout,
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
	if _, err := userStore.EnsureAdmin(ctx, cfg.AdminIdentifier, hash); err != nil {
		return err
	}
	log.Println("auth: admin account ready")
	return nil
}

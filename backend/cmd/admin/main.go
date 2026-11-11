package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"github.com/raHI-79/250-project/backend/internal/auth"
	"github.com/raHI-79/250-project/backend/internal/config"
	"github.com/raHI-79/250-project/backend/internal/db"
	"github.com/raHI-79/250-project/backend/internal/store"
	"github.com/raHI-79/250-project/backend/internal/validate"
	"log"
	"os"
	"time"
)

const (
	passwordEnv = "NEW_SUPER_ADMIN_PASSWORD"
	timeout     = 30 * time.Second
)

var errNoSuperAdmin = errors.New("admin: no super admin account exists yet")

func main() {
	log.SetFlags(0)
	reset := flag.Bool("reset-super", false, "reset the super admin password from "+passwordEnv)
	flag.Parse()
	if !*reset {
		log.Fatalf("admin: nothing to do, pass -reset-super")
	}
	if err := resetSuper(); err != nil {
		log.Fatalf("admin: %v", err)
	}
	fmt.Println("Password updated")
}

func resetSuper() error {
	password := os.Getenv(passwordEnv)
	if err := validate.OfficePassword("The new password", password, password); err != nil {
		return errors.New(err.Error() + " Set " + passwordEnv + " in the environment.")
	}
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	pool, err := db.Connect(ctx, cfg.DatabaseURL, cfg.DBMaxConns, cfg.Timezone, db.ConnectWindow)
	if err != nil {
		return err
	}
	defer pool.Close()
	if err := db.Migrate(ctx, pool); err != nil {
		return err
	}
	hash, err := auth.HashPassword(password)
	if err != nil {
		return err
	}
	people := store.New(pool)
	identifier, err := people.ResetSuperAdminPassword(ctx, hash)
	if errors.Is(err, store.ErrNotFound) {
		return errNoSuperAdmin
	}
	if err != nil {
		return err
	}
	event := store.AuditEvent{
		ActorRole:  store.RoleSuperAdmin,
		ActorLabel: identifier,
		Action:     store.ActionPasswordReset,
		TargetType: "account",
		Summary:    "The super admin password was reset from the server",
		Client:     store.ClientWeb,
	}
	if err := people.WriteEvent(ctx, event); err != nil {
		return err
	}
	return nil
}

package dbtest

import (
	"context"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/raHI-79/250-project/backend/internal/config"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const (
	DatabaseSuffix = "_test"
	Timeout        = 60 * time.Second
	lockKey        = 250250250
)

func LoadEnv() error {
	return config.LoadEnvFile(filepath.Join("..", "..", ".env"))
}

func Pool(t *testing.T) (context.Context, *pgxpool.Pool) {
	t.Helper()
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set, skipping the database integration tests")
	}
	poolConfig, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		t.Fatalf("TEST_DATABASE_URL could not be parsed: %v", err)
	}
	name := poolConfig.ConnConfig.Database
	if !strings.HasSuffix(name, DatabaseSuffix) {
		t.Fatalf("refusing to run against database %q because its name does not end with %q", name, DatabaseSuffix)
	}
	ctx, cancel := context.WithTimeout(context.Background(), Timeout)
	t.Cleanup(cancel)
	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		t.Fatalf("could not create the test connection pool: %v", err)
	}
	t.Cleanup(pool.Close)
	lockDatabase(t, ctx, pool)
	if _, err := pool.Exec(ctx, "DROP SCHEMA public CASCADE; CREATE SCHEMA public"); err != nil {
		t.Fatalf("could not reset the public schema of %q: %v", name, err)
	}
	return ctx, pool
}

func lockDatabase(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()
	conn, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatalf("could not acquire a connection for the test lock: %v", err)
	}
	if _, err := conn.Exec(ctx, "SELECT pg_advisory_lock($1)", lockKey); err != nil {
		conn.Release()
		t.Fatalf("could not take the test database lock: %v", err)
	}
	t.Cleanup(func() {
		unlockContext, cancel := context.WithTimeout(context.Background(), Timeout)
		defer cancel()
		if _, err := conn.Exec(unlockContext, "SELECT pg_advisory_unlock($1)", lockKey); err != nil {
			t.Errorf("could not release the test database lock: %v", err)
		}
		conn.Release()
	})
}

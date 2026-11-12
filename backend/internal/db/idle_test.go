package db

import (
	"context"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/raHI-79/250-project/backend/internal/dbtest"
	"os"
	"testing"
	"time"
)

func afterItsConnectionDied(t *testing.T, ctx context.Context, other *pgxpool.Pool, databaseURL string,
	idleFor time.Duration) error {
	t.Helper()
	pool, err := OpenGuarded(ctx, databaseURL, 1, "Asia/Dhaka", idleFor)
	if err != nil {
		t.Fatalf("OpenGuarded() returned an unexpected error: %v", err)
	}
	defer pool.Close()
	var pid int
	if err := pool.QueryRow(ctx, "SELECT pg_backend_pid()").Scan(&pid); err != nil {
		t.Fatalf("could not read the backend of the pooled connection: %v", err)
	}
	if _, err := other.Exec(ctx, "SELECT pg_terminate_backend($1)", pid); err != nil {
		t.Fatalf("could not end the pooled connection: %v", err)
	}
	time.Sleep(200 * time.Millisecond)
	var one int
	return pool.QueryRow(ctx, "SELECT 1").Scan(&one)
}

func TestAConnectionThatDiedWhileIdleIsReplacedBeforeTheNextRequest(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set, skipping the database integration tests")
	}
	ctx, other := dbtest.Pool(t)
	if err := afterItsConnectionDied(t, ctx, other, databaseURL, -1); err == nil {
		t.Fatal("without the guard the dead connection still answered, so the test proves nothing")
	}
	if err := afterItsConnectionDied(t, ctx, other, databaseURL, 0); err != nil {
		t.Fatalf("the request after an idle spell failed: %v", err)
	}
}

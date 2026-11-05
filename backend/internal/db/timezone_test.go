package db

import (
	"os"
	"testing"

	"github.com/raHI-79/250-project/backend/internal/dbtest"
)

func TestOpenPinsTheSessionToTheAppTimezone(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set, skipping the database integration tests")
	}
	ctx, _ := dbtest.Pool(t)
	pool, err := Open(ctx, databaseURL, 2, "Asia/Dhaka")
	if err != nil {
		t.Fatalf("Open() returned an unexpected error: %v", err)
	}
	defer pool.Close()
	var zone string
	if err := pool.QueryRow(ctx, "SHOW TimeZone").Scan(&zone); err != nil {
		t.Fatalf("could not read the session timezone: %v", err)
	}
	if zone != "Asia/Dhaka" {
		t.Errorf("the session timezone = %q, want Asia/Dhaka", zone)
	}
}

func TestSessionZoneMakesSqlDatesFollowDhaka(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set, skipping the database integration tests")
	}
	ctx, _ := dbtest.Pool(t)
	pool, err := Open(ctx, databaseURL, 2, "Asia/Dhaka")
	if err != nil {
		t.Fatalf("Open() returned an unexpected error: %v", err)
	}
	defer pool.Close()
	// 2026-11-05 18:03 UTC is already 2026-11-06 in Dhaka. Every date the server
	// derives in SQL has to agree with that, or the day flips six hours early.
	var date, truncated, cast string
	query := `SELECT to_char(TIMESTAMPTZ '2026-11-05 18:03:00+00'::date, 'YYYY-MM-DD'),
to_char(date_trunc('day', TIMESTAMPTZ '2026-11-05 18:03:00+00'), 'YYYY-MM-DD'),
to_char(DATE '2026-11-06'::timestamptz, 'YYYY-MM-DD HH24:MI')`
	if err := pool.QueryRow(ctx, query).Scan(&date, &truncated, &cast); err != nil {
		t.Fatalf("could not read the derived dates: %v", err)
	}
	if date != "2026-11-06" {
		t.Errorf("the SQL date = %q, want the Dhaka date 2026-11-06", date)
	}
	if truncated != "2026-11-06" {
		t.Errorf("date_trunc landed on %q, want 2026-11-06", truncated)
	}
	if cast != "2026-11-06 00:00" {
		t.Errorf("a date cast to a stamp = %q, want the Dhaka midnight 2026-11-06 00:00", cast)
	}
}

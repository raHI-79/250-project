package main

import (
	"context"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/raHI-79/250-project/backend/internal/db"
	"github.com/raHI-79/250-project/backend/internal/dbtest"
	"log"
	"os"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	if err := dbtest.LoadEnv(); err != nil {
		log.Fatalf("seed: could not load backend/.env: %v", err)
	}
	os.Exit(m.Run())
}

func newLoader(t *testing.T, ctx context.Context, pool *pgxpool.Pool) *loader {
	t.Helper()
	location, err := time.LoadLocation("Asia/Dhaka")
	if err != nil {
		t.Fatalf("could not load the timezone: %v", err)
	}
	return &loader{
		ctx:  ctx,
		pool: pool,
		hash: "not-a-real-hash",
		now:  time.Date(2026, 11, 2, 9, 0, 0, 0, location),
	}
}

func migratedPool(t *testing.T) (context.Context, *pgxpool.Pool) {
	t.Helper()
	ctx, pool := dbtest.Pool(t)
	if err := db.Migrate(ctx, pool); err != nil {
		t.Fatalf("db.Migrate() returned an unexpected error: %v", err)
	}
	return ctx, pool
}

func uniqueSemesters() map[string]bool {
	seen := make(map[string]bool)
	for index := range demoBatches {
		seen[demoRunning[index]] = true
		seen[demoOlder[index]] = true
	}
	return seen
}

func count(t *testing.T, ctx context.Context, pool *pgxpool.Pool, query string) int {
	t.Helper()
	var total int
	if err := pool.QueryRow(ctx, query).Scan(&total); err != nil {
		t.Fatalf("could not run %q: %v", query, err)
	}
	return total
}

func TestDemoDataLoadsAndRunsTwice(t *testing.T) {
	ctx, pool := migratedPool(t)
	if err := newLoader(t, ctx, pool).load(); err != nil {
		t.Fatalf("the first run returned an unexpected error: %v", err)
	}
	counts := []struct {
		name  string
		query string
		want  int
	}{
		{name: "courses", query: "SELECT count(*) FROM courses", want: len(demoCourses)},
		{name: "teachers", query: "SELECT count(*) FROM teachers", want: len(demoTeachers)},
		{name: "students", query: "SELECT count(*) FROM students", want: len(demoBatches) * 2 * 4},
		{name: "class representatives", query: "SELECT count(*) FROM students WHERE is_cr", want: len(demoBatches) * 2},
		{name: "batches", query: "SELECT count(DISTINCT batch) FROM students", want: len(demoBatches)},
		{name: "sections", query: "SELECT count(DISTINCT section) FROM students", want: 2},
		{name: "polls", query: "SELECT count(*) FROM polls", want: 3},
		{name: "running semesters", query: "SELECT count(*) FROM batch_semesters", want: len(demoBatches)},
		{name: "semesters in the routine", query: "SELECT count(DISTINCT semester) FROM routines",
			want: len(uniqueSemesters())},
		{name: "publications", query: "SELECT count(*) FROM publications", want: 1},
	}
	for _, check := range counts {
		if got := count(t, ctx, pool, check.query); got != check.want {
			t.Errorf("%s = %d, want %d", check.name, got, check.want)
		}
	}
	for _, query := range []string{
		"SELECT count(*) FROM routines",
		"SELECT count(*) FROM schedule_items",
		"SELECT count(*) FROM notices",
		"SELECT count(*) FROM published_items",
		"SELECT count(*) FROM conversations WHERE kind = 'group'",
		"SELECT count(*) FROM conversations WHERE kind = 'direct'",
		"SELECT count(*) FROM messages",
		"SELECT count(*) FROM poll_votes",
	} {
		if got := count(t, ctx, pool, query); got == 0 {
			t.Errorf("%q returned nothing, want demo rows", query)
		}
	}
	states := count(t, ctx, pool,
		"SELECT count(DISTINCT status) FROM schedule_items WHERE origin = 'cr'")
	if states != 9 {
		t.Errorf("schedule item states = %d, want all nine", states)
	}
	sections := count(t, ctx, pool,
		"SELECT count(*) FROM students WHERE section NOT IN ('A', 'B')")
	if sections != 0 {
		t.Errorf("%d students sit outside section A and B", sections)
	}
	for index, batch := range demoBatches {
		var running string
		err := pool.QueryRow(ctx,
			"SELECT running_semester FROM batch_semesters WHERE batch = $1", batch).Scan(&running)
		if err != nil {
			t.Fatalf("could not read the running semester of %s: %v", batch, err)
		}
		if running != demoRunning[index] {
			t.Errorf("%s runs %q, want %q", batch, running, demoRunning[index])
		}
		older := count(t, ctx, pool,
			"SELECT count(*) FROM routines WHERE batch = '"+batch+"' AND semester = '"+demoOlder[index]+"'")
		if older == 0 {
			t.Errorf("%s has no routine entry in the older semester %s", batch, demoOlder[index])
		}
	}
	idle := count(t, ctx, pool, `SELECT count(*) FROM schedule_items i
JOIN routines r ON r.id = i.routine_id
JOIN batch_semesters bs ON bs.batch = r.batch
WHERE r.semester <> bs.running_semester`)
	if idle != 0 {
		t.Errorf("%d dated items came from a semester that is not running", idle)
	}
	before := map[string]int{}
	for _, table := range []string{"users", "students", "teachers", "courses", "routines",
		"schedule_items", "notices", "polls", "poll_options", "poll_votes", "conversations",
		"conversation_members", "messages", "publications", "published_items"} {
		before[table] = count(t, ctx, pool, "SELECT count(*) FROM "+table)
	}
	if err := newLoader(t, ctx, pool).load(); err != nil {
		t.Fatalf("the second run returned an unexpected error: %v", err)
	}
	for table, want := range before {
		if got := count(t, ctx, pool, "SELECT count(*) FROM "+table); got != want {
			t.Errorf("%s = %d after the second run, want %d", table, got, want)
		}
	}
}

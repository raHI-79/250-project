package store

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5/pgxpool"
	"testing"
)

type batchFixture struct {
	userID   int64
	courseID int64
}

func newBatchFixture(t *testing.T, ctx context.Context, store *Store, pool *pgxpool.Pool) batchFixture {
	t.Helper()
	if _, err := store.CreateBatch(ctx, "CSE-23", "3-1", 0); err != nil {
		t.Fatalf("CreateBatch() returned an unexpected error: %v", err)
	}
	userID := insertUser(t, ctx, pool, RoleStudent, "2023331079", StatusActive, 0)
	insertStudentRow(t, ctx, pool, userID, "2023331079", "CSE-23", "A")
	courseID := insertCourseRow(t, ctx, pool, "CSE245", "Databases")
	teacherUser := insertUser(t, ctx, pool, RoleTeacher, "T-500", StatusActive, 0)
	teacherID := insertTeacherRow(t, ctx, pool, teacherUser, "T-500", "Main Teacher")
	insertRoutineRow(t, ctx, pool, courseID, teacherID, "CSE-23", "A", "3-1")
	statements := []string{
		`INSERT INTO user_sessions (user_id, expires_at) VALUES ($1, now() + interval '1 day')`,
		`INSERT INTO fcm_tokens (user_id, token, platform) VALUES ($1, 'fcm-token', 'android')`,
		`INSERT INTO push_subscriptions (user_id, endpoint, p256dh, auth)
VALUES ($1, 'https://push.example/one', 'key', 'auth')`,
	}
	for _, statement := range statements {
		if _, err := pool.Exec(ctx, statement, userID); err != nil {
			t.Fatalf("could not prepare the signed-in student: %v", err)
		}
	}
	return batchFixture{userID: userID, courseID: courseID}
}

func countRows(t *testing.T, ctx context.Context, pool *pgxpool.Pool, query string, args ...any) int {
	t.Helper()
	var total int
	if err := pool.QueryRow(ctx, query, args...).Scan(&total); err != nil {
		t.Fatalf("could not count with %q: %v", query, err)
	}
	return total
}

func TestDeactivatedBatchIsBlockedEverywhere(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	fixture := newBatchFixture(t, ctx, store, pool)
	before, err := store.FindLoginUser(ctx, "2023331079")
	if err != nil {
		t.Fatalf("FindLoginUser() returned an unexpected error: %v", err)
	}
	if before.BatchClosed {
		t.Fatal("an active batch was reported as closed")
	}
	students, err := store.SetBatchStatus(ctx, "CSE-23", BatchInactive, 0)
	if err != nil {
		t.Fatalf("SetBatchStatus() returned an unexpected error: %v", err)
	}
	if students != 1 {
		t.Errorf("students = %d, want 1", students)
	}
	after, err := store.FindLoginUser(ctx, "2023331079")
	if err != nil {
		t.Fatalf("FindLoginUser() returned an unexpected error: %v", err)
	}
	if !after.BatchClosed {
		t.Error("the student of an inactive batch is not blocked")
	}
	if after.TokenVersion != before.TokenVersion+1 {
		t.Errorf("token version = %d, want %d", after.TokenVersion, before.TokenVersion+1)
	}
	open := countRows(t, ctx, pool,
		"SELECT count(*) FROM user_sessions WHERE user_id = $1 AND revoked_at IS NULL", fixture.userID)
	fcm := countRows(t, ctx, pool, "SELECT count(*) FROM fcm_tokens WHERE user_id = $1", fixture.userID)
	push := countRows(t, ctx, pool,
		"SELECT count(*) FROM push_subscriptions WHERE user_id = $1", fixture.userID)
	if open != 0 || fcm != 0 || push != 0 {
		t.Errorf("open sessions = %d, fcm tokens = %d, push subscriptions = %d, want none", open, fcm, push)
	}
	batches, err := store.ListBatches(ctx)
	if err != nil {
		t.Fatalf("ListBatches() returned an unexpected error: %v", err)
	}
	if len(batches) != 0 {
		t.Errorf("batches = %v, want the inactive batch hidden", batches)
	}
	if open, err := store.BatchOpen(ctx, "CSE-23"); err != nil || open {
		t.Errorf("BatchOpen() = %v, %v, want false", open, err)
	}
	generated, err := store.GenerateFixedItems(ctx, "2026-11-15", "2026-11-21", nil)
	if err != nil {
		t.Fatalf("GenerateFixedItems() returned an unexpected error: %v", err)
	}
	if generated != 0 {
		t.Errorf("generated = %d, want nothing for an inactive batch", generated)
	}
	if kept := countRows(t, ctx, pool, "SELECT count(*) FROM routines WHERE batch = 'CSE-23'"); kept != 1 {
		t.Errorf("routines = %d, want the record kept", kept)
	}
	if _, err := store.SetBatchStatus(ctx, "CSE-23", BatchInactive, 0); !errors.Is(err, ErrBatchUnchanged) {
		t.Errorf("second deactivation error = %v, want ErrBatchUnchanged", err)
	}
}

func TestActivatedBatchComesBack(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	newBatchFixture(t, ctx, store, pool)
	if _, err := store.SetBatchStatus(ctx, "CSE-23", BatchInactive, 0); err != nil {
		t.Fatalf("SetBatchStatus() returned an unexpected error: %v", err)
	}
	if _, err := store.SetBatchStatus(ctx, "CSE-23", BatchActive, 0); err != nil {
		t.Fatalf("SetBatchStatus() returned an unexpected error: %v", err)
	}
	user, err := store.FindLoginUser(ctx, "2023331079")
	if err != nil {
		t.Fatalf("FindLoginUser() returned an unexpected error: %v", err)
	}
	if user.BatchClosed {
		t.Error("the student of a reactivated batch is still blocked")
	}
	batches, err := store.ListBatches(ctx)
	if err != nil {
		t.Fatalf("ListBatches() returned an unexpected error: %v", err)
	}
	if len(batches) != 1 || batches[0] != "CSE-23" {
		t.Errorf("batches = %v, want CSE-23", batches)
	}
	sections, err := store.ListSections(ctx, "CSE-23")
	if err != nil {
		t.Fatalf("ListSections() returned an unexpected error: %v", err)
	}
	if len(sections) != 2 || sections[0] != "A" || sections[1] != "B" {
		t.Errorf("sections = %v, want A and B", sections)
	}
	generated, err := store.GenerateFixedItems(ctx, "2026-11-15", "2026-11-21", nil)
	if err != nil {
		t.Fatalf("GenerateFixedItems() returned an unexpected error: %v", err)
	}
	if generated == 0 {
		t.Error("nothing was generated for the reactivated batch")
	}
}

func TestDeleteBatchRemovesItsRecords(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	fixture := newBatchFixture(t, ctx, store, pool)
	if _, err := store.GenerateFixedItems(ctx, "2026-11-15", "2026-11-21", nil); err != nil {
		t.Fatalf("GenerateFixedItems() returned an unexpected error: %v", err)
	}
	if _, err := store.DeleteBatch(ctx, "CSE-99"); !errors.Is(err, ErrNotFound) {
		t.Errorf("unknown batch error = %v, want ErrNotFound", err)
	}
	removal, err := store.DeleteBatch(ctx, "CSE-23")
	if err != nil {
		t.Fatalf("DeleteBatch() returned an unexpected error: %v", err)
	}
	if removal.Students != 1 || removal.Routines != 1 || removal.Items == 0 {
		t.Errorf("removal = %+v, want one student, one routine and its classes", removal)
	}
	for _, table := range []string{"routines", "schedule_items", "attendance_sessions"} {
		if left := countRows(t, ctx, pool, "SELECT count(*) FROM "+table+" WHERE batch = 'CSE-23'"); left != 0 {
			t.Errorf("%s rows left = %d, want 0", table, left)
		}
	}
	if left := countRows(t, ctx, pool, "SELECT count(*) FROM batches WHERE name = 'CSE-23'"); left != 0 {
		t.Errorf("batches left = %d, want 0", left)
	}
	var status string
	if err := pool.QueryRow(ctx, "SELECT status FROM users WHERE id = $1", fixture.userID).Scan(&status); err != nil {
		t.Fatalf("could not read the student account: %v", err)
	}
	if status != StatusRemoved {
		t.Errorf("student status = %q, want removed", status)
	}
	user, err := store.FindLoginUser(ctx, "2023331079")
	if err != nil {
		t.Fatalf("FindLoginUser() returned an unexpected error: %v", err)
	}
	if !user.BatchClosed {
		t.Error("the student of a deleted batch is not blocked")
	}
}

func TestBatchRowsCountStudentsAndRefuseDuplicates(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	newBatchFixture(t, ctx, store, pool)
	if _, err := store.CreateBatch(ctx, "CSE-23", "1-1", 0); !errors.Is(err, ErrDuplicateBatch) {
		t.Errorf("duplicate error = %v, want ErrDuplicateBatch", err)
	}
	if err := store.SetBatchSemester(ctx, "CSE-23", "3-2", 0); err != nil {
		t.Fatalf("SetBatchSemester() returned an unexpected error: %v", err)
	}
	rows, err := store.ListBatchRows(ctx)
	if err != nil {
		t.Fatalf("ListBatchRows() returned an unexpected error: %v", err)
	}
	if len(rows) != 1 || rows[0].Students != 1 || rows[0].RunningSemester != "3-2" ||
		rows[0].Status != BatchActive {
		t.Errorf("rows = %+v, want CSE-23 active in 3-2 with one student", rows)
	}
}

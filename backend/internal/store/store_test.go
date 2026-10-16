package store

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/raHI-79/250-project/backend/internal/db"
	"github.com/raHI-79/250-project/backend/internal/dbtest"
	"log"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	if err := dbtest.LoadEnv(); err != nil {
		log.Fatalf("store: could not load backend/.env: %v", err)
	}
	os.Exit(m.Run())
}

func newTestStore(t *testing.T) (context.Context, *Store, *pgxpool.Pool) {
	t.Helper()
	ctx, pool := dbtest.Pool(t)
	if err := db.Migrate(ctx, pool); err != nil {
		t.Fatalf("db.Migrate() returned an unexpected error: %v", err)
	}
	return ctx, New(pool), pool
}

func storedPasswordHash(t *testing.T, ctx context.Context, pool *pgxpool.Pool, identifier string) string {
	t.Helper()
	var hash string
	if err := pool.QueryRow(ctx, "SELECT password_hash FROM users WHERE identifier = $1", identifier).Scan(&hash); err != nil {
		t.Fatalf("could not read back the stored row: %v", err)
	}
	return hash
}

func insertUser(t *testing.T, ctx context.Context, pool *pgxpool.Pool, role, identifier, status string, tokenVersion int) int64 {
	t.Helper()
	var id int64
	query := `INSERT INTO users (role, identifier, password_hash, status, token_version)
VALUES ($1, $2, 'stored-hash', $3, $4) RETURNING id`
	if err := pool.QueryRow(ctx, query, role, identifier, status, tokenVersion).Scan(&id); err != nil {
		t.Fatalf("could not insert the %s user: %v", role, err)
	}
	return id
}

func insertStudent(t *testing.T, ctx context.Context, pool *pgxpool.Pool, userID int64, regNo string, isCR bool) {
	t.Helper()
	query := `INSERT INTO students (user_id, full_name, reg_no, batch, section, email, phone, blood_group, is_cr)
VALUES ($1, 'Test Student', $2, 'CSE-23', 'A', 'student@example.com', '01700000000', 'O+', $3)`
	if _, err := pool.Exec(ctx, query, userID, regNo, isCR); err != nil {
		t.Fatalf("could not insert the student row: %v", err)
	}
}

func insertTeacher(t *testing.T, ctx context.Context, pool *pgxpool.Pool, userID int64, teacherID string) {
	t.Helper()
	query := `INSERT INTO teachers (user_id, teacher_id, name, email, phone)
VALUES ($1, $2, 'Test Teacher', 'teacher@example.com', '01800000000')`
	if _, err := pool.Exec(ctx, query, userID, teacherID); err != nil {
		t.Fatalf("could not insert the teacher row: %v", err)
	}
}

func TestEnsureAdminCreatesTheAccountOnce(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	created, err := store.EnsureAdmin(ctx, "admin", "first-stored-hash")
	if err != nil {
		t.Fatalf("EnsureAdmin() returned an unexpected error: %v", err)
	}
	if !created {
		t.Fatal("EnsureAdmin() reported no creation on an empty database, want true")
	}
	firstHash := storedPasswordHash(t, ctx, pool, "admin")
	if firstHash != "first-stored-hash" {
		t.Error("the stored hash is not the one that was passed in")
	}
	for attempt := 2; attempt <= 3; attempt++ {
		created, err = store.EnsureAdmin(ctx, "admin", "second-stored-hash")
		if err != nil {
			t.Fatalf("EnsureAdmin() returned an unexpected error on attempt %d: %v", attempt, err)
		}
		if created {
			t.Errorf("EnsureAdmin() reported a creation on attempt %d, want false", attempt)
		}
		if storedPasswordHash(t, ctx, pool, "admin") != firstHash {
			t.Fatalf("attempt %d overwrote the stored password hash, want it left alone", attempt)
		}
	}
	var count int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM users WHERE identifier = $1", "admin").Scan(&count); err != nil {
		t.Fatalf("count query failed: %v", err)
	}
	if count != 1 {
		t.Errorf("found %d rows for the admin identifier, want 1", count)
	}
}

func TestEnsureAdminCreatesAnActiveAdmin(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	if _, err := store.EnsureAdmin(ctx, "admin", "stored-hash"); err != nil {
		t.Fatalf("EnsureAdmin() returned an unexpected error: %v", err)
	}
	var id int64
	if err := pool.QueryRow(ctx, "SELECT id FROM users WHERE identifier = $1", "admin").Scan(&id); err != nil {
		t.Fatalf("could not read back the admin row: %v", err)
	}
	user, err := store.GetAuthUser(ctx, id)
	if err != nil {
		t.Fatalf("GetAuthUser() returned an unexpected error: %v", err)
	}
	if user.Role != "admin" {
		t.Errorf("Role = %q, want %q", user.Role, "admin")
	}
	if user.Status != StatusActive {
		t.Errorf("Status = %q, want %q", user.Status, StatusActive)
	}
	if !user.IsActive() {
		t.Error("IsActive() = false, want true")
	}
	if user.IsCR {
		t.Error("IsCR = true for an admin, want false")
	}
	if user.TokenVersion != 0 {
		t.Errorf("TokenVersion = %d, want 0", user.TokenVersion)
	}
}

func TestEnsureAdminValidatesItsInput(t *testing.T) {
	ctx, store, _ := newTestStore(t)
	tests := []struct {
		name       string
		identifier string
		hash       string
		wantErr    error
	}{
		{name: "empty identifier", identifier: "", hash: "stored-hash", wantErr: ErrEmptyIdentifier},
		{name: "empty hash", identifier: "admin", hash: "", wantErr: ErrEmptyPasswordHash},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			created, err := store.EnsureAdmin(ctx, tc.identifier, tc.hash)
			if !errors.Is(err, tc.wantErr) {
				t.Errorf("EnsureAdmin() error = %v, want %v", err, tc.wantErr)
			}
			if created {
				t.Error("EnsureAdmin() reported a creation, want false")
			}
		})
	}
}

func TestGetAuthUser(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	plainStudentID := insertUser(t, ctx, pool, "student", "2023331001", StatusActive, 0)
	insertStudent(t, ctx, pool, plainStudentID, "2023331001", false)
	crID := insertUser(t, ctx, pool, "student", "2023331002", StatusActive, 4)
	insertStudent(t, ctx, pool, crID, "2023331002", true)
	pendingID := insertUser(t, ctx, pool, "student", "2023331003", StatusPending, 0)
	insertStudent(t, ctx, pool, pendingID, "2023331003", false)
	removedID := insertUser(t, ctx, pool, "student", "2023331004", StatusRemoved, 2)
	insertStudent(t, ctx, pool, removedID, "2023331004", true)
	teacherID := insertUser(t, ctx, pool, "teacher", "T-001", StatusActive, 1)
	insertTeacher(t, ctx, pool, teacherID, "T-001")
	adminID := insertUser(t, ctx, pool, "admin", "admin", StatusActive, 0)
	tests := []struct {
		name             string
		userID           int64
		wantIdentifier   string
		wantRole         string
		wantStatus       string
		wantTokenVersion int
		wantIsCR         bool
		wantActive       bool
	}{
		{
			name:           "student without the cr flag",
			userID:         plainStudentID,
			wantIdentifier: "2023331001",
			wantRole:       "student",
			wantStatus:     StatusActive,
			wantActive:     true,
		},
		{
			name:             "student with the cr flag",
			userID:           crID,
			wantIdentifier:   "2023331002",
			wantRole:         "student",
			wantStatus:       StatusActive,
			wantTokenVersion: 4,
			wantIsCR:         true,
			wantActive:       true,
		},
		{
			name:           "pending student",
			userID:         pendingID,
			wantIdentifier: "2023331003",
			wantRole:       "student",
			wantStatus:     StatusPending,
		},
		{
			name:             "removed class representative",
			userID:           removedID,
			wantIdentifier:   "2023331004",
			wantRole:         "student",
			wantStatus:       StatusRemoved,
			wantTokenVersion: 2,
			wantIsCR:         true,
		},
		{
			name:             "teacher has no student row",
			userID:           teacherID,
			wantIdentifier:   "T-001",
			wantRole:         "teacher",
			wantStatus:       StatusActive,
			wantTokenVersion: 1,
			wantActive:       true,
		},
		{
			name:           "admin has no student row",
			userID:         adminID,
			wantIdentifier: "admin",
			wantRole:       "admin",
			wantStatus:     StatusActive,
			wantActive:     true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			user, err := store.GetAuthUser(ctx, tc.userID)
			if err != nil {
				t.Fatalf("GetAuthUser() returned an unexpected error: %v", err)
			}
			if user.ID != tc.userID {
				t.Errorf("ID = %d, want %d", user.ID, tc.userID)
			}
			if user.Identifier != tc.wantIdentifier {
				t.Errorf("Identifier = %q, want %q", user.Identifier, tc.wantIdentifier)
			}
			if user.Role != tc.wantRole {
				t.Errorf("Role = %q, want %q", user.Role, tc.wantRole)
			}
			if user.Status != tc.wantStatus {
				t.Errorf("Status = %q, want %q", user.Status, tc.wantStatus)
			}
			if user.TokenVersion != tc.wantTokenVersion {
				t.Errorf("TokenVersion = %d, want %d", user.TokenVersion, tc.wantTokenVersion)
			}
			if user.IsCR != tc.wantIsCR {
				t.Errorf("IsCR = %t, want %t", user.IsCR, tc.wantIsCR)
			}
			if user.IsActive() != tc.wantActive {
				t.Errorf("IsActive() = %t, want %t", user.IsActive(), tc.wantActive)
			}
		})
	}
}

func TestGetAuthUserReportsMissingUsers(t *testing.T) {
	ctx, store, _ := newTestStore(t)
	tests := []struct {
		name   string
		userID int64
	}{
		{name: "unknown id", userID: 999999},
		{name: "zero id", userID: 0},
		{name: "negative id", userID: -1},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			user, err := store.GetAuthUser(ctx, tc.userID)
			if !errors.Is(err, ErrNotFound) {
				t.Fatalf("GetAuthUser() error = %v, want ErrNotFound", err)
			}
			if user != (AuthUser{}) {
				t.Error("GetAuthUser() returned a user alongside the error, want the zero value")
			}
		})
	}
}

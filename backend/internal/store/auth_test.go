package store

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5/pgxpool"
	"testing"
	"time"
)

func newRegistration(regNo, email string) Registration {
	return Registration{
		FullName:     "Ratul Islam",
		RegNo:        regNo,
		Batch:        "CSE-23",
		Section:      "A",
		Email:        email,
		Phone:        "01712345678",
		BloodGroup:   "O+",
		PasswordHash: "stored-hash",
	}
}

func countUsers(t *testing.T, ctx context.Context, pool *pgxpool.Pool) int {
	t.Helper()
	var count int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM users").Scan(&count); err != nil {
		t.Fatalf("count query failed: %v", err)
	}
	return count
}

func TestCreateRegistration(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	userID, err := store.CreateRegistration(ctx, newRegistration("2023331079", "ratul@student.sust.edu"))
	if err != nil {
		t.Fatalf("CreateRegistration() returned an unexpected error: %v", err)
	}
	if userID == 0 {
		t.Fatal("CreateRegistration() returned user id 0")
	}
	user, err := store.GetAuthUser(ctx, userID)
	if err != nil {
		t.Fatalf("GetAuthUser() returned an unexpected error: %v", err)
	}
	checks := []struct {
		name string
		got  any
		want any
	}{
		{name: "Identifier", got: user.Identifier, want: "2023331079"},
		{name: "Role", got: user.Role, want: "student"},
		{name: "Status", got: user.Status, want: StatusPending},
		{name: "TokenVersion", got: user.TokenVersion, want: 0},
		{name: "IsCR", got: user.IsCR, want: false},
		{name: "IsActive", got: user.IsActive(), want: false},
	}
	for _, check := range checks {
		if check.got != check.want {
			t.Errorf("%s = %v, want %v", check.name, check.got, check.want)
		}
	}
	var fullName, batch, section, email, phone, bloodGroup string
	var hasDropCourse bool
	var dropCourseCode *string
	query := `SELECT full_name, batch, section, email, phone, blood_group, has_drop_course, drop_course_code
FROM students WHERE user_id = $1`
	if err := pool.QueryRow(ctx, query, userID).Scan(
		&fullName, &batch, &section, &email, &phone, &bloodGroup, &hasDropCourse, &dropCourseCode,
	); err != nil {
		t.Fatalf("could not read back the student row: %v", err)
	}
	if fullName != "Ratul Islam" || batch != "CSE-23" || section != "A" {
		t.Errorf("student row = %q/%q/%q, want Ratul Islam/CSE-23/A", fullName, batch, section)
	}
	if email != "ratul@student.sust.edu" || phone != "01712345678" || bloodGroup != "O+" {
		t.Errorf("contact row = %q/%q/%q", email, phone, bloodGroup)
	}
	if hasDropCourse {
		t.Error("has_drop_course = true, want false")
	}
	if dropCourseCode != nil {
		t.Errorf("drop_course_code = %q, want NULL", *dropCourseCode)
	}
}

func TestCreateRegistrationWithADropCourse(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	registration := newRegistration("2023331080", "drop@student.sust.edu")
	registration.HasDropCourse = true
	registration.DropCourseCode = "CSE-233"
	userID, err := store.CreateRegistration(ctx, registration)
	if err != nil {
		t.Fatalf("CreateRegistration() returned an unexpected error: %v", err)
	}
	var hasDropCourse bool
	var dropCourseCode *string
	query := "SELECT has_drop_course, drop_course_code FROM students WHERE user_id = $1"
	if err := pool.QueryRow(ctx, query, userID).Scan(&hasDropCourse, &dropCourseCode); err != nil {
		t.Fatalf("could not read back the student row: %v", err)
	}
	if !hasDropCourse {
		t.Error("has_drop_course = false, want true")
	}
	if dropCourseCode == nil || *dropCourseCode != "CSE-233" {
		t.Errorf("drop_course_code = %v, want CSE-233", dropCourseCode)
	}
}

func TestCreateRegistrationRejectsDuplicates(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	if _, err := store.CreateRegistration(ctx, newRegistration("2023331079", "ratul@student.sust.edu")); err != nil {
		t.Fatalf("the first registration failed: %v", err)
	}
	tests := []struct {
		name         string
		registration Registration
	}{
		{
			name:         "same registration number",
			registration: newRegistration("2023331079", "other@student.sust.edu"),
		},
		{
			name:         "same email",
			registration: newRegistration("2023331081", "ratul@student.sust.edu"),
		},
		{
			name:         "same email in a different case",
			registration: newRegistration("2023331082", "Ratul@Student.SUST.edu"),
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			before := countUsers(t, ctx, pool)
			userID, err := store.CreateRegistration(ctx, tc.registration)
			if !errors.Is(err, ErrDuplicateAccount) {
				t.Fatalf("CreateRegistration() error = %v, want ErrDuplicateAccount", err)
			}
			if userID != 0 {
				t.Errorf("CreateRegistration() returned user id %d alongside the error, want 0", userID)
			}
			if after := countUsers(t, ctx, pool); after != before {
				t.Errorf("the users table grew from %d to %d rows, want the transaction rolled back", before, after)
			}
		})
	}
}

func TestFindLoginUser(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	studentID := insertUser(t, ctx, pool, "student", "2023331001", StatusActive, 0)
	insertStudent(t, ctx, pool, studentID, "2023331001", false)
	crID := insertUser(t, ctx, pool, "student", "2023331002", StatusPending, 2)
	insertStudent(t, ctx, pool, crID, "2023331002", true)
	teacherID := insertUser(t, ctx, pool, "teacher", "T-001", StatusActive, 1)
	insertTeacher(t, ctx, pool, teacherID, "T-001")
	adminID := insertUser(t, ctx, pool, "admin", "admin", StatusActive, 0)
	tests := []struct {
		name             string
		identifier       string
		wantID           int64
		wantRole         string
		wantStatus       string
		wantTokenVersion int
		wantIsCR         bool
		wantName         string
		wantBatch        string
		wantSection      string
	}{
		{
			name:        "student",
			identifier:  "2023331001",
			wantID:      studentID,
			wantRole:    "student",
			wantStatus:  StatusActive,
			wantName:    "Test Student",
			wantBatch:   "CSE-23",
			wantSection: "A",
		},
		{
			name:             "pending class representative",
			identifier:       "2023331002",
			wantID:           crID,
			wantRole:         "student",
			wantStatus:       StatusPending,
			wantTokenVersion: 2,
			wantIsCR:         true,
			wantName:         "Test Student",
			wantBatch:        "CSE-23",
			wantSection:      "A",
		},
		{
			name:             "teacher takes its name from the teachers table",
			identifier:       "T-001",
			wantID:           teacherID,
			wantRole:         "teacher",
			wantStatus:       StatusActive,
			wantTokenVersion: 1,
			wantName:         "Test Teacher",
		},
		{
			name:       "admin has no profile row",
			identifier: "admin",
			wantID:     adminID,
			wantRole:   "admin",
			wantStatus: StatusActive,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			user, err := store.FindLoginUser(ctx, tc.identifier)
			if err != nil {
				t.Fatalf("FindLoginUser() returned an unexpected error: %v", err)
			}
			if user.ID != tc.wantID {
				t.Errorf("ID = %d, want %d", user.ID, tc.wantID)
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
			if user.Name != tc.wantName {
				t.Errorf("Name = %q, want %q", user.Name, tc.wantName)
			}
			if user.Batch != tc.wantBatch {
				t.Errorf("Batch = %q, want %q", user.Batch, tc.wantBatch)
			}
			if user.Section != tc.wantSection {
				t.Errorf("Section = %q, want %q", user.Section, tc.wantSection)
			}
			if user.PasswordHash == "" {
				t.Error("PasswordHash is empty, want the stored hash")
			}
		})
	}
}

func TestFindLoginUserReportsUnknownIdentifiers(t *testing.T) {
	ctx, store, _ := newTestStore(t)
	tests := []struct {
		name       string
		identifier string
	}{
		{name: "unknown", identifier: "2023339999"},
		{name: "empty", identifier: ""},
		{name: "sql injection attempt", identifier: "' OR 1=1 --"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			user, err := store.FindLoginUser(ctx, tc.identifier)
			if !errors.Is(err, ErrNotFound) {
				t.Fatalf("FindLoginUser() error = %v, want ErrNotFound", err)
			}
			if user.ID != 0 {
				t.Error("FindLoginUser() returned a user alongside the error")
			}
		})
	}
}

func TestIncrementTokenVersion(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	userID := insertUser(t, ctx, pool, "student", "2023331001", StatusActive, 0)
	for want := 1; want <= 3; want++ {
		got, err := store.IncrementTokenVersion(ctx, userID)
		if err != nil {
			t.Fatalf("IncrementTokenVersion() returned an unexpected error: %v", err)
		}
		if got != want {
			t.Errorf("IncrementTokenVersion() = %d, want %d", got, want)
		}
		user, err := store.GetAuthUser(ctx, userID)
		if err != nil {
			t.Fatalf("GetAuthUser() returned an unexpected error: %v", err)
		}
		if user.TokenVersion != want {
			t.Errorf("the stored token version = %d, want %d", user.TokenVersion, want)
		}
	}
	if _, err := store.IncrementTokenVersion(ctx, 999999); !errors.Is(err, ErrNotFound) {
		t.Errorf("IncrementTokenVersion() error for an unknown user = %v, want ErrNotFound", err)
	}
}

func TestSetResetToken(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	activeID := insertUser(t, ctx, pool, "student", "2023331001", StatusActive, 0)
	pendingID := insertUser(t, ctx, pool, "student", "2023331002", StatusPending, 0)
	removedID := insertUser(t, ctx, pool, "student", "2023331003", StatusRemoved, 0)
	expiresAt := time.Now().Add(15 * time.Minute)
	if err := store.SetResetToken(ctx, activeID, "token-hash", expiresAt); err != nil {
		t.Fatalf("SetResetToken() returned an unexpected error: %v", err)
	}
	var storedHash string
	var storedExpiry time.Time
	query := "SELECT reset_token, reset_token_expires_at FROM users WHERE id = $1"
	if err := pool.QueryRow(ctx, query, activeID).Scan(&storedHash, &storedExpiry); err != nil {
		t.Fatalf("could not read back the reset token: %v", err)
	}
	if storedHash != "token-hash" {
		t.Errorf("stored reset token = %q, want %q", storedHash, "token-hash")
	}
	if storedExpiry.Sub(expiresAt).Abs() > time.Second {
		t.Errorf("stored expiry = %v, want about %v", storedExpiry, expiresAt)
	}
	tests := []struct {
		name   string
		userID int64
	}{
		{name: "pending account", userID: pendingID},
		{name: "removed account", userID: removedID},
		{name: "unknown account", userID: 999999},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if err := store.SetResetToken(ctx, tc.userID, "token-hash-2", expiresAt); !errors.Is(err, ErrNotFound) {
				t.Errorf("SetResetToken() error = %v, want ErrNotFound", err)
			}
		})
	}
}

func TestResetPassword(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	userID := insertUser(t, ctx, pool, "student", "2023331001", StatusActive, 4)
	if err := store.SetResetToken(ctx, userID, "correct-hash", time.Now().Add(15*time.Minute)); err != nil {
		t.Fatalf("SetResetToken() returned an unexpected error: %v", err)
	}
	if err := store.ResetPassword(ctx, "2023331001", "wrong-hash", "new-stored-hash"); !errors.Is(err, ErrInvalidResetToken) {
		t.Fatalf("ResetPassword() with a wrong token = %v, want ErrInvalidResetToken", err)
	}
	if err := store.ResetPassword(ctx, "2023331001", "correct-hash", "new-stored-hash"); err != nil {
		t.Fatalf("ResetPassword() returned an unexpected error: %v", err)
	}
	var passwordHash string
	var resetToken, resetExpiry *string
	var tokenVersion int
	query := "SELECT password_hash, reset_token, reset_token_expires_at::text, token_version FROM users WHERE id = $1"
	if err := pool.QueryRow(ctx, query, userID).Scan(&passwordHash, &resetToken, &resetExpiry, &tokenVersion); err != nil {
		t.Fatalf("could not read back the user row: %v", err)
	}
	if passwordHash != "new-stored-hash" {
		t.Error("the password hash was not replaced")
	}
	if resetToken != nil || resetExpiry != nil {
		t.Error("the reset token fields were not cleared")
	}
	if tokenVersion != 5 {
		t.Errorf("token version = %d, want 5 so that old tokens stop working", tokenVersion)
	}
	if err := store.ResetPassword(ctx, "2023331001", "correct-hash", "third-stored-hash"); !errors.Is(err, ErrInvalidResetToken) {
		t.Errorf("reusing the same reset token = %v, want ErrInvalidResetToken", err)
	}
}

func TestResetPasswordRejectsExpiredAndInactive(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	expiredID := insertUser(t, ctx, pool, "student", "2023331001", StatusActive, 0)
	pendingID := insertUser(t, ctx, pool, "student", "2023331002", StatusPending, 0)
	expireQuery := `UPDATE users SET reset_token = $2, reset_token_expires_at = now() - INTERVAL '1 minute' WHERE id = $1`
	if _, err := pool.Exec(ctx, expireQuery, expiredID, "expired-hash"); err != nil {
		t.Fatalf("could not plant an expired reset token: %v", err)
	}
	if _, err := pool.Exec(ctx, expireQuery, pendingID, "pending-hash"); err != nil {
		t.Fatalf("could not plant a reset token on a pending account: %v", err)
	}
	tests := []struct {
		name       string
		identifier string
		tokenHash  string
	}{
		{name: "expired token", identifier: "2023331001", tokenHash: "expired-hash"},
		{name: "pending account", identifier: "2023331002", tokenHash: "pending-hash"},
		{name: "unknown identifier", identifier: "2023339999", tokenHash: "expired-hash"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := store.ResetPassword(ctx, tc.identifier, tc.tokenHash, "new-stored-hash")
			if !errors.Is(err, ErrInvalidResetToken) {
				t.Errorf("ResetPassword() error = %v, want ErrInvalidResetToken", err)
			}
		})
	}
}

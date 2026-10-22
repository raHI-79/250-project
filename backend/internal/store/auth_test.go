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
		DateOfBirth:  "2003-05-14",
		PasswordHash: "stored-hash",
		PhotoPath:    "/uploads/photos/" + regNo + ".jpg",
		Status:       StatusPending,
	}
}

func newDropCourse(t *testing.T, ctx context.Context, store *Store, code, title string) int64 {
	t.Helper()
	course, err := store.CreateCourse(ctx, code, title, 3.0)
	if err != nil {
		t.Fatalf("CreateCourse() returned an unexpected error: %v", err)
	}
	return course.ID
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
	result, err := store.CreateRegistration(ctx, newRegistration("2023331079", "ratul@student.sust.edu"))
	if err != nil {
		t.Fatalf("CreateRegistration() returned an unexpected error: %v", err)
	}
	userID := result.UserID
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
	var fullName, batch, section, email, phone, bloodGroup, birthDate string
	var studentID int64
	query := `SELECT id, full_name, batch, section, email, phone, blood_group, to_char(date_of_birth, 'YYYY-MM-DD')
FROM students WHERE user_id = $1`
	if err := pool.QueryRow(ctx, query, userID).Scan(
		&studentID, &fullName, &batch, &section, &email, &phone, &bloodGroup, &birthDate,
	); err != nil {
		t.Fatalf("could not read back the student row: %v", err)
	}
	if fullName != "Ratul Islam" || batch != "CSE-23" || section != "A" {
		t.Errorf("student row = %q/%q/%q, want Ratul Islam/CSE-23/A", fullName, batch, section)
	}
	if email != "ratul@student.sust.edu" || phone != "01712345678" || bloodGroup != "O+" {
		t.Errorf("contact row = %q/%q/%q", email, phone, bloodGroup)
	}
	if birthDate != "2003-05-14" {
		t.Errorf("date_of_birth = %q, want 2003-05-14", birthDate)
	}
	if user.PhotoPath != "/uploads/photos/2023331079.jpg" {
		t.Errorf("photo_path = %q, want the saved photo", user.PhotoPath)
	}
	courses, err := store.ListStudentDropCourses(ctx, studentID)
	if err != nil {
		t.Fatalf("ListStudentDropCourses() returned an unexpected error: %v", err)
	}
	if len(courses) != 0 {
		t.Errorf("drop courses = %d rows, want none", len(courses))
	}
}

func TestCreateRegistrationWithDropCourses(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	first := newDropCourse(t, ctx, store, "CSE 233", "Algorithms")
	second := newDropCourse(t, ctx, store, "CSE 245", "Databases")
	registration := newRegistration("2023331080", "drop@student.sust.edu")
	registration.DropCourseIDs = []int64{first, second}
	result, err := store.CreateRegistration(ctx, registration)
	if err != nil {
		t.Fatalf("CreateRegistration() returned an unexpected error: %v", err)
	}
	var studentID int64
	if err := pool.QueryRow(ctx, "SELECT id FROM students WHERE user_id = $1", result.UserID).Scan(&studentID); err != nil {
		t.Fatalf("could not read back the student row: %v", err)
	}
	courses, err := store.ListStudentDropCourses(ctx, studentID)
	if err != nil {
		t.Fatalf("ListStudentDropCourses() returned an unexpected error: %v", err)
	}
	if len(courses) != 2 || courses[0].Code != "CSE 233" || courses[1].Code != "CSE 245" {
		t.Errorf("drop courses = %+v, want CSE 233 and CSE 245", courses)
	}
}

func TestCreateRegistrationReplacesAnUnverifiedAccount(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	pending := newRegistration("2023331090", "retry@student.sust.edu")
	pending.Status = StatusUnverified
	first, err := store.CreateRegistration(ctx, pending)
	if err != nil {
		t.Fatalf("the first registration failed: %v", err)
	}
	second, err := store.CreateRegistration(ctx, newRegistration("2023331090", "retry@student.sust.edu"))
	if err != nil {
		t.Fatalf("the replacement registration failed: %v", err)
	}
	if second.UserID == first.UserID {
		t.Error("the replacement reused the old user row, want a fresh one")
	}
	if len(second.ReplacedPhotos) != 1 || second.ReplacedPhotos[0] != pending.PhotoPath {
		t.Errorf("ReplacedPhotos = %v, want %q", second.ReplacedPhotos, pending.PhotoPath)
	}
	if got := countUsers(t, ctx, pool); got != 1 {
		t.Errorf("users = %d rows, want 1", got)
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
			result, err := store.CreateRegistration(ctx, tc.registration)
			if !errors.Is(err, ErrDuplicateAccount) {
				t.Fatalf("CreateRegistration() error = %v, want ErrDuplicateAccount", err)
			}
			if result.UserID != 0 {
				t.Errorf("CreateRegistration() returned user id %d alongside the error, want 0", result.UserID)
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

func TestStartPasswordReset(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	activeID := insertUser(t, ctx, pool, "student", "2023331001", StatusActive, 0)
	pendingID := insertUser(t, ctx, pool, "student", "2023331002", StatusPending, 0)
	removedID := insertUser(t, ctx, pool, "student", "2023331003", StatusRemoved, 0)
	sentAt := time.Now()
	expiresAt := sentAt.Add(10 * time.Minute)
	if err := store.StartPasswordReset(ctx, activeID, "code-hash", expiresAt, sentAt); err != nil {
		t.Fatalf("StartPasswordReset() returned an unexpected error: %v", err)
	}
	var storedHash, storedStage string
	var storedExpiry, storedSentAt time.Time
	var attempts int
	query := `SELECT reset_token, reset_stage, reset_token_expires_at, reset_sent_at, reset_attempts
FROM users WHERE id = $1`
	if err := pool.QueryRow(ctx, query, activeID).Scan(&storedHash, &storedStage, &storedExpiry, &storedSentAt, &attempts); err != nil {
		t.Fatalf("could not read back the reset fields: %v", err)
	}
	if storedHash != "code-hash" {
		t.Errorf("stored code hash = %q, want %q", storedHash, "code-hash")
	}
	if storedStage != ResetStageCode {
		t.Errorf("stored stage = %q, want %q", storedStage, ResetStageCode)
	}
	if attempts != 0 {
		t.Errorf("stored attempts = %d, want 0", attempts)
	}
	if storedExpiry.Sub(expiresAt).Abs() > time.Second {
		t.Errorf("stored expiry = %v, want about %v", storedExpiry, expiresAt)
	}
	if storedSentAt.Sub(sentAt).Abs() > time.Second {
		t.Errorf("stored sent_at = %v, want about %v", storedSentAt, sentAt)
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
			err := store.StartPasswordReset(ctx, tc.userID, "code-hash-2", expiresAt, sentAt)
			if !errors.Is(err, ErrNotFound) {
				t.Errorf("StartPasswordReset() error = %v, want ErrNotFound", err)
			}
		})
	}
}

func TestFindResetTarget(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	studentID := insertUser(t, ctx, pool, "student", "2023331001", StatusActive, 0)
	insertStudent(t, ctx, pool, studentID, "2023331001", false)
	teacherID := insertUser(t, ctx, pool, "teacher", "T-001", StatusActive, 0)
	insertTeacher(t, ctx, pool, teacherID, "T-001")
	if _, err := store.EnsureAdmin(ctx, "admin", "stored-hash", "admin@sust.edu", "+8801999999999"); err != nil {
		t.Fatalf("EnsureAdmin() returned an unexpected error: %v", err)
	}
	tests := []struct {
		name       string
		identifier string
		wantEmail  string
		wantPhone  string
	}{
		{
			name:       "student reads the student row",
			identifier: "2023331001",
			wantEmail:  "2023331001@student.sust.edu",
			wantPhone:  "01700000000",
		},
		{name: "teacher reads the teacher row", identifier: "T-001", wantEmail: "teacher@example.com", wantPhone: "01800000000"},
		{name: "admin reads the contact columns", identifier: "admin", wantEmail: "admin@sust.edu", wantPhone: "+8801999999999"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			target, err := store.FindResetTarget(ctx, tc.identifier)
			if err != nil {
				t.Fatalf("FindResetTarget() returned an unexpected error: %v", err)
			}
			if target.Email != tc.wantEmail {
				t.Errorf("email = %q, want %q", target.Email, tc.wantEmail)
			}
			if target.Phone != tc.wantPhone {
				t.Errorf("phone = %q, want %q", target.Phone, tc.wantPhone)
			}
			if !target.IsActive() {
				t.Errorf("status = %q, want active", target.Status)
			}
			if target.ResetSentAt != nil {
				t.Error("reset_sent_at is set on a fresh account, want nil")
			}
		})
	}
	if _, err := store.FindResetTarget(ctx, "2023339999"); !errors.Is(err, ErrNotFound) {
		t.Errorf("FindResetTarget() error for an unknown id = %v, want ErrNotFound", err)
	}
}

func TestResetChallengeAttemptsAndClear(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	userID := insertUser(t, ctx, pool, "student", "2023331001", StatusActive, 0)
	now := time.Now()
	if err := store.StartPasswordReset(ctx, userID, "code-hash", now.Add(10*time.Minute), now); err != nil {
		t.Fatalf("StartPasswordReset() returned an unexpected error: %v", err)
	}
	challenge, err := store.FindResetChallenge(ctx, "2023331001")
	if err != nil {
		t.Fatalf("FindResetChallenge() returned an unexpected error: %v", err)
	}
	if challenge.UserID != userID || challenge.TokenHash != "code-hash" || challenge.Stage != ResetStageCode {
		t.Errorf("challenge = %+v, want the stored code challenge", challenge)
	}
	if challenge.Attempts != 0 || challenge.ExpiresAt == nil {
		t.Errorf("challenge attempts = %d and expiry = %v, want 0 and a timestamp", challenge.Attempts, challenge.ExpiresAt)
	}
	for want := 1; want <= 3; want++ {
		got, err := store.CountResetAttempt(ctx, userID)
		if err != nil {
			t.Fatalf("CountResetAttempt() returned an unexpected error: %v", err)
		}
		if got != want {
			t.Errorf("CountResetAttempt() = %d, want %d", got, want)
		}
	}
	if _, err := store.CountResetAttempt(ctx, 999999); !errors.Is(err, ErrNotFound) {
		t.Errorf("CountResetAttempt() error for an unknown user = %v, want ErrNotFound", err)
	}
	if err := store.ClearPasswordReset(ctx, userID); err != nil {
		t.Fatalf("ClearPasswordReset() returned an unexpected error: %v", err)
	}
	cleared, err := store.FindResetChallenge(ctx, "2023331001")
	if err != nil {
		t.Fatalf("FindResetChallenge() returned an unexpected error: %v", err)
	}
	if cleared.TokenHash != "" || cleared.Stage != "" || cleared.Attempts != 0 || cleared.ExpiresAt != nil {
		t.Errorf("challenge after clearing = %+v, want empty reset fields", cleared)
	}
	if _, err := store.FindResetChallenge(ctx, "2023339999"); !errors.Is(err, ErrNotFound) {
		t.Errorf("FindResetChallenge() error for an unknown id = %v, want ErrNotFound", err)
	}
}

func TestVerifyResetCodeNeedsTheCodeStage(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	userID := insertUser(t, ctx, pool, "student", "2023331001", StatusActive, 0)
	now := time.Now()
	if err := store.VerifyResetCode(ctx, userID, "token-hash", now.Add(10*time.Minute)); !errors.Is(err, ErrInvalidResetToken) {
		t.Fatalf("VerifyResetCode() without a code stage = %v, want ErrInvalidResetToken", err)
	}
	if err := store.StartPasswordReset(ctx, userID, "code-hash", now.Add(10*time.Minute), now); err != nil {
		t.Fatalf("StartPasswordReset() returned an unexpected error: %v", err)
	}
	if _, err := store.CountResetAttempt(ctx, userID); err != nil {
		t.Fatalf("CountResetAttempt() returned an unexpected error: %v", err)
	}
	if err := store.VerifyResetCode(ctx, userID, "token-hash", now.Add(10*time.Minute)); err != nil {
		t.Fatalf("VerifyResetCode() returned an unexpected error: %v", err)
	}
	challenge, err := store.FindResetChallenge(ctx, "2023331001")
	if err != nil {
		t.Fatalf("FindResetChallenge() returned an unexpected error: %v", err)
	}
	if challenge.Stage != ResetStageVerified {
		t.Errorf("stage = %q, want %q", challenge.Stage, ResetStageVerified)
	}
	if challenge.TokenHash != "token-hash" {
		t.Errorf("token hash = %q, want %q", challenge.TokenHash, "token-hash")
	}
	if challenge.Attempts != 0 {
		t.Errorf("attempts = %d, want them reset to 0", challenge.Attempts)
	}
	if err := store.VerifyResetCode(ctx, userID, "second-hash", now.Add(10*time.Minute)); !errors.Is(err, ErrInvalidResetToken) {
		t.Errorf("VerifyResetCode() on an already verified reset = %v, want ErrInvalidResetToken", err)
	}
}

func TestFinishPasswordReset(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	userID := insertUser(t, ctx, pool, "student", "2023331001", StatusActive, 4)
	now := time.Now()
	if err := store.StartPasswordReset(ctx, userID, "code-hash", now.Add(10*time.Minute), now); err != nil {
		t.Fatalf("StartPasswordReset() returned an unexpected error: %v", err)
	}
	err := store.FinishPasswordReset(ctx, "2023331001", "code-hash", "new-stored-hash")
	if !errors.Is(err, ErrInvalidResetToken) {
		t.Fatalf("FinishPasswordReset() at the code stage = %v, want ErrInvalidResetToken", err)
	}
	if err := store.VerifyResetCode(ctx, userID, "token-hash", now.Add(10*time.Minute)); err != nil {
		t.Fatalf("VerifyResetCode() returned an unexpected error: %v", err)
	}
	if err := store.FinishPasswordReset(ctx, "2023331001", "wrong-hash", "new-stored-hash"); !errors.Is(err, ErrInvalidResetToken) {
		t.Fatalf("FinishPasswordReset() with a wrong token = %v, want ErrInvalidResetToken", err)
	}
	if err := store.FinishPasswordReset(ctx, "2023331001", "token-hash", "new-stored-hash"); err != nil {
		t.Fatalf("FinishPasswordReset() returned an unexpected error: %v", err)
	}
	var passwordHash string
	var resetToken, resetExpiry, resetStage, resetSentAt *string
	var tokenVersion, attempts int
	query := `SELECT password_hash, reset_token, reset_token_expires_at::text, reset_stage, reset_sent_at::text,
token_version, reset_attempts FROM users WHERE id = $1`
	if err := pool.QueryRow(ctx, query, userID).Scan(&passwordHash, &resetToken, &resetExpiry,
		&resetStage, &resetSentAt, &tokenVersion, &attempts); err != nil {
		t.Fatalf("could not read back the user row: %v", err)
	}
	if passwordHash != "new-stored-hash" {
		t.Error("the password hash was not replaced")
	}
	if resetToken != nil || resetExpiry != nil || resetStage != nil || resetSentAt != nil || attempts != 0 {
		t.Error("the reset fields were not cleared")
	}
	if tokenVersion != 5 {
		t.Errorf("token version = %d, want 5 so that old tokens stop working", tokenVersion)
	}
	if err := store.FinishPasswordReset(ctx, "2023331001", "token-hash", "third-stored-hash"); !errors.Is(err, ErrInvalidResetToken) {
		t.Errorf("reusing the same reset token = %v, want ErrInvalidResetToken", err)
	}
}

func TestFinishPasswordResetRejectsExpiredAndInactive(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	expiredID := insertUser(t, ctx, pool, "student", "2023331001", StatusActive, 0)
	pendingID := insertUser(t, ctx, pool, "student", "2023331002", StatusPending, 0)
	plant := `UPDATE users SET reset_token = $2, reset_stage = 'verified',
reset_token_expires_at = now() - INTERVAL '1 minute' WHERE id = $1`
	if _, err := pool.Exec(ctx, plant, expiredID, "expired-hash"); err != nil {
		t.Fatalf("could not plant an expired reset token: %v", err)
	}
	if _, err := pool.Exec(ctx, plant, pendingID, "pending-hash"); err != nil {
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
			err := store.FinishPasswordReset(ctx, tc.identifier, tc.tokenHash, "new-stored-hash")
			if !errors.Is(err, ErrInvalidResetToken) {
				t.Errorf("FinishPasswordReset() error = %v, want ErrInvalidResetToken", err)
			}
		})
	}
}

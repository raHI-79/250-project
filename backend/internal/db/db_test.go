package db

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/raHI-79/250-project/backend/internal/dbtest"
	"github.com/raHI-79/250-project/backend/internal/validate"
	"log"
	"os"
	"testing"
)

const (
	uniqueViolationCode     = "23505"
	checkViolationCode      = "23514"
	foreignKeyViolationCode = "23503"
	notNullViolationCode    = "23502"
)

var expectedTables = []string{
	"admin_audit",
	"batch_drive_links",
	"courses",
	"messages",
	"notices",
	"notifications",
	"poll_options",
	"poll_votes",
	"polls",
	"push_subscriptions",
	"publications",
	"published_items",
	"schedule_items",
	"routines",
	"conversation_members",
	"conversation_reads",
	"conversations",
	"schema_migrations",
	"student_drop_courses",
	"students",
	"teachers",
	"users",
}

func TestMain(m *testing.M) {
	if err := dbtest.LoadEnv(); err != nil {
		log.Fatalf("db: could not load backend/.env: %v", err)
	}
	os.Exit(m.Run())
}

func migratedTestDB(t *testing.T) (context.Context, *pgxpool.Pool) {
	t.Helper()
	ctx, pool := dbtest.Pool(t)
	if err := Migrate(ctx, pool); err != nil {
		t.Fatalf("Migrate() returned an unexpected error: %v", err)
	}
	return ctx, pool
}

func pgErrorCode(t *testing.T, err error) string {
	t.Helper()
	if err == nil {
		t.Fatal("expected a PostgreSQL error, got nil")
	}
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		t.Fatalf("error %v is not a PostgreSQL error", err)
	}
	return pgErr.Code
}

func tableExists(t *testing.T, ctx context.Context, pool *pgxpool.Pool, table string) bool {
	t.Helper()
	var exists bool
	query := "SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'public' AND table_name = $1)"
	if err := pool.QueryRow(ctx, query, table).Scan(&exists); err != nil {
		t.Fatalf("could not check whether table %q exists: %v", table, err)
	}
	return exists
}

func countRows(t *testing.T, ctx context.Context, pool *pgxpool.Pool, query string, args ...any) int {
	t.Helper()
	var count int
	if err := pool.QueryRow(ctx, query, args...).Scan(&count); err != nil {
		t.Fatalf("count query failed: %v", err)
	}
	return count
}

func insertUser(t *testing.T, ctx context.Context, pool *pgxpool.Pool, role, identifier string) int64 {
	t.Helper()
	var id int64
	query := "INSERT INTO users (role, identifier, password_hash, status) VALUES ($1, $2, 'not-a-real-hash', 'active') RETURNING id"
	if err := pool.QueryRow(ctx, query, role, identifier).Scan(&id); err != nil {
		t.Fatalf("could not insert the %s user %q: %v", role, identifier, err)
	}
	return id
}

func insertStudent(t *testing.T, ctx context.Context, pool *pgxpool.Pool, userID int64, regNo string) int64 {
	t.Helper()
	var id int64
	query := `INSERT INTO students (user_id, full_name, reg_no, batch, section, email, phone, blood_group)
VALUES ($1, 'Test Student', $2, 'CSE-23', 'A', $2 || '@student.sust.edu', '01700000000', 'O+') RETURNING id`
	if err := pool.QueryRow(ctx, query, userID, regNo).Scan(&id); err != nil {
		t.Fatalf("could not insert the student %q: %v", regNo, err)
	}
	return id
}

func insertTeacher(t *testing.T, ctx context.Context, pool *pgxpool.Pool, userID int64, teacherID string) int64 {
	t.Helper()
	var id int64
	query := `INSERT INTO teachers (user_id, teacher_id, name, email, phone)
VALUES ($1, $2, 'Test Teacher', 'teacher@example.com', '01800000000') RETURNING id`
	if err := pool.QueryRow(ctx, query, userID, teacherID).Scan(&id); err != nil {
		t.Fatalf("could not insert the teacher %q: %v", teacherID, err)
	}
	return id
}

func insertCourse(t *testing.T, ctx context.Context, pool *pgxpool.Pool, code string) int64 {
	t.Helper()
	var id int64
	query := "INSERT INTO courses (code, title) VALUES ($1, 'Digital Logic Design') RETURNING id"
	if err := pool.QueryRow(ctx, query, validate.NormalizeCourseCode(code)).Scan(&id); err != nil {
		t.Fatalf("could not insert the course %q: %v", code, err)
	}
	return id
}

func TestMigrateCreatesEveryTable(t *testing.T) {
	ctx, pool := migratedTestDB(t)
	for _, table := range expectedTables {
		if !tableExists(t, ctx, pool, table) {
			t.Errorf("table %q was not created", table)
		}
	}
	got := countRows(t, ctx, pool, "SELECT count(*) FROM information_schema.tables WHERE table_schema = 'public'")
	if got != len(expectedTables) {
		t.Errorf("public schema has %d tables, want %d", got, len(expectedTables))
	}
	versions, err := migrationVersions(migrationFiles)
	if err != nil {
		t.Fatalf("migrationVersions() returned an unexpected error: %v", err)
	}
	applied := countRows(t, ctx, pool, "SELECT count(*) FROM schema_migrations")
	if applied != len(versions) {
		t.Errorf("schema_migrations has %d rows, want %d", applied, len(versions))
	}
}

func TestMigrateIsIdempotent(t *testing.T) {
	ctx, pool := migratedTestDB(t)
	versions, err := migrationVersions(migrationFiles)
	if err != nil {
		t.Fatalf("migrationVersions() returned an unexpected error: %v", err)
	}
	for run := 2; run <= 4; run++ {
		if err := Migrate(ctx, pool); err != nil {
			t.Fatalf("Migrate() failed on run %d: %v", run, err)
		}
		applied := countRows(t, ctx, pool, "SELECT count(*) FROM schema_migrations")
		if applied != len(versions) {
			t.Fatalf("after run %d schema_migrations has %d rows, want %d", run, applied, len(versions))
		}
	}
	for _, table := range expectedTables {
		if !tableExists(t, ctx, pool, table) {
			t.Errorf("table %q disappeared after repeated migration runs", table)
		}
	}
}

func TestPollVotesAllowSeveralOptionsOnce(t *testing.T) {
	ctx, pool := migratedTestDB(t)
	crUserID := insertUser(t, ctx, pool, "student", "2023331001")
	studentUserID := insertUser(t, ctx, pool, "student", "2023331002")
	studentID := insertStudent(t, ctx, pool, studentUserID, "2023331002")
	var pollID int64
	pollQuery := "INSERT INTO polls (question, batch, section, created_by) VALUES ('Lab on Friday?', 'CSE-23', 'A', $1) RETURNING id"
	if err := pool.QueryRow(ctx, pollQuery, crUserID).Scan(&pollID); err != nil {
		t.Fatalf("could not insert the poll: %v", err)
	}
	var firstOptionID, secondOptionID int64
	optionQuery := "INSERT INTO poll_options (poll_id, option_text, position) VALUES ($1, $2, $3) RETURNING id"
	if err := pool.QueryRow(ctx, optionQuery, pollID, "Yes", 0).Scan(&firstOptionID); err != nil {
		t.Fatalf("could not insert the first poll option: %v", err)
	}
	if err := pool.QueryRow(ctx, optionQuery, pollID, "No", 1).Scan(&secondOptionID); err != nil {
		t.Fatalf("could not insert the second poll option: %v", err)
	}
	voteQuery := "INSERT INTO poll_votes (poll_id, option_id, student_id) VALUES ($1, $2, $3)"
	if _, err := pool.Exec(ctx, voteQuery, pollID, firstOptionID, studentID); err != nil {
		t.Fatalf("the first vote was rejected: %v", err)
	}
	if _, err := pool.Exec(ctx, voteQuery, pollID, secondOptionID, studentID); err != nil {
		t.Fatalf("a second option of the same poll was rejected: %v", err)
	}
	_, err := pool.Exec(ctx, voteQuery, pollID, firstOptionID, studentID)
	if code := pgErrorCode(t, err); code != uniqueViolationCode {
		t.Errorf("repeating one option error code = %s, want %s (unique violation)", code, uniqueViolationCode)
	}
	votes := countRows(t, ctx, pool, "SELECT count(*) FROM poll_votes WHERE poll_id = $1", pollID)
	if votes != 2 {
		t.Errorf("poll has %d votes, want 2", votes)
	}
	badChoices := "UPDATE polls SET max_choices = 0 WHERE id = $1"
	if _, err := pool.Exec(ctx, badChoices, pollID); pgErrorCode(t, err) != checkViolationCode {
		t.Errorf("max_choices 0 was accepted, want the check constraint to refuse it")
	}
}

func TestRoutineAndClassEventTimeChecks(t *testing.T) {
	ctx, pool := migratedTestDB(t)
	adminUserID := insertUser(t, ctx, pool, "admin", "admin")
	teacherUserID := insertUser(t, ctx, pool, "teacher", "T-001")
	teacherRowID := insertTeacher(t, ctx, pool, teacherUserID, "T-001")
	courseID := insertCourse(t, ctx, pool, "CSE 250")
	routineQuery := `INSERT INTO routines (batch, section, course_id, teacher_id, day_of_week, start_time, end_time, room, type)
VALUES ('CSE-23', 'A', $1, $2, 1, $3, $4, '1003', 'class')`
	itemQuery := `INSERT INTO schedule_items
(batch, section, course_id, teacher_id, kind, origin, item_date, start_time, end_time, room, status, created_by)
VALUES ('CSE-23', 'A', $1, $2, 'class', 'cr', DATE '2026-10-20', $3, $4, '1003', 'scheduled', $5)`
	tests := []struct {
		name      string
		startTime string
		endTime   string
		wantCode  string
		itemCode  string
	}{
		{name: "inside class hours is accepted", startTime: "08:00", endTime: "09:30"},
		{name: "starting at 07:00 is rejected", startTime: "07:00", endTime: "09:30", wantCode: checkViolationCode,
			itemCode: ""},
		{name: "ending after 17:00 is rejected", startTime: "16:00", endTime: "17:30", wantCode: checkViolationCode,
			itemCode: ""},
		{name: "end before start is rejected", startTime: "11:00", endTime: "10:00", wantCode: checkViolationCode,
			itemCode: checkViolationCode},
		{name: "end equal to start is rejected", startTime: "11:00", endTime: "11:00", wantCode: checkViolationCode,
			itemCode: checkViolationCode},
	}
	for _, tc := range tests {
		t.Run("routines: "+tc.name, func(t *testing.T) {
			_, err := pool.Exec(ctx, routineQuery, courseID, teacherRowID, tc.startTime, tc.endTime)
			if tc.wantCode == "" {
				if err != nil {
					t.Fatalf("insert was rejected unexpectedly: %v", err)
				}
				return
			}
			if code := pgErrorCode(t, err); code != tc.wantCode {
				t.Errorf("error code = %s, want %s (check violation)", code, tc.wantCode)
			}
		})
		t.Run("schedule items: "+tc.name, func(t *testing.T) {
			_, err := pool.Exec(ctx, itemQuery, courseID, teacherRowID, tc.startTime, tc.endTime, adminUserID)
			if tc.itemCode == "" {
				if err != nil {
					t.Fatalf("insert was rejected unexpectedly: %v", err)
				}
				return
			}
			if code := pgErrorCode(t, err); code != tc.itemCode {
				t.Errorf("error code = %s, want %s (check violation)", code, tc.itemCode)
			}
		})
	}
	outOfRange := `INSERT INTO routines (batch, section, course_id, teacher_id, day_of_week, start_time, end_time, room, type)
VALUES ('CSE-23', 'A', $1, $2, 7, TIME '08:00', TIME '09:30', '1003', 'class')`
	if _, err := pool.Exec(ctx, outOfRange, courseID, teacherRowID); pgErrorCode(t, err) != checkViolationCode {
		t.Errorf("day_of_week = 7 error = %v, want a check violation", err)
	}
}

func TestForeignKeysAndChecksRejectBadRows(t *testing.T) {
	ctx, pool := migratedTestDB(t)
	studentUserID := insertUser(t, ctx, pool, "student", "2023331003")
	insertStudent(t, ctx, pool, studentUserID, "2023331003")
	teacherUserID := insertUser(t, ctx, pool, "teacher", "T-301")
	teacherID := insertTeacher(t, ctx, pool, teacherUserID, "T-301")
	courseID := insertCourse(t, ctx, pool, "CSE-301")
	var directConversationID int64
	err := pool.QueryRow(ctx,
		"INSERT INTO conversations (kind, created_by) VALUES ('direct', $1) RETURNING id",
		studentUserID).Scan(&directConversationID)
	if err != nil {
		t.Fatalf("could not insert a conversation: %v", err)
	}
	tests := []struct {
		name     string
		query    string
		args     []any
		wantCode string
	}{
		{
			name: "student without a user row",
			query: `INSERT INTO students (user_id, full_name, reg_no, batch, section, email, phone, blood_group)
VALUES (999999, 'Ghost Student', '2023331999', 'CSE-23', 'A', 'ghost@example.com', '01900000000', 'A+')`,
			wantCode: foreignKeyViolationCode,
		},
		{
			name: "second student profile for the same user",
			query: `INSERT INTO students (user_id, full_name, reg_no, batch, section, email, phone, blood_group)
VALUES ($1, 'Duplicate Student', '2023331004', 'CSE-23', 'A', 'dup@example.com', '01900000001', 'A+')`,
			args:     []any{studentUserID},
			wantCode: uniqueViolationCode,
		},
		{
			name: "invalid blood group",
			query: `INSERT INTO students (user_id, full_name, reg_no, batch, section, email, phone, blood_group)
VALUES (999999, 'Odd Blood', '2023331005', 'CSE-23', 'A', 'odd@example.com', '01900000002', 'C+')`,
			wantCode: checkViolationCode,
		},
		{
			name:     "drop course for an unknown student",
			query:    "INSERT INTO student_drop_courses (student_id, course_id) VALUES (999999, 999998)",
			wantCode: foreignKeyViolationCode,
		},
		{
			name: "invalid teacher blood group",
			query: `INSERT INTO teachers (user_id, teacher_id, name, email, phone, blood_group)
VALUES (999999, 'T-900', 'Odd Teacher', 'odd@sust.edu', '01900000004', 'C+')`,
			wantCode: checkViolationCode,
		},
		{
			name:     "invalid user role",
			query:    "INSERT INTO users (role, identifier, password_hash) VALUES ('principal', 'p-001', 'not-a-real-hash')",
			wantCode: checkViolationCode,
		},
		{
			name:     "invalid user status",
			query:    "INSERT INTO users (role, identifier, password_hash, status) VALUES ('student', '2023331006', 'not-a-real-hash', 'banned')",
			wantCode: checkViolationCode,
		},
		{
			name:     "duplicate identifier",
			query:    "INSERT INTO users (role, identifier, password_hash) VALUES ('student', '2023331003', 'not-a-real-hash')",
			wantCode: uniqueViolationCode,
		},
		{
			name:     "user without a password hash",
			query:    "INSERT INTO users (role, identifier) VALUES ('student', '2023331007')",
			wantCode: notNullViolationCode,
		},
		{
			name:     "notification without any target",
			query:    "INSERT INTO notifications (type, title, body) VALUES ('notice', 'Hello', 'Body')",
			wantCode: checkViolationCode,
		},
		{
			name:     "message without a body, a link or a file",
			query:    "INSERT INTO messages (conversation_id, sender_id, body) VALUES ($1, $2, '')",
			args:     []any{directConversationID, studentUserID},
			wantCode: checkViolationCode,
		},
		{
			name:     "message over the body limit",
			query:    "INSERT INTO messages (conversation_id, sender_id, body) VALUES ($1, $2, repeat('a', 4001))",
			args:     []any{directConversationID, studentUserID},
			wantCode: checkViolationCode,
		},
		{
			name: "message with a file but no name",
			query: `INSERT INTO messages (conversation_id, sender_id, body, file_path)
VALUES ($1, $2, 'Here', '/uploads/files/0123456789abcdef0123456789abcdef.pdf')`,
			args:     []any{directConversationID, studentUserID},
			wantCode: checkViolationCode,
		},
		{
			name:     "group conversation with a blank title",
			query:    "INSERT INTO conversations (kind, title, created_by) VALUES ('group', '', $1)",
			args:     []any{studentUserID},
			wantCode: checkViolationCode,
		},
		{
			name:     "direct conversation with a title",
			query:    "INSERT INTO conversations (kind, title, created_by) VALUES ('direct', 'Named chat', $1)",
			args:     []any{studentUserID},
			wantCode: checkViolationCode,
		},
		{
			name:     "group conversation without a title",
			query:    "INSERT INTO conversations (kind, created_by) VALUES ('group', $1)",
			args:     []any{studentUserID},
			wantCode: checkViolationCode,
		},
		{
			name: "a schedule item with only a start time",
			query: `INSERT INTO schedule_items
(batch, section, course_id, teacher_id, kind, origin, item_date, start_time, status, created_by)
VALUES ('CSE-23', 'A', $1, $2, 'class', 'cr', CURRENT_DATE + 1, TIME '08:00', 'scheduled', $3)`,
			args:     []any{courseID, teacherID, studentUserID},
			wantCode: checkViolationCode,
		},
		{
			name: "a schedule item with an unknown status",
			query: `INSERT INTO schedule_items
(batch, section, course_id, teacher_id, kind, origin, item_date, status, created_by)
VALUES ('CSE-23', 'A', $1, $2, 'class', 'cr', CURRENT_DATE + 1, 'maybe', $3)`,
			args:     []any{courseID, teacherID, studentUserID},
			wantCode: checkViolationCode,
		},
		{
			name: "a schedule item with an unknown origin",
			query: `INSERT INTO schedule_items
(batch, section, course_id, teacher_id, kind, origin, item_date, status, created_by)
VALUES ('CSE-23', 'A', $1, $2, 'class', 'office', CURRENT_DATE + 1, 'scheduled', $3)`,
			args:     []any{courseID, teacherID, studentUserID},
			wantCode: checkViolationCode,
		},
		{
			name:     "a course code without a digit",
			query:    "INSERT INTO courses (code, title) VALUES ('CSE CODE', 'No digits')",
			wantCode: checkViolationCode,
		},
		{
			name:     "teacher notice without a batch",
			query:    "INSERT INTO notices (source, title, created_by) VALUES ('teacher', 'Hello', $1)",
			args:     []any{studentUserID},
			wantCode: checkViolationCode,
		},
		{
			name:     "routine referencing a missing course",
			query:    "INSERT INTO routines (batch, section, course_id, teacher_id, day_of_week, start_time, end_time, room, type) VALUES ('CSE-23', 'A', 999999, 999999, 1, TIME '08:00', TIME '09:30', '1003', 'class')",
			wantCode: foreignKeyViolationCode,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := pool.Exec(ctx, tc.query, tc.args...)
			if code := pgErrorCode(t, err); code != tc.wantCode {
				t.Errorf("error code = %s, want %s (error was %v)", code, tc.wantCode, err)
			}
		})
	}
}

func TestNotificationTargetsAreAccepted(t *testing.T) {
	ctx, pool := migratedTestDB(t)
	userID := insertUser(t, ctx, pool, "student", "2023331008")
	tests := []struct {
		name  string
		query string
		args  []any
	}{
		{
			name:  "direct to one user",
			query: "INSERT INTO notifications (recipient_user_id, type, title, body) VALUES ($1, 'message', 'New message', 'Body')",
			args:  []any{userID},
		},
		{
			name:  "broadcast to a role",
			query: "INSERT INTO notifications (recipient_role, type, title, body) VALUES ('teacher', 'request', 'New request', 'Body')",
		},
		{
			name:  "broadcast to a batch and section",
			query: "INSERT INTO notifications (recipient_batch, recipient_section, type, title, body) VALUES ('CSE-23', 'A', 'notice', 'New notice', 'Body')",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := pool.Exec(ctx, tc.query, tc.args...); err != nil {
				t.Errorf("insert was rejected unexpectedly: %v", err)
			}
		})
	}
}

func TestGroupConversationsBecomeMemberGroups(t *testing.T) {
	ctx, pool := migratedTestDB(t)
	for _, column := range []string{"role", "added_by", "added_at"} {
		query := `SELECT COUNT(*) FROM information_schema.columns
WHERE table_name = 'conversation_members' AND column_name = $1`
		if countRows(t, ctx, pool, query, column) != 1 {
			t.Errorf("conversation_members has no %q column", column)
		}
	}
	titleQuery := `SELECT COUNT(*) FROM information_schema.columns
WHERE table_name = 'conversations' AND column_name = 'title'`
	if countRows(t, ctx, pool, titleQuery) != 1 {
		t.Error("conversations has no title column")
	}
	for _, column := range []string{"scope", "batch", "section"} {
		query := `SELECT COUNT(*) FROM information_schema.columns
WHERE table_name = 'conversations' AND column_name = $1`
		if countRows(t, ctx, pool, query, column) != 0 {
			t.Errorf("conversations still carries the old %q column", column)
		}
	}
	var userID int64
	insertUser := `INSERT INTO users (role, identifier, password_hash, status)
VALUES ('teacher', 'migration-teacher', 'stored-hash', 'active') RETURNING id`
	if err := pool.QueryRow(ctx, insertUser).Scan(&userID); err != nil {
		t.Fatalf("could not insert the teacher: %v", err)
	}
	untitled := `INSERT INTO conversations (kind, created_by) VALUES ('group', $1)`
	if _, err := pool.Exec(ctx, untitled, userID); err == nil {
		t.Error("a group without a title was accepted, want the shape constraint to refuse it")
	}
	titled := `INSERT INTO conversations (kind, title, created_by) VALUES ('group', 'CSE-23 section A', $1)
RETURNING id`
	var conversationID int64
	if err := pool.QueryRow(ctx, titled, userID).Scan(&conversationID); err != nil {
		t.Fatalf("could not insert the group: %v", err)
	}
	member := `INSERT INTO conversation_members (conversation_id, user_id, role, added_by)
VALUES ($1, $2, 'admin', $2)`
	if _, err := pool.Exec(ctx, member, conversationID, userID); err != nil {
		t.Fatalf("could not insert the group admin: %v", err)
	}
	wrongRole := `INSERT INTO conversation_members (conversation_id, user_id, role)
VALUES ($1, $2, 'owner')`
	if _, err := pool.Exec(ctx, wrongRole, conversationID, userID); err == nil {
		t.Error("an unknown member role was accepted, want the check constraint to refuse it")
	}
}

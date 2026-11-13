package db

import (
	"context"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/raHI-79/250-project/backend/internal/dbtest"
	"strings"
	"testing"
)

func migrateUpTo(t *testing.T, last string) (context.Context, *pgxpool.Pool) {
	t.Helper()
	ctx, pool := dbtest.Pool(t)
	if err := MigrateUpTo(ctx, pool, last); err != nil {
		t.Fatalf("MigrateUpTo(%q) returned an unexpected error: %v", last, err)
	}
	return ctx, pool
}

func TestSectionMigrationStopsOnAThirdSection(t *testing.T) {
	ctx, pool := migrateUpTo(t, "0020_message_edits.sql")
	userID := insertUser(t, ctx, pool, "student", "2023331700")
	query := `INSERT INTO students (user_id, full_name, reg_no, batch, section, email, phone, blood_group)
VALUES ($1, 'Third Section Student', '2023331700', 'CSE-23', 'C', 'c@student.sust.edu', '01700000700', 'O+')`
	if _, err := pool.Exec(ctx, query, userID); err != nil {
		t.Fatalf("could not insert the student: %v", err)
	}
	err := Migrate(ctx, pool)
	if err == nil {
		t.Fatal("Migrate() accepted a section that is not A or B")
	}
	if !strings.Contains(err.Error(), "every section must be A, B or ALL") {
		t.Errorf("error = %v, want the section complaint", err)
	}
	if !strings.Contains(err.Error(), "'C'") {
		t.Errorf("error = %v, want the offending section named", err)
	}
}

func TestSectionMigrationAcceptsAAndB(t *testing.T) {
	ctx, pool := migrateUpTo(t, "0020_message_edits.sql")
	for index, section := range []string{"A", "B"} {
		regNo := "202333171" + string(rune('0'+index))
		userID := insertUser(t, ctx, pool, "student", regNo)
		query := `INSERT INTO students (user_id, full_name, reg_no, batch, section, email, phone, blood_group)
VALUES ($1, 'Seeded Student', $2, 'CSE-23', $3, $2 || '@student.sust.edu', '01700000710', 'O+')`
		if _, err := pool.Exec(ctx, query, userID, regNo, section); err != nil {
			t.Fatalf("could not insert the %s student: %v", section, err)
		}
	}
	if err := Migrate(ctx, pool); err != nil {
		t.Fatalf("Migrate() returned an unexpected error: %v", err)
	}
	var stored int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM students").Scan(&stored); err != nil {
		t.Fatalf("could not count the students: %v", err)
	}
	if stored != 2 {
		t.Errorf("students = %d, want 2", stored)
	}
}

func TestGroupMigrationTurnsScopeGroupsIntoMemberGroups(t *testing.T) {
	ctx, pool := migrateUpTo(t, "0016_poll_choices.sql")
	teacherUserID := insertUser(t, ctx, pool, "teacher", "legacy.teacher")
	teacherQuery := `INSERT INTO teachers (user_id, teacher_id, name, email, phone)
VALUES ($1, 'legacy.teacher', 'Legacy Teacher', 'legacy@sust.edu', '01800000900')`
	if _, err := pool.Exec(ctx, teacherQuery, teacherUserID); err != nil {
		t.Fatalf("could not insert the teacher: %v", err)
	}
	members := make([]int64, 0, 2)
	for index, section := range []string{"A", "B"} {
		regNo := "202333190" + string(rune('0'+index))
		userID := insertUser(t, ctx, pool, "student", regNo)
		query := `INSERT INTO students (user_id, full_name, reg_no, batch, section, email, phone, blood_group)
VALUES ($1, 'Legacy Student', $2, 'CSE-23', $3, $2 || '@student.sust.edu', '01700000900', 'O+')`
		if _, err := pool.Exec(ctx, query, userID, regNo, section); err != nil {
			t.Fatalf("could not insert the %s student: %v", section, err)
		}
		members = append(members, userID)
	}
	var conversationID int64
	groupQuery := `INSERT INTO conversations (kind, scope, batch, section, created_by)
VALUES ('group', 'section', 'CSE-23', 'A', $1) RETURNING id`
	if err := pool.QueryRow(ctx, groupQuery, teacherUserID).Scan(&conversationID); err != nil {
		t.Fatalf("could not insert the legacy group: %v", err)
	}
	if err := Migrate(ctx, pool); err != nil {
		t.Fatalf("Migrate() returned an unexpected error: %v", err)
	}
	var title string
	if err := pool.QueryRow(ctx,
		"SELECT title FROM conversations WHERE id = $1", conversationID).Scan(&title); err != nil {
		t.Fatalf("could not read the converted group: %v", err)
	}
	if title != "CSE-23 section A" {
		t.Errorf("title = %q, want %q", title, "CSE-23 section A")
	}
	var creatorRole string
	if err := pool.QueryRow(ctx,
		"SELECT role FROM conversation_members WHERE conversation_id = $1 AND user_id = $2",
		conversationID, teacherUserID).Scan(&creatorRole); err != nil {
		t.Fatalf("the creator did not become a member: %v", err)
	}
	if creatorRole != "admin" {
		t.Errorf("creator role = %q, want admin", creatorRole)
	}
	var sectionMember string
	if err := pool.QueryRow(ctx,
		"SELECT role FROM conversation_members WHERE conversation_id = $1 AND user_id = $2",
		conversationID, members[0]).Scan(&sectionMember); err != nil {
		t.Fatalf("the section A student did not become a member: %v", err)
	}
	if sectionMember != "member" {
		t.Errorf("member role = %q, want member", sectionMember)
	}
	var otherSection int
	if err := pool.QueryRow(ctx,
		"SELECT count(*) FROM conversation_members WHERE conversation_id = $1 AND user_id = $2",
		conversationID, members[1]).Scan(&otherSection); err != nil {
		t.Fatalf("could not count the section B member: %v", err)
	}
	if otherSection != 0 {
		t.Error("the section B student joined a section A group")
	}
}

func TestPeriodMigrationSeedsTheTeachingDayOnce(t *testing.T) {
	ctx, pool := migratedTestDB(t)
	if err := Migrate(ctx, pool); err != nil {
		t.Fatalf("Migrate() returned an unexpected error on the second run: %v", err)
	}
	seeded := countRows(t, ctx, pool, "SELECT count(*) FROM class_periods")
	if seeded != 10 {
		t.Errorf("class periods = %d, want 10", seeded)
	}
	breaks := countRows(t, ctx, pool, "SELECT count(*) FROM class_periods WHERE is_break")
	if breaks != 1 {
		t.Errorf("breaks = %d, want 1", breaks)
	}
}

func TestCourseCreditMigrationGivesOldRowsThreeCredits(t *testing.T) {
	ctx, pool := migrateUpTo(t, "0022_class_periods.sql")
	if _, err := pool.Exec(ctx,
		"INSERT INTO courses (code, title) VALUES ('CSE 2243 7637D', 'Compilers')"); err != nil {
		t.Fatalf("could not insert the course: %v", err)
	}
	if err := Migrate(ctx, pool); err != nil {
		t.Fatalf("Migrate() returned an unexpected error: %v", err)
	}
	var credit float64
	if err := pool.QueryRow(ctx, "SELECT credit FROM courses WHERE code = 'CSE 2243 7637D'").
		Scan(&credit); err != nil {
		t.Fatalf("could not read the credit: %v", err)
	}
	if credit != 3.0 {
		t.Errorf("credit = %v, want 3", credit)
	}
	_, err := pool.Exec(ctx,
		"INSERT INTO courses (code, title, credit) VALUES ('CSE 1101 1101A', 'Programming', 1.3)")
	if err == nil {
		t.Error("the database accepted a credit that is not a quarter step")
	}
}

func TestBatchMigrationFillsTheBatchesInUse(t *testing.T) {
	ctx, pool := migrateUpTo(t, "0044_site_content.sql")
	userID := insertUser(t, ctx, pool, "student", "2023331079")
	statements := []string{
		`INSERT INTO students (user_id, full_name, reg_no, batch, section, email, phone, blood_group)
VALUES ($1, 'Batch Student', '2023331079', 'CSE-23', 'A', 'b@student.sust.edu', '01700000079', 'O+')`,
	}
	for _, statement := range statements {
		if _, err := pool.Exec(ctx, statement, userID); err != nil {
			t.Fatalf("could not insert the student: %v", err)
		}
	}
	if _, err := pool.Exec(ctx, `INSERT INTO batch_semesters (batch, running_semester)
VALUES ('CSE-23', '3-1'), ('CSE-25', '1-2')`); err != nil {
		t.Fatalf("could not set the running semesters: %v", err)
	}
	if err := Migrate(ctx, pool); err != nil {
		t.Fatalf("Migrate() returned an unexpected error: %v", err)
	}
	want := map[string][2]string{
		"CSE-23": {"3-1", "active"},
		"CSE-25": {"1-2", "inactive"},
	}
	rows, err := pool.Query(ctx, "SELECT name, running_semester, status FROM batches ORDER BY name")
	if err != nil {
		t.Fatalf("could not read the batches: %v", err)
	}
	defer rows.Close()
	found := 0
	for rows.Next() {
		var name, semester, status string
		if err := rows.Scan(&name, &semester, &status); err != nil {
			t.Fatalf("could not scan a batch: %v", err)
		}
		expected, ok := want[name]
		if !ok {
			t.Errorf("unexpected batch %s", name)
			continue
		}
		found++
		if semester != expected[0] || status != expected[1] {
			t.Errorf("%s = %s %s, want %s %s", name, semester, status, expected[0], expected[1])
		}
	}
	if found != len(want) {
		t.Errorf("batches found = %d, want %d", found, len(want))
	}
}

func TestOwnMarksMigrationSetsAutomaticPresentMarksBack(t *testing.T) {
	ctx, pool := migrateUpTo(t, "0045_batches.sql")
	teacherUser := insertUser(t, ctx, pool, "teacher", "T-900")
	var teacherID, courseID int64
	if err := pool.QueryRow(ctx, `INSERT INTO teachers (user_id, teacher_id, name, email, phone)
VALUES ($1, 'T-900', 'Sheet Teacher', 't900@sust.edu', '01700000900') RETURNING id`, teacherUser).Scan(&teacherID); err != nil {
		t.Fatalf("could not insert the teacher: %v", err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO courses (code, title) VALUES ('CSE 900', 'Cleanup')
RETURNING id`).Scan(&courseID); err != nil {
		t.Fatalf("could not insert the course: %v", err)
	}
	students := make([]int64, 0, 4)
	for index := 0; index < 4; index++ {
		regNo := "202333190" + string(rune('0'+index))
		userID := insertUser(t, ctx, pool, "student", regNo)
		var id int64
		if err := pool.QueryRow(ctx, `INSERT INTO students
(user_id, full_name, reg_no, batch, section, email, phone, blood_group)
VALUES ($1, 'Cleanup Student', $2, 'CSE-23', 'A', $2 || '@student.sust.edu', '01700000901', 'O+')
RETURNING id`, userID, regNo).Scan(&id); err != nil {
			t.Fatalf("could not insert a student: %v", err)
		}
		students = append(students, id)
	}
	session := func(status string) int64 {
		var id int64
		closed := "NULL"
		if status == "closed" {
			closed = "now()"
		}
		query := `INSERT INTO attendance_sessions
(course_id, batch, section, teacher_id, started_at, ends_at, original_ends_at, token_key, status, closed_at)
VALUES ($1, 'CSE-23', 'A', $2, now() - interval '30 minutes', now() + interval '30 minutes',
 now() + interval '30 minutes', 'key', $3, ` + closed + `) RETURNING id`
		if err := pool.QueryRow(ctx, query, courseID, teacherID, status).Scan(&id); err != nil {
			t.Fatalf("could not insert a %s session: %v", status, err)
		}
		return id
	}
	open, ended := session("open"), session("closed")
	rows := []struct {
		session int64
		student int64
		values  string
	}{
		{open, students[0], "'present', 'system', '', NULL, NULL, false, now()"},
		{open, students[1], "'present', 'code', '', NULL, NULL, true, now()"},
		{open, students[2], "'present', 'web', '', 24.9, 91.8, false, now()"},
		{open, students[3], "'present', 'manual', '', NULL, NULL, false, now()"},
		{ended, students[0], "'late', 'code', '', NULL, NULL, false, now()"},
		{ended, students[1], "'present', 'app', 'phone-1', 24.9, 91.8, false, now() - interval '2 hours'"},
		{ended, students[2], "'absent', 'system', '', NULL, NULL, false, now()"},
		{ended, students[3], "'present', 'app', 'phone-2', 24.9, 91.8, false, now()"},
	}
	for _, row := range rows {
		query := `INSERT INTO attendance_records
(session_id, student_id, status, method, device_id, lat, lng, no_location, marked_at)
VALUES ($1, $2, ` + row.values + `)`
		if _, err := pool.Exec(ctx, query, row.session, row.student); err != nil {
			t.Fatalf("could not insert a record: %v", err)
		}
	}
	if err := Migrate(ctx, pool); err != nil {
		t.Fatalf("Migrate() returned an unexpected error: %v", err)
	}
	read := func(sessionID, studentID int64) string {
		var status, method string
		err := pool.QueryRow(ctx, `SELECT status, method FROM attendance_records
WHERE session_id = $1 AND student_id = $2`, sessionID, studentID).Scan(&status, &method)
		if err != nil {
			return "unmarked"
		}
		return status + " " + method
	}
	want := []struct {
		session int64
		student int64
		state   string
	}{
		{open, students[0], "unmarked"},
		{open, students[1], "present code"},
		{open, students[2], "present web"},
		{open, students[3], "present manual"},
		{ended, students[0], "absent system"},
		{ended, students[1], "absent system"},
		{ended, students[2], "absent system"},
		{ended, students[3], "present app"},
	}
	for index, entry := range want {
		if got := read(entry.session, entry.student); got != entry.state {
			t.Errorf("record %d = %q, want %q", index, got, entry.state)
		}
	}
	var summary string
	if err := pool.QueryRow(ctx, `SELECT summary FROM audit_events
WHERE action = 'attendance_cleanup'`).Scan(&summary); err != nil {
		t.Fatalf("the cleanup left no audit entry: %v", err)
	}
	if !strings.HasPrefix(summary, "3 automatic present marks set back: 1 unmarked") {
		t.Errorf("audit summary = %q, want three corrected marks", summary)
	}
}

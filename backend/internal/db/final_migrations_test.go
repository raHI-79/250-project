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

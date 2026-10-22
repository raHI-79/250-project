package store

import (
	"context"
	"github.com/jackc/pgx/v5/pgxpool"
	"testing"
)

func insertPublicTeacher(t *testing.T, ctx context.Context, pool *pgxpool.Pool, userID int64, teacherID string) int64 {
	t.Helper()
	var id int64
	query := `INSERT INTO teachers (user_id, teacher_id, name, email, phone)
VALUES ($1, $2, 'Stats Teacher', $2 || '@sust.edu', '01800000000') RETURNING id`
	if err := pool.QueryRow(ctx, query, userID, teacherID).Scan(&id); err != nil {
		t.Fatalf("could not insert the teacher row: %v", err)
	}
	return id
}

func TestPublicStatsCountsActivePeopleAndBatches(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	activeStudent := insertUser(t, ctx, pool, "student", "2023331100", StatusActive, 0)
	insertStudent(t, ctx, pool, activeStudent, "2023331100", false)
	removedStudent := insertUser(t, ctx, pool, "student", "2023331101", StatusRemoved, 0)
	insertStudent(t, ctx, pool, removedStudent, "2023331101", false)
	activeTeacher := insertUser(t, ctx, pool, "teacher", "nasreen", StatusActive, 0)
	teacherRow := insertPublicTeacher(t, ctx, pool, activeTeacher, "T-100")
	invitedTeacher := insertUser(t, ctx, pool, "teacher", "rumana", StatusInvited, 0)
	insertPublicTeacher(t, ctx, pool, invitedTeacher, "T-101")
	course, err := store.CreateCourse(ctx, "CSE 245", "Databases", 3.0)
	if err != nil {
		t.Fatalf("CreateCourse() returned an unexpected error: %v", err)
	}
	if _, err := store.CreateCourse(ctx, "CSE-299", "Project Work", 3.0); err != nil {
		t.Fatalf("CreateCourse() returned an unexpected error: %v", err)
	}
	setRunningSemester(t, ctx, pool, "CSE-22", "1-1")
	routine := `INSERT INTO routines (batch, semester, section, course_id, teacher_id, day_of_week, start_time, end_time, room, type)
VALUES ('CSE-22', '1-1', 'A', $1, $2, 1, '08:00', '09:00', '1003', 'class')`
	if _, err := pool.Exec(ctx, routine, course.ID, teacherRow); err != nil {
		t.Fatalf("could not insert the routine entry: %v", err)
	}
	stats, err := store.PublicStats(ctx)
	if err != nil {
		t.Fatalf("PublicStats() returned an unexpected error: %v", err)
	}
	for _, test := range []struct {
		name string
		got  int
		want int
	}{
		{name: "Students", got: stats.Students, want: 1},
		{name: "Teachers", got: stats.Teachers, want: 1},
		{name: "Courses", got: stats.Courses, want: 2},
		{name: "Batches", got: stats.Batches, want: 2},
	} {
		if test.got != test.want {
			t.Errorf("%s = %d, want %d", test.name, test.got, test.want)
		}
	}
}

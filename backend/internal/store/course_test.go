package store

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5/pgxpool"
	"testing"
)

func insertRoutineFor(t *testing.T, ctx context.Context, pool *pgxpool.Pool, courseID int64, teacherID string) {
	t.Helper()
	userID := insertUser(t, ctx, pool, "teacher", teacherID, StatusActive, 0)
	row := insertPublicTeacher(t, ctx, pool, userID, teacherID)
	query := `INSERT INTO routines (batch, section, course_id, teacher_id, day_of_week, start_time, end_time, room, type)
VALUES ('CSE-23', 'A', $1, $2, 1, '08:00', '09:00', '1003', 'class')`
	if _, err := pool.Exec(ctx, query, courseID, row); err != nil {
		t.Fatalf("could not insert the routine entry: %v", err)
	}
}

func insertScheduleItem(t *testing.T, ctx context.Context, pool *pgxpool.Pool, courseID int64) {
	t.Helper()
	query := `INSERT INTO schedule_items
(batch, section, course_id, kind, origin, item_date, start_time, end_time, mode, room, status)
VALUES ('CSE-23', 'A', $1, 'class', 'cr', CURRENT_DATE, '08:00', '09:00', 'room', '1003', 'scheduled')`
	if _, err := pool.Exec(ctx, query, courseID); err != nil {
		t.Fatalf("could not insert the schedule item: %v", err)
	}
}

func insertDropCourse(t *testing.T, ctx context.Context, pool *pgxpool.Pool, courseID int64, regNo string) {
	t.Helper()
	userID := insertUser(t, ctx, pool, "student", regNo, StatusActive, 0)
	insertStudent(t, ctx, pool, userID, regNo, false)
	query := `INSERT INTO student_drop_courses (student_id, course_id)
SELECT s.id, $2 FROM students s WHERE s.user_id = $1`
	if _, err := pool.Exec(ctx, query, userID, courseID); err != nil {
		t.Fatalf("could not insert the drop course row: %v", err)
	}
}

func TestDeleteCourseOnlyWhenNothingUsesIt(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	for _, test := range []struct {
		name    string
		code    string
		seed    func(courseID int64)
		wantErr error
		want    CourseUsage
	}{
		{
			name: "an unused course goes",
			code: "CSE 101",
			seed: func(int64) {},
		},
		{
			name:    "a routine entry keeps it",
			code:    "CSE 102",
			seed:    func(courseID int64) { insertRoutineFor(t, ctx, pool, courseID, "T-102") },
			wantErr: ErrCourseInUse,
			want:    CourseUsage{Routines: 1},
		},
		{
			name:    "a class keeps it",
			code:    "CSE 103",
			seed:    func(courseID int64) { insertScheduleItem(t, ctx, pool, courseID) },
			wantErr: ErrCourseInUse,
			want:    CourseUsage{Items: 1},
		},
		{
			name:    "a drop course student keeps it",
			code:    "CSE 104",
			seed:    func(courseID int64) { insertDropCourse(t, ctx, pool, courseID, "2023331104") },
			wantErr: ErrCourseInUse,
			want:    CourseUsage{DropCourses: 1},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			course, err := store.CreateCourse(ctx, test.code, "Test Course")
			if err != nil {
				t.Fatalf("CreateCourse() returned an unexpected error: %v", err)
			}
			test.seed(course.ID)
			usage, err := store.DeleteCourse(ctx, course.ID)
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("DeleteCourse() error = %v, want %v", err, test.wantErr)
			}
			if test.wantErr == nil {
				if _, err := store.DeleteCourse(ctx, course.ID); !errors.Is(err, ErrNotFound) {
					t.Errorf("second DeleteCourse() error = %v, want ErrNotFound", err)
				}
				return
			}
			if usage != test.want {
				t.Errorf("usage = %+v, want %+v", usage, test.want)
			}
		})
	}
}

func TestDeleteCourseKeepsThePublishedCopies(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	course, err := store.CreateCourse(ctx, "CSE 210", "Algorithms")
	if err != nil {
		t.Fatalf("CreateCourse() returned an unexpected error: %v", err)
	}
	insertScheduleItem(t, ctx, pool, course.ID)
	if _, err := store.DeleteCourse(ctx, course.ID); !errors.Is(err, ErrCourseInUse) {
		t.Fatalf("DeleteCourse() error = %v, want ErrCourseInUse", err)
	}
	if _, err := pool.Exec(ctx, "DELETE FROM schedule_items WHERE course_id = $1", course.ID); err != nil {
		t.Fatalf("could not remove the schedule item: %v", err)
	}
	if _, err := store.DeleteCourse(ctx, course.ID); err != nil {
		t.Fatalf("DeleteCourse() returned an unexpected error: %v", err)
	}
	var columns int
	query := `SELECT COUNT(*) FROM information_schema.columns
WHERE table_name = 'published_items' AND column_name IN ('course_code', 'course_title')`
	if err := pool.QueryRow(ctx, query).Scan(&columns); err != nil {
		t.Fatalf("could not read the published_items columns: %v", err)
	}
	if columns != 2 {
		t.Errorf("published_items keeps %d copied course columns, want 2", columns)
	}
}

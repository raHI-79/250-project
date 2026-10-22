package store

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5/pgxpool"
	"testing"
)

func insertAccount(t *testing.T, ctx context.Context, pool *pgxpool.Pool, role, identifier, batch, section string, isCR bool) int64 {
	t.Helper()
	var userID int64
	err := pool.QueryRow(ctx,
		`INSERT INTO users (role, identifier, password_hash, status) VALUES ($1, $2, 'placeholder', 'active') RETURNING id`,
		role, identifier).Scan(&userID)
	if err != nil {
		t.Fatalf("could not insert the %s account %s: %v", role, identifier, err)
	}
	if role == "teacher" {
		_, err = pool.Exec(ctx,
			`INSERT INTO teachers (user_id, teacher_id, name, email, phone) VALUES ($1, $2, $3, $4, '01712345678')`,
			userID, identifier, "Teacher "+identifier, identifier+"@sust.edu")
		if err != nil {
			t.Fatalf("could not insert the teacher row for %s: %v", identifier, err)
		}
		return userID
	}
	_, err = pool.Exec(ctx,
		`INSERT INTO students (user_id, full_name, reg_no, batch, section, email, phone, blood_group, is_cr)
VALUES ($1, $2, $3, $4, $5, $6, '01712345678', 'O+', $7)`,
		userID, "Student "+identifier, identifier, batch, section, identifier+"@student.sust.edu", isCR)
	if err != nil {
		t.Fatalf("could not insert the student row for %s: %v", identifier, err)
	}
	return userID
}

func seedCourseAndTeacher(t *testing.T, ctx context.Context, store *Store, pool *pgxpool.Pool) (int64, int64) {
	t.Helper()
	course, err := store.CreateCourse(ctx, "CSE 233", "Digital Logic Design", 3.0)
	if err != nil {
		t.Fatalf("CreateCourse() returned an unexpected error: %v", err)
	}
	var teacherID int64
	if err := pool.QueryRow(ctx, "SELECT id FROM teachers ORDER BY id LIMIT 1").Scan(&teacherID); err != nil {
		t.Fatalf("could not read the seeded teacher: %v", err)
	}
	return course.ID, teacherID
}

func TestCreateRoutineRejectsRoomClashes(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	insertAccount(t, ctx, pool, "teacher", "T-001", "", "", false)
	courseID, teacherID := seedCourseAndTeacher(t, ctx, store, pool)
	base := NewRoutine{
		Batch:     "CSE-23",
		Semester:  "1-1",
		Section:   "A",
		CourseID:  courseID,
		TeacherID: teacherID,
		DayOfWeek: 1,
		StartTime: "10:30",
		EndTime:   "12:45",
		Room:      "1003",
		Type:      "lab",
	}
	created, err := store.CreateRoutine(ctx, base)
	if err != nil {
		t.Fatalf("CreateRoutine() returned an unexpected error: %v", err)
	}
	if created.ID == 0 || created.CourseCode != "CSE 233" {
		t.Fatalf("created = %+v, want an id and the course code", created)
	}
	tests := []struct {
		name      string
		change    func(NewRoutine) NewRoutine
		wantClash bool
	}{
		{
			name:      "inside the booking",
			change:    func(r NewRoutine) NewRoutine { r.StartTime, r.EndTime = "11:00", "12:00"; return r },
			wantClash: true,
		},
		{
			name:      "overlapping the end",
			change:    func(r NewRoutine) NewRoutine { r.StartTime, r.EndTime = "12:00", "13:30"; return r },
			wantClash: true,
		},
		{
			name:   "right after the booking",
			change: func(r NewRoutine) NewRoutine { r.StartTime, r.EndTime = "12:45", "14:00"; return r },
		},
		{
			name:   "another room",
			change: func(r NewRoutine) NewRoutine { r.Room = "1004"; return r },
		},
		{
			name:   "another day",
			change: func(r NewRoutine) NewRoutine { r.DayOfWeek = 2; return r },
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := store.CreateRoutine(ctx, test.change(base))
			var booked *RoomBookedError
			if test.wantClash {
				if !errors.As(err, &booked) {
					t.Fatalf("error = %v, want a room clash", err)
				}
				if booked.Booking.Room != "1003" || booked.Booking.CourseCode != "CSE 233" || booked.Booking.StartTime != "10:30" {
					t.Errorf("booking = %+v, want the first entry", booked.Booking)
				}
				return
			}
			if err != nil {
				t.Fatalf("CreateRoutine() returned an unexpected error: %v", err)
			}
		})
	}
}

func TestCreateRoutineChecksTheCourseAndTeacher(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	insertAccount(t, ctx, pool, "teacher", "T-001", "", "", false)
	courseID, teacherID := seedCourseAndTeacher(t, ctx, store, pool)
	removedTeacher := insertAccount(t, ctx, pool, "teacher", "T-002", "", "", false)
	if _, err := pool.Exec(ctx, "UPDATE users SET status = 'removed' WHERE id = $1", removedTeacher); err != nil {
		t.Fatalf("could not deactivate the second teacher: %v", err)
	}
	var removedTeacherID int64
	err := pool.QueryRow(ctx, "SELECT id FROM teachers WHERE user_id = $1", removedTeacher).Scan(&removedTeacherID)
	if err != nil {
		t.Fatalf("could not read the second teacher row: %v", err)
	}
	base := NewRoutine{
		Batch:     "CSE-23",
		Semester:  "1-1",
		Section:   "ALL",
		CourseID:  courseID,
		TeacherID: teacherID,
		DayOfWeek: 3,
		StartTime: "09:00",
		EndTime:   "10:00",
		Room:      "2001",
		Type:      "class",
	}
	tests := []struct {
		name    string
		change  func(NewRoutine) NewRoutine
		wantErr error
	}{
		{name: "valid", change: func(r NewRoutine) NewRoutine { return r }},
		{
			name:    "unknown course",
			change:  func(r NewRoutine) NewRoutine { r.CourseID = 9999; return r },
			wantErr: ErrCourseNotFound,
		},
		{
			name:    "unknown teacher",
			change:  func(r NewRoutine) NewRoutine { r.TeacherID = 9999; r.Room = "2002"; return r },
			wantErr: ErrTeacherNotFound,
		},
		{
			name:    "removed teacher",
			change:  func(r NewRoutine) NewRoutine { r.TeacherID = removedTeacherID; r.Room = "2003"; return r },
			wantErr: ErrTeacherInactive,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := store.CreateRoutine(ctx, test.change(base))
			if test.wantErr != nil {
				if !errors.Is(err, test.wantErr) {
					t.Fatalf("error = %v, want %v", err, test.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("CreateRoutine() returned an unexpected error: %v", err)
			}
		})
	}
}

func TestDeleteRoutine(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	insertAccount(t, ctx, pool, "teacher", "T-001", "", "", false)
	courseID, teacherID := seedCourseAndTeacher(t, ctx, store, pool)
	created, err := store.CreateRoutine(ctx, NewRoutine{
		Batch:     "CSE-23",
		Semester:  "1-1",
		Section:   "A",
		CourseID:  courseID,
		TeacherID: teacherID,
		DayOfWeek: 0,
		StartTime: "08:00",
		EndTime:   "09:00",
		Room:      "1003",
		Type:      "class",
	})
	if err != nil {
		t.Fatalf("CreateRoutine() returned an unexpected error: %v", err)
	}
	if err := store.DeleteRoutine(ctx, created.ID); err != nil {
		t.Fatalf("DeleteRoutine() returned an unexpected error: %v", err)
	}
	if err := store.DeleteRoutine(ctx, created.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("second DeleteRoutine() error = %v, want %v", err, ErrNotFound)
	}
	entries, err := store.ListRoutines(ctx, "", "", "")
	if err != nil {
		t.Fatalf("ListRoutines() returned an unexpected error: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("got %d routine entries, want none", len(entries))
	}
}

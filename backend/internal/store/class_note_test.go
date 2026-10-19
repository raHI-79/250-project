package store

import (
	"context"
	"github.com/jackc/pgx/v5/pgxpool"
	"testing"
	"time"
)

func insertNotedEvent(t *testing.T, ctx context.Context, pool *pgxpool.Pool, fixture noteFixture) int64 {
	t.Helper()
	var id int64
	query := `INSERT INTO class_events
(batch, section, course_id, teacher_id, event_date, start_time, end_time, room, event_type, status,
created_by, note, note_audience)
VALUES ('CSE-23', 'A', $1, $2, $3::date, TIME '10:30', TIME '12:00', $4, $5, $6, $7,
NULLIF($8::text, ''), NULLIF($9::text, ''))
RETURNING id`
	err := pool.QueryRow(ctx, query,
		fixture.courseID,
		fixture.teacherID,
		fixture.date,
		fixture.room,
		fixture.eventType,
		fixture.status,
		fixture.createdBy,
		fixture.note,
		fixture.audience,
	).Scan(&id)
	if err != nil {
		t.Fatalf("could not insert the class event: %v", err)
	}
	return id
}

type noteFixture struct {
	courseID  int64
	teacherID int64
	createdBy int64
	date      string
	room      string
	eventType string
	status    string
	note      string
	audience  string
}

func TestClassEventNoteAudience(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	people := seedChatPeople(t, ctx, pool)
	course, err := store.CreateCourse(ctx, "CSE-233", "Digital Logic Design")
	if err != nil {
		t.Fatalf("CreateCourse() returned an unexpected error: %v", err)
	}
	var teacherRowID int64
	err = pool.QueryRow(ctx, "SELECT id FROM teachers WHERE user_id = $1", people["teacher"]).Scan(&teacherRowID)
	if err != nil {
		t.Fatalf("could not read the teacher row: %v", err)
	}
	now := time.Now()
	date := now.AddDate(0, 0, 1).Format("2006-01-02")
	tests := []struct {
		name            string
		audience        string
		status          string
		wantForTeacher  bool
		wantForStudents bool
	}{
		{name: "for the teacher only", audience: "teacher", status: "approved", wantForTeacher: true},
		{name: "for the students only", audience: "students", status: "approved", wantForStudents: true},
		{
			name:            "for both",
			audience:        "both",
			status:          "approved",
			wantForTeacher:  true,
			wantForStudents: true,
		},
		{
			name:           "students wait for the approval",
			audience:       "both",
			status:         "pending",
			wantForTeacher: true,
		},
		{name: "no note at all", audience: "", status: "approved"},
	}
	for index, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			note := ""
			if tc.audience != "" {
				note = "Bring the lab sheet"
			}
			room := "110" + string(rune('0'+index))
			eventID := insertNotedEvent(t, ctx, pool, noteFixture{
				courseID:  course.ID,
				teacherID: teacherRowID,
				createdBy: people["cr"],
				date:      date,
				room:      room,
				eventType: "class",
				status:    tc.status,
				note:      note,
				audience:  tc.audience,
			})
			requests, err := store.ListClassRequests(ctx, people["cr"], "")
			if err != nil {
				t.Fatalf("ListClassRequests() returned an unexpected error: %v", err)
			}
			var mine ClassEventRow
			for _, request := range requests {
				if request.ID == eventID {
					mine = request
				}
			}
			if mine.Note != note {
				t.Errorf("the class representative sees note %q, want %q", mine.Note, note)
			}
			if mine.NoteAudience != tc.audience {
				t.Errorf("audience = %q, want %q", mine.NoteAudience, tc.audience)
			}
			teacherNote := ""
			if tc.status == "approved" {
				classes, err := store.ListTeacherClasses(ctx, teacherRowID, date, now)
				if err != nil {
					t.Fatalf("ListTeacherClasses() returned an unexpected error: %v", err)
				}
				for _, class := range classes {
					if class.ID == eventID {
						teacherNote = class.Note
					}
				}
			} else {
				pending, err := store.ListTeacherRequests(ctx, teacherRowID)
				if err != nil {
					t.Fatalf("ListTeacherRequests() returned an unexpected error: %v", err)
				}
				for _, request := range pending {
					if request.ID == eventID {
						teacherNote = request.Note
					}
				}
			}
			if (teacherNote != "") != tc.wantForTeacher {
				t.Errorf("the teacher sees note %q, want visible = %t", teacherNote, tc.wantForTeacher)
			}
			plan, err := store.ListDayPlan(ctx, "CSE-23", date, nil)
			if err != nil {
				t.Fatalf("ListDayPlan() returned an unexpected error: %v", err)
			}
			studentNote := ""
			for _, item := range plan {
				if item.Room == room {
					studentNote = item.Note
				}
			}
			if (studentNote != "") != tc.wantForStudents {
				t.Errorf("the students see note %q, want visible = %t", studentNote, tc.wantForStudents)
			}
			if _, err := pool.Exec(ctx, "DELETE FROM class_events WHERE id = $1", eventID); err != nil {
				t.Fatalf("could not clean up the class event: %v", err)
			}
		})
	}
}

func TestExamNoteAudience(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	people := seedChatPeople(t, ctx, pool)
	course, err := store.CreateCourse(ctx, "CSE-245", "Databases")
	if err != nil {
		t.Fatalf("CreateCourse() returned an unexpected error: %v", err)
	}
	var teacherRowID int64
	err = pool.QueryRow(ctx, "SELECT id FROM teachers WHERE user_id = $1", people["teacher"]).Scan(&teacherRowID)
	if err != nil {
		t.Fatalf("could not read the teacher row: %v", err)
	}
	now := time.Now()
	date := now.AddDate(0, 0, 2).Format("2006-01-02")
	tests := []struct {
		name     string
		audience string
		wantNote bool
	}{
		{name: "students see a students note", audience: "students", wantNote: true},
		{name: "students see a both note", audience: "both", wantNote: true},
		{name: "students never see a teacher note", audience: "teacher"},
	}
	for index, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			room := "220" + string(rune('0'+index))
			eventID := insertNotedEvent(t, ctx, pool, noteFixture{
				courseID:  course.ID,
				teacherID: teacherRowID,
				createdBy: people["cr"],
				date:      date,
				room:      room,
				eventType: "exam",
				status:    "approved",
				note:      "Open book",
				audience:  tc.audience,
			})
			exams, err := store.ListExams(ctx, "CSE-23", "A", now)
			if err != nil {
				t.Fatalf("ListExams() returned an unexpected error: %v", err)
			}
			note := ""
			for _, exam := range exams {
				if exam.Room == room {
					note = exam.Note
				}
			}
			if (note != "") != tc.wantNote {
				t.Errorf("the students see note %q, want visible = %t", note, tc.wantNote)
			}
			if _, err := pool.Exec(ctx, "DELETE FROM class_events WHERE id = $1", eventID); err != nil {
				t.Fatalf("could not clean up the class event: %v", err)
			}
		})
	}
}

func TestTeacherRequestNoteAudience(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	people := seedChatPeople(t, ctx, pool)
	course, err := store.CreateCourse(ctx, "CSE-250", "Software Engineering")
	if err != nil {
		t.Fatalf("CreateCourse() returned an unexpected error: %v", err)
	}
	var teacherRowID int64
	err = pool.QueryRow(ctx, "SELECT id FROM teachers WHERE user_id = $1", people["teacher"]).Scan(&teacherRowID)
	if err != nil {
		t.Fatalf("could not read the teacher row: %v", err)
	}
	date := time.Now().AddDate(0, 0, 3).Format("2006-01-02")
	tests := []struct {
		name     string
		audience string
		wantNote bool
	}{
		{name: "a teacher note reaches the teacher", audience: "teacher", wantNote: true},
		{name: "a both note reaches the teacher", audience: "both", wantNote: true},
		{name: "a students note stays hidden", audience: "students"},
	}
	for index, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			room := "330" + string(rune('0'+index))
			eventID := insertNotedEvent(t, ctx, pool, noteFixture{
				courseID:  course.ID,
				teacherID: teacherRowID,
				createdBy: people["cr"],
				date:      date,
				room:      room,
				eventType: "class",
				status:    "pending",
				note:      "The projector is broken",
				audience:  tc.audience,
			})
			requests, err := store.ListTeacherRequests(ctx, teacherRowID)
			if err != nil {
				t.Fatalf("ListTeacherRequests() returned an unexpected error: %v", err)
			}
			note := ""
			for _, request := range requests {
				if request.ID == eventID {
					note = request.Note
				}
			}
			if (note != "") != tc.wantNote {
				t.Errorf("the teacher sees note %q, want visible = %t", note, tc.wantNote)
			}
			if _, err := pool.Exec(ctx, "DELETE FROM class_events WHERE id = $1", eventID); err != nil {
				t.Fatalf("could not clean up the class event: %v", err)
			}
		})
	}
}

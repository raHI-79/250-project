package store

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5/pgxpool"
	"math"
	"testing"
	"time"
)

func newStudent(t *testing.T, ctx context.Context, pool *pgxpool.Pool, regNo, batch, section string) int64 {
	t.Helper()
	user := insertUser(t, ctx, pool, RoleStudent, regNo, StatusActive, 0)
	return insertStudentRow(t, ctx, pool, user, regNo, batch, section)
}

func heldSession(t *testing.T, ctx context.Context, pool *pgxpool.Pool, class classFixture,
	batch, section string, daysAgo, hours int) int64 {
	t.Helper()
	var id int64
	err := pool.QueryRow(ctx, `INSERT INTO attendance_sessions
(course_id, batch, section, teacher_id, started_at, ends_at, original_ends_at, status, closed_at, token_key, hours)
VALUES ($1, $2, $3, $4, now() - make_interval(days => $5), now() - make_interval(days => $5) + interval '10 minutes',
        now() - make_interval(days => $5) + interval '10 minutes', 'closed', now(), 'held-key', $6)
RETURNING id`, class.courseID, batch, section, class.teacherID, daysAgo, hours).Scan(&id)
	if err != nil {
		t.Fatalf("could not insert a held session: %v", err)
	}
	return id
}

func record(t *testing.T, ctx context.Context, pool *pgxpool.Pool, sessionID, studentID int64,
	status, method string) {
	t.Helper()
	_, err := pool.Exec(ctx, `INSERT INTO attendance_records (session_id, student_id, status, method)
VALUES ($1, $2, $3, $4)`, sessionID, studentID, status, method)
	if err != nil {
		t.Fatalf("could not insert a %s record: %v", status, err)
	}
}

func digitalMark(t *testing.T, ctx context.Context, store *Store, sessionID, studentID int64,
	device string) string {
	t.Helper()
	status, err := store.MarkAttendance(ctx, MarkInput{
		SessionID: sessionID, StudentID: studentID, Method: "app", DeviceID: device,
		Lat: 24.9169, Lng: 91.8316, AccuracyM: 10, DistanceM: 8,
	})
	if err != nil {
		t.Fatalf("MarkAttendance() returned an unexpected error: %v", err)
	}
	return status
}

func TestMarksAfterTheLateBoundaryAreLate(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	class := newClass(t, ctx, store, pool, "CSE-23", "A")
	onTime := newStudent(t, ctx, pool, "2023331001", "CSE-23", "A")
	afterExtension := newStudent(t, ctx, pool, "2023331002", "CSE-23", "A")
	afterLateMinutes := newStudent(t, ctx, pool, "2023331003", "CSE-23", "A")
	beforeLateMinutes := newStudent(t, ctx, pool, "2023331004", "CSE-23", "A")
	if got := digitalMark(t, ctx, store, class.sessionID, onTime, "phone-1"); got != "present" {
		t.Fatalf("a mark before the original end is %q, want present", got)
	}
	if _, err := store.ExtendAttendanceSession(ctx, class.sessionID, class.teacherID, 15); err != nil {
		t.Fatalf("ExtendAttendanceSession() returned an unexpected error: %v", err)
	}
	if _, err := pool.Exec(ctx, `UPDATE attendance_sessions
SET original_ends_at = now() - interval '1 second' WHERE id = $1`, class.sessionID); err != nil {
		t.Fatalf("could not move the original end into the past: %v", err)
	}
	if got := digitalMark(t, ctx, store, class.sessionID, afterExtension, "phone-2"); got != "late" {
		t.Fatalf("a mark after the original end of an extended session is %q, want late", got)
	}
	if _, err := pool.Exec(ctx, `UPDATE attendance_sessions
SET original_ends_at = now() + interval '5 minutes', late_after = now() - interval '1 second'
WHERE id = $1`, class.sessionID); err != nil {
		t.Fatalf("could not set the late boundary: %v", err)
	}
	if got := digitalMark(t, ctx, store, class.sessionID, afterLateMinutes, "phone-3"); got != "late" {
		t.Fatalf("a mark after late-after is %q, want late", got)
	}
	if _, err := pool.Exec(ctx, `UPDATE attendance_sessions
SET late_after = now() + interval '2 minutes' WHERE id = $1`, class.sessionID); err != nil {
		t.Fatalf("could not move the late boundary: %v", err)
	}
	if got := digitalMark(t, ctx, store, class.sessionID, beforeLateMinutes, "phone-4"); got != "present" {
		t.Fatalf("a mark before late-after is %q, want present", got)
	}
	session, err := store.AttendanceSession(ctx, class.sessionID)
	if err != nil {
		t.Fatalf("AttendanceSession() returned an unexpected error: %v", err)
	}
	if session.Present != 4 || session.Late != 2 {
		t.Fatalf("the session counts %d attending and %d late, want 4 and 2", session.Present, session.Late)
	}
}

func TestASessionCarriesItsHoursLateAfterAndFourHourLimit(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	courseID := insertCourseRow(t, ctx, pool, "CSE245", "Databases")
	teacherUser := insertUser(t, ctx, pool, RoleTeacher, "T-500", StatusActive, 0)
	teacherID := insertTeacherRow(t, ctx, pool, teacherUser, "T-500", "Main Teacher")
	session, err := store.OpenAttendanceSession(ctx, NewAttendanceSession{
		CourseID: courseID, Batch: "CSE-23", Section: "A", TeacherID: teacherID,
		Minutes: 47, Hours: 3, LateAfter: 12, TokenKey: "custom-key",
	})
	if err != nil {
		t.Fatalf("OpenAttendanceSession() returned an unexpected error: %v", err)
	}
	if session.Hours != 3 {
		t.Fatalf("the session is worth %d hours, want 3", session.Hours)
	}
	if minutes := math.Round(session.EndsAt.Sub(session.StartedAt).Minutes()); minutes != 47 {
		t.Fatalf("a custom duration of 47 minutes became %v", minutes)
	}
	if minutes := math.Round(session.LateAfter.Sub(session.StartedAt).Minutes()); minutes != 12 {
		t.Fatalf("late after 12 minutes became %v", minutes)
	}
	if !session.OriginalEndsAt.Equal(session.EndsAt) {
		t.Fatalf("the original end %v differs from the end %v", session.OriginalEndsAt, session.EndsAt)
	}
	longer, err := store.ExtendAttendanceSession(ctx, session.ID, teacherID, 7)
	if err != nil {
		t.Fatalf("ExtendAttendanceSession() returned an unexpected error: %v", err)
	}
	if minutes := math.Round(longer.EndsAt.Sub(longer.StartedAt).Minutes()); minutes != 54 {
		t.Fatalf("a custom extension of 7 minutes ends after %v minutes", minutes)
	}
	if !longer.OriginalEndsAt.Equal(session.OriginalEndsAt) {
		t.Fatal("extending moved the original end, so nobody could be late")
	}
	if _, err := store.ExtendAttendanceSession(ctx, session.ID, teacherID, 200); !errors.Is(err, ErrAttendanceClosed) {
		t.Fatalf("an extension past four hours returned %v", err)
	}
	plain, err := store.OpenAttendanceSession(ctx, NewAttendanceSession{
		CourseID: courseID, Batch: "CSE-23", Section: "B", TeacherID: teacherID,
		Minutes: 10, TokenKey: "plain-key",
	})
	if err != nil {
		t.Fatalf("OpenAttendanceSession() returned an unexpected error: %v", err)
	}
	if plain.Hours != 1 || !plain.LateAfter.IsZero() {
		t.Fatalf("a plain session has %d hours and late after %v", plain.Hours, plain.LateAfter)
	}
}

func TestCreditWeighsHoursAndLates(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	class := newClass(t, ctx, store, pool, "CSE-23", "A")
	student := newStudent(t, ctx, pool, "2023331001", "CSE-23", "A")
	other := newStudent(t, ctx, pool, "2023331002", "CSE-23", "A")
	twoHours := heldSession(t, ctx, pool, class, "CSE-23", "A", 3, 2)
	threeHours := heldSession(t, ctx, pool, class, "CSE-23", "A", 2, 3)
	oneHour := heldSession(t, ctx, pool, class, "CSE-23", "A", 1, 1)
	record(t, ctx, pool, twoHours, student, "present", "app")
	record(t, ctx, pool, threeHours, student, "late", "app")
	record(t, ctx, pool, oneHour, student, "absent", "system")
	for _, id := range []int64{twoHours, threeHours, oneHour} {
		record(t, ctx, pool, id, other, "absent", "manual")
	}
	courses, err := store.StudentCourseAttendance(ctx, student)
	if err != nil || len(courses) != 1 {
		t.Fatalf("StudentCourseAttendance() = %v, %v", courses, err)
	}
	got := courses[0]
	if got.Held != 6 || got.Present != 2 || got.Lates != 1 || got.Credit != 3.5 || got.Sessions != 3 {
		t.Fatalf("the course counts %+v, want 6 held, 2 present hours, 1 late, 3.5 credit, 3 sessions", got)
	}
	low, err := store.ListLowAttendance(ctx, 3, 50)
	if err != nil {
		t.Fatalf("ListLowAttendance() returned an unexpected error: %v", err)
	}
	if len(low) != 1 || low[0].StudentID != other {
		t.Fatalf("the students under 50 %% are %+v, want only the absent one", low)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO settings (key, value) VALUES ('attendance_late_weight', '0')
ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value`); err != nil {
		t.Fatalf("could not set the late weight: %v", err)
	}
	courses, err = store.StudentCourseAttendance(ctx, student)
	if err != nil || courses[0].Credit != 2 {
		t.Fatalf("with a late weight of 0 the credit is %v, %v, want 2", courses, err)
	}
	low, err = store.ListLowAttendance(ctx, 3, 50)
	if err != nil || len(low) != 2 {
		t.Fatalf("with a late weight of 0 %d students are under 50 %%, want 2 (%v)", len(low), err)
	}
	rows, err := store.BatchExportRows(ctx, class.courseID, "CSE-23", "A",
		courseWindowStart(), courseWindowEnd())
	if err != nil || len(rows) != 6 {
		t.Fatalf("BatchExportRows() = %d rows, %v, want 6", len(rows), err)
	}
	hours := map[int64]int{}
	for _, row := range rows {
		hours[row.SessionID] = row.Hours
		if row.LateWeight != 0 {
			t.Fatalf("the export carries the late weight %v, want 0", row.LateWeight)
		}
	}
	if hours[twoHours] != 2 || hours[threeHours] != 3 || hours[oneHour] != 1 {
		t.Fatalf("the export carries the hours %v", hours)
	}
}

func TestAStudentWhoJoinsLaterIsAbsentFromTheEarlierClasses(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	class := newClass(t, ctx, store, pool, "CSE-23", "A")
	early := newStudent(t, ctx, pool, "2023331001", "CSE-23", "A")
	if _, err := pool.Exec(ctx, `INSERT INTO batches (name, running_semester, semester_since)
VALUES ('CSE-23', '3-1', now() - interval '10 days')
ON CONFLICT (name) DO UPDATE SET semester_since = EXCLUDED.semester_since`); err != nil {
		t.Fatalf("could not start the semester: %v", err)
	}
	lastSemester := heldSession(t, ctx, pool, class, "CSE-23", "A", 30, 1)
	first := heldSession(t, ctx, pool, class, "CSE-23", "A", 6, 2)
	second := heldSession(t, ctx, pool, class, "CSE-23", "A", 4, 1)
	empty := heldSession(t, ctx, pool, class, "CSE-23", "A", 3, 1)
	otherSection := heldSession(t, ctx, pool, class, "CSE-23", "B", 2, 1)
	for _, id := range []int64{lastSemester, first, second} {
		record(t, ctx, pool, id, early, "present", "app")
	}
	sectionB := newStudent(t, ctx, pool, "2023331009", "CSE-23", "B")
	record(t, ctx, pool, otherSection, sectionB, "present", "app")
	joiner := newStudent(t, ctx, pool, "2023331002", "CSE-23", "A")
	counted, err := store.BackfillJoinedLater(ctx, joiner, 0)
	if err != nil {
		t.Fatalf("BackfillJoinedLater() returned an unexpected error: %v", err)
	}
	if counted != 2 {
		t.Fatalf("counted %d earlier classes, want the 2 held this semester", counted)
	}
	again, err := store.BackfillJoinedLater(ctx, 0, 0)
	if err != nil || again != 0 {
		t.Fatalf("a second backfill added %d records, %v", again, err)
	}
	rows, err := store.StudentAttendanceSessions(ctx, joiner)
	if err != nil || len(rows) != 2 {
		t.Fatalf("the late joiner has %d classes, %v, want 2", len(rows), err)
	}
	for _, row := range rows {
		if row.Status != "absent" || row.Method != "system" || row.Note != "Joined later" {
			t.Fatalf("an earlier class is recorded as %+v", row)
		}
		if row.SessionID == lastSemester || row.SessionID == empty || row.SessionID == otherSection {
			t.Fatalf("the class %d should not count for the late joiner", row.SessionID)
		}
	}
	courses, err := store.StudentCourseAttendance(ctx, joiner)
	if err != nil || len(courses) != 1 || courses[0].Held != 3 || courses[0].Credit != 0 {
		t.Fatalf("the late joiner counts %+v, %v, want 3 hours held and no credit", courses, err)
	}
	marks, err := store.ListSessionRecords(ctx, first)
	if err != nil {
		t.Fatalf("ListSessionRecords() returned an unexpected error: %v", err)
	}
	listed := false
	for _, mark := range marks {
		if mark.StudentID == joiner && mark.Status == "absent" {
			listed = true
		}
	}
	if !listed {
		t.Fatal("the late joiner is missing from the roster of an earlier class")
	}
	if err := store.EditAttendanceRecord(ctx, first, joiner, class.teacherUA, "late", "Was in the class"); err != nil {
		t.Fatalf("EditAttendanceRecord() returned an unexpected error: %v", err)
	}
	courses, err = store.StudentCourseAttendance(ctx, joiner)
	if err != nil || courses[0].Credit != 1 || courses[0].Lates != 1 {
		t.Fatalf("after the teacher changed a class to late the joiner counts %+v, %v", courses, err)
	}
	dropCourse := newStudent(t, ctx, pool, "2022331050", "CSE-22", "A")
	addDropCourse(t, ctx, pool, dropCourse, class.courseID)
	counted, err = store.BackfillJoinedLater(ctx, dropCourse, 0)
	if err != nil || counted != 3 {
		t.Fatalf("a drop course added mid-semester counted %d earlier classes, %v, want 3", counted, err)
	}
}

func courseWindowStart() time.Time {
	return time.Now().AddDate(0, 0, -60)
}

func courseWindowEnd() time.Time {
	return time.Now().AddDate(0, 0, 1)
}

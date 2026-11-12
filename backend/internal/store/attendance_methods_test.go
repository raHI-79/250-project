package store

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5/pgxpool"
	"testing"
	"time"
)

func newRollCall(t *testing.T, ctx context.Context, store *Store, class classFixture,
	batch, section string) AttendanceSession {
	t.Helper()
	session, err := store.OpenAttendanceSession(ctx, NewAttendanceSession{
		CourseID:  class.courseID,
		Batch:     batch,
		Section:   section,
		TeacherID: class.teacherID,
		Minutes:   10,
		TokenKey:  "roll-call-key",
		Method:    SessionRollCall,
	})
	if err != nil {
		t.Fatalf("OpenAttendanceSession() returned an unexpected error: %v", err)
	}
	return session
}

func ageSession(t *testing.T, ctx context.Context, pool *pgxpool.Pool, sessionID int64, days int) {
	t.Helper()
	query := `UPDATE attendance_sessions
SET started_at = started_at - make_interval(days => $2),
    ends_at = ends_at - make_interval(days => $2),
    closed_at = closed_at - make_interval(days => $2),
    location_updated_at = location_updated_at - make_interval(days => $2)
WHERE id = $1`
	if _, err := pool.Exec(ctx, query, sessionID, days); err != nil {
		t.Fatalf("could not move the session %d back %d days: %v", sessionID, days, err)
	}
}

func markRow(t *testing.T, ctx context.Context, pool *pgxpool.Pool,
	sessionID, studentID int64, status, method string) {
	t.Helper()
	query := `INSERT INTO attendance_records (session_id, student_id, status, method)
VALUES ($1, $2, $3, $4)`
	if _, err := pool.Exec(ctx, query, sessionID, studentID, status, method); err != nil {
		t.Fatalf("could not insert a %s mark: %v", method, err)
	}
}

func TestARollCallSessionKeepsItsMethod(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	class := newClass(t, ctx, store, pool, "CSE-23", "A")
	bluetooth, err := store.AttendanceSession(ctx, class.sessionID)
	if err != nil {
		t.Fatalf("AttendanceSession() returned an unexpected error: %v", err)
	}
	if bluetooth.Method != SessionBluetooth {
		t.Errorf("method = %q, want %q by default", bluetooth.Method, SessionBluetooth)
	}
	userID := insertUser(t, ctx, pool, RoleStudent, "2023331100", StatusActive, 0)
	studentID := insertStudentRow(t, ctx, pool, userID, "2023331100", "CSE-23", "A")
	markRow(t, ctx, pool, class.sessionID, studentID, "present", "app")
	closed, err := store.CloseAttendanceSession(ctx, class.sessionID, class.teacherID)
	if err != nil {
		t.Fatalf("CloseAttendanceSession() returned an unexpected error: %v", err)
	}
	if closed.Status != "closed" {
		t.Fatalf("status = %q, want closed", closed.Status)
	}
	ageSession(t, ctx, pool, class.sessionID, 1)
	roll := newRollCall(t, ctx, store, class, "CSE-23", "A")
	if roll.Method != SessionRollCall {
		t.Errorf("method = %q, want %q", roll.Method, SessionRollCall)
	}
}

func TestARollCallSetsEveryStudentWithANote(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	class := newClass(t, ctx, store, pool, "CSE-23", "A")
	if _, err := store.CloseAttendanceSession(ctx, class.sessionID, class.teacherID); !errors.Is(err, ErrAttendanceEmpty) {
		t.Fatalf("closing a session nobody marked returned %v, want ErrAttendanceEmpty", err)
	}
	first := insertUser(t, ctx, pool, RoleStudent, "2023331101", StatusActive, 0)
	firstID := insertStudentRow(t, ctx, pool, first, "2023331101", "CSE-23", "A")
	second := insertUser(t, ctx, pool, RoleStudent, "2023331102", StatusActive, 0)
	secondID := insertStudentRow(t, ctx, pool, second, "2023331102", "CSE-23", "A")
	roll := newRollCall(t, ctx, store, class, "CSE-23", "A")
	if err := store.EditAttendanceRecord(ctx, roll.ID, firstID, class.teacherUA, "present", "on time"); err != nil {
		t.Fatalf("EditAttendanceRecord() returned an unexpected error: %v", err)
	}
	if err := store.EditAttendanceRecord(ctx, roll.ID, secondID, class.teacherUA, "absent", ""); err != nil {
		t.Fatalf("EditAttendanceRecord() returned an unexpected error: %v", err)
	}
	if err := store.UpdateAttendanceNote(ctx, roll.ID, secondID, class.teacherUA, "sick leave"); err != nil {
		t.Fatalf("UpdateAttendanceNote() returned an unexpected error: %v", err)
	}
	records, err := store.ListSessionRecords(ctx, roll.ID)
	if err != nil {
		t.Fatalf("ListSessionRecords() returned an unexpected error: %v", err)
	}
	byStudent := make(map[int64]AttendanceRecord)
	for _, record := range records {
		byStudent[record.StudentID] = record
	}
	if got := byStudent[firstID]; got.Status != "present" || got.Note != "on time" {
		t.Errorf("the first student = %+v, want present with the note", got)
	}
	if got := byStudent[secondID]; got.Status != "absent" || got.Note != "sick leave" {
		t.Errorf("the second student = %+v, want absent with the later note", got)
	}
	if err := store.UpdateAttendanceNote(ctx, roll.ID, 999999, class.teacherUA, "nobody"); !errors.Is(err, ErrNotFound) {
		t.Errorf("a note for an unmarked student returned %v, want ErrNotFound", err)
	}
}

func TestMarkAllPresentFillsTheRosterOnce(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	class := newClass(t, ctx, store, pool, "CSE-23", "A")
	first := insertUser(t, ctx, pool, RoleStudent, "2023331111", StatusActive, 0)
	firstID := insertStudentRow(t, ctx, pool, first, "2023331111", "CSE-23", "A")
	second := insertUser(t, ctx, pool, RoleStudent, "2023331112", StatusActive, 0)
	insertStudentRow(t, ctx, pool, second, "2023331112", "CSE-23", "A")
	if err := store.EditAttendanceRecord(ctx, class.sessionID, firstID, class.teacherUA, "absent", "late"); err != nil {
		t.Fatalf("EditAttendanceRecord() returned an unexpected error: %v", err)
	}
	added, err := store.MarkAllPresent(ctx, class.sessionID, class.teacherUA)
	if err != nil {
		t.Fatalf("MarkAllPresent() returned an unexpected error: %v", err)
	}
	if added != 1 {
		t.Errorf("added = %d, want only the unmarked student", added)
	}
	records, err := store.ListSessionRecords(ctx, class.sessionID)
	if err != nil {
		t.Fatalf("ListSessionRecords() returned an unexpected error: %v", err)
	}
	for _, record := range records {
		if record.StudentID == firstID && record.Status != "absent" {
			t.Error("Mark all present overwrote a student the teacher had already set absent")
		}
	}
}

func TestEveryMethodLandsInOneSheet(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	class := newClass(t, ctx, store, pool, "CSE-23", "A")
	userID := insertUser(t, ctx, pool, RoleStudent, "2023331121", StatusActive, 0)
	studentID := insertStudentRow(t, ctx, pool, userID, "2023331121", "CSE-23", "A")
	markRow(t, ctx, pool, class.sessionID, studentID, "present", "app")
	if _, err := store.CloseAttendanceSession(ctx, class.sessionID, class.teacherID); err != nil {
		t.Fatalf("CloseAttendanceSession() returned an unexpected error: %v", err)
	}
	ageSession(t, ctx, pool, class.sessionID, 2)
	second := newRollCall(t, ctx, store, class, "CSE-23", "A")
	markRow(t, ctx, pool, second.ID, studentID, "present", "manual")
	if _, err := store.CloseAttendanceSession(ctx, second.ID, class.teacherID); err != nil {
		t.Fatalf("CloseAttendanceSession() returned an unexpected error: %v", err)
	}
	ageSession(t, ctx, pool, second.ID, 1)
	third, err := store.OpenAttendanceSession(ctx, NewAttendanceSession{
		CourseID:  class.courseID,
		Batch:     "CSE-23",
		Section:   "A",
		TeacherID: class.teacherID,
		Minutes:   10,
		TokenKey:  "web-key",
	})
	if err != nil {
		t.Fatalf("OpenAttendanceSession() returned an unexpected error: %v", err)
	}
	markRow(t, ctx, pool, third.ID, studentID, "present", "web")
	from := time.Now().AddDate(0, 0, -3)
	to := time.Now().AddDate(0, 0, 1)
	classes, err := store.ListTeacherClasses(ctx, class.teacherID, from, to)
	if err != nil {
		t.Fatalf("ListTeacherClasses() returned an unexpected error: %v", err)
	}
	if len(classes) != 1 {
		t.Fatalf("sheets = %d, want one sheet for the course, batch and section", len(classes))
	}
	if classes[0].Sessions != 3 {
		t.Errorf("sessions = %d, want the Bluetooth, the roll call and the web session", classes[0].Sessions)
	}
	rows, err := store.TeacherExportRows(ctx, class.teacherID, class.courseID, "CSE-23", "A", from, to)
	if err != nil {
		t.Fatalf("TeacherExportRows() returned an unexpected error: %v", err)
	}
	if len(rows) != 3 {
		t.Errorf("export rows = %d, want one per session for the student", len(rows))
	}
}

func TestDeletingASessionRecountsTheStudent(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	class := newClass(t, ctx, store, pool, "CSE-23", "A")
	userID := insertUser(t, ctx, pool, RoleStudent, "2023331131", StatusActive, 0)
	studentID := insertStudentRow(t, ctx, pool, userID, "2023331131", "CSE-23", "A")
	markRow(t, ctx, pool, class.sessionID, studentID, "present", "app")
	if _, err := store.CloseAttendanceSession(ctx, class.sessionID, class.teacherID); err != nil {
		t.Fatalf("CloseAttendanceSession() returned an unexpected error: %v", err)
	}
	ageSession(t, ctx, pool, class.sessionID, 1)
	second := newRollCall(t, ctx, store, class, "CSE-23", "A")
	markRow(t, ctx, pool, second.ID, studentID, "absent", "manual")
	before, err := store.StudentCourseAttendance(ctx, studentID)
	if err != nil {
		t.Fatalf("StudentCourseAttendance() returned an unexpected error: %v", err)
	}
	if len(before) != 1 || before[0].Held != 2 || before[0].Present != 1 {
		t.Fatalf("before the delete = %+v, want two classes with one present", before)
	}
	if err := store.DeleteAttendanceSession(ctx, second.ID); err != nil {
		t.Fatalf("DeleteAttendanceSession() returned an unexpected error: %v", err)
	}
	after, err := store.StudentCourseAttendance(ctx, studentID)
	if err != nil {
		t.Fatalf("StudentCourseAttendance() returned an unexpected error: %v", err)
	}
	if len(after) != 1 || after[0].Held != 1 || after[0].Present != 1 {
		t.Fatalf("after the delete = %+v, want one class with one present", after)
	}
	sessions, err := store.StudentAttendanceSessions(ctx, studentID)
	if err != nil {
		t.Fatalf("StudentAttendanceSessions() returned an unexpected error: %v", err)
	}
	for _, row := range sessions {
		if row.SessionID == second.ID {
			t.Error("the student still sees the deleted session")
		}
	}
	if err := store.DeleteAttendanceSession(ctx, second.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("deleting the same session twice returned %v, want ErrNotFound", err)
	}
}

func TestDeletingASheetTakesEverySession(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	class := newClass(t, ctx, store, pool, "CSE-23", "A")
	userID := insertUser(t, ctx, pool, RoleStudent, "2023331141", StatusActive, 0)
	studentID := insertStudentRow(t, ctx, pool, userID, "2023331141", "CSE-23", "A")
	markRow(t, ctx, pool, class.sessionID, studentID, "present", "app")
	if _, err := store.CloseAttendanceSession(ctx, class.sessionID, class.teacherID); err != nil {
		t.Fatalf("CloseAttendanceSession() returned an unexpected error: %v", err)
	}
	ageSession(t, ctx, pool, class.sessionID, 1)
	second := newRollCall(t, ctx, store, class, "CSE-23", "A")
	deleted, err := store.DeleteAttendanceSheet(ctx, class.courseID, "CSE-23", "A")
	if err != nil {
		t.Fatalf("DeleteAttendanceSheet() returned an unexpected error: %v", err)
	}
	if deleted != 2 {
		t.Errorf("deleted = %d sessions, want both of the sheet", deleted)
	}
	if _, err := store.AttendanceSession(ctx, second.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("the roll call survived the sheet delete: %v", err)
	}
	if _, err := store.DeleteAttendanceSheet(ctx, class.courseID, "CSE-23", "A"); !errors.Is(err, ErrNotFound) {
		t.Errorf("deleting an empty sheet returned %v, want ErrNotFound", err)
	}
}

func TestHistoryShowsPresentOverTheRosterTotal(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	class := newClass(t, ctx, store, pool, "CSE-23", "A")
	ids := make([]int64, 0, 4)
	for index, regNo := range []string{"2023331151", "2023331152", "2023331153", "2023331154"} {
		userID := insertUser(t, ctx, pool, RoleStudent, regNo, StatusActive, 0)
		ids = append(ids, insertStudentRow(t, ctx, pool, userID, regNo, "CSE-23", "A"))
		if index < 2 {
			markRow(t, ctx, pool, class.sessionID, ids[index], "present", "app")
		}
	}
	from := time.Now().AddDate(0, 0, -1)
	to := time.Now().AddDate(0, 0, 1)
	open, err := store.ListTeacherAttendance(ctx, class.teacherID, from, to)
	if err != nil {
		t.Fatalf("ListTeacherAttendance() returned an unexpected error: %v", err)
	}
	if len(open) != 1 || open[0].Present != 2 || open[0].Roster != 4 {
		t.Fatalf("the open session = %+v, want 2 of a roster of 4", open)
	}
	if _, err := store.CloseAttendanceSession(ctx, class.sessionID, class.teacherID); err != nil {
		t.Fatalf("CloseAttendanceSession() returned an unexpected error: %v", err)
	}
	closed, err := store.ListTeacherAttendance(ctx, class.teacherID, from, to)
	if err != nil {
		t.Fatalf("ListTeacherAttendance() returned an unexpected error: %v", err)
	}
	if len(closed) != 1 || closed[0].Present != 2 || closed[0].Roster != 4 {
		t.Fatalf("the closed session = %+v, want 2 of 4", closed)
	}
	single, err := store.AttendanceSession(ctx, class.sessionID)
	if err != nil {
		t.Fatalf("AttendanceSession() returned an unexpected error: %v", err)
	}
	if single.Present != 2 || single.Roster != 4 {
		t.Errorf("the session = %d/%d, want 2/4", single.Present, single.Roster)
	}
}

func TestASessionNobodyMarkedIsDroppedAndNeverListed(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	class := newClass(t, ctx, store, pool, "CSE-23", "A")
	userID := insertUser(t, ctx, pool, RoleStudent, "2023331161", StatusActive, 0)
	studentID := insertStudentRow(t, ctx, pool, userID, "2023331161", "CSE-23", "A")
	if _, err := store.CloseAttendanceSession(ctx, class.sessionID, class.teacherID); !errors.Is(err, ErrAttendanceEmpty) {
		t.Fatalf("closing an unmarked session returned %v, want ErrAttendanceEmpty", err)
	}
	if _, err := store.AttendanceSession(ctx, class.sessionID); !errors.Is(err, ErrNotFound) {
		t.Errorf("the empty session survived its close: %v", err)
	}
	legacy := newRollCall(t, ctx, store, class, "CSE-23", "A")
	markRow(t, ctx, pool, legacy.ID, studentID, "absent", "system")
	if _, err := pool.Exec(ctx, `UPDATE attendance_sessions SET status = 'closed', closed_at = now() WHERE id = $1`,
		legacy.ID); err != nil {
		t.Fatalf("could not close the legacy session: %v", err)
	}
	listed, err := store.ListTeacherAttendance(ctx, class.teacherID, time.Now().AddDate(0, 0, -1), time.Now().AddDate(0, 0, 1))
	if err != nil {
		t.Fatalf("ListTeacherAttendance() returned an unexpected error: %v", err)
	}
	if len(listed) != 0 {
		t.Errorf("the history lists %+v, want no session without a mark", listed)
	}
}

func TestStartingTheSameClassTheSameDayContinuesIt(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	class := newClass(t, ctx, store, pool, "CSE-23", "A")
	firstUser := insertUser(t, ctx, pool, RoleStudent, "2023331171", StatusActive, 0)
	first := insertStudentRow(t, ctx, pool, firstUser, "2023331171", "CSE-23", "A")
	secondUser := insertUser(t, ctx, pool, RoleStudent, "2023331172", StatusActive, 0)
	insertStudentRow(t, ctx, pool, secondUser, "2023331172", "CSE-23", "A")
	markRow(t, ctx, pool, class.sessionID, first, "present", "app")
	if _, err := store.CloseAttendanceSession(ctx, class.sessionID, class.teacherID); err != nil {
		t.Fatalf("CloseAttendanceSession() returned an unexpected error: %v", err)
	}
	again, err := store.OpenAttendanceSession(ctx, NewAttendanceSession{
		CourseID:  class.courseID,
		Batch:     "CSE-23",
		Section:   "A",
		TeacherID: class.teacherID,
		Minutes:   15,
		TokenKey:  "another-key",
		CodeOnly:  true,
	})
	if err != nil {
		t.Fatalf("OpenAttendanceSession() returned an unexpected error: %v", err)
	}
	if again.ID != class.sessionID {
		t.Fatalf("a second start the same day opened %d, want the morning session %d", again.ID, class.sessionID)
	}
	if again.Status != SessionOpen || !again.CodeOnly {
		t.Errorf("the continued session = %+v, want it open again as code only", again)
	}
	if again.Present != 1 || again.Roster != 2 {
		t.Errorf("the continued session = %d/%d, want the earlier mark kept of 2", again.Present, again.Roster)
	}
	records, err := store.ListSessionRecords(ctx, again.ID)
	if err != nil {
		t.Fatalf("ListSessionRecords() returned an unexpected error: %v", err)
	}
	for _, record := range records {
		if record.Method == "system" {
			t.Errorf("the absentees of the first close are still there: %+v", record)
		}
	}
	ageSession(t, ctx, pool, again.ID, 1)
	if _, err := pool.Exec(ctx, `UPDATE attendance_sessions SET status = 'closed', closed_at = ends_at WHERE id = $1`,
		again.ID); err != nil {
		t.Fatalf("could not close yesterday's session: %v", err)
	}
	tomorrow, err := store.OpenAttendanceSession(ctx, NewAttendanceSession{
		CourseID:  class.courseID,
		Batch:     "CSE-23",
		Section:   "A",
		TeacherID: class.teacherID,
		Minutes:   10,
		TokenKey:  "next-day-key",
	})
	if err != nil {
		t.Fatalf("OpenAttendanceSession() returned an unexpected error: %v", err)
	}
	if tomorrow.ID == again.ID {
		t.Error("a start on another day continued yesterday's session")
	}
}

package store

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5/pgxpool"
	"testing"
	"time"
)

type classFixture struct {
	courseID  int64
	teacherID int64
	teacherUA int64
	sessionID int64
}

func insertCourseRow(t *testing.T, ctx context.Context, pool *pgxpool.Pool, code, title string) int64 {
	t.Helper()
	var id int64
	query := "INSERT INTO courses (code, title) VALUES ($1, $2) RETURNING id"
	if err := pool.QueryRow(ctx, query, code, title).Scan(&id); err != nil {
		t.Fatalf("could not insert the course %q: %v", code, err)
	}
	return id
}

func insertTeacherRow(t *testing.T, ctx context.Context, pool *pgxpool.Pool,
	userID int64, teacherID, name string) int64 {
	t.Helper()
	var id int64
	query := `INSERT INTO teachers (user_id, teacher_id, name, email, phone)
VALUES ($1, $2, $3, $2 || '@sust.edu', '01800000000') RETURNING id`
	if err := pool.QueryRow(ctx, query, userID, teacherID, name).Scan(&id); err != nil {
		t.Fatalf("could not insert the teacher %q: %v", teacherID, err)
	}
	return id
}

func insertStudentRow(t *testing.T, ctx context.Context, pool *pgxpool.Pool,
	userID int64, regNo, batch, section string) int64 {
	t.Helper()
	var id int64
	query := `INSERT INTO students (user_id, full_name, reg_no, batch, section, email, phone, blood_group)
VALUES ($1, 'Student ' || $2, $2, $3, $4, $2 || '@student.sust.edu', '01700000000', 'O+')
RETURNING id`
	if err := pool.QueryRow(ctx, query, userID, regNo, batch, section).Scan(&id); err != nil {
		t.Fatalf("could not insert the student %q: %v", regNo, err)
	}
	return id
}

func insertRoutineRow(t *testing.T, ctx context.Context, pool *pgxpool.Pool,
	courseID, teacherID int64, batch, section, semester string) {
	t.Helper()
	query := `INSERT INTO routines
(batch, section, course_id, teacher_id, day_of_week, start_time, end_time, room, type, semester)
VALUES ($1, $2, $3, $4, 1, TIME '09:00', TIME '09:45', '2104', 'class', $5)`
	if _, err := pool.Exec(ctx, query, batch, section, courseID, teacherID, semester); err != nil {
		t.Fatalf("could not insert the routine of %s: %v", batch, err)
	}
	setRunningSemester(t, ctx, pool, batch, semester)
}

func addDropCourse(t *testing.T, ctx context.Context, pool *pgxpool.Pool, studentID, courseID int64) {
	t.Helper()
	query := "INSERT INTO student_drop_courses (student_id, course_id) VALUES ($1, $2)"
	if _, err := pool.Exec(ctx, query, studentID, courseID); err != nil {
		t.Fatalf("could not add the drop course: %v", err)
	}
}

func newClass(t *testing.T, ctx context.Context, store *Store, pool *pgxpool.Pool,
	batch, section string) classFixture {
	t.Helper()
	courseID := insertCourseRow(t, ctx, pool, "CSE245", "Databases")
	teacherUser := insertUser(t, ctx, pool, RoleTeacher, "T-500", StatusActive, 0)
	teacherID := insertTeacherRow(t, ctx, pool, teacherUser, "T-500", "Main Teacher")
	insertRoutineRow(t, ctx, pool, courseID, teacherID, batch, section, "3-1")
	session, err := store.OpenAttendanceSession(ctx, NewAttendanceSession{
		CourseID:  courseID,
		Batch:     batch,
		Section:   section,
		TeacherID: teacherID,
		EndsAt:    time.Now().Add(10 * time.Minute),
		TokenKey:  "session-key",
	})
	if err != nil {
		t.Fatalf("OpenAttendanceSession() returned an unexpected error: %v", err)
	}
	return classFixture{
		courseID:  courseID,
		teacherID: teacherID,
		teacherUA: teacherUser,
		sessionID: session.ID,
	}
}

func TestBindDeviceKeepsTheFirstPhone(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	userID := insertUser(t, ctx, pool, RoleStudent, "2023331001", StatusActive, 0)
	insertStudentRow(t, ctx, pool, userID, "2023331001", "CSE-23", "A")
	first, fresh, err := store.BindDevice(ctx, userID, "phone-one")
	if err != nil {
		t.Fatalf("BindDevice() returned an unexpected error: %v", err)
	}
	if !fresh || first.DeviceID != "phone-one" {
		t.Fatalf("BindDevice() = %+v, fresh = %t, want a fresh binding to phone-one", first, fresh)
	}
	again, fresh, err := store.BindDevice(ctx, userID, "phone-two")
	if err != nil {
		t.Fatalf("BindDevice() returned an unexpected error on the second phone: %v", err)
	}
	if fresh {
		t.Error("BindDevice() reported a fresh binding for a second phone, want false")
	}
	if again.DeviceID != "phone-one" {
		t.Errorf("the registered phone = %q, want phone-one", again.DeviceID)
	}
}

func TestDeviceChangeRequestRules(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	userID := insertUser(t, ctx, pool, RoleStudent, "2023331001", StatusActive, 0)
	insertStudentRow(t, ctx, pool, userID, "2023331001", "CSE-23", "A")
	otherUser := insertUser(t, ctx, pool, RoleStudent, "2023331002", StatusActive, 0)
	insertStudentRow(t, ctx, pool, otherUser, "2023331002", "CSE-23", "A")
	if _, _, err := store.BindDevice(ctx, userID, "phone-one"); err != nil {
		t.Fatalf("BindDevice() returned an unexpected error: %v", err)
	}
	if _, _, err := store.BindDevice(ctx, otherUser, "phone-taken"); err != nil {
		t.Fatalf("BindDevice() returned an unexpected error: %v", err)
	}
	if _, err := store.CreateDeviceChangeRequest(ctx, userID, "phone-one", "same phone", "3-1"); !errors.Is(err, ErrSameDeviceBound) {
		t.Errorf("a request for the bound phone returned %v, want ErrSameDeviceBound", err)
	}
	if _, err := store.CreateDeviceChangeRequest(ctx, userID, "phone-taken", "another student's phone", "3-1"); !errors.Is(err, ErrDeviceTaken) {
		t.Errorf("a request for a taken phone returned %v, want ErrDeviceTaken", err)
	}
	if _, err := store.CreateDeviceChangeRequest(ctx, userID, "phone-two", "my phone broke", "3-1"); err != nil {
		t.Fatalf("CreateDeviceChangeRequest() returned an unexpected error: %v", err)
	}
	if _, err := store.CreateDeviceChangeRequest(ctx, userID, "phone-three", "again", "3-1"); !errors.Is(err, ErrRequestPending) {
		t.Errorf("a second pending request returned %v, want ErrRequestPending", err)
	}
}

func TestOnlyTheBatchTeachersDecideAPhoneChange(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	class := newClass(t, ctx, store, pool, "CSE-23", "A")
	studentUser := insertUser(t, ctx, pool, RoleStudent, "2023331001", StatusActive, 0)
	insertStudentRow(t, ctx, pool, studentUser, "2023331001", "CSE-23", "A")
	strangerUser := insertUser(t, ctx, pool, RoleTeacher, "T-900", StatusActive, 0)
	strangerID := insertTeacherRow(t, ctx, pool, strangerUser, "T-900", "Other Teacher")
	otherCourse := insertCourseRow(t, ctx, pool, "CSE299", "Another Course")
	insertRoutineRow(t, ctx, pool, otherCourse, strangerID, "CSE-22", "A", "4-1")
	if _, _, err := store.BindDevice(ctx, studentUser, "phone-one"); err != nil {
		t.Fatalf("BindDevice() returned an unexpected error: %v", err)
	}
	request, err := store.CreateDeviceChangeRequest(ctx, studentUser, "phone-two", "broken screen", "3-1")
	if err != nil {
		t.Fatalf("CreateDeviceChangeRequest() returned an unexpected error: %v", err)
	}
	if _, err := store.DecideDeviceChange(ctx, request.ID, strangerUser, true, "", 2); !errors.Is(err, ErrNotTheirStudent) {
		t.Errorf("a teacher of another batch got %v, want ErrNotTheirStudent", err)
	}
	mine, err := store.ListTeacherDeviceRequests(ctx, class.teacherUA)
	if err != nil {
		t.Fatalf("ListTeacherDeviceRequests() returned an unexpected error: %v", err)
	}
	if len(mine) != 1 || mine[0].RegNo != "2023331001" {
		t.Fatalf("the teacher sees %v, want the one request of their student", mine)
	}
	theirs, err := store.ListTeacherDeviceRequests(ctx, strangerUser)
	if err != nil {
		t.Fatalf("ListTeacherDeviceRequests() returned an unexpected error: %v", err)
	}
	if len(theirs) != 0 {
		t.Errorf("a teacher of another batch sees %v, want nothing", theirs)
	}
	decided, err := store.DecideDeviceChange(ctx, request.ID, class.teacherUA, true, "", 2)
	if err != nil {
		t.Fatalf("DecideDeviceChange() returned an unexpected error: %v", err)
	}
	if decided.Status != RequestApproved || decided.DeciderName != "Main Teacher" {
		t.Errorf("the decision = %+v, want an approval by Main Teacher", decided)
	}
	binding, err := store.DeviceBinding(ctx, studentUser)
	if err != nil {
		t.Fatalf("DeviceBinding() returned an unexpected error: %v", err)
	}
	if binding.DeviceID != "phone-two" {
		t.Errorf("the registered phone = %q, want phone-two", binding.DeviceID)
	}
	if _, err := store.DecideDeviceChange(ctx, request.ID, class.teacherUA, true, "", 2); !errors.Is(err, ErrRequestDecided) {
		t.Errorf("a second decision returned %v, want ErrRequestDecided", err)
	}
}

func TestPhoneChangesBeyondTheSemesterLimitNeedANote(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	class := newClass(t, ctx, store, pool, "CSE-23", "A")
	studentUser := insertUser(t, ctx, pool, RoleStudent, "2023331001", StatusActive, 0)
	insertStudentRow(t, ctx, pool, studentUser, "2023331001", "CSE-23", "A")
	if _, _, err := store.BindDevice(ctx, studentUser, "phone-0"); err != nil {
		t.Fatalf("BindDevice() returned an unexpected error: %v", err)
	}
	for attempt := 1; attempt <= 2; attempt++ {
		request, err := store.CreateDeviceChangeRequest(ctx, studentUser,
			"phone-"+string(rune('0'+attempt)), "lost it", "3-1")
		if err != nil {
			t.Fatalf("CreateDeviceChangeRequest() returned an unexpected error: %v", err)
		}
		if _, err := store.DecideDeviceChange(ctx, request.ID, class.teacherUA, true, "", 2); err != nil {
			t.Fatalf("DecideDeviceChange() returned an unexpected error on change %d: %v", attempt, err)
		}
	}
	changes, err := store.ApprovedDeviceChanges(ctx, studentUser, "3-1")
	if err != nil {
		t.Fatalf("ApprovedDeviceChanges() returned an unexpected error: %v", err)
	}
	if changes != 2 {
		t.Fatalf("ApprovedDeviceChanges() = %d, want 2", changes)
	}
	third, err := store.CreateDeviceChangeRequest(ctx, studentUser, "phone-9", "again", "3-1")
	if err != nil {
		t.Fatalf("CreateDeviceChangeRequest() returned an unexpected error: %v", err)
	}
	if _, err := store.DecideDeviceChange(ctx, third.ID, class.teacherUA, true, "", 2); !errors.Is(err, ErrNoteRequired) {
		t.Errorf("a third approval without a note returned %v, want ErrNoteRequired", err)
	}
	withNote, err := store.DecideDeviceChange(ctx, third.ID, class.teacherUA, true,
		"Checked the broken phone myself", 2)
	if err != nil {
		t.Fatalf("DecideDeviceChange() with a note returned an unexpected error: %v", err)
	}
	if withNote.DecisionNote != "Checked the broken phone myself" {
		t.Errorf("the decision note = %q, want the teacher's note", withNote.DecisionNote)
	}
	if _, err := store.CreateDeviceChangeRequest(ctx, studentUser, "phone-8", "next term", "3-2"); err != nil {
		t.Fatalf("a request in the next semester returned an unexpected error: %v", err)
	}
	fresh, err := store.ApprovedDeviceChanges(ctx, studentUser, "3-2")
	if err != nil {
		t.Fatalf("ApprovedDeviceChanges() returned an unexpected error: %v", err)
	}
	if fresh != 0 {
		t.Errorf("the next semester starts at %d approved changes, want 0", fresh)
	}
}

func TestRosterTakesTheSectionAllAndTheDropCourses(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	class := newClass(t, ctx, store, pool, "CSE-23", "ALL")
	sectionA := insertUser(t, ctx, pool, RoleStudent, "2023331001", StatusActive, 0)
	insertStudentRow(t, ctx, pool, sectionA, "2023331001", "CSE-23", "A")
	sectionB := insertUser(t, ctx, pool, RoleStudent, "2023331002", StatusActive, 0)
	insertStudentRow(t, ctx, pool, sectionB, "2023331002", "CSE-23", "B")
	otherBatch := insertUser(t, ctx, pool, RoleStudent, "2022331003", StatusActive, 0)
	otherID := insertStudentRow(t, ctx, pool, otherBatch, "2022331003", "CSE-22", "A")
	addDropCourse(t, ctx, pool, otherID, class.courseID)
	stranger := insertUser(t, ctx, pool, RoleStudent, "2022331004", StatusActive, 0)
	insertStudentRow(t, ctx, pool, stranger, "2022331004", "CSE-22", "B")
	removed := insertUser(t, ctx, pool, RoleStudent, "2023331005", StatusRemoved, 0)
	insertStudentRow(t, ctx, pool, removed, "2023331005", "CSE-23", "A")
	roster, err := store.ListRoster(ctx, class.sessionID)
	if err != nil {
		t.Fatalf("ListRoster() returned an unexpected error: %v", err)
	}
	want := map[string]bool{"2023331001": true, "2023331002": true, "2022331003": true}
	if len(roster) != len(want) {
		t.Fatalf("the roster = %v, want %v", roster, want)
	}
	for _, entry := range roster {
		if !want[entry.RegNo] {
			t.Errorf("the roster carries %q, want only the batch and the drop courses", entry.RegNo)
		}
	}
}

func TestRosterOfOneSectionLeavesTheOtherOut(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	class := newClass(t, ctx, store, pool, "CSE-23", "A")
	sectionA := insertUser(t, ctx, pool, RoleStudent, "2023331001", StatusActive, 0)
	studentA := insertStudentRow(t, ctx, pool, sectionA, "2023331001", "CSE-23", "A")
	sectionB := insertUser(t, ctx, pool, RoleStudent, "2023331002", StatusActive, 0)
	studentB := insertStudentRow(t, ctx, pool, sectionB, "2023331002", "CSE-23", "B")
	roster, err := store.ListRoster(ctx, class.sessionID)
	if err != nil {
		t.Fatalf("ListRoster() returned an unexpected error: %v", err)
	}
	if len(roster) != 1 || roster[0].StudentID != studentA {
		t.Fatalf("the roster = %v, want only section A", roster)
	}
	listed, err := store.OnRoster(ctx, class.sessionID, studentB)
	if err != nil {
		t.Fatalf("OnRoster() returned an unexpected error: %v", err)
	}
	if listed {
		t.Error("OnRoster() accepted a section B student in a section A class")
	}
}

func TestOnePhoneMarksOneStudentPerSession(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	class := newClass(t, ctx, store, pool, "CSE-23", "A")
	firstUser := insertUser(t, ctx, pool, RoleStudent, "2023331001", StatusActive, 0)
	first := insertStudentRow(t, ctx, pool, firstUser, "2023331001", "CSE-23", "A")
	secondUser := insertUser(t, ctx, pool, RoleStudent, "2023331002", StatusActive, 0)
	second := insertStudentRow(t, ctx, pool, secondUser, "2023331002", "CSE-23", "A")
	mark := MarkInput{
		SessionID: class.sessionID,
		StudentID: first,
		Method:    "app",
		DeviceID:  "phone-one",
		Lat:       24.9169,
		Lng:       91.8316,
		AccuracyM: 10,
		DistanceM: 8,
	}
	if err := store.MarkAttendance(ctx, mark); err != nil {
		t.Fatalf("MarkAttendance() returned an unexpected error: %v", err)
	}
	if err := store.MarkAttendance(ctx, mark); !errors.Is(err, ErrAlreadyMarked) {
		t.Errorf("a second mark of the same student returned %v, want ErrAlreadyMarked", err)
	}
	borrowed := mark
	borrowed.StudentID = second
	if err := store.MarkAttendance(ctx, borrowed); !errors.Is(err, ErrDeviceUsed) {
		t.Errorf("a second student on the same phone returned %v, want ErrDeviceUsed", err)
	}
	borrowed.DeviceID = "phone-two"
	if err := store.MarkAttendance(ctx, borrowed); err != nil {
		t.Fatalf("MarkAttendance() on a second phone returned an unexpected error: %v", err)
	}
}

func TestMarkingStopsWhenTheSessionCloses(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	class := newClass(t, ctx, store, pool, "CSE-23", "A")
	presentUser := insertUser(t, ctx, pool, RoleStudent, "2023331001", StatusActive, 0)
	present := insertStudentRow(t, ctx, pool, presentUser, "2023331001", "CSE-23", "A")
	missingUser := insertUser(t, ctx, pool, RoleStudent, "2023331002", StatusActive, 0)
	missing := insertStudentRow(t, ctx, pool, missingUser, "2023331002", "CSE-23", "A")
	err := store.MarkAttendance(ctx, MarkInput{
		SessionID: class.sessionID,
		StudentID: present,
		Method:    "app",
		DeviceID:  "phone-one",
		Lat:       24.9169,
		Lng:       91.8316,
		AccuracyM: 9,
		DistanceM: 4,
	})
	if err != nil {
		t.Fatalf("MarkAttendance() returned an unexpected error: %v", err)
	}
	closed, err := store.CloseAttendanceSession(ctx, class.sessionID, class.teacherID)
	if err != nil {
		t.Fatalf("CloseAttendanceSession() returned an unexpected error: %v", err)
	}
	if closed.Status != "closed" || closed.Present != 1 || closed.Roster != 2 {
		t.Fatalf("the closed session = %+v, want one of two present", closed)
	}
	records, err := store.ListSessionRecords(ctx, class.sessionID)
	if err != nil {
		t.Fatalf("ListSessionRecords() returned an unexpected error: %v", err)
	}
	byStudent := make(map[int64]AttendanceRecord)
	for _, record := range records {
		byStudent[record.StudentID] = record
	}
	if got := byStudent[present]; got.Status != "present" || got.Method != "app" {
		t.Errorf("the marked student = %+v, want present by the app", got)
	}
	if got := byStudent[missing]; got.Status != "absent" || got.Method != "system" {
		t.Errorf("the missing student = %+v, want absent by the system", got)
	}
	err = store.MarkAttendance(ctx, MarkInput{
		SessionID: class.sessionID,
		StudentID: missing,
		Method:    "app",
		DeviceID:  "phone-two",
	})
	if !errors.Is(err, ErrAttendanceClosed) && !errors.Is(err, ErrAlreadyMarked) {
		t.Errorf("a mark after the close returned %v, want the session refused", err)
	}
	if _, err := store.CloseAttendanceSession(ctx, class.sessionID, class.teacherID); !errors.Is(err, ErrAttendanceClosed) {
		t.Errorf("a second close returned %v, want ErrAttendanceClosed", err)
	}
}

func TestManualEditKeepsTheNoteAndTheEditor(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	class := newClass(t, ctx, store, pool, "CSE-23", "A")
	studentUser := insertUser(t, ctx, pool, RoleStudent, "2023331001", StatusActive, 0)
	studentID := insertStudentRow(t, ctx, pool, studentUser, "2023331001", "CSE-23", "A")
	err := store.EditAttendanceRecord(ctx, class.sessionID, studentID, class.teacherUA,
		"present", "Marked by hand, this student uses an iPhone")
	if err != nil {
		t.Fatalf("EditAttendanceRecord() returned an unexpected error: %v", err)
	}
	records, err := store.ListSessionRecords(ctx, class.sessionID)
	if err != nil {
		t.Fatalf("ListSessionRecords() returned an unexpected error: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("the session carries %d rows, want 1", len(records))
	}
	record := records[0]
	if record.Status != "present" || record.Method != "manual" {
		t.Errorf("the record = %+v, want a manual present", record)
	}
	if record.Note != "Marked by hand, this student uses an iPhone" {
		t.Errorf("the note = %q, want the teacher's note", record.Note)
	}
	if record.EditedBy != "Main Teacher" || record.EditedAt.IsZero() {
		t.Errorf("the edit is credited to %q at %v, want Main Teacher with a time", record.EditedBy, record.EditedAt)
	}
	err = store.EditAttendanceRecord(ctx, class.sessionID, studentID, class.teacherUA, "absent", "")
	if err != nil {
		t.Fatalf("EditAttendanceRecord() returned an unexpected error on the second edit: %v", err)
	}
	again, err := store.ListSessionRecords(ctx, class.sessionID)
	if err != nil {
		t.Fatalf("ListSessionRecords() returned an unexpected error: %v", err)
	}
	if again[0].Status != "absent" || again[0].Note != "" {
		t.Errorf("the second edit = %+v, want an absence with an empty note", again[0])
	}
}

func TestStudentAttendanceCountsPresentOverHeld(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	class := newClass(t, ctx, store, pool, "CSE-23", "A")
	studentUser := insertUser(t, ctx, pool, RoleStudent, "2023331001", StatusActive, 0)
	studentID := insertStudentRow(t, ctx, pool, studentUser, "2023331001", "CSE-23", "A")
	statuses := []string{"present", "absent", "present", "absent"}
	for index, status := range statuses {
		sessionID := class.sessionID
		if index > 0 {
			session, err := store.OpenAttendanceSession(ctx, NewAttendanceSession{
				CourseID:  class.courseID,
				Batch:     "CSE-23",
				Section:   "A",
				TeacherID: class.teacherID,
				EndsAt:    time.Now().Add(10 * time.Minute),
				TokenKey:  "later-key",
			})
			if err != nil {
				t.Fatalf("OpenAttendanceSession() returned an unexpected error: %v", err)
			}
			if session.ID == sessionID {
				if _, err := store.CloseAttendanceSession(ctx, sessionID, class.teacherID); err != nil {
					t.Fatalf("CloseAttendanceSession() returned an unexpected error: %v", err)
				}
				session, err = store.OpenAttendanceSession(ctx, NewAttendanceSession{
					CourseID:  class.courseID,
					Batch:     "CSE-23",
					Section:   "A",
					TeacherID: class.teacherID,
					EndsAt:    time.Now().Add(10 * time.Minute),
					TokenKey:  "later-key",
				})
				if err != nil {
					t.Fatalf("OpenAttendanceSession() returned an unexpected error: %v", err)
				}
			}
			sessionID = session.ID
		}
		err := store.EditAttendanceRecord(ctx, sessionID, studentID, class.teacherUA, status, "")
		if err != nil {
			t.Fatalf("EditAttendanceRecord() returned an unexpected error: %v", err)
		}
		if _, err := store.CloseAttendanceSession(ctx, sessionID, class.teacherID); err != nil {
			t.Fatalf("CloseAttendanceSession() returned an unexpected error: %v", err)
		}
	}
	courses, err := store.StudentCourseAttendance(ctx, studentID)
	if err != nil {
		t.Fatalf("StudentCourseAttendance() returned an unexpected error: %v", err)
	}
	if len(courses) != 1 {
		t.Fatalf("the student has %d courses, want 1", len(courses))
	}
	if courses[0].Held != 4 || courses[0].Present != 2 {
		t.Errorf("the course = %+v, want two of four", courses[0])
	}
	rows, err := store.StudentAttendanceSessions(ctx, studentID)
	if err != nil {
		t.Fatalf("StudentAttendanceSessions() returned an unexpected error: %v", err)
	}
	if len(rows) != 4 {
		t.Errorf("the student sees %d classes, want 4", len(rows))
	}
}

func TestLowAttendanceListsOnlyTheStudentsUnderFifty(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	class := newClass(t, ctx, store, pool, "CSE-23", "A")
	lowUser := insertUser(t, ctx, pool, RoleStudent, "2023331001", StatusActive, 0)
	low := insertStudentRow(t, ctx, pool, lowUser, "2023331001", "CSE-23", "A")
	highUser := insertUser(t, ctx, pool, RoleStudent, "2023331002", StatusActive, 0)
	high := insertStudentRow(t, ctx, pool, highUser, "2023331002", "CSE-23", "A")
	sessionID := class.sessionID
	for round := 0; round < 4; round++ {
		if round > 0 {
			session, err := store.OpenAttendanceSession(ctx, NewAttendanceSession{
				CourseID:  class.courseID,
				Batch:     "CSE-23",
				Section:   "A",
				TeacherID: class.teacherID,
				EndsAt:    time.Now().Add(10 * time.Minute),
				TokenKey:  "round-key",
			})
			if err != nil {
				t.Fatalf("OpenAttendanceSession() returned an unexpected error: %v", err)
			}
			sessionID = session.ID
		}
		if err := store.EditAttendanceRecord(ctx, sessionID, high, class.teacherUA, "present", ""); err != nil {
			t.Fatalf("EditAttendanceRecord() returned an unexpected error: %v", err)
		}
		status := "absent"
		if round == 0 {
			status = "present"
		}
		if err := store.EditAttendanceRecord(ctx, sessionID, low, class.teacherUA, status, ""); err != nil {
			t.Fatalf("EditAttendanceRecord() returned an unexpected error: %v", err)
		}
		if _, err := store.CloseAttendanceSession(ctx, sessionID, class.teacherID); err != nil {
			t.Fatalf("CloseAttendanceSession() returned an unexpected error: %v", err)
		}
	}
	alerts, err := store.ListLowAttendance(ctx, 3, 50)
	if err != nil {
		t.Fatalf("ListLowAttendance() returned an unexpected error: %v", err)
	}
	if len(alerts) != 1 {
		t.Fatalf("ListLowAttendance() = %v, want only the student under 50 %%", alerts)
	}
	if alerts[0].StudentID != low || alerts[0].Held != 4 || alerts[0].Present != 1 {
		t.Errorf("the alert = %+v, want one of four for the low student", alerts[0])
	}
	before := time.Now().Add(-7 * 24 * time.Hour)
	first, err := store.ClaimAttendanceAlert(ctx, low, class.courseID, before)
	if err != nil {
		t.Fatalf("ClaimAttendanceAlert() returned an unexpected error: %v", err)
	}
	if !first {
		t.Error("the first claim was refused, want it taken")
	}
	second, err := store.ClaimAttendanceAlert(ctx, low, class.courseID, before)
	if err != nil {
		t.Fatalf("ClaimAttendanceAlert() returned an unexpected error: %v", err)
	}
	if second {
		t.Error("the same week was claimed twice, want the second refused")
	}
	later, err := store.ClaimAttendanceAlert(ctx, low, class.courseID, time.Now().Add(time.Minute))
	if err != nil {
		t.Fatalf("ClaimAttendanceAlert() returned an unexpected error: %v", err)
	}
	if !later {
		t.Error("a claim a week later was refused, want it taken")
	}
}

func TestOpenSessionForStudentFindsTheirClassOnly(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	class := newClass(t, ctx, store, pool, "CSE-23", "A")
	mineUser := insertUser(t, ctx, pool, RoleStudent, "2023331001", StatusActive, 0)
	mine := insertStudentRow(t, ctx, pool, mineUser, "2023331001", "CSE-23", "A")
	otherUser := insertUser(t, ctx, pool, RoleStudent, "2023331002", StatusActive, 0)
	other := insertStudentRow(t, ctx, pool, otherUser, "2023331002", "CSE-23", "B")
	found, err := store.OpenSessionForStudent(ctx, mine)
	if err != nil {
		t.Fatalf("OpenSessionForStudent() returned an unexpected error: %v", err)
	}
	if found.ID != class.sessionID {
		t.Errorf("the open session = %d, want %d", found.ID, class.sessionID)
	}
	if _, err := store.OpenSessionForStudent(ctx, other); !errors.Is(err, ErrNotFound) {
		t.Errorf("a section B student got %v, want ErrNotFound", err)
	}
	dropUser := insertUser(t, ctx, pool, RoleStudent, "2022331003", StatusActive, 0)
	drop := insertStudentRow(t, ctx, pool, dropUser, "2022331003", "CSE-22", "A")
	addDropCourse(t, ctx, pool, drop, class.courseID)
	dropped, err := store.OpenSessionForStudent(ctx, drop)
	if err != nil {
		t.Fatalf("OpenSessionForStudent() returned an unexpected error for a drop course: %v", err)
	}
	if dropped.ID != class.sessionID {
		t.Errorf("the drop course student sees session %d, want %d", dropped.ID, class.sessionID)
	}
}

func TestSessionLocationAndWindow(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	class := newClass(t, ctx, store, pool, "CSE-23", "A")
	if err := store.SetAttendanceLocation(ctx, class.sessionID, class.teacherID, 24.9169, 91.8316, 12); err != nil {
		t.Fatalf("SetAttendanceLocation() returned an unexpected error: %v", err)
	}
	session, err := store.AttendanceSession(ctx, class.sessionID)
	if err != nil {
		t.Fatalf("AttendanceSession() returned an unexpected error: %v", err)
	}
	if !session.HasLocation || session.TeacherLat != 24.9169 || session.LocationAt.IsZero() {
		t.Fatalf("the session = %+v, want the teacher position kept", session)
	}
	longer, err := store.ExtendAttendanceSession(ctx, class.sessionID, class.teacherID, 10)
	if err != nil {
		t.Fatalf("ExtendAttendanceSession() returned an unexpected error: %v", err)
	}
	if !longer.EndsAt.After(session.EndsAt) {
		t.Errorf("the session now ends at %v, want later than %v", longer.EndsAt, session.EndsAt)
	}
	stranger := insertUser(t, ctx, pool, RoleTeacher, "T-901", StatusActive, 0)
	strangerID := insertTeacherRow(t, ctx, pool, stranger, "T-901", "Other Teacher")
	err = store.SetAttendanceLocation(ctx, class.sessionID, strangerID, 24.9, 91.8, 10)
	if !errors.Is(err, ErrAttendanceClosed) {
		t.Errorf("another teacher setting the position got %v, want it refused", err)
	}
	if _, err := store.CloseAttendanceSession(ctx, class.sessionID, strangerID); !errors.Is(err, ErrAttendanceClosed) {
		t.Errorf("another teacher closing the session got %v, want it refused", err)
	}
}

func TestOpenAttendanceSessionReusesTheRunningOne(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	class := newClass(t, ctx, store, pool, "CSE-23", "A")
	again, err := store.OpenAttendanceSession(ctx, NewAttendanceSession{
		CourseID:  class.courseID,
		Batch:     "CSE-23",
		Section:   "A",
		TeacherID: class.teacherID,
		EndsAt:    time.Now().Add(20 * time.Minute),
		TokenKey:  "second-key",
	})
	if err != nil {
		t.Fatalf("OpenAttendanceSession() returned an unexpected error: %v", err)
	}
	if again.ID != class.sessionID {
		t.Errorf("a second start opened session %d, want the running %d", again.ID, class.sessionID)
	}
	if again.TokenKey != "session-key" {
		t.Errorf("the session key changed to %q, want the first key kept", again.TokenKey)
	}
}

func TestTeacherClassesAndExportRows(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	class := newClass(t, ctx, store, pool, "CSE-23", "A")
	studentUser := insertUser(t, ctx, pool, RoleStudent, "2023331001", StatusActive, 0)
	studentID := insertStudentRow(t, ctx, pool, studentUser, "2023331001", "CSE-23", "A")
	if err := store.EditAttendanceRecord(ctx, class.sessionID, studentID, class.teacherUA, "present", "On time"); err != nil {
		t.Fatalf("EditAttendanceRecord() returned an unexpected error: %v", err)
	}
	if _, err := store.CloseAttendanceSession(ctx, class.sessionID, class.teacherID); err != nil {
		t.Fatalf("CloseAttendanceSession() returned an unexpected error: %v", err)
	}
	from := time.Now().Add(-24 * time.Hour)
	to := time.Now().Add(24 * time.Hour)
	classes, err := store.ListTeacherClasses(ctx, class.teacherID, from, to)
	if err != nil {
		t.Fatalf("ListTeacherClasses() returned an unexpected error: %v", err)
	}
	if len(classes) != 1 || classes[0].Sessions != 1 || classes[0].CourseCode != "CSE245" {
		t.Fatalf("the classes = %v, want one CSE245 session", classes)
	}
	rates, err := store.ClassRates(ctx, class.teacherID, class.courseID, "CSE-23", "A", from, to)
	if err != nil {
		t.Fatalf("ClassRates() returned an unexpected error: %v", err)
	}
	if len(rates) != 1 || rates[0].Present != 1 || rates[0].Held != 1 {
		t.Fatalf("the rates = %v, want one of one", rates)
	}
	rows, err := store.TeacherExportRows(ctx, class.teacherID, class.courseID, "CSE-23", "A", from, to)
	if err != nil {
		t.Fatalf("TeacherExportRows() returned an unexpected error: %v", err)
	}
	if len(rows) != 1 || rows[0].Status != "present" || rows[0].Note != "On time" {
		t.Fatalf("the export rows = %v, want the present row with its note", rows)
	}
	batchRows, err := store.BatchExportRows(ctx, class.courseID, "CSE-23", "A", from, to)
	if err != nil {
		t.Fatalf("BatchExportRows() returned an unexpected error: %v", err)
	}
	if len(batchRows) != len(rows) {
		t.Errorf("the batch export has %d rows, want %d", len(batchRows), len(rows))
	}
	batchClasses, err := store.ListBatchClasses(ctx, "CSE-23", from, to)
	if err != nil {
		t.Fatalf("ListBatchClasses() returned an unexpected error: %v", err)
	}
	if len(batchClasses) != 1 {
		t.Errorf("the batch classes = %v, want one", batchClasses)
	}
	batchRates, err := store.BatchClassRates(ctx, class.courseID, "CSE-23", "A", from, to)
	if err != nil {
		t.Fatalf("BatchClassRates() returned an unexpected error: %v", err)
	}
	if len(batchRates) != 1 {
		t.Errorf("the batch rates = %v, want one", batchRates)
	}
}

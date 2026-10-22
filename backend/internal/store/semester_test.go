package store

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5/pgxpool"
	"testing"
	"time"
)

func seedSemesterRoutines(t *testing.T, ctx context.Context, store *Store,
	pool *pgxpool.Pool) (engineFixture, int64) {
	t.Helper()
	fixture := seedEngine(t, ctx, store, pool, 1)
	older, err := store.CreateCourse(ctx, "CSE 1101 1101A", "Structured Programming", 3.0)
	if err != nil {
		t.Fatalf("CreateCourse() returned an unexpected error: %v", err)
	}
	query := `INSERT INTO routines (batch, semester, section, course_id, teacher_id, day_of_week,
start_time, end_time, room, type)
VALUES ('CSE-23', '1-2', 'A', $1, $2, 2, TIME '10:30', TIME '11:15', '2104', 'class') RETURNING id`
	var olderRoutine int64
	if err := pool.QueryRow(ctx, query, older.ID, fixture.teacherID).Scan(&olderRoutine); err != nil {
		t.Fatalf("could not insert the older routine entry: %v", err)
	}
	return fixture, older.ID
}

func TestRunningSemesterDefaultsAndChanges(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	semester, err := store.RunningSemester(ctx, "CSE-23")
	if err != nil {
		t.Fatalf("RunningSemester() returned an unexpected error: %v", err)
	}
	if semester != "1-1" {
		t.Errorf("semester = %q, want the default 1-1", semester)
	}
	seedSemesterRoutines(t, ctx, store, pool)
	running, err := store.RunningSemester(ctx, "CSE-23")
	if err != nil {
		t.Fatalf("RunningSemester() returned an unexpected error: %v", err)
	}
	if running != "1-1" {
		t.Errorf("semester = %q, want the one the first routine entry used", running)
	}
	if err := store.SetRunningSemester(ctx, "CSE-23", "1-2"); err != nil {
		t.Fatalf("SetRunningSemester() returned an unexpected error: %v", err)
	}
	moved, err := store.RunningSemester(ctx, "CSE-23")
	if err != nil {
		t.Fatalf("RunningSemester() returned an unexpected error: %v", err)
	}
	if moved != "1-2" {
		t.Errorf("semester = %q, want 1-2", moved)
	}
	all, err := store.ListRunningSemesters(ctx)
	if err != nil {
		t.Fatalf("ListRunningSemesters() returned an unexpected error: %v", err)
	}
	if all["CSE-23"] != "1-2" {
		t.Errorf("running semesters = %v, want CSE-23 on 1-2", all)
	}
}

func TestOnlyTheRunningSemesterMakesDatedItems(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	now := time.Now()
	fixture := seedEngine(t, ctx, store, pool, int(now.Weekday()))
	olderCourse, err := store.CreateCourse(ctx, "CSE 1101 1101A", "Structured Programming", 3.0)
	if err != nil {
		t.Fatalf("CreateCourse() returned an unexpected error: %v", err)
	}
	query := `INSERT INTO routines (batch, semester, section, course_id, teacher_id, day_of_week,
start_time, end_time, room, type)
VALUES ('CSE-23', '1-2', 'A', $1, $2, $3, TIME '10:30', TIME '11:15', '2104', 'class')`
	if _, err := pool.Exec(ctx, query, olderCourse.ID, fixture.teacherID, int(now.Weekday())); err != nil {
		t.Fatalf("could not insert the older routine entry: %v", err)
	}
	today := now.Format(dateLayout)
	week := now.AddDate(0, 0, 6).Format(dateLayout)
	if _, err := store.GenerateFixedItems(ctx, today, week); err != nil {
		t.Fatalf("GenerateFixedItems() returned an unexpected error: %v", err)
	}
	items, err := store.ListBatchItems(ctx, "CSE-23", today)
	if err != nil {
		t.Fatalf("ListBatchItems() returned an unexpected error: %v", err)
	}
	if len(items) == 0 {
		t.Fatal("no dated items were generated at all")
	}
	for _, item := range items {
		if item.CourseID == olderCourse.ID {
			t.Errorf("an item was generated from the older semester: %+v", item)
		}
	}
	if err := store.SetRunningSemester(ctx, "CSE-23", "1-2"); err != nil {
		t.Fatalf("SetRunningSemester() returned an unexpected error: %v", err)
	}
	if _, err := store.GenerateFixedItems(ctx, today, week); err != nil {
		t.Fatalf("GenerateFixedItems() returned an unexpected error: %v", err)
	}
	after, err := store.ListBatchItems(ctx, "CSE-23", today)
	if err != nil {
		t.Fatalf("ListBatchItems() returned an unexpected error: %v", err)
	}
	found := false
	for _, item := range after {
		if item.CourseID == olderCourse.ID {
			found = true
		}
	}
	if !found {
		t.Error("the new running semester produced no dated items")
	}
}

func TestDropCourseOptionsFollowTheRunningSemester(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	_, olderCourseID := seedSemesterRoutines(t, ctx, store, pool)
	options, err := store.SearchDropCourseOptions(ctx, "")
	if err != nil {
		t.Fatalf("SearchDropCourseOptions() returned an unexpected error: %v", err)
	}
	for _, option := range options {
		if option.ID == olderCourseID {
			t.Error("a course of a semester that is not running is offered as a drop course")
		}
	}
	if len(options) == 0 {
		t.Fatal("no drop course option at all, want the running semester course")
	}
	if err := store.SetRunningSemester(ctx, "CSE-23", "1-2"); err != nil {
		t.Fatalf("SetRunningSemester() returned an unexpected error: %v", err)
	}
	moved, err := store.SearchDropCourseOptions(ctx, "")
	if err != nil {
		t.Fatalf("SearchDropCourseOptions() returned an unexpected error: %v", err)
	}
	found := false
	for _, option := range moved {
		if option.ID == olderCourseID {
			found = true
		}
	}
	if !found {
		t.Error("the course of the new running semester is not offered as a drop course")
	}
}

func TestRoomAvailabilityUsesTheRunningSemesterRooms(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	seedSemesterRoutines(t, ctx, store, pool)
	from := "2026-11-02"
	to := "2026-11-08"
	slots, err := store.ListRoomAvailability(ctx, from, to)
	if err != nil {
		t.Fatalf("ListRoomAvailability() returned an unexpected error: %v", err)
	}
	rooms := make(map[string]bool)
	for _, slot := range slots {
		rooms[slot.Room] = true
	}
	if !rooms["1003"] {
		t.Error("the running semester room is missing from the availability grid")
	}
	if rooms["2104"] {
		t.Error("a room only an older semester uses shows up in the availability grid")
	}
}

func TestCourseHelperAndSectionRoutinesFollowTheSemester(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	_, olderCourseID := seedSemesterRoutines(t, ctx, store, pool)
	slots, err := store.ListBatchCourseSlots(ctx, "CSE-23", "A")
	if err != nil {
		t.Fatalf("ListBatchCourseSlots() returned an unexpected error: %v", err)
	}
	for _, slot := range slots {
		if slot.CourseID == olderCourseID {
			t.Error("the course helper offers a course that is not running")
		}
	}
	running, err := store.ListSectionRoutines(ctx, "CSE-23", "A", "")
	if err != nil {
		t.Fatalf("ListSectionRoutines() returned an unexpected error: %v", err)
	}
	for _, entry := range running {
		if entry.Semester != "1-1" {
			t.Errorf("entry = %+v, want only the running semester", entry)
		}
	}
	older, err := store.ListSectionRoutines(ctx, "CSE-23", "A", "1-2")
	if err != nil {
		t.Fatalf("ListSectionRoutines() returned an unexpected error: %v", err)
	}
	if len(older) != 1 || older[0].CourseID != olderCourseID {
		t.Errorf("older entries = %+v, want the single 1-2 entry", older)
	}
}

func TestListRoutinesAndMoveBySemester(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	_, olderCourseID := seedSemesterRoutines(t, ctx, store, pool)
	everything, err := store.ListRoutines(ctx, "CSE-23", "", "")
	if err != nil {
		t.Fatalf("ListRoutines() returned an unexpected error: %v", err)
	}
	if len(everything) != 2 {
		t.Fatalf("got %d entries, want both semesters", len(everything))
	}
	filtered, err := store.ListRoutines(ctx, "CSE-23", "", "1-2")
	if err != nil {
		t.Fatalf("ListRoutines() returned an unexpected error: %v", err)
	}
	if len(filtered) != 1 || filtered[0].CourseID != olderCourseID {
		t.Fatalf("filtered = %+v, want only the 1-2 entry", filtered)
	}
	if err := store.UpdateRoutineSemester(ctx, filtered[0].ID, "2-1"); err != nil {
		t.Fatalf("UpdateRoutineSemester() returned an unexpected error: %v", err)
	}
	moved, err := store.ListRoutines(ctx, "CSE-23", "", "2-1")
	if err != nil {
		t.Fatalf("ListRoutines() returned an unexpected error: %v", err)
	}
	if len(moved) != 1 {
		t.Errorf("got %d entries in 2-1, want the moved one", len(moved))
	}
	if err := store.UpdateRoutineSemester(ctx, 999999, "2-1"); !errors.Is(err, ErrNotFound) {
		t.Errorf("moving an unknown entry: error = %v, want ErrNotFound", err)
	}
}

func TestRoomClashOnlyInsideTheSameSemester(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	fixture := seedEngine(t, ctx, store, pool, 1)
	base := NewRoutine{
		Batch:     "CSE-23",
		Semester:  "1-2",
		Section:   "B",
		CourseID:  fixture.courseID,
		TeacherID: fixture.teacherID,
		DayOfWeek: 1,
		StartTime: "08:00",
		EndTime:   "09:30",
		Room:      "1003",
		Type:      "class",
	}
	if _, err := store.CreateRoutine(ctx, base); err != nil {
		t.Fatalf("the same room in another semester was refused: %v", err)
	}
	same := base
	same.Semester = "1-1"
	var clash *RoomBookedError
	if _, err := store.CreateRoutine(ctx, same); !errors.As(err, &clash) {
		t.Fatalf("the same room in the same semester: error = %v, want a room clash", err)
	}
}

func TestRunningCourseSlotsCarryCreditsAndTeachers(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	seedSemesterRoutines(t, ctx, store, pool)
	slots, err := store.ListRunningCourseSlots(ctx, "CSE-23")
	if err != nil {
		t.Fatalf("ListRunningCourseSlots() returned an unexpected error: %v", err)
	}
	if len(slots) != 1 {
		t.Fatalf("got %d slots, want the single running entry", len(slots))
	}
	slot := slots[0]
	if slot.CourseCode != "CSE 2243 7637D" || slot.Credit != 3.0 {
		t.Errorf("slot = %+v, want the running course with its credit", slot)
	}
	if slot.TeacherName != "Engine Teacher" || slot.Section != "A" {
		t.Errorf("slot = %+v, want the teacher and section of the entry", slot)
	}
}

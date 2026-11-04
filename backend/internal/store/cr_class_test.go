package store

import (
	"context"
	"github.com/jackc/pgx/v5/pgxpool"
	"testing"
	"time"
)

type routineFixture struct {
	courseID  int64
	teacherID int64
	teacherUA int64
}

func newRoutineClass(t *testing.T, ctx context.Context, pool *pgxpool.Pool,
	batch, section string) routineFixture {
	t.Helper()
	courseID := insertCourseRow(t, ctx, pool, "CSE247", "Operating Systems")
	userID := insertUser(t, ctx, pool, RoleTeacher, "T-700", StatusActive, 0)
	teacherID := insertTeacherRow(t, ctx, pool, userID, "T-700", "Routine Teacher")
	insertRoutineRow(t, ctx, pool, courseID, teacherID, batch, section, "3-1")
	return routineFixture{courseID: courseID, teacherID: teacherID, teacherUA: userID}
}

func TestACRClassWaitsForTheCourseTeacher(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	class := newRoutineClass(t, ctx, pool, "CSE-23", "A")
	tomorrow := time.Now().AddDate(0, 0, 1).Format(dateLayout)
	created, err := store.CreateCRItem(ctx, NewScheduleItem{
		Batch:     "CSE-23",
		Section:   "A",
		CourseID:  class.courseID,
		TeacherID: class.teacherID,
		Kind:      "class",
		ItemDate:  tomorrow,
		StartTime: "09:00",
		EndTime:   "09:45",
		Mode:      ModeRoom,
		Room:      "2104",
	})
	if err != nil {
		t.Fatalf("CreateCRItem() returned an unexpected error: %v", err)
	}
	if created.Status != ItemAwaiting {
		t.Errorf("status = %q, want %q so the teacher decides it", created.Status, ItemAwaiting)
	}
	if created.Origin != OriginCR {
		t.Errorf("origin = %q, want %q", created.Origin, OriginCR)
	}
	teachers, err := store.ListUndecidedTeachers(ctx, tomorrow)
	if err != nil {
		t.Fatalf("ListUndecidedTeachers() returned an unexpected error: %v", err)
	}
	found := false
	for _, teacher := range teachers {
		if teacher.UserID == class.teacherUA && teacher.Count == 1 {
			found = true
		}
	}
	if !found {
		t.Errorf("the undecided teachers = %+v, want the teacher of the created class", teachers)
	}
	day, err := store.ListTeacherDayItems(ctx, class.teacherID, tomorrow)
	if err != nil {
		t.Fatalf("ListTeacherDayItems() returned an unexpected error: %v", err)
	}
	if len(day) != 1 || day[0].ID != created.ID {
		t.Fatalf("the teacher's day = %+v, want the class the representative created", day)
	}
	confirmed, err := store.ConfirmItem(ctx, created.ID, class.teacherID, ModeRoom, "2104", "", time.Now())
	if err != nil {
		t.Fatalf("ConfirmItem() returned an unexpected error: %v", err)
	}
	if confirmed.Status != ItemConfirmed {
		t.Errorf("status = %q, want %q after the teacher confirms", confirmed.Status, ItemConfirmed)
	}
	final, err := store.ListTeacherFinalItems(ctx, class.teacherID, tomorrow)
	if err != nil {
		t.Fatalf("ListTeacherFinalItems() returned an unexpected error: %v", err)
	}
	if len(final) != 1 || final[0].ID != created.ID {
		t.Errorf("the final routine = %+v, want the confirmed class", final)
	}
}

func TestACRExamStaysScheduled(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	class := newRoutineClass(t, ctx, pool, "CSE-23", "A")
	created, err := store.CreateCRItem(ctx, NewScheduleItem{
		Batch:     "CSE-23",
		Section:   "A",
		CourseID:  class.courseID,
		TeacherID: class.teacherID,
		Kind:      KindExam,
		ItemDate:  time.Now().AddDate(0, 0, 2).Format(dateLayout),
		StartTime: "09:00",
		EndTime:   "09:45",
		Mode:      ModeRoom,
		Room:      "2105",
	})
	if err != nil {
		t.Fatalf("CreateCRItem() returned an unexpected error: %v", err)
	}
	if created.Status != ItemScheduled {
		t.Errorf("status = %q, want %q for an exam", created.Status, ItemScheduled)
	}
}

func TestListCourseTeachersNamesEveryTeacherOfTheCourse(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	class := newRoutineClass(t, ctx, pool, "CSE-23", "A")
	secondUser := insertUser(t, ctx, pool, RoleTeacher, "T-701", StatusActive, 0)
	secondID := insertTeacherRow(t, ctx, pool, secondUser, "T-701", "Another Teacher")
	insertRoutineRow(t, ctx, pool, class.courseID, secondID, "CSE-23", "A", "3-1")
	teachers, err := store.ListCourseTeachers(ctx, "CSE-23", "A", class.courseID)
	if err != nil {
		t.Fatalf("ListCourseTeachers() returned an unexpected error: %v", err)
	}
	if len(teachers) != 2 {
		t.Fatalf("teachers = %+v, want both teachers of the course", teachers)
	}
	single, err := store.ListCourseTeachers(ctx, "CSE-23", "A", class.courseID+1000)
	if err != nil {
		t.Fatalf("ListCourseTeachers() returned an unexpected error for an unknown course: %v", err)
	}
	if len(single) != 0 {
		t.Errorf("teachers of an unknown course = %+v, want none", single)
	}
}

func TestTheDateViewKeepsPastDaysOutOfTheFinalFlag(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	class := newRoutineClass(t, ctx, pool, "CSE-23", "A")
	today := time.Now().Format(dateLayout)
	past := time.Now().AddDate(0, 0, -3).Format(dateLayout)
	future := time.Now().AddDate(0, 0, 3).Format(dateLayout)
	older, err := store.CreateCRItem(ctx, NewScheduleItem{
		Batch: "CSE-23", Section: "A", CourseID: class.courseID, TeacherID: class.teacherID,
		Kind: "class", ItemDate: past, StartTime: "09:00", EndTime: "09:45",
		Mode: ModeRoom, Room: "2106",
	})
	if err != nil {
		t.Fatalf("CreateCRItem() returned an unexpected error: %v", err)
	}
	later, err := store.CreateCRItem(ctx, NewScheduleItem{
		Batch: "CSE-23", Section: "A", CourseID: class.courseID, TeacherID: class.teacherID,
		Kind: "class", ItemDate: future, StartTime: "09:00", EndTime: "09:45",
		Mode: ModeRoom, Room: "2107",
	})
	if err != nil {
		t.Fatalf("CreateCRItem() returned an unexpected error: %v", err)
	}
	if _, err := store.SetItemFinal(ctx, older.ID, "CSE-23", true); err != nil {
		t.Fatalf("SetItemFinal() returned an unexpected error: %v", err)
	}
	if _, err := store.SetItemFinal(ctx, later.ID, "CSE-23", true); err != nil {
		t.Fatalf("SetItemFinal() returned an unexpected error: %v", err)
	}
	if _, err := store.ClearPastFinal(ctx, today); err != nil {
		t.Fatalf("ClearPastFinal() returned an unexpected error: %v", err)
	}
	kept, err := store.ListFinalItemsOnDate(ctx, "CSE-23", past, today)
	if err != nil {
		t.Fatalf("ListFinalItemsOnDate() returned an unexpected error: %v", err)
	}
	if len(kept) != 1 || kept[0].ID != older.ID {
		t.Errorf("the past day = %+v, want the kept row even after the cleanup", kept)
	}
	if kept[0].InFinal {
		t.Error("the cleanup left the past row in the final routine")
	}
	ahead, err := store.ListFinalItemsOnDate(ctx, "CSE-23", future, today)
	if err != nil {
		t.Fatalf("ListFinalItemsOnDate() returned an unexpected error: %v", err)
	}
	if len(ahead) != 1 || !ahead[0].InFinal {
		t.Errorf("the future day = %+v, want the item still in the final routine", ahead)
	}
	day, err := store.ListBatchItemsOnDate(ctx, "CSE-23", past)
	if err != nil {
		t.Fatalf("ListBatchItemsOnDate() returned an unexpected error: %v", err)
	}
	if len(day) != 1 || day[0].ID != older.ID {
		t.Errorf("the batch day = %+v, want the one class of that date", day)
	}
}

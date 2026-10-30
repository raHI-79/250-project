package store

import (
	"errors"
	"testing"
	"time"
)

func TestRemoveFromFinalWithdrawsThePublishedCopy(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	fixture := seedEngine(t, ctx, store, pool, 1)
	date := time.Now().AddDate(0, 0, 2).Format(dateLayout)
	item, err := store.CreateCRItem(ctx, newItem(fixture, date, "10:00", "11:30", "A"))
	if err != nil {
		t.Fatalf("CreateCRItem() returned an unexpected error: %v", err)
	}
	if _, err := store.SetItemFinal(ctx, item.ID, "CSE-23", true); err != nil {
		t.Fatalf("SetItemFinal() returned an unexpected error: %v", err)
	}
	if _, _, err := store.PublishFinal(ctx, "CSE-23", fixture.crUserID, []int64{item.ID}); err != nil {
		t.Fatalf("PublishFinal() returned an unexpected error: %v", err)
	}
	removal, err := store.RemoveFromFinal(ctx, item.ID, "CSE-23")
	if err != nil {
		t.Fatalf("RemoveFromFinal() returned an unexpected error: %v", err)
	}
	if !removal.Withdrawn {
		t.Error("the published copy was kept")
	}
	stored, err := store.GetScheduleItem(ctx, item.ID)
	if err != nil {
		t.Fatalf("GetScheduleItem() returned an unexpected error: %v", err)
	}
	if stored.InFinal || stored.Published() {
		t.Errorf("item = %+v, want it out of the final routine and unpublished", stored)
	}
	final, err := store.ListFinalItems(ctx, "CSE-23", time.Now().Format(dateLayout))
	if err != nil {
		t.Fatalf("ListFinalItems() returned an unexpected error: %v", err)
	}
	if len(final) != 0 {
		t.Errorf("final = %d items, want none after the removal", len(final))
	}
	gone, err := store.RemoveFromFinal(ctx, item.ID, "CSE-23")
	if err != nil {
		t.Fatalf("the second RemoveFromFinal() returned an unexpected error: %v", err)
	}
	if gone.Withdrawn {
		t.Error("the second removal withdrew a published copy again")
	}
}

func TestTeacherCancelUpdatesThePublishedCopy(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	fixture := seedEngine(t, ctx, store, pool, 1)
	date := time.Now().AddDate(0, 0, 3).Format(dateLayout)
	item, err := store.CreateCRItem(ctx, newItem(fixture, date, "10:00", "11:30", "A"))
	if err != nil {
		t.Fatalf("CreateCRItem() returned an unexpected error: %v", err)
	}
	if _, err := store.SetItemFinal(ctx, item.ID, "CSE-23", true); err != nil {
		t.Fatalf("SetItemFinal() returned an unexpected error: %v", err)
	}
	if _, _, err := store.PublishFinal(ctx, "CSE-23", fixture.crUserID, []int64{item.ID}); err != nil {
		t.Fatalf("PublishFinal() returned an unexpected error: %v", err)
	}
	cancelled, err := store.CancelItemAsTeacher(ctx, item.ID, fixture.teacherID, "A conference.",
		time.Now())
	if err != nil {
		t.Fatalf("CancelItemAsTeacher() returned an unexpected error: %v", err)
	}
	if cancelled.Status != ItemCancelled || cancelled.CancelReason != "A conference." {
		t.Errorf("item = %+v, want a cancelled item with its reason", cancelled)
	}
	if cancelled.Snapshot.Status != ItemCancelled {
		t.Errorf("the published copy = %q, want %q", cancelled.Snapshot.Status, ItemCancelled)
	}
	free, err := store.CreateCRItem(ctx, newItem(fixture, date, "10:00", "11:30", "B"))
	if err != nil {
		t.Fatalf("the room of the cancelled class stayed booked: %v", err)
	}
	if free.ID == 0 {
		t.Error("the replacement class was not created")
	}
}

func TestClearPastFinalKeepsTheRows(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	fixture := seedEngine(t, ctx, store, pool, 1)
	yesterday := time.Now().AddDate(0, 0, -1).Format(dateLayout)
	today := time.Now().Format(dateLayout)
	past, err := store.CreateCRItem(ctx, newItem(fixture, yesterday, "10:00", "11:30", "A"))
	if err != nil {
		t.Fatalf("CreateCRItem() returned an unexpected error: %v", err)
	}
	ahead, err := store.CreateCRItem(ctx, newItem(fixture, today, "12:00", "13:30", "A"))
	if err != nil {
		t.Fatalf("CreateCRItem() returned an unexpected error: %v", err)
	}
	for _, id := range []int64{past.ID, ahead.ID} {
		if _, err := store.SetItemFinal(ctx, id, "CSE-23", true); err != nil {
			t.Fatalf("SetItemFinal() returned an unexpected error: %v", err)
		}
	}
	cleared, err := store.ClearPastFinal(ctx, today)
	if err != nil {
		t.Fatalf("ClearPastFinal() returned an unexpected error: %v", err)
	}
	if cleared != 1 {
		t.Errorf("cleared = %d, want the item of yesterday only", cleared)
	}
	stored, err := store.GetScheduleItem(ctx, past.ID)
	if err != nil {
		t.Fatalf("the past item is gone: %v", err)
	}
	if stored.InFinal {
		t.Error("the past item stayed in the final routine")
	}
	final, err := store.ListFinalItems(ctx, "CSE-23", today)
	if err != nil {
		t.Fatalf("ListFinalItems() returned an unexpected error: %v", err)
	}
	if len(final) != 1 || final[0].ID != ahead.ID {
		t.Errorf("final = %+v, want today only", final)
	}
}

func TestOnlyTheTeacherDeletesAnExam(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	fixture := seedEngine(t, ctx, store, pool, 1)
	date := time.Now().AddDate(0, 0, 4).Format(dateLayout)
	exam, err := store.CreateTeacherExam(ctx, NewScheduleItem{
		Batch:     "CSE-23",
		Section:   "ALL",
		CourseID:  fixture.courseID,
		TeacherID: fixture.teacherID,
		ItemDate:  date,
		Syllabus:  "Everything",
		CreatedBy: fixture.teacherDB,
	})
	if err != nil {
		t.Fatalf("CreateTeacherExam() returned an unexpected error: %v", err)
	}
	if _, err := store.SetItemFinal(ctx, exam.ID, "CSE-23", true); err != nil {
		t.Fatalf("SetItemFinal() returned an unexpected error: %v", err)
	}
	if _, err := store.DeleteCRItem(ctx, exam.ID, "CSE-23"); !errors.Is(err, ErrItemState) {
		t.Errorf("DeleteCRItem() on an exam of the teacher error = %v, want ErrItemState", err)
	}
	removal, err := store.RemoveFromFinal(ctx, exam.ID, "CSE-23")
	if err != nil {
		t.Fatalf("RemoveFromFinal() on an exam of the teacher returned an unexpected error: %v", err)
	}
	if removal.Item.InFinal {
		t.Error("the exam stayed in the final routine")
	}
	if _, err := store.DeleteTeacherExam(ctx, exam.ID, fixture.teacherID); err != nil {
		t.Fatalf("DeleteTeacherExam() returned an unexpected error: %v", err)
	}
	if _, err := store.GetScheduleItem(ctx, exam.ID); !errors.Is(err, ErrItemNotFound) {
		t.Errorf("the exam survived the deletion: %v", err)
	}
}

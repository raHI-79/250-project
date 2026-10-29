package store

import (
	"errors"
	"testing"
	"time"
)

func clockAt(t *testing.T, hour, minute int) time.Time {
	t.Helper()
	loc, err := time.LoadLocation("Asia/Dhaka")
	if err != nil {
		t.Fatalf("could not load the timezone: %v", err)
	}
	return time.Date(2026, 11, 2, hour, minute, 0, 0, loc)
}

func TestTeacherDecidesAtAnyHourBeforeTheClass(t *testing.T) {
	clocks := []struct {
		name   string
		hour   int
		minute int
	}{
		{name: "early in the morning", hour: 6},
		{name: "at midday", hour: 12},
		{name: "late at night", hour: 23, minute: 30},
	}
	for _, clock := range clocks {
		t.Run(clock.name, func(t *testing.T) {
			ctx, store, pool := newTestStore(t)
			fixture := seedEngine(t, ctx, store, pool, 1)
			now := clockAt(t, clock.hour, clock.minute)
			date := "2026-11-09"
			pending, err := store.insertItem(ctx, awaitingItem(fixture, date, "11:15", "12:00"))
			if err != nil {
				t.Fatalf("could not create the awaiting class: %v", err)
			}
			if _, err := store.ConfirmItem(ctx, pending.ID, fixture.teacherID, ModeRoom, "1003", "",
				now); err != nil {
				t.Errorf("the first confirm was refused: %v", err)
			}
			later, err := store.insertItem(ctx, awaitingItem(fixture, date, "13:00", "13:45"))
			if err != nil {
				t.Fatalf("could not create the second class: %v", err)
			}
			if _, err := store.CancelItemAsTeacher(ctx, later.ID, fixture.teacherID, "Not today",
				now); err != nil {
				t.Errorf("the first cancel was refused: %v", err)
			}
			third, err := store.insertItem(ctx, awaitingItem(fixture, date, "14:00", "14:45"))
			if err != nil {
				t.Fatalf("could not create the third class: %v", err)
			}
			if _, err := store.TeacherRequestReschedule(ctx, third.ID, fixture.teacherID,
				"Please move it", now); err != nil {
				t.Errorf("the first reschedule request was refused: %v", err)
			}
		})
	}
}

func TestTeacherDecidesOnANoResponseItem(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	fixture := seedEngine(t, ctx, store, pool, 1)
	date := "2026-11-09"
	item := awaitingItem(fixture, date, "09:00", "09:45")
	item.Status = ItemNoResponse
	stale, err := store.insertItem(ctx, item)
	if err != nil {
		t.Fatalf("could not create the no_response class: %v", err)
	}
	confirmed, err := store.ConfirmItem(ctx, stale.ID, fixture.teacherID, ModeRoom, "1003", "",
		clockAt(t, 9, 0))
	if err != nil {
		t.Fatalf("a no_response class could not be confirmed: %v", err)
	}
	if confirmed.Status != ItemConfirmed {
		t.Errorf("status = %q, want %q", confirmed.Status, ItemConfirmed)
	}
}

func TestTeacherCannotDecideAfterTheClassStarted(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	fixture := seedEngine(t, ctx, store, pool, 1)
	date := "2026-11-02"
	item, err := store.insertItem(ctx, awaitingItem(fixture, date, "09:00", "09:45"))
	if err != nil {
		t.Fatalf("could not create the awaiting class: %v", err)
	}
	after := clockAt(t, 10, 0)
	if _, err := store.ConfirmItem(ctx, item.ID, fixture.teacherID, ModeRoom, "1003", "",
		after); !errors.Is(err, ErrItemStarted) {
		t.Errorf("confirm after the start: error = %v, want ErrItemStarted", err)
	}
	if _, err := store.CancelItemAsTeacher(ctx, item.ID, fixture.teacherID, "Too late",
		after); !errors.Is(err, ErrItemStarted) {
		t.Errorf("cancel after the start: error = %v, want ErrItemStarted", err)
	}
	if _, err := store.TeacherRequestReschedule(ctx, item.ID, fixture.teacherID, "Move it",
		after); !errors.Is(err, ErrItemStarted) {
		t.Errorf("reschedule after the start: error = %v, want ErrItemStarted", err)
	}
}

func TestClassRepresentativeIsNotLimitedByTheClock(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	fixture := seedEngine(t, ctx, store, pool, 1)
	date := "2026-11-09"
	fixed, err := store.insertItem(ctx, awaitingItem(fixture, date, "09:00", "09:45"))
	if err != nil {
		t.Fatalf("could not create the fixed class: %v", err)
	}
	if _, err := store.CRRequestReschedule(ctx, fixed.ID, "CSE-23", Proposal{
		ItemDate:  date,
		StartTime: "13:45",
		EndTime:   "14:30",
		Mode:      ModeRoom,
		Room:      "1003",
	}); err != nil {
		t.Errorf("the class representative could not ask for another slot: %v", err)
	}
	extra, err := store.CreateCRItem(ctx, newItem(fixture, date, "10:30", "11:15", "A"))
	if err != nil {
		t.Fatalf("the class representative could not create a class: %v", err)
	}
	if _, err := store.SetItemFinal(ctx, extra.ID, "CSE-23", true); err != nil {
		t.Errorf("the class representative could not add the class to the final routine: %v", err)
	}
	if _, err := store.CancelItemAsCR(ctx, extra.ID, "CSE-23", "Teacher is away"); err != nil {
		t.Errorf("the class representative could not cancel the class: %v", err)
	}
}

func awaitingItem(fixture engineFixture, date, start, end string) NewScheduleItem {
	item := newItem(fixture, date, start, end, "A")
	item.Origin = OriginFixed
	item.Status = ItemAwaiting
	return item
}

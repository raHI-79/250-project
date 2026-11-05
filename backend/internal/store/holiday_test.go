package store

import (
	"errors"
	"testing"
	"time"
)

// Friday 6 and Saturday 7 November 2026 are the weekly off days, Sunday 8 November is a
// teaching day and the fixed routine in the fixture carries a Sunday class.
const (
	offFriday     = "2026-11-06"
	offSaturday   = "2026-11-07"
	workingSunday = "2026-11-08"
	nextSunday    = "2026-11-15"
)

var weekendOff = []int{5, 6}

func TestGenerateSkipsTheWeeklyOffDays(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	fixture := seedEngine(t, ctx, store, pool, 0)
	// A routine entry on every other weekday too, so only the off days can keep a date out.
	for day := 1; day < 7; day++ {
		addRoutineDay(t, ctx, pool, fixture, day)
	}
	created, err := store.GenerateFixedItems(ctx, offFriday, workingSunday, weekendOff)
	if err != nil {
		t.Fatalf("GenerateFixedItems() returned an unexpected error: %v", err)
	}
	if created != 1 {
		t.Fatalf("created = %d items, want only the Sunday one", created)
	}
	for _, date := range []string{offFriday, offSaturday} {
		items, err := store.ListBatchItemsOnDate(ctx, "CSE-23", date)
		if err != nil {
			t.Fatalf("ListBatchItemsOnDate(%s) returned an unexpected error: %v", date, err)
		}
		if len(items) != 0 {
			t.Errorf("%s carries %d generated items, want none on a weekly off day", date, len(items))
		}
	}
	sunday, err := store.ListBatchItemsOnDate(ctx, "CSE-23", workingSunday)
	if err != nil {
		t.Fatalf("ListBatchItemsOnDate(Sunday) returned an unexpected error: %v", err)
	}
	if len(sunday) != 1 {
		t.Errorf("Sunday carries %d items, want the one fixed class", len(sunday))
	}
}

func TestGenerateSkipsAHolidayOnATeachingDay(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	seedEngine(t, ctx, store, pool, 0)
	if _, err := store.SaveHoliday(ctx, Holiday{Date: workingSunday, Name: "Victory Day"}); err != nil {
		t.Fatalf("SaveHoliday() returned an unexpected error: %v", err)
	}
	created, err := store.GenerateFixedItems(ctx, workingSunday, nextSunday, weekendOff)
	if err != nil {
		t.Fatalf("GenerateFixedItems() returned an unexpected error: %v", err)
	}
	if created != 1 {
		t.Fatalf("created = %d, want only the Sunday a week later", created)
	}
	items, err := store.ListBatchItemsOnDate(ctx, "CSE-23", workingSunday)
	if err != nil {
		t.Fatalf("ListBatchItemsOnDate() returned an unexpected error: %v", err)
	}
	if len(items) != 0 {
		t.Errorf("the holiday carries %d generated items, want none", len(items))
	}
}

func TestAWorkingDayOverrideGeneratesTheChosenWeekday(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	seedEngine(t, ctx, store, pool, 0)
	// Saturday 14 November follows Sunday, so the Sunday routine has to appear on it.
	override := Holiday{Date: "2026-11-14", Name: "Make-up Saturday", WorkingDay: true, Follows: 0}
	if _, err := store.SaveHoliday(ctx, override); err != nil {
		t.Fatalf("SaveHoliday() returned an unexpected error: %v", err)
	}
	created, err := store.GenerateFixedItems(ctx, "2026-11-13", "2026-11-14", weekendOff)
	if err != nil {
		t.Fatalf("GenerateFixedItems() returned an unexpected error: %v", err)
	}
	if created != 1 {
		t.Fatalf("created = %d, want the Sunday routine on the working Saturday", created)
	}
	items, err := store.ListBatchItemsOnDate(ctx, "CSE-23", "2026-11-14")
	if err != nil {
		t.Fatalf("ListBatchItemsOnDate() returned an unexpected error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("the working Saturday carries %d items, want one", len(items))
	}
	if items[0].Origin != OriginFixed || items[0].Status != ItemAwaiting {
		t.Errorf("the generated item = %s/%s, want a fixed item awaiting its teacher",
			items[0].Origin, items[0].Status)
	}
}

func TestAddingAHolidayRemovesOnlyTheUndecidedGeneratedItems(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	fixture := seedEngine(t, ctx, store, pool, 0)
	if _, err := store.GenerateFixedItems(ctx, workingSunday, workingSunday, weekendOff); err != nil {
		t.Fatalf("GenerateFixedItems() returned an unexpected error: %v", err)
	}
	generated, err := store.ListBatchItemsOnDate(ctx, "CSE-23", workingSunday)
	if err != nil {
		t.Fatalf("ListBatchItemsOnDate() returned an unexpected error: %v", err)
	}
	if len(generated) != 1 {
		t.Fatalf("got %d generated items, want one to remove later", len(generated))
	}
	// A class the class representative created for the same day has to survive.
	created, err := store.CreateCRItem(ctx, newItem(fixture, workingSunday, "10:00", "11:00", "A"))
	if err != nil {
		t.Fatalf("CreateCRItem() returned an unexpected error: %v", err)
	}
	removed, err := store.SaveHoliday(ctx, Holiday{Date: workingSunday, Name: "Strike"})
	if err != nil {
		t.Fatalf("SaveHoliday() returned an unexpected error: %v", err)
	}
	if len(removed) != 1 || removed[0].ID != generated[0].ID {
		t.Fatalf("removed = %+v, want only the generated item", removed)
	}
	left, err := store.ListBatchItemsOnDate(ctx, "CSE-23", workingSunday)
	if err != nil {
		t.Fatalf("ListBatchItemsOnDate() returned an unexpected error: %v", err)
	}
	if len(left) != 1 || left[0].ID != created.ID {
		t.Errorf("the day holds %+v, want only the class the representative created", left)
	}
}

func TestAddingAHolidayKeepsADecidedGeneratedItem(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	fixture := seedEngine(t, ctx, store, pool, 0)
	if _, err := store.GenerateFixedItems(ctx, workingSunday, workingSunday, weekendOff); err != nil {
		t.Fatalf("GenerateFixedItems() returned an unexpected error: %v", err)
	}
	items, err := store.ListBatchItemsOnDate(ctx, "CSE-23", workingSunday)
	if err != nil {
		t.Fatalf("ListBatchItemsOnDate() returned an unexpected error: %v", err)
	}
	confirmed, err := store.ConfirmItem(ctx, items[0].ID, fixture.teacherID, ModeRoom, "1003", "",
		time.Date(2026, 11, 7, 10, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("ConfirmItem() returned an unexpected error: %v", err)
	}
	if confirmed.Status != ItemConfirmed {
		t.Fatalf("the item is %s, want it confirmed before the holiday is added", confirmed.Status)
	}
	removed, err := store.SaveHoliday(ctx, Holiday{Date: workingSunday, Name: "Strike"})
	if err != nil {
		t.Fatalf("SaveHoliday() returned an unexpected error: %v", err)
	}
	if len(removed) != 0 {
		t.Errorf("removed = %+v, want nothing because the teacher had already decided", removed)
	}
	left, err := store.ListBatchItemsOnDate(ctx, "CSE-23", workingSunday)
	if err != nil {
		t.Fatalf("ListBatchItemsOnDate() returned an unexpected error: %v", err)
	}
	if len(left) != 1 || left[0].Status != ItemConfirmed {
		t.Errorf("the day holds %+v, want the confirmed class", left)
	}
}

func TestHolidaysAreListedEditedAndDeleted(t *testing.T) {
	ctx, store, _ := newTestStore(t)
	if _, err := store.SaveHoliday(ctx, Holiday{Date: workingSunday, Name: "Victory Day"}); err != nil {
		t.Fatalf("SaveHoliday() returned an unexpected error: %v", err)
	}
	working := Holiday{Date: offFriday, Name: "Make-up Friday", WorkingDay: true, Follows: 1}
	if _, err := store.SaveHoliday(ctx, working); err != nil {
		t.Fatalf("SaveHoliday() returned an unexpected error: %v", err)
	}
	listed, err := store.ListHolidays(ctx, offFriday)
	if err != nil {
		t.Fatalf("ListHolidays() returned an unexpected error: %v", err)
	}
	if len(listed) != 2 {
		t.Fatalf("listed = %+v, want both entries", listed)
	}
	if listed[0].Date != offFriday || !listed[0].WorkingDay || listed[0].Follows != 1 {
		t.Errorf("the first entry = %+v, want the working Friday that follows Monday", listed[0])
	}
	if got := listed[0].FollowsName(); got != "Monday" {
		t.Errorf("FollowsName() = %q, want Monday", got)
	}
	if listed[1].WorkingDay || listed[1].Name != "Victory Day" {
		t.Errorf("the second entry = %+v, want the plain holiday", listed[1])
	}
	if _, err := store.SaveHoliday(ctx, Holiday{Date: workingSunday, Name: "Renamed"}); err != nil {
		t.Fatalf("SaveHoliday() returned an unexpected error: %v", err)
	}
	found, err := store.FindHoliday(ctx, workingSunday)
	if err != nil {
		t.Fatalf("FindHoliday() returned an unexpected error: %v", err)
	}
	if found.Name != "Renamed" {
		t.Errorf("the saved name = %q, want Renamed", found.Name)
	}
	if err := store.DeleteHoliday(ctx, workingSunday); err != nil {
		t.Fatalf("DeleteHoliday() returned an unexpected error: %v", err)
	}
	if _, err := store.FindHoliday(ctx, workingSunday); !errors.Is(err, ErrNotAHoliday) {
		t.Errorf("FindHoliday() after the delete = %v, want ErrNotAHoliday", err)
	}
	if err := store.DeleteHoliday(ctx, workingSunday); !errors.Is(err, ErrNotAHoliday) {
		t.Errorf("a second DeleteHoliday() = %v, want ErrNotAHoliday", err)
	}
}

func TestDayKindsCombineTheOffDaysAndTheCalendar(t *testing.T) {
	ctx, store, _ := newTestStore(t)
	if _, err := store.SaveHoliday(ctx, Holiday{Date: workingSunday, Name: "Victory Day"}); err != nil {
		t.Fatalf("SaveHoliday() returned an unexpected error: %v", err)
	}
	working := Holiday{Date: offSaturday, Name: "Make-up Saturday", WorkingDay: true, Follows: 0}
	if _, err := store.SaveHoliday(ctx, working); err != nil {
		t.Fatalf("SaveHoliday() returned an unexpected error: %v", err)
	}
	kinds, err := store.DayKinds(ctx, offFriday, "2026-11-09", weekendOff)
	if err != nil {
		t.Fatalf("DayKinds() returned an unexpected error: %v", err)
	}
	friday, ok := kinds[offFriday]
	if !ok || !friday.Holiday || friday.Name != "Friday" {
		t.Errorf("Friday = %+v, %v, want the weekly off day named after its weekday", friday, ok)
	}
	saturday, ok := kinds[offSaturday]
	if !ok || !saturday.WorkingDay || saturday.FollowsName() != "Sunday" {
		t.Errorf("Saturday = %+v, %v, want a working day that follows Sunday", saturday, ok)
	}
	sunday, ok := kinds[workingSunday]
	if !ok || !sunday.Holiday || sunday.Name != "Victory Day" {
		t.Errorf("Sunday = %+v, %v, want the named holiday", sunday, ok)
	}
	if _, ok := kinds["2026-11-09"]; ok {
		t.Error("Monday 9 November was called a holiday, want an ordinary teaching day")
	}
}

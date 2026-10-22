package store

import (
	"errors"
	"github.com/raHI-79/250-project/backend/internal/config"
	"testing"
	"time"
)

func windowConfig(t *testing.T) *config.Config {
	t.Helper()
	loc, err := time.LoadLocation("Asia/Dhaka")
	if err != nil {
		t.Fatalf("could not load the timezone: %v", err)
	}
	return &config.Config{
		Location:      loc,
		DecisionStart: "16:00",
		DecisionEnd:   "23:00",
		DecisionFrom:  16 * 60,
		DecisionTo:    23 * 60,
	}
}

func clockAt(t *testing.T, cfg *config.Config, hour, minute int) time.Time {
	t.Helper()
	return time.Date(2026, 11, 2, hour, minute, 0, 0, cfg.Location)
}

func TestDecisionWindowLimitsOnlyTheTeacher(t *testing.T) {
	cfg := windowConfig(t)
	clocks := []struct {
		name   string
		hour   int
		minute int
	}{
		{name: "at 10:00, before the window opens", hour: 10},
		{name: "at 23:30, after the window closes", hour: 23, minute: 30},
	}
	for _, clock := range clocks {
		t.Run(clock.name, func(t *testing.T) {
			ctx, store, pool := newTestStore(t)
			fixture := seedEngine(t, ctx, store, pool, 1)
			now := clockAt(t, cfg, clock.hour, clock.minute)
			if cfg.DecisionOpen(now) {
				t.Fatalf("the window is open at %02d:%02d, want it closed", clock.hour, clock.minute)
			}
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
			pending, err := store.insertItem(ctx, awaitingItem(fixture, date, "11:15", "12:00"))
			if err != nil {
				t.Fatalf("could not create the awaiting class: %v", err)
			}
			open := cfg.DecisionOpen(now)
			_, err = store.ConfirmItem(ctx, pending.ID, fixture.teacherID, ModeRoom, "1003", "", now, open)
			if !errors.Is(err, ErrDecisionClosed) {
				t.Errorf("the first confirm: error = %v, want ErrDecisionClosed", err)
			}
			_, err = store.CancelItemAsTeacher(ctx, pending.ID, fixture.teacherID, "Not today", open)
			if !errors.Is(err, ErrDecisionClosed) {
				t.Errorf("the first cancel: error = %v, want ErrDecisionClosed", err)
			}
			_, err = store.TeacherRequestReschedule(ctx, pending.ID, fixture.teacherID, "Please move it", open)
			if !errors.Is(err, ErrDecisionClosed) {
				t.Errorf("the first reschedule request: error = %v, want ErrDecisionClosed", err)
			}
		})
	}
}

func TestDecisionWindowLetsALaterChangeThrough(t *testing.T) {
	cfg := windowConfig(t)
	ctx, store, pool := newTestStore(t)
	fixture := seedEngine(t, ctx, store, pool, 1)
	date := "2026-11-09"
	item, err := store.insertItem(ctx, awaitingItem(fixture, date, "09:00", "09:45"))
	if err != nil {
		t.Fatalf("could not create the awaiting class: %v", err)
	}
	inside := clockAt(t, cfg, 17, 0)
	if !cfg.DecisionOpen(inside) {
		t.Fatal("the window is closed at 17:00, want it open")
	}
	confirmed, err := store.ConfirmItem(ctx, item.ID, fixture.teacherID, ModeRoom, "1003", "",
		inside, cfg.DecisionOpen(inside))
	if err != nil {
		t.Fatalf("the first confirm inside the window failed: %v", err)
	}
	if confirmed.Status != ItemConfirmed {
		t.Fatalf("status = %q, want %q", confirmed.Status, ItemConfirmed)
	}
	outside := clockAt(t, cfg, 23, 30)
	_, err = store.ConfirmItem(ctx, item.ID, fixture.teacherID, ModeRoom, "2104", "",
		outside, cfg.DecisionOpen(outside))
	if err != nil {
		t.Errorf("a later change to a confirmed class was refused: %v", err)
	}
	_, err = store.CancelItemAsTeacher(ctx, item.ID, fixture.teacherID, "Called away",
		cfg.DecisionOpen(outside))
	if err != nil {
		t.Errorf("cancelling a class already confirmed was refused: %v", err)
	}
}

func awaitingItem(fixture engineFixture, date, start, end string) NewScheduleItem {
	item := newItem(fixture, date, start, end, "A")
	item.Origin = OriginFixed
	item.Status = ItemAwaiting
	return item
}

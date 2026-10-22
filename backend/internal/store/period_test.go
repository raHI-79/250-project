package store

import (
	"errors"
	"testing"
)

func TestSeededPeriodsCoverTheTeachingDay(t *testing.T) {
	ctx, store, _ := newTestStore(t)
	periods, err := store.ListPeriods(ctx)
	if err != nil {
		t.Fatalf("ListPeriods() returned an unexpected error: %v", err)
	}
	if len(periods) != 10 {
		t.Fatalf("got %d class periods, want the ten seeded ones", len(periods))
	}
	if periods[0].Start != "09:00" || periods[len(periods)-1].End != "16:45" {
		t.Errorf("day runs %s to %s, want 09:00 to 16:45", periods[0].Start, periods[len(periods)-1].End)
	}
	breaks := 0
	for _, period := range periods {
		if period.IsBreak {
			breaks++
			if period.Start != "12:45" || period.End != "13:45" {
				t.Errorf("break = %s to %s, want 12:45 to 13:45", period.Start, period.End)
			}
		}
	}
	if breaks != 1 {
		t.Errorf("got %d breaks, want exactly one", breaks)
	}
}

func TestCreatePeriodRefusesAnOverlap(t *testing.T) {
	ctx, store, _ := newTestStore(t)
	tests := []struct {
		name      string
		period    NewPeriod
		wantError error
	}{
		{name: "after the last period", period: NewPeriod{Label: "Period 10", Start: "16:45", End: "17:30"}},
		{name: "before the first period", period: NewPeriod{Label: "Period 0", Start: "08:00", End: "09:00"}},
		{name: "over a teaching period",
			period: NewPeriod{Label: "Clash", Start: "09:30", End: "10:00"}, wantError: ErrPeriodOverlap},
		{name: "over the break",
			period: NewPeriod{Label: "Tea", Start: "13:00", End: "13:30"}, wantError: ErrPeriodOverlap},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			created, err := store.CreatePeriod(ctx, test.period)
			if test.wantError != nil {
				if !errors.Is(err, test.wantError) {
					t.Fatalf("error = %v, want %v", err, test.wantError)
				}
				return
			}
			if err != nil {
				t.Fatalf("CreatePeriod() returned an unexpected error: %v", err)
			}
			if created.Label != test.period.Label {
				t.Errorf("label = %q, want %q", created.Label, test.period.Label)
			}
		})
	}
}

func TestUpdatePeriodChecksTheOtherPeriods(t *testing.T) {
	ctx, store, _ := newTestStore(t)
	periods, err := store.ListPeriods(ctx)
	if err != nil {
		t.Fatalf("ListPeriods() returned an unexpected error: %v", err)
	}
	first := periods[0]
	label := "Morning"
	updated, err := store.UpdatePeriod(ctx, first.ID, PeriodPatch{Label: &label})
	if err != nil {
		t.Fatalf("UpdatePeriod() returned an unexpected error: %v", err)
	}
	if updated.Label != label {
		t.Errorf("label = %q, want %q", updated.Label, label)
	}
	clash := "10:00"
	if _, err := store.UpdatePeriod(ctx, first.ID, PeriodPatch{End: &clash}); !errors.Is(err, ErrPeriodOverlap) {
		t.Fatalf("error = %v, want %v", err, ErrPeriodOverlap)
	}
	if _, err := store.UpdatePeriod(ctx, 9999, PeriodPatch{Label: &label}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("error = %v, want %v", err, ErrNotFound)
	}
}

func TestReorderPeriodsKeepsEveryPeriod(t *testing.T) {
	ctx, store, _ := newTestStore(t)
	periods, err := store.ListPeriods(ctx)
	if err != nil {
		t.Fatalf("ListPeriods() returned an unexpected error: %v", err)
	}
	ids := make([]int64, 0, len(periods))
	for _, period := range periods {
		ids = append(ids, period.ID)
	}
	ids[0], ids[1] = ids[1], ids[0]
	reordered, err := store.ReorderPeriods(ctx, ids)
	if err != nil {
		t.Fatalf("ReorderPeriods() returned an unexpected error: %v", err)
	}
	if reordered[0].ID != ids[0] || reordered[1].ID != ids[1] {
		t.Errorf("order = %d, %d, want %d, %d", reordered[0].ID, reordered[1].ID, ids[0], ids[1])
	}
	if _, err := store.ReorderPeriods(ctx, ids[:2]); !errors.Is(err, ErrNotFound) {
		t.Fatalf("error = %v, want %v for a short list", err, ErrNotFound)
	}
}

func TestDeletePeriodKeepsTheLastOne(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	periods, err := store.ListPeriods(ctx)
	if err != nil {
		t.Fatalf("ListPeriods() returned an unexpected error: %v", err)
	}
	if err := store.DeletePeriod(ctx, periods[0].ID); err != nil {
		t.Fatalf("DeletePeriod() returned an unexpected error: %v", err)
	}
	if err := store.DeletePeriod(ctx, periods[0].ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("error = %v, want %v", err, ErrNotFound)
	}
	if _, err := pool.Exec(ctx, "DELETE FROM class_periods WHERE id <> $1", periods[1].ID); err != nil {
		t.Fatalf("could not trim the class periods: %v", err)
	}
	if err := store.DeletePeriod(ctx, periods[1].ID); !errors.Is(err, ErrPeriodInUse) {
		t.Fatalf("error = %v, want %v", err, ErrPeriodInUse)
	}
}

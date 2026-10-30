package jobs

import (
	"context"
	"testing"
	"time"
)

func TestClearPastFinalRunsOnStartupThenAfterMidnight(t *testing.T) {
	silenceLog(t)
	cfg := testConfig(t)
	items := &fakeItems{cleared: 3}
	scheduler := NewScheduler(cfg, items, &fakeAlerts{})
	startup := time.Date(2026, 11, 1, 14, 20, 0, 0, cfg.Location)
	cleared, err := scheduler.ClearPastFinal(context.Background(), startup)
	if err != nil {
		t.Fatalf("ClearPastFinal() returned an unexpected error: %v", err)
	}
	if cleared != 3 {
		t.Errorf("cleared = %d, want 3 on startup", cleared)
	}
	if len(items.clearedFrom) != 1 || items.clearedFrom[0] != "2026-11-01" {
		t.Fatalf("asked for %v, want today once", items.clearedFrom)
	}
	again, err := scheduler.ClearPastFinal(context.Background(),
		startup.Add(10*time.Minute))
	if err != nil {
		t.Fatalf("the second ClearPastFinal() returned an unexpected error: %v", err)
	}
	if again != 0 || len(items.clearedFrom) != 1 {
		t.Errorf("the job ran again on the same day: cleared = %d, asked for %v", again, items.clearedFrom)
	}
	midnight := time.Date(2026, 11, 2, 0, 0, 0, 0, cfg.Location)
	if _, err := scheduler.ClearPastFinal(context.Background(), midnight); err != nil {
		t.Fatalf("ClearPastFinal() at midnight returned an unexpected error: %v", err)
	}
	if len(items.clearedFrom) != 1 {
		t.Errorf("asked for %v, want nothing before five past midnight", items.clearedFrom)
	}
	past := midnight.Add(5 * time.Minute)
	if _, err := scheduler.ClearPastFinal(context.Background(), past); err != nil {
		t.Fatalf("ClearPastFinal() at five past midnight returned an unexpected error: %v", err)
	}
	if len(items.clearedFrom) != 2 || items.clearedFrom[1] != "2026-11-02" {
		t.Errorf("asked for %v, want the new day at five past midnight", items.clearedFrom)
	}
}

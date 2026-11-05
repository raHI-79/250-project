package jobs

import (
	"context"
	"testing"
	"time"

	"github.com/raHI-79/250-project/backend/internal/daydate"
	"github.com/raHI-79/250-project/backend/internal/settings"
	"github.com/raHI-79/250-project/backend/internal/store"
)

// The jobs decide which day they work on from the wall clock in Asia/Dhaka. Between six
// in the evening and midnight the UTC date is already the day before, so every one of
// these windows would pick the wrong day if any of them read a UTC date.
func dhakaAt(t *testing.T, year int, month time.Month, day, hour, minute int) time.Time {
	t.Helper()
	loc, err := time.LoadLocation("Asia/Dhaka")
	if err != nil {
		t.Fatalf("could not load the timezone: %v", err)
	}
	return time.Date(year, month, day, hour, minute, 0, 0, loc)
}

func TestEveryJobReadsTheDhakaDayAcrossMidnight(t *testing.T) {
	silenceLog(t)
	cases := []struct {
		name     string
		at       time.Time
		today    string
		tomorrow string
	}{
		{
			name:     "Thursday 23:57",
			at:       dhakaAt(t, 2026, time.November, 5, 23, 57),
			today:    "2026-11-05",
			tomorrow: "2026-11-06",
		},
		{
			name:     "Friday 00:03",
			at:       dhakaAt(t, 2026, time.November, 6, 0, 3),
			today:    "2026-11-06",
			tomorrow: "2026-11-07",
		},
		{
			name:     "Friday 05:59",
			at:       dhakaAt(t, 2026, time.November, 6, 5, 59),
			today:    "2026-11-06",
			tomorrow: "2026-11-07",
		},
		{
			name:     "Friday 06:01",
			at:       dhakaAt(t, 2026, time.November, 6, 6, 1),
			today:    "2026-11-06",
			tomorrow: "2026-11-07",
		},
		{
			name:     "Saturday 00:03",
			at:       dhakaAt(t, 2026, time.November, 7, 0, 3),
			today:    "2026-11-07",
			tomorrow: "2026-11-08",
		},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			cfg := testConfig(t)
			cfg.Freeze(test.at)
			items := &fakeItems{created: 1}
			scheduler := NewScheduler(cfg, items, &fakeAlerts{})
			if _, err := scheduler.Generate(context.Background(), cfg.Now()); err != nil {
				t.Fatalf("Generate() returned an unexpected error: %v", err)
			}
			if len(items.generated) != 1 {
				t.Fatalf("Generate() asked %d times, want once", len(items.generated))
			}
			window := items.generated[0]
			if window.from != test.today {
				t.Errorf("the window starts on %q, want today %q", window.from, test.today)
			}
			if want := daydate.Shift(test.today, cfg.DaysAhead); window.to != want {
				t.Errorf("the window ends on %q, want %q", window.to, want)
			}
			if _, err := scheduler.PromoteExams(context.Background(), cfg.Now()); err != nil {
				t.Fatalf("PromoteExams() returned an unexpected error: %v", err)
			}
			if last := items.askedDates[len(items.askedDates)-1]; last != test.tomorrow {
				t.Errorf("PromoteExams() looked at %q, want tomorrow %q", last, test.tomorrow)
			}
			if _, err := scheduler.ClearPastFinal(context.Background(), cfg.Now()); err != nil {
				t.Fatalf("ClearPastFinal() returned an unexpected error: %v", err)
			}
			if len(items.clearedFrom) != 1 || items.clearedFrom[0] != test.today {
				t.Errorf("ClearPastFinal() looked at %v, want today %q", items.clearedFrom, test.today)
			}
		})
	}
}

func TestTheDecisionDeadlineClosesTomorrowOfTheDhakaDay(t *testing.T) {
	silenceLog(t)
	// The last reminder is at 21:00, so at 23:57 on Thursday the deadline has passed and
	// the items of Friday are the ones that get no answer.
	cfg := testConfig(t)
	cfg.Freeze(dhakaAt(t, 2026, time.November, 5, 23, 57))
	items := &fakeItems{}
	scheduler := NewScheduler(cfg, items, &fakeAlerts{})
	if _, err := scheduler.CloseDecisions(context.Background(), cfg.Now()); err != nil {
		t.Fatalf("CloseDecisions() returned an unexpected error: %v", err)
	}
	if len(items.askedDates) != 1 || items.askedDates[0] != "2026-11-06" {
		t.Fatalf("CloseDecisions() looked at %v, want 2026-11-06", items.askedDates)
	}
	// Three minutes later the day has turned over. The deadline of the new day is its own
	// evening, so nothing closes yet.
	after := testConfig(t)
	after.Freeze(dhakaAt(t, 2026, time.November, 6, 0, 3))
	later := &fakeItems{}
	closed, err := NewScheduler(after, later, &fakeAlerts{}).
		CloseDecisions(context.Background(), after.Now())
	if err != nil {
		t.Fatalf("CloseDecisions() returned an unexpected error: %v", err)
	}
	if closed != 0 || len(later.askedDates) != 0 {
		t.Errorf("CloseDecisions() at 00:03 touched %v, want nothing before the evening deadline",
			later.askedDates)
	}
	// On the evening of the new day the deadline closes the items of Saturday.
	evening := testConfig(t)
	evening.Freeze(dhakaAt(t, 2026, time.November, 6, 21, 1))
	tonight := &fakeItems{}
	if _, err := NewScheduler(evening, tonight, &fakeAlerts{}).
		CloseDecisions(context.Background(), evening.Now()); err != nil {
		t.Fatalf("CloseDecisions() returned an unexpected error: %v", err)
	}
	if len(tonight.askedDates) != 1 || tonight.askedDates[0] != "2026-11-07" {
		t.Errorf("CloseDecisions() in the evening looked at %v, want 2026-11-07", tonight.askedDates)
	}
}

func TestTheMorningReminderUsesTheDhakaClock(t *testing.T) {
	silenceLog(t)
	waiting := []store.UndecidedTeacher{{UserID: 7, Count: 2}}
	// The morning reminder is set for 07:30, so 05:59 is too early and 07:31 is due.
	early := testConfig(t)
	early.Freeze(dhakaAt(t, 2026, time.November, 6, 5, 59))
	quiet := &fakeItems{todays: waiting}
	sent, err := NewScheduler(early, quiet, &fakeAlerts{}).
		RemindTeachers(context.Background(), early.Now())
	if err != nil {
		t.Fatalf("RemindTeachers() returned an unexpected error: %v", err)
	}
	if sent != 0 {
		t.Errorf("RemindTeachers() sent %d reminders at 05:59, want none", sent)
	}
	due := testConfig(t)
	due.Freeze(dhakaAt(t, 2026, time.November, 6, 7, 31))
	ready := &fakeItems{todays: waiting}
	alerts := &fakeAlerts{}
	sent, err = NewScheduler(due, ready, alerts).RemindTeachers(context.Background(), due.Now())
	if err != nil {
		t.Fatalf("RemindTeachers() returned an unexpected error: %v", err)
	}
	if sent != 1 {
		t.Fatalf("RemindTeachers() sent %d reminders at 07:31, want one", sent)
	}
	if len(ready.askedDates) != 1 || ready.askedDates[0] != "2026-11-06" {
		t.Errorf("the morning reminder looked at %v, want 2026-11-06", ready.askedDates)
	}
	if len(alerts.sent) != 1 || alerts.sent[0].title != TitleTodayPending {
		t.Errorf("the reminder = %+v, want the one about today", alerts.sent)
	}
}

func TestRemindersOnlyFireForItemsThatExist(t *testing.T) {
	silenceLog(t)
	cfg := testConfig(t)
	cfg.Freeze(dhakaAt(t, 2026, time.November, 6, 7, 31))
	// A holiday leaves no undecided item behind, so no reminder goes out even though the
	// morning slot is due.
	quiet := &fakeItems{}
	sent, err := NewScheduler(cfg, quiet, &fakeAlerts{}).
		RemindTeachers(context.Background(), cfg.Now())
	if err != nil {
		t.Fatalf("RemindTeachers() returned an unexpected error: %v", err)
	}
	if sent != 0 {
		t.Errorf("RemindTeachers() sent %d reminders, want none while nothing is undecided", sent)
	}
}

func TestOffDaysComeFromTheSettings(t *testing.T) {
	cfg := testConfig(t)
	plain := NewScheduler(cfg, &fakeItems{}, &fakeAlerts{})
	if got := plain.OffDays(context.Background()); len(got) != 2 || got[0] != 5 || got[1] != 6 {
		t.Errorf("OffDays() without settings = %v, want Friday and Saturday", got)
	}
	values := settings.New(nil, map[string]string{settings.KeyOffDays: "6"})
	only := NewScheduler(cfg, &fakeItems{}, &fakeAlerts{}).WithSettings(values)
	if got := only.OffDays(context.Background()); len(got) != 1 || got[0] != 6 {
		t.Errorf("OffDays() with one off day = %v, want only Saturday", got)
	}
	empty := settings.New(nil, map[string]string{settings.KeyOffDays: ""})
	none := NewScheduler(cfg, &fakeItems{}, &fakeAlerts{}).WithSettings(empty)
	if got := none.OffDays(context.Background()); len(got) != 2 {
		t.Errorf("OffDays() with a blank setting = %v, want the Friday and Saturday default", got)
	}
}

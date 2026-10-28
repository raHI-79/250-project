package jobs

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/raHI-79/250-project/backend/internal/store"
)

func reminderScheduler(t *testing.T, items *fakeItems) (*Scheduler, *fakeAlerts, *time.Location) {
	t.Helper()
	cfg := testConfig(t)
	alerts := &fakeAlerts{}
	return NewScheduler(cfg, items, alerts), alerts, cfg.Location
}

func at(t *testing.T, loc *time.Location, clock string) time.Time {
	t.Helper()
	moment, err := time.ParseInLocation("2006-01-02 15:04", "2026-10-28 "+clock, loc)
	if err != nil {
		t.Fatalf("could not build the clock: %v", err)
	}
	return moment
}

func TestRemindTeachersOnlyFiresInsideItsSlot(t *testing.T) {
	silenceLog(t)
	tests := []struct {
		name  string
		clock string
		want  int
	}{
		{name: "before the first slot", clock: "15:40", want: 0},
		{name: "on the first slot", clock: "16:00", want: 1},
		{name: "a tick after the first slot", clock: "16:04", want: 1},
		{name: "well after the first slot", clock: "17:30", want: 0},
		{name: "on the second slot", clock: "21:02", want: 1},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			items := &fakeItems{undecided: []store.UndecidedTeacher{{UserID: 9, Count: 3}}}
			scheduler, alerts, loc := reminderScheduler(t, items)
			sent, err := scheduler.RemindTeachers(context.Background(), at(t, loc, test.clock))
			if err != nil {
				t.Fatalf("RemindTeachers() returned %v", err)
			}
			if sent != test.want {
				t.Fatalf("sent = %d, want %d", sent, test.want)
			}
			if len(alerts.sent) != test.want {
				t.Fatalf("notified %d teachers, want %d", len(alerts.sent), test.want)
			}
		})
	}
}

func TestRemindTeachersSendsEachSlotOncePerDay(t *testing.T) {
	silenceLog(t)
	items := &fakeItems{undecided: []store.UndecidedTeacher{{UserID: 9, Count: 2}}}
	scheduler, alerts, loc := reminderScheduler(t, items)
	for _, clock := range []string{"16:00", "16:03", "16:06"} {
		if _, err := scheduler.RemindTeachers(context.Background(), at(t, loc, clock)); err != nil {
			t.Fatalf("RemindTeachers() returned %v", err)
		}
	}
	if len(alerts.sent) != 1 {
		t.Fatalf("notified %d times, want 1", len(alerts.sent))
	}
	if _, err := scheduler.RemindTeachers(context.Background(), at(t, loc, "21:00")); err != nil {
		t.Fatalf("RemindTeachers() returned %v", err)
	}
	if len(alerts.sent) != 2 {
		t.Fatalf("notified %d times, want the second slot too", len(alerts.sent))
	}
	if alerts.sent[0].title != TitleDecideNeeded {
		t.Errorf("first title = %q, want %q", alerts.sent[0].title, TitleDecideNeeded)
	}
	if alerts.sent[1].title != TitleStillWaiting {
		t.Errorf("second title = %q, want %q", alerts.sent[1].title, TitleStillWaiting)
	}
}

func TestRemindTeachersWordsTheCount(t *testing.T) {
	silenceLog(t)
	items := &fakeItems{undecided: []store.UndecidedTeacher{{UserID: 1, Count: 1}, {UserID: 2, Count: 4}}}
	scheduler, alerts, loc := reminderScheduler(t, items)
	if _, err := scheduler.RemindTeachers(context.Background(), at(t, loc, "16:00")); err != nil {
		t.Fatalf("RemindTeachers() returned %v", err)
	}
	if len(alerts.sent) != 2 {
		t.Fatalf("notified %d teachers, want 2", len(alerts.sent))
	}
	for _, sent := range alerts.sent {
		if sent.link.Kind != "teacher_next_day" {
			t.Errorf("link = %q, want the next day page", sent.link.Kind)
		}
	}
}

func TestMorningReminderLooksAtTodayFromNow(t *testing.T) {
	silenceLog(t)
	items := &fakeItems{todays: []store.UndecidedTeacher{{UserID: 7, Count: 2}}}
	scheduler, alerts, loc := reminderScheduler(t, items)
	sent, err := scheduler.RemindTeachers(context.Background(), at(t, loc, "07:31"))
	if err != nil {
		t.Fatalf("RemindTeachers() returned %v", err)
	}
	if sent != 1 || len(alerts.sent) != 1 {
		t.Fatalf("sent = %d, notified %d, want one morning reminder", sent, len(alerts.sent))
	}
	if alerts.sent[0].title != TitleTodayPending {
		t.Errorf("title = %q, want %q", alerts.sent[0].title, TitleTodayPending)
	}
	if len(items.askedDates) != 1 || !strings.HasSuffix(items.askedDates[0], "2026-10-28") {
		t.Errorf("asked dates = %v, want today", items.askedDates)
	}
}

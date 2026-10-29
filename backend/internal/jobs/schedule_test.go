package jobs

import (
	"context"
	"errors"
	"fmt"
	"github.com/raHI-79/250-project/backend/internal/config"
	"github.com/raHI-79/250-project/backend/internal/notify"
	"github.com/raHI-79/250-project/backend/internal/store"
	"io"
	"log"
	"os"
	"testing"
	"time"
)

type generateCall struct {
	from string
	to   string
}

type sentNotification struct {
	userID int64
	kind   string
	title  string
	link   store.NotificationLink
}

type fakeItems struct {
	generated  []generateCall
	created    int
	noResponse []store.ScheduleItem
	exams      []store.ScheduleItem
	crs        []int64
	askedDates []string
	undecided  []store.UndecidedTeacher
	todays     []store.UndecidedTeacher
	claimed    []string
	taken      map[string]bool
	low        []store.AttendanceAlert
	warned     []string
	err        error
}

func (f *fakeItems) GenerateFixedItems(ctx context.Context, from, to string) (int, error) {
	f.generated = append(f.generated, generateCall{from: from, to: to})
	return f.created, f.err
}

func (f *fakeItems) MarkNoResponse(ctx context.Context, date string) ([]store.ScheduleItem, error) {
	f.askedDates = append(f.askedDates, date)
	return f.noResponse, f.err
}

func (f *fakeItems) PromoteExamsToFinal(ctx context.Context, date string) ([]store.ScheduleItem, error) {
	f.askedDates = append(f.askedDates, date)
	return f.exams, f.err
}

func (f *fakeItems) ListActiveCRs(ctx context.Context, batch, section string) ([]int64, error) {
	return f.crs, nil
}

func (f *fakeItems) ListUndecidedTeachers(ctx context.Context, date string) ([]store.UndecidedTeacher, error) {
	f.askedDates = append(f.askedDates, date)
	return f.undecided, f.err
}

func (f *fakeItems) ListUndecidedTeachersFrom(ctx context.Context, date,
	after string) ([]store.UndecidedTeacher, error) {
	f.askedDates = append(f.askedDates, date)
	return f.todays, f.err
}

func (f *fakeItems) ClaimReminder(ctx context.Context, userID int64, slot, date string) (bool, error) {
	if f.taken == nil {
		f.taken = make(map[string]bool)
	}
	key := fmt.Sprintf("%d|%s|%s", userID, slot, date)
	if f.taken[key] {
		return false, nil
	}
	f.taken[key] = true
	f.claimed = append(f.claimed, key)
	return true, nil
}

func (f *fakeItems) ListLowAttendance(ctx context.Context, minSessions int,
	threshold float64) ([]store.AttendanceAlert, error) {
	return f.low, f.err
}

func (f *fakeItems) ClaimAttendanceAlert(ctx context.Context, studentID, courseID int64,
	before time.Time) (bool, error) {
	if f.taken == nil {
		f.taken = make(map[string]bool)
	}
	key := fmt.Sprintf("alert|%d|%d", studentID, courseID)
	if f.taken[key] {
		return false, nil
	}
	f.taken[key] = true
	f.warned = append(f.warned, key)
	return true, nil
}

type fakeAlerts struct {
	sent []sentNotification
}

func (f *fakeAlerts) NotifyUser(ctx context.Context, userID int64, kind, title, body string,
	link store.NotificationLink) error {
	f.sent = append(f.sent, sentNotification{userID: userID, kind: kind, title: title, link: link})
	return nil
}

func testConfig(t *testing.T) *config.Config {
	t.Helper()
	loc, err := time.LoadLocation("Asia/Dhaka")
	if err != nil {
		t.Fatalf("could not load the timezone: %v", err)
	}
	return &config.Config{
		Location:        loc,
		DaysAhead:       7,
		ReminderTimes:   []string{"16:00", "21:00"},
		MorningReminder: "07:30",
	}
}

func silenceLog(t *testing.T) {
	t.Helper()
	log.SetOutput(io.Discard)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })
}

func plannedItem() store.ScheduleItem {
	return store.ScheduleItem{
		ID:         4,
		Batch:      "CSE-23",
		Section:    "A",
		CourseCode: "CSE 2243 7637D",
		ItemDate:   "2026-10-23",
		StartTime:  "08:00",
		EndTime:    "09:30",
		Status:     store.ItemAwaiting,
	}
}

func TestGenerateAsksForTheWholeWindow(t *testing.T) {
	silenceLog(t)
	items := &fakeItems{created: 6}
	cfg := testConfig(t)
	scheduler := NewScheduler(cfg, items, &fakeAlerts{})
	now := time.Date(2026, 10, 22, 9, 0, 0, 0, cfg.Location)
	created, err := scheduler.Generate(context.Background(), now)
	if err != nil {
		t.Fatalf("Generate() returned an unexpected error: %v", err)
	}
	if created != 6 {
		t.Errorf("created = %d, want 6", created)
	}
	if len(items.generated) != 1 {
		t.Fatalf("the store was asked %d times, want once", len(items.generated))
	}
	want := generateCall{from: "2026-10-22", to: "2026-10-29"}
	if items.generated[0] != want {
		t.Errorf("generated = %+v, want %+v", items.generated[0], want)
	}
}

func TestGenerateReportsStoreFailures(t *testing.T) {
	silenceLog(t)
	items := &fakeItems{err: errors.New("database is down")}
	cfg := testConfig(t)
	scheduler := NewScheduler(cfg, items, &fakeAlerts{})
	if _, err := scheduler.Generate(context.Background(), cfg.Now()); err == nil {
		t.Fatal("Generate() swallowed the store failure")
	}
}

func TestCloseDecisions(t *testing.T) {
	silenceLog(t)
	cfg := testConfig(t)
	tests := []struct {
		name       string
		clock      time.Time
		items      []store.ScheduleItem
		wantClosed int
		wantAsked  int
		wantAlerts int
		wantLink   string
	}{
		{
			name:       "before the window opens nothing closes",
			clock:      time.Date(2026, 10, 22, 9, 0, 0, 0, cfg.Location),
			items:      []store.ScheduleItem{plannedItem()},
			wantClosed: 0,
			wantAsked:  0,
		},
		{
			name:       "inside the window nothing closes",
			clock:      time.Date(2026, 10, 22, 18, 0, 0, 0, cfg.Location),
			items:      []store.ScheduleItem{plannedItem()},
			wantClosed: 0,
			wantAsked:  0,
		},
		{
			name:       "at the deadline the items close",
			clock:      time.Date(2026, 10, 22, 23, 0, 0, 0, cfg.Location),
			items:      []store.ScheduleItem{plannedItem()},
			wantClosed: 1,
			wantAsked:  1,
			wantAlerts: 2,
			wantLink:   notify.LinkCRUpcoming,
		},
		{
			name:       "after the deadline with nothing left",
			clock:      time.Date(2026, 10, 22, 23, 30, 0, 0, cfg.Location),
			items:      nil,
			wantClosed: 0,
			wantAsked:  1,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			items := &fakeItems{noResponse: test.items, crs: []int64{11, 12}}
			alerts := &fakeAlerts{}
			scheduler := NewScheduler(cfg, items, alerts)
			closed, err := scheduler.CloseDecisions(context.Background(), test.clock)
			if err != nil {
				t.Fatalf("CloseDecisions() returned an unexpected error: %v", err)
			}
			if closed != test.wantClosed {
				t.Errorf("closed = %d, want %d", closed, test.wantClosed)
			}
			if len(items.askedDates) != test.wantAsked {
				t.Fatalf("the store was asked %v, want %d call(s)", items.askedDates, test.wantAsked)
			}
			if test.wantAsked > 0 && items.askedDates[0] != "2026-10-23" {
				t.Errorf("asked for %q, want tomorrow 2026-10-23", items.askedDates[0])
			}
			if len(alerts.sent) != test.wantAlerts {
				t.Fatalf("notifications = %d, want %d", len(alerts.sent), test.wantAlerts)
			}
			for _, sent := range alerts.sent {
				if sent.title != TitleNoResponse {
					t.Errorf("title = %q, want %q", sent.title, TitleNoResponse)
				}
				if sent.link.Kind != test.wantLink || sent.link.Ref != 4 {
					t.Errorf("link = %+v, want %q for item 4", sent.link, test.wantLink)
				}
			}
		})
	}
}

func TestPromoteExams(t *testing.T) {
	silenceLog(t)
	cfg := testConfig(t)
	exam := plannedItem()
	exam.Kind = store.KindExam
	exam.Origin = store.OriginTeacherExam
	exam.StartTime, exam.EndTime = "", ""
	items := &fakeItems{exams: []store.ScheduleItem{exam}, crs: []int64{11}}
	alerts := &fakeAlerts{}
	scheduler := NewScheduler(cfg, items, alerts)
	now := time.Date(2026, 10, 22, 9, 0, 0, 0, cfg.Location)
	promoted, err := scheduler.PromoteExams(context.Background(), now)
	if err != nil {
		t.Fatalf("PromoteExams() returned an unexpected error: %v", err)
	}
	if promoted != 1 {
		t.Errorf("promoted = %d, want 1", promoted)
	}
	if len(items.askedDates) != 1 || items.askedDates[0] != "2026-10-23" {
		t.Errorf("asked for %v, want tomorrow", items.askedDates)
	}
	if len(alerts.sent) != 1 {
		t.Fatalf("notifications = %d, want one to the class representative", len(alerts.sent))
	}
	if alerts.sent[0].title != TitleExamInFinal {
		t.Errorf("title = %q, want %q", alerts.sent[0].title, TitleExamInFinal)
	}
	if alerts.sent[0].link.Kind != notify.LinkCRFinal {
		t.Errorf("link = %+v, want the final routine page", alerts.sent[0].link)
	}
}

func TestRunStopsWithTheContext(t *testing.T) {
	silenceLog(t)
	cfg := testConfig(t)
	items := &fakeItems{}
	scheduler := NewScheduler(cfg, items, &fakeAlerts{})
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		scheduler.Run(ctx, time.Hour)
		close(done)
	}()
	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Run() did not stop when the context was cancelled")
	}
	if len(items.generated) == 0 {
		t.Error("Run() did not do a first pass before waiting for the ticker")
	}
}

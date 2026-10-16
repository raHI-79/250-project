package jobs

import (
	"context"
	"errors"
	"github.com/raHI-79/250-project/backend/internal/store"
	"io"
	"log"
	"os"
	"strings"
	"testing"
	"time"
)

type notifyCall struct {
	userID int64
	kind   string
	title  string
	body   string
}

type fakeStore struct {
	cancelled []store.EventDecision
	err       error
	askedNow  time.Time
	calls     int
}

func (f *fakeStore) AutoCancelStaleRequests(ctx context.Context, now time.Time) ([]store.EventDecision, error) {
	f.calls++
	f.askedNow = now
	if f.err != nil {
		return nil, f.err
	}
	return f.cancelled, nil
}

type fakeNotifier struct {
	calls []notifyCall
	err   error
}

func (f *fakeNotifier) NotifyUser(ctx context.Context, userID int64, kind, title, body string) error {
	f.calls = append(f.calls, notifyCall{userID: userID, kind: kind, title: title, body: body})
	return f.err
}

func silenceLog(t *testing.T) {
	t.Helper()
	log.SetOutput(io.Discard)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })
}

func staleRequest(id, createdBy int64, eventType, date string) store.EventDecision {
	return store.EventDecision{
		ID:         id,
		CreatedBy:  createdBy,
		EventType:  eventType,
		CourseCode: "CSE-233",
		Batch:      "CSE-23",
		Section:    "A",
		EventDate:  date,
		StartTime:  "10:30",
		EndTime:    "12:45",
		Room:       "1003",
		Status:     store.EventStatusAutoCancelled,
	}
}

func TestAutoCancel(t *testing.T) {
	silenceLog(t)
	now := time.Date(2026, 10, 17, 9, 0, 0, 0, time.UTC)
	tests := []struct {
		name       string
		cancelled  []store.EventDecision
		storeErr   error
		notifyErr  error
		wantCount  int
		wantCalls  int
		wantFailed bool
	}{
		{name: "nothing to cancel", wantCount: 0, wantCalls: 0},
		{
			name:      "one request",
			cancelled: []store.EventDecision{staleRequest(12, 31, "class", "2026-10-17")},
			wantCount: 1,
			wantCalls: 1,
		},
		{
			name: "several requests",
			cancelled: []store.EventDecision{
				staleRequest(12, 31, "class", "2026-10-17"),
				staleRequest(13, 32, "exam", "2026-10-16"),
			},
			wantCount: 2,
			wantCalls: 2,
		},
		{
			name:      "notification failure is not fatal",
			cancelled: []store.EventDecision{staleRequest(12, 31, "class", "2026-10-17")},
			notifyErr: errors.New("notification store is down"),
			wantCount: 1,
			wantCalls: 1,
		},
		{name: "store failure", storeErr: errors.New("database is down"), wantFailed: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			events := &fakeStore{cancelled: test.cancelled, err: test.storeErr}
			alerts := &fakeNotifier{err: test.notifyErr}
			count, err := NewAutoCanceller(events, alerts).AutoCancel(context.Background(), now)
			if test.wantFailed {
				if err == nil {
					t.Fatal("the store failure was swallowed")
				}
				if len(alerts.calls) != 0 {
					t.Error("a notification was sent although the update failed")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if count != test.wantCount {
				t.Errorf("count = %d, want %d", count, test.wantCount)
			}
			if !events.askedNow.Equal(now) {
				t.Errorf("the store was asked with %v, want %v", events.askedNow, now)
			}
			if len(alerts.calls) != test.wantCalls {
				t.Fatalf("notifications = %v, want %d", alerts.calls, test.wantCalls)
			}
			for i, call := range alerts.calls {
				if call.userID != test.cancelled[i].CreatedBy {
					t.Errorf("notification %d went to %d, want %d", i, call.userID, test.cancelled[i].CreatedBy)
				}
				if call.title != NotificationTitle {
					t.Errorf("notification title = %q, want %q", call.title, NotificationTitle)
				}
				if !strings.Contains(call.body, NotificationBody) || !strings.Contains(call.body, test.cancelled[i].EventDate) {
					t.Errorf("notification body = %q, want the reason and the date", call.body)
				}
			}
		})
	}
}

func TestRunStopsWithTheContext(t *testing.T) {
	silenceLog(t)
	events := &fakeStore{}
	alerts := &fakeNotifier{}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		NewAutoCanceller(events, alerts).Run(ctx, time.Hour, func() time.Time { return time.Now() })
		close(done)
	}()
	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Run() did not stop when the context was cancelled")
	}
	if events.calls != 1 {
		t.Errorf("the store was asked %d times, want once at startup", events.calls)
	}
}

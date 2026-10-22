package push

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/raHI-79/250-project/backend/internal/store"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"testing"
	"time"
)

type sentPush struct {
	endpoint string
	payload  Payload
}

type fakeSender struct {
	mu      sync.Mutex
	sent    []sentPush
	status  map[string]int
	failing map[string]bool
}

func (f *fakeSender) Send(ctx context.Context, subscription store.PushSubscription,
	message []byte) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.failing[subscription.Endpoint] {
		return 0, errors.New("the push service is unreachable")
	}
	var payload Payload
	if err := json.Unmarshal(message, &payload); err != nil {
		return 0, err
	}
	f.sent = append(f.sent, sentPush{endpoint: subscription.Endpoint, payload: payload})
	if status, ok := f.status[subscription.Endpoint]; ok {
		return status, nil
	}
	return http.StatusCreated, nil
}

func (f *fakeSender) calls() []sentPush {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]sentPush(nil), f.sent...)
}

type fakeSubscriptions struct {
	mu      sync.Mutex
	rows    []store.PushSubscription
	asked   []int64
	dropped []string
	touched []string
	err     error
}

func (f *fakeSubscriptions) ListPushSubscriptions(ctx context.Context,
	userIDs []int64) ([]store.PushSubscription, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return nil, f.err
	}
	f.asked = append(f.asked, userIDs...)
	return f.rows, nil
}

func (f *fakeSubscriptions) DropPushSubscription(ctx context.Context, endpoint string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.dropped = append(f.dropped, endpoint)
	return nil
}

func (f *fakeSubscriptions) TouchPushSubscription(ctx context.Context, endpoint string, at time.Time) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.touched = append(f.touched, endpoint)
	return nil
}

func (f *fakeSubscriptions) state() ([]string, []string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.dropped...), append([]string(nil), f.touched...)
}

func silence(t *testing.T) {
	t.Helper()
	log.SetOutput(io.Discard)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })
}

func waitFor(t *testing.T, check func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if check() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("the worker did not finish in time")
}

func TestWorkerSendsToEverySubscription(t *testing.T) {
	silence(t)
	subscriptions := &fakeSubscriptions{rows: []store.PushSubscription{
		{ID: 1, UserID: 4, Endpoint: "https://push.example/a", P256dh: "p", Auth: "a"},
		{ID: 2, UserID: 5, Endpoint: "https://push.example/b", P256dh: "p", Auth: "a"},
	}}
	sender := &fakeSender{}
	worker := NewWorker(subscriptions, sender, 4)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	worker.Run(ctx, 2)
	worker.Queue([]int64{4, 5}, Payload{
		Title: "New notice",
		Body:  "Quiz on Monday",
		URL:   "http://localhost:5173/student/notices#notice-3",
		Tag:   "notice",
	})
	waitFor(t, func() bool { return len(sender.calls()) == 2 })
	for _, call := range sender.calls() {
		if call.payload.Title != "New notice" || call.payload.Tag != "notice" {
			t.Errorf("payload = %+v, want the queued notification", call.payload)
		}
		if call.payload.URL == "" {
			t.Error("the payload has no url for the service worker to open")
		}
	}
	_, touched := subscriptions.state()
	waitFor(t, func() bool {
		_, used := subscriptions.state()
		return len(used) == 2
	})
	if len(touched) > 2 {
		t.Errorf("touched = %v, want at most one entry per subscription", touched)
	}
}

func TestWorkerDropsDeadSubscriptions(t *testing.T) {
	silence(t)
	subscriptions := &fakeSubscriptions{rows: []store.PushSubscription{
		{ID: 1, UserID: 4, Endpoint: "https://push.example/gone", P256dh: "p", Auth: "a"},
		{ID: 2, UserID: 4, Endpoint: "https://push.example/missing", P256dh: "p", Auth: "a"},
		{ID: 3, UserID: 4, Endpoint: "https://push.example/live", P256dh: "p", Auth: "a"},
		{ID: 4, UserID: 4, Endpoint: "https://push.example/broken", P256dh: "p", Auth: "a"},
	}}
	sender := &fakeSender{
		status: map[string]int{
			"https://push.example/gone":    http.StatusGone,
			"https://push.example/missing": http.StatusNotFound,
		},
		failing: map[string]bool{"https://push.example/broken": true},
	}
	worker := NewWorker(subscriptions, sender, 4)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	worker.Run(ctx, 1)
	worker.Queue([]int64{4}, Payload{Title: "New message", Body: "Hello", Tag: "message"})
	waitFor(t, func() bool {
		dropped, touched := subscriptions.state()
		return len(dropped) == 2 && len(touched) == 1
	})
	dropped, touched := subscriptions.state()
	for _, endpoint := range dropped {
		if endpoint != "https://push.example/gone" && endpoint != "https://push.example/missing" {
			t.Errorf("dropped %q, want only the 404 and 410 endpoints", endpoint)
		}
	}
	if len(touched) != 1 || touched[0] != "https://push.example/live" {
		t.Errorf("touched = %v, want only the living subscription", touched)
	}
}

func TestWorkerIgnoresEmptyWork(t *testing.T) {
	silence(t)
	subscriptions := &fakeSubscriptions{}
	sender := &fakeSender{}
	worker := NewWorker(subscriptions, sender, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	worker.Run(ctx, 1)
	worker.Queue(nil, Payload{Title: "Nobody"})
	worker.Queue([]int64{4}, Payload{})
	time.Sleep(50 * time.Millisecond)
	if len(sender.calls()) != 0 {
		t.Errorf("sent %d notifications, want none", len(sender.calls()))
	}
}

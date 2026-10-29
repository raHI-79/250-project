package fcm

import (
	"context"
	"errors"
	"io"
	"log"
	"os"
	"testing"

	"github.com/raHI-79/250-project/backend/internal/store"
)

type fakeTokenStore struct {
	tokens  []store.FCMToken
	dropped []string
	touched []string
	listErr error
}

func (f *fakeTokenStore) ListFCMTokens(ctx context.Context, userIDs []int64) ([]store.FCMToken, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	return f.tokens, nil
}

func (f *fakeTokenStore) DropFCMToken(ctx context.Context, token string) error {
	f.dropped = append(f.dropped, token)
	return nil
}

func (f *fakeTokenStore) TouchFCMTokens(ctx context.Context, tokens []string) error {
	f.touched = append(f.touched, tokens...)
	return nil
}

type fakeSender struct {
	failures  map[string]error
	delivered []string
	off       bool
}

func (f *fakeSender) Configured() bool {
	return !f.off
}

func (f *fakeSender) Send(ctx context.Context, message Message) error {
	if err := f.failures[message.Token]; err != nil {
		return err
	}
	f.delivered = append(f.delivered, message.Token)
	return nil
}

func quietLog(t *testing.T) {
	t.Helper()
	log.SetOutput(io.Discard)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })
}

func deviceTokens(values ...string) []store.FCMToken {
	tokens := make([]store.FCMToken, 0, len(values))
	for index, value := range values {
		tokens = append(tokens, store.FCMToken{ID: int64(index + 1), UserID: 7, Token: value})
	}
	return tokens
}

func TestWorkerDeliversToEveryDevice(t *testing.T) {
	quietLog(t)
	tokens := &fakeTokenStore{tokens: deviceTokens("a", "b")}
	sender := &fakeSender{}
	sent := NewWorker(tokens, sender).Deliver(context.Background(), []int64{7}, "Title", "Body",
		"https://portal.example/student/today", "notice")
	if sent != 2 {
		t.Fatalf("sent = %d, want 2", sent)
	}
	if len(sender.delivered) != 2 {
		t.Errorf("delivered = %v, want both tokens", sender.delivered)
	}
	if len(tokens.touched) != 2 {
		t.Errorf("touched = %v, want both tokens", tokens.touched)
	}
}

func TestWorkerRemovesAnUnregisteredToken(t *testing.T) {
	quietLog(t)
	tokens := &fakeTokenStore{tokens: deviceTokens("good", "stale")}
	sender := &fakeSender{failures: map[string]error{"stale": ErrGone}}
	sent := NewWorker(tokens, sender).Deliver(context.Background(), []int64{7}, "Title", "Body", "", "tag")
	if sent != 1 {
		t.Fatalf("sent = %d, want 1", sent)
	}
	if len(tokens.dropped) != 1 || tokens.dropped[0] != "stale" {
		t.Errorf("dropped = %v, want the stale token", tokens.dropped)
	}
	if len(tokens.touched) != 1 || tokens.touched[0] != "good" {
		t.Errorf("touched = %v, want only the delivered token", tokens.touched)
	}
}

func TestWorkerKeepsATokenAfterAPlainFailure(t *testing.T) {
	quietLog(t)
	tokens := &fakeTokenStore{tokens: deviceTokens("flaky")}
	sender := &fakeSender{failures: map[string]error{"flaky": errors.New("the service is down")}}
	sent := NewWorker(tokens, sender).Deliver(context.Background(), []int64{7}, "Title", "Body", "", "tag")
	if sent != 0 {
		t.Fatalf("sent = %d, want 0", sent)
	}
	if len(tokens.dropped) != 0 {
		t.Errorf("dropped = %v, want the token kept", tokens.dropped)
	}
}

func TestWorkerStaysQuietWithoutASender(t *testing.T) {
	quietLog(t)
	tokens := &fakeTokenStore{tokens: deviceTokens("a")}
	worker := NewWorker(tokens, &fakeSender{off: true})
	if worker.Ready() {
		t.Error("Ready() is true although the sender is not configured")
	}
	if sent := worker.Deliver(context.Background(), []int64{7}, "Title", "Body", "", "tag"); sent != 0 {
		t.Errorf("sent = %d, want 0", sent)
	}
	if sent := NewWorker(tokens, &fakeSender{}).Deliver(context.Background(), nil, "T", "B", "", ""); sent != 0 {
		t.Errorf("sent = %d with no recipients, want 0", sent)
	}
}

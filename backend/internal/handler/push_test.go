package handler

import (
	"context"
	"errors"
	"github.com/raHI-79/250-project/backend/internal/auth"
	"github.com/raHI-79/250-project/backend/internal/config"
	"github.com/raHI-79/250-project/backend/internal/store"
	"net/http"
	"testing"
)

type fakePushStore struct {
	saved   []store.PushSubscription
	deleted []string
	err     error
}

func (f *fakePushStore) SavePushSubscription(ctx context.Context,
	subscription store.PushSubscription) (int64, error) {
	if f.err != nil {
		return 0, f.err
	}
	f.saved = append(f.saved, subscription)
	return int64(len(f.saved)), nil
}

func (f *fakePushStore) DeletePushSubscription(ctx context.Context, userID int64, endpoint string) error {
	if f.err != nil {
		return f.err
	}
	f.deleted = append(f.deleted, endpoint)
	return nil
}

type fakePushTester struct {
	sent   int
	userID int64
	title  string
	err    error
}

func (f *fakePushTester) SendTest(ctx context.Context, userID int64, title, body, target string) (int, error) {
	if f.err != nil {
		return 0, f.err
	}
	f.userID = userID
	f.title = title
	return f.sent, nil
}

func pushConfig(t *testing.T, enabled bool) *config.Config {
	t.Helper()
	cfg := accountConfig(t, "test")
	if enabled {
		cfg.VAPIDPublicKey = "public-key"
		cfg.VAPIDPrivateKey = "private-key"
		cfg.VAPIDSubject = "mailto:cse@sust.edu"
	}
	return cfg
}

func TestPushPublicKeyTellsWhetherPushIsOn(t *testing.T) {
	for _, test := range []struct {
		name        string
		enabled     bool
		wantEnabled bool
		wantKey     string
	}{
		{name: "with keys", enabled: true, wantEnabled: true, wantKey: "public-key"},
		{name: "without keys", enabled: false},
	} {
		t.Run(test.name, func(t *testing.T) {
			handler := NewPush(pushConfig(t, test.enabled), &fakePushStore{}, &fakePushTester{})
			request := jsonRequest(t, http.MethodGet, "/api/push/public-key", "", 0, "")
			rec := serve(handler.PublicKey(), request)
			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200 (body = %q)", rec.Code, rec.Body.String())
			}
			payload := decodeBody(t, rec)
			if payload["enabled"] != test.wantEnabled {
				t.Errorf("enabled = %v, want %v", payload["enabled"], test.wantEnabled)
			}
			if test.wantKey == "" {
				if payload["public_key"] != nil {
					t.Error("the response carries a key although push is off")
				}
				return
			}
			if payload["public_key"] != test.wantKey {
				t.Errorf("public_key = %v, want %q", payload["public_key"], test.wantKey)
			}
		})
	}
}

func TestPushSubscribeRules(t *testing.T) {
	silenceLog(t)
	complete := `{"endpoint":"https://push.example/abc","keys":{"p256dh":"key","auth":"secret"}}`
	tests := []struct {
		name       string
		enabled    bool
		body       string
		storeErr   error
		wantStatus int
		wantError  string
	}{
		{name: "a complete subscription", enabled: true, body: complete, wantStatus: http.StatusCreated},
		{
			name:       "without the keys",
			enabled:    true,
			body:       `{"endpoint":"https://push.example/abc"}`,
			wantStatus: http.StatusBadRequest,
			wantError:  MessagePushIncomplete,
		},
		{
			name:       "without an endpoint",
			enabled:    true,
			body:       `{"keys":{"p256dh":"key","auth":"secret"}}`,
			wantStatus: http.StatusBadRequest,
			wantError:  MessagePushIncomplete,
		},
		{
			name:       "while push is off",
			enabled:    false,
			body:       complete,
			wantStatus: http.StatusServiceUnavailable,
			wantError:  MessagePushOff,
		},
		{
			name:       "a broken store",
			enabled:    true,
			body:       complete,
			storeErr:   errors.New("database is down"),
			wantStatus: http.StatusInternalServerError,
			wantError:  "Internal server error",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			devices := &fakePushStore{err: test.storeErr}
			handler := NewPush(pushConfig(t, test.enabled), devices, &fakePushTester{})
			request := jsonRequest(t, http.MethodPost, "/api/push/subscribe", test.body,
				noticeCRID, auth.RoleCR)
			rec := serve(handler.Subscribe(), request)
			if rec.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d (body = %q)", rec.Code, test.wantStatus, rec.Body.String())
			}
			if test.wantError != "" {
				if got := errorMessage(t, rec); got != test.wantError {
					t.Errorf("error = %q, want %q", got, test.wantError)
				}
				if len(devices.saved) != 0 {
					t.Error("the store saved a subscription although the request was refused")
				}
				return
			}
			if len(devices.saved) != 1 {
				t.Fatalf("the store saved %d subscriptions, want 1", len(devices.saved))
			}
			saved := devices.saved[0]
			if saved.UserID != noticeCRID || saved.Endpoint != "https://push.example/abc" {
				t.Errorf("saved = %+v, want the caller and the endpoint", saved)
			}
			if saved.P256dh != "key" || saved.Auth != "secret" {
				t.Errorf("saved keys = %+v, want the browser keys", saved)
			}
		})
	}
}

func TestPushUnsubscribeNeedsAnEndpoint(t *testing.T) {
	silenceLog(t)
	devices := &fakePushStore{}
	handler := NewPush(pushConfig(t, true), devices, &fakePushTester{})
	missing := serve(handler.Unsubscribe(),
		jsonRequest(t, http.MethodDelete, "/api/push/subscribe", `{}`, noticeCRID, auth.RoleCR))
	if missing.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 (body = %q)", missing.Code, missing.Body.String())
	}
	rec := serve(handler.Unsubscribe(), jsonRequest(t, http.MethodDelete, "/api/push/subscribe",
		`{"endpoint":"https://push.example/abc"}`, noticeCRID, auth.RoleCR))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body = %q)", rec.Code, rec.Body.String())
	}
	if len(devices.deleted) != 1 || devices.deleted[0] != "https://push.example/abc" {
		t.Errorf("deleted = %v, want the endpoint the browser sent", devices.deleted)
	}
}

func TestPushTestAnswersHowManyWereSent(t *testing.T) {
	silenceLog(t)
	tests := []struct {
		name       string
		enabled    bool
		sent       int
		err        error
		wantStatus int
		wantError  string
		wantSent   float64
	}{
		{name: "two devices", enabled: true, sent: 2, wantStatus: http.StatusOK, wantSent: 2},
		{
			name:       "no device",
			enabled:    true,
			sent:       0,
			wantStatus: http.StatusNotFound,
			wantError:  MessagePushNoDevice,
		},
		{
			name:       "while push is off",
			enabled:    false,
			sent:       1,
			wantStatus: http.StatusServiceUnavailable,
			wantError:  MessagePushOff,
		},
		{
			name:       "a broken sender",
			enabled:    true,
			err:        errors.New("the push service is down"),
			wantStatus: http.StatusInternalServerError,
			wantError:  "Internal server error",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tester := &fakePushTester{sent: test.sent, err: test.err}
			handler := NewPush(pushConfig(t, test.enabled), &fakePushStore{}, tester)
			request := jsonRequest(t, http.MethodPost, "/api/push/test", "", noticeCRID, auth.RoleCR)
			rec := serve(handler.Test(), request)
			if rec.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d (body = %q)", rec.Code, test.wantStatus, rec.Body.String())
			}
			if test.wantError != "" {
				if got := errorMessage(t, rec); got != test.wantError {
					t.Errorf("error = %q, want %q", got, test.wantError)
				}
				return
			}
			payload := decodeBody(t, rec)
			if payload["sent"] != test.wantSent {
				t.Errorf("sent = %v, want %v", payload["sent"], test.wantSent)
			}
			if tester.userID != noticeCRID {
				t.Errorf("userID = %d, want the caller %d", tester.userID, noticeCRID)
			}
			if tester.title != TestPushTitle {
				t.Errorf("title = %q, want %q", tester.title, TestPushTitle)
			}
		})
	}
}

package mailer

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type captured struct {
	Key     string
	Payload brevoMessage
}

func brevoServer(t *testing.T, status int, body string) (*httptest.Server, *captured) {
	t.Helper()
	seen := &captured{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen.Key = r.Header.Get(BrevoKeyHeader)
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &seen.Payload)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)
	return server, seen
}

func TestBrevoConfigured(t *testing.T) {
	tests := []struct {
		name    string
		options Options
		want    bool
	}{
		{name: "a key and a sender", options: Options{APIKey: "secret", From: "cse@sust.edu"}, want: true},
		{name: "no key", options: Options{From: "cse@sust.edu"}},
		{name: "no sender", options: Options{APIKey: "secret"}},
		{name: "blank key", options: Options{APIKey: "   ", From: "cse@sust.edu"}},
		{name: "nothing at all", options: Options{}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := NewBrevo(test.options).Configured(); got != test.want {
				t.Errorf("Configured() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestBrevoSendsTheMessage(t *testing.T) {
	server, seen := brevoServer(t, http.StatusCreated, `{"messageId":"1"}`)
	sender := NewBrevo(Options{
		APIKey:   "secret-key",
		From:     "cse@sust.edu",
		FromName: "CSE Office",
		Endpoint: server.URL,
	})
	err := sender.Send(context.Background(), "student@student.sust.edu", "Your code", "The code is 123456.")
	if err != nil {
		t.Fatalf("Send() returned an unexpected error: %v", err)
	}
	if seen.Key != "secret-key" {
		t.Errorf("the key header was %q, want the configured key", seen.Key)
	}
	if seen.Payload.Sender.Email != "cse@sust.edu" || seen.Payload.Sender.Name != "CSE Office" {
		t.Errorf("sender = %+v, want the configured sender", seen.Payload.Sender)
	}
	if len(seen.Payload.To) != 1 || seen.Payload.To[0].Email != "student@student.sust.edu" {
		t.Errorf("recipients = %+v, want the single student", seen.Payload.To)
	}
	if seen.Payload.Subject != "Your code" || seen.Payload.TextContent != "The code is 123456." {
		t.Errorf("message = %+v, want the subject and body as given", seen.Payload)
	}
}

func TestBrevoUsesTheDefaultSenderName(t *testing.T) {
	server, seen := brevoServer(t, http.StatusCreated, `{}`)
	sender := NewBrevo(Options{APIKey: "secret", From: "cse@sust.edu", Endpoint: server.URL})
	if err := sender.Send(context.Background(), "student@student.sust.edu", "Hello", "Body"); err != nil {
		t.Fatalf("Send() returned an unexpected error: %v", err)
	}
	if seen.Payload.Sender.Name != defaultFromName {
		t.Errorf("sender name = %q, want %q", seen.Payload.Sender.Name, defaultFromName)
	}
}

func TestBrevoReportsRefusals(t *testing.T) {
	tests := []struct {
		name       string
		status     int
		body       string
		wantInText string
	}{
		{name: "a bad key", status: http.StatusUnauthorized,
			body: `{"code":"unauthorized","message":"Key not found"}`, wantInText: "Key not found"},
		{name: "an unverified sender", status: http.StatusBadRequest,
			body: `{"code":"invalid_parameter","message":"Sender not valid"}`, wantInText: "Sender not valid"},
		{name: "a silent refusal", status: http.StatusInternalServerError,
			body: "", wantInText: "no reason given"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server, _ := brevoServer(t, test.status, test.body)
			sender := NewBrevo(Options{APIKey: "secret", From: "cse@sust.edu", Endpoint: server.URL})
			err := sender.Send(context.Background(), "student@student.sust.edu", "Hello", "Body")
			if err == nil {
				t.Fatal("Send() accepted a refused message")
			}
			if !strings.Contains(err.Error(), test.wantInText) {
				t.Errorf("error = %q, want it to mention %q", err.Error(), test.wantInText)
			}
			if strings.Contains(err.Error(), "secret") {
				t.Error("the error carries the api key")
			}
		})
	}
}

func TestBrevoRefusesWithoutSettingsOrRecipient(t *testing.T) {
	if err := NewBrevo(Options{}).Send(context.Background(), "a@b.c", "s", "b"); !errors.Is(err, ErrNotConfigured) {
		t.Errorf("error = %v, want ErrNotConfigured", err)
	}
	sender := NewBrevo(Options{APIKey: "secret", From: "cse@sust.edu"})
	if err := sender.Send(context.Background(), "  ", "s", "b"); !errors.Is(err, ErrNoRecipient) {
		t.Errorf("error = %v, want ErrNoRecipient", err)
	}
}

func TestNewPicksTheProvider(t *testing.T) {
	tests := []struct {
		provider string
		wantSMTP bool
	}{
		{provider: "smtp", wantSMTP: true},
		{provider: "", wantSMTP: true},
		{provider: "SMTP", wantSMTP: true},
		{provider: "brevo"},
		{provider: "Brevo"},
	}
	for _, test := range tests {
		t.Run("provider "+test.provider, func(t *testing.T) {
			sender := New(test.provider, Options{Host: "localhost", APIKey: "secret", From: "cse@sust.edu"})
			_, isSMTP := sender.(*SMTP)
			if isSMTP != test.wantSMTP {
				t.Errorf("provider %q gave %T, want smtp = %v", test.provider, sender, test.wantSMTP)
			}
		})
	}
}

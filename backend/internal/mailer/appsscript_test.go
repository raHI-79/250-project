package mailer

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func scriptOptions(endpoint string) Options {
	return Options{Endpoint: endpoint, Secret: "shared-secret", FromName: "SUST CSE Portal"}
}

func TestAppsScriptNeedsAUrlAndASecret(t *testing.T) {
	tests := []struct {
		name    string
		options Options
		want    bool
	}{
		{name: "a url and a secret", options: scriptOptions("https://script.example/exec"), want: true},
		{name: "no secret", options: Options{Endpoint: "https://script.example/exec"}},
		{name: "no url", options: Options{Secret: "shared-secret"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := NewAppsScript(test.options).Configured(); got != test.want {
				t.Errorf("Configured() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestAppsScriptSendsTheMessageThroughARedirect(t *testing.T) {
	var sent appsScriptMessage
	var followed bool
	mux := http.NewServeMux()
	mux.HandleFunc("/exec", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %q, want POST", r.Method)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Errorf("content type = %q, want application/json", got)
		}
		if err := json.NewDecoder(r.Body).Decode(&sent); err != nil {
			t.Fatalf("could not decode the request: %v", err)
		}
		http.Redirect(w, r, "/echo", http.StatusFound)
	})
	mux.HandleFunc("/echo", func(w http.ResponseWriter, r *http.Request) {
		followed = true
		_, _ = io.WriteString(w, `{"ok":true}`)
	})
	server := httptest.NewServer(mux)
	defer server.Close()
	sender := NewAppsScript(scriptOptions(server.URL + "/exec"))
	if err := sender.Send(context.Background(), "teacher@sust.edu", "Your User ID", "Body text"); err != nil {
		t.Fatalf("Send() returned %v, want no error", err)
	}
	if !followed {
		t.Error("the client did not follow the redirect")
	}
	if sent.Secret != "shared-secret" || sent.To != "teacher@sust.edu" {
		t.Errorf("sent = %+v, want the secret and the recipient", sent)
	}
	if sent.Subject != "Your User ID" || sent.Body != "Body text" {
		t.Errorf("sent = %+v, want the subject and the body", sent)
	}
	if sent.Name != "SUST CSE Portal" {
		t.Errorf("name = %q, want the configured sender name", sent.Name)
	}
}

func TestAppsScriptFailsWhenTheScriptSaysNo(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"ok":false,"error":"quota reached"}`)
	}))
	defer server.Close()
	err := NewAppsScript(scriptOptions(server.URL)).Send(context.Background(), "a@b.c", "Subject", "Body")
	if err == nil {
		t.Fatal("Send() returned no error, want a refusal")
	}
	if !strings.Contains(err.Error(), "quota reached") {
		t.Errorf("error = %v, want the reason from the script", err)
	}
	if strings.Contains(err.Error(), "shared-secret") {
		t.Error("the error exposes the shared secret")
	}
}

func TestAppsScriptFailsOnABadStatusAndOnHTML(t *testing.T) {
	tests := []struct {
		name   string
		status int
		body   string
	}{
		{name: "a server error", status: http.StatusInternalServerError, body: "boom"},
		{name: "an HTML sign-in page", status: http.StatusOK, body: "<html>Sign in</html>"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(test.status)
				_, _ = io.WriteString(w, test.body)
			}))
			defer server.Close()
			err := NewAppsScript(scriptOptions(server.URL)).Send(context.Background(), "a@b.c", "S", "B")
			if err == nil {
				t.Fatal("Send() returned no error, want a failure")
			}
			if strings.Contains(err.Error(), "shared-secret") {
				t.Error("the error exposes the shared secret")
			}
		})
	}
}

func TestAppsScriptRefusesAnEmptyRecipient(t *testing.T) {
	sender := NewAppsScript(scriptOptions("https://script.example/exec"))
	if err := sender.Send(context.Background(), "  ", "S", "B"); err != ErrNoRecipient {
		t.Errorf("Send() returned %v, want %v", err, ErrNoRecipient)
	}
}

func TestNewPicksTheAppsScriptProvider(t *testing.T) {
	sender := New("AppsScript", scriptOptions("https://script.example/exec"))
	if _, ok := sender.(*AppsScript); !ok {
		t.Fatalf("New() returned %T, want *AppsScript", sender)
	}
}

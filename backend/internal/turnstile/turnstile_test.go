package turnstile

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestVerifyNeedsASecretAndAToken(t *testing.T) {
	if err := New("", "").Verify(context.Background(), "token", ""); !errors.Is(err, ErrNotConfigured) {
		t.Errorf("without a secret: error = %v, want ErrNotConfigured", err)
	}
	if err := New("secret", "").Verify(context.Background(), "  ", ""); !errors.Is(err, ErrNoToken) {
		t.Errorf("without a token: error = %v, want ErrNoToken", err)
	}
}

func TestVerifyPassesTheTokenToCloudflare(t *testing.T) {
	var gotSecret, gotToken, gotIP string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("could not read the form: %v", err)
		}
		gotSecret = r.PostForm.Get("secret")
		gotToken = r.PostForm.Get("response")
		gotIP = r.PostForm.Get("remoteip")
		_, _ = io.WriteString(w, `{"success":true}`)
	}))
	defer server.Close()
	err := New("shared-secret", server.URL).Verify(context.Background(), "a-token", "203.0.113.4")
	if err != nil {
		t.Fatalf("Verify() returned %v, want no error", err)
	}
	if gotSecret != "shared-secret" || gotToken != "a-token" || gotIP != "203.0.113.4" {
		t.Errorf("sent secret = %q token = %q ip = %q, want all three", gotSecret, gotToken, gotIP)
	}
}

func TestVerifyReportsARefusal(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"success":false,"error-codes":["timeout-or-duplicate"]}`)
	}))
	defer server.Close()
	err := New("shared-secret", server.URL).Verify(context.Background(), "a-token", "")
	if !errors.Is(err, ErrRefused) {
		t.Fatalf("Verify() returned %v, want ErrRefused", err)
	}
	if !strings.Contains(err.Error(), "timeout-or-duplicate") {
		t.Errorf("error = %v, want the Cloudflare reason", err)
	}
	if strings.Contains(err.Error(), "shared-secret") {
		t.Error("the error exposes the secret key")
	}
}

func TestVerifyFailsOnABadAnswer(t *testing.T) {
	tests := []struct {
		name   string
		status int
		body   string
	}{
		{name: "a server error", status: http.StatusBadGateway, body: "boom"},
		{name: "an HTML page", status: http.StatusOK, body: "<html>nope</html>"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(test.status)
				_, _ = io.WriteString(w, test.body)
			}))
			defer server.Close()
			err := New("shared-secret", server.URL).Verify(context.Background(), "a-token", "")
			if err == nil {
				t.Fatal("Verify() returned no error, want a failure")
			}
			if strings.Contains(err.Error(), "shared-secret") {
				t.Error("the error exposes the secret key")
			}
		})
	}
}

func TestNewFallsBackToTheCloudflareEndpoint(t *testing.T) {
	if got := New("secret", "   ").endpoint; got != VerifyEndpoint {
		t.Errorf("endpoint = %q, want %q", got, VerifyEndpoint)
	}
}

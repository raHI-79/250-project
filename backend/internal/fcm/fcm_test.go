package fcm

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"golang.org/x/oauth2"
)

const projectID = "sust-cse-portal"

func serviceAccount(t *testing.T) string {
	t.Helper()
	raw, err := json.Marshal(map[string]string{
		"type":       "service_account",
		"project_id": projectID,
	})
	if err != nil {
		t.Fatalf("could not build the service account: %v", err)
	}
	return base64.StdEncoding.EncodeToString(raw)
}

func testClient(t *testing.T, endpoint string) *Client {
	t.Helper()
	return &Client{
		projectID: projectID,
		endpoint:  strings.TrimRight(endpoint, "/"),
		tokens:    oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-access-token"}),
		client:    &http.Client{Timeout: Timeout},
		channelID: "sust_cse",
	}
}

func TestNewNeedsAServiceAccount(t *testing.T) {
	if _, err := New("   ", "", "sust_cse"); !errors.Is(err, ErrNotConfigured) {
		t.Errorf("empty account: error = %v, want ErrNotConfigured", err)
	}
	if _, err := New(base64.StdEncoding.EncodeToString([]byte("{}")), "", "sust_cse"); err == nil {
		t.Error("New() accepted a service account without a project id")
	}
	if _, err := New("not base64 and not json", "", "sust_cse"); err == nil {
		t.Error("New() accepted a broken service account")
	}
}

func TestNewReadsAPlainOrEncodedAccount(t *testing.T) {
	encoded := serviceAccount(t)
	raw, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		t.Fatalf("could not decode the fixture: %v", err)
	}
	for _, value := range []string{encoded, string(raw)} {
		client, err := New(value, "https://fcm.example", "sust_cse")
		if err != nil {
			t.Fatalf("New() returned %v", err)
		}
		if client.projectID != projectID {
			t.Errorf("projectID = %q, want %q", client.projectID, projectID)
		}
	}
}

func TestSendPostsTheMessage(t *testing.T) {
	var got envelope
	var gotAuth, gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotPath = r.URL.Path
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatalf("could not decode the request: %v", err)
		}
		_, _ = io.WriteString(w, `{"name":"projects/x/messages/1"}`)
	}))
	defer server.Close()
	err := testClient(t, server.URL).Send(context.Background(), Message{
		Token: "device-token",
		Title: "New notice",
		Body:  "Mid term routine is out",
		URL:   "https://portal.example/student/notices",
		Tag:   "notice",
	})
	if err != nil {
		t.Fatalf("Send() returned %v", err)
	}
	if gotPath != "/v1/projects/"+projectID+"/messages:send" {
		t.Errorf("path = %q, want the v1 send path", gotPath)
	}
	if gotAuth != "Bearer test-access-token" {
		t.Errorf("authorization = %q, want the bearer token", gotAuth)
	}
	if got.Message.Token != "device-token" || got.Message.Notification.Title != "New notice" {
		t.Errorf("message = %+v, want the token and the title", got.Message)
	}
	if got.Message.Data["url"] != "https://portal.example/student/notices" {
		t.Errorf("data url = %q, want the page url", got.Message.Data["url"])
	}
	if got.Message.Android.Priority != PriorityHigh || got.Message.Android.TTL != "86400s" {
		t.Errorf("android = %+v, want high priority and a 24 hour ttl", got.Message.Android)
	}
	if got.Message.Android.Notification.ChannelID != "sust_cse" {
		t.Errorf("channel = %q, want sust_cse", got.Message.Android.Notification.ChannelID)
	}
}

func TestSendNeedsAToken(t *testing.T) {
	if err := testClient(t, "https://fcm.example").Send(context.Background(),
		Message{Title: "Hi"}); !errors.Is(err, ErrNoToken) {
		t.Errorf("Send() returned %v, want ErrNoToken", err)
	}
}

func TestSendReportsAGoneToken(t *testing.T) {
	tests := []struct {
		name   string
		status int
		body   string
	}{
		{name: "unregistered", status: http.StatusNotFound, body: `{"error":{"status":"NOT_FOUND"}}`},
		{name: "gone", status: http.StatusGone, body: `{"error":{"status":"UNREGISTERED"}}`},
		{
			name:   "an invalid token",
			status: http.StatusBadRequest,
			body:   `{"error":{"status":"INVALID_ARGUMENT"}}`,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(test.status)
				_, _ = io.WriteString(w, test.body)
			}))
			defer server.Close()
			err := testClient(t, server.URL).Send(context.Background(), Message{Token: "t", Title: "Hi"})
			if !errors.Is(err, ErrGone) {
				t.Errorf("Send() returned %v, want ErrGone", err)
			}
		})
	}
}

func TestSendReportsAServerFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = io.WriteString(w, `{"error":{"status":"INTERNAL"}}`)
	}))
	defer server.Close()
	err := testClient(t, server.URL).Send(context.Background(), Message{Token: "t", Title: "Hi"})
	if err == nil || errors.Is(err, ErrGone) {
		t.Fatalf("Send() returned %v, want a plain failure", err)
	}
	if strings.Contains(err.Error(), "test-access-token") {
		t.Error("the error exposes the access token")
	}
}

package handler

import (
	"context"
	"github.com/raHI-79/250-project/backend/internal/auth"
	"github.com/raHI-79/250-project/backend/internal/config"
	"github.com/raHI-79/250-project/backend/internal/middleware"
	"github.com/raHI-79/250-project/backend/internal/store"
	"net/http"
	"net/http/httptest"
	"testing"
)

type recordingSink struct {
	events []store.AuditEvent
}

func (s *recordingSink) WriteEvent(ctx context.Context, event store.AuditEvent) error {
	s.events = append(s.events, event)
	return nil
}

func TestTrailActionFollowsTheMethod(t *testing.T) {
	tests := []struct {
		method string
		path   string
		want   string
	}{
		{method: http.MethodGet, path: "/api/admin/students", want: ""},
		{method: http.MethodPost, path: "/api/admin/courses", want: store.ActionCreated},
		{method: http.MethodPatch, path: "/api/admin/teachers/4", want: store.ActionUpdated},
		{method: http.MethodPut, path: "/api/cr/drive-link", want: store.ActionUpdated},
		{method: http.MethodDelete, path: "/api/admin/courses/4", want: store.ActionDeleted},
		{method: http.MethodPost, path: "/api/cr/final/publish", want: store.ActionPublished},
	}
	for _, tc := range tests {
		if got := TrailAction(tc.method, tc.path); got != tc.want {
			t.Errorf("TrailAction(%q, %q) = %q, want %q", tc.method, tc.path, got, tc.want)
		}
	}
}

func TestTrailTargetNamesTheArea(t *testing.T) {
	tests := map[string]string{
		"/api/admin/courses":      "courses",
		"/api/teacher/attendance": "attendance",
		"/api/messages/direct":    "direct",
		"/api/super-admin/admins": "admins",
	}
	for path, want := range tests {
		if got := TrailTarget(path); got != want {
			t.Errorf("TrailTarget(%q) = %q, want %q", path, got, want)
		}
	}
}

func TestTrailSkipsSignInAndTheBeacon(t *testing.T) {
	skipped := []string{
		"/api/auth/login",
		"/api/auth/logout",
		"/api/analytics/view",
		"/api/admin/students/4/remove",
		"/api/admin/users/4/restore",
		"/api/admin/teachers/4/deactivate",
	}
	for _, path := range skipped {
		if !skipTrail(path) {
			t.Errorf("skipTrail(%q) = false, want true", path)
		}
	}
	for _, path := range []string{"/api/admin/courses", "/api/cr/items/4", "/api/teacher/exams"} {
		if skipTrail(path) {
			t.Errorf("skipTrail(%q) = true, want false", path)
		}
	}
}

func TestTrailRecordsOnlySuccessfulChanges(t *testing.T) {
	sink := &recordingSink{}
	recorder := NewRecorder(&config.Config{}, sink)
	current := middleware.User{ID: 44, Identifier: "office.one", Role: auth.RoleAdmin}
	answer := http.StatusOK
	guarded := Trail(recorder)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(answer)
	}))
	call := func(method, path string) {
		request := httptest.NewRequest(method, path, nil)
		request = request.WithContext(middleware.WithUser(request.Context(), current))
		guarded.ServeHTTP(httptest.NewRecorder(), request)
	}
	call(http.MethodGet, "/api/admin/students")
	if len(sink.events) != 0 {
		t.Fatalf("a read was recorded: %+v", sink.events)
	}
	call(http.MethodPost, "/api/admin/courses")
	if len(sink.events) != 1 {
		t.Fatalf("recorded %d events for one change, want 1", len(sink.events))
	}
	event := sink.events[0]
	if event.Action != store.ActionCreated || event.TargetType != "courses" {
		t.Errorf("event = %+v, want a create on the courses", event)
	}
	if event.ActorID != current.ID || event.ActorRole != auth.RoleAdmin {
		t.Errorf("event actor = %d %q, want the signed-in admin", event.ActorID, event.ActorRole)
	}
	if event.Client != store.ClientWeb {
		t.Errorf("client = %q, want %q", event.Client, store.ClientWeb)
	}
	answer = http.StatusForbidden
	call(http.MethodPost, "/api/admin/courses")
	if len(sink.events) != 1 {
		t.Errorf("a refused change was recorded: %+v", sink.events)
	}
}

func TestTheRecorderHashesTheAddress(t *testing.T) {
	first := HashAddress("salt", "203.0.113.4")
	again := HashAddress("salt", "203.0.113.4")
	other := HashAddress("salt", "203.0.113.5")
	salted := HashAddress("other-salt", "203.0.113.4")
	if first == "" || first != again {
		t.Errorf("HashAddress() is not stable: %q and %q", first, again)
	}
	if first == other || first == salted {
		t.Error("HashAddress() does not separate addresses or salts")
	}
	if first == "203.0.113.4" {
		t.Error("HashAddress() kept the address in the clear")
	}
	if HashAddress("salt", "") != "" {
		t.Error("HashAddress() invented a hash for an empty address")
	}
}

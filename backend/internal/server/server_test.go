package server

import (
	"context"
	"encoding/json"
	"github.com/raHI-79/250-project/backend/internal/config"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type stubPinger struct{}

func (stubPinger) Ping(ctx context.Context) error {
	return nil
}

func newTestRouter(t *testing.T) http.Handler {
	t.Helper()
	loc, err := time.LoadLocation("Asia/Dhaka")
	if err != nil {
		t.Fatalf("could not load timezone: %v", err)
	}
	return NewRouter(&config.Config{
		Port:     "8080",
		AppEnv:   "test",
		Timezone: "Asia/Dhaka",
		Location: loc,
	}, stubPinger{})
}

func TestRouterRoutes(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		path       string
		wantStatus int
	}{
		{name: "health", method: http.MethodGet, path: "/api/health", wantStatus: http.StatusOK},
		{name: "unknown api route", method: http.MethodGet, path: "/api/does-not-exist", wantStatus: http.StatusNotFound},
		{name: "root", method: http.MethodGet, path: "/", wantStatus: http.StatusNotFound},
		{name: "wrong method on health", method: http.MethodPost, path: "/api/health", wantStatus: http.StatusNotFound},
	}
	router := newTestRouter(t)
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, nil)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)
			if rec.Code != tc.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tc.wantStatus)
			}
			if got := rec.Header().Get("Content-Type"); got != "application/json" {
				t.Errorf("Content-Type = %q, want application/json", got)
			}
			var body map[string]any
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatalf("body is not valid JSON: %v (body = %q)", err, rec.Body.String())
			}
			if tc.wantStatus == http.StatusNotFound {
				if body["error"] != "Route not found" {
					t.Errorf(`body["error"] = %v, want "Route not found"`, body["error"])
				}
				return
			}
			if body["database"] != "ok" {
				t.Errorf(`body["database"] = %v, want "ok"`, body["database"])
			}
		})
	}
}

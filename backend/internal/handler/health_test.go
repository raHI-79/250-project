package handler

import (
	"encoding/json"
	"github.com/raHI-79/250-project/backend/internal/config"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func testConfig(t *testing.T, appEnv, timezone string) *config.Config {
	t.Helper()
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		t.Fatalf("could not load timezone %q: %v", timezone, err)
	}
	return &config.Config{
		Port:     "8080",
		AppEnv:   appEnv,
		Timezone: timezone,
		Location: loc,
	}
}

func TestHealth(t *testing.T) {
	tests := []struct {
		name     string
		appEnv   string
		timezone string
	}{
		{name: "development", appEnv: "development", timezone: "Asia/Dhaka"},
		{name: "production", appEnv: "production", timezone: "UTC"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := testConfig(t, tc.appEnv, tc.timezone)
			req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
			rec := httptest.NewRecorder()
			Health(cfg).ServeHTTP(rec, req)
			if rec.Code != http.StatusOK {
				t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
			}
			if got := rec.Header().Get("Content-Type"); got != "application/json" {
				t.Errorf("Content-Type = %q, want application/json", got)
			}
			var body healthResponse
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatalf("body is not valid JSON: %v (body = %q)", err, rec.Body.String())
			}
			if body.Status != "ok" {
				t.Errorf("status field = %q, want %q", body.Status, "ok")
			}
			if body.Environment != tc.appEnv {
				t.Errorf("environment field = %q, want %q", body.Environment, tc.appEnv)
			}
			parsed, err := time.Parse(time.RFC3339, body.Time)
			if err != nil {
				t.Fatalf("time field %q is not RFC3339: %v", body.Time, err)
			}
			_, wantOffset := time.Now().In(cfg.Location).Zone()
			_, gotOffset := parsed.Zone()
			if gotOffset != wantOffset {
				t.Errorf("time offset = %d seconds, want %d seconds", gotOffset, wantOffset)
			}
		})
	}
}

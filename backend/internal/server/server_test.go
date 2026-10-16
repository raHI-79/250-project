package server

import (
	"context"
	"encoding/json"
	"github.com/raHI-79/250-project/backend/internal/auth"
	"github.com/raHI-79/250-project/backend/internal/config"
	"github.com/raHI-79/250-project/backend/internal/httpx"
	"github.com/raHI-79/250-project/backend/internal/middleware"
	"github.com/raHI-79/250-project/backend/internal/store"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

const (
	testSecret   = "0123456789abcdef0123456789abcdef"
	testFrontend = "http://localhost:5173"
	studentID    = int64(11)
	adminID      = int64(44)
)

type stubPinger struct{}

func (stubPinger) Ping(ctx context.Context) error {
	return nil
}

type stubUserStore struct{}

func (stubUserStore) GetAuthUser(ctx context.Context, userID int64) (store.AuthUser, error) {
	switch userID {
	case studentID:
		return store.AuthUser{ID: studentID, Identifier: "2023331001", Role: "student", Status: store.StatusActive}, nil
	case adminID:
		return store.AuthUser{ID: adminID, Identifier: "admin", Role: "admin", Status: store.StatusActive}, nil
	default:
		return store.AuthUser{}, store.ErrNotFound
	}
}

func newTestDependencies(t *testing.T) Dependencies {
	t.Helper()
	loc, err := time.LoadLocation("Asia/Dhaka")
	if err != nil {
		t.Fatalf("could not load timezone: %v", err)
	}
	tokens, err := auth.NewTokens(testSecret, time.Hour)
	if err != nil {
		t.Fatalf("auth.NewTokens() returned an unexpected error: %v", err)
	}
	return Dependencies{
		Config: &config.Config{
			Port:           "8080",
			AppEnv:         "test",
			Timezone:       "Asia/Dhaka",
			Location:       loc,
			JWTSecret:      testSecret,
			JWTTTL:         time.Hour,
			FrontendOrigin: testFrontend,
		},
		Database: stubPinger{},
		Tokens:   tokens,
		Users:    stubUserStore{},
	}
}

func issue(t *testing.T, deps Dependencies, userID int64, role string) string {
	t.Helper()
	tokens, err := auth.NewTokens(deps.Config.JWTSecret, deps.Config.JWTTTL)
	if err != nil {
		t.Fatalf("auth.NewTokens() returned an unexpected error: %v", err)
	}
	token, err := tokens.IssueToken(userID, role, 0)
	if err != nil {
		t.Fatalf("IssueToken() returned an unexpected error: %v", err)
	}
	return token
}

func TestRouterRoutes(t *testing.T) {
	deps := newTestDependencies(t)
	studentToken := issue(t, deps, studentID, auth.RoleStudent)
	adminToken := issue(t, deps, adminID, auth.RoleAdmin)
	tests := []struct {
		name       string
		method     string
		path       string
		token      string
		wantStatus int
		wantError  string
	}{
		{name: "health is public", method: http.MethodGet, path: "/api/health", wantStatus: http.StatusOK},
		{
			name:       "me without a token",
			method:     http.MethodGet,
			path:       "/api/me",
			wantStatus: http.StatusUnauthorized,
			wantError:  middleware.MessageUnauthorized,
		},
		{
			name:       "me with a nonsense token",
			method:     http.MethodGet,
			path:       "/api/me",
			token:      "not-a-token",
			wantStatus: http.StatusUnauthorized,
			wantError:  middleware.MessageUnauthorized,
		},
		{name: "me with a student token", method: http.MethodGet, path: "/api/me", token: studentToken, wantStatus: http.StatusOK},
		{name: "me with an admin token", method: http.MethodGet, path: "/api/me", token: adminToken, wantStatus: http.StatusOK},
		{
			name:       "unknown api route",
			method:     http.MethodGet,
			path:       "/api/does-not-exist",
			wantStatus: http.StatusNotFound,
			wantError:  "Route not found",
		},
		{
			name:       "root",
			method:     http.MethodGet,
			path:       "/",
			wantStatus: http.StatusNotFound,
			wantError:  "Route not found",
		},
		{
			name:       "wrong method on health",
			method:     http.MethodPost,
			path:       "/api/health",
			wantStatus: http.StatusNotFound,
			wantError:  "Route not found",
		},
		{
			name:       "wrong method on me",
			method:     http.MethodPost,
			path:       "/api/me",
			token:      studentToken,
			wantStatus: http.StatusNotFound,
			wantError:  "Route not found",
		},
	}
	router := NewRouter(deps)
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, nil)
			if tc.token != "" {
				req.Header.Set("Authorization", "Bearer "+tc.token)
			}
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)
			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d (body = %q)", rec.Code, tc.wantStatus, rec.Body.String())
			}
			if got := rec.Header().Get("Content-Type"); got != "application/json" {
				t.Errorf("Content-Type = %q, want application/json", got)
			}
			var body map[string]any
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatalf("body is not valid JSON: %v (body = %q)", err, rec.Body.String())
			}
			if tc.wantError != "" && body["error"] != tc.wantError {
				t.Errorf(`body["error"] = %v, want %q`, body["error"], tc.wantError)
			}
		})
	}
}

func TestMeReturnsTheCurrentUser(t *testing.T) {
	deps := newTestDependencies(t)
	tests := []struct {
		name           string
		userID         int64
		role           string
		wantIdentifier string
		wantRole       string
	}{
		{name: "student", userID: studentID, role: auth.RoleStudent, wantIdentifier: "2023331001", wantRole: auth.RoleStudent},
		{name: "admin", userID: adminID, role: auth.RoleAdmin, wantIdentifier: "admin", wantRole: auth.RoleAdmin},
	}
	router := NewRouter(deps)
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
			req.Header.Set("Authorization", "Bearer "+issue(t, deps, tc.userID, tc.role))
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)
			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200 (body = %q)", rec.Code, rec.Body.String())
			}
			var payload struct {
				ID         int64  `json:"id"`
				Identifier string `json:"identifier"`
				Role       string `json:"role"`
			}
			if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
				t.Fatalf("body is not valid JSON: %v", err)
			}
			if payload.ID != tc.userID {
				t.Errorf("id = %d, want %d", payload.ID, tc.userID)
			}
			if payload.Identifier != tc.wantIdentifier {
				t.Errorf("identifier = %q, want %q", payload.Identifier, tc.wantIdentifier)
			}
			if payload.Role != tc.wantRole {
				t.Errorf("role = %q, want %q", payload.Role, tc.wantRole)
			}
		})
	}
}

func TestAdminGroupProtectsItsRoutes(t *testing.T) {
	deps := newTestDependencies(t)
	mux := http.NewServeMux()
	AdminGroup(mux, deps).Handle("GET /api/admin/ping", func(w http.ResponseWriter, r *http.Request) {
		httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	tests := []struct {
		name       string
		token      string
		wantStatus int
		wantError  string
	}{
		{name: "without a token", wantStatus: http.StatusUnauthorized, wantError: middleware.MessageUnauthorized},
		{
			name:       "with a student token",
			token:      issue(t, deps, studentID, auth.RoleStudent),
			wantStatus: http.StatusForbidden,
			wantError:  middleware.MessageAccessDenied,
		},
		{name: "with an admin token", token: issue(t, deps, adminID, auth.RoleAdmin), wantStatus: http.StatusOK},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/admin/ping", nil)
			if tc.token != "" {
				req.Header.Set("Authorization", "Bearer "+tc.token)
			}
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)
			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d (body = %q)", rec.Code, tc.wantStatus, rec.Body.String())
			}
			if tc.wantError == "" {
				return
			}
			var body map[string]any
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatalf("body is not valid JSON: %v", err)
			}
			if body["error"] != tc.wantError {
				t.Errorf(`body["error"] = %v, want %q`, body["error"], tc.wantError)
			}
		})
	}
}

func TestRouterAppliesTheMiddlewareChain(t *testing.T) {
	deps := newTestDependencies(t)
	router := NewRouter(deps)
	tests := []struct {
		name            string
		method          string
		origin          string
		wantStatus      int
		wantAllowOrigin string
	}{
		{
			name:            "preflight from the frontend",
			method:          http.MethodOptions,
			origin:          testFrontend,
			wantStatus:      http.StatusNoContent,
			wantAllowOrigin: testFrontend,
		},
		{
			name:       "preflight from another origin",
			method:     http.MethodOptions,
			origin:     "http://evil.example.com",
			wantStatus: http.StatusNoContent,
		},
		{
			name:            "health request from the frontend",
			method:          http.MethodGet,
			origin:          testFrontend,
			wantStatus:      http.StatusOK,
			wantAllowOrigin: testFrontend,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, "/api/health", nil)
			req.Header.Set("Origin", tc.origin)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)
			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tc.wantStatus)
			}
			if got := rec.Header().Get("Access-Control-Allow-Origin"); got != tc.wantAllowOrigin {
				t.Errorf("Access-Control-Allow-Origin = %q, want %q", got, tc.wantAllowOrigin)
			}
		})
	}
}

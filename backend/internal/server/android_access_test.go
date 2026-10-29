package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/raHI-79/250-project/backend/internal/auth"
	"github.com/raHI-79/250-project/backend/internal/middleware"
)

func serveAsApp(t *testing.T, router http.Handler, method, path, token string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(method, path, nil)
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	request.Header.Set(middleware.ClientHeader, middleware.ClientAndroid)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, request)
	return rec
}

func TestAndroidAppRefusesAnOfficeToken(t *testing.T) {
	deps, router := newAccessRouter(t)
	tokens := accessTokens(t, deps)
	paths := []string{"/api/me", "/api/admin/students", "/api/notifications"}
	for _, path := range paths {
		t.Run("admin on "+path, func(t *testing.T) {
			rec := serveAsApp(t, router, http.MethodGet, path, tokens[auth.RoleAdmin])
			if rec.Code != http.StatusForbidden {
				t.Fatalf("status = %d, want 403 (body = %q)", rec.Code, rec.Body.String())
			}
			if got := errorField(t, rec); got != middleware.MessageAppForbidden {
				t.Errorf("error = %q, want %q", got, middleware.MessageAppForbidden)
			}
		})
	}
}

func TestAndroidAppAcceptsTheOtherRoles(t *testing.T) {
	deps, router := newAccessRouter(t)
	tokens := accessTokens(t, deps)
	for _, role := range []string{auth.RoleStudent, auth.RoleCR, auth.RoleTeacher} {
		t.Run(role, func(t *testing.T) {
			rec := serveAsApp(t, router, http.MethodGet, "/api/me", tokens[role])
			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200 (body = %q)", rec.Code, rec.Body.String())
			}
		})
	}
}

func TestOfficeTokenStillWorksOnTheWebsite(t *testing.T) {
	deps, router := newAccessRouter(t)
	tokens := accessTokens(t, deps)
	rec := serve(t, router, http.MethodGet, "/api/me", "", tokens[auth.RoleAdmin])
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body = %q)", rec.Code, rec.Body.String())
	}
}

func TestAdminSignInIsRefusedFromTheApp(t *testing.T) {
	_, router := newAccessRouter(t)
	body := `{"identifier":"admin","password":"a-long-password"}`
	request := httptest.NewRequest(http.MethodPost, "/api/auth/admin/login", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set(middleware.ClientHeader, middleware.ClientAndroid)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, request)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403 (body = %q)", rec.Code, rec.Body.String())
	}
	if got := errorField(t, rec); got != middleware.MessageAppForbidden {
		t.Errorf("error = %q, want %q", got, middleware.MessageAppForbidden)
	}
}

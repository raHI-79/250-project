package server

import (
	"context"
	"github.com/raHI-79/250-project/backend/internal/auth"
	"github.com/raHI-79/250-project/backend/internal/middleware"
	"github.com/raHI-79/250-project/backend/internal/store"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type emptyCRStore struct{}

func (emptyCRStore) GetStudentProfile(ctx context.Context, userID int64) (store.StudentProfile, error) {
	return store.StudentProfile{UserID: userID, Batch: "CSE-23", Section: "A", IsCR: true}, nil
}

func (emptyCRStore) ListSectionRoutines(ctx context.Context, batch, section string) ([]store.RoutineRow, error) {
	return nil, nil
}

func (emptyCRStore) CreateClassRequest(ctx context.Context, event store.NewClassEvent) (store.ClassEventRow, error) {
	return store.ClassEventRow{}, store.ErrRoomUnavailable
}

func (emptyCRStore) ListRoomBookings(ctx context.Context, date, room string) ([]store.Booking, error) {
	return nil, nil
}

func (emptyCRStore) ListClassRequests(ctx context.Context, createdBy int64, status string) ([]store.ClassEventRow, error) {
	return nil, nil
}

type silentNotifier struct{}

func (silentNotifier) NotifyUser(ctx context.Context, userID int64, kind, title, body string) error {
	return nil
}

type crRoute struct {
	method    string
	path      string
	body      string
	wantForCR int
}

func crRoutes() []crRoute {
	return []crRoute{
		{method: http.MethodGet, path: "/api/cr/routine", wantForCR: http.StatusOK},
		{method: http.MethodGet, path: "/api/cr/requests", wantForCR: http.StatusOK},
		{method: http.MethodGet, path: "/api/cr/room-availability?date=2026-10-17", wantForCR: http.StatusOK},
		{method: http.MethodPost, path: "/api/cr/class-requests", body: `{`, wantForCR: http.StatusBadRequest},
	}
}

func TestCRRoutesNeedACRToken(t *testing.T) {
	deps, router := newAccessRouter(t)
	callers := []struct {
		name       string
		token      string
		wantStatus int
		wantError  string
	}{
		{name: "without a token", wantStatus: http.StatusUnauthorized, wantError: middleware.MessageUnauthorized},
		{
			name:       "with a student token",
			token:      issue(t, deps, accessStudentID, auth.RoleStudent),
			wantStatus: http.StatusForbidden,
			wantError:  middleware.MessageAccessDenied,
		},
		{
			name:       "with a teacher token",
			token:      issue(t, deps, accessTeacherID, auth.RoleTeacher),
			wantStatus: http.StatusForbidden,
			wantError:  middleware.MessageAccessDenied,
		},
		{
			name:       "with an admin token",
			token:      issue(t, deps, accessAdminID, auth.RoleAdmin),
			wantStatus: http.StatusForbidden,
			wantError:  middleware.MessageAccessDenied,
		},
	}
	crToken := issue(t, deps, accessCRID, auth.RoleCR)
	for _, route := range crRoutes() {
		for _, caller := range callers {
			t.Run(route.method+" "+route.path+" "+caller.name, func(t *testing.T) {
				rec := serve(t, router, route.method, route.path, route.body, caller.token)
				if rec.Code != caller.wantStatus {
					t.Fatalf("status = %d, want %d (body = %q)", rec.Code, caller.wantStatus, rec.Body.String())
				}
				if got := errorField(t, rec); got != caller.wantError {
					t.Errorf("error = %q, want %q", got, caller.wantError)
				}
			})
		}
		t.Run(route.method+" "+route.path+" with a class representative token", func(t *testing.T) {
			rec := serve(t, router, route.method, route.path, route.body, crToken)
			if rec.Code != route.wantForCR {
				t.Fatalf("status = %d, want %d (body = %q)", rec.Code, route.wantForCR, rec.Body.String())
			}
		})
	}
}

func serve(t *testing.T, router http.Handler, method, path, body, token string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	if body != "" {
		req = httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

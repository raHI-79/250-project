package server

import (
	"context"
	"github.com/raHI-79/250-project/backend/internal/auth"
	"github.com/raHI-79/250-project/backend/internal/middleware"
	"github.com/raHI-79/250-project/backend/internal/store"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type emptyCatalogStore struct{}

func (emptyCatalogStore) CreateCourse(ctx context.Context, code, title string) (store.Course, error) {
	return store.Course{}, store.ErrDuplicateCourse
}

func (emptyCatalogStore) ListCourses(ctx context.Context) ([]store.Course, error) {
	return nil, nil
}

func (emptyCatalogStore) SearchCourses(ctx context.Context, search string) ([]store.Course, error) {
	return nil, nil
}

func (emptyCatalogStore) CreateRoutine(ctx context.Context, routine store.NewRoutine) (store.RoutineRow, error) {
	return store.RoutineRow{}, store.ErrCourseNotFound
}

func (emptyCatalogStore) ListRoutines(ctx context.Context, batch, section string) ([]store.RoutineRow, error) {
	return nil, nil
}

func (emptyCatalogStore) DeleteRoutine(ctx context.Context, routineID int64) error {
	return store.ErrNotFound
}

type emptyLookupStore struct{}

func (emptyLookupStore) ListBatches(ctx context.Context) ([]string, error) {
	return []string{"CSE-23"}, nil
}

func (emptyLookupStore) ListSections(ctx context.Context, batch string) ([]string, error) {
	return []string{"A"}, nil
}

func (emptyLookupStore) ListCourses(ctx context.Context) ([]store.Course, error) {
	return nil, nil
}

func (emptyLookupStore) ListTeacherOptions(ctx context.Context) ([]store.TeacherOption, error) {
	return nil, nil
}

type emptyNotificationService struct{}

func (emptyNotificationService) Recent(ctx context.Context, userID int64, limit int) ([]store.Notification, error) {
	return nil, nil
}

func (emptyNotificationService) UnreadCount(ctx context.Context, userID int64, since time.Time) (int, error) {
	return 0, nil
}

func lookupRoutes() []string {
	return []string{
		"/api/lookups/batches",
		"/api/lookups/sections?batch=CSE-23",
		"/api/lookups/courses",
		"/api/lookups/teachers",
	}
}

func TestLookupRoutesNeedAToken(t *testing.T) {
	deps, router := newAccessRouter(t)
	tokens := map[string]string{
		"student": issue(t, deps, accessStudentID, auth.RoleStudent),
		"cr":      issue(t, deps, accessCRID, auth.RoleCR),
		"teacher": issue(t, deps, accessTeacherID, auth.RoleTeacher),
		"admin":   issue(t, deps, accessAdminID, auth.RoleAdmin),
	}
	for _, path := range lookupRoutes() {
		t.Run("without a token "+path, func(t *testing.T) {
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, want 401 (body = %q)", rec.Code, rec.Body.String())
			}
			if got := errorField(t, rec); got != middleware.MessageUnauthorized {
				t.Errorf("error = %q, want %q", got, middleware.MessageUnauthorized)
			}
		})
		for role, token := range tokens {
			t.Run("as "+role+" "+path, func(t *testing.T) {
				req := httptest.NewRequest(http.MethodGet, path, nil)
				req.Header.Set("Authorization", "Bearer "+token)
				rec := httptest.NewRecorder()
				router.ServeHTTP(rec, req)
				if rec.Code != http.StatusOK {
					t.Fatalf("status = %d, want 200 (body = %q)", rec.Code, rec.Body.String())
				}
			})
		}
	}
}

func TestNotificationRoutesRejectAdmins(t *testing.T) {
	deps, router := newAccessRouter(t)
	paths := []string{"/api/notifications", "/api/notifications/unread-count"}
	callers := []struct {
		name       string
		token      string
		wantStatus int
		wantError  string
	}{
		{name: "without a token", wantStatus: http.StatusUnauthorized, wantError: middleware.MessageUnauthorized},
		{name: "as a student", token: issue(t, deps, accessStudentID, auth.RoleStudent), wantStatus: http.StatusOK},
		{name: "as a class representative", token: issue(t, deps, accessCRID, auth.RoleCR), wantStatus: http.StatusOK},
		{name: "as a teacher", token: issue(t, deps, accessTeacherID, auth.RoleTeacher), wantStatus: http.StatusOK},
		{
			name:       "as an admin",
			token:      issue(t, deps, accessAdminID, auth.RoleAdmin),
			wantStatus: http.StatusForbidden,
			wantError:  middleware.MessageAccessDenied,
		},
	}
	for _, path := range paths {
		for _, caller := range callers {
			t.Run(path+" "+caller.name, func(t *testing.T) {
				req := httptest.NewRequest(http.MethodGet, path, nil)
				if caller.token != "" {
					req.Header.Set("Authorization", "Bearer "+caller.token)
				}
				rec := httptest.NewRecorder()
				router.ServeHTTP(rec, req)
				if rec.Code != caller.wantStatus {
					t.Fatalf("status = %d, want %d (body = %q)", rec.Code, caller.wantStatus, rec.Body.String())
				}
				if caller.wantError != "" && errorField(t, rec) != caller.wantError {
					t.Errorf("error = %q, want %q", errorField(t, rec), caller.wantError)
				}
			})
		}
	}
}

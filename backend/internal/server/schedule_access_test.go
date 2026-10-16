package server

import (
	"context"
	"github.com/raHI-79/250-project/backend/internal/auth"
	"github.com/raHI-79/250-project/backend/internal/middleware"
	"github.com/raHI-79/250-project/backend/internal/store"
	"net/http"
	"testing"
	"time"
)

type emptyTeacherStore struct{}

func (emptyTeacherStore) GetTeacherByUser(ctx context.Context, userID int64) (store.Teacher, error) {
	return store.Teacher{ID: 7, UserID: userID, Name: "Active Teacher"}, nil
}

func (emptyTeacherStore) ListTeacherRequests(ctx context.Context, teacherID int64) ([]store.RequestRow, error) {
	return nil, nil
}

func (emptyTeacherStore) DecideRequest(ctx context.Context, eventID, teacherID int64, status string, now time.Time) (store.EventDecision, error) {
	return store.EventDecision{}, store.ErrNotFound
}

func (emptyTeacherStore) CancelClassEvent(ctx context.Context, eventID, teacherID int64, now time.Time) (store.EventDecision, error) {
	return store.EventDecision{}, store.ErrNotFound
}

func (emptyTeacherStore) ListTeacherSchedule(ctx context.Context, teacherID int64, date string) ([]store.ScheduleItem, error) {
	return nil, nil
}

func (emptyTeacherStore) ListTeacherClasses(ctx context.Context, teacherID int64, date string, now time.Time) ([]store.ClassEventRow, error) {
	return nil, nil
}

func (emptyTeacherStore) GetTeacherRoutine(ctx context.Context, routineID, teacherID int64) (store.RoutineRow, error) {
	return store.RoutineRow{}, store.ErrNotFound
}

func (emptyTeacherStore) CancelRoutineOccurrence(ctx context.Context, routineID, cancelledBy int64, date string) error {
	return store.ErrAlreadyCancelled
}

func (emptyTeacherStore) ListActiveCRs(ctx context.Context, batch, section string) ([]int64, error) {
	return nil, nil
}

type emptyStudentStore struct{}

func (emptyStudentStore) GetStudentProfile(ctx context.Context, userID int64) (store.StudentProfile, error) {
	return store.StudentProfile{UserID: userID, Batch: "CSE-23", Section: "A"}, nil
}

func (emptyStudentStore) ListDayPlan(ctx context.Context, batch, date string) ([]store.PlanItem, error) {
	return nil, nil
}

func (emptyStudentStore) ListDropCourseSchedule(ctx context.Context, courseCode, ownBatch string, now time.Time) ([]store.DropCourseItem, error) {
	return nil, nil
}

func (emptyStudentStore) ListExams(ctx context.Context, batch, section string, now time.Time) ([]store.ExamRow, error) {
	return nil, nil
}

func (emptyStudentStore) ListFilteredRoutines(ctx context.Context, filter store.RoutineFilter) ([]store.RoutineRow, error) {
	return nil, nil
}

type silentAlerts struct{}

func (silentAlerts) NotifyUser(ctx context.Context, userID int64, kind, title, body string) error {
	return nil
}

func (silentAlerts) NotifyBroadcast(ctx context.Context, role, batch, section, kind, title, body string) error {
	return nil
}

type scheduleRoute struct {
	method     string
	path       string
	body       string
	wantForOwn int
}

func teacherRoutes() []scheduleRoute {
	return []scheduleRoute{
		{method: http.MethodGet, path: "/api/teacher/requests", wantForOwn: http.StatusOK},
		{method: http.MethodPost, path: "/api/teacher/requests/1/approve", wantForOwn: http.StatusNotFound},
		{method: http.MethodPost, path: "/api/teacher/requests/1/reject", wantForOwn: http.StatusNotFound},
		{method: http.MethodGet, path: "/api/teacher/schedule?date=2026-10-20", wantForOwn: http.StatusOK},
		{method: http.MethodGet, path: "/api/teacher/classes/upcoming", wantForOwn: http.StatusOK},
		{method: http.MethodPost, path: "/api/teacher/classes/1/cancel", wantForOwn: http.StatusNotFound},
		{method: http.MethodPost, path: "/api/teacher/routines/1/cancel", body: `{"date":"2026-10-20"}`, wantForOwn: http.StatusNotFound},
	}
}

func studentRoutes() []scheduleRoute {
	return []scheduleRoute{
		{method: http.MethodGet, path: "/api/students/upcoming-routine", wantForOwn: http.StatusOK},
		{method: http.MethodGet, path: "/api/students/drop-course-routine", wantForOwn: http.StatusOK},
		{method: http.MethodGet, path: "/api/students/exams", wantForOwn: http.StatusOK},
		{method: http.MethodGet, path: "/api/students/routine", wantForOwn: http.StatusOK},
	}
}

func TestTeacherRoutesNeedATeacherToken(t *testing.T) {
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
			name:       "with a class representative token",
			token:      issue(t, deps, accessCRID, auth.RoleCR),
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
	teacherToken := issue(t, deps, accessTeacherID, auth.RoleTeacher)
	for _, route := range teacherRoutes() {
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
		t.Run(route.method+" "+route.path+" with a teacher token", func(t *testing.T) {
			rec := serve(t, router, route.method, route.path, route.body, teacherToken)
			if rec.Code != route.wantForOwn {
				t.Fatalf("status = %d, want %d (body = %q)", rec.Code, route.wantForOwn, rec.Body.String())
			}
		})
	}
}

func TestStudentRoutesAllowStudentsAndCRs(t *testing.T) {
	deps, router := newAccessRouter(t)
	allowed := map[string]string{
		"student":              issue(t, deps, accessStudentID, auth.RoleStudent),
		"class representative": issue(t, deps, accessCRID, auth.RoleCR),
	}
	refused := map[string]string{
		"teacher": issue(t, deps, accessTeacherID, auth.RoleTeacher),
		"admin":   issue(t, deps, accessAdminID, auth.RoleAdmin),
	}
	for _, route := range studentRoutes() {
		t.Run(route.path+" without a token", func(t *testing.T) {
			rec := serve(t, router, route.method, route.path, route.body, "")
			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, want 401 (body = %q)", rec.Code, rec.Body.String())
			}
			if got := errorField(t, rec); got != middleware.MessageUnauthorized {
				t.Errorf("error = %q, want %q", got, middleware.MessageUnauthorized)
			}
		})
		for role, token := range allowed {
			t.Run(route.path+" as a "+role, func(t *testing.T) {
				rec := serve(t, router, route.method, route.path, route.body, token)
				if rec.Code != route.wantForOwn {
					t.Fatalf("status = %d, want %d (body = %q)", rec.Code, route.wantForOwn, rec.Body.String())
				}
			})
		}
		for role, token := range refused {
			t.Run(route.path+" as a "+role, func(t *testing.T) {
				rec := serve(t, router, route.method, route.path, route.body, token)
				if rec.Code != http.StatusForbidden {
					t.Fatalf("status = %d, want 403 (body = %q)", rec.Code, rec.Body.String())
				}
				if got := errorField(t, rec); got != middleware.MessageAccessDenied {
					t.Errorf("error = %q, want %q", got, middleware.MessageAccessDenied)
				}
			})
		}
	}
}

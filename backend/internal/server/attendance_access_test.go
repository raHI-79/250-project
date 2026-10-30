package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/raHI-79/250-project/backend/internal/attendance"
	"github.com/raHI-79/250-project/backend/internal/auth"
	"github.com/raHI-79/250-project/backend/internal/handler"
	"github.com/raHI-79/250-project/backend/internal/middleware"
	"github.com/raHI-79/250-project/backend/internal/store"
)

type stubPresenceStore struct {
	marks int
}

func (stubPresenceStore) GetStudentProfile(ctx context.Context, userID int64) (store.StudentProfile, error) {
	if userID != accessStudentID && userID != accessCRID {
		return store.StudentProfile{}, store.ErrNotFound
	}
	return store.StudentProfile{UserID: userID, StudentID: 7, Batch: "CSE-23", Section: "A"}, nil
}

func (stubPresenceStore) FindAccount(ctx context.Context, userID int64) (store.AccountRef, error) {
	return store.AccountRef{ID: userID, Name: "Access Student", Identifier: "2023331001"}, nil
}

func (stubPresenceStore) RunningSemester(ctx context.Context, batch string) (string, error) {
	return "3-1", nil
}

func (stubPresenceStore) OpenSessionForStudent(ctx context.Context,
	studentID int64) (store.AttendanceSession, error) {
	return store.AttendanceSession{}, store.ErrNotFound
}

func (stubPresenceStore) AttendanceSession(ctx context.Context,
	sessionID int64) (store.AttendanceSession, error) {
	return store.AttendanceSession{}, store.ErrNotFound
}

func (stubPresenceStore) OnRoster(ctx context.Context, sessionID, studentID int64) (bool, error) {
	return true, nil
}

func (s *stubPresenceStore) MarkAttendance(ctx context.Context, mark store.MarkInput) error {
	s.marks++
	return nil
}

func (stubPresenceStore) StudentCourseAttendance(ctx context.Context,
	studentID int64) ([]store.CourseAttendance, error) {
	return nil, nil
}

func (stubPresenceStore) StudentAttendanceSessions(ctx context.Context,
	studentID int64) ([]store.StudentSessionRow, error) {
	return nil, nil
}

func (stubPresenceStore) DeviceBinding(ctx context.Context, userID int64) (store.DeviceBinding, error) {
	return store.DeviceBinding{}, store.ErrNotFound
}

func (stubPresenceStore) BindDevice(ctx context.Context, userID int64,
	deviceID string) (store.DeviceBinding, bool, error) {
	return store.DeviceBinding{UserID: userID, DeviceID: deviceID}, true, nil
}

func (stubPresenceStore) TouchDevice(ctx context.Context, userID int64, deviceID string) error {
	return nil
}

func (stubPresenceStore) CreateDeviceChangeRequest(ctx context.Context, userID int64,
	deviceID, reason, semester string) (store.DeviceChangeRequest, error) {
	return store.DeviceChangeRequest{ID: 1, UserID: userID, NewDeviceID: deviceID, Reason: reason}, nil
}

func (stubPresenceStore) LatestDeviceRequest(ctx context.Context,
	userID int64) (store.DeviceChangeRequest, error) {
	return store.DeviceChangeRequest{}, store.ErrNotFound
}

func (stubPresenceStore) ApprovedDeviceChanges(ctx context.Context, userID int64,
	semester string) (int, error) {
	return 0, nil
}

func (stubPresenceStore) ListBatchTeachers(ctx context.Context, batch, section string) ([]int64, error) {
	return nil, nil
}

func newAttendanceRouter(t *testing.T) (Dependencies, http.Handler, *stubPresenceStore) {
	t.Helper()
	deps, _ := newAccessRouter(t)
	presence := &stubPresenceStore{}
	deps.Presence = presence
	return deps, NewRouter(deps), presence
}

func serveJSON(t *testing.T, router http.Handler, method, path, body, token string,
	fromApp bool) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	if fromApp {
		request.Header.Set(middleware.ClientHeader, middleware.ClientAndroid)
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, request)
	return rec
}

func TestStudentsCannotMarkAttendanceFromTheWebsite(t *testing.T) {
	deps, router, presence := newAttendanceRouter(t)
	tokens := accessTokens(t, deps)
	body := `{"session_id":1,"token":"0011223344556677","lat":24.9,"lng":91.8,` +
		`"accuracy_m":10,"is_mock":false,"device_id":"phone-one"}`
	routes := []struct {
		name   string
		method string
		path   string
		body   string
	}{
		{name: "the mark itself", method: http.MethodPost,
			path: "/api/students/attendance/mark", body: body},
		{name: "registering a phone", method: http.MethodPost,
			path: "/api/students/attendance/phone", body: `{"device_id":"phone-one"}`},
		{name: "asking for a new phone", method: http.MethodPost,
			path: "/api/students/attendance/phone-request",
			body: `{"device_id":"phone-two","reason":"my phone broke"}`},
	}
	for _, role := range []string{auth.RoleStudent, auth.RoleCR} {
		for _, route := range routes {
			t.Run(role+" "+route.name, func(t *testing.T) {
				rec := serveJSON(t, router, route.method, route.path, route.body, tokens[role], false)
				if rec.Code != http.StatusForbidden {
					t.Fatalf("status = %d, want 403 (body = %q)", rec.Code, rec.Body.String())
				}
				if got := errorField(t, rec); got != attendance.MessageWebsiteOnly {
					t.Errorf("error = %q, want %q", got, attendance.MessageWebsiteOnly)
				}
			})
		}
	}
	if presence.marks != 0 {
		t.Errorf("the website saved %d marks, want none", presence.marks)
	}
}

func TestTheWebsiteExplainsWhereAttendanceIsMarked(t *testing.T) {
	deps, router, _ := newAttendanceRouter(t)
	tokens := accessTokens(t, deps)
	rec := serveJSON(t, router, http.MethodGet, "/api/students/attendance/open", "",
		tokens[auth.RoleStudent], false)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body = %q)", rec.Code, rec.Body.String())
	}
	if got := rec.Body.String(); !strings.Contains(got, attendance.MessageWebsiteOnly) {
		t.Errorf("body = %q, want it to explain that the app marks attendance", got)
	}
}

func TestMarkingFromTheAppNeedsTheRegisteredPhone(t *testing.T) {
	deps, router, presence := newAttendanceRouter(t)
	tokens := accessTokens(t, deps)
	body := `{"session_id":1,"token":"0011223344556677","lat":24.9,"lng":91.8,` +
		`"accuracy_m":10,"is_mock":false,"device_id":"phone-one"}`
	rec := serveJSON(t, router, http.MethodPost, "/api/students/attendance/mark", body,
		tokens[auth.RoleStudent], true)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403 (body = %q)", rec.Code, rec.Body.String())
	}
	if got := errorField(t, rec); got != attendance.MessageWrongPhone {
		t.Errorf("error = %q, want %q", got, attendance.MessageWrongPhone)
	}
	if presence.marks != 0 {
		t.Errorf("an unregistered phone saved %d marks, want none", presence.marks)
	}
}

func TestMarkingWithoutAPhoneIdIsRefused(t *testing.T) {
	deps, router, _ := newAttendanceRouter(t)
	tokens := accessTokens(t, deps)
	body := `{"session_id":1,"token":"0011223344556677","lat":24.9,"lng":91.8,` +
		`"accuracy_m":10,"is_mock":false,"device_id":""}`
	rec := serveJSON(t, router, http.MethodPost, "/api/students/attendance/mark", body,
		tokens[auth.RoleStudent], true)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 (body = %q)", rec.Code, rec.Body.String())
	}
	if got := errorField(t, rec); got != handler.MessageDeviceIDRequired {
		t.Errorf("error = %q, want %q", got, handler.MessageDeviceIDRequired)
	}
}

func TestTeachersCannotReachTheStudentMarkScreen(t *testing.T) {
	deps, router, _ := newAttendanceRouter(t)
	tokens := accessTokens(t, deps)
	paths := []string{
		"/api/students/attendance/open",
		"/api/students/attendance/mine",
	}
	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			rec := serveJSON(t, router, http.MethodGet, path, "", tokens[auth.RoleTeacher], true)
			if rec.Code != http.StatusForbidden {
				t.Fatalf("status = %d, want 403 (body = %q)", rec.Code, rec.Body.String())
			}
			if got := errorField(t, rec); got != middleware.MessageAccessDenied {
				t.Errorf("error = %q, want %q", got, middleware.MessageAccessDenied)
			}
		})
	}
}

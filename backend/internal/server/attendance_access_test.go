package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/raHI-79/250-project/backend/internal/attendance"
	"github.com/raHI-79/250-project/backend/internal/auth"
	"github.com/raHI-79/250-project/backend/internal/handler"
	"github.com/raHI-79/250-project/backend/internal/middleware"
	"github.com/raHI-79/250-project/backend/internal/store"
)

type stubPresenceStore struct {
	marks    int
	session  store.AttendanceSession
	hasOpen  bool
	lastMark store.MarkInput
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

func (s *stubPresenceStore) OpenSessionForStudent(ctx context.Context,
	studentID int64) (store.AttendanceSession, error) {
	if !s.hasOpen {
		return store.AttendanceSession{}, store.ErrNotFound
	}
	return s.session, nil
}

func (s *stubPresenceStore) AttendanceSession(ctx context.Context,
	sessionID int64) (store.AttendanceSession, error) {
	if !s.hasOpen || s.session.ID != sessionID {
		return store.AttendanceSession{}, store.ErrNotFound
	}
	return s.session, nil
}

func (stubPresenceStore) OnRoster(ctx context.Context, sessionID, studentID int64) (bool, error) {
	return true, nil
}

func (s *stubPresenceStore) MarkAttendance(ctx context.Context, mark store.MarkInput) error {
	s.marks++
	s.lastMark = mark
	return nil
}

func openWebSession(presence *stubPresenceStore, method string) {
	presence.hasOpen = true
	presence.session = store.AttendanceSession{
		ID:          9,
		CourseID:    4,
		CourseCode:  "CSE245",
		CourseTitle: "Databases",
		Batch:       "CSE-23",
		Section:     "A",
		TeacherID:   3,
		StartedAt:   time.Now().Add(-time.Minute),
		EndsAt:      time.Now().Add(9 * time.Minute),
		Status:      attendance.StatusOpen,
		Method:      method,
		TokenKey:    "a-session-key",
		TeacherLat:  24.9,
		TeacherLng:  91.8,
		LocationAt:  time.Now(),
		HasLocation: true,
	}
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

func TestThePhoneRoutesStayInTheApp(t *testing.T) {
	deps, router, _ := newAttendanceRouter(t)
	tokens := accessTokens(t, deps)
	routes := []struct {
		name   string
		method string
		path   string
		body   string
	}{
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
}

func TestTheWebsiteNeedsTheCodeToMark(t *testing.T) {
	deps, router, presence := newAttendanceRouter(t)
	tokens := accessTokens(t, deps)
	body := `{"session_id":1,"token":"0011223344556677","lat":24.9,"lng":91.8,` +
		`"accuracy_m":10,"is_mock":false}`
	for _, role := range []string{auth.RoleStudent, auth.RoleCR} {
		t.Run(role, func(t *testing.T) {
			rec := serveJSON(t, router, http.MethodPost, "/api/students/attendance/mark", body,
				tokens[role], false)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400 (body = %q)", rec.Code, rec.Body.String())
			}
			if got := errorField(t, rec); got != attendance.MessageWebNeedsCode {
				t.Errorf("error = %q, want %q", got, attendance.MessageWebNeedsCode)
			}
		})
	}
	if presence.marks != 0 {
		t.Errorf("a mark without a code saved %d rows, want none", presence.marks)
	}
}

func TestTheWebsiteAsksForTheCodeOnTheOpenSession(t *testing.T) {
	deps, router, _ := newAttendanceRouter(t)
	tokens := accessTokens(t, deps)
	rec := serveJSON(t, router, http.MethodGet, "/api/students/attendance/open", "",
		tokens[auth.RoleStudent], false)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body = %q)", rec.Code, rec.Body.String())
	}
	if got := rec.Body.String(); !strings.Contains(got, `"needs_code":true`) {
		t.Errorf("body = %q, want the website to be told to ask for the code", got)
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

func TestTheWebsiteMarksWithTheCodeAndThePosition(t *testing.T) {
	deps, router, presence := newAttendanceRouter(t)
	openWebSession(presence, attendance.TakeBluetooth)
	tokens := accessTokens(t, deps)
	code := attendance.CurrentCode(presence.session.TokenKey, presence.session.ID, time.Now())
	tests := []struct {
		name       string
		body       string
		wantStatus int
		wantError  string
	}{
		{
			name:       "too far from the teacher",
			body:       `{"session_id":9,"code":"` + code + `","lat":25.9,"lng":91.8,"accuracy_m":10}`,
			wantStatus: http.StatusForbidden,
			wantError:  attendance.MessageTooFar,
		},
		{
			name:       "a weak position",
			body:       `{"session_id":9,"code":"` + code + `","lat":24.9,"lng":91.8,"accuracy_m":400}`,
			wantStatus: http.StatusForbidden,
			wantError:  attendance.MessageWeakSignal,
		},
		{
			name:       "a code from another slice",
			body:       `{"session_id":9,"code":"9999","lat":24.9,"lng":91.8,"accuracy_m":10}`,
			wantStatus: http.StatusForbidden,
			wantError:  attendance.MessageWrongToken,
		},
		{
			name:       "the code and a position inside the class",
			body:       `{"session_id":9,"code":"` + code + `","lat":24.9,"lng":91.8,"accuracy_m":10}`,
			wantStatus: http.StatusCreated,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.name == "a code from another slice" &&
				attendance.CodeMatches(presence.session.TokenKey, presence.session.ID, time.Now(), "9999") {
				t.Skip("9999 happens to be the current code")
			}
			rec := serveJSON(t, router, http.MethodPost, "/api/students/attendance/mark", test.body,
				tokens[auth.RoleStudent], false)
			if rec.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d (body = %q)", rec.Code, test.wantStatus, rec.Body.String())
			}
			if test.wantStatus != http.StatusCreated {
				if got := errorField(t, rec); got != test.wantError {
					t.Errorf("error = %q, want %q", got, test.wantError)
				}
				return
			}
			if presence.marks != 1 {
				t.Fatalf("marks = %d, want one", presence.marks)
			}
			if presence.lastMark.Method != attendance.MethodWeb {
				t.Errorf("method = %q, want %q", presence.lastMark.Method, attendance.MethodWeb)
			}
			if presence.lastMark.DeviceID != "" {
				t.Errorf("device = %q, want no phone binding on the website", presence.lastMark.DeviceID)
			}
		})
	}
}

func TestARollCallIsNotOfferedToStudents(t *testing.T) {
	deps, router, presence := newAttendanceRouter(t)
	openWebSession(presence, attendance.TakeRollCall)
	tokens := accessTokens(t, deps)
	code := attendance.CurrentCode(presence.session.TokenKey, presence.session.ID, time.Now())
	body := `{"session_id":9,"code":"` + code + `","lat":24.9,"lng":91.8,"accuracy_m":10}`
	rec := serveJSON(t, router, http.MethodPost, "/api/students/attendance/mark", body,
		tokens[auth.RoleStudent], false)
	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409 (body = %q)", rec.Code, rec.Body.String())
	}
	if got := errorField(t, rec); got != attendance.MessageRollCallOnly {
		t.Errorf("error = %q, want %q", got, attendance.MessageRollCallOnly)
	}
	if presence.marks != 0 {
		t.Errorf("marks = %d, want none in a roll call", presence.marks)
	}
}

func TestACodeOnlySessionMarksFromTheWebsiteWithoutAPosition(t *testing.T) {
	deps, router, presence := newAttendanceRouter(t)
	openWebSession(presence, attendance.TakeBluetooth)
	presence.session.CodeOnly = true
	presence.session.HasLocation = false
	tokens := accessTokens(t, deps)
	listed := serveJSON(t, router, http.MethodGet, "/api/students/attendance/open", "",
		tokens[auth.RoleStudent], false)
	if !strings.Contains(listed.Body.String(), `"code_only":true`) {
		t.Fatalf("open session = %q, want it to say the session is code only", listed.Body.String())
	}
	withoutCode := serveJSON(t, router, http.MethodPost, "/api/students/attendance/mark",
		`{"session_id":9}`, tokens[auth.RoleStudent], false)
	if withoutCode.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 without a code (body = %q)", withoutCode.Code, withoutCode.Body.String())
	}
	code := attendance.CurrentCode(presence.session.TokenKey, presence.session.ID, time.Now())
	rec := serveJSON(t, router, http.MethodPost, "/api/students/attendance/mark",
		`{"session_id":9,"code":"`+code+`"}`, tokens[auth.RoleStudent], false)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201 (body = %q)", rec.Code, rec.Body.String())
	}
	if presence.lastMark.Method != attendance.MethodCode {
		t.Errorf("method = %q, want %q", presence.lastMark.Method, attendance.MethodCode)
	}
	if !presence.lastMark.NoLocation {
		t.Error("a code only mark was not flagged as having no location")
	}
	if presence.lastMark.Lat != 0 || presence.lastMark.DistanceM != 0 {
		t.Errorf("a code only mark kept a position: %+v", presence.lastMark)
	}
}

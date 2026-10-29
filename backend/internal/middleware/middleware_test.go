package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/raHI-79/250-project/backend/internal/auth"
	"github.com/raHI-79/250-project/backend/internal/httpx"
	"github.com/raHI-79/250-project/backend/internal/store"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

const (
	testSecret        = "0123456789abcdef0123456789abcdef"
	testTTL           = time.Hour
	testFrontend      = "http://localhost:5173"
	testOtherOrigin   = "http://evil.example.com"
	testStudentID     = int64(11)
	testCRID          = int64(22)
	testTeacherID     = int64(33)
	testAdminID       = int64(44)
	testUnknownUserID = int64(99)
)

type fakeUserStore struct {
	users map[int64]store.AuthUser
	err   error
	calls int
}

func (f *fakeUserStore) GetAuthUser(ctx context.Context, userID int64) (store.AuthUser, error) {
	f.calls++
	if f.err != nil {
		return store.AuthUser{}, f.err
	}
	user, ok := f.users[userID]
	if !ok {
		return store.AuthUser{}, store.ErrNotFound
	}
	return user, nil
}

func newFakeUserStore() *fakeUserStore {
	return &fakeUserStore{users: map[int64]store.AuthUser{
		testStudentID: {ID: testStudentID, Identifier: "2023331001", Role: "student", Status: store.StatusActive},
		testCRID:      {ID: testCRID, Identifier: "2023331002", Role: "student", Status: store.StatusActive, IsCR: true, TokenVersion: 2},
		testTeacherID: {ID: testTeacherID, Identifier: "T-001", Role: "teacher", Status: store.StatusActive},
		testAdminID:   {ID: testAdminID, Identifier: "admin", Role: "admin", Status: store.StatusActive},
	}}
}

func newTestTokens(t *testing.T) *auth.Tokens {
	t.Helper()
	tokens, err := auth.NewTokens(testSecret, testTTL)
	if err != nil {
		t.Fatalf("auth.NewTokens() returned an unexpected error: %v", err)
	}
	return tokens
}

func issue(t *testing.T, tokens *auth.Tokens, userID int64, role string, version int) string {
	t.Helper()
	token, err := tokens.IssueToken(userID, role, version)
	if err != nil {
		t.Fatalf("IssueToken() returned an unexpected error: %v", err)
	}
	return token
}

func captureLog(t *testing.T) *bytes.Buffer {
	t.Helper()
	buffer := &bytes.Buffer{}
	flags := log.Flags()
	log.SetOutput(buffer)
	log.SetFlags(0)
	t.Cleanup(func() {
		log.SetOutput(os.Stderr)
		log.SetFlags(flags)
	})
	return buffer
}

func decodeError(t *testing.T, body *bytes.Buffer) string {
	t.Helper()
	var payload struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(body.Bytes(), &payload); err != nil {
		t.Fatalf("body is not valid JSON: %v (body = %q)", err, body.String())
	}
	return payload.Error
}

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, ok := CurrentUser(r.Context())
		if !ok {
			httpx.WriteJSON(w, http.StatusOK, map[string]any{"user": nil})
			return
		}
		httpx.WriteJSON(w, http.StatusOK, map[string]any{"id": user.ID, "identifier": user.Identifier, "role": user.Role})
	})
}

func TestVerifyToken(t *testing.T) {
	tokens := newTestTokens(t)
	expired, err := auth.NewTokens(testSecret, time.Nanosecond)
	if err != nil {
		t.Fatalf("auth.NewTokens() returned an unexpected error: %v", err)
	}
	expiredToken := issue(t, expired, testStudentID, auth.RoleStudent, 0)
	time.Sleep(2 * time.Nanosecond)
	otherTokens, err := auth.NewTokens(strings.Repeat("z", 32), testTTL)
	if err != nil {
		t.Fatalf("auth.NewTokens() returned an unexpected error: %v", err)
	}
	inactive := map[int64]store.AuthUser{
		testStudentID: {ID: testStudentID, Identifier: "2023331001", Role: "student", Status: store.StatusPending},
	}
	removed := map[int64]store.AuthUser{
		testStudentID: {ID: testStudentID, Identifier: "2023331001", Role: "student", Status: store.StatusRemoved},
	}
	demoted := map[int64]store.AuthUser{
		testCRID: {ID: testCRID, Identifier: "2023331002", Role: "student", Status: store.StatusActive, IsCR: false, TokenVersion: 2},
	}
	promoted := map[int64]store.AuthUser{
		testStudentID: {ID: testStudentID, Identifier: "2023331001", Role: "student", Status: store.StatusActive, IsCR: true},
	}
	tests := []struct {
		name        string
		header      string
		users       map[int64]store.AuthUser
		storeErr    error
		wantStatus  int
		wantMessage string
		wantRole    string
	}{
		{
			name:        "no authorization header",
			wantStatus:  http.StatusUnauthorized,
			wantMessage: MessageUnauthorized,
		},
		{
			name:        "empty bearer token",
			header:      "Bearer ",
			wantStatus:  http.StatusUnauthorized,
			wantMessage: MessageUnauthorized,
		},
		{
			name:        "wrong scheme",
			header:      "Basic " + issue(t, tokens, testStudentID, auth.RoleStudent, 0),
			wantStatus:  http.StatusUnauthorized,
			wantMessage: MessageUnauthorized,
		},
		{
			name:        "token without a scheme",
			header:      issue(t, tokens, testStudentID, auth.RoleStudent, 0),
			wantStatus:  http.StatusUnauthorized,
			wantMessage: MessageUnauthorized,
		},
		{
			name:        "malformed token",
			header:      "Bearer not-a-token",
			wantStatus:  http.StatusUnauthorized,
			wantMessage: MessageUnauthorized,
		},
		{
			name:        "expired token",
			header:      "Bearer " + expiredToken,
			wantStatus:  http.StatusUnauthorized,
			wantMessage: MessageUnauthorized,
		},
		{
			name:        "token signed with another secret",
			header:      "Bearer " + issue(t, otherTokens, testStudentID, auth.RoleStudent, 0),
			wantStatus:  http.StatusUnauthorized,
			wantMessage: MessageUnauthorized,
		},
		{
			name:        "user no longer exists",
			header:      "Bearer " + issue(t, tokens, testUnknownUserID, auth.RoleStudent, 0),
			wantStatus:  http.StatusUnauthorized,
			wantMessage: MessageSessionExpired,
		},
		{
			name:        "pending user",
			header:      "Bearer " + issue(t, tokens, testStudentID, auth.RoleStudent, 0),
			users:       inactive,
			wantStatus:  http.StatusUnauthorized,
			wantMessage: MessageSessionExpired,
		},
		{
			name:        "removed user",
			header:      "Bearer " + issue(t, tokens, testStudentID, auth.RoleStudent, 0),
			users:       removed,
			wantStatus:  http.StatusUnauthorized,
			wantMessage: MessageSessionExpired,
		},
		{
			name:        "token version is behind the database",
			header:      "Bearer " + issue(t, tokens, testCRID, auth.RoleCR, 1),
			wantStatus:  http.StatusUnauthorized,
			wantMessage: MessageSessionExpired,
		},
		{
			name:        "class representative demoted to student",
			header:      "Bearer " + issue(t, tokens, testCRID, auth.RoleCR, 2),
			users:       demoted,
			wantStatus:  http.StatusUnauthorized,
			wantMessage: MessageSessionExpired,
		},
		{
			name:        "student promoted to class representative",
			header:      "Bearer " + issue(t, tokens, testStudentID, auth.RoleStudent, 0),
			users:       promoted,
			wantStatus:  http.StatusUnauthorized,
			wantMessage: MessageSessionExpired,
		},
		{
			name:       "active student",
			header:     "Bearer " + issue(t, tokens, testStudentID, auth.RoleStudent, 0),
			wantStatus: http.StatusOK,
			wantRole:   auth.RoleStudent,
		},
		{
			name:       "active class representative",
			header:     "Bearer " + issue(t, tokens, testCRID, auth.RoleCR, 2),
			wantStatus: http.StatusOK,
			wantRole:   auth.RoleCR,
		},
		{
			name:       "lowercase bearer scheme is accepted",
			header:     "bearer " + issue(t, tokens, testTeacherID, auth.RoleTeacher, 0),
			wantStatus: http.StatusOK,
			wantRole:   auth.RoleTeacher,
		},
		{
			name:        "database failure",
			header:      "Bearer " + issue(t, tokens, testStudentID, auth.RoleStudent, 0),
			storeErr:    errors.New("connection refused"),
			wantStatus:  http.StatusInternalServerError,
			wantMessage: MessageServerError,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			captureLog(t)
			users := newFakeUserStore()
			if tc.users != nil {
				users.users = tc.users
			}
			users.err = tc.storeErr
			req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
			if tc.header != "" {
				req.Header.Set(authorizationHeader, tc.header)
			}
			rec := httptest.NewRecorder()
			VerifyToken(tokens, users)(okHandler()).ServeHTTP(rec, req)
			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d (body = %q)", rec.Code, tc.wantStatus, rec.Body.String())
			}
			if got := rec.Header().Get("Content-Type"); got != "application/json" {
				t.Errorf("Content-Type = %q, want application/json", got)
			}
			if tc.wantMessage != "" {
				if got := decodeError(t, rec.Body); got != tc.wantMessage {
					t.Errorf("error message = %q, want %q", got, tc.wantMessage)
				}
				return
			}
			var payload struct {
				ID         int64  `json:"id"`
				Identifier string `json:"identifier"`
				Role       string `json:"role"`
			}
			if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
				t.Fatalf("body is not valid JSON: %v", err)
			}
			if payload.Role != tc.wantRole {
				t.Errorf("role in context = %q, want %q", payload.Role, tc.wantRole)
			}
			if payload.ID == 0 || payload.Identifier == "" {
				t.Error("the handler did not receive the current user from the context")
			}
		})
	}
}

func TestVerifyTokenChecksTheDatabaseOnEveryRequest(t *testing.T) {
	captureLog(t)
	tokens := newTestTokens(t)
	users := newFakeUserStore()
	token := issue(t, tokens, testStudentID, auth.RoleStudent, 0)
	handler := VerifyToken(tokens, users)(okHandler())
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
		req.Header.Set(authorizationHeader, "Bearer "+token)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("request %d: status = %d, want 200", i+1, rec.Code)
		}
	}
	if users.calls != 3 {
		t.Errorf("the store was queried %d times, want 3", users.calls)
	}
}

func TestVerifyTokenNeverLogsTheToken(t *testing.T) {
	logged := captureLog(t)
	tokens := newTestTokens(t)
	users := newFakeUserStore()
	users.err = errors.New("connection refused")
	token := issue(t, tokens, testStudentID, auth.RoleStudent, 0)
	req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	req.Header.Set(authorizationHeader, "Bearer "+token)
	rec := httptest.NewRecorder()
	VerifyToken(tokens, users)(okHandler()).ServeHTTP(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
	if strings.Contains(logged.String(), token) {
		t.Error("the log output contains the token, want it hidden")
	}
}

func TestRequireRole(t *testing.T) {
	tests := []struct {
		name        string
		userRole    string
		hasUser     bool
		allowed     []string
		wantStatus  int
		wantMessage string
	}{
		{
			name:       "admin on an admin route",
			userRole:   auth.RoleAdmin,
			hasUser:    true,
			allowed:    []string{auth.RoleAdmin},
			wantStatus: http.StatusOK,
		},
		{
			name:       "class representative on a shared student route",
			userRole:   auth.RoleCR,
			hasUser:    true,
			allowed:    []string{auth.RoleStudent, auth.RoleCR},
			wantStatus: http.StatusOK,
		},
		{
			name:        "student on an admin route",
			userRole:    auth.RoleStudent,
			hasUser:     true,
			allowed:     []string{auth.RoleAdmin},
			wantStatus:  http.StatusForbidden,
			wantMessage: MessageAccessDenied,
		},
		{
			name:        "class representative on an admin route",
			userRole:    auth.RoleCR,
			hasUser:     true,
			allowed:     []string{auth.RoleAdmin},
			wantStatus:  http.StatusForbidden,
			wantMessage: MessageAccessDenied,
		},
		{
			name:        "teacher on a cr route",
			userRole:    auth.RoleTeacher,
			hasUser:     true,
			allowed:     []string{auth.RoleCR},
			wantStatus:  http.StatusForbidden,
			wantMessage: MessageAccessDenied,
		},
		{
			name:        "student on a cr route",
			userRole:    auth.RoleStudent,
			hasUser:     true,
			allowed:     []string{auth.RoleCR},
			wantStatus:  http.StatusForbidden,
			wantMessage: MessageAccessDenied,
		},
		{
			name:        "no authenticated user in the context",
			hasUser:     false,
			allowed:     []string{auth.RoleAdmin},
			wantStatus:  http.StatusUnauthorized,
			wantMessage: MessageUnauthorized,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/admin/ping", nil)
			if tc.hasUser {
				current := User{ID: 1, Identifier: "someone", Role: tc.userRole}
				req = req.WithContext(context.WithValue(req.Context(), currentUserKey, current))
			}
			rec := httptest.NewRecorder()
			RequireRole(tc.allowed...)(okHandler()).ServeHTTP(rec, req)
			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tc.wantStatus)
			}
			if tc.wantMessage == "" {
				return
			}
			if got := decodeError(t, rec.Body); got != tc.wantMessage {
				t.Errorf("error message = %q, want %q", got, tc.wantMessage)
			}
		})
	}
}

func TestRecoverer(t *testing.T) {
	tests := []struct {
		name       string
		handler    http.Handler
		wantStatus int
		wantError  string
	}{
		{
			name: "panic with a string",
			handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				panic("something went very wrong")
			}),
			wantStatus: http.StatusInternalServerError,
			wantError:  MessageServerError,
		},
		{
			name: "panic with an error",
			handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				panic(errors.New("boom"))
			}),
			wantStatus: http.StatusInternalServerError,
			wantError:  MessageServerError,
		},
		{
			name: "nil map write",
			handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var broken map[string]string
				broken["key"] = "value"
			}),
			wantStatus: http.StatusInternalServerError,
			wantError:  MessageServerError,
		},
		{
			name: "handler without a panic",
			handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				httpx.WriteJSON(w, http.StatusCreated, map[string]string{"status": "created"})
			}),
			wantStatus: http.StatusCreated,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			logged := captureLog(t)
			req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
			rec := httptest.NewRecorder()
			Recoverer(tc.handler).ServeHTTP(rec, req)
			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tc.wantStatus)
			}
			if tc.wantError == "" {
				if strings.Contains(logged.String(), "panic") {
					t.Error("a panic was logged for a handler that did not panic")
				}
				return
			}
			if got := rec.Header().Get("Content-Type"); got != "application/json" {
				t.Errorf("Content-Type = %q, want application/json", got)
			}
			if got := decodeError(t, rec.Body); got != tc.wantError {
				t.Errorf("error message = %q, want %q", got, tc.wantError)
			}
			if !strings.Contains(logged.String(), "panic") {
				t.Error("the panic was not logged")
			}
		})
	}
}

func TestRecovererKeepsAnAlreadyWrittenResponse(t *testing.T) {
	captureLog(t)
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "partial"})
		panic("after writing")
	})
	rec := httptest.NewRecorder()
	Recoverer(handler).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/health", nil))
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200 because the response was already written", rec.Code)
	}
}

func TestLogger(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		target     string
		handler    http.Handler
		wantStatus int
		wantLogged []string
	}{
		{
			name:   "successful request",
			method: http.MethodGet,
			target: "/api/health",
			handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
			}),
			wantStatus: http.StatusOK,
			wantLogged: []string{"GET", "/api/health", "200"},
		},
		{
			name:   "error response",
			method: http.MethodPost,
			target: "/api/admin/courses",
			handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				httpx.WriteError(w, http.StatusForbidden, MessageAccessDenied)
			}),
			wantStatus: http.StatusForbidden,
			wantLogged: []string{"POST", "/api/admin/courses", "403"},
		},
		{
			name:       "handler that writes nothing",
			method:     http.MethodGet,
			target:     "/api/health",
			handler:    http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}),
			wantStatus: http.StatusOK,
			wantLogged: []string{"GET", "200"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			logged := captureLog(t)
			rec := httptest.NewRecorder()
			Logger(tc.handler).ServeHTTP(rec, httptest.NewRequest(tc.method, tc.target, nil))
			if rec.Code != tc.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tc.wantStatus)
			}
			for _, want := range tc.wantLogged {
				if !strings.Contains(logged.String(), want) {
					t.Errorf("log line %q does not contain %q", strings.TrimSpace(logged.String()), want)
				}
			}
		})
	}
}

func TestLoggerNeverLogsTokensOrQueries(t *testing.T) {
	logged := captureLog(t)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/me?token=super-secret-value", nil)
	req.Header.Set(authorizationHeader, "Bearer another-secret-value")
	Logger(okHandler()).ServeHTTP(rec, req)
	for _, secret := range []string{"super-secret-value", "another-secret-value", "Bearer"} {
		if strings.Contains(logged.String(), secret) {
			t.Errorf("the log output contains %q, want it hidden", secret)
		}
	}
}

func TestCORS(t *testing.T) {
	tests := []struct {
		name            string
		method          string
		origin          string
		wantStatus      int
		wantAllowOrigin string
		wantHandlerRun  bool
	}{
		{
			name:            "preflight from the allowed origin",
			method:          http.MethodOptions,
			origin:          testFrontend,
			wantStatus:      http.StatusNoContent,
			wantAllowOrigin: testFrontend,
		},
		{
			name:       "preflight from another origin",
			method:     http.MethodOptions,
			origin:     testOtherOrigin,
			wantStatus: http.StatusNoContent,
		},
		{
			name:       "preflight without an origin",
			method:     http.MethodOptions,
			wantStatus: http.StatusNoContent,
		},
		{
			name:            "request from the allowed origin",
			method:          http.MethodGet,
			origin:          testFrontend,
			wantStatus:      http.StatusOK,
			wantAllowOrigin: testFrontend,
			wantHandlerRun:  true,
		},
		{
			name:           "request from another origin",
			method:         http.MethodGet,
			origin:         testOtherOrigin,
			wantStatus:     http.StatusOK,
			wantHandlerRun: true,
		},
		{
			name:           "request without an origin",
			method:         http.MethodGet,
			wantStatus:     http.StatusOK,
			wantHandlerRun: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			handlerRun := false
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				handlerRun = true
				httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
			})
			req := httptest.NewRequest(tc.method, "/api/health", nil)
			if tc.origin != "" {
				req.Header.Set("Origin", tc.origin)
			}
			rec := httptest.NewRecorder()
			CORS(testFrontend)(next).ServeHTTP(rec, req)
			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tc.wantStatus)
			}
			if handlerRun != tc.wantHandlerRun {
				t.Errorf("handler ran = %t, want %t", handlerRun, tc.wantHandlerRun)
			}
			if got := rec.Header().Get("Access-Control-Allow-Origin"); got != tc.wantAllowOrigin {
				t.Errorf("Access-Control-Allow-Origin = %q, want %q", got, tc.wantAllowOrigin)
			}
			if got := rec.Header().Get("Vary"); !strings.Contains(got, "Origin") {
				t.Errorf("Vary = %q, want it to contain Origin", got)
			}
			if tc.wantAllowOrigin == "" {
				return
			}
			if got := rec.Header().Get("Access-Control-Allow-Methods"); got != allowedMethods {
				t.Errorf("Access-Control-Allow-Methods = %q, want %q", got, allowedMethods)
			}
			if got := rec.Header().Get("Access-Control-Allow-Headers"); got != allowedHeaders {
				t.Errorf("Access-Control-Allow-Headers = %q, want %q", got, allowedHeaders)
			}
		})
	}
}

func TestCORSAllowsTheNativeOrigins(t *testing.T) {
	tests := []struct {
		name   string
		origin string
		want   string
	}{
		{name: "the website", origin: "https://portal.example", want: "https://portal.example"},
		{name: "the https app origin", origin: CapacitorHTTPS, want: CapacitorHTTPS},
		{name: "the capacitor scheme", origin: CapacitorScheme, want: CapacitorScheme},
		{name: "a stranger", origin: "https://evil.example", want: ""},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			handler := CORS("https://portal.example")(http.HandlerFunc(
				func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) }))
			request := httptest.NewRequest(http.MethodGet, "/api/me", nil)
			request.Header.Set("Origin", test.origin)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, request)
			if got := rec.Header().Get("Access-Control-Allow-Origin"); got != test.want {
				t.Errorf("allow origin = %q, want %q", got, test.want)
			}
		})
	}
}

func TestFromAndroidAppReadsTheClientHeader(t *testing.T) {
	tests := []struct {
		value string
		want  bool
	}{
		{value: "android", want: true},
		{value: " Android ", want: true},
		{value: "web", want: false},
		{value: "", want: false},
	}
	for _, test := range tests {
		request := httptest.NewRequest(http.MethodGet, "/api/me", nil)
		if test.value != "" {
			request.Header.Set(ClientHeader, test.value)
		}
		if got := FromAndroidApp(request); got != test.want {
			t.Errorf("FromAndroidApp(%q) = %v, want %v", test.value, got, test.want)
		}
	}
}

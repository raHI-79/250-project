package handler

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/raHI-79/250-project/backend/internal/auth"
	"github.com/raHI-79/250-project/backend/internal/config"
	"github.com/raHI-79/250-project/backend/internal/middleware"
	"github.com/raHI-79/250-project/backend/internal/store"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

const (
	studentPassword = "student-pass"
	teacherPassword = "teacher-pass"
	adminPassword   = "admin-pass"
	issuedToken     = "issued-test-token"
)

var hashCache sync.Map

func cachedHash(t *testing.T, password string) string {
	t.Helper()
	if value, ok := hashCache.Load(password); ok {
		return value.(string)
	}
	hash, err := auth.HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword() returned an unexpected error: %v", err)
	}
	hashCache.Store(password, hash)
	return hash
}

type resetCall struct {
	identifier   string
	tokenHash    string
	passwordHash string
}

type setResetCall struct {
	userID    int64
	tokenHash string
	expiresAt time.Time
}

type fakeAuthStore struct {
	users         map[string]store.LoginUser
	createErr     error
	findErr       error
	incrementErr  error
	setResetErr   error
	resetErr      error
	created       []store.Registration
	incremented   []int64
	setResetCalls []setResetCall
	resetCalls    []resetCall
	nextUserID    int64
}

func (f *fakeAuthStore) CreateRegistration(ctx context.Context, registration store.Registration) (int64, error) {
	if f.createErr != nil {
		return 0, f.createErr
	}
	f.created = append(f.created, registration)
	f.nextUserID++
	return f.nextUserID, nil
}

func (f *fakeAuthStore) FindLoginUser(ctx context.Context, identifier string) (store.LoginUser, error) {
	if f.findErr != nil {
		return store.LoginUser{}, f.findErr
	}
	user, ok := f.users[identifier]
	if !ok {
		return store.LoginUser{}, store.ErrNotFound
	}
	return user, nil
}

func (f *fakeAuthStore) IncrementTokenVersion(ctx context.Context, userID int64) (int, error) {
	if f.incrementErr != nil {
		return 0, f.incrementErr
	}
	f.incremented = append(f.incremented, userID)
	return 1, nil
}

func (f *fakeAuthStore) SetResetToken(ctx context.Context, userID int64, tokenHash string, expiresAt time.Time) error {
	if f.setResetErr != nil {
		return f.setResetErr
	}
	f.setResetCalls = append(f.setResetCalls, setResetCall{userID: userID, tokenHash: tokenHash, expiresAt: expiresAt})
	return nil
}

func (f *fakeAuthStore) ResetPassword(ctx context.Context, identifier, tokenHash, passwordHash string) error {
	if f.resetErr != nil {
		return f.resetErr
	}
	f.resetCalls = append(f.resetCalls, resetCall{identifier: identifier, tokenHash: tokenHash, passwordHash: passwordHash})
	return nil
}

type fakeIssuer struct {
	token  string
	err    error
	userID int64
	role   string
	calls  int
}

func (f *fakeIssuer) IssueToken(userID int64, effectiveRole string, tokenVersion int) (string, error) {
	f.calls++
	f.userID = userID
	f.role = effectiveRole
	if f.err != nil {
		return "", f.err
	}
	return f.token, nil
}

func newAuthStore(t *testing.T) *fakeAuthStore {
	t.Helper()
	studentHash := cachedHash(t, studentPassword)
	return &fakeAuthStore{users: map[string]store.LoginUser{
		"2023331001": {
			AuthUser:     store.AuthUser{ID: 1, Identifier: "2023331001", Role: "student", Status: store.StatusActive},
			PasswordHash: studentHash,
			Name:         "Active Student",
			Batch:        "CSE-23",
			Section:      "A",
		},
		"2023331002": {
			AuthUser:     store.AuthUser{ID: 2, Identifier: "2023331002", Role: "student", Status: store.StatusActive, IsCR: true, TokenVersion: 3},
			PasswordHash: studentHash,
			Name:         "Class Representative",
			Batch:        "CSE-23",
			Section:      "B",
		},
		"2023331003": {
			AuthUser:     store.AuthUser{ID: 3, Identifier: "2023331003", Role: "student", Status: store.StatusPending},
			PasswordHash: studentHash,
			Name:         "Pending Student",
			Batch:        "CSE-23",
			Section:      "A",
		},
		"2023331004": {
			AuthUser:     store.AuthUser{ID: 4, Identifier: "2023331004", Role: "student", Status: store.StatusRejected},
			PasswordHash: studentHash,
			Name:         "Rejected Student",
		},
		"2023331005": {
			AuthUser:     store.AuthUser{ID: 5, Identifier: "2023331005", Role: "student", Status: store.StatusRemoved},
			PasswordHash: studentHash,
			Name:         "Removed Student",
		},
		"T-001": {
			AuthUser:     store.AuthUser{ID: 6, Identifier: "T-001", Role: "teacher", Status: store.StatusActive},
			PasswordHash: cachedHash(t, teacherPassword),
			Name:         "Active Teacher",
		},
		"admin": {
			AuthUser:     store.AuthUser{ID: 7, Identifier: "admin", Role: "admin", Status: store.StatusActive},
			PasswordHash: cachedHash(t, adminPassword),
		},
		"removed-admin": {
			AuthUser:     store.AuthUser{ID: 8, Identifier: "removed-admin", Role: "admin", Status: store.StatusRemoved},
			PasswordHash: cachedHash(t, adminPassword),
		},
	}}
}

func newAuthHandler(t *testing.T, users AuthStore, tokens TokenIssuer) *Auth {
	t.Helper()
	loc, err := time.LoadLocation("Asia/Dhaka")
	if err != nil {
		t.Fatalf("could not load timezone: %v", err)
	}
	cfg := &config.Config{AppEnv: "test", Timezone: "Asia/Dhaka", Location: loc}
	return NewAuth(cfg, users, tokens)
}

func silenceLog(t *testing.T) {
	t.Helper()
	log.SetOutput(io.Discard)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })
}

func post(t *testing.T, handler http.HandlerFunc, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

func errorMessage(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var payload struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("body is not valid JSON: %v (body = %q)", err, rec.Body.String())
	}
	return payload.Error
}

func successMessage(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var payload struct {
		Message string `json:"message"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("body is not valid JSON: %v (body = %q)", err, rec.Body.String())
	}
	return payload.Message
}

const validRegistration = `{"full_name":"Ratul Islam","reg_no":"2023331079","batch":"CSE-23","section":"A",
"email":"ratul@student.sust.edu","phone":"01712345678","blood_group":"O+",
"password":"sust-250-project","confirm_password":"sust-250-project","has_drop_course":false}`

func TestRegisterValidation(t *testing.T) {
	silenceLog(t)
	tests := []struct {
		name       string
		body       string
		wantStatus int
		wantError  string
	}{
		{name: "invalid json", body: `{`, wantStatus: http.StatusBadRequest, wantError: "Invalid request body"},
		{name: "empty body", body: ``, wantStatus: http.StatusBadRequest, wantError: "Invalid request body"},
		{
			name:       "unknown field",
			body:       `{"full_name":"A","role":"admin"}`,
			wantStatus: http.StatusBadRequest,
			wantError:  "Invalid request body",
		},
		{
			name:       "missing full name",
			body:       strings.Replace(validRegistration, `"full_name":"Ratul Islam"`, `"full_name":"  "`, 1),
			wantStatus: http.StatusBadRequest,
			wantError:  "Full name is required.",
		},
		{
			name:       "missing registration number",
			body:       strings.Replace(validRegistration, `"reg_no":"2023331079"`, `"reg_no":""`, 1),
			wantStatus: http.StatusBadRequest,
			wantError:  "Registration number is required.",
		},
		{
			name:       "bad batch",
			body:       strings.Replace(validRegistration, `"batch":"CSE-23"`, `"batch":"CSE2023"`, 1),
			wantStatus: http.StatusBadRequest,
			wantError:  "Batch must look like CSE-23.",
		},
		{
			name:       "bad section",
			body:       strings.Replace(validRegistration, `"section":"A"`, `"section":"AB"`, 1),
			wantStatus: http.StatusBadRequest,
			wantError:  "Section must be a single letter from A to Z.",
		},
		{
			name:       "bad email",
			body:       strings.Replace(validRegistration, `"email":"ratul@student.sust.edu"`, `"email":"ratul-at-sust"`, 1),
			wantStatus: http.StatusBadRequest,
			wantError:  "Email must be a valid email address.",
		},
		{
			name:       "bad phone",
			body:       strings.Replace(validRegistration, `"phone":"01712345678"`, `"phone":"0171"`, 1),
			wantStatus: http.StatusBadRequest,
			wantError:  "Phone must be 10 to 15 digits, with an optional leading +.",
		},
		{
			name:       "bad blood group",
			body:       strings.Replace(validRegistration, `"blood_group":"O+"`, `"blood_group":"C+"`, 1),
			wantStatus: http.StatusBadRequest,
			wantError:  "Blood group must be one of A+, A-, B+, B-, AB+, AB-, O+, O-.",
		},
		{
			name:       "short password",
			body:       strings.ReplaceAll(validRegistration, `"sust-250-project"`, `"short"`),
			wantStatus: http.StatusBadRequest,
			wantError:  "Password must be at least 8 characters long.",
		},
		{
			name:       "confirmation does not match",
			body:       strings.Replace(validRegistration, `"confirm_password":"sust-250-project"`, `"confirm_password":"sust-250-projecT"`, 1),
			wantStatus: http.StatusBadRequest,
			wantError:  "Password and the confirmation do not match.",
		},
		{
			name:       "drop course without a code",
			body:       strings.Replace(validRegistration, `"has_drop_course":false`, `"has_drop_course":true`, 1),
			wantStatus: http.StatusBadRequest,
			wantError:  "Drop course code is required.",
		},
		{
			name: "drop course with a bad code",
			body: strings.Replace(validRegistration, `"has_drop_course":false`,
				`"has_drop_course":true,"drop_course_code":"CSE-23"`, 1),
			wantStatus: http.StatusBadRequest,
			wantError:  "Drop course code must look like CSE-233 or CSE233.",
		},
		{
			name: "the first invalid field is reported",
			body: strings.Replace(strings.Replace(validRegistration, `"full_name":"Ratul Islam"`, `"full_name":""`, 1),
				`"email":"ratul@student.sust.edu"`, `"email":"nonsense"`, 1),
			wantStatus: http.StatusBadRequest,
			wantError:  "Full name is required.",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			users := newAuthStore(t)
			handler := newAuthHandler(t, users, &fakeIssuer{token: issuedToken})
			rec := post(t, handler.Register(), "/api/auth/register", tc.body)
			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d (body = %q)", rec.Code, tc.wantStatus, rec.Body.String())
			}
			if got := errorMessage(t, rec); got != tc.wantError {
				t.Errorf("error = %q, want %q", got, tc.wantError)
			}
			if len(users.created) != 0 {
				t.Errorf("the store was asked to create %d registrations, want 0", len(users.created))
			}
		})
	}
}

func TestRegisterSuccess(t *testing.T) {
	silenceLog(t)
	tests := []struct {
		name               string
		body               string
		wantBatch          string
		wantSection        string
		wantEmail          string
		wantHasDropCourse  bool
		wantDropCourseCode string
	}{
		{
			name:        "without a drop course",
			body:        validRegistration,
			wantBatch:   "CSE-23",
			wantSection: "A",
			wantEmail:   "ratul@student.sust.edu",
		},
		{
			name: "with a drop course",
			body: strings.Replace(validRegistration, `"has_drop_course":false`,
				`"has_drop_course":true,"drop_course_code":"cse-233"`, 1),
			wantBatch:          "CSE-23",
			wantSection:        "A",
			wantEmail:          "ratul@student.sust.edu",
			wantHasDropCourse:  true,
			wantDropCourseCode: "CSE-233",
		},
		{
			name: "values are normalised",
			body: strings.Replace(strings.Replace(validRegistration, `"batch":"CSE-23"`, `"batch":"cse-23"`, 1),
				`"email":"ratul@student.sust.edu"`, `"email":" Ratul@Student.SUST.edu "`, 1),
			wantBatch:   "CSE-23",
			wantSection: "A",
			wantEmail:   "ratul@student.sust.edu",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			users := newAuthStore(t)
			handler := newAuthHandler(t, users, &fakeIssuer{token: issuedToken})
			rec := post(t, handler.Register(), "/api/auth/register", tc.body)
			if rec.Code != http.StatusCreated {
				t.Fatalf("status = %d, want %d (body = %q)", rec.Code, http.StatusCreated, rec.Body.String())
			}
			if got := successMessage(t, rec); got != MessageRegistrationDone {
				t.Errorf("message = %q, want %q", got, MessageRegistrationDone)
			}
			if len(users.created) != 1 {
				t.Fatalf("the store received %d registrations, want 1", len(users.created))
			}
			created := users.created[0]
			if created.Batch != tc.wantBatch {
				t.Errorf("Batch = %q, want %q", created.Batch, tc.wantBatch)
			}
			if created.Section != tc.wantSection {
				t.Errorf("Section = %q, want %q", created.Section, tc.wantSection)
			}
			if created.Email != tc.wantEmail {
				t.Errorf("Email = %q, want %q", created.Email, tc.wantEmail)
			}
			if created.HasDropCourse != tc.wantHasDropCourse {
				t.Errorf("HasDropCourse = %t, want %t", created.HasDropCourse, tc.wantHasDropCourse)
			}
			if created.DropCourseCode != tc.wantDropCourseCode {
				t.Errorf("DropCourseCode = %q, want %q", created.DropCourseCode, tc.wantDropCourseCode)
			}
			if created.PasswordHash == "sust-250-project" || created.PasswordHash == "" {
				t.Fatal("the registration was stored without a real password hash")
			}
			if err := auth.CheckPassword(created.PasswordHash, "sust-250-project"); err != nil {
				t.Errorf("the stored hash does not verify against the submitted password: %v", err)
			}
			if strings.Contains(rec.Body.String(), created.PasswordHash) {
				t.Error("the response body contains the password hash")
			}
		})
	}
}

func TestRegisterStoreErrors(t *testing.T) {
	silenceLog(t)
	tests := []struct {
		name       string
		createErr  error
		wantStatus int
		wantError  string
	}{
		{
			name:       "duplicate account",
			createErr:  store.ErrDuplicateAccount,
			wantStatus: http.StatusConflict,
			wantError:  MessageDuplicateAccount,
		},
		{
			name:       "database failure",
			createErr:  errors.New("connection refused"),
			wantStatus: http.StatusInternalServerError,
			wantError:  middleware.MessageServerError,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			users := newAuthStore(t)
			users.createErr = tc.createErr
			handler := newAuthHandler(t, users, &fakeIssuer{token: issuedToken})
			rec := post(t, handler.Register(), "/api/auth/register", validRegistration)
			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tc.wantStatus)
			}
			if got := errorMessage(t, rec); got != tc.wantError {
				t.Errorf("error = %q, want %q", got, tc.wantError)
			}
		})
	}
}

func TestLogin(t *testing.T) {
	silenceLog(t)
	tests := []struct {
		name        string
		body        string
		wantStatus  int
		wantError   string
		wantRole    string
		wantName    string
		wantBatch   string
		wantSection string
	}{
		{
			name:        "active student",
			body:        `{"identifier":"2023331001","password":"student-pass"}`,
			wantStatus:  http.StatusOK,
			wantRole:    auth.RoleStudent,
			wantName:    "Active Student",
			wantBatch:   "CSE-23",
			wantSection: "A",
		},
		{
			name:        "class representative",
			body:        `{"identifier":"2023331002","password":"student-pass"}`,
			wantStatus:  http.StatusOK,
			wantRole:    auth.RoleCR,
			wantName:    "Class Representative",
			wantBatch:   "CSE-23",
			wantSection: "B",
		},
		{
			name:       "teacher has no batch or section",
			body:       `{"identifier":"T-001","password":"teacher-pass"}`,
			wantStatus: http.StatusOK,
			wantRole:   auth.RoleTeacher,
			wantName:   "Active Teacher",
		},
		{
			name:       "unknown identifier",
			body:       `{"identifier":"2023339999","password":"student-pass"}`,
			wantStatus: http.StatusUnauthorized,
			wantError:  MessageInvalidLogin,
		},
		{
			name:       "wrong password",
			body:       `{"identifier":"2023331001","password":"student-pasS"}`,
			wantStatus: http.StatusUnauthorized,
			wantError:  MessageInvalidLogin,
		},
		{
			name:       "admin cannot use the normal login",
			body:       `{"identifier":"admin","password":"admin-pass"}`,
			wantStatus: http.StatusUnauthorized,
			wantError:  MessageInvalidLogin,
		},
		{
			name:       "pending account",
			body:       `{"identifier":"2023331003","password":"student-pass"}`,
			wantStatus: http.StatusForbidden,
			wantError:  MessagePendingAccount,
		},
		{
			name:       "rejected account",
			body:       `{"identifier":"2023331004","password":"student-pass"}`,
			wantStatus: http.StatusForbidden,
			wantError:  MessageRejectedAccount,
		},
		{
			name:       "removed account",
			body:       `{"identifier":"2023331005","password":"student-pass"}`,
			wantStatus: http.StatusForbidden,
			wantError:  MessageRemovedAccount,
		},
		{
			name:       "wrong password on a pending account is rejected as invalid login",
			body:       `{"identifier":"2023331003","password":"wrong-password"}`,
			wantStatus: http.StatusUnauthorized,
			wantError:  MessageInvalidLogin,
		},
		{
			name:       "missing identifier",
			body:       `{"identifier":"","password":"student-pass"}`,
			wantStatus: http.StatusBadRequest,
			wantError:  "ID is required.",
		},
		{
			name:       "missing password",
			body:       `{"identifier":"2023331001","password":""}`,
			wantStatus: http.StatusBadRequest,
			wantError:  "Password is required.",
		},
		{
			name:       "invalid body",
			body:       `nonsense`,
			wantStatus: http.StatusBadRequest,
			wantError:  "Invalid request body",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			users := newAuthStore(t)
			issuer := &fakeIssuer{token: issuedToken}
			handler := newAuthHandler(t, users, issuer)
			rec := post(t, handler.Login(), "/api/auth/login", tc.body)
			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d (body = %q)", rec.Code, tc.wantStatus, rec.Body.String())
			}
			if tc.wantStatus != http.StatusOK {
				if got := errorMessage(t, rec); got != tc.wantError {
					t.Errorf("error = %q, want %q", got, tc.wantError)
				}
				if issuer.calls != 0 {
					t.Error("a token was issued for a failed login")
				}
				return
			}
			var body loginResponse
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatalf("body is not valid JSON: %v", err)
			}
			if body.Token != issuedToken {
				t.Errorf("token = %q, want %q", body.Token, issuedToken)
			}
			if body.User.Role != tc.wantRole {
				t.Errorf("role = %q, want %q", body.User.Role, tc.wantRole)
			}
			if body.User.Name != tc.wantName {
				t.Errorf("name = %q, want %q", body.User.Name, tc.wantName)
			}
			if body.User.Batch != tc.wantBatch {
				t.Errorf("batch = %q, want %q", body.User.Batch, tc.wantBatch)
			}
			if body.User.Section != tc.wantSection {
				t.Errorf("section = %q, want %q", body.User.Section, tc.wantSection)
			}
			if issuer.role != tc.wantRole {
				t.Errorf("the token was issued for role %q, want %q", issuer.role, tc.wantRole)
			}
			if strings.Contains(rec.Body.String(), "password") {
				t.Error("the response body mentions a password")
			}
		})
	}
}

func TestLoginFailures(t *testing.T) {
	silenceLog(t)
	tests := []struct {
		name       string
		findErr    error
		issueErr   error
		wantStatus int
		wantError  string
	}{
		{
			name:       "database failure",
			findErr:    errors.New("connection refused"),
			wantStatus: http.StatusInternalServerError,
			wantError:  middleware.MessageServerError,
		},
		{
			name:       "token cannot be issued",
			issueErr:   errors.New("signing failed"),
			wantStatus: http.StatusInternalServerError,
			wantError:  middleware.MessageServerError,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			users := newAuthStore(t)
			users.findErr = tc.findErr
			handler := newAuthHandler(t, users, &fakeIssuer{token: issuedToken, err: tc.issueErr})
			rec := post(t, handler.Login(), "/api/auth/login", `{"identifier":"2023331001","password":"student-pass"}`)
			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tc.wantStatus)
			}
			if got := errorMessage(t, rec); got != tc.wantError {
				t.Errorf("error = %q, want %q", got, tc.wantError)
			}
		})
	}
}

func TestAdminLogin(t *testing.T) {
	silenceLog(t)
	tests := []struct {
		name       string
		body       string
		wantStatus int
		wantError  string
	}{
		{name: "active admin", body: `{"identifier":"admin","password":"admin-pass"}`, wantStatus: http.StatusOK},
		{
			name:       "wrong admin password",
			body:       `{"identifier":"admin","password":"admin-pasS"}`,
			wantStatus: http.StatusUnauthorized,
			wantError:  MessageInvalidLogin,
		},
		{
			name:       "student cannot use the admin login",
			body:       `{"identifier":"2023331001","password":"student-pass"}`,
			wantStatus: http.StatusUnauthorized,
			wantError:  MessageInvalidLogin,
		},
		{
			name:       "class representative cannot use the admin login",
			body:       `{"identifier":"2023331002","password":"student-pass"}`,
			wantStatus: http.StatusUnauthorized,
			wantError:  MessageInvalidLogin,
		},
		{
			name:       "teacher cannot use the admin login",
			body:       `{"identifier":"T-001","password":"teacher-pass"}`,
			wantStatus: http.StatusUnauthorized,
			wantError:  MessageInvalidLogin,
		},
		{
			name:       "unknown identifier",
			body:       `{"identifier":"nobody","password":"admin-pass"}`,
			wantStatus: http.StatusUnauthorized,
			wantError:  MessageInvalidLogin,
		},
		{
			name:       "removed admin",
			body:       `{"identifier":"removed-admin","password":"admin-pass"}`,
			wantStatus: http.StatusUnauthorized,
			wantError:  MessageInvalidLogin,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			users := newAuthStore(t)
			issuer := &fakeIssuer{token: issuedToken}
			handler := newAuthHandler(t, users, issuer)
			rec := post(t, handler.AdminLogin(), "/api/auth/admin/login", tc.body)
			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d (body = %q)", rec.Code, tc.wantStatus, rec.Body.String())
			}
			if tc.wantStatus != http.StatusOK {
				if got := errorMessage(t, rec); got != tc.wantError {
					t.Errorf("error = %q, want %q", got, tc.wantError)
				}
				return
			}
			var body loginResponse
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatalf("body is not valid JSON: %v", err)
			}
			if body.Token != issuedToken {
				t.Errorf("token = %q, want %q", body.Token, issuedToken)
			}
			if body.User.Role != auth.RoleAdmin {
				t.Errorf("role = %q, want %q", body.User.Role, auth.RoleAdmin)
			}
			if body.User.Batch != "" || body.User.Section != "" {
				t.Error("the admin response contains a batch or section, want neither")
			}
		})
	}
}

func TestLogout(t *testing.T) {
	silenceLog(t)
	tests := []struct {
		name         string
		withUser     bool
		incrementErr error
		wantStatus   int
		wantMessage  string
		wantError    string
	}{
		{name: "signed in", withUser: true, wantStatus: http.StatusOK, wantMessage: MessageLoggedOut},
		{
			name:       "no user in the context",
			wantStatus: http.StatusUnauthorized,
			wantError:  middleware.MessageUnauthorized,
		},
		{
			name:         "user disappeared",
			withUser:     true,
			incrementErr: store.ErrNotFound,
			wantStatus:   http.StatusUnauthorized,
			wantError:    middleware.MessageSessionExpired,
		},
		{
			name:         "database failure",
			withUser:     true,
			incrementErr: errors.New("connection refused"),
			wantStatus:   http.StatusInternalServerError,
			wantError:    middleware.MessageServerError,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			users := newAuthStore(t)
			users.incrementErr = tc.incrementErr
			handler := newAuthHandler(t, users, &fakeIssuer{token: issuedToken})
			req := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
			if tc.withUser {
				current := middleware.User{ID: 42, Identifier: "2023331001", Role: auth.RoleStudent}
				req = req.WithContext(middleware.WithUser(req.Context(), current))
			}
			rec := httptest.NewRecorder()
			handler.Logout().ServeHTTP(rec, req)
			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d (body = %q)", rec.Code, tc.wantStatus, rec.Body.String())
			}
			if tc.wantMessage != "" {
				if got := successMessage(t, rec); got != tc.wantMessage {
					t.Errorf("message = %q, want %q", got, tc.wantMessage)
				}
				if len(users.incremented) != 1 || users.incremented[0] != 42 {
					t.Errorf("the store was asked to invalidate %v, want [42]", users.incremented)
				}
				return
			}
			if got := errorMessage(t, rec); got != tc.wantError {
				t.Errorf("error = %q, want %q", got, tc.wantError)
			}
		})
	}
}

func TestForgotPassword(t *testing.T) {
	silenceLog(t)
	tests := []struct {
		name        string
		body        string
		setResetErr error
		findErr     error
		wantStatus  int
		wantError   string
	}{
		{name: "active student", body: `{"identifier":"2023331001"}`, wantStatus: http.StatusOK},
		{
			name:       "unknown identifier",
			body:       `{"identifier":"2023339999"}`,
			wantStatus: http.StatusNotFound,
			wantError:  MessageNoActiveAccount,
		},
		{
			name:       "pending account",
			body:       `{"identifier":"2023331003"}`,
			wantStatus: http.StatusNotFound,
			wantError:  MessageNoActiveAccount,
		},
		{
			name:       "removed account",
			body:       `{"identifier":"2023331005"}`,
			wantStatus: http.StatusNotFound,
			wantError:  MessageNoActiveAccount,
		},
		{
			name:       "missing identifier",
			body:       `{"identifier":"  "}`,
			wantStatus: http.StatusBadRequest,
			wantError:  "ID is required.",
		},
		{
			name:       "invalid body",
			body:       `{"identifier":}`,
			wantStatus: http.StatusBadRequest,
			wantError:  "Invalid request body",
		},
		{
			name:        "account became inactive between the two queries",
			body:        `{"identifier":"2023331001"}`,
			setResetErr: store.ErrNotFound,
			wantStatus:  http.StatusNotFound,
			wantError:   MessageNoActiveAccount,
		},
		{
			name:       "database failure",
			body:       `{"identifier":"2023331001"}`,
			findErr:    errors.New("connection refused"),
			wantStatus: http.StatusInternalServerError,
			wantError:  middleware.MessageServerError,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			users := newAuthStore(t)
			users.setResetErr = tc.setResetErr
			users.findErr = tc.findErr
			handler := newAuthHandler(t, users, &fakeIssuer{token: issuedToken})
			before := time.Now()
			rec := post(t, handler.ForgotPassword(), "/api/auth/forgot-password", tc.body)
			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d (body = %q)", rec.Code, tc.wantStatus, rec.Body.String())
			}
			if tc.wantStatus != http.StatusOK {
				if got := errorMessage(t, rec); got != tc.wantError {
					t.Errorf("error = %q, want %q", got, tc.wantError)
				}
				return
			}
			var body forgotPasswordResponse
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatalf("body is not valid JSON: %v", err)
			}
			if body.Message != MessageResetTokenIssued {
				t.Errorf("message = %q, want %q", body.Message, MessageResetTokenIssued)
			}
			if len(body.ResetToken) != resetTokenBytes*2 {
				t.Errorf("reset token has %d characters, want %d", len(body.ResetToken), resetTokenBytes*2)
			}
			if _, err := hex.DecodeString(body.ResetToken); err != nil {
				t.Errorf("reset token is not hexadecimal: %v", err)
			}
			expiresAt, err := time.Parse(time.RFC3339, body.ExpiresAt)
			if err != nil {
				t.Fatalf("expires_at %q is not RFC3339: %v", body.ExpiresAt, err)
			}
			lifetime := expiresAt.Sub(before)
			if lifetime < 14*time.Minute || lifetime > 16*time.Minute {
				t.Errorf("the token expires in %v, want about 15 minutes", lifetime)
			}
			if _, offset := expiresAt.Zone(); offset != int((6 * time.Hour).Seconds()) {
				t.Errorf("expires_at offset = %d seconds, want the Asia/Dhaka offset", offset)
			}
			if len(users.setResetCalls) != 1 {
				t.Fatalf("the store received %d reset tokens, want 1", len(users.setResetCalls))
			}
			call := users.setResetCalls[0]
			if call.userID != 1 {
				t.Errorf("the reset token was stored for user %d, want 1", call.userID)
			}
			if call.tokenHash == body.ResetToken {
				t.Error("the raw reset token was stored, want only its hash")
			}
			if call.tokenHash != hashResetToken(body.ResetToken) {
				t.Error("the stored value is not the SHA-256 hash of the issued token")
			}
		})
	}
}

func TestForgotPasswordIssuesADifferentTokenEveryTime(t *testing.T) {
	silenceLog(t)
	users := newAuthStore(t)
	handler := newAuthHandler(t, users, &fakeIssuer{token: issuedToken})
	seen := make(map[string]bool)
	for i := 0; i < 5; i++ {
		rec := post(t, handler.ForgotPassword(), "/api/auth/forgot-password", `{"identifier":"2023331001"}`)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}
		var body forgotPasswordResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatalf("body is not valid JSON: %v", err)
		}
		if seen[body.ResetToken] {
			t.Fatal("the same reset token was issued twice")
		}
		seen[body.ResetToken] = true
	}
}

func TestResetPassword(t *testing.T) {
	silenceLog(t)
	const token = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	tests := []struct {
		name       string
		body       string
		resetErr   error
		wantStatus int
		wantError  string
	}{
		{
			name:       "valid token",
			body:       `{"identifier":"2023331001","token":"` + token + `","new_password":"brand-new-pass","confirm_password":"brand-new-pass"}`,
			wantStatus: http.StatusOK,
		},
		{
			name:       "wrong token",
			body:       `{"identifier":"2023331001","token":"wrong-token","new_password":"brand-new-pass","confirm_password":"brand-new-pass"}`,
			resetErr:   store.ErrInvalidResetToken,
			wantStatus: http.StatusBadRequest,
			wantError:  MessageInvalidResetToken,
		},
		{
			name:       "expired token",
			body:       `{"identifier":"2023331001","token":"` + token + `","new_password":"brand-new-pass","confirm_password":"brand-new-pass"}`,
			resetErr:   store.ErrInvalidResetToken,
			wantStatus: http.StatusBadRequest,
			wantError:  MessageInvalidResetToken,
		},
		{
			name:       "unknown identifier",
			body:       `{"identifier":"2023339999","token":"` + token + `","new_password":"brand-new-pass","confirm_password":"brand-new-pass"}`,
			resetErr:   store.ErrNotFound,
			wantStatus: http.StatusBadRequest,
			wantError:  MessageInvalidResetToken,
		},
		{
			name:       "missing token",
			body:       `{"identifier":"2023331001","token":"","new_password":"brand-new-pass","confirm_password":"brand-new-pass"}`,
			wantStatus: http.StatusBadRequest,
			wantError:  "Reset token is required.",
		},
		{
			name:       "missing identifier",
			body:       `{"identifier":"","token":"` + token + `","new_password":"brand-new-pass","confirm_password":"brand-new-pass"}`,
			wantStatus: http.StatusBadRequest,
			wantError:  "ID is required.",
		},
		{
			name:       "short new password",
			body:       `{"identifier":"2023331001","token":"` + token + `","new_password":"short","confirm_password":"short"}`,
			wantStatus: http.StatusBadRequest,
			wantError:  "New password must be at least 8 characters long.",
		},
		{
			name:       "confirmation does not match",
			body:       `{"identifier":"2023331001","token":"` + token + `","new_password":"brand-new-pass","confirm_password":"other-pass"}`,
			wantStatus: http.StatusBadRequest,
			wantError:  "New password and the confirmation do not match.",
		},
		{
			name:       "invalid body",
			body:       `{"identifier":"x","token":"y","new_password":"z","confirm_password":"z","extra":1}`,
			wantStatus: http.StatusBadRequest,
			wantError:  "Invalid request body",
		},
		{
			name:       "database failure",
			body:       `{"identifier":"2023331001","token":"` + token + `","new_password":"brand-new-pass","confirm_password":"brand-new-pass"}`,
			resetErr:   errors.New("connection refused"),
			wantStatus: http.StatusInternalServerError,
			wantError:  middleware.MessageServerError,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			users := newAuthStore(t)
			users.resetErr = tc.resetErr
			handler := newAuthHandler(t, users, &fakeIssuer{token: issuedToken})
			rec := post(t, handler.ResetPassword(), "/api/auth/reset-password", tc.body)
			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d (body = %q)", rec.Code, tc.wantStatus, rec.Body.String())
			}
			if tc.wantStatus != http.StatusOK {
				if got := errorMessage(t, rec); got != tc.wantError {
					t.Errorf("error = %q, want %q", got, tc.wantError)
				}
				return
			}
			if got := successMessage(t, rec); got != MessagePasswordUpdated {
				t.Errorf("message = %q, want %q", got, MessagePasswordUpdated)
			}
			if len(users.resetCalls) != 1 {
				t.Fatalf("the store received %d reset calls, want 1", len(users.resetCalls))
			}
			call := users.resetCalls[0]
			if call.identifier != "2023331001" {
				t.Errorf("identifier = %q, want %q", call.identifier, "2023331001")
			}
			if call.tokenHash != hashResetToken(token) {
				t.Error("the store did not receive the SHA-256 hash of the submitted token")
			}
			if call.tokenHash == token {
				t.Error("the raw token was passed to the store, want only its hash")
			}
			if err := auth.CheckPassword(call.passwordHash, "brand-new-pass"); err != nil {
				t.Errorf("the stored hash does not verify against the new password: %v", err)
			}
		})
	}
}

package handler

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/raHI-79/250-project/backend/internal/auth"
	"github.com/raHI-79/250-project/backend/internal/mailer"
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

type startResetCall struct {
	userID    int64
	codeHash  string
	expiresAt time.Time
	sentAt    time.Time
}

type verifyCall struct {
	userID    int64
	tokenHash string
	expiresAt time.Time
}

type sentMail struct {
	to      string
	subject string
	body    string
}

type fakeMailer struct {
	configured bool
	err        error
	guard      sync.Mutex
	sent       []sentMail
	notify     chan sentMail
}

func (f *fakeMailer) delivered() []sentMail {
	f.guard.Lock()
	defer f.guard.Unlock()
	return append([]sentMail(nil), f.sent...)
}

func (f *fakeMailer) waitForMail(t *testing.T) sentMail {
	t.Helper()
	select {
	case mail := <-f.notify:
		return mail
	case <-time.After(2 * time.Second):
		t.Fatal("no email was sent within two seconds")
		return sentMail{}
	}
}

func (f *fakeMailer) Configured() bool {
	return f.configured
}

func (f *fakeMailer) Send(ctx context.Context, to, subject, body string) error {
	if !f.configured {
		return mailer.ErrNotConfigured
	}
	if f.err != nil {
		return f.err
	}
	mail := sentMail{to: to, subject: subject, body: body}
	f.guard.Lock()
	f.sent = append(f.sent, mail)
	f.guard.Unlock()
	if f.notify != nil {
		select {
		case f.notify <- mail:
		default:
		}
	}
	return nil
}

type fakeAuthStore struct {
	*accountStore
	users           map[string]store.LoginUser
	targets         map[string]store.ResetTarget
	challenges      map[string]store.ResetChallenge
	findErr         error
	incrementErr    error
	targetErr       error
	challengeErr    error
	startResetErr   error
	attemptErr      error
	verifyErr       error
	resetErr        error
	attempts        int
	incremented     []int64
	startResetCalls []startResetCall
	verifyCalls     []verifyCall
	clearedUsers    []int64
	resetCalls      []resetCall
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

func (f *fakeAuthStore) FindResetTarget(ctx context.Context, identifier string) (store.ResetTarget, error) {
	if f.targetErr != nil {
		return store.ResetTarget{}, f.targetErr
	}
	target, ok := f.targets[identifier]
	if !ok {
		return store.ResetTarget{}, store.ErrNotFound
	}
	return target, nil
}

func (f *fakeAuthStore) FindResetChallenge(ctx context.Context, identifier string) (store.ResetChallenge, error) {
	if f.challengeErr != nil {
		return store.ResetChallenge{}, f.challengeErr
	}
	challenge, ok := f.challenges[identifier]
	if !ok {
		return store.ResetChallenge{}, store.ErrNotFound
	}
	return challenge, nil
}

func (f *fakeAuthStore) StartPasswordReset(ctx context.Context, userID int64, codeHash string, expiresAt, sentAt time.Time) error {
	if f.startResetErr != nil {
		return f.startResetErr
	}
	f.startResetCalls = append(f.startResetCalls, startResetCall{
		userID:    userID,
		codeHash:  codeHash,
		expiresAt: expiresAt,
		sentAt:    sentAt,
	})
	return nil
}

func (f *fakeAuthStore) CountResetAttempt(ctx context.Context, userID int64) (int, error) {
	if f.attemptErr != nil {
		return 0, f.attemptErr
	}
	f.attempts++
	return f.attempts, nil
}

func (f *fakeAuthStore) ClearPasswordReset(ctx context.Context, userID int64) error {
	f.clearedUsers = append(f.clearedUsers, userID)
	return nil
}

func (f *fakeAuthStore) VerifyResetCode(ctx context.Context, userID int64, tokenHash string, expiresAt time.Time) error {
	if f.verifyErr != nil {
		return f.verifyErr
	}
	f.verifyCalls = append(f.verifyCalls, verifyCall{userID: userID, tokenHash: tokenHash, expiresAt: expiresAt})
	return nil
}

func (f *fakeAuthStore) FinishPasswordReset(ctx context.Context, identifier, tokenHash, passwordHash string) error {
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
	return &fakeAuthStore{accountStore: newAccountStore(), users: map[string]store.LoginUser{
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
	},
		targets: map[string]store.ResetTarget{
			"2023331001": {
				AuthUser:    store.AuthUser{ID: 1, Identifier: "2023331001", Role: "student", Status: store.StatusActive},
				Email:       "nayeem@student.sust.edu",
				Phone:       "01712345678",
				DateOfBirth: "2003-05-14",
			},
			"2023331003": {
				AuthUser:    store.AuthUser{ID: 3, Identifier: "2023331003", Role: "student", Status: store.StatusPending},
				Email:       "pending@student.sust.edu",
				Phone:       "01712345679",
				DateOfBirth: "2004-01-02",
			},
			"T-001": {
				AuthUser:    store.AuthUser{ID: 6, Identifier: "T-001", Role: "teacher", Status: store.StatusActive},
				Email:       "teacher@sust.edu",
				Phone:       "01812345678",
				DateOfBirth: "1988-03-09",
			},
			"admin": {
				AuthUser: store.AuthUser{ID: 7, Identifier: "admin", Role: "admin", Status: store.StatusActive},
				Email:    "admin@sust.edu",
				Phone:    "01912345678",
			},
		},
		challenges: map[string]store.ResetChallenge{},
	}
}

func newAuthHandlerWithMailer(t *testing.T, users AuthStore, mail mailer.Sender, appEnv string) *Auth {
	t.Helper()
	return NewAuth(accountConfig(t, appEnv), users, &fakeIssuer{token: issuedToken}, mail,
		testProofs(t, auth.ProofTTL), nil, testPhotos(t))
}

func planCodeChallenge(users *fakeAuthStore, identifier string, userID int64, code string, stage string, expiresIn time.Duration, attempts int) {
	expiresAt := time.Now().Add(expiresIn)
	users.challenges[identifier] = store.ResetChallenge{
		UserID:    userID,
		Status:    store.StatusActive,
		TokenHash: hashResetSecret(code),
		Stage:     stage,
		Attempts:  attempts,
		ExpiresAt: &expiresAt,
	}
}

func newAuthHandler(t *testing.T, users AuthStore, tokens TokenIssuer) *Auth {
	t.Helper()
	return NewAuth(accountConfig(t, "test"), users, tokens, &fakeMailer{},
		testProofs(t, auth.ProofTTL), nil, testPhotos(t))
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
		name          string
		body          string
		startResetErr error
		targetErr     error
		sentAgo       time.Duration
		wantStatus    int
		wantError     string
		wantHint      string
		wantUserID    int64
	}{
		{
			name:       "active student by date of birth",
			body:       `{"identifier":"2023331001","date_of_birth":"2003-05-14"}`,
			wantStatus: http.StatusOK,
			wantHint:   "n***@student.sust.edu",
			wantUserID: 1,
		},
		{
			name:       "teacher by date of birth",
			body:       `{"identifier":"T-001","date_of_birth":"1988-03-09"}`,
			wantStatus: http.StatusOK,
			wantHint:   "t***@sust.edu",
			wantUserID: 6,
		},
		{
			name:       "admin by phone",
			body:       `{"identifier":"admin","phone":"01912345678"}`,
			wantStatus: http.StatusOK,
			wantHint:   "a***@sust.edu",
			wantUserID: 7,
		},
		{
			name:       "admin phone written with the country code",
			body:       `{"identifier":"admin","phone":"+8801912345678"}`,
			wantStatus: http.StatusOK,
			wantHint:   "a***@sust.edu",
			wantUserID: 7,
		},
		{
			name:       "student date of birth does not match",
			body:       `{"identifier":"2023331001","date_of_birth":"2003-05-15"}`,
			wantStatus: http.StatusBadRequest,
			wantError:  MessageResetBirthMismatch,
		},
		{
			name:       "student phone is not accepted",
			body:       `{"identifier":"2023331001","phone":"01712345678"}`,
			wantStatus: http.StatusBadRequest,
			wantError:  MessageResetBirthMismatch,
		},
		{
			name:       "admin date of birth is not accepted",
			body:       `{"identifier":"admin","date_of_birth":"2003-05-14"}`,
			wantStatus: http.StatusBadRequest,
			wantError:  MessageResetMismatch,
		},
		{
			name:       "unknown identifier",
			body:       `{"identifier":"2023339999","date_of_birth":"2003-05-14"}`,
			wantStatus: http.StatusBadRequest,
			wantError:  MessageResetBirthMismatch,
		},
		{
			name:       "pending account",
			body:       `{"identifier":"2023331003","date_of_birth":"2004-01-02"}`,
			wantStatus: http.StatusBadRequest,
			wantError:  MessageResetBirthMismatch,
		},
		{
			name:       "nothing to match on",
			body:       `{"identifier":"2023331001"}`,
			wantStatus: http.StatusBadRequest,
			wantError:  MessageDateOfBirthRequired,
		},
		{
			name:       "missing identifier",
			body:       `{"identifier":"  ","date_of_birth":"2003-05-14"}`,
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
			name:       "inside the resend cooldown",
			body:       `{"identifier":"2023331001","date_of_birth":"2003-05-14"}`,
			sentAgo:    20 * time.Second,
			wantStatus: http.StatusTooManyRequests,
			wantError:  MessageResetCooldown,
		},
		{
			name:       "after the resend cooldown",
			body:       `{"identifier":"2023331001","date_of_birth":"2003-05-14"}`,
			sentAgo:    90 * time.Second,
			wantStatus: http.StatusOK,
			wantHint:   "n***@student.sust.edu",
			wantUserID: 1,
		},
		{
			name:          "account became inactive between the two queries",
			body:          `{"identifier":"2023331001","date_of_birth":"2003-05-14"}`,
			startResetErr: store.ErrNotFound,
			wantStatus:    http.StatusBadRequest,
			wantError:     MessageResetBirthMismatch,
		},
		{
			name:       "database failure",
			body:       `{"identifier":"2023331001","date_of_birth":"2003-05-14"}`,
			targetErr:  errors.New("connection refused"),
			wantStatus: http.StatusInternalServerError,
			wantError:  middleware.MessageServerError,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			users := newAuthStore(t)
			users.startResetErr = tc.startResetErr
			users.targetErr = tc.targetErr
			if tc.sentAgo != 0 {
				sentAt := time.Now().Add(-tc.sentAgo)
				target := users.targets["2023331001"]
				target.ResetSentAt = &sentAt
				users.targets["2023331001"] = target
			}
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
			if body.Message != MessageResetCodeSent {
				t.Errorf("message = %q, want %q", body.Message, MessageResetCodeSent)
			}
			if body.EmailHint != tc.wantHint {
				t.Errorf("email hint = %q, want %q", body.EmailHint, tc.wantHint)
			}
			if !isSixDigitCode(body.DevCode) {
				t.Errorf("dev code = %q, want 6 digits while no mailer is configured", body.DevCode)
			}
			expiresAt, err := time.Parse(time.RFC3339, body.ExpiresAt)
			if err != nil {
				t.Fatalf("expires_at %q is not RFC3339: %v", body.ExpiresAt, err)
			}
			lifetime := expiresAt.Sub(before)
			if lifetime < 9*time.Minute || lifetime > 11*time.Minute {
				t.Errorf("the code expires in %v, want about 10 minutes", lifetime)
			}
			if _, offset := expiresAt.Zone(); offset != int((6 * time.Hour).Seconds()) {
				t.Errorf("expires_at offset = %d seconds, want the Asia/Dhaka offset", offset)
			}
			if len(users.startResetCalls) != 1 {
				t.Fatalf("the store received %d reset codes, want 1", len(users.startResetCalls))
			}
			call := users.startResetCalls[0]
			if call.userID != tc.wantUserID {
				t.Errorf("the code was stored for user %d, want %d", call.userID, tc.wantUserID)
			}
			if call.codeHash == body.DevCode {
				t.Error("the raw code was stored, want only its hash")
			}
			if call.codeHash != hashResetSecret(body.DevCode) {
				t.Error("the stored value is not the SHA-256 hash of the issued code")
			}
			if call.sentAt.IsZero() {
				t.Error("the store did not receive a sent_at timestamp for the cooldown")
			}
		})
	}
}

func TestForgotPasswordIssuesADifferentCodeEveryTime(t *testing.T) {
	silenceLog(t)
	users := newAuthStore(t)
	handler := newAuthHandler(t, users, &fakeIssuer{token: issuedToken})
	seen := make(map[string]bool)
	for i := 0; i < 5; i++ {
		rec := post(t, handler.ForgotPassword(), "/api/auth/forgot-password",
			`{"identifier":"2023331001","date_of_birth":"2003-05-14"}`)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200 (body = %q)", rec.Code, rec.Body.String())
		}
		var body forgotPasswordResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatalf("body is not valid JSON: %v", err)
		}
		seen[body.DevCode] = true
	}
	if len(seen) < 2 {
		t.Error("the same reset code was issued every time, want random codes")
	}
}

func TestForgotPasswordDevCodeAndMail(t *testing.T) {
	silenceLog(t)
	tests := []struct {
		name        string
		appEnv      string
		configured  bool
		wantDevCode bool
		wantMail    bool
	}{
		{name: "development without a mailer", appEnv: "development", configured: false, wantDevCode: true},
		{name: "production without a mailer", appEnv: "production", configured: false},
		{name: "development with a mailer", appEnv: "development", configured: true, wantMail: true},
		{name: "production with a mailer", appEnv: "production", configured: true, wantMail: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			users := newAuthStore(t)
			mail := &fakeMailer{configured: tc.configured}
			handler := newAuthHandlerWithMailer(t, users, mail, tc.appEnv)
			rec := post(t, handler.ForgotPassword(), "/api/auth/forgot-password",
				`{"identifier":"2023331001","date_of_birth":"2003-05-14"}`)
			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200 (body = %q)", rec.Code, rec.Body.String())
			}
			var body forgotPasswordResponse
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatalf("body is not valid JSON: %v", err)
			}
			if tc.wantDevCode && !isSixDigitCode(body.DevCode) {
				t.Errorf("dev code = %q, want 6 digits", body.DevCode)
			}
			if !tc.wantDevCode && body.DevCode != "" {
				t.Errorf("dev code = %q, want it left out", body.DevCode)
			}
			if !tc.wantMail {
				if len(mail.sent) != 0 {
					t.Errorf("the mailer sent %d messages, want none", len(mail.sent))
				}
				return
			}
			if len(mail.sent) != 1 {
				t.Fatalf("the mailer sent %d messages, want 1", len(mail.sent))
			}
			sent := mail.sent[0]
			if sent.to != "nayeem@student.sust.edu" {
				t.Errorf("recipient = %q, want the registered email", sent.to)
			}
			if sent.subject != ResetEmailSubject {
				t.Errorf("subject = %q, want %q", sent.subject, ResetEmailSubject)
			}
			if !strings.HasPrefix(sent.body, "Your password reset code is ") ||
				!strings.HasSuffix(sent.body, ". It expires in 10 minutes.") {
				t.Errorf("body = %q, want the reset code sentence", sent.body)
			}
			if len(users.startResetCalls) != 1 {
				t.Fatalf("the store received %d reset codes, want 1", len(users.startResetCalls))
			}
		})
	}
}

func TestForgotPasswordClearsTheCodeWhenTheMailFails(t *testing.T) {
	silenceLog(t)
	users := newAuthStore(t)
	mail := &fakeMailer{configured: true, err: errors.New("mailbox unavailable")}
	handler := newAuthHandlerWithMailer(t, users, mail, "production")
	rec := post(t, handler.ForgotPassword(), "/api/auth/forgot-password",
		`{"identifier":"2023331001","date_of_birth":"2003-05-14"}`)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500 (body = %q)", rec.Code, rec.Body.String())
	}
	if got := errorMessage(t, rec); got != middleware.MessageServerError {
		t.Errorf("error = %q, want %q", got, middleware.MessageServerError)
	}
	if len(users.clearedUsers) != 1 || users.clearedUsers[0] != 1 {
		t.Errorf("cleared users = %v, want the reset fields of user 1 cleared", users.clearedUsers)
	}
}

func TestVerifyResetCode(t *testing.T) {
	silenceLog(t)
	const code = "123456"
	tests := []struct {
		name         string
		body         string
		stage        string
		expiresIn    time.Duration
		attempts     int
		noChallenge  bool
		challengeErr error
		verifyErr    error
		wantStatus   int
		wantError    string
	}{
		{name: "right code", body: `{"identifier":"2023331001","code":"123456"}`, wantStatus: http.StatusOK},
		{
			name:       "wrong code",
			body:       `{"identifier":"2023331001","code":"999999"}`,
			wantStatus: http.StatusBadRequest,
			wantError:  MessageInvalidResetCode,
		},
		{
			name:       "fifth wrong code",
			body:       `{"identifier":"2023331001","code":"999999"}`,
			attempts:   4,
			wantStatus: http.StatusBadRequest,
			wantError:  MessageTooManyResetCodes,
		},
		{
			name:       "expired code",
			body:       `{"identifier":"2023331001","code":"123456"}`,
			expiresIn:  -time.Minute,
			wantStatus: http.StatusBadRequest,
			wantError:  MessageInvalidResetCode,
		},
		{
			name:       "already verified stage",
			body:       `{"identifier":"2023331001","code":"123456"}`,
			stage:      store.ResetStageVerified,
			wantStatus: http.StatusBadRequest,
			wantError:  MessageInvalidResetCode,
		},
		{
			name:        "no reset in progress",
			body:        `{"identifier":"2023331001","code":"123456"}`,
			noChallenge: true,
			wantStatus:  http.StatusBadRequest,
			wantError:   MessageInvalidResetCode,
		},
		{
			name:       "code is not six digits",
			body:       `{"identifier":"2023331001","code":"12345"}`,
			wantStatus: http.StatusBadRequest,
			wantError:  MessageInvalidResetCode,
		},
		{
			name:       "code is not numeric",
			body:       `{"identifier":"2023331001","code":"12345a"}`,
			wantStatus: http.StatusBadRequest,
			wantError:  MessageInvalidResetCode,
		},
		{
			name:       "missing identifier",
			body:       `{"identifier":"  ","code":"123456"}`,
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
			name:       "the reset moved on between the two queries",
			body:       `{"identifier":"2023331001","code":"123456"}`,
			verifyErr:  store.ErrInvalidResetToken,
			wantStatus: http.StatusBadRequest,
			wantError:  MessageInvalidResetCode,
		},
		{
			name:         "database failure",
			body:         `{"identifier":"2023331001","code":"123456"}`,
			challengeErr: errors.New("connection refused"),
			wantStatus:   http.StatusInternalServerError,
			wantError:    middleware.MessageServerError,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			users := newAuthStore(t)
			users.challengeErr = tc.challengeErr
			users.verifyErr = tc.verifyErr
			users.attempts = tc.attempts
			if !tc.noChallenge {
				stage := tc.stage
				if stage == "" {
					stage = store.ResetStageCode
				}
				expiresIn := tc.expiresIn
				if expiresIn == 0 {
					expiresIn = 10 * time.Minute
				}
				planCodeChallenge(users, "2023331001", 1, code, stage, expiresIn, tc.attempts)
			}
			handler := newAuthHandler(t, users, &fakeIssuer{token: issuedToken})
			before := time.Now()
			rec := post(t, handler.VerifyResetCode(), "/api/auth/verify-reset-code", tc.body)
			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d (body = %q)", rec.Code, tc.wantStatus, rec.Body.String())
			}
			if tc.wantStatus != http.StatusOK {
				if got := errorMessage(t, rec); got != tc.wantError {
					t.Errorf("error = %q, want %q", got, tc.wantError)
				}
				if tc.wantError == MessageTooManyResetCodes {
					if len(users.clearedUsers) != 1 || users.clearedUsers[0] != 1 {
						t.Errorf("cleared users = %v, want the reset fields of user 1 cleared", users.clearedUsers)
					}
				} else if len(users.clearedUsers) != 0 {
					t.Errorf("cleared users = %v, want none", users.clearedUsers)
				}
				return
			}
			var body verifyResetCodeResponse
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatalf("body is not valid JSON: %v", err)
			}
			if len(body.ResetToken) != resetTokenBytes*2 {
				t.Errorf("reset token has %d characters, want %d", len(body.ResetToken), resetTokenBytes*2)
			}
			if _, err := hex.DecodeString(body.ResetToken); err != nil {
				t.Errorf("reset token is not hexadecimal: %v", err)
			}
			if len(users.verifyCalls) != 1 {
				t.Fatalf("the store received %d verified tokens, want 1", len(users.verifyCalls))
			}
			call := users.verifyCalls[0]
			if call.userID != 1 {
				t.Errorf("the token was stored for user %d, want 1", call.userID)
			}
			if call.tokenHash == body.ResetToken {
				t.Error("the raw reset token was stored, want only its hash")
			}
			if call.tokenHash != hashResetSecret(body.ResetToken) {
				t.Error("the stored value is not the SHA-256 hash of the issued token")
			}
			lifetime := call.expiresAt.Sub(before)
			if lifetime < 9*time.Minute || lifetime > 11*time.Minute {
				t.Errorf("the token expires in %v, want about 10 minutes", lifetime)
			}
		})
	}
}

func TestVerifyResetCodeCountsEveryWrongCode(t *testing.T) {
	silenceLog(t)
	users := newAuthStore(t)
	planCodeChallenge(users, "2023331001", 1, "123456", store.ResetStageCode, 10*time.Minute, 0)
	handler := newAuthHandler(t, users, &fakeIssuer{token: issuedToken})
	for attempt := 1; attempt <= maxResetAttempts; attempt++ {
		rec := post(t, handler.VerifyResetCode(), "/api/auth/verify-reset-code",
			`{"identifier":"2023331001","code":"999999"}`)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("attempt %d status = %d, want 400", attempt, rec.Code)
		}
		want := MessageInvalidResetCode
		if attempt == maxResetAttempts {
			want = MessageTooManyResetCodes
		}
		if got := errorMessage(t, rec); got != want {
			t.Errorf("attempt %d error = %q, want %q", attempt, got, want)
		}
	}
	if len(users.clearedUsers) != 1 {
		t.Errorf("the reset fields were cleared %d times, want once", len(users.clearedUsers))
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
			name:       "verified token",
			body:       `{"identifier":"2023331001","reset_token":"` + token + `","new_password":"brand-new-pass","confirm_password":"brand-new-pass"}`,
			wantStatus: http.StatusOK,
		},
		{
			name:       "wrong token",
			body:       `{"identifier":"2023331001","reset_token":"wrong-token","new_password":"brand-new-pass","confirm_password":"brand-new-pass"}`,
			resetErr:   store.ErrInvalidResetToken,
			wantStatus: http.StatusBadRequest,
			wantError:  MessageInvalidResetToken,
		},
		{
			name:       "token still at the code stage",
			body:       `{"identifier":"2023331001","reset_token":"` + token + `","new_password":"brand-new-pass","confirm_password":"brand-new-pass"}`,
			resetErr:   store.ErrInvalidResetToken,
			wantStatus: http.StatusBadRequest,
			wantError:  MessageInvalidResetToken,
		},
		{
			name:       "unknown identifier",
			body:       `{"identifier":"2023339999","reset_token":"` + token + `","new_password":"brand-new-pass","confirm_password":"brand-new-pass"}`,
			resetErr:   store.ErrNotFound,
			wantStatus: http.StatusBadRequest,
			wantError:  MessageInvalidResetToken,
		},
		{
			name:       "missing token",
			body:       `{"identifier":"2023331001","reset_token":"","new_password":"brand-new-pass","confirm_password":"brand-new-pass"}`,
			wantStatus: http.StatusBadRequest,
			wantError:  "Reset token is required.",
		},
		{
			name:       "missing identifier",
			body:       `{"identifier":"","reset_token":"` + token + `","new_password":"brand-new-pass","confirm_password":"brand-new-pass"}`,
			wantStatus: http.StatusBadRequest,
			wantError:  "ID is required.",
		},
		{
			name:       "short new password",
			body:       `{"identifier":"2023331001","reset_token":"` + token + `","new_password":"short","confirm_password":"short"}`,
			wantStatus: http.StatusBadRequest,
			wantError:  "New password must be at least 8 characters long.",
		},
		{
			name:       "confirmation does not match",
			body:       `{"identifier":"2023331001","reset_token":"` + token + `","new_password":"brand-new-pass","confirm_password":"other-pass"}`,
			wantStatus: http.StatusBadRequest,
			wantError:  "New password and the confirmation do not match.",
		},
		{
			name:       "invalid body",
			body:       `{"identifier":"x","reset_token":"y","new_password":"z","confirm_password":"z","extra":1}`,
			wantStatus: http.StatusBadRequest,
			wantError:  "Invalid request body",
		},
		{
			name:       "database failure",
			body:       `{"identifier":"2023331001","reset_token":"` + token + `","new_password":"brand-new-pass","confirm_password":"brand-new-pass"}`,
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
			if call.tokenHash != hashResetSecret(token) {
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

func TestEmailHint(t *testing.T) {
	tests := []struct {
		name  string
		email string
		want  string
	}{
		{name: "student address", email: "nayeem@student.sust.edu", want: "n***@student.sust.edu"},
		{name: "single letter local part", email: "a@sust.edu", want: "a***@sust.edu"},
		{name: "no at sign", email: "nayeem", want: ""},
		{name: "empty", email: "", want: ""},
		{name: "leading at sign", email: "@sust.edu", want: ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := emailHint(tc.email); got != tc.want {
				t.Errorf("emailHint(%q) = %q, want %q", tc.email, got, tc.want)
			}
		})
	}
}

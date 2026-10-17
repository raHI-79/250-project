package server

import (
	"context"
	"encoding/json"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/raHI-79/250-project/backend/internal/auth"
	"github.com/raHI-79/250-project/backend/internal/config"
	"github.com/raHI-79/250-project/backend/internal/db"
	"github.com/raHI-79/250-project/backend/internal/dbtest"
	"github.com/raHI-79/250-project/backend/internal/handler"
	"github.com/raHI-79/250-project/backend/internal/middleware"
	"github.com/raHI-79/250-project/backend/internal/notify"
	"github.com/raHI-79/250-project/backend/internal/store"
	"github.com/raHI-79/250-project/backend/internal/upload"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

const flowRegistration = `{"full_name":"Flow Student","reg_no":"2023331500","batch":"CSE-23","section":"A",
"email":"flow@student.sust.edu","phone":"01712345678","blood_group":"B+",
"password":"first-password","confirm_password":"first-password","has_drop_course":false}`

func TestMain(m *testing.M) {
	if err := dbtest.LoadEnv(); err != nil {
		log.Fatalf("server: could not load backend/.env: %v", err)
	}
	os.Exit(m.Run())
}

func newFlowRouter(t *testing.T) (context.Context, *pgxpool.Pool, http.Handler) {
	t.Helper()
	ctx, pool := dbtest.Pool(t)
	if err := db.Migrate(ctx, pool); err != nil {
		t.Fatalf("db.Migrate() returned an unexpected error: %v", err)
	}
	loc, err := time.LoadLocation("Asia/Dhaka")
	if err != nil {
		t.Fatalf("could not load timezone: %v", err)
	}
	tokens, err := auth.NewTokens(testSecret, time.Hour)
	if err != nil {
		t.Fatalf("auth.NewTokens() returned an unexpected error: %v", err)
	}
	userStore := store.New(pool)
	alerts := notify.New(userStore)
	log.SetOutput(io.Discard)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })
	return ctx, pool, NewRouter(Dependencies{
		Config: &config.Config{
			AppEnv:         "test",
			Timezone:       "Asia/Dhaka",
			Location:       loc,
			FrontendOrigin: testFrontend,
		},
		Database: stubPinger{},
		Tokens:   tokens,
		Users:    userStore,
		Auth:     userStore,
		People:   userStore,
		Catalog:  userStore,
		Lookups:  userStore,
		Alerts:   alerts,
		Classes:  userStore,
		Notifier: alerts,
		Teachers: userStore,
		Students: userStore,
		Alarms:   alerts,
		Notices:  userStore,
		Library:  userStore,
		Contacts: userStore,
		Files:    upload.NewSaver(t.TempDir(), 1),
		Mail:     userStore,
		Votes:    userStore,
	})
}

func call(t *testing.T, router http.Handler, method, path, body, token string) *httptest.ResponseRecorder {
	t.Helper()
	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, reader)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func field(t *testing.T, rec *httptest.ResponseRecorder, name string) string {
	t.Helper()
	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("body is not valid JSON: %v (body = %q)", err, rec.Body.String())
	}
	value, ok := payload[name]
	if !ok {
		t.Fatalf("the response has no %q field (body = %q)", name, rec.Body.String())
	}
	text, ok := value.(string)
	if !ok {
		t.Fatalf("the %q field is not a string (body = %q)", name, rec.Body.String())
	}
	return text
}

func TestAuthenticationFlow(t *testing.T) {
	ctx, pool, router := newFlowRouter(t)
	registered := call(t, router, http.MethodPost, "/api/auth/register", flowRegistration, "")
	if registered.Code != http.StatusCreated {
		t.Fatalf("register status = %d, want 201 (body = %q)", registered.Code, registered.Body.String())
	}
	if got := field(t, registered, "message"); got != handler.MessageRegistrationDone {
		t.Errorf("register message = %q, want %q", got, handler.MessageRegistrationDone)
	}
	duplicate := call(t, router, http.MethodPost, "/api/auth/register", flowRegistration, "")
	if duplicate.Code != http.StatusConflict {
		t.Fatalf("duplicate register status = %d, want 409", duplicate.Code)
	}
	if got := field(t, duplicate, "error"); got != handler.MessageDuplicateAccount {
		t.Errorf("duplicate error = %q, want %q", got, handler.MessageDuplicateAccount)
	}
	const loginBody = `{"identifier":"2023331500","password":"first-password"}`
	pending := call(t, router, http.MethodPost, "/api/auth/login", loginBody, "")
	if pending.Code != http.StatusForbidden {
		t.Fatalf("pending login status = %d, want 403 (body = %q)", pending.Code, pending.Body.String())
	}
	if got := field(t, pending, "error"); got != handler.MessagePendingAccount {
		t.Errorf("pending login error = %q, want %q", got, handler.MessagePendingAccount)
	}
	if _, err := pool.Exec(ctx, "UPDATE users SET status = 'active' WHERE identifier = $1", "2023331500"); err != nil {
		t.Fatalf("could not activate the account: %v", err)
	}
	loggedIn := call(t, router, http.MethodPost, "/api/auth/login", loginBody, "")
	if loggedIn.Code != http.StatusOK {
		t.Fatalf("login status = %d, want 200 (body = %q)", loggedIn.Code, loggedIn.Body.String())
	}
	token := field(t, loggedIn, "token")
	if token == "" {
		t.Fatal("login returned an empty token")
	}
	me := call(t, router, http.MethodGet, "/api/me", "", token)
	if me.Code != http.StatusOK {
		t.Fatalf("me status = %d, want 200 (body = %q)", me.Code, me.Body.String())
	}
	if got := field(t, me, "identifier"); got != "2023331500" {
		t.Errorf("me identifier = %q, want %q", got, "2023331500")
	}
	if got := field(t, me, "role"); got != auth.RoleStudent {
		t.Errorf("me role = %q, want %q", got, auth.RoleStudent)
	}
	loggedOut := call(t, router, http.MethodPost, "/api/auth/logout", "", token)
	if loggedOut.Code != http.StatusOK {
		t.Fatalf("logout status = %d, want 200 (body = %q)", loggedOut.Code, loggedOut.Body.String())
	}
	if got := field(t, loggedOut, "message"); got != handler.MessageLoggedOut {
		t.Errorf("logout message = %q, want %q", got, handler.MessageLoggedOut)
	}
	afterLogout := call(t, router, http.MethodGet, "/api/me", "", token)
	if afterLogout.Code != http.StatusUnauthorized {
		t.Fatalf("me after logout status = %d, want 401", afterLogout.Code)
	}
	if got := field(t, afterLogout, "error"); got != middleware.MessageSessionExpired {
		t.Errorf("me after logout error = %q, want %q", got, middleware.MessageSessionExpired)
	}
	mismatch := call(t, router, http.MethodPost, "/api/auth/forgot-password",
		`{"identifier":"2023331500","phone":"01799999999"}`, "")
	if mismatch.Code != http.StatusBadRequest {
		t.Fatalf("forgot password with a wrong phone status = %d, want 400", mismatch.Code)
	}
	if got := field(t, mismatch, "error"); got != handler.MessageResetMismatch {
		t.Errorf("forgot password error = %q, want %q", got, handler.MessageResetMismatch)
	}
	forgot := call(t, router, http.MethodPost, "/api/auth/forgot-password",
		`{"identifier":"2023331500","phone":"+8801712345678"}`, "")
	if forgot.Code != http.StatusOK {
		t.Fatalf("forgot password status = %d, want 200 (body = %q)", forgot.Code, forgot.Body.String())
	}
	if got := field(t, forgot, "email_hint"); got != "f***@student.sust.edu" {
		t.Errorf("email hint = %q, want %q", got, "f***@student.sust.edu")
	}
	code := field(t, forgot, "dev_code")
	if len(code) != 6 {
		t.Fatalf("dev code has %d characters, want 6", len(code))
	}
	var storedCode, storedStage string
	readReset := "SELECT reset_token, reset_stage FROM users WHERE identifier = $1"
	if err := pool.QueryRow(ctx, readReset, "2023331500").Scan(&storedCode, &storedStage); err != nil {
		t.Fatalf("could not read back the reset code: %v", err)
	}
	if storedCode == code {
		t.Error("the database stores the raw reset code, want only its hash")
	}
	if storedStage != store.ResetStageCode {
		t.Errorf("stored stage = %q, want %q", storedStage, store.ResetStageCode)
	}
	tooSoon := call(t, router, http.MethodPost, "/api/auth/forgot-password",
		`{"identifier":"2023331500","phone":"01712345678"}`, "")
	if tooSoon.Code != http.StatusTooManyRequests {
		t.Fatalf("second forgot password status = %d, want 429", tooSoon.Code)
	}
	if got := field(t, tooSoon, "error"); got != handler.MessageResetCooldown {
		t.Errorf("cooldown error = %q, want %q", got, handler.MessageResetCooldown)
	}
	wrongCode := call(t, router, http.MethodPost, "/api/auth/verify-reset-code",
		`{"identifier":"2023331500","code":"000000"}`, "")
	if wrongCode.Code != http.StatusBadRequest {
		t.Fatalf("verify with a wrong code status = %d, want 400", wrongCode.Code)
	}
	if got := field(t, wrongCode, "error"); got != handler.MessageInvalidResetCode {
		t.Errorf("verify error = %q, want %q", got, handler.MessageInvalidResetCode)
	}
	verified := call(t, router, http.MethodPost, "/api/auth/verify-reset-code",
		`{"identifier":"2023331500","code":"`+code+`"}`, "")
	if verified.Code != http.StatusOK {
		t.Fatalf("verify status = %d, want 200 (body = %q)", verified.Code, verified.Body.String())
	}
	resetToken := field(t, verified, "reset_token")
	if len(resetToken) != 64 {
		t.Fatalf("reset token has %d characters, want 64", len(resetToken))
	}
	if err := pool.QueryRow(ctx, readReset, "2023331500").Scan(&storedCode, &storedStage); err != nil {
		t.Fatalf("could not read back the reset token: %v", err)
	}
	if storedCode == resetToken {
		t.Error("the database stores the raw reset token, want only its hash")
	}
	if storedStage != store.ResetStageVerified {
		t.Errorf("stored stage = %q, want %q", storedStage, store.ResetStageVerified)
	}
	wrongToken := `{"identifier":"2023331500","reset_token":"` + strings.Repeat("0", 64) +
		`","new_password":"second-password","confirm_password":"second-password"}`
	rejected := call(t, router, http.MethodPost, "/api/auth/reset-password", wrongToken, "")
	if rejected.Code != http.StatusBadRequest {
		t.Fatalf("reset with a wrong token status = %d, want 400", rejected.Code)
	}
	if got := field(t, rejected, "error"); got != handler.MessageInvalidResetToken {
		t.Errorf("reset error = %q, want %q", got, handler.MessageInvalidResetToken)
	}
	resetBody := `{"identifier":"2023331500","reset_token":"` + resetToken +
		`","new_password":"second-password","confirm_password":"second-password"}`
	reset := call(t, router, http.MethodPost, "/api/auth/reset-password", resetBody, "")
	if reset.Code != http.StatusOK {
		t.Fatalf("reset status = %d, want 200 (body = %q)", reset.Code, reset.Body.String())
	}
	if got := field(t, reset, "message"); got != handler.MessagePasswordUpdated {
		t.Errorf("reset message = %q, want %q", got, handler.MessagePasswordUpdated)
	}
	oldPassword := call(t, router, http.MethodPost, "/api/auth/login", loginBody, "")
	if oldPassword.Code != http.StatusUnauthorized {
		t.Fatalf("login with the old password status = %d, want 401", oldPassword.Code)
	}
	if got := field(t, oldPassword, "error"); got != handler.MessageInvalidLogin {
		t.Errorf("login error = %q, want %q", got, handler.MessageInvalidLogin)
	}
	newLogin := call(t, router, http.MethodPost, "/api/auth/login",
		`{"identifier":"2023331500","password":"second-password"}`, "")
	if newLogin.Code != http.StatusOK {
		t.Fatalf("login with the new password status = %d, want 200 (body = %q)", newLogin.Code, newLogin.Body.String())
	}
	if field(t, newLogin, "token") == "" {
		t.Error("the second login returned an empty token")
	}
	reusedReset := call(t, router, http.MethodPost, "/api/auth/reset-password", resetBody, "")
	if reusedReset.Code != http.StatusBadRequest {
		t.Errorf("reusing the reset token status = %d, want 400", reusedReset.Code)
	}
}

func TestPasswordResetStopsAfterFiveWrongCodes(t *testing.T) {
	ctx, pool, router := newFlowRouter(t)
	registered := call(t, router, http.MethodPost, "/api/auth/register", flowRegistration, "")
	if registered.Code != http.StatusCreated {
		t.Fatalf("register status = %d, want 201 (body = %q)", registered.Code, registered.Body.String())
	}
	if _, err := pool.Exec(ctx, "UPDATE users SET status = 'active' WHERE identifier = $1", "2023331500"); err != nil {
		t.Fatalf("could not activate the account: %v", err)
	}
	forgot := call(t, router, http.MethodPost, "/api/auth/forgot-password",
		`{"identifier":"2023331500","phone":"01712345678"}`, "")
	if forgot.Code != http.StatusOK {
		t.Fatalf("forgot password status = %d, want 200 (body = %q)", forgot.Code, forgot.Body.String())
	}
	code := field(t, forgot, "dev_code")
	for attempt := 1; attempt <= 5; attempt++ {
		rec := call(t, router, http.MethodPost, "/api/auth/verify-reset-code",
			`{"identifier":"2023331500","code":"000000"}`, "")
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("attempt %d status = %d, want 400", attempt, rec.Code)
		}
		want := handler.MessageInvalidResetCode
		if attempt == 5 {
			want = handler.MessageTooManyResetCodes
		}
		if got := field(t, rec, "error"); got != want {
			t.Errorf("attempt %d error = %q, want %q", attempt, got, want)
		}
	}
	var stage, token *string
	query := "SELECT reset_stage, reset_token FROM users WHERE identifier = $1"
	if err := pool.QueryRow(ctx, query, "2023331500").Scan(&stage, &token); err != nil {
		t.Fatalf("could not read back the reset fields: %v", err)
	}
	if stage != nil || token != nil {
		t.Error("the reset fields survived five wrong codes, want them cleared")
	}
	afterLockout := call(t, router, http.MethodPost, "/api/auth/verify-reset-code",
		`{"identifier":"2023331500","code":"`+code+`"}`, "")
	if afterLockout.Code != http.StatusBadRequest {
		t.Fatalf("verify after the lockout status = %d, want 400", afterLockout.Code)
	}
	if got := field(t, afterLockout, "error"); got != handler.MessageInvalidResetCode {
		t.Errorf("verify after the lockout error = %q, want %q", got, handler.MessageInvalidResetCode)
	}
}

func TestAdminPasswordResetFlow(t *testing.T) {
	ctx, pool, router := newFlowRouter(t)
	hash, err := auth.HashPassword("first-admin-password")
	if err != nil {
		t.Fatalf("HashPassword() returned an unexpected error: %v", err)
	}
	const adminID = "flow-admin"
	if _, err := store.New(pool).EnsureAdmin(ctx, adminID, hash, "office@sust.edu", "01799999999"); err != nil {
		t.Fatalf("EnsureAdmin() returned an unexpected error: %v", err)
	}
	forgot := call(t, router, http.MethodPost, "/api/auth/forgot-password",
		`{"identifier":"`+adminID+`","phone":"01799999999"}`, "")
	if forgot.Code != http.StatusOK {
		t.Fatalf("admin forgot password status = %d, want 200 (body = %q)", forgot.Code, forgot.Body.String())
	}
	if got := field(t, forgot, "email_hint"); got != "o***@sust.edu" {
		t.Errorf("email hint = %q, want %q", got, "o***@sust.edu")
	}
	verified := call(t, router, http.MethodPost, "/api/auth/verify-reset-code",
		`{"identifier":"`+adminID+`","code":"`+field(t, forgot, "dev_code")+`"}`, "")
	if verified.Code != http.StatusOK {
		t.Fatalf("admin verify status = %d, want 200 (body = %q)", verified.Code, verified.Body.String())
	}
	resetBody := `{"identifier":"` + adminID + `","reset_token":"` + field(t, verified, "reset_token") +
		`","new_password":"second-admin-password","confirm_password":"second-admin-password"}`
	reset := call(t, router, http.MethodPost, "/api/auth/reset-password", resetBody, "")
	if reset.Code != http.StatusOK {
		t.Fatalf("admin reset status = %d, want 200 (body = %q)", reset.Code, reset.Body.String())
	}
	oldPassword := call(t, router, http.MethodPost, "/api/auth/admin/login",
		`{"identifier":"`+adminID+`","password":"first-admin-password"}`, "")
	if oldPassword.Code != http.StatusUnauthorized {
		t.Fatalf("admin login with the old password status = %d, want 401", oldPassword.Code)
	}
	newLogin := call(t, router, http.MethodPost, "/api/auth/admin/login",
		`{"identifier":"`+adminID+`","password":"second-admin-password"}`, "")
	if newLogin.Code != http.StatusOK {
		t.Fatalf("admin login with the new password status = %d, want 200 (body = %q)", newLogin.Code, newLogin.Body.String())
	}
	if field(t, newLogin, "token") == "" {
		t.Error("the admin login returned an empty token")
	}
}

func TestLoginRateLimitOnTheRealRouter(t *testing.T) {
	_, _, router := newFlowRouter(t)
	body := `{"identifier":"2023339999","password":"whatever-password"}`
	for i := 1; i <= middleware.DefaultRateLimit; i++ {
		rec := call(t, router, http.MethodPost, "/api/auth/login", body, "")
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("attempt %d status = %d, want 401", i, rec.Code)
		}
	}
	blocked := call(t, router, http.MethodPost, "/api/auth/login", body, "")
	if blocked.Code != http.StatusTooManyRequests {
		t.Fatalf("attempt %d status = %d, want 429", middleware.DefaultRateLimit+1, blocked.Code)
	}
	if got := field(t, blocked, "error"); got != middleware.MessageTooManyAttempts {
		t.Errorf("error = %q, want %q", got, middleware.MessageTooManyAttempts)
	}
	health := call(t, router, http.MethodGet, "/api/health", "", "")
	if health.Code != http.StatusOK {
		t.Errorf("health status = %d, want 200 because only the auth routes are throttled", health.Code)
	}
}

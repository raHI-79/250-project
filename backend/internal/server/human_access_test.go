package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/raHI-79/250-project/backend/internal/auth"
	"github.com/raHI-79/250-project/backend/internal/handler"
	"github.com/raHI-79/250-project/backend/internal/store"
)

const inviteBody = `{"token":"invite-token","username":"office.rakib",` +
	`"password":"office2026x","confirm_password":"office2026x"}`

type inviteStore struct {
	activated string
}

func (i *inviteStore) FindAdminInvite(ctx context.Context, inviteHash string) (store.AdminInvite, error) {
	return store.AdminInvite{UserID: 77, Email: "office@sust.edu"}, nil
}

func (i *inviteStore) ActivateAdmin(ctx context.Context, userID int64, chosenID,
	passwordHash string) error {
	i.activated = chosenID
	return nil
}

func (i *inviteStore) TouchSignIn(ctx context.Context, userID int64) error {
	return nil
}

func (i *inviteStore) CreateSession(ctx context.Context, session store.NewSession) (int64, error) {
	return 1, nil
}

func (i *inviteStore) RevokeSession(ctx context.Context, userID, sessionID int64) error {
	return nil
}

func (i *inviteStore) StartTwoStep(ctx context.Context, userID int64, codeHash string,
	expiresAt, sentAt time.Time) error {
	return nil
}

func (i *inviteStore) TwoStepChallenge(ctx context.Context,
	identifier string) (store.TwoStepChallenge, error) {
	return store.TwoStepChallenge{}, store.ErrNotFound
}

func (i *inviteStore) CountTwoStepAttempt(ctx context.Context, userID int64) (int, error) {
	return 0, nil
}

func (i *inviteStore) ClearTwoStep(ctx context.Context, userID int64) error {
	return nil
}

func (i *inviteStore) SetPermissions(ctx context.Context, userID int64, names []string,
	grantedBy int64) error {
	return nil
}

func newHumanRouter(t *testing.T) (Dependencies, http.Handler, *inviteStore) {
	t.Helper()
	deps, _, _ := newControlRouter(t)
	invites := &inviteStore{}
	deps.Invites = invites
	deps.Config.TurnstileSiteKey = "site-key"
	deps.Config.TurnstileSecretKey = "secret-key"
	passes, err := auth.NewHumanPasses(deps.Config.JWTSecret)
	if err != nil {
		t.Fatalf("auth.NewHumanPasses() returned an unexpected error: %v", err)
	}
	deps.Passes = passes
	return deps, NewRouter(deps), invites
}

func humanPass(t *testing.T, deps Dependencies) string {
	t.Helper()
	passes, err := auth.NewHumanPasses(deps.Config.JWTSecret)
	if err != nil {
		t.Fatalf("auth.NewHumanPasses() returned an unexpected error: %v", err)
	}
	pass, _, err := passes.Issue(auth.HumanPassBrowser)
	if err != nil {
		t.Fatalf("Issue() returned an unexpected error: %v", err)
	}
	return pass
}

func postInvite(t *testing.T, router http.Handler, pass, client string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodPost, "/api/auth/admin-invite/accept",
		strings.NewReader(inviteBody))
	request.Header.Set("Content-Type", "application/json")
	if pass != "" {
		request.Header.Set(handler.HumanPassHeader, pass)
	}
	if client != "" {
		request.Header.Set("X-Client", client)
	}
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	return recorder
}

func TestInviteAcceptanceNeedsTheHumanPass(t *testing.T) {
	deps, router, _ := newHumanRouter(t)
	rec := postInvite(t, router, "", "")
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status without a pass = %d, want 403 (body = %q)", rec.Code, rec.Body.String())
	}
	if got := errorField(t, rec); got != handler.MessageHumanCheckNeeded {
		t.Errorf("error = %q, want %q", got, handler.MessageHumanCheckNeeded)
	}
	_ = deps
}

func TestInviteAcceptanceTakesAValidHumanPass(t *testing.T) {
	deps, router, invites := newHumanRouter(t)
	rec := postInvite(t, router, humanPass(t, deps), "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status with a pass = %d, want 200 (body = %q)", rec.Code, rec.Body.String())
	}
	if invites.activated != "office.rakib" {
		t.Errorf("activated = %q, want the chosen username", invites.activated)
	}
}

func TestTheAppNeedsNoHumanPass(t *testing.T) {
	_, router, _ := newHumanRouter(t)
	rec := postInvite(t, router, "", "android")
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403 because the app may not accept office invites", rec.Code)
	}
	if got := errorField(t, rec); got == handler.MessageHumanCheckNeeded {
		t.Error("the app was asked for a human pass, want the office refusal instead")
	}
}

func TestTheHumanCheckIsSkippedWithoutASiteKey(t *testing.T) {
	deps, _, invites := newHumanRouter(t)
	deps.Config.TurnstileSiteKey = ""
	deps.Config.TurnstileSecretKey = ""
	router := NewRouter(deps)
	rec := postInvite(t, router, "", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 when no site key is set (body = %q)", rec.Code, rec.Body.String())
	}
	if invites.activated != "office.rakib" {
		t.Errorf("activated = %q, want the chosen username", invites.activated)
	}
}

func TestTheConfigCarriesTheSiteKey(t *testing.T) {
	_, router, _ := newHumanRouter(t)
	rec := serve(t, router, http.MethodGet, "/api/auth/config", "", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `"human_check_enabled":true`) {
		t.Errorf("config = %s, want the human check reported on", body)
	}
	if !strings.Contains(body, `"turnstile_site_key":"site-key"`) {
		t.Errorf("config = %s, want the site key", body)
	}
}

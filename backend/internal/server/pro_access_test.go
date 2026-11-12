package server

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/raHI-79/250-project/backend/internal/auth"
	"github.com/raHI-79/250-project/backend/internal/handler"
	"github.com/raHI-79/250-project/backend/internal/settings"
	"github.com/raHI-79/250-project/backend/internal/store"
)

type memorySettings struct {
	values map[string]string
}

func (m *memorySettings) AllSettings(ctx context.Context) (map[string]string, error) {
	copied := make(map[string]string, len(m.values))
	for key, value := range m.values {
		copied[key] = value
	}
	return copied, nil
}

func (m *memorySettings) SaveSettings(ctx context.Context, values map[string]string, by int64) error {
	for key, value := range values {
		m.values[key] = value
	}
	return nil
}

type memoryBlocks struct {
	ranges []string
}

func (m *memoryBlocks) ActiveIPBlocks(ctx context.Context) ([]string, error) {
	return m.ranges, nil
}

func newSiteRouter(t *testing.T, values map[string]string) (Dependencies, http.Handler) {
	t.Helper()
	deps, _, _ := newControlRouter(t)
	keys := &memorySettings{values: values}
	deps.Keys = keys
	deps.Values = settings.New(keys, SettingDefaults(deps.Config))
	return deps, NewRouter(deps)
}

func TestMaintenanceModeClosesTheStudentPages(t *testing.T) {
	deps, router := newSiteRouter(t, map[string]string{
		settings.KeyMaintenance:        settings.On,
		settings.KeyMaintenanceMessage: "Back at noon",
	})
	tokens := controlTokens(t, deps)
	rec := serve(t, router, http.MethodGet, "/api/students/today", "", tokens[auth.RoleStudent])
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503 (body = %q)", rec.Code, rec.Body.String())
	}
	if got := errorField(t, rec); got != "Back at noon" {
		t.Errorf("error = %q, want the maintenance message", got)
	}
	office := serve(t, router, http.MethodGet, "/api/admin/students", "", tokens[auth.RoleAdmin])
	if office.Code != http.StatusOK {
		t.Errorf("the office was closed out during maintenance: %d", office.Code)
	}
	chief := serve(t, router, http.MethodGet, "/api/super-admin/admins", "",
		tokens[auth.RoleSuperAdmin])
	if chief.Code != http.StatusOK {
		t.Errorf("the control room was closed out during maintenance: %d", chief.Code)
	}
	health := serve(t, router, http.MethodGet, "/api/health", "", "")
	if health.Code != http.StatusOK {
		t.Errorf("the health check answered %d during maintenance", health.Code)
	}
}

func TestTheConfigTellsTheSiteState(t *testing.T) {
	_, router := newSiteRouter(t, map[string]string{
		settings.KeyMaintenance:      settings.Off,
		settings.KeyRegistration:     settings.Off,
		settings.KeyRegistrationNote: "Closed until July",
		settings.KeyPolls:            settings.Off,
		settings.KeyMinAppVersion:    "2.6.0",
	})
	rec := serve(t, router, http.MethodGet, "/api/auth/config", "", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body = %q)", rec.Code, rec.Body.String())
	}
	var body struct {
		Maintenance         bool            `json:"maintenance"`
		RegistrationOpen    bool            `json:"registration_open"`
		RegistrationMessage string          `json:"registration_message"`
		MinAppVersion       string          `json:"min_app_version"`
		Features            map[string]bool `json:"features"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("body is not valid JSON: %v", err)
	}
	if body.Maintenance {
		t.Error("the config reports maintenance although it is off")
	}
	if body.RegistrationOpen || body.RegistrationMessage != "Closed until July" {
		t.Errorf("registration = %v %q, want closed with the message", body.RegistrationOpen,
			body.RegistrationMessage)
	}
	if body.MinAppVersion != "2.6.0" {
		t.Errorf("minimum app version = %q, want 2.6.0", body.MinAppVersion)
	}
	if body.Features["polls"] {
		t.Error("polls report themselves on although they are switched off")
	}
	if !body.Features["messages"] {
		t.Error("messages report themselves off although they are on")
	}
}

func TestSwitchedOffFeaturesRefuseTheirRoutes(t *testing.T) {
	deps, router := newSiteRouter(t, map[string]string{
		settings.KeyPolls:      settings.Off,
		settings.KeyAttendance: settings.Off,
		settings.KeyMessages:   settings.On,
	})
	tokens := controlTokens(t, deps)
	closed := []struct {
		path  string
		token string
	}{
		{path: "/api/students/polls", token: tokens[auth.RoleStudent]},
		{path: "/api/students/attendance/open", token: tokens[auth.RoleStudent]},
		{path: "/api/teacher/attendance/options", token: tokens[auth.RoleTeacher]},
	}
	for _, entry := range closed {
		rec := serve(t, router, http.MethodGet, entry.path, "", entry.token)
		if rec.Code != http.StatusForbidden || !strings.Contains(rec.Body.String(), `"code":"feature_off"`) {
			t.Errorf("%s answered %d, want 403 feature_off (body = %q)", entry.path, rec.Code, rec.Body.String())
		}
	}
	open := serve(t, router, http.MethodGet, "/api/messages/conversations", "",
		tokens[auth.RoleStudent])
	if open.Code == http.StatusServiceUnavailable {
		t.Error("messages were refused although they are on")
	}
}

func TestRegistrationClosedRefusesTheForm(t *testing.T) {
	_, router := newSiteRouter(t, map[string]string{
		settings.KeyRegistration:     settings.Off,
		settings.KeyRegistrationNote: "Closed until July",
	})
	rec := serve(t, router, http.MethodPost, "/api/auth/register", `{}`, "")
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503 (body = %q)", rec.Code, rec.Body.String())
	}
	if got := errorField(t, rec); got != "Closed until July" {
		t.Errorf("error = %q, want the closed message", got)
	}
}

func TestABlockedAddressIsRefused(t *testing.T) {
	deps, _ := newSiteRouter(t, map[string]string{})
	deps.Walls = &memoryBlocks{ranges: []string{"192.0.2.0/24"}}
	router := NewRouter(deps)
	rec := serve(t, router, http.MethodGet, "/api/health", "", "")
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403 for a blocked address (body = %q)", rec.Code, rec.Body.String())
	}
	if got := errorField(t, rec); got != handler.MessageBlocked {
		t.Errorf("error = %q, want %q", got, handler.MessageBlocked)
	}
	deps.Walls = &memoryBlocks{ranges: []string{"203.0.113.0/24"}}
	router = NewRouter(deps)
	rec = serve(t, router, http.MethodGet, "/api/health", "", "")
	if rec.Code != http.StatusOK {
		t.Errorf("an address outside the block answered %d", rec.Code)
	}
}

func TestARevokedSessionCannotBeUsed(t *testing.T) {
	deps, _ := newSiteRouter(t, map[string]string{})
	deps.Live = revokedSessions{}
	router := NewRouter(deps)
	token := issueWithSession(t, deps, chiefID, auth.RoleSuperAdmin, 9)
	rec := serve(t, router, http.MethodGet, "/api/super-admin/admins", "", token)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401 for a revoked session (body = %q)", rec.Code, rec.Body.String())
	}
}

type revokedSessions struct{}

func (revokedSessions) LoadSession(ctx context.Context, sessionID int64) (store.Session, error) {
	revoked := time.Now().Add(-time.Minute)
	return store.Session{
		ID:         sessionID,
		UserID:     chiefID,
		LastSeenAt: time.Now(),
		ExpiresAt:  time.Now().Add(time.Hour),
		RevokedAt:  &revoked,
	}, nil
}

func (revokedSessions) TouchSession(ctx context.Context, sessionID int64, ipHash,
	country string) error {
	return nil
}

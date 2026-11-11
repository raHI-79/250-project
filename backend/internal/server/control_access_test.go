package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/raHI-79/250-project/backend/internal/auth"
	"github.com/raHI-79/250-project/backend/internal/handler"
	"github.com/raHI-79/250-project/backend/internal/middleware"
	"github.com/raHI-79/250-project/backend/internal/store"
)

const (
	chiefID        = int64(201)
	officeTeacher  = int64(202)
	controlSecret  = "backup-secret"
	controlBeacon  = `{"visitor_id":"v1","items":[{"path":"/student/today"}]}`
	controlConfirm = `{"confirm":"office.one"}`
)

type controlUserStore struct{}

func (controlUserStore) GetAuthUser(ctx context.Context, userID int64) (store.AuthUser, error) {
	switch userID {
	case chiefID:
		return store.AuthUser{
			ID: chiefID, Identifier: "chief", Role: auth.RoleSuperAdmin, Status: store.StatusActive,
		}, nil
	case officeTeacher:
		return store.AuthUser{
			ID: officeTeacher, Identifier: "T-401", Role: auth.RoleTeacher,
			Status: store.StatusActive, OfficeAccess: true,
		}, nil
	default:
		return accessUserStore{}.GetAuthUser(ctx, userID)
	}
}

type stubOfficeStore struct{}

func (stubOfficeStore) ListOfficeAdmins(ctx context.Context) ([]store.OfficeAdmin, error) {
	return []store.OfficeAdmin{}, nil
}

func (stubOfficeStore) CreateAdminInvite(ctx context.Context, email, inviteHash string,
	lifetime time.Duration, invitedBy int64) (int64, error) {
	return 7, nil
}

func (stubOfficeStore) ResendAdminInvite(ctx context.Context, userID int64, inviteHash string,
	lifetime time.Duration) (string, error) {
	return "office@sust.edu", nil
}

func (stubOfficeStore) CancelAdminInvite(ctx context.Context, userID int64) (string, error) {
	return "office@sust.edu", nil
}

func (stubOfficeStore) SuspendAdmin(ctx context.Context, userID int64) (string, error) {
	return "office.one", nil
}

func (stubOfficeStore) RestoreAdmin(ctx context.Context, userID int64) (string, error) {
	return "office.one", nil
}

func (stubOfficeStore) RemoveAdmin(ctx context.Context, userID int64) (string, error) {
	return "office.one", nil
}

func (stubOfficeStore) OfficeAccount(ctx context.Context, userID int64) (string, string, string, error) {
	return "office.one", auth.RoleAdmin, store.StatusActive, nil
}

func (stubOfficeStore) SetTeacherOfficeAccess(ctx context.Context, userID int64,
	granted bool) (string, error) {
	return "Ratul Islam", nil
}

func (stubOfficeStore) ListOfficeTeachers(ctx context.Context, search string,
	limit int) ([]store.OfficeTeacher, error) {
	return []store.OfficeTeacher{}, nil
}

func (stubOfficeStore) Permissions(ctx context.Context, userID int64) ([]string, error) {
	return auth.DefaultPermissions(), nil
}

func (stubOfficeStore) SetPermissions(ctx context.Context, userID int64, names []string,
	grantedBy int64) error {
	return nil
}

func (stubOfficeStore) AllPermissions(ctx context.Context) (map[int64][]string, error) {
	return map[int64][]string{}, nil
}

func (s *stubViewStore) LoadSession(ctx context.Context, sessionID int64) (store.Session, error) {
	return store.Session{}, store.ErrNotFound
}

func (s *stubViewStore) TouchSession(ctx context.Context, sessionID int64, ipHash,
	country string) error {
	return nil
}

type stubTrailStore struct{}

func (stubTrailStore) ListEvents(ctx context.Context, filter store.AuditFilter) ([]store.AuditEntry, error) {
	return []store.AuditEntry{}, nil
}

func (stubTrailStore) CountActions(ctx context.Context, actorID int64,
	from, to *time.Time) ([]store.ActionCount, error) {
	return []store.ActionCount{}, nil
}

func (stubTrailStore) ListAuditPeople(ctx context.Context) ([]store.AuditPerson, error) {
	return []store.AuditPerson{}, nil
}

type stubViewStore struct{ backups int }

func (s *stubViewStore) RecordViews(ctx context.Context, views []store.PageView) error {
	return nil
}

func (s *stubViewStore) SetBackupStatus(ctx context.Context, at time.Time, note string) error {
	s.backups++
	return nil
}

func (s *stubViewStore) WriteEvent(ctx context.Context, event store.AuditEvent) error {
	return nil
}

type stubFigureStore struct{}

func (stubFigureStore) RequestVolume(ctx context.Context,
	window store.Window) (store.RequestVolume, error) {
	return store.RequestVolume{}, nil
}

func (stubFigureStore) PushVolume(ctx context.Context, window store.Window) (store.PushVolume, error) {
	return store.PushVolume{}, nil
}

func (stubFigureStore) PushBreakdown(ctx context.Context,
	window store.Window) ([]store.PushBreakdown, error) {
	return nil, nil
}

func (stubFigureStore) FailingPaths(ctx context.Context, window store.Window,
	limit int) ([]store.FailingPath, error) {
	return nil, nil
}

func (stubFigureStore) QuietAlerts(ctx context.Context) (map[string]time.Time, error) {
	return nil, nil
}

func (stubFigureStore) DismissAlert(ctx context.Context, key string, by int64,
	quiet time.Duration) error {
	return nil
}

func (stubFigureStore) OverviewTotals(ctx context.Context, window store.Window,
	dayStart time.Time) (store.OverviewTotals, error) {
	return store.OverviewTotals{}, nil
}

func (stubFigureStore) VisitorSeries(ctx context.Context, window store.Window) ([]store.DayPoint, error) {
	return []store.DayPoint{}, nil
}

func (stubFigureStore) SignInsByRole(ctx context.Context, window store.Window) ([]store.Slice, error) {
	return []store.Slice{}, nil
}

func (stubFigureStore) ClientSplit(ctx context.Context, window store.Window) ([]store.Slice, error) {
	return []store.Slice{}, nil
}

func (stubFigureStore) AppVersions(ctx context.Context, window store.Window) ([]store.Slice, error) {
	return []store.Slice{}, nil
}

func (stubFigureStore) Devices(ctx context.Context, window store.Window) ([]store.Slice, error) {
	return []store.Slice{}, nil
}

func (stubFigureStore) TopPages(ctx context.Context, window store.Window) ([]store.PagePoint, error) {
	return []store.PagePoint{}, nil
}

func (stubFigureStore) BatchAttendance(ctx context.Context,
	window store.Window) ([]store.BatchAttendance, error) {
	return []store.BatchAttendance{}, nil
}

func (stubFigureStore) CourseClasses(ctx context.Context,
	window store.Window) ([]store.CourseClasses, error) {
	return []store.CourseClasses{}, nil
}

func (stubFigureStore) UndecidedTeachers(ctx context.Context,
	day time.Time) ([]store.TeacherPending, error) {
	return []store.TeacherPending{}, nil
}

func (stubFigureStore) AcademicTotals(ctx context.Context, window store.Window,
	tomorrow time.Time) (store.AcademicTotals, error) {
	return store.AcademicTotals{}, nil
}

func (stubFigureStore) SystemTotals(ctx context.Context, window store.Window) (store.SystemTotals, error) {
	return store.SystemTotals{}, nil
}

func (stubFigureStore) LatestServerErrors(ctx context.Context) ([]store.ErrorPath, error) {
	return []store.ErrorPath{}, nil
}

func (stubFigureStore) FailedSignInSpike(ctx context.Context, since time.Time) (int, error) {
	return 0, nil
}

func (stubFigureStore) SheetDeletions(ctx context.Context, since time.Time) (int, error) {
	return 0, nil
}

func (stubFigureStore) BackupStatus(ctx context.Context) (store.BackupState, error) {
	return store.BackupState{}, nil
}

func newControlRouter(t *testing.T) (Dependencies, http.Handler, *stubViewStore) {
	t.Helper()
	deps, _ := newAccessRouter(t)
	views := &stubViewStore{}
	deps.Users = controlUserStore{}
	deps.Office = stubOfficeStore{}
	deps.Trail = stubTrailStore{}
	deps.Views = views
	deps.Figures = stubFigureStore{}
	deps.Events = views
	deps.Config.BackupSecret = controlSecret
	deps.RateLimit = 1000
	return deps, NewRouter(deps), views
}

func controlTokens(t *testing.T, deps Dependencies) map[string]string {
	t.Helper()
	tokens := accessTokens(t, deps)
	tokens[auth.RoleSuperAdmin] = issue(t, deps, chiefID, auth.RoleSuperAdmin)
	return tokens
}

func TestSuperAdminRoutesNeedTheControlRoomToken(t *testing.T) {
	deps, router, _ := newControlRouter(t)
	tokens := controlTokens(t, deps)
	routes := []struct {
		method string
		path   string
		body   string
	}{
		{method: http.MethodGet, path: "/api/super-admin/admins"},
		{method: http.MethodGet, path: "/api/super-admin/teachers"},
		{method: http.MethodGet, path: "/api/super-admin/dashboard"},
		{method: http.MethodGet, path: "/api/super-admin/activity/104"},
		{method: http.MethodPost, path: "/api/super-admin/admins/invite", body: `{"email":"new@sust.edu"}`},
		{method: http.MethodPost, path: "/api/super-admin/teachers/103/office-access"},
	}
	for _, route := range routes {
		t.Run("no token on "+route.path, func(t *testing.T) {
			rec := serve(t, router, route.method, route.path, route.body, "")
			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, want 401 (body = %q)", rec.Code, rec.Body.String())
			}
		})
		for _, role := range []string{auth.RoleStudent, auth.RoleCR, auth.RoleTeacher, auth.RoleAdmin} {
			t.Run(role+" on "+route.path, func(t *testing.T) {
				rec := serve(t, router, route.method, route.path, route.body, tokens[role])
				if rec.Code != http.StatusForbidden {
					t.Fatalf("status = %d, want 403 (body = %q)", rec.Code, rec.Body.String())
				}
				if got := errorField(t, rec); got != middleware.MessageAccessDenied {
					t.Errorf("error = %q, want %q", got, middleware.MessageAccessDenied)
				}
			})
		}
		t.Run("super admin on "+route.path, func(t *testing.T) {
			rec := serve(t, router, route.method, route.path, route.body, tokens[auth.RoleSuperAdmin])
			if route.method == http.MethodGet {
				if rec.Code >= http.StatusBadRequest {
					t.Fatalf("status = %d, want a success (body = %q)", rec.Code, rec.Body.String())
				}
				return
			}
			if rec.Code != http.StatusPreconditionRequired {
				t.Fatalf("status = %d, want 428 without a password confirmation (body = %q)",
					rec.Code, rec.Body.String())
			}
			if got := errorField(t, rec); got != handler.MessageConfirmNeeded {
				t.Errorf("error = %q, want %q", got, handler.MessageConfirmNeeded)
			}
		})
	}
}

func TestTheControlRoomOpensTheOfficePagesReadOnly(t *testing.T) {
	deps, router, _ := newControlRouter(t)
	tokens := controlTokens(t, deps)
	rec := serve(t, router, http.MethodGet, "/api/admin/students", "", tokens[auth.RoleSuperAdmin])
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 on a read (body = %q)", rec.Code, rec.Body.String())
	}
	writes := []struct {
		method string
		path   string
		body   string
	}{
		{method: http.MethodPost, path: "/api/admin/crs/promote", body: `{"reg_no":"2023331001"}`},
		{method: http.MethodDelete, path: "/api/admin/courses/1"},
		{method: http.MethodPatch, path: "/api/admin/teachers/103", body: `{"name":"New"}`},
	}
	for _, write := range writes {
		t.Run(write.method+" "+write.path, func(t *testing.T) {
			rec := serve(t, router, write.method, write.path, write.body, tokens[auth.RoleSuperAdmin])
			if rec.Code != http.StatusForbidden {
				t.Fatalf("status = %d, want 403 (body = %q)", rec.Code, rec.Body.String())
			}
			if got := errorField(t, rec); got != middleware.MessageReadOnly {
				t.Errorf("error = %q, want %q", got, middleware.MessageReadOnly)
			}
		})
	}
}

func TestTheOfficeAdminStillWritesOnTheOfficePages(t *testing.T) {
	deps, router, _ := newControlRouter(t)
	tokens := controlTokens(t, deps)
	rec := serve(t, router, http.MethodDelete, "/api/admin/courses/1", "", tokens[auth.RoleAdmin])
	if rec.Code == http.StatusForbidden {
		t.Fatalf("the office admin was refused a write (body = %q)", rec.Body.String())
	}
}

func TestATeacherWithOfficeAccessUsesTheOfficePages(t *testing.T) {
	deps, router, _ := newControlRouter(t)
	token := issue(t, deps, officeTeacher, auth.RoleAdmin)
	rec := serve(t, router, http.MethodGet, "/api/admin/students", "", token)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body = %q)", rec.Code, rec.Body.String())
	}
	teacherOnly := issue(t, deps, accessTeacherID, auth.RoleAdmin)
	rec = serve(t, router, http.MethodGet, "/api/admin/students", "", teacherOnly)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401 for a teacher without office access (body = %q)",
			rec.Code, rec.Body.String())
	}
}

func TestTheAppCannotReachTheControlRoom(t *testing.T) {
	deps, router, _ := newControlRouter(t)
	tokens := controlTokens(t, deps)
	rec := serveAsApp(t, router, http.MethodGet, "/api/super-admin/admins", tokens[auth.RoleSuperAdmin])
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403 (body = %q)", rec.Code, rec.Body.String())
	}
	if got := errorField(t, rec); got != middleware.MessageAppForbidden {
		t.Errorf("error = %q, want %q", got, middleware.MessageAppForbidden)
	}
	login := httptest.NewRequest(http.MethodPost, "/api/auth/control-room/login", nil)
	login.Header.Set(middleware.ClientHeader, middleware.ClientAndroid)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, login)
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("control room login from the app = %d, want 403 (body = %q)",
			recorder.Code, recorder.Body.String())
	}
}

func TestTheBackupStatusEndpointNeedsItsSecret(t *testing.T) {
	deps, router, views := newControlRouter(t)
	_ = deps
	request := httptest.NewRequest(http.MethodPost, "/api/system/backup-status", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status without the secret = %d, want 403", recorder.Code)
	}
	if got := errorField(t, recorder); got != handler.MessageBackupDenied {
		t.Errorf("error = %q, want %q", got, handler.MessageBackupDenied)
	}
	wrong := httptest.NewRequest(http.MethodPost, "/api/system/backup-status", nil)
	wrong.Header.Set(handler.BackupSecretHeader, "nope")
	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, wrong)
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status with a wrong secret = %d, want 403", recorder.Code)
	}
	right := httptest.NewRequest(http.MethodPost, "/api/system/backup-status", nil)
	right.Header.Set(handler.BackupSecretHeader, controlSecret)
	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, right)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status with the secret = %d, want 200 (body = %q)", recorder.Code, recorder.Body.String())
	}
	if views.backups != 1 {
		t.Errorf("recorded %d backups, want 1", views.backups)
	}
}

func TestTheAnalyticsBeaconIsRateLimited(t *testing.T) {
	_, router, _ := newControlRouter(t)
	allowed := middleware.ClientLogRateLimit
	for attempt := 1; attempt <= allowed; attempt++ {
		rec := serve(t, router, http.MethodPost, "/api/analytics/view", controlBeacon, "")
		if rec.Code != http.StatusAccepted {
			t.Fatalf("beacon %d = %d, want 202 (body = %q)", attempt, rec.Code, rec.Body.String())
		}
	}
	rec := serve(t, router, http.MethodPost, "/api/analytics/view", controlBeacon, "")
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("beacon past the limit = %d, want 429 (body = %q)", rec.Code, rec.Body.String())
	}
}

type sudoSessionStore struct {
	sudo bool
}

func (s sudoSessionStore) LoadSession(ctx context.Context, sessionID int64) (store.Session, error) {
	session := store.Session{
		ID:         sessionID,
		UserID:     chiefID,
		Device:     "Windows",
		Client:     store.ClientWeb,
		LastSeenAt: time.Now(),
		ExpiresAt:  time.Now().Add(time.Hour),
	}
	if s.sudo {
		until := time.Now().Add(5 * time.Minute)
		session.SudoUntil = &until
	}
	return session, nil
}

func (s sudoSessionStore) TouchSession(ctx context.Context, sessionID int64, ipHash,
	country string) error {
	return nil
}

func issueWithSession(t *testing.T, deps Dependencies, userID int64, role string,
	sessionID int64) string {
	t.Helper()
	tokens, err := auth.NewTokens(deps.Config.JWTSecret, deps.Config.JWTTTL)
	if err != nil {
		t.Fatalf("auth.NewTokens() returned an unexpected error: %v", err)
	}
	token, err := tokens.IssueSessionToken(userID, role, 0, sessionID)
	if err != nil {
		t.Fatalf("IssueSessionToken() returned an unexpected error: %v", err)
	}
	return token
}

func TestTheControlRoomWritesNeedAPasswordConfirmation(t *testing.T) {
	deps, _, _ := newControlRouter(t)
	deps.Live = sudoSessionStore{sudo: false}
	router := NewRouter(deps)
	token := issueWithSession(t, deps, chiefID, auth.RoleSuperAdmin, 7)
	rec := serve(t, router, http.MethodPost, "/api/super-admin/admins/invite",
		`{"email":"new@sust.edu"}`, token)
	if rec.Code != http.StatusPreconditionRequired {
		t.Fatalf("status = %d, want 428 (body = %q)", rec.Code, rec.Body.String())
	}
	deps.Live = sudoSessionStore{sudo: true}
	router = NewRouter(deps)
	token = issueWithSession(t, deps, chiefID, auth.RoleSuperAdmin, 7)
	rec = serve(t, router, http.MethodPost, "/api/super-admin/admins/invite",
		`{"email":"new@sust.edu"}`, token)
	if rec.Code >= http.StatusBadRequest {
		t.Fatalf("status = %d, want a success after the confirmation (body = %q)",
			rec.Code, rec.Body.String())
	}
}

func TestEveryOfficeRouteDeclaresAPermission(t *testing.T) {
	deps, _, _ := newControlRouter(t)
	needs := OfficeNeeds()
	deps.Needs = needs
	NewRouter(deps)
	if missing := needs.Undeclared(); len(missing) > 0 {
		t.Errorf("office routes without a permission: %v", missing)
	}
	if extra := needs.Unused(); len(extra) > 0 {
		t.Errorf("the permission table names routes that do not exist: %v", extra)
	}
	for pattern, permission := range needs.Table() {
		if permission != "" && !auth.IsPermission(permission) {
			t.Errorf("route %q needs the unknown permission %q", pattern, permission)
		}
	}
}

func TestOfficeRoutesCheckThePermission(t *testing.T) {
	deps, _, _ := newControlRouter(t)
	deps.Granted = limitedPermissions{allowed: auth.PermStudents}
	router := NewRouter(deps)
	tokens := controlTokens(t, deps)
	allowed := serve(t, router, http.MethodGet, "/api/admin/students", "", tokens[auth.RoleAdmin])
	if allowed.Code != http.StatusOK {
		t.Fatalf("status on the allowed route = %d, want 200 (body = %q)",
			allowed.Code, allowed.Body.String())
	}
	refused := serve(t, router, http.MethodGet, "/api/admin/teachers", "", tokens[auth.RoleAdmin])
	if refused.Code != http.StatusForbidden {
		t.Fatalf("status on the refused route = %d, want 403 (body = %q)",
			refused.Code, refused.Body.String())
	}
	if got := errorField(t, refused); got != middleware.MessageNoPermission {
		t.Errorf("error = %q, want %q", got, middleware.MessageNoPermission)
	}
	chief := serve(t, router, http.MethodGet, "/api/admin/teachers", "", tokens[auth.RoleSuperAdmin])
	if chief.Code != http.StatusOK {
		t.Errorf("the super admin was refused a read: %d (body = %q)", chief.Code, chief.Body.String())
	}
}

type limitedPermissions struct {
	allowed string
}

func (l limitedPermissions) Permissions(ctx context.Context, userID int64) ([]string, error) {
	return []string{l.allowed}, nil
}

package server

import (
	"context"
	"encoding/json"
	"github.com/raHI-79/250-project/backend/internal/auth"
	"github.com/raHI-79/250-project/backend/internal/middleware"
	"github.com/raHI-79/250-project/backend/internal/store"
	"github.com/raHI-79/250-project/backend/internal/upload"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type emptyAccountStore struct{}

func (emptyAccountStore) GetProfile(ctx context.Context, userID int64) (store.Profile, error) {
	return store.Profile{}, store.ErrNotFound
}

func (emptyAccountStore) UpdateProfile(ctx context.Context, userID int64, role string, update store.ProfileUpdate) (string, error) {
	return "", store.ErrNotFound
}

func (emptyAccountStore) GetPasswordHash(ctx context.Context, userID int64) (string, error) {
	return "", store.ErrNotFound
}

func (emptyAccountStore) SetPassword(ctx context.Context, userID int64, passwordHash string) (int, error) {
	return 0, store.ErrNotFound
}

func (emptyAccountStore) FilterDropCourseIDs(ctx context.Context, courseIDs []int64) ([]int64, error) {
	return nil, nil
}

func (emptyAccountStore) FindPerson(ctx context.Context, userID int64) (store.PersonCard, error) {
	return store.PersonCard{}, store.ErrNotFound
}

const (
	accessStudentID = int64(101)
	accessCRID      = int64(102)
	accessTeacherID = int64(103)
	accessAdminID   = int64(104)
)

type accessUserStore struct{}

func (accessUserStore) GetAuthUser(ctx context.Context, userID int64) (store.AuthUser, error) {
	switch userID {
	case accessStudentID:
		return store.AuthUser{ID: accessStudentID, Identifier: "2023331001", Role: "student", Status: store.StatusActive}, nil
	case accessCRID:
		return store.AuthUser{ID: accessCRID, Identifier: "2023331002", Role: "student", Status: store.StatusActive, IsCR: true}, nil
	case accessTeacherID:
		return store.AuthUser{ID: accessTeacherID, Identifier: "T-001", Role: "teacher", Status: store.StatusActive}, nil
	case accessAdminID:
		return store.AuthUser{ID: accessAdminID, Identifier: "admin", Role: "admin", Status: store.StatusActive}, nil
	default:
		return store.AuthUser{}, store.ErrNotFound
	}
}

type emptyAdminStore struct{}

func (emptyAdminStore) ListPendingRegistrations(ctx context.Context) ([]store.PendingRegistration, error) {
	return nil, nil
}

func (emptyAdminStore) SetRegistrationStatus(ctx context.Context, userID int64, status string) error {
	return store.ErrNotFound
}

func (emptyAdminStore) ListStudents(ctx context.Context, filter store.StudentFilter) ([]store.StudentRow, int, error) {
	return nil, 0, nil
}

func (emptyAdminStore) RemoveStudent(ctx context.Context, userID int64) error {
	return store.ErrNotFound
}

func (emptyAdminStore) PromoteCR(ctx context.Context, userID int64) (store.PromotedCR, error) {
	return store.PromotedCR{}, store.ErrNotFound
}

func (emptyAdminStore) ListCRs(ctx context.Context, filter store.CRFilter) ([]store.StudentRow, error) {
	return nil, nil
}

func (emptyAdminStore) DemoteCR(ctx context.Context, userID int64) error {
	return store.ErrNotFound
}

func (emptyAdminStore) DeactivateCR(ctx context.Context, userID int64) error {
	return store.ErrNotFound
}

func (emptyAdminStore) CreateTeacher(ctx context.Context, teacher store.NewTeacher) (int64, error) {
	return 0, store.ErrDuplicateAccount
}

func (emptyAdminStore) UpdateTeacher(ctx context.Context, userID int64, update store.TeacherUpdate) error {
	return store.ErrNotFound
}

func (emptyAdminStore) ResetTeacherInvite(ctx context.Context, userID int64, inviteHash string, expiresAt time.Time) (store.TeacherInvite, error) {
	return store.TeacherInvite{}, store.ErrNotFound
}

func (emptyAdminStore) ListTeachers(ctx context.Context, filter store.TeacherFilter) ([]store.TeacherRow, error) {
	return nil, nil
}

func (emptyAdminStore) DeactivateTeacher(ctx context.Context, userID int64) error {
	return store.ErrNotFound
}

func (emptyAdminStore) Stats(ctx context.Context) (store.Stats, error) {
	return store.Stats{}, nil
}

func (emptyAdminStore) FindAccount(ctx context.Context, userID int64) (store.AccountRef, error) {
	return store.AccountRef{}, store.ErrNotFound
}

func (emptyAdminStore) RestoreAccount(ctx context.Context, userID int64) (string, error) {
	return "", store.ErrNotFound
}

func (emptyAdminStore) DeleteAccount(ctx context.Context, userID int64) (string, error) {
	return "", store.ErrNotFound
}

func (emptyAdminStore) WriteAudit(ctx context.Context, adminID int64, action, targetSummary string) error {
	return nil
}

func (emptyAdminStore) ListAudit(ctx context.Context, limit int) ([]store.AuditRow, error) {
	return nil, nil
}

type adminRoute struct {
	method string
	path   string
}

func adminRoutes() []adminRoute {
	return []adminRoute{
		{method: http.MethodGet, path: "/api/admin/registrations"},
		{method: http.MethodPost, path: "/api/admin/registrations/1/approve"},
		{method: http.MethodPost, path: "/api/admin/registrations/1/reject"},
		{method: http.MethodGet, path: "/api/admin/students"},
		{method: http.MethodPost, path: "/api/admin/students/1/remove"},
		{method: http.MethodPost, path: "/api/admin/crs/promote"},
		{method: http.MethodGet, path: "/api/admin/crs"},
		{method: http.MethodPost, path: "/api/admin/crs/1/demote"},
		{method: http.MethodPost, path: "/api/admin/crs/1/deactivate"},
		{method: http.MethodPost, path: "/api/admin/teachers"},
		{method: http.MethodGet, path: "/api/admin/teachers"},
		{method: http.MethodPatch, path: "/api/admin/teachers/1"},
		{method: http.MethodPost, path: "/api/admin/teachers/1/resend-invite"},
		{method: http.MethodPost, path: "/api/admin/teachers/1/deactivate"},
		{method: http.MethodPost, path: "/api/admin/users/1/restore"},
		{method: http.MethodDelete, path: "/api/admin/users/1/permanent"},
		{method: http.MethodGet, path: "/api/admin/audit"},
		{method: http.MethodGet, path: "/api/admin/stats"},
		{method: http.MethodPost, path: "/api/admin/courses"},
		{method: http.MethodGet, path: "/api/admin/courses"},
		{method: http.MethodPost, path: "/api/admin/routines"},
		{method: http.MethodGet, path: "/api/admin/routines"},
		{method: http.MethodDelete, path: "/api/admin/routines/1"},
	}
}

func newAccessRouter(t *testing.T) (Dependencies, http.Handler) {
	t.Helper()
	deps := newTestDependencies(t)
	deps.Users = accessUserStore{}
	deps.People = emptyAdminStore{}
	deps.Catalog = emptyCatalogStore{}
	deps.Lookups = emptyLookupStore{}
	deps.Alerts = emptyNotificationService{}
	deps.Classes = emptyCRStore{}
	deps.Items = emptyItemStore{}
	deps.Notifier = silentNotifier{}
	deps.Teachers = emptyTeacherStore{}
	deps.Students = emptyStudentStore{}
	deps.Alarms = silentAlerts{}
	deps.Notices = emptyNoticeStore{}
	deps.Drives = emptyDriveLinkStore{}
	deps.Contacts = emptyDirectoryStore{}
	deps.Files = upload.NewSaver(t.TempDir(), 1)
	deps.Photos = upload.NewPhotos(t.TempDir())
	deps.Accounts = emptyAccountStore{}
	deps.Chats = emptyChatStore{}
	deps.Votes = emptyPollStore{}
	deps.Devices = emptyPushStore{}
	return deps, NewRouter(deps)
}

func errorField(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("body is not valid JSON: %v (body = %q)", err, rec.Body.String())
	}
	text, _ := payload["error"].(string)
	return text
}

func TestAdminRoutesNeedAnAdminToken(t *testing.T) {
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
			name:       "with a teacher token",
			token:      issue(t, deps, accessTeacherID, auth.RoleTeacher),
			wantStatus: http.StatusForbidden,
			wantError:  middleware.MessageAccessDenied,
		},
	}
	for _, route := range adminRoutes() {
		for _, caller := range callers {
			t.Run(route.method+" "+route.path+" "+caller.name, func(t *testing.T) {
				req := httptest.NewRequest(route.method, route.path, nil)
				if caller.token != "" {
					req.Header.Set("Authorization", "Bearer "+caller.token)
				}
				rec := httptest.NewRecorder()
				router.ServeHTTP(rec, req)
				if rec.Code != caller.wantStatus {
					t.Fatalf("status = %d, want %d (body = %q)", rec.Code, caller.wantStatus, rec.Body.String())
				}
				if got := errorField(t, rec); got != caller.wantError {
					t.Errorf("error = %q, want %q", got, caller.wantError)
				}
			})
		}
	}
}

func TestAdminRoutesReachTheHandlerForAnAdmin(t *testing.T) {
	deps, router := newAccessRouter(t)
	token := issue(t, deps, accessAdminID, auth.RoleAdmin)
	tests := []struct {
		name       string
		method     string
		path       string
		wantStatus int
	}{
		{name: "registrations", method: http.MethodGet, path: "/api/admin/registrations", wantStatus: http.StatusOK},
		{name: "students", method: http.MethodGet, path: "/api/admin/students", wantStatus: http.StatusOK},
		{name: "class representatives", method: http.MethodGet, path: "/api/admin/crs", wantStatus: http.StatusOK},
		{name: "teachers", method: http.MethodGet, path: "/api/admin/teachers", wantStatus: http.StatusOK},
		{name: "stats", method: http.MethodGet, path: "/api/admin/stats", wantStatus: http.StatusOK},
		{name: "approve", method: http.MethodPost, path: "/api/admin/registrations/1/approve", wantStatus: http.StatusNotFound},
		{name: "remove", method: http.MethodPost, path: "/api/admin/students/1/remove", wantStatus: http.StatusNotFound},
		{name: "demote", method: http.MethodPost, path: "/api/admin/crs/1/demote", wantStatus: http.StatusNotFound},
		{name: "deactivate a teacher", method: http.MethodPost, path: "/api/admin/teachers/1/deactivate", wantStatus: http.StatusNotFound},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := httptest.NewRequest(test.method, test.path, nil)
			req.Header.Set("Authorization", "Bearer "+token)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)
			if rec.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d (body = %q)", rec.Code, test.wantStatus, rec.Body.String())
			}
		})
	}
	stats := httptest.NewRequest(http.MethodGet, "/api/admin/stats", nil)
	stats.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, stats)
	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("stats body is not valid JSON: %v", err)
	}
	for _, name := range []string{"total_students", "total_crs", "total_teachers", "pending_approvals"} {
		if _, ok := payload[name]; !ok {
			t.Errorf("the stats response has no %q field (body = %q)", name, rec.Body.String())
		}
	}
	if _, ok := payload["error"]; ok {
		t.Errorf("the stats response carries an error (body = %q)", rec.Body.String())
	}
}

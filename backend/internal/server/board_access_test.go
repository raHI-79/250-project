package server

import (
	"context"
	"github.com/raHI-79/250-project/backend/internal/auth"
	"github.com/raHI-79/250-project/backend/internal/middleware"
	"github.com/raHI-79/250-project/backend/internal/store"
	"net/http"
	"testing"
)

type emptyNoticeStore struct{}

func (emptyNoticeStore) GetStudentProfile(ctx context.Context, userID int64) (store.StudentProfile, error) {
	return store.StudentProfile{UserID: userID, Batch: "CSE-23", Section: "A", IsCR: userID == accessCRID}, nil
}

func (emptyNoticeStore) CreateNotice(ctx context.Context, notice store.NewNotice) (store.NoticeRow, error) {
	return store.NoticeRow{}, store.ErrNotFound
}

func (emptyNoticeStore) ListAdminNotices(ctx context.Context) ([]store.NoticeRow, error) {
	return nil, nil
}

func (emptyNoticeStore) ListStudentAdminNotices(ctx context.Context, batch string) ([]store.NoticeRow, error) {
	return nil, nil
}

func (emptyNoticeStore) ListCRNotices(ctx context.Context, createdBy int64) ([]store.NoticeRow, error) {
	return nil, nil
}

func (emptyNoticeStore) ListBatchCRNotices(ctx context.Context, batch string, otherBatches bool) ([]store.NoticeRow, error) {
	return nil, nil
}

func (emptyNoticeStore) ListTeacherNotices(ctx context.Context, createdBy int64) ([]store.NoticeRow, error) {
	return nil, nil
}

func (emptyNoticeStore) ListStudentNotices(ctx context.Context, batch, section, source, scope string) ([]store.NoticeRow, error) {
	return nil, nil
}

func (emptyNoticeStore) DeleteOwnNotice(ctx context.Context, noticeID int64, source string, createdBy int64) (string, error) {
	return "", store.ErrNotFound
}

func (emptyNoticeStore) DeleteAdminNotice(ctx context.Context, noticeID int64) (string, error) {
	return "", store.ErrNotFound
}

func (emptyNoticeStore) DeleteAnyNotice(ctx context.Context, noticeID int64) (string, error) {
	return "", store.ErrNotFound
}

func (emptyNoticeStore) EditNotice(ctx context.Context, noticeID int64, title, description,
	linkURL, source string, createdBy int64) error {
	return store.ErrNotFound
}

type emptyDriveLinkStore struct{}

func (emptyDriveLinkStore) GetStudentProfile(ctx context.Context, userID int64) (store.StudentProfile, error) {
	return store.StudentProfile{UserID: userID, Batch: "CSE-23", Section: "A", IsCR: userID == accessCRID}, nil
}

func (emptyDriveLinkStore) GetBatchDriveLink(ctx context.Context, batch string) (store.DriveLinkRow, error) {
	return store.DriveLinkRow{}, store.ErrNotFound
}

func (emptyDriveLinkStore) SaveBatchDriveLink(ctx context.Context, link store.NewDriveLink) (store.DriveLinkRow, error) {
	return store.DriveLinkRow{}, store.ErrNotFound
}

func (emptyDriveLinkStore) DeleteBatchDriveLink(ctx context.Context, batch string) error {
	return store.ErrNotFound
}

func (emptyDriveLinkStore) ListBatchDriveLinks(ctx context.Context) ([]store.DriveLinkRow, error) {
	return nil, nil
}

type emptyDirectoryStore struct{}

func (emptyDirectoryStore) ListDirectory(ctx context.Context, filter store.DirectoryFilter) ([]store.DirectoryRow, error) {
	return nil, nil
}

func (emptyDirectoryStore) ListTeacherInfo(ctx context.Context, search string) ([]store.TeacherInfoRow, error) {
	return nil, nil
}

func boardAdminRoutes() []scheduleRoute {
	return []scheduleRoute{
		{method: http.MethodPost, path: "/api/admin/notices", wantForOwn: http.StatusBadRequest},
		{method: http.MethodGet, path: "/api/admin/notices", wantForOwn: http.StatusOK},
		{method: http.MethodDelete, path: "/api/admin/notices/1", wantForOwn: http.StatusNotFound},
	}
}

func boardCRRoutes() []scheduleRoute {
	return []scheduleRoute{
		{method: http.MethodPost, path: "/api/cr/notices", wantForOwn: http.StatusBadRequest},
		{method: http.MethodGet, path: "/api/cr/notices", wantForOwn: http.StatusOK},
		{method: http.MethodGet, path: "/api/cr/drive-link", wantForOwn: http.StatusOK},
		{method: http.MethodPut, path: "/api/cr/drive-link", body: `{`, wantForOwn: http.StatusBadRequest},
		{method: http.MethodDelete, path: "/api/cr/drive-link", wantForOwn: http.StatusNotFound},
	}
}

func boardStudentRoutes() []scheduleRoute {
	return []scheduleRoute{
		{method: http.MethodGet, path: "/api/students/notices", wantForOwn: http.StatusOK},
		{method: http.MethodGet, path: "/api/students/drive-links", wantForOwn: http.StatusOK},
		{method: http.MethodGet, path: "/api/students/directory", wantForOwn: http.StatusOK},
	}
}

func TestBoardAdminRoutesNeedAnAdminToken(t *testing.T) {
	deps, router := newAccessRouter(t)
	refused := map[string]string{
		"student":              issue(t, deps, accessStudentID, auth.RoleStudent),
		"class representative": issue(t, deps, accessCRID, auth.RoleCR),
		"teacher":              issue(t, deps, accessTeacherID, auth.RoleTeacher),
	}
	adminToken := issue(t, deps, accessAdminID, auth.RoleAdmin)
	for _, route := range boardAdminRoutes() {
		assertRouteAccess(t, router, route, adminToken, refused)
	}
}

func TestBoardCRRoutesNeedACRToken(t *testing.T) {
	deps, router := newAccessRouter(t)
	refused := map[string]string{
		"student": issue(t, deps, accessStudentID, auth.RoleStudent),
		"teacher": issue(t, deps, accessTeacherID, auth.RoleTeacher),
		"admin":   issue(t, deps, accessAdminID, auth.RoleAdmin),
	}
	crToken := issue(t, deps, accessCRID, auth.RoleCR)
	for _, route := range boardCRRoutes() {
		assertRouteAccess(t, router, route, crToken, refused)
	}
}

func TestBoardStudentRoutesRefuseTeachersAndAdmins(t *testing.T) {
	deps, router := newAccessRouter(t)
	refused := map[string]string{
		"teacher": issue(t, deps, accessTeacherID, auth.RoleTeacher),
		"admin":   issue(t, deps, accessAdminID, auth.RoleAdmin),
	}
	allowed := map[string]string{
		"student":              issue(t, deps, accessStudentID, auth.RoleStudent),
		"class representative": issue(t, deps, accessCRID, auth.RoleCR),
	}
	for _, route := range boardStudentRoutes() {
		assertRouteAccess(t, router, route, "", refused)
		for role, token := range allowed {
			t.Run(route.path+" as a "+role, func(t *testing.T) {
				rec := serve(t, router, route.method, route.path, route.body, token)
				if rec.Code != route.wantForOwn {
					t.Fatalf("status = %d, want %d (body = %q)", rec.Code, route.wantForOwn, rec.Body.String())
				}
			})
		}
	}
}

func TestTeachersInfoRefusesTheAdmin(t *testing.T) {
	deps, router := newAccessRouter(t)
	route := scheduleRoute{method: http.MethodGet, path: "/api/teachers-info", wantForOwn: http.StatusOK}
	allowed := map[string]string{
		"student":              issue(t, deps, accessStudentID, auth.RoleStudent),
		"class representative": issue(t, deps, accessCRID, auth.RoleCR),
		"teacher":              issue(t, deps, accessTeacherID, auth.RoleTeacher),
	}
	refused := map[string]string{"admin": issue(t, deps, accessAdminID, auth.RoleAdmin)}
	for role, token := range allowed {
		t.Run("as a "+role, func(t *testing.T) {
			rec := serve(t, router, route.method, route.path, route.body, token)
			if rec.Code != route.wantForOwn {
				t.Fatalf("status = %d, want %d (body = %q)", rec.Code, route.wantForOwn, rec.Body.String())
			}
		})
	}
	assertRouteAccess(t, router, route, "", refused)
}

func assertRouteAccess(t *testing.T, router http.Handler, route scheduleRoute, allowedToken string, refused map[string]string) {
	t.Helper()
	t.Run(route.method+" "+route.path+" without a token", func(t *testing.T) {
		rec := serve(t, router, route.method, route.path, route.body, "")
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want 401 (body = %q)", rec.Code, rec.Body.String())
		}
		if got := errorField(t, rec); got != middleware.MessageUnauthorized {
			t.Errorf("error = %q, want %q", got, middleware.MessageUnauthorized)
		}
	})
	for role, token := range refused {
		t.Run(route.method+" "+route.path+" as a "+role, func(t *testing.T) {
			rec := serve(t, router, route.method, route.path, route.body, token)
			if rec.Code != http.StatusForbidden {
				t.Fatalf("status = %d, want 403 (body = %q)", rec.Code, rec.Body.String())
			}
			if got := errorField(t, rec); got != middleware.MessageAccessDenied {
				t.Errorf("error = %q, want %q", got, middleware.MessageAccessDenied)
			}
		})
	}
	if allowedToken == "" {
		return
	}
	t.Run(route.method+" "+route.path+" with the right token", func(t *testing.T) {
		rec := serve(t, router, route.method, route.path, route.body, allowedToken)
		if rec.Code != route.wantForOwn {
			t.Fatalf("status = %d, want %d (body = %q)", rec.Code, route.wantForOwn, rec.Body.String())
		}
	})
}

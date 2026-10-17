package server

import (
	"context"
	"github.com/raHI-79/250-project/backend/internal/auth"
	"github.com/raHI-79/250-project/backend/internal/handler"
	"github.com/raHI-79/250-project/backend/internal/middleware"
	"github.com/raHI-79/250-project/backend/internal/store"
	"net/http"
	"testing"
	"time"
)

type emptyMessageStore struct{}

func (emptyMessageStore) GetMessageSender(ctx context.Context, userID int64) (store.Sender, error) {
	switch userID {
	case accessCRID:
		return store.Sender{UserID: userID, Role: auth.RoleCR, Name: "Access CR", Batch: "CSE-23", Section: "A", IsCR: true}, nil
	case accessTeacherID:
		return store.Sender{UserID: userID, Role: auth.RoleTeacher, Name: "Access Teacher"}, nil
	case accessStudentID:
		return store.Sender{UserID: userID, Role: auth.RoleStudent, Name: "Access Student", Batch: "CSE-23", Section: "A"}, nil
	default:
		return store.Sender{}, store.ErrNotFound
	}
}

func (emptyMessageStore) FindMessageTarget(ctx context.Context, userID int64, kind string) (store.MessageTarget, error) {
	return store.MessageTarget{}, store.ErrNotFound
}

func (emptyMessageStore) FindReplyTarget(ctx context.Context, messageID, teacherUserID int64) (store.MessageTarget, error) {
	return store.MessageTarget{}, store.ErrNotFound
}

func (emptyMessageStore) CreateMessage(ctx context.Context, message store.NewMessage) (store.MessageRow, error) {
	return store.MessageRow{}, store.ErrNotFound
}

func (emptyMessageStore) ListCRInbox(ctx context.Context, crUserID int64) ([]store.MessageRow, error) {
	return nil, nil
}

func (emptyMessageStore) ListTeacherInbox(ctx context.Context, teacherUserID int64) ([]store.MessageRow, error) {
	return nil, nil
}

func (emptyMessageStore) ListSentMessages(ctx context.Context, senderID int64) ([]store.MessageRow, error) {
	return nil, nil
}

func (emptyMessageStore) ListStudentMessages(ctx context.Context, userID int64, batch, section string) ([]store.MessageRow, error) {
	return nil, nil
}

func (emptyMessageStore) ListCROptions(ctx context.Context, batch string) ([]store.CROption, error) {
	return nil, nil
}

func (emptyMessageStore) SearchStudents(ctx context.Context, search string) ([]store.StudentOption, error) {
	return nil, nil
}

type emptyPollStore struct{}

func (emptyPollStore) GetStudentProfile(ctx context.Context, userID int64) (store.StudentProfile, error) {
	return store.StudentProfile{UserID: userID, Batch: "CSE-23", Section: "A", IsCR: userID == accessCRID}, nil
}

func (emptyPollStore) CreatePoll(ctx context.Context, poll store.NewPoll) (store.PollRow, error) {
	return store.PollRow{}, store.ErrNotFound
}

func (emptyPollStore) ClosePoll(ctx context.Context, pollID, createdBy int64, now time.Time) error {
	return store.ErrNotFound
}

func (emptyPollStore) ListCreatorPolls(ctx context.Context, viewerUserID, createdBy int64) ([]store.PollRow, error) {
	return nil, nil
}

func (emptyPollStore) ListStudentPolls(ctx context.Context, viewerUserID int64, batch, section string) ([]store.PollRow, error) {
	return nil, nil
}

func (emptyPollStore) VoteInPoll(ctx context.Context, pollID, optionID, userID int64, batch, section string) error {
	return store.ErrNotFound
}

type protectedRoute struct {
	method  string
	path    string
	allowed []string
}

func everyRole() []string {
	return []string{auth.RoleStudent, auth.RoleCR, auth.RoleTeacher, auth.RoleAdmin}
}

func panelRoles() []string {
	return []string{auth.RoleStudent, auth.RoleCR, auth.RoleTeacher}
}

func studentRoles() []string {
	return []string{auth.RoleStudent, auth.RoleCR}
}

func adminOnly() []string {
	return []string{auth.RoleAdmin}
}

func crOnly() []string {
	return []string{auth.RoleCR}
}

func teacherOnly() []string {
	return []string{auth.RoleTeacher}
}

func protectedRoutes() []protectedRoute {
	return []protectedRoute{
		{method: http.MethodPost, path: "/api/auth/logout", allowed: everyRole()},
		{method: http.MethodGet, path: "/api/me", allowed: everyRole()},
		{method: http.MethodGet, path: "/api/lookups/batches", allowed: everyRole()},
		{method: http.MethodGet, path: "/api/lookups/sections?batch=CSE-23", allowed: everyRole()},
		{method: http.MethodGet, path: "/api/lookups/courses", allowed: everyRole()},
		{method: http.MethodGet, path: "/api/lookups/teachers", allowed: everyRole()},
		{method: http.MethodGet, path: "/api/notifications", allowed: panelRoles()},
		{method: http.MethodGet, path: "/api/notifications/unread-count", allowed: panelRoles()},
		{method: http.MethodGet, path: "/api/teachers-info", allowed: panelRoles()},
		{method: http.MethodGet, path: "/api/admin/registrations", allowed: adminOnly()},
		{method: http.MethodPost, path: "/api/admin/registrations/1/approve", allowed: adminOnly()},
		{method: http.MethodPost, path: "/api/admin/registrations/1/reject", allowed: adminOnly()},
		{method: http.MethodGet, path: "/api/admin/students", allowed: adminOnly()},
		{method: http.MethodPost, path: "/api/admin/students/1/remove", allowed: adminOnly()},
		{method: http.MethodPost, path: "/api/admin/crs", allowed: adminOnly()},
		{method: http.MethodGet, path: "/api/admin/crs", allowed: adminOnly()},
		{method: http.MethodPost, path: "/api/admin/crs/1/demote", allowed: adminOnly()},
		{method: http.MethodPost, path: "/api/admin/crs/1/deactivate", allowed: adminOnly()},
		{method: http.MethodPost, path: "/api/admin/teachers", allowed: adminOnly()},
		{method: http.MethodGet, path: "/api/admin/teachers", allowed: adminOnly()},
		{method: http.MethodPost, path: "/api/admin/teachers/1/deactivate", allowed: adminOnly()},
		{method: http.MethodGet, path: "/api/admin/stats", allowed: adminOnly()},
		{method: http.MethodPost, path: "/api/admin/courses", allowed: adminOnly()},
		{method: http.MethodGet, path: "/api/admin/courses", allowed: adminOnly()},
		{method: http.MethodPost, path: "/api/admin/routines", allowed: adminOnly()},
		{method: http.MethodGet, path: "/api/admin/routines", allowed: adminOnly()},
		{method: http.MethodDelete, path: "/api/admin/routines/1", allowed: adminOnly()},
		{method: http.MethodPost, path: "/api/admin/notices", allowed: adminOnly()},
		{method: http.MethodGet, path: "/api/admin/notices", allowed: adminOnly()},
		{method: http.MethodDelete, path: "/api/admin/notices/1", allowed: adminOnly()},
		{method: http.MethodPost, path: "/api/admin/materials", allowed: adminOnly()},
		{method: http.MethodGet, path: "/api/admin/materials", allowed: adminOnly()},
		{method: http.MethodGet, path: "/api/cr/routine", allowed: crOnly()},
		{method: http.MethodPost, path: "/api/cr/class-requests", allowed: crOnly()},
		{method: http.MethodGet, path: "/api/cr/room-availability?date=2026-10-20", allowed: crOnly()},
		{method: http.MethodGet, path: "/api/cr/requests", allowed: crOnly()},
		{method: http.MethodPost, path: "/api/cr/notices", allowed: crOnly()},
		{method: http.MethodGet, path: "/api/cr/notices", allowed: crOnly()},
		{method: http.MethodPost, path: "/api/cr/materials", allowed: crOnly()},
		{method: http.MethodGet, path: "/api/cr/materials", allowed: crOnly()},
		{method: http.MethodDelete, path: "/api/cr/materials/1", allowed: crOnly()},
		{method: http.MethodPost, path: "/api/cr/messages", allowed: crOnly()},
		{method: http.MethodGet, path: "/api/cr/messages/inbox", allowed: crOnly()},
		{method: http.MethodGet, path: "/api/cr/messages/sent", allowed: crOnly()},
		{method: http.MethodPost, path: "/api/cr/polls", allowed: crOnly()},
		{method: http.MethodPost, path: "/api/cr/polls/1/close", allowed: crOnly()},
		{method: http.MethodGet, path: "/api/cr/polls", allowed: crOnly()},
		{method: http.MethodGet, path: "/api/teacher/requests", allowed: teacherOnly()},
		{method: http.MethodPost, path: "/api/teacher/requests/1/approve", allowed: teacherOnly()},
		{method: http.MethodPost, path: "/api/teacher/requests/1/reject", allowed: teacherOnly()},
		{method: http.MethodGet, path: "/api/teacher/schedule?date=2026-10-20", allowed: teacherOnly()},
		{method: http.MethodGet, path: "/api/teacher/classes/upcoming", allowed: teacherOnly()},
		{method: http.MethodPost, path: "/api/teacher/classes/1/cancel", allowed: teacherOnly()},
		{method: http.MethodPost, path: "/api/teacher/routines/1/cancel", allowed: teacherOnly()},
		{method: http.MethodPost, path: "/api/teacher/messages", allowed: teacherOnly()},
		{method: http.MethodGet, path: "/api/teacher/messages/inbox", allowed: teacherOnly()},
		{method: http.MethodGet, path: "/api/teacher/messages/sent", allowed: teacherOnly()},
		{method: http.MethodGet, path: "/api/teacher/lookups/crs", allowed: teacherOnly()},
		{method: http.MethodGet, path: "/api/teacher/lookups/students", allowed: teacherOnly()},
		{method: http.MethodGet, path: "/api/students/upcoming-routine", allowed: studentRoles()},
		{method: http.MethodGet, path: "/api/students/drop-course-routine", allowed: studentRoles()},
		{method: http.MethodGet, path: "/api/students/exams", allowed: studentRoles()},
		{method: http.MethodGet, path: "/api/students/routine", allowed: studentRoles()},
		{method: http.MethodGet, path: "/api/students/notices/cr", allowed: studentRoles()},
		{method: http.MethodGet, path: "/api/students/notices/admin", allowed: studentRoles()},
		{method: http.MethodGet, path: "/api/students/materials", allowed: studentRoles()},
		{method: http.MethodGet, path: "/api/students/directory", allowed: studentRoles()},
		{method: http.MethodGet, path: "/api/students/messages", allowed: studentRoles()},
		{method: http.MethodGet, path: "/api/students/polls", allowed: studentRoles()},
		{method: http.MethodPost, path: "/api/students/polls/1/vote", allowed: studentRoles()},
	}
}

func accessTokens(t *testing.T, deps Dependencies) map[string]string {
	t.Helper()
	return map[string]string{
		auth.RoleStudent: issue(t, deps, accessStudentID, auth.RoleStudent),
		auth.RoleCR:      issue(t, deps, accessCRID, auth.RoleCR),
		auth.RoleTeacher: issue(t, deps, accessTeacherID, auth.RoleTeacher),
		auth.RoleAdmin:   issue(t, deps, accessAdminID, auth.RoleAdmin),
	}
}

func allows(route protectedRoute, role string) bool {
	for _, allowed := range route.allowed {
		if allowed == role {
			return true
		}
	}
	return false
}

func TestEveryProtectedRouteChecksTheCaller(t *testing.T) {
	deps, router := newAccessRouter(t)
	tokens := accessTokens(t, deps)
	for _, route := range protectedRoutes() {
		t.Run(route.method+" "+route.path+" without a token", func(t *testing.T) {
			rec := serve(t, router, route.method, route.path, "", "")
			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, want 401 (body = %q)", rec.Code, rec.Body.String())
			}
			if got := errorField(t, rec); got != middleware.MessageUnauthorized {
				t.Errorf("error = %q, want %q", got, middleware.MessageUnauthorized)
			}
		})
		for _, role := range everyRole() {
			if allows(route, role) {
				continue
			}
			t.Run(route.method+" "+route.path+" as a "+role, func(t *testing.T) {
				rec := serve(t, router, route.method, route.path, "", tokens[role])
				if rec.Code != http.StatusForbidden {
					t.Fatalf("status = %d, want 403 (body = %q)", rec.Code, rec.Body.String())
				}
				if got := errorField(t, rec); got != middleware.MessageAccessDenied {
					t.Errorf("error = %q, want %q", got, middleware.MessageAccessDenied)
				}
			})
		}
	}
}

func TestStudentsCannotSendMessages(t *testing.T) {
	deps, router := newAccessRouter(t)
	tokens := accessTokens(t, deps)
	callers := []struct {
		name  string
		token string
	}{
		{name: "without a token"},
		{name: "as a student", token: tokens[auth.RoleStudent]},
		{name: "as a class representative", token: tokens[auth.RoleCR]},
	}
	for _, caller := range callers {
		t.Run(caller.name, func(t *testing.T) {
			rec := serve(t, router, http.MethodPost, "/api/students/messages", `{"content":"hello"}`, caller.token)
			if rec.Code != http.StatusNotFound {
				t.Fatalf("status = %d, want 404 (body = %q)", rec.Code, rec.Body.String())
			}
		})
	}
}

func TestCRCannotCloseAnotherPoll(t *testing.T) {
	deps, router := newAccessRouter(t)
	rec := serve(t, router, http.MethodPost, "/api/cr/polls/9/close", "", issue(t, deps, accessCRID, auth.RoleCR))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404 (body = %q)", rec.Code, rec.Body.String())
	}
	if got := errorField(t, rec); got != handler.MessagePollUnknown {
		t.Errorf("error = %q, want %q", got, handler.MessagePollUnknown)
	}
}

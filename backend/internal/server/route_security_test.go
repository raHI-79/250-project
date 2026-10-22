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

type emptyPushStore struct{}

func (emptyPushStore) SavePushSubscription(ctx context.Context,
	subscription store.PushSubscription) (int64, error) {
	return 0, store.ErrNotFound
}

func (emptyPushStore) DeletePushSubscription(ctx context.Context, userID int64, endpoint string) error {
	return nil
}

type emptyChatStore struct{}

func (emptyChatStore) GetMessageViewer(ctx context.Context, userID int64) (store.MessageViewer, error) {
	switch userID {
	case accessCRID:
		return store.MessageViewer{UserID: userID, Name: "Access CR", Role: auth.RoleCR, Batch: "CSE-23", Section: "A"}, nil
	case accessTeacherID:
		return store.MessageViewer{UserID: userID, Name: "Access Teacher", Role: auth.RoleTeacher}, nil
	case accessStudentID:
		return store.MessageViewer{UserID: userID, Name: "Access Student", Role: auth.RoleStudent, Batch: "CSE-23", Section: "A"}, nil
	case accessAdminID:
		return store.MessageViewer{UserID: userID, Name: "Access Admin", Role: auth.RoleAdmin}, nil
	default:
		return store.MessageViewer{}, store.ErrNotFound
	}
}

func (emptyChatStore) SearchContacts(ctx context.Context, viewerID int64, search, role string) ([]store.ContactRow, error) {
	return nil, nil
}

func (emptyChatStore) GetOrCreateDirect(ctx context.Context, viewerID, otherID int64) (store.Conversation, error) {
	return store.Conversation{}, store.ErrInactiveTarget
}

func (emptyChatStore) CreateGroup(ctx context.Context, group store.NewGroup) (store.Conversation, error) {
	return store.Conversation{}, store.ErrNotFound
}

func (emptyChatStore) ListConversationMembers(ctx context.Context, conversationID int64) ([]store.MemberRow, error) {
	return nil, nil
}

func (emptyChatStore) AddConversationMembers(ctx context.Context, conversationID, actorID int64,
	userIDs []int64) (int, error) {
	return 0, store.ErrNotFound
}

func (emptyChatStore) RemoveConversationMember(ctx context.Context, conversationID, userID int64) error {
	return store.ErrNotAMember
}

func (emptyChatStore) SetConversationMemberAdmin(ctx context.Context, conversationID, userID int64,
	admin bool) error {
	return store.ErrNotAMember
}

func (emptyChatStore) LeaveConversation(ctx context.Context, conversationID, userID int64) error {
	return store.ErrNotAMember
}

func (emptyChatStore) RenameConversation(ctx context.Context, conversationID int64, title string) error {
	return store.ErrNotAGroup
}

func (emptyChatStore) ActiveUserIDs(ctx context.Context, userIDs []int64) ([]int64, error) {
	return userIDs, nil
}

func (emptyChatStore) BatchMembers(ctx context.Context, batch, section string) ([]int64, error) {
	return nil, nil
}

func (emptyChatStore) ListConversations(ctx context.Context, viewer store.MessageViewer, limit int) ([]store.ConversationRow, error) {
	return nil, nil
}

func (emptyChatStore) FindConversationAccess(ctx context.Context, conversationID int64, viewer store.MessageViewer) (store.ConversationAccess, error) {
	return store.ConversationAccess{}, store.ErrNotFound
}

func (emptyChatStore) ListChatMessages(ctx context.Context, conversationID int64, before *time.Time, limit int) ([]store.ChatMessageRow, error) {
	return nil, nil
}

func (emptyChatStore) CreateChatMessage(ctx context.Context, message store.NewChatMessage) (store.ChatMessageRow, error) {
	return store.ChatMessageRow{}, store.ErrNotFound
}

func (emptyChatStore) MarkConversationRead(ctx context.Context, userID, conversationID int64, at time.Time) error {
	return nil
}

func (emptyChatStore) CountUnreadMessages(ctx context.Context, viewer store.MessageViewer) (int, error) {
	return 0, nil
}

func (emptyChatStore) ConversationRecipients(ctx context.Context, conversationID,
	senderID int64) ([]int64, error) {
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

func (emptyPollStore) VoteInPoll(ctx context.Context, pollID int64, optionIDs []int64, userID int64,
	batch, section string) error {
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
		{method: http.MethodPost, path: "/api/admin/crs/promote", allowed: adminOnly()},
		{method: http.MethodGet, path: "/api/admin/crs", allowed: adminOnly()},
		{method: http.MethodPost, path: "/api/admin/crs/1/demote", allowed: adminOnly()},
		{method: http.MethodPost, path: "/api/admin/crs/1/deactivate", allowed: adminOnly()},
		{method: http.MethodPost, path: "/api/admin/teachers", allowed: adminOnly()},
		{method: http.MethodPatch, path: "/api/admin/teachers/1", allowed: adminOnly()},
		{method: http.MethodPost, path: "/api/admin/teachers/1/resend-invite", allowed: adminOnly()},
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
		{method: http.MethodGet, path: "/api/cr/routine", allowed: crOnly()},
		{method: http.MethodGet, path: "/api/cr/courses", allowed: crOnly()},
		{method: http.MethodGet, path: "/api/cr/room-availability", allowed: crOnly()},
		{method: http.MethodGet, path: "/api/cr/upcoming", allowed: crOnly()},
		{method: http.MethodPost, path: "/api/cr/items", allowed: crOnly()},
		{method: http.MethodPatch, path: "/api/cr/items/1", allowed: crOnly()},
		{method: http.MethodDelete, path: "/api/cr/items/1", allowed: crOnly()},
		{method: http.MethodPost, path: "/api/cr/items/1/request-reschedule", allowed: crOnly()},
		{method: http.MethodPost, path: "/api/cr/items/1/cancel", allowed: crOnly()},
		{method: http.MethodPost, path: "/api/cr/items/1/final", allowed: crOnly()},
		{method: http.MethodDelete, path: "/api/cr/items/1/final", allowed: crOnly()},
		{method: http.MethodGet, path: "/api/cr/final", allowed: crOnly()},
		{method: http.MethodGet, path: "/api/cr/final/preview?ids=1", allowed: crOnly()},
		{method: http.MethodPost, path: "/api/cr/final/publish", allowed: crOnly()},
		{method: http.MethodPost, path: "/api/cr/notices", allowed: crOnly()},
		{method: http.MethodGet, path: "/api/cr/notices", allowed: crOnly()},
		{method: http.MethodGet, path: "/api/cr/drive-link", allowed: crOnly()},
		{method: http.MethodPut, path: "/api/cr/drive-link", allowed: crOnly()},
		{method: http.MethodDelete, path: "/api/cr/drive-link", allowed: crOnly()},
		{method: http.MethodGet, path: "/api/messages/contacts", allowed: everyRole()},
		{method: http.MethodGet, path: "/api/messages/conversations", allowed: everyRole()},
		{method: http.MethodPost, path: "/api/messages/direct", allowed: everyRole()},
		{method: http.MethodPost, path: "/api/cr/polls", allowed: crOnly()},
		{method: http.MethodPost, path: "/api/cr/polls/1/close", allowed: crOnly()},
		{method: http.MethodGet, path: "/api/cr/polls", allowed: crOnly()},
		{method: http.MethodGet, path: "/api/teacher/fixed-routine", allowed: teacherOnly()},
		{method: http.MethodGet, path: "/api/teacher/upcoming/next-day", allowed: teacherOnly()},
		{method: http.MethodGet, path: "/api/teacher/upcoming/final", allowed: teacherOnly()},
		{method: http.MethodPost, path: "/api/teacher/items/1/confirm", allowed: teacherOnly()},
		{method: http.MethodPost, path: "/api/teacher/items/1/cancel", allowed: teacherOnly()},
		{method: http.MethodPost, path: "/api/teacher/items/1/request-reschedule", allowed: teacherOnly()},
		{method: http.MethodPost, path: "/api/teacher/items/1/reschedule-request/accept", allowed: teacherOnly()},
		{method: http.MethodPost, path: "/api/teacher/items/1/reschedule-request/decline", allowed: teacherOnly()},
		{method: http.MethodGet, path: "/api/teacher/exams", allowed: teacherOnly()},
		{method: http.MethodPost, path: "/api/teacher/exams", allowed: teacherOnly()},
		{method: http.MethodPatch, path: "/api/teacher/exams/1", allowed: teacherOnly()},
		{method: http.MethodPost, path: "/api/teacher/exams/1/cancel", allowed: teacherOnly()},
		{method: http.MethodDelete, path: "/api/teacher/exams/1", allowed: teacherOnly()},
		{method: http.MethodPatch, path: "/api/schedule/items/1/link", allowed: everyRole()},
		{method: http.MethodPost, path: "/api/messages/groups", allowed: everyRole()},
		{method: http.MethodGet, path: "/api/messages/conversations/1/messages", allowed: everyRole()},
		{method: http.MethodPost, path: "/api/messages/conversations/1/messages", allowed: everyRole()},
		{method: http.MethodGet, path: "/api/messages/unread-count", allowed: everyRole()},
		{method: http.MethodPatch, path: "/api/messages/conversations/1", allowed: everyRole()},
		{method: http.MethodGet, path: "/api/messages/conversations/1/members", allowed: everyRole()},
		{method: http.MethodPost, path: "/api/messages/conversations/1/members", allowed: everyRole()},
		{method: http.MethodDelete, path: "/api/messages/conversations/1/members/2", allowed: everyRole()},
		{method: http.MethodPost, path: "/api/messages/conversations/1/members/2/admin", allowed: everyRole()},
		{method: http.MethodDelete, path: "/api/messages/conversations/1/members/2/admin", allowed: everyRole()},
		{method: http.MethodPost, path: "/api/messages/conversations/1/leave", allowed: everyRole()},
		{method: http.MethodPost, path: "/api/push/subscribe", allowed: panelRoles()},
		{method: http.MethodDelete, path: "/api/push/subscribe", allowed: panelRoles()},
		{method: http.MethodPost, path: "/api/teacher/notices", allowed: teacherOnly()},
		{method: http.MethodGet, path: "/api/students/today", allowed: studentRoles()},
		{method: http.MethodGet, path: "/api/students/upcoming-routine", allowed: studentRoles()},
		{method: http.MethodGet, path: "/api/students/exams", allowed: studentRoles()},
		{method: http.MethodGet, path: "/api/students/routine", allowed: studentRoles()},
		{method: http.MethodGet, path: "/api/students/notices", allowed: studentRoles()},
		{method: http.MethodGet, path: "/api/students/drive-links", allowed: studentRoles()},
		{method: http.MethodGet, path: "/api/students/directory", allowed: studentRoles()},
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

func TestRetiredMessageRoutesAreGone(t *testing.T) {
	deps, router := newAccessRouter(t)
	tokens := accessTokens(t, deps)
	retired := []struct {
		method string
		path   string
	}{
		{method: http.MethodPost, path: "/api/students/messages"},
		{method: http.MethodGet, path: "/api/students/messages"},
		{method: http.MethodPost, path: "/api/cr/messages"},
		{method: http.MethodGet, path: "/api/cr/messages/inbox"},
		{method: http.MethodGet, path: "/api/cr/messages/sent"},
		{method: http.MethodPost, path: "/api/teacher/messages"},
		{method: http.MethodGet, path: "/api/teacher/messages/inbox"},
		{method: http.MethodGet, path: "/api/teacher/messages/sent"},
		{method: http.MethodGet, path: "/api/teacher/lookups/crs"},
		{method: http.MethodGet, path: "/api/teacher/lookups/students"},
		{method: http.MethodGet, path: "/api/students/notices/cr"},
		{method: http.MethodGet, path: "/api/students/notices/admin"},
		{method: http.MethodGet, path: "/api/students/materials"},
		{method: http.MethodGet, path: "/api/students/drop-course-routine"},
		{method: http.MethodPost, path: "/api/cr/materials"},
		{method: http.MethodGet, path: "/api/cr/materials"},
		{method: http.MethodPost, path: "/api/admin/materials"},
		{method: http.MethodGet, path: "/api/admin/materials"},
	}
	for _, route := range retired {
		t.Run(route.method+" "+route.path, func(t *testing.T) {
			for _, role := range []string{auth.RoleStudent, auth.RoleCR, auth.RoleTeacher, auth.RoleAdmin} {
				rec := serve(t, router, route.method, route.path, "", tokens[role])
				if rec.Code != http.StatusNotFound {
					t.Fatalf("%s as a %s = %d, want 404 (body = %q)",
						route.path, role, rec.Code, rec.Body.String())
				}
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

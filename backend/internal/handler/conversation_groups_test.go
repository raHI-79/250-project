package handler

import (
	"errors"
	"github.com/raHI-79/250-project/backend/internal/auth"
	"github.com/raHI-79/250-project/backend/internal/store"
	"net/http"
	"testing"
)

func groupAccess(role string) store.ConversationAccess {
	return store.ConversationAccess{
		Conversation: store.Conversation{
			ID: 5, Kind: store.ConversationGroup, Title: "CSE-23 section A", CreatedBy: chatTeacherID,
		},
		MemberRole:  role,
		MemberCount: 3,
	}
}

func TestOnlyTeachersAndCRsStartGroups(t *testing.T) {
	silenceLog(t)
	tests := []struct {
		name       string
		userID     int64
		role       string
		body       string
		wantStatus int
		wantError  string
		wantBatch  string
	}{
		{
			name:       "a teacher starts a group with people and a batch",
			userID:     chatTeacherID,
			role:       auth.RoleTeacher,
			body:       `{"title":"CSE-23 section A","user_ids":[101],"batches":[{"batch":"CSE-22","section":"B"}]}`,
			wantStatus: http.StatusCreated,
			wantBatch:  "CSE-22/B",
		},
		{
			name:       "a class representative uses their own batch",
			userID:     chatCRID,
			role:       auth.RoleCR,
			body:       `{"title":"Reps","batches":[{"batch":"CSE-23"}]}`,
			wantStatus: http.StatusCreated,
			wantBatch:  "CSE-23/",
		},
		{
			name:       "a class representative cannot add another batch",
			userID:     chatCRID,
			role:       auth.RoleCR,
			body:       `{"title":"Reps","batches":[{"batch":"CSE-21"}]}`,
			wantStatus: http.StatusForbidden,
			wantError:  MessageOwnBatchOnly,
		},
		{
			name:       "a student cannot start a group",
			userID:     chatStudentID,
			role:       auth.RoleStudent,
			body:       `{"title":"Study circle","user_ids":[101]}`,
			wantStatus: http.StatusForbidden,
			wantError:  MessageGroupNotAllowed,
		},
		{
			name:       "the office cannot start a group",
			userID:     chatAdminID,
			role:       auth.RoleAdmin,
			body:       `{"title":"Everyone","user_ids":[101]}`,
			wantStatus: http.StatusForbidden,
			wantError:  MessageGroupNotAllowed,
		},
		{
			name:       "a group needs a name",
			userID:     chatTeacherID,
			role:       auth.RoleTeacher,
			body:       `{"title":"  ","user_ids":[101]}`,
			wantStatus: http.StatusBadRequest,
			wantError:  "Group name is required.",
		},
		{
			name:       "a group needs somebody else",
			userID:     chatTeacherID,
			role:       auth.RoleTeacher,
			body:       `{"title":"Alone"}`,
			wantStatus: http.StatusBadRequest,
			wantError:  MessageGroupNeedsPeople,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			chats := newChatStore()
			chats.batchMembers = []int64{202, 203}
			handler := newChatHandler(t, chats, &fakeFileStore{}, &recordingAlerts{})
			request := chatRequest(t, http.MethodPost, "/api/messages/groups", test.body,
				test.userID, test.role)
			rec := serve(handler.StartGroup(), request)
			if rec.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d (body = %q)", rec.Code, test.wantStatus, rec.Body.String())
			}
			if test.wantError != "" {
				if got := errorMessage(t, rec); got != test.wantError {
					t.Errorf("error = %q, want %q", got, test.wantError)
				}
				if len(chats.groups) != 0 {
					t.Error("the store made a group although the request was refused")
				}
				return
			}
			if len(chats.groups) != 1 {
				t.Fatalf("the store received %d groups, want 1", len(chats.groups))
			}
			created := chats.groups[0]
			if created.CreatedBy != test.userID {
				t.Errorf("created by %d, want %d", created.CreatedBy, test.userID)
			}
			if len(created.Members) == 0 {
				t.Error("the group has no members")
			}
			if test.wantBatch != "" && (len(chats.batchAsked) != 1 || chats.batchAsked[0] != test.wantBatch) {
				t.Errorf("batches asked = %v, want %q", chats.batchAsked, test.wantBatch)
			}
			if decodeBody(t, rec)["member_role"] != store.MemberAdmin {
				t.Error("the creator is not the group admin in the response")
			}
		})
	}
}

func TestGroupAdminActionsNeedAGroupAdmin(t *testing.T) {
	silenceLog(t)
	tests := []struct {
		name       string
		role       string
		kind       string
		call       func(*Conversations) http.HandlerFunc
		method     string
		path       string
		body       string
		wantStatus int
		wantError  string
	}{
		{
			name:       "an admin adds members",
			role:       store.MemberAdmin,
			call:       func(c *Conversations) http.HandlerFunc { return c.AddMembers() },
			method:     http.MethodPost,
			path:       "/api/messages/conversations/5/members",
			body:       `{"user_ids":[101]}`,
			wantStatus: http.StatusOK,
		},
		{
			name:       "a member cannot add members",
			role:       store.MemberRegular,
			call:       func(c *Conversations) http.HandlerFunc { return c.AddMembers() },
			method:     http.MethodPost,
			path:       "/api/messages/conversations/5/members",
			body:       `{"user_ids":[101]}`,
			wantStatus: http.StatusForbidden,
			wantError:  MessageNotAGroupAdmin,
		},
		{
			name:       "a member cannot rename the group",
			role:       store.MemberRegular,
			call:       func(c *Conversations) http.HandlerFunc { return c.Rename() },
			method:     http.MethodPatch,
			path:       "/api/messages/conversations/5",
			body:       `{"title":"New name"}`,
			wantStatus: http.StatusForbidden,
			wantError:  MessageNotAGroupAdmin,
		},
		{
			name:       "a direct chat has no admins",
			role:       store.MemberRegular,
			kind:       store.ConversationDirect,
			call:       func(c *Conversations) http.HandlerFunc { return c.Rename() },
			method:     http.MethodPatch,
			path:       "/api/messages/conversations/5",
			body:       `{"title":"New name"}`,
			wantStatus: http.StatusBadRequest,
			wantError:  MessageNotAGroup,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			chats := newChatStore()
			chats.access = groupAccess(test.role)
			if test.kind == store.ConversationDirect {
				chats.access.Kind = store.ConversationDirect
				chats.access.Title = ""
			}
			handler := newChatHandler(t, chats, &fakeFileStore{}, &recordingAlerts{})
			request := chatRequest(t, test.method, test.path, test.body, chatTeacherID, auth.RoleTeacher)
			request.SetPathValue("id", "5")
			rec := serve(test.call(handler), request)
			if rec.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d (body = %q)", rec.Code, test.wantStatus, rec.Body.String())
			}
			if test.wantError != "" {
				if got := errorMessage(t, rec); got != test.wantError {
					t.Errorf("error = %q, want %q", got, test.wantError)
				}
			}
		})
	}
}

func TestGroupMemberChangesReportStoreRules(t *testing.T) {
	silenceLog(t)
	tests := []struct {
		name       string
		storeErr   error
		call       func(*Conversations) http.HandlerFunc
		wantStatus int
		wantError  string
	}{
		{
			name:       "removing the last admin",
			storeErr:   store.ErrLastAdmin,
			call:       func(c *Conversations) http.HandlerFunc { return c.RemoveMember() },
			wantStatus: http.StatusConflict,
			wantError:  MessageLastGroupAdmin,
		},
		{
			name:       "removing somebody who left",
			storeErr:   store.ErrNotAMember,
			call:       func(c *Conversations) http.HandlerFunc { return c.RemoveMember() },
			wantStatus: http.StatusNotFound,
			wantError:  MessageNotAGroupMember,
		},
		{
			name:       "demoting the last admin",
			storeErr:   store.ErrLastAdmin,
			call:       func(c *Conversations) http.HandlerFunc { return c.SetMemberAdmin(false) },
			wantStatus: http.StatusConflict,
			wantError:  MessageLastGroupAdmin,
		},
		{
			name:       "a broken store",
			storeErr:   errors.New("database is down"),
			call:       func(c *Conversations) http.HandlerFunc { return c.RemoveMember() },
			wantStatus: http.StatusInternalServerError,
			wantError:  "Internal server error",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			chats := newChatStore()
			chats.access = groupAccess(store.MemberAdmin)
			chats.memberErr = test.storeErr
			handler := newChatHandler(t, chats, &fakeFileStore{}, &recordingAlerts{})
			request := chatRequest(t, http.MethodDelete, "/api/messages/conversations/5/members/8", "",
				chatTeacherID, auth.RoleTeacher)
			request.SetPathValue("id", "5")
			request.SetPathValue("user_id", "8")
			rec := serve(test.call(handler), request)
			if rec.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d (body = %q)", rec.Code, test.wantStatus, rec.Body.String())
			}
			if got := errorMessage(t, rec); got != test.wantError {
				t.Errorf("error = %q, want %q", got, test.wantError)
			}
		})
	}
}

func TestGroupMembersAndLeaveAreOpenToEveryMember(t *testing.T) {
	silenceLog(t)
	chats := newChatStore()
	chats.access = groupAccess(store.MemberRegular)
	chats.members = []store.MemberRow{
		{UserID: chatTeacherID, Name: "Active Teacher", Role: store.MemberAdmin, AccountRole: auth.RoleTeacher},
		{UserID: chatStudentID, Name: "Plain Student", Role: store.MemberRegular, AccountRole: auth.RoleStudent},
	}
	handler := newChatHandler(t, chats, &fakeFileStore{}, &recordingAlerts{})
	listing := chatRequest(t, http.MethodGet, "/api/messages/conversations/5/members", "",
		chatStudentID, auth.RoleStudent)
	listing.SetPathValue("id", "5")
	rec := serve(handler.Members(), listing)
	if rec.Code != http.StatusOK {
		t.Fatalf("members status = %d, want 200 (body = %q)", rec.Code, rec.Body.String())
	}
	people := items(t, rec)
	if len(people) != 2 || people[0]["role"] != store.MemberAdmin {
		t.Fatalf("members = %v, want the admin first", people)
	}
	for _, person := range people {
		if person["added_at"] == nil {
			t.Errorf("member = %v, want the time they joined", person)
		}
	}
	leaving := chatRequest(t, http.MethodPost, "/api/messages/conversations/5/leave", "",
		chatStudentID, auth.RoleStudent)
	leaving.SetPathValue("id", "5")
	left := serve(handler.Leave(), leaving)
	if left.Code != http.StatusOK {
		t.Fatalf("leave status = %d, want 200 (body = %q)", left.Code, left.Body.String())
	}
	if len(chats.left) != 1 || chats.left[0] != chatStudentID {
		t.Errorf("left = %v, want the student", chats.left)
	}
}

package handler

import (
	"github.com/raHI-79/250-project/backend/internal/auth"
	"github.com/raHI-79/250-project/backend/internal/store"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"
)

func editCall(t *testing.T, chats *fakeChatStore, method, body string, userID int64,
	role string) *httptest.ResponseRecorder {
	t.Helper()
	handler := newChatHandler(t, chats, &fakeFileStore{}, &recordingAlerts{})
	req := chatRequest(t, method, "/api/messages/5", body, userID, role)
	req.SetPathValue("id", strconv.Itoa(5))
	rec := httptest.NewRecorder()
	if method == http.MethodDelete {
		handler.Delete()(rec, req)
		return rec
	}
	handler.Edit()(rec, req)
	return rec
}

func ownMessage(t *testing.T, age time.Duration, attachment bool) store.OwnMessage {
	t.Helper()
	return store.OwnMessage{
		ConversationID: 3,
		CreatedAt:      accountConfig(t, "test").Now().Add(-age),
		HasAttachment:  attachment,
		FilePath:       "uploads/attachments/note.pdf",
	}
}

func TestEditMessageStaysInsideTheWindow(t *testing.T) {
	silenceLog(t)
	tests := []struct {
		name       string
		age        time.Duration
		deleted    bool
		attachment bool
		body       string
		ownedErr   error
		wantStatus int
		wantError  string
	}{
		{name: "just sent", age: 10 * time.Second, body: `{"body":"Corrected"}`,
			wantStatus: http.StatusOK},
		{name: "one second before the limit", age: 119 * time.Second, body: `{"body":"Corrected"}`,
			wantStatus: http.StatusOK},
		{name: "exactly two minutes later", age: 2 * time.Minute, body: `{"body":"Too late"}`,
			wantStatus: http.StatusConflict, wantError: MessageEditWindowOver},
		{name: "an hour later", age: time.Hour, body: `{"body":"Too late"}`,
			wantStatus: http.StatusConflict, wantError: MessageEditWindowOver},
		{name: "already deleted", age: time.Second, deleted: true, body: `{"body":"Again"}`,
			wantStatus: http.StatusConflict, wantError: MessageEditWindowOver},
		{name: "somebody else's message", age: time.Second, body: `{"body":"Mine now"}`,
			ownedErr: store.ErrNotFound, wantStatus: http.StatusNotFound, wantError: MessageMessageMissing},
		{name: "an empty body without an attachment", age: time.Second, body: `{"body":"   "}`,
			wantStatus: http.StatusBadRequest,
			wantError:  "Message is required when nothing is attached."},
		{name: "an empty body with an attachment", age: time.Second, attachment: true, body: `{"body":""}`,
			wantStatus: http.StatusOK},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			owned := ownMessage(t, test.age, test.attachment)
			owned.Deleted = test.deleted
			chats := newChatStore()
			chats.owned = owned
			chats.ownedErr = test.ownedErr
			rec := editCall(t, chats, http.MethodPatch, test.body, chatStudentID, auth.RoleStudent)
			if rec.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d (body = %q)", rec.Code, test.wantStatus, rec.Body.String())
			}
			if test.wantError != "" {
				if got := errorMessage(t, rec); got != test.wantError {
					t.Errorf("error = %q, want %q", got, test.wantError)
				}
				return
			}
			if rec.Code == http.StatusOK && decodeBody(t, rec)["edited_at"] == nil {
				t.Error("the edited message carries no edited_at stamp")
			}
		})
	}
}

func TestDeleteMessageStaysInsideTheWindow(t *testing.T) {
	silenceLog(t)
	tests := []struct {
		name       string
		age        time.Duration
		deleted    bool
		ownedErr   error
		deleteErr  error
		wantStatus int
		wantError  string
	}{
		{name: "just sent", age: 5 * time.Second, wantStatus: http.StatusOK},
		{name: "after two minutes", age: 3 * time.Minute, wantStatus: http.StatusConflict,
			wantError: MessageEditWindowOver},
		{name: "already deleted", age: time.Second, deleted: true, wantStatus: http.StatusConflict,
			wantError: MessageEditWindowOver},
		{name: "not the sender", age: time.Second, ownedErr: store.ErrNotFound,
			wantStatus: http.StatusNotFound, wantError: MessageMessageMissing},
		{name: "the window closed between the two calls", age: time.Second,
			deleteErr: store.ErrEditWindowOver, wantStatus: http.StatusConflict,
			wantError: MessageEditWindowOver},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			owned := ownMessage(t, test.age, false)
			owned.Deleted = test.deleted
			chats := newChatStore()
			chats.owned = owned
			chats.ownedErr = test.ownedErr
			chats.deleteErr = test.deleteErr
			rec := editCall(t, chats, http.MethodDelete, "", chatStudentID, auth.RoleStudent)
			if rec.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d (body = %q)", rec.Code, test.wantStatus, rec.Body.String())
			}
			if test.wantError != "" {
				if got := errorMessage(t, rec); got != test.wantError {
					t.Errorf("error = %q, want %q", got, test.wantError)
				}
				return
			}
			if chats.deletedMsg != 5 {
				t.Errorf("deleted message = %d, want 5", chats.deletedMsg)
			}
		})
	}
}

func TestContactsSearchCarriesTheViewerRole(t *testing.T) {
	silenceLog(t)
	tests := []struct {
		name   string
		userID int64
		role   string
	}{
		{name: "a plain student", userID: chatStudentID, role: auth.RoleStudent},
		{name: "a class representative", userID: chatCRID, role: auth.RoleCR},
		{name: "a teacher", userID: chatTeacherID, role: auth.RoleTeacher},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			chats := newChatStore()
			req := chatRequest(t, http.MethodGet, "/api/messages/contacts?search=a", "", test.userID, test.role)
			rec := httptest.NewRecorder()
			newChatHandler(t, chats, &fakeFileStore{}, &recordingAlerts{}).Contacts()(rec, req)
			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200 (body = %q)", rec.Code, rec.Body.String())
			}
			if chats.searchViewer != test.role {
				t.Errorf("viewer role = %q, want %q", chats.searchViewer, test.role)
			}
		})
	}
}

func TestStartDirectRefusesStudentToStudent(t *testing.T) {
	silenceLog(t)
	chats := newChatStore()
	chats.directErr = store.ErrContactBlocked
	req := chatRequest(t, http.MethodPost, "/api/messages/direct", `{"user_id":15}`,
		chatStudentID, auth.RoleStudent)
	rec := httptest.NewRecorder()
	newChatHandler(t, chats, &fakeFileStore{}, &recordingAlerts{}).StartDirect()(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403 (body = %q)", rec.Code, rec.Body.String())
	}
	if got := errorMessage(t, rec); got != MessageContactBlocked {
		t.Errorf("error = %q, want %q", got, MessageContactBlocked)
	}
}

func TestClearChatRecordsOnlyForTheCaller(t *testing.T) {
	silenceLog(t)
	chats := newChatStore()
	chats.access = store.ConversationAccess{
		Conversation: store.Conversation{ID: 3, Kind: store.ConversationGroup, Title: "CSE-23 section A"},
		MemberRole:   store.MemberRegular,
		MemberCount:  4,
	}
	req := chatRequest(t, http.MethodDelete, "/api/messages/conversations/3", "",
		chatStudentID, auth.RoleStudent)
	req.SetPathValue("id", "3")
	rec := httptest.NewRecorder()
	newChatHandler(t, chats, &fakeFileStore{}, &recordingAlerts{}).ClearChat()(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body = %q)", rec.Code, rec.Body.String())
	}
	if len(chats.cleared) != 1 || chats.cleared[0] != 3 {
		t.Errorf("cleared = %v, want the one conversation", chats.cleared)
	}
	if len(chats.left) != 0 {
		t.Errorf("left = %v, want clearing a chat to leave the membership alone", chats.left)
	}
	if got := decodeBody(t, rec)["message"]; got != MessageChatCleared {
		t.Errorf("message = %v, want %q", got, MessageChatCleared)
	}
}

func TestClearChatNeedsMembership(t *testing.T) {
	silenceLog(t)
	chats := newChatStore()
	chats.accessErr = store.ErrNotFound
	req := chatRequest(t, http.MethodDelete, "/api/messages/conversations/9", "",
		chatStudentID, auth.RoleStudent)
	req.SetPathValue("id", "9")
	rec := httptest.NewRecorder()
	newChatHandler(t, chats, &fakeFileStore{}, &recordingAlerts{}).ClearChat()(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404 (body = %q)", rec.Code, rec.Body.String())
	}
	if len(chats.cleared) != 0 {
		t.Errorf("cleared = %v, want nothing recorded", chats.cleared)
	}
}

package handler

import (
	"bytes"
	"context"
	"errors"
	"github.com/raHI-79/250-project/backend/internal/auth"
	"github.com/raHI-79/250-project/backend/internal/store"
	"github.com/raHI-79/250-project/backend/internal/upload"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

const (
	chatStudentID = int64(11)
	chatCRID      = int64(12)
	chatTeacherID = int64(13)
	chatAdminID   = int64(14)
	chatBatch     = "CSE-23"
)

type sentChatMessage struct {
	conversationID int64
	senderID       int64
	body           string
	link           string
	filePath       string
	fileName       string
}

type fakeChatStore struct {
	viewers      map[int64]store.MessageViewer
	contacts     []store.ContactRow
	conversation store.Conversation
	access       store.ConversationAccess
	accessErr    error
	rows         []store.ConversationRow
	messages     []store.ChatMessageRow
	unread       int
	recipients   []int64
	groups       []store.NewGroup
	members      []store.MemberRow
	addedMembers []int64
	removed      []int64
	roleChanges  map[int64]bool
	left         []int64
	renamed      []string
	activeIDs    []int64
	batchAsked   []string
	batchMembers []int64
	memberErr    error
	directWith   []int64
	sent         []sentChatMessage
	reads        []int64
	createErr    error
	directErr    error
	searchRole   string
	searchText   string
	err          error
}

func newChatStore() *fakeChatStore {
	return &fakeChatStore{
		viewers: map[int64]store.MessageViewer{
			chatStudentID: {
				UserID: chatStudentID, Name: "Plain Student", Role: auth.RoleStudent,
				Batch: chatBatch, Section: "A",
			},
			chatCRID: {
				UserID: chatCRID, Name: "Class Representative", Role: auth.RoleCR,
				Batch: chatBatch, Section: "B",
			},
			chatTeacherID: {UserID: chatTeacherID, Name: "Active Teacher", Role: auth.RoleTeacher},
			chatAdminID:   {UserID: chatAdminID, Name: "admin", Role: auth.RoleAdmin},
		},
		recipients:  []int64{chatStudentID},
		roleChanges: make(map[int64]bool),
	}
}

func (f *fakeChatStore) GetMessageViewer(ctx context.Context, userID int64) (store.MessageViewer, error) {
	viewer, ok := f.viewers[userID]
	if !ok {
		return store.MessageViewer{}, store.ErrNotFound
	}
	return viewer, nil
}

func (f *fakeChatStore) SearchContacts(ctx context.Context, viewerID int64, search, role string) ([]store.ContactRow, error) {
	f.searchText, f.searchRole = search, role
	return f.contacts, f.err
}

func (f *fakeChatStore) GetOrCreateDirect(ctx context.Context, viewerID, otherID int64) (store.Conversation, error) {
	if f.directErr != nil {
		return store.Conversation{}, f.directErr
	}
	f.directWith = append(f.directWith, otherID)
	return store.Conversation{ID: 7, Kind: store.ConversationDirect, CreatedBy: viewerID}, nil
}

func (f *fakeChatStore) CreateGroup(ctx context.Context, group store.NewGroup) (store.Conversation, error) {
	if f.err != nil {
		return store.Conversation{}, f.err
	}
	f.groups = append(f.groups, group)
	return store.Conversation{
		ID: 9, Kind: store.ConversationGroup, Title: group.Title, CreatedBy: group.CreatedBy,
	}, nil
}

func (f *fakeChatStore) ListConversationMembers(ctx context.Context, conversationID int64) ([]store.MemberRow, error) {
	return f.members, f.memberErr
}

func (f *fakeChatStore) AddConversationMembers(ctx context.Context, conversationID, actorID int64,
	userIDs []int64) (int, error) {
	if f.memberErr != nil {
		return 0, f.memberErr
	}
	f.addedMembers = append(f.addedMembers, userIDs...)
	return len(userIDs), nil
}

func (f *fakeChatStore) RemoveConversationMember(ctx context.Context, conversationID, userID int64) error {
	if f.memberErr != nil {
		return f.memberErr
	}
	f.removed = append(f.removed, userID)
	return nil
}

func (f *fakeChatStore) SetConversationMemberAdmin(ctx context.Context, conversationID, userID int64,
	admin bool) error {
	if f.memberErr != nil {
		return f.memberErr
	}
	f.roleChanges[userID] = admin
	return nil
}

func (f *fakeChatStore) LeaveConversation(ctx context.Context, conversationID, userID int64) error {
	if f.memberErr != nil {
		return f.memberErr
	}
	f.left = append(f.left, userID)
	return nil
}

func (f *fakeChatStore) RenameConversation(ctx context.Context, conversationID int64, title string) error {
	if f.memberErr != nil {
		return f.memberErr
	}
	f.renamed = append(f.renamed, title)
	return nil
}

func (f *fakeChatStore) ActiveUserIDs(ctx context.Context, userIDs []int64) ([]int64, error) {
	if f.memberErr != nil {
		return nil, f.memberErr
	}
	if f.activeIDs != nil {
		return f.activeIDs, nil
	}
	return userIDs, nil
}

func (f *fakeChatStore) BatchMembers(ctx context.Context, batch, section string) ([]int64, error) {
	if f.memberErr != nil {
		return nil, f.memberErr
	}
	f.batchAsked = append(f.batchAsked, batch+"/"+section)
	return f.batchMembers, nil
}

func (f *fakeChatStore) ListConversations(ctx context.Context, viewer store.MessageViewer, limit int) ([]store.ConversationRow, error) {
	return f.rows, f.err
}

func (f *fakeChatStore) FindConversationAccess(ctx context.Context, conversationID int64, viewer store.MessageViewer) (store.ConversationAccess, error) {
	if f.accessErr != nil {
		return store.ConversationAccess{}, f.accessErr
	}
	return f.access, nil
}

func (f *fakeChatStore) ListChatMessages(ctx context.Context, conversationID int64, before *time.Time, limit int) ([]store.ChatMessageRow, error) {
	return f.messages, f.err
}

func (f *fakeChatStore) CreateChatMessage(ctx context.Context, message store.NewChatMessage) (store.ChatMessageRow, error) {
	if f.createErr != nil {
		return store.ChatMessageRow{}, f.createErr
	}
	f.sent = append(f.sent, sentChatMessage{
		conversationID: message.ConversationID,
		senderID:       message.SenderID,
		body:           message.Body,
		link:           message.LinkURL,
		filePath:       message.FilePath,
		fileName:       message.FileName,
	})
	return store.ChatMessageRow{
		ID: 21, SenderID: message.SenderID, Body: message.Body, LinkURL: message.LinkURL,
		FilePath: message.FilePath, FileName: message.FileName, FileMime: message.FileMime,
		FileSize: message.FileSize, CreatedAt: time.Now(),
	}, nil
}

func (f *fakeChatStore) MarkConversationRead(ctx context.Context, userID, conversationID int64, at time.Time) error {
	f.reads = append(f.reads, conversationID)
	return nil
}

func (f *fakeChatStore) CountUnreadMessages(ctx context.Context, viewer store.MessageViewer) (int, error) {
	return f.unread, f.err
}

func (f *fakeChatStore) ConversationRecipients(ctx context.Context, conversationID,
	senderID int64) ([]int64, error) {
	return f.recipients, nil
}

func newChatHandler(t *testing.T, chats ConversationStore, files AttachmentStore, alerts BroadcastNotifier) *Conversations {
	t.Helper()
	return NewConversations(accountConfig(t, "test"), chats, files, alerts)
}

func chatRequest(t *testing.T, method, path, body string, userID int64, role string) *http.Request {
	t.Helper()
	return jsonRequest(t, method, path, body, userID, role)
}

func chatForm(t *testing.T, fields map[string]string, filename string, data []byte) *http.Request {
	t.Helper()
	var buffer bytes.Buffer
	writer := multipart.NewWriter(&buffer)
	for name, value := range fields {
		if err := writer.WriteField(name, value); err != nil {
			t.Fatalf("could not write the field %q: %v", name, err)
		}
	}
	if filename != "" {
		part, err := writer.CreateFormFile(upload.AttachmentField, filename)
		if err != nil {
			t.Fatalf("could not create the file part: %v", err)
		}
		if _, err := part.Write(data); err != nil {
			t.Fatalf("could not write the file part: %v", err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("could not close the multipart writer: %v", err)
	}
	request := httptest.NewRequest(http.MethodPost, "/api/messages/conversations/5/messages",
		bytes.NewReader(buffer.Bytes()))
	request.Header.Set("Content-Type", writer.FormDataContentType())
	return request
}

func TestConversationContacts(t *testing.T) {
	silenceLog(t)
	tests := []struct {
		name       string
		query      string
		wantStatus int
		wantSearch string
		wantRole   string
		wantError  string
	}{
		{name: "no filters", query: "", wantStatus: http.StatusOK},
		{name: "by name", query: "?search=ratul", wantStatus: http.StatusOK, wantSearch: "ratul"},
		{
			name:       "only teachers",
			query:      "?role=teacher",
			wantStatus: http.StatusOK,
			wantRole:   "teacher",
		},
		{
			name:       "only class representatives",
			query:      "?search=cse&role=cr",
			wantStatus: http.StatusOK,
			wantSearch: "cse",
			wantRole:   "cr",
		},
		{
			name:       "a broken role",
			query:      "?role=principal",
			wantStatus: http.StatusBadRequest,
			wantError:  "role must be one of student, cr, teacher, admin.",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			chats := newChatStore()
			chats.contacts = []store.ContactRow{
				{
					UserID: chatTeacherID, Name: "Active Teacher", Role: auth.RoleTeacher,
					Designation: "Assistant Professor", PhotoPath: "/uploads/photos/a.png",
				},
			}
			handler := newChatHandler(t, chats, &fakeFileStore{}, &recordingAlerts{})
			request := chatRequest(t, http.MethodGet, "/api/messages/contacts"+tc.query, "",
				chatStudentID, auth.RoleStudent)
			rec := serve(handler.Contacts(), request)
			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d (body = %q)", rec.Code, tc.wantStatus, rec.Body.String())
			}
			if tc.wantStatus != http.StatusOK {
				if got := errorMessage(t, rec); got != tc.wantError {
					t.Errorf("error = %q, want %q", got, tc.wantError)
				}
				return
			}
			if chats.searchText != tc.wantSearch || chats.searchRole != tc.wantRole {
				t.Errorf("searched %q/%q, want %q/%q",
					chats.searchText, chats.searchRole, tc.wantSearch, tc.wantRole)
			}
			items := items(t, rec)
			if len(items) != 1 || items[0]["user_id"] != float64(chatTeacherID) {
				t.Fatalf("items = %v, want the teacher", items)
			}
			if items[0]["designation"] != "Assistant Professor" {
				t.Errorf("designation = %v, want the teacher designation", items[0]["designation"])
			}
		})
	}
}

func TestConversationStartDirect(t *testing.T) {
	silenceLog(t)
	tests := []struct {
		name       string
		role       string
		userID     int64
		body       string
		directErr  error
		wantStatus int
		wantError  string
	}{
		{
			name:       "a student messages a teacher",
			role:       auth.RoleStudent,
			userID:     chatStudentID,
			body:       `{"user_id":13}`,
			wantStatus: http.StatusOK,
		},
		{
			name:       "a teacher messages a student",
			role:       auth.RoleTeacher,
			userID:     chatTeacherID,
			body:       `{"user_id":11}`,
			wantStatus: http.StatusOK,
		},
		{
			name:       "an admin messages a class representative",
			role:       auth.RoleAdmin,
			userID:     chatAdminID,
			body:       `{"user_id":12}`,
			wantStatus: http.StatusOK,
		},
		{
			name:       "a class representative messages a student",
			role:       auth.RoleCR,
			userID:     chatCRID,
			body:       `{"user_id":11}`,
			wantStatus: http.StatusOK,
		},
		{
			name:       "a missing user id",
			role:       auth.RoleStudent,
			userID:     chatStudentID,
			body:       `{}`,
			wantStatus: http.StatusBadRequest,
			wantError:  MessageInvalidUserID,
		},
		{
			name:       "yourself",
			role:       auth.RoleStudent,
			userID:     chatStudentID,
			body:       `{"user_id":11}`,
			directErr:  store.ErrSameUser,
			wantStatus: http.StatusBadRequest,
			wantError:  MessageSelfConversation,
		},
		{
			name:       "an account that is not active",
			role:       auth.RoleStudent,
			userID:     chatStudentID,
			body:       `{"user_id":99}`,
			directErr:  store.ErrInactiveTarget,
			wantStatus: http.StatusNotFound,
			wantError:  MessageUnknownContact,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			chats := newChatStore()
			chats.directErr = tc.directErr
			handler := newChatHandler(t, chats, &fakeFileStore{}, &recordingAlerts{})
			request := chatRequest(t, http.MethodPost, "/api/messages/direct", tc.body, tc.userID, tc.role)
			rec := serve(handler.StartDirect(), request)
			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d (body = %q)", rec.Code, tc.wantStatus, rec.Body.String())
			}
			if tc.wantStatus != http.StatusOK {
				if got := errorMessage(t, rec); got != tc.wantError {
					t.Errorf("error = %q, want %q", got, tc.wantError)
				}
				return
			}
			if len(chats.directWith) != 1 {
				t.Fatalf("the store opened %d conversations, want 1", len(chats.directWith))
			}
			if got := decodeBody(t, rec)["kind"]; got != store.ConversationDirect {
				t.Errorf("kind = %v, want %q", got, store.ConversationDirect)
			}
		})
	}
}

func TestConversationSendRules(t *testing.T) {
	silenceLog(t)
	tests := []struct {
		name       string
		access     store.ConversationAccess
		fields     map[string]string
		filename   string
		data       []byte
		wantStatus int
		wantError  string
		wantBody   string
		wantNotify string
	}{
		{
			name: "a direct message with a body",
			access: store.ConversationAccess{
				Conversation: store.Conversation{ID: 5, Kind: store.ConversationDirect},
				MemberRole:   store.MemberRegular,
			},
			fields:     map[string]string{"body": "Good morning"},
			wantStatus: http.StatusCreated,
			wantBody:   "Good morning",
			wantNotify: "user",
		},
		{
			name: "a direct message with only a file",
			access: store.ConversationAccess{
				Conversation: store.Conversation{ID: 5, Kind: store.ConversationDirect},
				MemberRole:   store.MemberRegular,
			},
			fields:     map[string]string{},
			filename:   "notice.pdf",
			data:       []byte("%PDF-1.7 attached"),
			wantStatus: http.StatusCreated,
			wantNotify: "user",
		},
		{
			name: "a direct message with only a link",
			access: store.ConversationAccess{
				Conversation: store.Conversation{ID: 5, Kind: store.ConversationDirect},
				MemberRole:   store.MemberRegular,
			},
			fields:     map[string]string{"link": "https://drive.google.com/file/x"},
			wantStatus: http.StatusCreated,
			wantNotify: "user",
		},
		{
			name: "an empty message",
			access: store.ConversationAccess{
				Conversation: store.Conversation{ID: 5, Kind: store.ConversationDirect},
				MemberRole:   store.MemberRegular,
			},
			fields:     map[string]string{"body": "   "},
			wantStatus: http.StatusBadRequest,
			wantError:  "Message is required when nothing is attached.",
		},
		{
			name: "a broken link",
			access: store.ConversationAccess{
				Conversation: store.Conversation{ID: 5, Kind: store.ConversationDirect},
				MemberRole:   store.MemberRegular,
			},
			fields:     map[string]string{"body": "see this", "link": "drive.google.com"},
			wantStatus: http.StatusBadRequest,
			wantError:  "Link must be a full link that starts with http:// or https://.",
		},
		{
			name: "the group admin posts",
			access: store.ConversationAccess{
				Conversation: store.Conversation{
					ID: 5, Kind: store.ConversationGroup, Title: "CSE-23 section A",
					CreatedBy: chatTeacherID,
				},
				MemberRole:  store.MemberAdmin,
				MemberCount: 3,
			},
			fields:     map[string]string{"body": "Extra class tomorrow"},
			wantStatus: http.StatusCreated,
			wantBody:   "Extra class tomorrow",
			wantNotify: "members",
		},
		{
			name: "a plain member posts too",
			access: store.ConversationAccess{
				Conversation: store.Conversation{
					ID: 5, Kind: store.ConversationGroup, Title: "CSE-23 section A",
					CreatedBy: chatTeacherID,
				},
				MemberRole:  store.MemberRegular,
				MemberCount: 3,
			},
			fields:     map[string]string{"body": "me too"},
			wantStatus: http.StatusCreated,
			wantBody:   "me too",
			wantNotify: "members",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			chats := newChatStore()
			chats.access = tc.access
			alerts := &recordingAlerts{}
			files := upload.NewAttachments(t.TempDir(), upload.DefaultAttachmentMB)
			handler := newChatHandler(t, chats, files, alerts)
			request := chatForm(t, tc.fields, tc.filename, tc.data)
			request.SetPathValue("id", "5")
			request = withCurrentUser(request, chatTeacherID, auth.RoleTeacher)
			rec := serve(handler.Send(), request)
			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d (body = %q)", rec.Code, tc.wantStatus, rec.Body.String())
			}
			if tc.wantStatus != http.StatusCreated {
				if got := errorMessage(t, rec); got != tc.wantError {
					t.Errorf("error = %q, want %q", got, tc.wantError)
				}
				if len(chats.sent) != 0 {
					t.Error("the store received a message although the request was refused")
				}
				return
			}
			if len(chats.sent) != 1 {
				t.Fatalf("the store received %d messages, want 1", len(chats.sent))
			}
			sent := chats.sent[0]
			if sent.conversationID != 5 || sent.senderID != chatTeacherID {
				t.Errorf("sent = %+v, want conversation 5 from the teacher", sent)
			}
			if tc.wantBody != "" && sent.body != tc.wantBody {
				t.Errorf("body = %q, want %q", sent.body, tc.wantBody)
			}
			if tc.filename != "" && sent.fileName != tc.filename {
				t.Errorf("file name = %q, want %q", sent.fileName, tc.filename)
			}
			switch tc.wantNotify {
			case "user":
				if alerts.users != 1 {
					t.Errorf("direct notifications = %d, want 1", alerts.users)
				}
				if alerts.title != "New message from Active Teacher" {
					t.Errorf("notification title = %q, want the sender name", alerts.title)
				}
			case "members":
				if alerts.users != len(chats.recipients) {
					t.Errorf("member notifications = %d, want %d", alerts.users, len(chats.recipients))
				}
				if alerts.calls != 0 {
					t.Errorf("broadcasts = %d, want none for a member group", alerts.calls)
				}
				if alerts.title != "Active Teacher in CSE-23 section A" {
					t.Errorf("notification title = %q, want the sender and the group", alerts.title)
				}
			}
		})
	}
}

func TestConversationSendRejectsABadAttachment(t *testing.T) {
	silenceLog(t)
	chats := newChatStore()
	chats.access = store.ConversationAccess{
		Conversation: store.Conversation{ID: 5, Kind: store.ConversationDirect},
		MemberRole:   store.MemberRegular,
	}
	files := upload.NewAttachments(t.TempDir(), upload.DefaultAttachmentMB)
	handler := newChatHandler(t, chats, files, &recordingAlerts{})
	request := chatForm(t, map[string]string{"body": "run this"}, "hack.exe", []byte("MZ binary"))
	request.SetPathValue("id", "5")
	request = withCurrentUser(request, chatTeacherID, auth.RoleTeacher)
	rec := serve(handler.Send(), request)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 (body = %q)", rec.Code, rec.Body.String())
	}
	if got := errorMessage(t, rec); got != MessageFileNotAllowed {
		t.Errorf("error = %q, want %q", got, MessageFileNotAllowed)
	}
	if len(chats.sent) != 0 {
		t.Error("the store received a message although the file was refused")
	}
}

func TestConversationSendRemovesTheFileWhenTheStoreFails(t *testing.T) {
	silenceLog(t)
	chats := newChatStore()
	chats.access = store.ConversationAccess{
		Conversation: store.Conversation{ID: 5, Kind: store.ConversationDirect},
		MemberRole:   store.MemberRegular,
	}
	chats.createErr = errors.New("database is down")
	files := upload.NewAttachments(t.TempDir(), upload.DefaultAttachmentMB)
	handler := newChatHandler(t, chats, files, &recordingAlerts{})
	request := chatForm(t, map[string]string{"body": "hello"}, "notice.pdf", []byte("%PDF-1.7 x"))
	request.SetPathValue("id", "5")
	request = withCurrentUser(request, chatTeacherID, auth.RoleTeacher)
	rec := serve(handler.Send(), request)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500 (body = %q)", rec.Code, rec.Body.String())
	}
	entries, err := os.ReadDir(files.Dir())
	if err == nil && len(entries) != 0 {
		t.Errorf("%d files stayed on disk, want none", len(entries))
	}
}

func TestConversationListAndUnread(t *testing.T) {
	silenceLog(t)
	last := time.Now()
	chats := newChatStore()
	chats.unread = 4
	chats.rows = []store.ConversationRow{
		{
			Conversation: store.Conversation{
				ID: 7, Kind: store.ConversationDirect, CreatedBy: chatStudentID, LastMessageAt: &last,
			},
			OtherUserID:    chatTeacherID,
			OtherName:      "Active Teacher",
			OtherPhotoPath: "/uploads/photos/a.png",
			LastBody:       "See you tomorrow",
			UnreadCount:    2,
		},
		{
			Conversation: store.Conversation{
				ID: 9, Kind: store.ConversationGroup, Title: "CSE-23 section A",
				CreatedBy: chatTeacherID, LastMessageAt: &last,
			},
			LastFileName: "slides.pptx",
			UnreadCount:  2,
			MemberRole:   store.MemberRegular,
			MemberCount:  12,
		},
		{
			Conversation: store.Conversation{
				ID: 10, Kind: store.ConversationGroup, Title: "Lab helpers",
				CreatedBy: chatTeacherID,
			},
			MemberRole:  store.MemberAdmin,
			MemberCount: 4,
		},
	}
	handler := newChatHandler(t, chats, &fakeFileStore{}, &recordingAlerts{})
	request := chatRequest(t, http.MethodGet, "/api/messages/conversations", "", chatStudentID, auth.RoleStudent)
	rec := serve(handler.List(), request)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body = %q)", rec.Code, rec.Body.String())
	}
	list := items(t, rec)
	if len(list) != 3 {
		t.Fatalf("got %d conversations, want 3", len(list))
	}
	if list[0]["title"] != "Active Teacher" || list[0]["photo_url"] != "/uploads/photos/a.png" {
		t.Errorf("direct conversation = %v, want the other person's name and photo", list[0])
	}
	if list[0]["can_post"] != true || list[0]["preview"] != "See you tomorrow" {
		t.Errorf("direct conversation = %v, want it postable with a preview", list[0])
	}
	if list[1]["title"] != "CSE-23 section A" {
		t.Errorf("group title = %v, want the stored title", list[1]["title"])
	}
	if list[1]["can_post"] != true || list[1]["member_role"] != store.MemberRegular {
		t.Errorf("group = %v, want a member who can post", list[1])
	}
	if list[1]["member_count"] != float64(12) {
		t.Errorf("member count = %v, want 12", list[1]["member_count"])
	}
	if list[1]["preview"] != "slides.pptx" {
		t.Errorf("group preview = %v, want the file name", list[1]["preview"])
	}
	if list[2]["title"] != "Lab helpers" || list[2]["member_role"] != store.MemberAdmin {
		t.Errorf("group = %v, want the admin's group", list[2])
	}
	if list[0]["unread_count"] != float64(2) {
		t.Errorf("unread_count = %v, want 2", list[0]["unread_count"])
	}
	counted := serve(handler.UnreadCount(),
		chatRequest(t, http.MethodGet, "/api/messages/unread-count", "", chatStudentID, auth.RoleStudent))
	if counted.Code != http.StatusOK {
		t.Fatalf("unread status = %d, want 200", counted.Code)
	}
	if got := decodeBody(t, counted)["count"]; got != float64(4) {
		t.Errorf("count = %v, want 4", got)
	}
}

func TestConversationMessagesMarkTheConversationRead(t *testing.T) {
	silenceLog(t)
	chats := newChatStore()
	chats.access = store.ConversationAccess{
		Conversation: store.Conversation{ID: 5, Kind: store.ConversationDirect},
		MemberRole:   store.MemberRegular,
	}
	chats.messages = []store.ChatMessageRow{
		{
			ID: 2, SenderID: chatTeacherID, SenderName: "Active Teacher", SenderRole: auth.RoleTeacher,
			Body: "Here are the slides", FilePath: "/uploads/files/" +
				"0123456789abcdef0123456789abcdef.pptx", FileName: "slides.pptx",
			FileMime: "application/vnd.openxmlformats-officedocument.presentationml.presentation",
			FileSize: 2048, CreatedAt: time.Now(),
		},
	}
	handler := newChatHandler(t, chats, &fakeFileStore{}, &recordingAlerts{})
	request := chatRequest(t, http.MethodGet, "/api/messages/conversations/5/messages", "",
		chatStudentID, auth.RoleStudent)
	request.SetPathValue("id", "5")
	rec := serve(handler.Messages(), request)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body = %q)", rec.Code, rec.Body.String())
	}
	list := items(t, rec)
	if len(list) != 1 {
		t.Fatalf("got %d messages, want 1", len(list))
	}
	wantURL := "/uploads/files/0123456789abcdef0123456789abcdef.pptx?name=slides.pptx"
	if list[0]["file_url"] != wantURL {
		t.Errorf("file_url = %v, want %q", list[0]["file_url"], wantURL)
	}
	if list[0]["sender_name"] != "Active Teacher" || list[0]["sender_role"] != auth.RoleTeacher {
		t.Errorf("message = %v, want the sender name and role", list[0])
	}
	if len(chats.reads) != 1 || chats.reads[0] != 5 {
		t.Errorf("reads = %v, want the conversation marked read", chats.reads)
	}
}

func TestConversationMessagesHideWhatIsNotVisible(t *testing.T) {
	silenceLog(t)
	tests := []struct {
		name       string
		pathValue  string
		accessErr  error
		wantStatus int
		wantError  string
	}{
		{
			name:       "a conversation somebody else owns",
			pathValue:  "5",
			accessErr:  store.ErrNotFound,
			wantStatus: http.StatusNotFound,
			wantError:  MessageConversationMissing,
		},
		{name: "not a number", pathValue: "abc", wantStatus: http.StatusBadRequest, wantError: MessageInvalidID},
		{name: "zero", pathValue: "0", wantStatus: http.StatusBadRequest, wantError: MessageInvalidID},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			chats := newChatStore()
			chats.accessErr = tc.accessErr
			handler := newChatHandler(t, chats, &fakeFileStore{}, &recordingAlerts{})
			request := chatRequest(t, http.MethodGet, "/api/messages/conversations/x/messages", "",
				chatStudentID, auth.RoleStudent)
			request.SetPathValue("id", tc.pathValue)
			rec := serve(handler.Messages(), request)
			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d (body = %q)", rec.Code, tc.wantStatus, rec.Body.String())
			}
			if got := errorMessage(t, rec); got != tc.wantError {
				t.Errorf("error = %q, want %q", got, tc.wantError)
			}
		})
	}
}

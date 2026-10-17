package handler

import (
	"context"
	"github.com/raHI-79/250-project/backend/internal/auth"
	"github.com/raHI-79/250-project/backend/internal/store"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

const (
	plainStudentID = int64(12)
	messageBatch   = "CSE-23"
)

type fakeMessageStore struct {
	senders      map[int64]store.Sender
	valid        map[int64]string
	replyTarget  store.MessageTarget
	replyErr     error
	created      []store.NewMessage
	messages     []store.MessageRow
	crOptions    []store.CROption
	students     []store.StudentOption
	askedUser    int64
	askedBatch   string
	askedSection string
	askedSearch  string
	askedKinds   []string
	err          error
}

func (f *fakeMessageStore) GetMessageSender(ctx context.Context, userID int64) (store.Sender, error) {
	sender, ok := f.senders[userID]
	if !ok {
		return store.Sender{}, store.ErrNotFound
	}
	return sender, nil
}

func (f *fakeMessageStore) FindMessageTarget(ctx context.Context, userID int64, kind string) (store.MessageTarget, error) {
	f.askedKinds = append(f.askedKinds, kind)
	if f.valid[userID] != kind {
		return store.MessageTarget{}, store.ErrNotFound
	}
	return store.MessageTarget{UserID: userID, Name: "Target", Batch: messageBatch, Section: "A"}, nil
}

func (f *fakeMessageStore) FindReplyTarget(ctx context.Context, messageID, teacherUserID int64) (store.MessageTarget, error) {
	if f.replyErr != nil {
		return store.MessageTarget{}, f.replyErr
	}
	return f.replyTarget, nil
}

func (f *fakeMessageStore) CreateMessage(ctx context.Context, message store.NewMessage) (store.MessageRow, error) {
	f.created = append(f.created, message)
	if f.err != nil {
		return store.MessageRow{}, f.err
	}
	return store.MessageRow{
		ID:            9,
		SenderID:      message.SenderID,
		SenderRole:    message.SenderRole,
		SenderName:    message.SenderName,
		TargetType:    message.TargetType,
		TargetBatch:   message.TargetBatch,
		TargetSection: message.TargetSection,
		TargetUserID:  message.TargetUserID,
		Content:       message.Content,
		FilePath:      message.FilePath,
		ReplyToID:     message.ReplyToID,
		CreatedAt:     time.Now(),
	}, nil
}

func (f *fakeMessageStore) ListCRInbox(ctx context.Context, crUserID int64) ([]store.MessageRow, error) {
	f.askedUser = crUserID
	return f.messages, f.err
}

func (f *fakeMessageStore) ListTeacherInbox(ctx context.Context, userID int64) ([]store.MessageRow, error) {
	f.askedUser = userID
	return f.messages, f.err
}

func (f *fakeMessageStore) ListSentMessages(ctx context.Context, senderID int64) ([]store.MessageRow, error) {
	f.askedUser = senderID
	return f.messages, f.err
}

func (f *fakeMessageStore) ListStudentMessages(ctx context.Context, userID int64, batch, section string) ([]store.MessageRow, error) {
	f.askedUser, f.askedBatch, f.askedSection = userID, batch, section
	return f.messages, f.err
}

func (f *fakeMessageStore) ListCROptions(ctx context.Context, batch string) ([]store.CROption, error) {
	f.askedBatch = batch
	return f.crOptions, f.err
}

func (f *fakeMessageStore) SearchStudents(ctx context.Context, search string) ([]store.StudentOption, error) {
	f.askedSearch = search
	return f.students, f.err
}

type mailAlerts struct {
	directID int64
	role     string
	batch    string
	section  string
	title    string
	direct   int
	casts    int
}

func (m *mailAlerts) NotifyUser(ctx context.Context, userID int64, kind, title, body string) error {
	m.directID, m.title = userID, title
	m.direct++
	return nil
}

func (m *mailAlerts) NotifyBroadcast(ctx context.Context, role, batch, section, kind, title, body string) error {
	m.role, m.batch, m.section, m.title = role, batch, section, title
	m.casts++
	return nil
}

func messageSenders() map[int64]store.Sender {
	return map[int64]store.Sender{
		noticeCRID:    {UserID: noticeCRID, Role: auth.RoleCR, Name: "Board CR", Batch: messageBatch, Section: "A", IsCR: true},
		teacherUserID: {UserID: teacherUserID, Role: auth.RoleTeacher, Name: "Board Teacher"},
		plainStudentID: {UserID: plainStudentID, Role: auth.RoleStudent, Name: "Board Student",
			Batch: messageBatch, Section: "B"},
	}
}

func TestMessageCRSend(t *testing.T) {
	tests := []struct {
		name          string
		senderID      int64
		fields        map[string]string
		filename      string
		wantStatus    int
		wantType      string
		wantBatch     string
		wantDirect    int64
		wantBroadcast bool
		wantError     string
	}{
		{
			name:          "to the own batch",
			senderID:      noticeCRID,
			fields:        map[string]string{"target": "own_batch", "content": "Bring the lab manual."},
			wantStatus:    http.StatusCreated,
			wantType:      store.TargetBatch,
			wantBatch:     messageBatch,
			wantBroadcast: true,
		},
		{
			name:          "to another batch",
			senderID:      noticeCRID,
			fields:        map[string]string{"target": "batch", "batch": "cse-22", "content": "Shared lab moved."},
			wantStatus:    http.StatusCreated,
			wantType:      store.TargetBatch,
			wantBatch:     otherBatch,
			wantBroadcast: true,
		},
		{
			name:       "to a teacher by user id",
			senderID:   noticeCRID,
			fields:     map[string]string{"target": "teacher", "teacher_user_id": "51", "content": "May we move the class?"},
			wantStatus: http.StatusCreated,
			wantType:   store.TargetUser,
			wantDirect: teacherUserID,
		},
		{
			name:          "with a pdf and no text",
			senderID:      noticeCRID,
			fields:        map[string]string{"target": "own_batch"},
			filename:      "notes.pdf",
			wantStatus:    http.StatusCreated,
			wantType:      store.TargetBatch,
			wantBatch:     messageBatch,
			wantBroadcast: true,
		},
		{
			name:       "with the teachers table id instead of the user id",
			senderID:   noticeCRID,
			fields:     map[string]string{"target": "teacher", "teacher_user_id": "7", "content": "Hello"},
			wantStatus: http.StatusBadRequest,
			wantError:  MessageUnknownTeacher,
		},
		{
			name:       "without a teacher id",
			senderID:   noticeCRID,
			fields:     map[string]string{"target": "teacher", "content": "Hello"},
			wantStatus: http.StatusBadRequest,
			wantError:  MessageInvalidID,
		},
		{
			name:       "to the own batch code typed out",
			senderID:   noticeCRID,
			fields:     map[string]string{"target": "batch", "batch": "CSE-23", "content": "Hello"},
			wantStatus: http.StatusBadRequest,
			wantError:  MessageOwnBatchMessage,
		},
		{
			name:       "without content and without a file",
			senderID:   noticeCRID,
			fields:     map[string]string{"target": "own_batch", "content": "   "},
			wantStatus: http.StatusBadRequest,
			wantError:  "Content is required when no file is attached.",
		},
		{
			name:       "with content over the limit",
			senderID:   noticeCRID,
			fields:     map[string]string{"target": "own_batch", "content": strings.Repeat("a", 2001)},
			wantStatus: http.StatusBadRequest,
			wantError:  "Content must be at most 2000 characters long.",
		},
		{
			name:       "with an unknown target",
			senderID:   noticeCRID,
			fields:     map[string]string{"target": "admin", "content": "Hello"},
			wantStatus: http.StatusBadRequest,
			wantError:  "Target must be one of own_batch, batch, teacher.",
		},
		{
			name:       "as a student who is not a class representative",
			senderID:   plainStudentID,
			fields:     map[string]string{"target": "own_batch", "content": "Hello"},
			wantStatus: http.StatusForbidden,
			wantError:  MessageNotACR,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			messages := &fakeMessageStore{senders: messageSenders(), valid: map[int64]string{teacherUserID: store.KindTeacher}}
			alerts := &mailAlerts{}
			files := &fakeFileStore{}
			if tc.filename != "" {
				files.path = savedFile
			}
			req := noticeRequest(t, "/api/cr/messages", tc.senderID, auth.RoleCR, tc.fields, tc.filename)
			rec := httptest.NewRecorder()
			NewMessage(messages, files, alerts).CRSend()(rec, req)
			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d (body = %q)", rec.Code, tc.wantStatus, rec.Body.String())
			}
			if tc.wantStatus != http.StatusCreated {
				if got := errorOf(t, rec); got != tc.wantError {
					t.Errorf("error = %q, want %q", got, tc.wantError)
				}
				if len(messages.created) != 0 {
					t.Errorf("the handler stored %d messages, want none", len(messages.created))
				}
				return
			}
			sent := messages.created[0]
			if sent.TargetType != tc.wantType || sent.TargetBatch != tc.wantBatch || sent.TargetUserID != tc.wantDirect {
				t.Errorf("stored message = %+v, want %s to %q/%d", sent, tc.wantType, tc.wantBatch, tc.wantDirect)
			}
			if sent.SenderRole != auth.RoleCR || sent.SenderName != "Board CR" || sent.SenderID != noticeCRID {
				t.Errorf("sender = %q/%q/%d, want the class representative", sent.SenderRole, sent.SenderName, sent.SenderID)
			}
			if tc.filename != "" && sent.FilePath != savedFile {
				t.Errorf("file_path = %q, want %q", sent.FilePath, savedFile)
			}
			if tc.wantBroadcast {
				if alerts.casts != 1 || alerts.role != auth.RoleStudent || alerts.batch != tc.wantBatch {
					t.Errorf("broadcast = %d to %q/%q, want one to the students of %q",
						alerts.casts, alerts.role, alerts.batch, tc.wantBatch)
				}
				return
			}
			if alerts.direct != 1 || alerts.directID != tc.wantDirect {
				t.Errorf("direct notifications = %d to %d, want one to %d", alerts.direct, alerts.directID, tc.wantDirect)
			}
		})
	}
}

func TestMessageTeacherSend(t *testing.T) {
	tests := []struct {
		name          string
		fields        map[string]string
		valid         map[int64]string
		replyErr      error
		wantStatus    int
		wantType      string
		wantBatch     string
		wantSection   string
		wantDirect    int64
		wantReplyTo   int64
		wantBroadcast bool
		wantError     string
	}{
		{
			name:       "to a class representative",
			fields:     map[string]string{"target": "cr", "target_user_id": "31", "content": "Approved."},
			valid:      map[int64]string{noticeCRID: store.KindCR},
			wantStatus: http.StatusCreated,
			wantType:   store.TargetUser,
			wantDirect: noticeCRID,
		},
		{
			name:          "to a whole batch",
			fields:        map[string]string{"target": "batch", "batch": "cse-23", "content": "Exam on Sunday."},
			wantStatus:    http.StatusCreated,
			wantType:      store.TargetBatch,
			wantBatch:     messageBatch,
			wantBroadcast: true,
		},
		{
			name:          "to one section",
			fields:        map[string]string{"target": "batch", "batch": "CSE-23", "section": "b", "content": "Lab at 2pm."},
			wantStatus:    http.StatusCreated,
			wantType:      store.TargetSection,
			wantBatch:     messageBatch,
			wantSection:   "B",
			wantBroadcast: true,
		},
		{
			name:       "to one student",
			fields:     map[string]string{"target": "student", "target_user_id": "12", "content": "See me after class."},
			valid:      map[int64]string{plainStudentID: store.KindStudent},
			wantStatus: http.StatusCreated,
			wantType:   store.TargetUser,
			wantDirect: plainStudentID,
		},
		{
			name:        "as a reply to a message from a class representative",
			fields:      map[string]string{"reply_to_id": "4", "content": "Yes, go ahead."},
			wantStatus:  http.StatusCreated,
			wantType:    store.TargetUser,
			wantDirect:  noticeCRID,
			wantReplyTo: 4,
		},
		{
			name:       "as a reply to a message that is not theirs",
			fields:     map[string]string{"reply_to_id": "4", "content": "Yes."},
			replyErr:   store.ErrNotFound,
			wantStatus: http.StatusBadRequest,
			wantError:  MessageUnknownReply,
		},
		{
			name:       "with a broken reply id",
			fields:     map[string]string{"reply_to_id": "abc", "content": "Yes."},
			wantStatus: http.StatusBadRequest,
			wantError:  MessageInvalidID,
		},
		{
			name:       "to a class representative id that is a plain student",
			fields:     map[string]string{"target": "cr", "target_user_id": "12", "content": "Hello"},
			valid:      map[int64]string{plainStudentID: store.KindStudent},
			wantStatus: http.StatusBadRequest,
			wantError:  MessageUnknownCR,
		},
		{
			name:       "to a student who is not active",
			fields:     map[string]string{"target": "student", "target_user_id": "99", "content": "Hello"},
			wantStatus: http.StatusBadRequest,
			wantError:  MessageUnknownStudent,
		},
		{
			name:       "with a broken batch",
			fields:     map[string]string{"target": "batch", "batch": "23", "content": "Hello"},
			wantStatus: http.StatusBadRequest,
			wantError:  "Batch must look like CSE-23.",
		},
		{
			name:       "with an unknown target",
			fields:     map[string]string{"target": "own_batch", "content": "Hello"},
			wantStatus: http.StatusBadRequest,
			wantError:  "Target must be one of cr, batch, student.",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			messages := &fakeMessageStore{
				senders:     messageSenders(),
				valid:       tc.valid,
				replyTarget: store.MessageTarget{UserID: noticeCRID, Name: "Board CR", Batch: messageBatch, Section: "A"},
				replyErr:    tc.replyErr,
			}
			alerts := &mailAlerts{}
			req := noticeRequest(t, "/api/teacher/messages", teacherUserID, auth.RoleTeacher, tc.fields, "")
			rec := httptest.NewRecorder()
			NewMessage(messages, &fakeFileStore{}, alerts).TeacherSend()(rec, req)
			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d (body = %q)", rec.Code, tc.wantStatus, rec.Body.String())
			}
			if tc.wantStatus != http.StatusCreated {
				if got := errorOf(t, rec); got != tc.wantError {
					t.Errorf("error = %q, want %q", got, tc.wantError)
				}
				return
			}
			sent := messages.created[0]
			if sent.TargetType != tc.wantType || sent.TargetBatch != tc.wantBatch || sent.TargetSection != tc.wantSection {
				t.Errorf("stored message = %+v, want %s to %q/%q", sent, tc.wantType, tc.wantBatch, tc.wantSection)
			}
			if sent.TargetUserID != tc.wantDirect || sent.ReplyToID != tc.wantReplyTo {
				t.Errorf("target user = %d and reply = %d, want %d and %d",
					sent.TargetUserID, sent.ReplyToID, tc.wantDirect, tc.wantReplyTo)
			}
			if sent.SenderRole != auth.RoleTeacher || sent.SenderName != "Board Teacher" {
				t.Errorf("sender = %q/%q, want the teacher", sent.SenderRole, sent.SenderName)
			}
			if tc.wantBroadcast {
				if alerts.casts != 1 || alerts.batch != tc.wantBatch || alerts.section != tc.wantSection {
					t.Errorf("broadcast = %d to %q/%q, want one to %q/%q",
						alerts.casts, alerts.batch, alerts.section, tc.wantBatch, tc.wantSection)
				}
				return
			}
			if alerts.direct != 1 || alerts.directID != tc.wantDirect {
				t.Errorf("direct notifications = %d to %d, want one to %d", alerts.direct, alerts.directID, tc.wantDirect)
			}
		})
	}
}

func TestMessageInboxes(t *testing.T) {
	rows := []store.MessageRow{
		{ID: 2, SenderName: "Board Teacher", SenderRole: auth.RoleTeacher, TargetType: store.TargetUser,
			TargetUserID: noticeCRID, Content: "Approved.", ReplyToID: 1, CreatedAt: time.Now()},
		{ID: 1, SenderName: "Board CR", SenderRole: auth.RoleCR, SenderBatch: messageBatch, SenderSection: "A",
			TargetType: store.TargetUser, TargetUserID: teacherUserID, Content: "May we move the class?",
			FilePath: savedFile, CreatedAt: time.Now()},
	}
	tests := []struct {
		name         string
		userID       int64
		role         string
		path         string
		pick         func(*Message) http.HandlerFunc
		wantUser     int64
		wantBatch    string
		wantSection  string
		wantStatus   int
		wantErrorMsg string
	}{
		{
			name:     "the inbox of a class representative",
			userID:   noticeCRID,
			role:     auth.RoleCR,
			path:     "/api/cr/messages/inbox",
			pick:     func(m *Message) http.HandlerFunc { return m.CRInbox() },
			wantUser: noticeCRID, wantStatus: http.StatusOK,
		},
		{
			name:     "what a class representative sent",
			userID:   noticeCRID,
			role:     auth.RoleCR,
			path:     "/api/cr/messages/sent",
			pick:     func(m *Message) http.HandlerFunc { return m.CRSent() },
			wantUser: noticeCRID, wantStatus: http.StatusOK,
		},
		{
			name:     "the inbox of a teacher",
			userID:   teacherUserID,
			role:     auth.RoleTeacher,
			path:     "/api/teacher/messages/inbox",
			pick:     func(m *Message) http.HandlerFunc { return m.TeacherInbox() },
			wantUser: teacherUserID, wantStatus: http.StatusOK,
		},
		{
			name:     "what a teacher sent",
			userID:   teacherUserID,
			role:     auth.RoleTeacher,
			path:     "/api/teacher/messages/sent",
			pick:     func(m *Message) http.HandlerFunc { return m.TeacherSent() },
			wantUser: teacherUserID, wantStatus: http.StatusOK,
		},
		{
			name:      "the read only inbox of a student",
			userID:    plainStudentID,
			role:      auth.RoleStudent,
			path:      "/api/students/messages",
			pick:      func(m *Message) http.HandlerFunc { return m.StudentInbox() },
			wantUser:  plainStudentID,
			wantBatch: messageBatch, wantSection: "B", wantStatus: http.StatusOK,
		},
		{
			name:         "the inbox of a teacher asked by a class representative",
			userID:       noticeCRID,
			role:         auth.RoleCR,
			path:         "/api/teacher/messages/inbox",
			pick:         func(m *Message) http.HandlerFunc { return m.TeacherInbox() },
			wantStatus:   http.StatusForbidden,
			wantErrorMsg: MessageNotATeacher,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			messages := &fakeMessageStore{senders: messageSenders(), messages: rows}
			req := jsonRequest(t, http.MethodGet, tc.path, "", tc.userID, tc.role)
			rec := httptest.NewRecorder()
			tc.pick(NewMessage(messages, &fakeFileStore{}, &mailAlerts{}))(rec, req)
			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d (body = %q)", rec.Code, tc.wantStatus, rec.Body.String())
			}
			if tc.wantStatus != http.StatusOK {
				if got := errorOf(t, rec); got != tc.wantErrorMsg {
					t.Errorf("error = %q, want %q", got, tc.wantErrorMsg)
				}
				return
			}
			if messages.askedUser != tc.wantUser || messages.askedBatch != tc.wantBatch || messages.askedSection != tc.wantSection {
				t.Errorf("asked for %d %q/%q, want %d %q/%q", messages.askedUser, messages.askedBatch,
					messages.askedSection, tc.wantUser, tc.wantBatch, tc.wantSection)
			}
			items := noticeItems(t, rec)
			if len(items) != len(rows) {
				t.Fatalf("got %d items, want %d", len(items), len(rows))
			}
			if items[0]["file_url"] != nil || items[0]["reply_to_id"] != float64(1) {
				t.Errorf("first item = %v, want no file and a reply id", items[0])
			}
			if items[1]["file_url"] != savedFile || items[1]["reply_to_id"] != nil {
				t.Errorf("second item = %v, want the attached file and no reply id", items[1])
			}
			if items[1]["sender_batch"] != messageBatch || items[1]["sender_section"] != "A" {
				t.Errorf("second item = %v, want the batch and section of the sender", items[1])
			}
		})
	}
}

func TestMessageLookups(t *testing.T) {
	messages := &fakeMessageStore{
		senders:   messageSenders(),
		crOptions: []store.CROption{{UserID: noticeCRID, FullName: "Board CR", Batch: messageBatch, Section: "A"}},
		students:  []store.StudentOption{{UserID: plainStudentID, FullName: "Board Student", RegNo: "2023331911", Batch: messageBatch, Section: "B"}},
	}
	handler := NewMessage(messages, &fakeFileStore{}, &mailAlerts{})
	tests := []struct {
		name       string
		path       string
		serve      http.HandlerFunc
		wantStatus int
		wantBatch  string
		wantSearch string
		wantField  string
		wantValue  any
		wantError  string
	}{
		{
			name:      "every class representative",
			path:      "/api/teacher/lookups/crs",
			serve:     handler.CRLookup(),
			wantField: "user_id", wantValue: float64(noticeCRID), wantStatus: http.StatusOK,
		},
		{
			name:      "the class representatives of one batch",
			path:      "/api/teacher/lookups/crs?batch=cse-23",
			serve:     handler.CRLookup(),
			wantBatch: messageBatch,
			wantField: "section", wantValue: "A", wantStatus: http.StatusOK,
		},
		{
			name:       "a broken batch filter",
			path:       "/api/teacher/lookups/crs?batch=23",
			serve:      handler.CRLookup(),
			wantStatus: http.StatusBadRequest,
			wantError:  "batch must look like CSE-23.",
		},
		{
			name:       "a student search",
			path:       "/api/teacher/lookups/students?search=board",
			serve:      handler.StudentLookup(),
			wantSearch: "board",
			wantField:  "reg_no", wantValue: "2023331911", wantStatus: http.StatusOK,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			messages.askedBatch, messages.askedSearch = "", ""
			req := jsonRequest(t, http.MethodGet, tc.path, "", teacherUserID, auth.RoleTeacher)
			rec := httptest.NewRecorder()
			tc.serve(rec, req)
			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d (body = %q)", rec.Code, tc.wantStatus, rec.Body.String())
			}
			if tc.wantStatus != http.StatusOK {
				if got := errorOf(t, rec); got != tc.wantError {
					t.Errorf("error = %q, want %q", got, tc.wantError)
				}
				return
			}
			if messages.askedBatch != tc.wantBatch || messages.askedSearch != tc.wantSearch {
				t.Errorf("asked for %q/%q, want %q/%q", messages.askedBatch, messages.askedSearch, tc.wantBatch, tc.wantSearch)
			}
			items := noticeItems(t, rec)
			if len(items) != 1 || items[0][tc.wantField] != tc.wantValue {
				t.Fatalf("items = %v, want one row with %s = %v", items, tc.wantField, tc.wantValue)
			}
		})
	}
}

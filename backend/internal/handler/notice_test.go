package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/raHI-79/250-project/backend/internal/auth"
	"github.com/raHI-79/250-project/backend/internal/middleware"
	"github.com/raHI-79/250-project/backend/internal/store"
	"github.com/raHI-79/250-project/backend/internal/upload"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

const (
	noticeAdminID = int64(44)
	noticeCRID    = int64(31)
	noticeBatch   = "CSE-23"
	otherBatch    = "CSE-22"
	savedFile     = "/uploads/" + savedName
	savedName     = "0123456789abcdef0123456789abcdef.pdf"
)

func jsonRequest(t *testing.T, method, path, body string, userID int64, role string) *http.Request {
	t.Helper()
	var req *http.Request
	if body == "" {
		req = httptest.NewRequest(method, path, nil)
	} else {
		req = httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	}
	return req.WithContext(middleware.WithUser(req.Context(), middleware.User{ID: userID, Role: role}))
}

type fakeNoticeStore struct {
	profile      store.StudentProfile
	profileErr   error
	created      []store.NewNotice
	notices      []store.NoticeRow
	deletedID    int64
	deletedFile  string
	deleteErr    error
	editedID     int64
	editedTitle  string
	editedSource string
	editedOwner  int64
	editErr      error
	askedBatch   string
	askedSection string
	askedSource  string
	askedScope   string
	askedOther   bool
	askedCreator int64
	askedUser    int64
	err          error
}

func (f *fakeNoticeStore) GetStudentProfile(ctx context.Context, userID int64) (store.StudentProfile, error) {
	if f.profileErr != nil {
		return store.StudentProfile{}, f.profileErr
	}
	return f.profile, nil
}

func (f *fakeNoticeStore) CreateNotice(ctx context.Context, notice store.NewNotice) (store.NoticeRow, error) {
	f.created = append(f.created, notice)
	if f.err != nil {
		return store.NoticeRow{}, f.err
	}
	return store.NoticeRow{
		ID:            7,
		Source:        notice.Source,
		Title:         notice.Title,
		Description:   notice.Description,
		Batch:         notice.Batch,
		Section:       notice.Section,
		FilePath:      notice.FilePath,
		CreatedByName: "Poster",
		CreatedAt:     time.Now(),
	}, nil
}

func (f *fakeNoticeStore) ListAdminNotices(ctx context.Context) ([]store.NoticeRow, error) {
	return f.notices, f.err
}

func (f *fakeNoticeStore) ListTeacherNotices(ctx context.Context, createdBy int64) ([]store.NoticeRow, error) {
	f.askedUser = createdBy
	return f.notices, f.err
}

func (f *fakeNoticeStore) ListStudentNotices(ctx context.Context, batch, section, source, scope string) ([]store.NoticeRow, error) {
	f.askedBatch = batch
	f.askedSection = section
	f.askedSource = source
	f.askedScope = scope
	return f.notices, f.err
}

func (f *fakeNoticeStore) DeleteOwnNotice(ctx context.Context, noticeID int64, source string, createdBy int64) (string, error) {
	f.deletedID = noticeID
	f.askedUser = createdBy
	return f.deletedFile, f.deleteErr
}

func (f *fakeNoticeStore) ListStudentAdminNotices(ctx context.Context, batch string) ([]store.NoticeRow, error) {
	f.askedBatch = batch
	return f.notices, f.err
}

func (f *fakeNoticeStore) ListCRNotices(ctx context.Context, createdBy int64) ([]store.NoticeRow, error) {
	f.askedCreator = createdBy
	return f.notices, f.err
}

func (f *fakeNoticeStore) ListBatchCRNotices(ctx context.Context, batch string, otherBatches bool) ([]store.NoticeRow, error) {
	f.askedBatch, f.askedOther = batch, otherBatches
	return f.notices, f.err
}

func (f *fakeNoticeStore) DeleteAnyNotice(ctx context.Context, noticeID int64) (string, error) {
	return f.DeleteAdminNotice(ctx, noticeID)
}

func (f *fakeNoticeStore) EditNotice(ctx context.Context, noticeID int64, title, description,
	linkURL, source string, createdBy int64) error {
	f.editedID = noticeID
	f.editedTitle = title
	f.editedSource = source
	f.editedOwner = createdBy
	return f.editErr
}

func (f *fakeNoticeStore) DeleteAdminNotice(ctx context.Context, noticeID int64) (string, error) {
	f.deletedID = noticeID
	if f.deleteErr != nil {
		return "", f.deleteErr
	}
	return f.deletedFile, nil
}

type fakeFileStore struct {
	path      string
	name      string
	mime      string
	size      int64
	saveErr   error
	parseErr  error
	deleted   []string
	deleteErr error
}

func (f *fakeFileStore) Parse(r *http.Request) error {
	if f.parseErr != nil {
		return f.parseErr
	}
	if err := r.ParseMultipartForm(1 << 20); err != nil {
		return upload.ErrBadForm
	}
	return nil
}

func (f *fakeFileStore) Save(r *http.Request) (upload.Attachment, error) {
	if err := f.Parse(r); err != nil {
		return upload.Attachment{}, err
	}
	if f.saveErr != nil {
		return upload.Attachment{}, f.saveErr
	}
	if f.path == "" {
		return upload.Attachment{}, nil
	}
	name := f.name
	if name == "" {
		name = "notice.pdf"
	}
	mime := f.mime
	if mime == "" {
		mime = "application/pdf"
	}
	size := f.size
	if size == 0 {
		size = 1024
	}
	return upload.Attachment{Path: f.path, Name: name, Mime: mime, Size: size}, nil
}

func (f *fakeFileStore) Delete(filePath string) error {
	f.deleted = append(f.deleted, filePath)
	return f.deleteErr
}

type recordingAlerts struct {
	role    string
	batch   string
	section string
	title   string
	link    store.NotificationLink
	calls   int
	users   int
	pushed  int
}

func (r *recordingAlerts) NotifyUser(ctx context.Context, userID int64, kind, title, body string,
	link store.NotificationLink) error {
	r.title, r.link = title, link
	r.users++
	return nil
}

func (r *recordingAlerts) PushUser(ctx context.Context, userID int64, kind, title, body string,
	link store.NotificationLink) error {
	r.title, r.link = title, link
	r.users++
	r.pushed++
	return nil
}

func (r *recordingAlerts) NotifyBroadcast(ctx context.Context, role, batch, section, kind, title, body string,
	link store.NotificationLink) error {
	r.role, r.batch, r.section, r.title, r.link = role, batch, section, title, link
	r.calls++
	return nil
}

func noticeForm(t *testing.T, fields map[string]string, filename, content string) (string, *bytes.Buffer) {
	t.Helper()
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	for name, value := range fields {
		if err := writer.WriteField(name, value); err != nil {
			t.Fatalf("could not write the field %q: %v", name, err)
		}
	}
	if filename != "" {
		part, err := writer.CreateFormFile("file", filename)
		if err != nil {
			t.Fatalf("could not create the file part: %v", err)
		}
		if _, err := part.Write([]byte(content)); err != nil {
			t.Fatalf("could not write the file part: %v", err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("could not close the multipart writer: %v", err)
	}
	return writer.FormDataContentType(), body
}

func noticeRequest(t *testing.T, path string, userID int64, role string, fields map[string]string, filename string) *http.Request {
	t.Helper()
	contentType, body := noticeForm(t, fields, filename, "%PDF-1.7")
	req := httptest.NewRequest(http.MethodPost, path, body)
	req.Header.Set("Content-Type", contentType)
	return req.WithContext(middleware.WithUser(req.Context(), middleware.User{ID: userID, Role: role}))
}

func noticeItems(t *testing.T, rec *httptest.ResponseRecorder) []map[string]any {
	t.Helper()
	var payload struct {
		Items []map[string]any `json:"items"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("body is not valid JSON: %v (body = %q)", err, rec.Body.String())
	}
	return payload.Items
}

func TestNoticeAdminCreate(t *testing.T) {
	tests := []struct {
		name       string
		fields     map[string]string
		filename   string
		files      *fakeFileStore
		wantStatus int
		wantBatch  string
		wantFile   bool
		wantError  string
	}{
		{
			name:       "every batch without a file",
			fields:     map[string]string{"title": "Semester fees", "description": "Pay by Sunday."},
			files:      &fakeFileStore{},
			wantStatus: http.StatusCreated,
		},
		{
			name:       "one batch with a file",
			fields:     map[string]string{"title": "Exam routine", "description": "Final exams.", "batch": "cse-23"},
			filename:   "routine.pdf",
			files:      &fakeFileStore{path: savedFile},
			wantStatus: http.StatusCreated,
			wantBatch:  noticeBatch,
			wantFile:   true,
		},
		{
			name:       "without a title",
			fields:     map[string]string{"description": "No title here."},
			files:      &fakeFileStore{},
			wantStatus: http.StatusBadRequest,
			wantError:  "Title is required.",
		},
		{
			name:       "with a broken batch",
			fields:     map[string]string{"title": "Exam routine", "batch": "23"},
			files:      &fakeFileStore{},
			wantStatus: http.StatusBadRequest,
			wantError:  "Batch must look like CSE-23.",
		},
		{
			name:       "with a file that is not a pdf",
			fields:     map[string]string{"title": "Exam routine"},
			filename:   "routine.pdf",
			files:      &fakeFileStore{saveErr: upload.ErrNotAllowedType},
			wantStatus: http.StatusBadRequest,
			wantError:  MessageFileNotAllowed,
		},
		{
			name:       "with a file over the limit",
			fields:     map[string]string{"title": "Exam routine"},
			filename:   "routine.pdf",
			files:      &fakeFileStore{saveErr: upload.ErrTooLarge},
			wantStatus: http.StatusRequestEntityTooLarge,
			wantError:  MessageFileTooLarge,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			notices := &fakeNoticeStore{}
			alerts := &recordingAlerts{}
			rec := httptest.NewRecorder()
			req := noticeRequest(t, "/api/admin/notices", noticeAdminID, auth.RoleAdmin, tc.fields, tc.filename)
			NewNotice(notices, tc.files, alerts).AdminCreate()(rec, req)
			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d (body = %q)", rec.Code, tc.wantStatus, rec.Body.String())
			}
			if tc.wantStatus != http.StatusCreated {
				if got := errorOf(t, rec); got != tc.wantError {
					t.Errorf("error = %q, want %q", got, tc.wantError)
				}
				if len(notices.created) != 0 {
					t.Errorf("the handler saved %d notices, want none", len(notices.created))
				}
				return
			}
			if len(notices.created) != 1 {
				t.Fatalf("the handler saved %d notices, want 1", len(notices.created))
			}
			saved := notices.created[0]
			if saved.Source != store.SourceAdmin || saved.Batch != tc.wantBatch {
				t.Errorf("saved notice = %+v, want source admin and batch %q", saved, tc.wantBatch)
			}
			if tc.wantFile && saved.FilePath != savedFile {
				t.Errorf("file_path = %q, want %q", saved.FilePath, savedFile)
			}
			var payload map[string]any
			if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
				t.Fatalf("body is not valid JSON: %v", err)
			}
			if tc.wantFile {
				if payload["file_url"] != savedFile {
					t.Errorf("file_url = %v, want %q", payload["file_url"], savedFile)
				}
			} else if payload["file_url"] != nil {
				t.Errorf("file_url = %v, want null", payload["file_url"])
			}
			if alerts.calls != 1 || alerts.role != auth.RoleStudent || alerts.batch != tc.wantBatch {
				t.Errorf("broadcast = %d call(s) to %q/%q, want 1 call to student/%q", alerts.calls, alerts.role, alerts.batch, tc.wantBatch)
			}
		})
	}
}

func TestNoticeAdminDelete(t *testing.T) {
	tests := []struct {
		name        string
		id          string
		store       *fakeNoticeStore
		wantStatus  int
		wantDeleted string
	}{
		{
			name:        "a notice with a file",
			id:          "7",
			store:       &fakeNoticeStore{deletedFile: savedFile},
			wantStatus:  http.StatusNoContent,
			wantDeleted: savedFile,
		},
		{name: "a notice without a file", id: "7", store: &fakeNoticeStore{}, wantStatus: http.StatusNoContent},
		{name: "an unknown notice", id: "7", store: &fakeNoticeStore{deleteErr: store.ErrNotFound}, wantStatus: http.StatusNotFound},
		{name: "a broken id", id: "abc", store: &fakeNoticeStore{}, wantStatus: http.StatusBadRequest},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			files := &fakeFileStore{}
			req := httptest.NewRequest(http.MethodDelete, "/api/admin/notices/"+tc.id, nil)
			req.SetPathValue("id", tc.id)
			req = req.WithContext(middleware.WithUser(req.Context(), middleware.User{ID: noticeAdminID, Role: auth.RoleAdmin}))
			rec := httptest.NewRecorder()
			NewNotice(tc.store, files, &recordingAlerts{}).AdminDelete()(rec, req)
			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d (body = %q)", rec.Code, tc.wantStatus, rec.Body.String())
			}
			if tc.wantDeleted == "" {
				if len(files.deleted) != 0 {
					t.Errorf("the handler deleted %v, want no file", files.deleted)
				}
				return
			}
			if len(files.deleted) != 1 || files.deleted[0] != tc.wantDeleted {
				t.Errorf("deleted files = %v, want [%q]", files.deleted, tc.wantDeleted)
			}
		})
	}
}

func TestNoticeCRCreate(t *testing.T) {
	tests := []struct {
		name       string
		target     string
		isCR       bool
		wantStatus int
		wantBatch  string
		wantError  string
	}{
		{name: "own batch", target: "own", isCR: true, wantStatus: http.StatusCreated, wantBatch: noticeBatch},
		{name: "another batch", target: "cse-22", isCR: true, wantStatus: http.StatusCreated, wantBatch: otherBatch},
		{
			name:       "the own batch code typed out",
			target:     noticeBatch,
			isCR:       true,
			wantStatus: http.StatusBadRequest,
			wantError:  MessageOwnBatchTarget,
		},
		{
			name:       "a broken target",
			target:     "all",
			isCR:       true,
			wantStatus: http.StatusBadRequest,
			wantError:  "Target must look like CSE-23.",
		},
		{name: "no target", isCR: true, wantStatus: http.StatusBadRequest, wantError: "Target is required."},
		{name: "a student who is not a class representative", target: "own", wantStatus: http.StatusForbidden, wantError: MessageNotACR},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			notices := &fakeNoticeStore{profile: store.StudentProfile{
				UserID:  noticeCRID,
				Batch:   noticeBatch,
				Section: "A",
				IsCR:    tc.isCR,
			}}
			alerts := &recordingAlerts{}
			fields := map[string]string{"title": "Class cancelled", "description": "No class tomorrow."}
			if tc.target != "" {
				fields["target"] = tc.target
			}
			rec := httptest.NewRecorder()
			req := noticeRequest(t, "/api/cr/notices", noticeCRID, auth.RoleCR, fields, "")
			NewNotice(notices, &fakeFileStore{}, alerts).CRCreate()(rec, req)
			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d (body = %q)", rec.Code, tc.wantStatus, rec.Body.String())
			}
			if tc.wantStatus != http.StatusCreated {
				if got := errorOf(t, rec); got != tc.wantError {
					t.Errorf("error = %q, want %q", got, tc.wantError)
				}
				return
			}
			saved := notices.created[0]
			if saved.Source != store.SourceCR || saved.Batch != tc.wantBatch || saved.Section != "" {
				t.Errorf("saved notice = %+v, want source cr, batch %q and no section", saved, tc.wantBatch)
			}
			if saved.CreatedBy != noticeCRID {
				t.Errorf("created_by = %d, want %d", saved.CreatedBy, noticeCRID)
			}
			if alerts.batch != tc.wantBatch || alerts.section != "" {
				t.Errorf("broadcast target = %q/%q, want %q and the whole batch", alerts.batch, alerts.section, tc.wantBatch)
			}
		})
	}
}

func TestNoticeStudentViews(t *testing.T) {
	rows := []store.NoticeRow{
		{ID: 2, Source: store.SourceCR, Title: "Second", Batch: noticeBatch, CreatedAt: time.Now()},
		{ID: 1, Source: store.SourceCR, Title: "First", Batch: noticeBatch, FilePath: savedFile, CreatedAt: time.Now()},
	}
	tests := []struct {
		name       string
		query      string
		wantStatus int
		wantSource string
		wantScope  string
		wantError  string
	}{
		{name: "everything by default", wantStatus: http.StatusOK, wantSource: "all", wantScope: "my"},
		{name: "my batch", query: "?scope=my", wantStatus: http.StatusOK, wantSource: "all", wantScope: "my"},
		{
			name:       "the other batches",
			query:      "?scope=other",
			wantStatus: http.StatusOK,
			wantSource: "all",
			wantScope:  "other",
		},
		{
			name:       "only the administration",
			query:      "?source=admin",
			wantStatus: http.StatusOK,
			wantSource: "admin",
			wantScope:  "my",
		},
		{
			name:       "only the teachers",
			query:      "?source=teacher",
			wantStatus: http.StatusOK,
			wantSource: "teacher",
			wantScope:  "my",
		},
		{
			name:       "only the class representatives of other batches",
			query:      "?source=cr&scope=other",
			wantStatus: http.StatusOK,
			wantSource: "cr",
			wantScope:  "other",
		},
		{
			name:       "a broken scope",
			query:      "?scope=everything",
			wantStatus: http.StatusBadRequest,
			wantError:  "scope must be one of my, other.",
		},
		{
			name:       "a broken source",
			query:      "?source=everyone",
			wantStatus: http.StatusBadRequest,
			wantError:  "source must be one of all, admin, teacher, cr.",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			notices := &fakeNoticeStore{
				profile: store.StudentProfile{UserID: 12, Batch: noticeBatch, Section: "A"},
				notices: rows,
			}
			req := httptest.NewRequest(http.MethodGet, "/api/students/notices"+tc.query, nil)
			req = req.WithContext(middleware.WithUser(req.Context(), middleware.User{ID: 12, Role: auth.RoleStudent}))
			rec := httptest.NewRecorder()
			NewNotice(notices, &fakeFileStore{}, &recordingAlerts{}).StudentNotices()(rec, req)
			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d (body = %q)", rec.Code, tc.wantStatus, rec.Body.String())
			}
			if tc.wantStatus != http.StatusOK {
				if got := errorOf(t, rec); got != tc.wantError {
					t.Errorf("error = %q, want %q", got, tc.wantError)
				}
				return
			}
			if notices.askedBatch != noticeBatch || notices.askedSection != "A" {
				t.Errorf("asked for %q/%q, want %q/A", notices.askedBatch, notices.askedSection, noticeBatch)
			}
			if notices.askedSource != tc.wantSource || notices.askedScope != tc.wantScope {
				t.Errorf("asked for source %q scope %q, want %q and %q",
					notices.askedSource, notices.askedScope, tc.wantSource, tc.wantScope)
			}
			items := noticeItems(t, rec)
			if len(items) != len(rows) {
				t.Fatalf("got %d items, want %d", len(items), len(rows))
			}
			if items[0]["id"] != float64(2) || items[0]["batch"] != noticeBatch || items[0]["file_url"] != nil {
				t.Errorf("first item = %v, want the newest notice without a file", items[0])
			}
			if items[1]["file_url"] != savedFile {
				t.Errorf("second file_url = %v, want %q", items[1]["file_url"], savedFile)
			}
		})
	}
}

func TestNoticeListsUseTheCallerProfile(t *testing.T) {
	notices := &fakeNoticeStore{profile: store.StudentProfile{UserID: noticeCRID, Batch: noticeBatch, IsCR: true}}
	handler := NewNotice(notices, &fakeFileStore{}, &recordingAlerts{})
	tests := []struct {
		name   string
		path   string
		role   string
		serve  http.HandlerFunc
		verify func(t *testing.T)
	}{
		{
			name:  "the notices a class representative posted",
			path:  "/api/cr/notices",
			role:  auth.RoleCR,
			serve: handler.CRList(),
			verify: func(t *testing.T) {
				if notices.askedCreator != noticeCRID {
					t.Errorf("asked for the notices of %d, want %d", notices.askedCreator, noticeCRID)
				}
			},
		},
		{
			name:  "the administration notices of a batch",
			path:  "/api/students/notices/admin",
			role:  auth.RoleStudent,
			serve: handler.StudentNotices(),
			verify: func(t *testing.T) {
				if notices.askedBatch != noticeBatch {
					t.Errorf("asked for the notices of %q, want %q", notices.askedBatch, noticeBatch)
				}
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			req = req.WithContext(middleware.WithUser(req.Context(), middleware.User{ID: noticeCRID, Role: tc.role}))
			rec := httptest.NewRecorder()
			tc.serve(rec, req)
			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200 (body = %q)", rec.Code, rec.Body.String())
			}
			tc.verify(t)
		})
	}
}

func TestNoticeCreateRejectsABrokenForm(t *testing.T) {
	notices := &fakeNoticeStore{}
	req := httptest.NewRequest(http.MethodPost, "/api/admin/notices", strings.NewReader("{}"))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(middleware.WithUser(req.Context(), middleware.User{ID: noticeAdminID, Role: auth.RoleAdmin}))
	rec := httptest.NewRecorder()
	NewNotice(notices, &fakeFileStore{}, &recordingAlerts{}).AdminCreate()(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 (body = %q)", rec.Code, rec.Body.String())
	}
}

func errorOf(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("body is not valid JSON: %v (body = %q)", err, rec.Body.String())
	}
	text, _ := payload["error"].(string)
	return text
}

func TestNoticeEditKeepsToTheOwner(t *testing.T) {
	silenceLog(t)
	body := `{"title":"Updated title","description":"Updated body"}`
	tests := []struct {
		name       string
		handler    func(*Notice) http.HandlerFunc
		userID     int64
		role       string
		wantSource string
		wantOwner  int64
	}{
		{
			name:       "the office may edit any notice",
			handler:    func(n *Notice) http.HandlerFunc { return n.AdminEdit() },
			userID:     noticeAdminID,
			role:       auth.RoleAdmin,
			wantSource: "",
			wantOwner:  0,
		},
		{
			name:       "a teacher only edits their own",
			handler:    func(n *Notice) http.HandlerFunc { return n.TeacherEdit() },
			userID:     noticeTeacherID,
			role:       auth.RoleTeacher,
			wantSource: store.SourceTeacher,
			wantOwner:  noticeTeacherID,
		},
		{
			name:       "a class representative only edits their own",
			handler:    func(n *Notice) http.HandlerFunc { return n.CREdit() },
			userID:     noticeCRID,
			role:       auth.RoleCR,
			wantSource: store.SourceCR,
			wantOwner:  noticeCRID,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			notices := &fakeNoticeStore{}
			handler := NewNotice(notices, &fakeFileStore{}, &recordingAlerts{})
			request := jsonRequest(t, http.MethodPatch, "/api/notices/9", body, test.userID, test.role)
			request.SetPathValue("id", "9")
			rec := serve(test.handler(handler), request)
			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200 (body = %q)", rec.Code, rec.Body.String())
			}
			if notices.editedID != 9 || notices.editedTitle != "Updated title" {
				t.Errorf("edited = %d %q, want notice 9 with the new title", notices.editedID,
					notices.editedTitle)
			}
			if notices.editedSource != test.wantSource || notices.editedOwner != test.wantOwner {
				t.Errorf("source = %q owner = %d, want %q and %d", notices.editedSource,
					notices.editedOwner, test.wantSource, test.wantOwner)
			}
		})
	}
}

func TestNoticeEditRefusesABlankTitle(t *testing.T) {
	silenceLog(t)
	notices := &fakeNoticeStore{}
	handler := NewNotice(notices, &fakeFileStore{}, &recordingAlerts{})
	request := jsonRequest(t, http.MethodPatch, "/api/cr/notices/9", `{"title":"  "}`,
		noticeCRID, auth.RoleCR)
	request.SetPathValue("id", "9")
	rec := serve(handler.CREdit(), request)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	if notices.editedID != 0 {
		t.Error("the store was asked to edit a notice although the title was blank")
	}
}

func TestNoticeEditReportsAMissingNotice(t *testing.T) {
	silenceLog(t)
	notices := &fakeNoticeStore{editErr: store.ErrNotFound}
	handler := NewNotice(notices, &fakeFileStore{}, &recordingAlerts{})
	request := jsonRequest(t, http.MethodPatch, "/api/cr/notices/9",
		`{"title":"New","description":"Body"}`, noticeCRID, auth.RoleCR)
	request.SetPathValue("id", "9")
	rec := serve(handler.CREdit(), request)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
	if got := errorMessage(t, rec); got != MessageNoticeUnknown {
		t.Errorf("error = %q, want %q", got, MessageNoticeUnknown)
	}
}

func TestCRNoticeDeleteKeepsToTheOwner(t *testing.T) {
	silenceLog(t)
	notices := &fakeNoticeStore{}
	handler := NewNotice(notices, &fakeFileStore{}, &recordingAlerts{})
	request := jsonRequest(t, http.MethodDelete, "/api/cr/notices/4", "", noticeCRID, auth.RoleCR)
	request.SetPathValue("id", "4")
	rec := serve(handler.CRDelete(), request)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204 (body = %q)", rec.Code, rec.Body.String())
	}
	if notices.deletedID != 4 {
		t.Errorf("deleted = %d, want 4", notices.deletedID)
	}
}

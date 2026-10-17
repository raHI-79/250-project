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

type fakeNoticeStore struct {
	profile      store.StudentProfile
	profileErr   error
	created      []store.NewNotice
	notices      []store.NoticeRow
	deletedID    int64
	deletedFile  string
	deleteErr    error
	askedBatch   string
	askedOther   bool
	askedCreator int64
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

func (f *fakeNoticeStore) DeleteAdminNotice(ctx context.Context, noticeID int64) (string, error) {
	f.deletedID = noticeID
	if f.deleteErr != nil {
		return "", f.deleteErr
	}
	return f.deletedFile, nil
}

type fakeFileStore struct {
	path      string
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

func (f *fakeFileStore) Save(r *http.Request) (string, error) {
	if err := f.Parse(r); err != nil {
		return "", err
	}
	return f.path, f.saveErr
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
	calls   int
}

func (r *recordingAlerts) NotifyUser(ctx context.Context, userID int64, kind, title, body string) error {
	return nil
}

func (r *recordingAlerts) NotifyBroadcast(ctx context.Context, role, batch, section, kind, title, body string) error {
	r.role, r.batch, r.section, r.title = role, batch, section, title
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
			files:      &fakeFileStore{saveErr: upload.ErrNotPDF},
			wantStatus: http.StatusBadRequest,
			wantError:  MessageFileNotPDF,
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
		scope      string
		wantStatus int
		wantOther  bool
		wantError  string
	}{
		{name: "my batch by default", wantStatus: http.StatusOK},
		{name: "my batch", scope: "my", wantStatus: http.StatusOK},
		{name: "the other batches", scope: "other", wantStatus: http.StatusOK, wantOther: true},
		{name: "a broken scope", scope: "everything", wantStatus: http.StatusBadRequest, wantError: "scope must be one of my, other."},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			notices := &fakeNoticeStore{
				profile: store.StudentProfile{UserID: 12, Batch: noticeBatch, Section: "A"},
				notices: rows,
			}
			path := "/api/students/notices/cr"
			if tc.scope != "" {
				path += "?scope=" + tc.scope
			}
			req := httptest.NewRequest(http.MethodGet, path, nil)
			req = req.WithContext(middleware.WithUser(req.Context(), middleware.User{ID: 12, Role: auth.RoleStudent}))
			rec := httptest.NewRecorder()
			NewNotice(notices, &fakeFileStore{}, &recordingAlerts{}).StudentCRNotices()(rec, req)
			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d (body = %q)", rec.Code, tc.wantStatus, rec.Body.String())
			}
			if tc.wantStatus != http.StatusOK {
				if got := errorOf(t, rec); got != tc.wantError {
					t.Errorf("error = %q, want %q", got, tc.wantError)
				}
				return
			}
			if notices.askedBatch != noticeBatch || notices.askedOther != tc.wantOther {
				t.Errorf("asked for %q with other = %t, want %q with other = %t",
					notices.askedBatch, notices.askedOther, noticeBatch, tc.wantOther)
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
			serve: handler.StudentAdminNotices(),
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

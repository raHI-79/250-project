package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/raHI-79/250-project/backend/internal/auth"
	"github.com/raHI-79/250-project/backend/internal/middleware"
	"github.com/raHI-79/250-project/backend/internal/store"
)

const noticeTeacherID = int64(61)

func teacherNoticeRequest(t *testing.T, fields map[string]string) *http.Request {
	t.Helper()
	contentType, body := noticeForm(t, fields, "", "")
	request := httptest.NewRequest(http.MethodPost, "/api/teacher/notices", body)
	request.Header.Set("Content-Type", contentType)
	return request.WithContext(middleware.WithUser(request.Context(),
		middleware.User{ID: noticeTeacherID, Role: auth.RoleTeacher}))
}

func TestNoticeTeacherCreate(t *testing.T) {
	silenceLog(t)
	tests := []struct {
		name        string
		fields      map[string]string
		wantStatus  int
		wantError   string
		wantSection string
		wantAlert   string
	}{
		{
			name:       "a whole batch",
			fields:     map[string]string{"title": "Quiz on Monday", "batch": "CSE-23"},
			wantStatus: http.StatusCreated,
		},
		{
			name: "one section",
			fields: map[string]string{
				"title": "Lab moved", "description": "Lab 2 moved to 1004",
				"batch": "CSE-23", "section": "A",
			},
			wantStatus:  http.StatusCreated,
			wantSection: "A",
			wantAlert:   "A",
		},
		{
			name: "with a link",
			fields: map[string]string{
				"title": "Slides", "batch": "CSE-23",
				"link": "https://drive.google.com/file/x",
			},
			wantStatus: http.StatusCreated,
		},
		{
			name:       "without a batch",
			fields:     map[string]string{"title": "Quiz on Monday"},
			wantStatus: http.StatusBadRequest,
			wantError:  "Batch is required.",
		},
		{
			name:       "a broken batch",
			fields:     map[string]string{"title": "Quiz", "batch": "CSE23"},
			wantStatus: http.StatusBadRequest,
			wantError:  "Batch must look like CSE-23.",
		},
		{
			name:       "a broken section",
			fields:     map[string]string{"title": "Quiz", "batch": "CSE-23", "section": "AB"},
			wantStatus: http.StatusBadRequest,
			wantError:  "Section must be a single letter from A to Z.",
		},
		{
			name:       "without a title",
			fields:     map[string]string{"batch": "CSE-23"},
			wantStatus: http.StatusBadRequest,
			wantError:  "Title is required.",
		},
		{
			name: "a broken link",
			fields: map[string]string{
				"title": "Slides", "batch": "CSE-23", "link": "drive.google.com",
			},
			wantStatus: http.StatusBadRequest,
			wantError:  "Link must be a full link that starts with http:// or https://.",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			notices := &fakeNoticeStore{}
			alerts := &recordingAlerts{}
			handler := NewNotice(notices, &fakeFileStore{}, alerts)
			rec := httptest.NewRecorder()
			handler.TeacherCreate()(rec, teacherNoticeRequest(t, tc.fields))
			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d (body = %q)", rec.Code, tc.wantStatus, rec.Body.String())
			}
			if tc.wantStatus != http.StatusCreated {
				if got := errorOf(t, rec); got != tc.wantError {
					t.Errorf("error = %q, want %q", got, tc.wantError)
				}
				if len(notices.created) != 0 {
					t.Error("the store received a notice although the form was refused")
				}
				return
			}
			if len(notices.created) != 1 {
				t.Fatalf("the store received %d notices, want 1", len(notices.created))
			}
			saved := notices.created[0]
			if saved.Source != store.SourceTeacher {
				t.Errorf("source = %q, want %q", saved.Source, store.SourceTeacher)
			}
			if saved.Batch != "CSE-23" || saved.Section != tc.wantSection {
				t.Errorf("target = %q/%q, want CSE-23/%q", saved.Batch, saved.Section, tc.wantSection)
			}
			if saved.CreatedBy != noticeTeacherID {
				t.Errorf("created_by = %d, want %d", saved.CreatedBy, noticeTeacherID)
			}
			if tc.fields["link"] != "" && saved.LinkURL != tc.fields["link"] {
				t.Errorf("link = %q, want %q", saved.LinkURL, tc.fields["link"])
			}
			if alerts.calls != 1 || alerts.role != auth.RoleStudent {
				t.Fatalf("broadcasts = %d as %q, want one to the students", alerts.calls, alerts.role)
			}
			if alerts.batch != "CSE-23" || alerts.section != tc.wantAlert {
				t.Errorf("broadcast target = %q/%q, want CSE-23/%q", alerts.batch, alerts.section, tc.wantAlert)
			}
		})
	}
}

func TestNoticeTeacherListAndDelete(t *testing.T) {
	silenceLog(t)
	notices := &fakeNoticeStore{
		notices: []store.NoticeRow{
			{
				ID: 4, Source: store.SourceTeacher, Title: "Quiz on Monday", Batch: "CSE-23",
				Section: "A", CreatedByName: "Active Teacher",
				CreatedByPhoto: "/uploads/photos/a.png", CreatedAt: time.Now(),
			},
		},
	}
	handler := NewNotice(notices, &fakeFileStore{}, &recordingAlerts{})
	listed := httptest.NewRecorder()
	handler.TeacherList()(listed, teacherNoticeRequest(t, map[string]string{}))
	if listed.Code != http.StatusOK {
		t.Fatalf("list status = %d, want 200 (body = %q)", listed.Code, listed.Body.String())
	}
	if notices.askedUser != noticeTeacherID {
		t.Errorf("listed for %d, want %d", notices.askedUser, noticeTeacherID)
	}
	items := noticeItems(t, listed)
	if len(items) != 1 || items[0]["section"] != "A" {
		t.Fatalf("items = %v, want the teacher notice with its section", items)
	}
	if items[0]["posted_by_photo_url"] != "/uploads/photos/a.png" {
		t.Errorf("posted_by_photo_url = %v, want the teacher photo", items[0]["posted_by_photo_url"])
	}
	tests := []struct {
		name       string
		id         string
		deleteErr  error
		wantStatus int
	}{
		{name: "mine", id: "4", wantStatus: http.StatusNoContent},
		{name: "somebody else's", id: "4", deleteErr: store.ErrNotFound, wantStatus: http.StatusNotFound},
		{name: "a broken id", id: "x", wantStatus: http.StatusBadRequest},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			target := &fakeNoticeStore{deleteErr: tc.deleteErr, deletedFile: savedFile}
			files := &fakeFileStore{}
			request := teacherNoticeRequest(t, map[string]string{})
			request.SetPathValue("id", tc.id)
			rec := httptest.NewRecorder()
			NewNotice(target, files, &recordingAlerts{}).TeacherDelete()(rec, request)
			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d (body = %q)", rec.Code, tc.wantStatus, rec.Body.String())
			}
			if tc.wantStatus != http.StatusNoContent {
				if len(files.deleted) != 0 {
					t.Error("a file was deleted although the notice was not")
				}
				return
			}
			if target.askedUser != noticeTeacherID {
				t.Errorf("deleted for %d, want %d", target.askedUser, noticeTeacherID)
			}
			if len(files.deleted) != 1 || files.deleted[0] != savedFile {
				t.Errorf("deleted files = %v, want %q", files.deleted, savedFile)
			}
		})
	}
}

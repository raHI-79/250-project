package handler

import (
	"context"
	"github.com/raHI-79/250-project/backend/internal/auth"
	"github.com/raHI-79/250-project/backend/internal/store"
	"net/http"
	"net/http/httptest"
	"testing"
)

type fakeDirectoryStore struct {
	filter     store.DirectoryFilter
	people     []store.DirectoryRow
	teachers   []store.TeacherInfoRow
	askedQuery string
	err        error
}

func (f *fakeDirectoryStore) ListDirectory(ctx context.Context, filter store.DirectoryFilter) ([]store.DirectoryRow, error) {
	f.filter = filter
	return f.people, f.err
}

func (f *fakeDirectoryStore) ListTeacherInfo(ctx context.Context, search string) ([]store.TeacherInfoRow, error) {
	f.askedQuery = search
	return f.teachers, f.err
}

func TestDirectoryStudents(t *testing.T) {
	people := []store.DirectoryRow{
		{FullName: "Ratul Islam", RegNo: "2023331079", Batch: noticeBatch, Section: "A", Phone: "01712345678",
			Email: "ratul@student.sust.edu", BloodGroup: "O+", IsCR: true},
		{FullName: "Rakibul Hasan", RegNo: "2023331080", Batch: noticeBatch, Section: "B", Phone: "01712345679",
			Email: "rakibul@student.sust.edu", BloodGroup: "A+"},
	}
	tests := []struct {
		name       string
		query      string
		wantStatus int
		wantFilter store.DirectoryFilter
		wantError  string
	}{
		{name: "without filters", wantStatus: http.StatusOK},
		{
			name:       "by batch",
			query:      "?batch=cse-23",
			wantStatus: http.StatusOK,
			wantFilter: store.DirectoryFilter{Batch: noticeBatch},
		},
		{
			name:       "by blood group",
			query:      "?blood_group=o%2B",
			wantStatus: http.StatusOK,
			wantFilter: store.DirectoryFilter{BloodGroup: "O+"},
		},
		{
			name:       "by search",
			query:      "?search=Ratul",
			wantStatus: http.StatusOK,
			wantFilter: store.DirectoryFilter{Search: "Ratul"},
		},
		{
			name:       "by every filter at once",
			query:      "?batch=CSE-23&blood_group=A%2B&search=2023331080",
			wantStatus: http.StatusOK,
			wantFilter: store.DirectoryFilter{Batch: noticeBatch, BloodGroup: "A+", Search: "2023331080"},
		},
		{
			name:       "with a broken batch",
			query:      "?batch=23",
			wantStatus: http.StatusBadRequest,
			wantError:  "batch must look like CSE-23.",
		},
		{
			name:       "with a broken blood group",
			query:      "?blood_group=Z%2B",
			wantStatus: http.StatusBadRequest,
			wantError:  "blood_group must be one of A+, A-, B+, B-, AB+, AB-, O+, O-.",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			contacts := &fakeDirectoryStore{people: people}
			req := jsonRequest(t, http.MethodGet, "/api/students/directory"+tc.query, "", noticeCRID, auth.RoleStudent)
			rec := httptest.NewRecorder()
			NewDirectory(contacts).Students()(rec, req)
			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d (body = %q)", rec.Code, tc.wantStatus, rec.Body.String())
			}
			if tc.wantStatus != http.StatusOK {
				if got := errorOf(t, rec); got != tc.wantError {
					t.Errorf("error = %q, want %q", got, tc.wantError)
				}
				return
			}
			if contacts.filter != tc.wantFilter {
				t.Errorf("filter = %+v, want %+v", contacts.filter, tc.wantFilter)
			}
			items := noticeItems(t, rec)
			if len(items) != len(people) {
				t.Fatalf("got %d items, want %d", len(items), len(people))
			}
			if items[0]["reg_no"] != "2023331079" || items[0]["is_cr"] != true || items[0]["blood_group"] != "O+" {
				t.Errorf("first item = %v, want the class representative with a blood group", items[0])
			}
			if items[1]["is_cr"] != false || items[1]["phone"] != "01712345679" {
				t.Errorf("second item = %v, want a student with a phone number", items[1])
			}
		})
	}
}

func TestDirectoryTeachers(t *testing.T) {
	teachers := []store.TeacherInfoRow{
		{Name: "Dr. Example", Phone: "01712345600", Email: "example@sust.edu"},
	}
	tests := []struct {
		name       string
		query      string
		wantSearch string
	}{
		{name: "every teacher"},
		{name: "with a search", query: "?search=example", wantSearch: "example"},
		{name: "with a padded search", query: "?search=%20Dr.%20", wantSearch: "Dr."},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			contacts := &fakeDirectoryStore{teachers: teachers}
			req := jsonRequest(t, http.MethodGet, "/api/teachers-info"+tc.query, "", noticeCRID, auth.RoleTeacher)
			rec := httptest.NewRecorder()
			NewDirectory(contacts).Teachers()(rec, req)
			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200 (body = %q)", rec.Code, rec.Body.String())
			}
			if contacts.askedQuery != tc.wantSearch {
				t.Errorf("search = %q, want %q", contacts.askedQuery, tc.wantSearch)
			}
			items := noticeItems(t, rec)
			if len(items) != 1 || items[0]["name"] != "Dr. Example" || items[0]["email"] != "example@sust.edu" {
				t.Fatalf("items = %v, want the one active teacher", items)
			}
		})
	}
}

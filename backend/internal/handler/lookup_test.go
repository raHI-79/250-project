package handler

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/raHI-79/250-project/backend/internal/store"
	"net/http"
	"net/http/httptest"
	"testing"
)

func decodeInto(t *testing.T, rec *httptest.ResponseRecorder, target any) {
	t.Helper()
	if err := json.Unmarshal(rec.Body.Bytes(), target); err != nil {
		t.Fatalf("body is not valid JSON: %v (body = %q)", err, rec.Body.String())
	}
}

type fakeLookupStore struct {
	batches     []string
	sections    []string
	courses     []store.Course
	teachers    []store.TeacherOption
	err         error
	batchAsked  string
	sectionCall int
}

func (f *fakeLookupStore) ListBatches(ctx context.Context) ([]string, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.batches, nil
}

func (f *fakeLookupStore) ListSections(ctx context.Context, batch string) ([]string, error) {
	f.batchAsked = batch
	f.sectionCall++
	if f.err != nil {
		return nil, f.err
	}
	return f.sections, nil
}

func (f *fakeLookupStore) ListCourses(ctx context.Context) ([]store.Course, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.courses, nil
}

func (f *fakeLookupStore) ListTeacherOptions(ctx context.Context) ([]store.TeacherOption, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.teachers, nil
}

func newLookupStore() *fakeLookupStore {
	return &fakeLookupStore{
		batches:  []string{"CSE-22", "CSE-23"},
		sections: []string{"A", "B"},
		courses:  []store.Course{{ID: 1, Code: "CSE-233", Title: "Digital Logic Design"}},
		teachers: []store.TeacherOption{{ID: 3, UserID: 12, Name: "Active Teacher"}},
	}
}

func TestLookupLists(t *testing.T) {
	silenceLog(t)
	lookups := NewLookups(newLookupStore())
	tests := []struct {
		name      string
		path      string
		handler   http.HandlerFunc
		wantFirst string
		wantCount int
	}{
		{name: "batches", path: "/api/lookups/batches", handler: lookups.Batches(), wantFirst: "CSE-22", wantCount: 2},
		{name: "sections", path: "/api/lookups/sections?batch=CSE-23", handler: lookups.Sections(), wantFirst: "A", wantCount: 2},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			rec := adminCall(t, test.handler, http.MethodGet, test.path, "", "")
			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200 (body = %q)", rec.Code, rec.Body.String())
			}
			var payload struct {
				Items []string `json:"items"`
			}
			decodeInto(t, rec, &payload)
			if len(payload.Items) != test.wantCount {
				t.Fatalf("got %d items, want %d", len(payload.Items), test.wantCount)
			}
			if payload.Items[0] != test.wantFirst {
				t.Errorf("first item = %q, want %q", payload.Items[0], test.wantFirst)
			}
		})
	}
}

func TestLookupSectionsNeedABatch(t *testing.T) {
	silenceLog(t)
	tests := []struct {
		name       string
		query      string
		wantStatus int
		wantError  string
		wantAsked  string
	}{
		{name: "with a batch", query: "?batch=cse-23", wantStatus: http.StatusOK, wantAsked: "CSE-23"},
		{name: "without a batch", query: "", wantStatus: http.StatusBadRequest, wantError: MessageBatchRequired},
		{name: "blank batch", query: "?batch=%20", wantStatus: http.StatusBadRequest, wantError: MessageBatchRequired},
		{name: "bad batch", query: "?batch=CSE23", wantStatus: http.StatusBadRequest, wantError: "batch must look like CSE-23."},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			lookups := newLookupStore()
			rec := adminCall(t, NewLookups(lookups).Sections(), http.MethodGet, "/api/lookups/sections"+test.query, "", "")
			if rec.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d (body = %q)", rec.Code, test.wantStatus, rec.Body.String())
			}
			if test.wantError != "" {
				if got := errorMessage(t, rec); got != test.wantError {
					t.Errorf("error = %q, want %q", got, test.wantError)
				}
				if lookups.sectionCall != 0 {
					t.Error("the store was asked for sections although the batch was rejected")
				}
				return
			}
			if lookups.batchAsked != test.wantAsked {
				t.Errorf("the store was asked for batch %q, want %q", lookups.batchAsked, test.wantAsked)
			}
		})
	}
}

func TestLookupCoursesAndTeachers(t *testing.T) {
	silenceLog(t)
	lookups := NewLookups(newLookupStore())
	courses := adminCall(t, lookups.Courses(), http.MethodGet, "/api/lookups/courses", "", "")
	if courses.Code != http.StatusOK {
		t.Fatalf("courses status = %d, want 200 (body = %q)", courses.Code, courses.Body.String())
	}
	course := items(t, courses)[0]
	if course["id"] != float64(1) || course["code"] != "CSE-233" || course["title"] != "Digital Logic Design" {
		t.Errorf("course = %v, want the id, code and title", course)
	}
	teachers := adminCall(t, lookups.Teachers(), http.MethodGet, "/api/lookups/teachers", "", "")
	if teachers.Code != http.StatusOK {
		t.Fatalf("teachers status = %d, want 200 (body = %q)", teachers.Code, teachers.Body.String())
	}
	teacher := items(t, teachers)[0]
	if teacher["id"] != float64(3) || teacher["user_id"] != float64(12) || teacher["name"] != "Active Teacher" {
		t.Errorf("teacher = %v, want the id, user id and name", teacher)
	}
	failing := NewLookups(&fakeLookupStore{err: errors.New("database is down")})
	for name, rec := range map[string]int{
		"batches":  adminCall(t, failing.Batches(), http.MethodGet, "/api/lookups/batches", "", "").Code,
		"sections": adminCall(t, failing.Sections(), http.MethodGet, "/api/lookups/sections?batch=CSE-23", "", "").Code,
		"courses":  adminCall(t, failing.Courses(), http.MethodGet, "/api/lookups/courses", "", "").Code,
		"teachers": adminCall(t, failing.Teachers(), http.MethodGet, "/api/lookups/teachers", "", "").Code,
	} {
		if rec != http.StatusInternalServerError {
			t.Errorf("%s with a failing store = %d, want 500", name, rec)
		}
	}
}

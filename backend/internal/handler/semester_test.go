package handler

import (
	"context"
	"github.com/raHI-79/250-project/backend/internal/auth"
	"github.com/raHI-79/250-project/backend/internal/store"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestCreateRoutineNeedsASemester(t *testing.T) {
	silenceLog(t)
	tests := []struct {
		name       string
		body       string
		wantStatus int
		wantError  string
	}{
		{name: "with a semester", body: validRoutine, wantStatus: http.StatusCreated},
		{name: "without a semester",
			body: `{"batch":"CSE-23","section":"A","course_id":4,"teacher_id":6,"day_of_week":1,
"start_time":"10:30","end_time":"12:45","room":"1003","type":"lab"}`,
			wantStatus: http.StatusBadRequest, wantError: "Semester is required."},
		{name: "with a fifth year",
			body: `{"batch":"CSE-23","semester":"5-1","section":"A","course_id":4,"teacher_id":6,
"day_of_week":1,"start_time":"10:30","end_time":"12:45","room":"1003","type":"lab"}`,
			wantStatus: http.StatusBadRequest,
			wantError:  "Semester must be one of 1-1, 1-2, 2-1, 2-2, 3-1, 3-2, 4-1, 4-2."},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			catalog := &fakeCatalogStore{}
			rec := adminCall(t, NewCatalog(catalog).CreateRoutine(), http.MethodPost,
				"/api/admin/routines", "", test.body)
			if rec.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d (body = %s)", rec.Code, test.wantStatus, rec.Body.String())
			}
			if test.wantError != "" {
				if got := decodeBody(t, rec)["error"]; got != test.wantError {
					t.Errorf("error = %v, want %q", got, test.wantError)
				}
				return
			}
			if len(catalog.routines) != 1 || catalog.routines[0].Semester != "2-1" {
				t.Errorf("stored routines = %+v, want one in 2-1", catalog.routines)
			}
		})
	}
}

func TestRoutineListPassesTheSemesterFilter(t *testing.T) {
	silenceLog(t)
	catalog := &fakeCatalogStore{}
	rec := adminCall(t, NewCatalog(catalog).Routines(), http.MethodGet,
		"/api/admin/routines?batch=CSE-23&semester=3-2", "", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body = %s)", rec.Code, rec.Body.String())
	}
	if catalog.semesterFilter != "3-2" {
		t.Errorf("semester filter = %q, want 3-2", catalog.semesterFilter)
	}
	bad := adminCall(t, NewCatalog(&fakeCatalogStore{}).Routines(), http.MethodGet,
		"/api/admin/routines?semester=9-9", "", "")
	if bad.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400 for an unknown semester", bad.Code)
	}
}

func TestMoveRoutineSemester(t *testing.T) {
	silenceLog(t)
	catalog := &fakeCatalogStore{}
	rec := adminCall(t, NewCatalog(catalog).MoveRoutine(), http.MethodPatch,
		"/api/admin/routines/4", "4", `{"semester":"4-2"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body = %s)", rec.Code, rec.Body.String())
	}
	if catalog.movedSemester != "4-2" {
		t.Errorf("moved to %q, want 4-2", catalog.movedSemester)
	}
	bad := adminCall(t, NewCatalog(&fakeCatalogStore{}).MoveRoutine(), http.MethodPatch,
		"/api/admin/routines/4", "4", `{"semester":"later"}`)
	if bad.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400 for an unknown semester", bad.Code)
	}
}

func TestSetRunningSemester(t *testing.T) {
	silenceLog(t)
	tests := []struct {
		name       string
		batch      string
		body       string
		wantStatus int
	}{
		{name: "set", batch: "CSE-23", body: `{"semester":"2-2"}`, wantStatus: http.StatusOK},
		{name: "a bad batch", batch: "CSE23", body: `{"semester":"2-2"}`,
			wantStatus: http.StatusBadRequest},
		{name: "a bad semester", batch: "CSE-23", body: `{"semester":"0-0"}`,
			wantStatus: http.StatusBadRequest},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			catalog := &fakeCatalogStore{}
			req := jsonRequest(t, http.MethodPut,
				"/api/admin/batches/"+test.batch+"/running-semester", test.body, 1, auth.RoleAdmin)
			req.SetPathValue("batch", test.batch)
			rec := httptest.NewRecorder()
			NewCatalog(catalog).SetRunningSemester()(rec, req)
			if rec.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d (body = %s)", rec.Code, test.wantStatus, rec.Body.String())
			}
			if test.wantStatus != http.StatusOK {
				return
			}
			if catalog.runningSet["CSE-23"] != "2-2" {
				t.Errorf("running semesters = %v, want CSE-23 on 2-2", catalog.runningSet)
			}
			if got := decodeBody(t, rec)["semester_label"]; got != "2nd year 2nd semester" {
				t.Errorf("label = %v, want the spelled out semester", got)
			}
		})
	}
}

func TestRunningSemestersList(t *testing.T) {
	silenceLog(t)
	catalog := &fakeCatalogStore{running: map[string]string{"CSE-24": "1-1", "CSE-23": "2-1"}}
	rec := adminCall(t, NewCatalog(catalog).RunningSemesters(), http.MethodGet,
		"/api/admin/batches/running-semester", "", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body = %s)", rec.Code, rec.Body.String())
	}
	items, ok := decodeBody(t, rec)["items"].([]any)
	if !ok || len(items) != 2 {
		t.Fatalf("items = %v, want two batches", decodeBody(t, rec)["items"])
	}
	first, _ := items[0].(map[string]any)
	if first["batch"] != "CSE-23" {
		t.Errorf("first batch = %v, want CSE-23 so the list is sorted", first["batch"])
	}
}

type fakeCourseStore struct {
	profile store.StudentProfile
	running string
	slots   []store.RunningCourseSlot
	asked   string
}

func (f *fakeCourseStore) GetStudentProfile(ctx context.Context, userID int64) (store.StudentProfile, error) {
	return f.profile, nil
}

func (f *fakeCourseStore) ListStudentDropCourses(ctx context.Context, studentID int64) ([]store.Course, error) {
	return nil, nil
}

func (f *fakeCourseStore) ListPublishedDay(ctx context.Context, batch, date string,
	dropCourseCodes []string) ([]store.PlanItem, error) {
	return nil, nil
}

func (f *fakeCourseStore) ListStudentExams(ctx context.Context, batch, section string,
	now time.Time) ([]store.StudentExam, error) {
	return nil, nil
}

func (f *fakeCourseStore) ListFilteredRoutines(ctx context.Context,
	filter store.RoutineFilter) ([]store.RoutineRow, error) {
	return nil, nil
}

func (f *fakeCourseStore) ListStudentNotices(ctx context.Context, batch, section, source,
	scope string) ([]store.NoticeRow, error) {
	return nil, nil
}

func (f *fakeCourseStore) ListStudentPolls(ctx context.Context, viewerUserID int64,
	batch, section string) ([]store.PollRow, error) {
	return nil, nil
}

func (f *fakeCourseStore) GetMessageViewer(ctx context.Context, userID int64) (store.MessageViewer, error) {
	return store.MessageViewer{UserID: userID}, nil
}

func (f *fakeCourseStore) CountUnreadMessages(ctx context.Context, viewer store.MessageViewer) (int, error) {
	return 0, nil
}

func (f *fakeCourseStore) RunningSemester(ctx context.Context, batch string) (string, error) {
	if f.running == "" {
		return "1-1", nil
	}
	return f.running, nil
}

func (f *fakeCourseStore) ListRunningCourseSlots(ctx context.Context,
	batch string) ([]store.RunningCourseSlot, error) {
	f.asked = batch
	return f.slots, nil
}

func TestStudentCoursesListAddsUpTheCredits(t *testing.T) {
	silenceLog(t)
	courses := &fakeCourseStore{
		running: "2-1",
		slots: []store.RunningCourseSlot{
			{CourseID: 1, CourseCode: "CSE 2243 7637D", CourseTitle: "Digital Logic Design",
				Credit: 3.0, TeacherUserID: 7, TeacherName: "Mahbubur Rahman",
				TeacherDesignation: "Professor", Section: "ALL", Type: "class"},
			{CourseID: 1, CourseCode: "CSE 2243 7637D", CourseTitle: "Digital Logic Design",
				Credit: 3.0, TeacherUserID: 7, TeacherName: "Mahbubur Rahman",
				TeacherDesignation: "Professor", Section: "ALL", Type: "class"},
			{CourseID: 2, CourseCode: "CSE 2244 7638E", CourseTitle: "Digital Logic Lab",
				Credit: 1.5, TeacherUserID: 8, TeacherName: "Farhana Sultana",
				TeacherDesignation: "Lecturer", Section: "A", Type: "lab"},
		},
	}
	courses.profile = store.StudentProfile{UserID: 11, Batch: "CSE-23", Section: "A"}
	req := jsonRequest(t, http.MethodGet, "/api/students/courses", "", 11, auth.RoleStudent)
	rec := httptest.NewRecorder()
	NewStudent(accountConfig(t, "test"), courses).Courses()(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body = %s)", rec.Code, rec.Body.String())
	}
	payload := decodeBody(t, rec)
	if payload["semester"] != "2-1" || payload["semester_label"] != "2nd year 1st semester" {
		t.Errorf("semester = %v / %v, want the running one spelled out",
			payload["semester"], payload["semester_label"])
	}
	if payload["total_credits"] != 4.5 {
		t.Errorf("total credits = %v, want 4.5", payload["total_credits"])
	}
	items, _ := payload["items"].([]any)
	if len(items) != 2 {
		t.Fatalf("items = %v, want two courses", payload["items"])
	}
	first, _ := items[0].(map[string]any)
	if first["sections"] != "Both sections" || first["classes_per_week"] != 2.0 {
		t.Errorf("first course = %v, want both sections and two weekly classes", first)
	}
	second, _ := items[1].(map[string]any)
	if second["sections"] != "Section A" || second["labs_per_week"] != 1.0 {
		t.Errorf("second course = %v, want section A and one weekly lab", second)
	}
	teachers, _ := first["teachers"].([]any)
	if len(teachers) != 1 {
		t.Fatalf("teachers = %v, want one entry per person", first["teachers"])
	}
	teacher, _ := teachers[0].(map[string]any)
	if teacher["user_id"] != 7.0 || teacher["designation"] != "Professor" {
		t.Errorf("teacher = %v, want the person link and designation", teacher)
	}
}

func TestStudentCoursesTakesABatch(t *testing.T) {
	silenceLog(t)
	courses := &fakeCourseStore{}
	courses.profile = store.StudentProfile{UserID: 11, Batch: "CSE-23", Section: "A"}
	req := jsonRequest(t, http.MethodGet, "/api/students/courses?batch=cse-21", "", 11, auth.RoleStudent)
	rec := httptest.NewRecorder()
	NewStudent(accountConfig(t, "test"), courses).Courses()(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body = %s)", rec.Code, rec.Body.String())
	}
	if courses.asked != "CSE-21" {
		t.Errorf("asked for %q, want the batch from the query", courses.asked)
	}
	bad := jsonRequest(t, http.MethodGet, "/api/students/courses?batch=nope", "", 11, auth.RoleStudent)
	badRec := httptest.NewRecorder()
	NewStudent(accountConfig(t, "test"), courses).Courses()(badRec, bad)
	if badRec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400 for a bad batch", badRec.Code)
	}
}

package handler

import (
	"context"
	"errors"
	"github.com/raHI-79/250-project/backend/internal/config"
	"github.com/raHI-79/250-project/backend/internal/middleware"
	"github.com/raHI-79/250-project/backend/internal/store"
	"github.com/raHI-79/250-project/backend/internal/validate"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

const studentUserID = int64(61)

type fakeStudentStore struct {
	profile    store.StudentProfile
	profileErr error
	plan       []store.PlanItem
	drops      []store.DropCourseItem
	exams      []store.ExamRow
	entries    []store.RoutineRow
	err        error
	askedBatch string
	askedDate  string
	askedCode  string
	askedOwn   string
	askedExam  string
	filter     store.RoutineFilter
}

func (f *fakeStudentStore) GetStudentProfile(ctx context.Context, userID int64) (store.StudentProfile, error) {
	if f.profileErr != nil {
		return store.StudentProfile{}, f.profileErr
	}
	return f.profile, nil
}

func (f *fakeStudentStore) ListDayPlan(ctx context.Context, batch, date string) ([]store.PlanItem, error) {
	f.askedBatch, f.askedDate = batch, date
	if f.err != nil {
		return nil, f.err
	}
	return f.plan, nil
}

func (f *fakeStudentStore) ListDropCourseSchedule(ctx context.Context, courseCode, ownBatch string, now time.Time) ([]store.DropCourseItem, error) {
	f.askedCode, f.askedOwn = courseCode, ownBatch
	if f.err != nil {
		return nil, f.err
	}
	return f.drops, nil
}

func (f *fakeStudentStore) ListExams(ctx context.Context, batch, section string, now time.Time) ([]store.ExamRow, error) {
	f.askedBatch, f.askedExam = batch, section
	if f.err != nil {
		return nil, f.err
	}
	return f.exams, nil
}

func (f *fakeStudentStore) ListFilteredRoutines(ctx context.Context, filter store.RoutineFilter) ([]store.RoutineRow, error) {
	f.filter = filter
	if f.err != nil {
		return nil, f.err
	}
	return f.entries, nil
}

func newStudentStore() *fakeStudentStore {
	return &fakeStudentStore{
		profile: store.StudentProfile{UserID: studentUserID, Batch: "CSE-23", Section: "A"},
	}
}

func newStudentHandler(t *testing.T, students StudentStore) *Student {
	t.Helper()
	loc, err := time.LoadLocation("Asia/Dhaka")
	if err != nil {
		t.Fatalf("could not load the timezone: %v", err)
	}
	cfg := &config.Config{AppEnv: "test", Timezone: "Asia/Dhaka", Location: loc}
	return NewStudent(cfg, students)
}

func studentCall(t *testing.T, handlerFunc http.HandlerFunc, target string) *httptest.ResponseRecorder {
	t.Helper()
	user := middleware.User{ID: studentUserID, Identifier: "2023331001", Role: "student"}
	return roleCall(t, handlerFunc, user, http.MethodGet, target, "", "")
}

func tomorrowInDhaka(t *testing.T) time.Time {
	t.Helper()
	loc, err := time.LoadLocation("Asia/Dhaka")
	if err != nil {
		t.Fatalf("could not load the timezone: %v", err)
	}
	return time.Now().In(loc).AddDate(0, 0, 1)
}

func TestStudentUpcomingRoutine(t *testing.T) {
	silenceLog(t)
	tomorrow := tomorrowInDhaka(t)
	students := newStudentStore()
	students.plan = []store.PlanItem{
		{Section: "A", StartTime: "10:30", EndTime: "12:45", CourseCode: "CSE-233", CourseTitle: "Digital Logic Design", Kind: "lab", Room: "1003", TeacherName: "First Teacher"},
		{Section: "A", StartTime: "08:00", EndTime: "09:00", CourseCode: "CSE-250", CourseTitle: "Software Engineering", Kind: "class", Room: "1004", TeacherName: "Second Teacher", Cancelled: true},
		{Section: "ALL", StartTime: "13:00", EndTime: "14:00", CourseCode: "CSE-233", CourseTitle: "Digital Logic Design", Kind: "exam", Room: "2001", TeacherName: "First Teacher"},
		{Section: "B", StartTime: "09:00", EndTime: "10:00", CourseCode: "CSE-250", CourseTitle: "Software Engineering", Kind: "class", Room: "1005", TeacherName: "Second Teacher"},
	}
	rec := studentCall(t, newStudentHandler(t, students).UpcomingRoutine(), "/api/students/upcoming-routine")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body = %q)", rec.Code, rec.Body.String())
	}
	if students.askedBatch != "CSE-23" || students.askedDate != tomorrow.Format(validate.DateLayout) {
		t.Errorf("the store was asked for %q on %q, want CSE-23 tomorrow", students.askedBatch, students.askedDate)
	}
	payload := decodeBody(t, rec)
	if payload["date"] != tomorrow.Format(validate.DateLayout) {
		t.Errorf("date = %v, want tomorrow", payload["date"])
	}
	if payload["date_label"] != tomorrow.Format("02.01.06") {
		t.Errorf("date_label = %v, want %q", payload["date_label"], tomorrow.Format("02.01.06"))
	}
	if payload["weekday"] != validate.DayName(int(tomorrow.Weekday())) || payload["batch"] != "CSE-23" {
		t.Errorf("payload = %v, want the weekday and batch", payload)
	}
	groups, ok := payload["groups"].([]any)
	if !ok || len(groups) != 3 {
		t.Fatalf("groups = %v, want three of them", payload["groups"])
	}
	wantSections := []string{"A", "B", "ALL"}
	wantLabels := []string{"Section A", "Section B", "Both Sections"}
	for i, group := range groups {
		entry := group.(map[string]any)
		if entry["section"] != wantSections[i] || entry["section_label"] != wantLabels[i] {
			t.Errorf("group %d = %v, want %q", i, entry, wantSections[i])
		}
	}
	first := groups[0].(map[string]any)["items"].([]any)
	if len(first) != 2 {
		t.Fatalf("section A has %d items, want 2", len(first))
	}
	earliest := first[0].(map[string]any)
	if earliest["start_time"] != "08:00" || earliest["status"] != StatusCancelled {
		t.Errorf("first item = %v, want the cancelled 08:00 class first", earliest)
	}
	if later := first[1].(map[string]any); later["status"] != StatusScheduled || later["kind"] != "lab" {
		t.Errorf("second item = %v, want a scheduled lab", later)
	}
	chosen := studentCall(t, newStudentHandler(t, students).UpcomingRoutine(), "/api/students/upcoming-routine?date=2026-10-20")
	if chosen.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body = %q)", chosen.Code, chosen.Body.String())
	}
	if students.askedDate != "2026-10-20" {
		t.Errorf("the store was asked for %q, want the requested date", students.askedDate)
	}
	bad := studentCall(t, newStudentHandler(t, newStudentStore()).UpcomingRoutine(), "/api/students/upcoming-routine?date=soon")
	if bad.Code != http.StatusBadRequest {
		t.Errorf("bad date status = %d, want 400", bad.Code)
	}
	missing := newStudentStore()
	missing.profileErr = store.ErrNotFound
	denied := studentCall(t, newStudentHandler(t, missing).UpcomingRoutine(), "/api/students/upcoming-routine")
	if denied.Code != http.StatusForbidden {
		t.Errorf("status without a profile = %d, want 403", denied.Code)
	}
	if got := errorMessage(t, denied); got != MessageNotAStudent {
		t.Errorf("error = %q, want %q", got, MessageNotAStudent)
	}
	failing := newStudentStore()
	failing.err = errors.New("database is down")
	broken := studentCall(t, newStudentHandler(t, failing).UpcomingRoutine(), "/api/students/upcoming-routine")
	if broken.Code != http.StatusInternalServerError {
		t.Errorf("failing store status = %d, want 500", broken.Code)
	}
}

func TestStudentDropCourseRoutine(t *testing.T) {
	silenceLog(t)
	without := newStudentStore()
	rec := studentCall(t, newStudentHandler(t, without).DropCourseRoutine(), "/api/students/drop-course-routine")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body = %q)", rec.Code, rec.Body.String())
	}
	payload := decodeBody(t, rec)
	if payload["has_drop_course"] != false {
		t.Errorf("has_drop_course = %v, want false", payload["has_drop_course"])
	}
	if list, ok := payload["items"].([]any); !ok || len(list) != 0 {
		t.Errorf("items = %v, want an empty list", payload["items"])
	}
	if without.askedCode != "" {
		t.Error("the store was asked for a drop course although the student has none")
	}
	students := newStudentStore()
	students.profile.HasDropCourse = true
	students.profile.DropCourseCode = "CSE-211"
	students.drops = []store.DropCourseItem{
		{Source: "fixed", Batch: "CSE-22", Section: "A", CourseCode: "CSE-211", CourseTitle: "Data Structures", Kind: "class", DayOfWeek: 1, StartTime: "08:00", EndTime: "09:00", Room: "1003", TeacherName: "First Teacher"},
		{Source: "dated", Batch: "CSE-22", Section: "ALL", CourseCode: "CSE-211", CourseTitle: "Data Structures", Kind: "exam", DayOfWeek: 2, EventDate: "2026-10-20", StartTime: "10:30", EndTime: "12:45", Room: "2001", TeacherName: "First Teacher"},
		{Source: "fixed", Batch: "CSE-24", Section: "B", CourseCode: "CSE-211", CourseTitle: "Data Structures", Kind: "lab", DayOfWeek: 3, StartTime: "13:00", EndTime: "14:00", Room: "1004", TeacherName: "Second Teacher"},
	}
	full := studentCall(t, newStudentHandler(t, students).DropCourseRoutine(), "/api/students/drop-course-routine")
	if full.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body = %q)", full.Code, full.Body.String())
	}
	if students.askedCode != "CSE-211" || students.askedOwn != "CSE-23" {
		t.Errorf("the store was asked for %q excluding %q, want CSE-211 excluding CSE-23", students.askedCode, students.askedOwn)
	}
	body := decodeBody(t, full)
	if body["has_drop_course"] != true || body["course_code"] != "CSE-211" {
		t.Errorf("payload = %v, want the drop course flag and code", body)
	}
	if list := items(t, full); len(list) != 3 || list[0]["day"] != "Monday" || list[1]["event_date"] != "2026-10-20" {
		t.Errorf("items = %v, want the three entries with day names", list)
	}
	groups, ok := body["groups"].([]any)
	if !ok || len(groups) != 2 {
		t.Fatalf("groups = %v, want one per batch", body["groups"])
	}
	first := groups[0].(map[string]any)
	if first["batch"] != "CSE-22" || len(first["items"].([]any)) != 2 {
		t.Errorf("first group = %v, want the two entries of CSE-22", first)
	}
	if second := groups[1].(map[string]any); second["batch"] != "CSE-24" {
		t.Errorf("second group = %v, want CSE-24", second)
	}
}

func TestStudentExams(t *testing.T) {
	silenceLog(t)
	students := newStudentStore()
	students.exams = []store.ExamRow{
		{ID: 5, EventDate: "2026-10-20", StartTime: "10:30", EndTime: "12:45", CourseCode: "CSE-233", CourseTitle: "Digital Logic Design", Batch: "CSE-23", Section: "ALL", Room: "2001"},
	}
	rec := studentCall(t, newStudentHandler(t, students).Exams(), "/api/students/exams")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body = %q)", rec.Code, rec.Body.String())
	}
	if students.askedBatch != "CSE-23" || students.askedExam != "A" {
		t.Errorf("the store was asked for %q and %q, want CSE-23 and A", students.askedBatch, students.askedExam)
	}
	item := items(t, rec)[0]
	if item["event_date"] != "2026-10-20" || item["course_code"] != "CSE-233" || item["room"] != "2001" {
		t.Errorf("item = %v, want the exam details", item)
	}
	failing := newStudentStore()
	failing.err = errors.New("database is down")
	broken := studentCall(t, newStudentHandler(t, failing).Exams(), "/api/students/exams")
	if broken.Code != http.StatusInternalServerError {
		t.Errorf("failing store status = %d, want 500", broken.Code)
	}
}

func TestStudentRoutineBrowser(t *testing.T) {
	silenceLog(t)
	tests := []struct {
		name       string
		query      string
		wantStatus int
		wantFilter store.RoutineFilter
		wantError  string
	}{
		{
			name:       "defaults",
			query:      "",
			wantStatus: http.StatusOK,
			wantFilter: store.RoutineFilter{Batch: "CSE-23", Section: "A", DayOfWeek: store.NoDayFilter},
		},
		{
			name:       "another batch and section",
			query:      "?batch=cse-22&section=b",
			wantStatus: http.StatusOK,
			wantFilter: store.RoutineFilter{Batch: "CSE-22", Section: "B", DayOfWeek: store.NoDayFilter},
		},
		{
			name:       "both sections",
			query:      "?section=all",
			wantStatus: http.StatusOK,
			wantFilter: store.RoutineFilter{Batch: "CSE-23", Section: "ALL", DayOfWeek: store.NoDayFilter},
		},
		{
			name:       "day and course",
			query:      "?day=0&course=cse233",
			wantStatus: http.StatusOK,
			wantFilter: store.RoutineFilter{Batch: "CSE-23", Section: "A", DayOfWeek: 0, CourseCode: "CSE233"},
		},
		{name: "bad batch", query: "?batch=CSE23", wantStatus: http.StatusBadRequest, wantError: "batch must look like CSE-23."},
		{name: "bad section", query: "?section=AB", wantStatus: http.StatusBadRequest, wantError: "section must be a single letter from A to Z or ALL."},
		{name: "day not a number", query: "?day=monday", wantStatus: http.StatusBadRequest, wantError: "day must be a number from 0 (Sunday) to 6 (Saturday)."},
		{name: "day out of range", query: "?day=9", wantStatus: http.StatusBadRequest, wantError: "day must be a number from 0 (Sunday) to 6 (Saturday)."},
		{name: "bad course", query: "?course=CSE", wantStatus: http.StatusBadRequest, wantError: "course must look like CSE-233 or CSE233."},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			students := newStudentStore()
			students.entries = []store.RoutineRow{
				{ID: 1, Batch: "CSE-23", Section: "A", CourseCode: "CSE-233", CourseTitle: "Digital Logic Design", TeacherName: "First Teacher", DayOfWeek: 0, StartTime: "08:00", EndTime: "09:00", Room: "1003", Type: "class"},
			}
			rec := studentCall(t, newStudentHandler(t, students).Routine(), "/api/students/routine"+test.query)
			if rec.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d (body = %q)", rec.Code, test.wantStatus, rec.Body.String())
			}
			if test.wantError != "" {
				if got := errorMessage(t, rec); got != test.wantError {
					t.Errorf("error = %q, want %q", got, test.wantError)
				}
				return
			}
			if students.filter != test.wantFilter {
				t.Errorf("filter = %+v, want %+v", students.filter, test.wantFilter)
			}
			if item := items(t, rec)[0]; item["day"] != "Sunday" || item["course_title"] != "Digital Logic Design" {
				t.Errorf("item = %v, want the day name and course title", item)
			}
		})
	}
}

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
	exams      []store.ExamRow
	entries    []store.RoutineRow
	notices    []store.NoticeRow
	polls      []store.PollRow
	unread     int
	err        error
	askedBatch string
	askedDate  string
	askedCodes []string
	courses    []store.Course
	courseErr  error
	askedExam  string
	filter     store.RoutineFilter
}

func (f *fakeStudentStore) GetStudentProfile(ctx context.Context, userID int64) (store.StudentProfile, error) {
	if f.profileErr != nil {
		return store.StudentProfile{}, f.profileErr
	}
	return f.profile, nil
}

func (f *fakeStudentStore) ListStudentDropCourses(ctx context.Context, studentID int64) ([]store.Course, error) {
	if f.courseErr != nil {
		return nil, f.courseErr
	}
	return f.courses, nil
}

func (f *fakeStudentStore) ListDayPlan(ctx context.Context, batch, date string, dropCourseCodes []string) ([]store.PlanItem, error) {
	f.askedBatch, f.askedDate, f.askedCodes = batch, date, dropCourseCodes
	if f.err != nil {
		return nil, f.err
	}
	return f.plan, nil
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

func (f *fakeStudentStore) ListStudentNotices(ctx context.Context, batch, section, source, scope string) ([]store.NoticeRow, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.notices, nil
}

func (f *fakeStudentStore) ListStudentPolls(ctx context.Context, viewerUserID int64, batch, section string) ([]store.PollRow, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.polls, nil
}

func (f *fakeStudentStore) GetMessageViewer(ctx context.Context, userID int64) (store.MessageViewer, error) {
	return store.MessageViewer{UserID: userID, Role: "student", Batch: f.profile.Batch, Section: f.profile.Section}, nil
}

func (f *fakeStudentStore) CountUnreadMessages(ctx context.Context, viewer store.MessageViewer) (int, error) {
	if f.err != nil {
		return 0, f.err
	}
	return f.unread, nil
}

func newStudentStore() *fakeStudentStore {
	return &fakeStudentStore{
		profile: store.StudentProfile{UserID: studentUserID, StudentID: 3, Batch: "CSE-23", Section: "A"},
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

func todayInDhaka(t *testing.T) time.Time {
	t.Helper()
	loc, err := time.LoadLocation("Asia/Dhaka")
	if err != nil {
		t.Fatalf("could not load the timezone: %v", err)
	}
	return time.Now().In(loc)
}

func ownBatchPlan() []store.PlanItem {
	return []store.PlanItem{
		{Batch: "CSE-23", Section: "A", StartTime: "10:30", EndTime: "12:45", CourseCode: "CSE-233", CourseTitle: "Digital Logic Design", Kind: "lab", Room: "1003", TeacherName: "First Teacher"},
		{Batch: "CSE-23", Section: "A", StartTime: "08:00", EndTime: "09:00", CourseCode: "CSE-250", CourseTitle: "Software Engineering", Kind: "class", Room: "1004", TeacherName: "Second Teacher", Cancelled: true},
		{Batch: "CSE-23", Section: "ALL", StartTime: "13:00", EndTime: "14:00", CourseCode: "CSE-233", CourseTitle: "Digital Logic Design", Kind: "exam", Room: "2001", TeacherName: "First Teacher"},
		{Batch: "CSE-23", Section: "B", StartTime: "09:00", EndTime: "10:00", CourseCode: "CSE-250", CourseTitle: "Software Engineering", Kind: "class", Room: "1005", TeacherName: "Second Teacher"},
	}
}

func dropCoursePlan() []store.PlanItem {
	return []store.PlanItem{
		{Batch: "CSE-24", Section: "B", StartTime: "14:00", EndTime: "15:00", CourseCode: "CSE-211", CourseTitle: "Data Structures", Kind: "class", Room: "1006", TeacherName: "Third Teacher"},
		{Batch: "CSE-22", Section: "A", StartTime: "11:00", EndTime: "12:00", CourseCode: "CSE-211", CourseTitle: "Data Structures", Kind: "exam", Room: "2002", TeacherName: "Third Teacher"},
	}
}

func TestStudentUpcomingRoutineGroups(t *testing.T) {
	silenceLog(t)
	tests := []struct {
		name       string
		plan       []store.PlanItem
		wantKeys   []string
		wantLabels []string
	}{
		{
			name:       "the own batch only",
			plan:       ownBatchPlan(),
			wantKeys:   []string{"A", "B", "ALL"},
			wantLabels: []string{"Section A", "Section B", "Both sections"},
		},
		{
			name:       "with the drop courses last",
			plan:       append(ownBatchPlan(), dropCoursePlan()...),
			wantKeys:   []string{"A", "B", "ALL", GroupDropCourses},
			wantLabels: []string{"Section A", "Section B", "Both sections", LabelDropCourses},
		},
		{
			name:       "the drop courses alone",
			plan:       dropCoursePlan(),
			wantKeys:   []string{GroupDropCourses},
			wantLabels: []string{LabelDropCourses},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			students := newStudentStore()
			students.plan = test.plan
			rec := studentCall(t, newStudentHandler(t, students).UpcomingRoutine(), "/api/students/upcoming-routine")
			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200 (body = %q)", rec.Code, rec.Body.String())
			}
			groups, ok := decodeBody(t, rec)["groups"].([]any)
			if !ok || len(groups) != len(test.wantKeys) {
				t.Fatalf("groups = %v, want %d of them", groups, len(test.wantKeys))
			}
			for i, group := range groups {
				entry := group.(map[string]any)
				if entry["key"] != test.wantKeys[i] || entry["section_label"] != test.wantLabels[i] {
					t.Errorf("group %d = %v, want %q labelled %q", i, entry, test.wantKeys[i], test.wantLabels[i])
				}
			}
			last := groups[len(groups)-1].(map[string]any)
			if last["key"] != GroupDropCourses {
				return
			}
			if last["drop_course"] != true {
				t.Errorf("the drop course group = %v, want drop_course true", last)
			}
			dropped := last["items"].([]any)
			if len(dropped) != 2 {
				t.Fatalf("the drop course group has %d items, want 2", len(dropped))
			}
			if first := dropped[0].(map[string]any); first["batch"] != "CSE-22" || first["start_time"] != "11:00" {
				t.Errorf("first drop course item = %v, want the CSE-22 entry at 11:00", first)
			}
		})
	}
}

func TestStudentUpcomingRoutineDate(t *testing.T) {
	silenceLog(t)
	today := todayInDhaka(t)
	students := newStudentStore()
	students.plan = ownBatchPlan()
	rec := studentCall(t, newStudentHandler(t, students).UpcomingRoutine(), "/api/students/upcoming-routine")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body = %q)", rec.Code, rec.Body.String())
	}
	if students.askedBatch != "CSE-23" || students.askedDate != today.Format(validate.DateLayout) {
		t.Errorf("the store was asked for %q on %q, want CSE-23 today", students.askedBatch, students.askedDate)
	}
	payload := decodeBody(t, rec)
	if payload["date"] != today.Format(validate.DateLayout) {
		t.Errorf("date = %v, want today", payload["date"])
	}
	if payload["date_label"] != today.Format(dateLabelLayout) {
		t.Errorf("date_label = %v, want %q", payload["date_label"], today.Format(dateLabelLayout))
	}
	if payload["weekday"] != validate.DayName(int(today.Weekday())) || payload["batch"] != "CSE-23" {
		t.Errorf("payload = %v, want the weekday and batch", payload)
	}
	first := payload["groups"].([]any)[0].(map[string]any)["items"].([]any)
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

func TestStudentUpcomingRoutineCarriesTheDropCourseCodes(t *testing.T) {
	silenceLog(t)
	students := newStudentStore()
	students.courses = []store.Course{{ID: 9, Code: "CSE-211", Title: "Data Structures"}}
	rec := studentCall(t, newStudentHandler(t, students).UpcomingRoutine(), "/api/students/upcoming-routine")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body = %q)", rec.Code, rec.Body.String())
	}
	if len(students.askedCodes) != 1 || students.askedCodes[0] != "CSE-211" {
		t.Errorf("the store was asked for %v, want CSE-211", students.askedCodes)
	}
}

func TestStudentToday(t *testing.T) {
	silenceLog(t)
	today := todayInDhaka(t)
	students := newStudentStore()
	students.plan = append(ownBatchPlan(), dropCoursePlan()...)
	students.exams = []store.ExamRow{
		{ID: 5, EventDate: "2026-10-20", CourseCode: "CSE-233"},
		{ID: 6, EventDate: "2026-10-27", CourseCode: "CSE-250"},
	}
	students.polls = []store.PollRow{
		{ID: 1, Status: store.PollOpen},
		{ID: 2, Status: store.PollClosed},
		{ID: 3, Status: store.PollOpen},
	}
	students.notices = []store.NoticeRow{
		{ID: 1, Title: "First"}, {ID: 2, Title: "Second"}, {ID: 3, Title: "Third"}, {ID: 4, Title: "Fourth"},
	}
	students.unread = 4
	rec := studentCall(t, newStudentHandler(t, students).Today(), "/api/students/today")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body = %q)", rec.Code, rec.Body.String())
	}
	payload := decodeBody(t, rec)
	if payload["date"] != today.Format(validate.DateLayout) || students.askedDate != today.Format(validate.DateLayout) {
		t.Errorf("date = %v, want today", payload["date"])
	}
	if payload["classes_today"] != float64(5) {
		t.Errorf("classes_today = %v, want the five scheduled items", payload["classes_today"])
	}
	if payload["upcoming_exams"] != float64(2) || payload["open_polls"] != float64(2) {
		t.Errorf("payload = %v, want two exams and two open polls", payload)
	}
	if payload["unread_messages"] != float64(4) {
		t.Errorf("unread_messages = %v, want 4", payload["unread_messages"])
	}
	next, ok := payload["next_exam"].(map[string]any)
	if !ok || next["date"] != "2026-10-20" || next["course"] != "CSE-233" {
		t.Errorf("next_exam = %v, want the first exam", payload["next_exam"])
	}
	latest, ok := payload["latest_notices"].([]any)
	if !ok || len(latest) != latestNoticeCount {
		t.Fatalf("latest_notices = %v, want %d of them", payload["latest_notices"], latestNoticeCount)
	}
	if latest[0].(map[string]any)["title"] != "First" {
		t.Errorf("the newest notice = %v, want the first one", latest[0])
	}
	empty := newStudentStore()
	none := studentCall(t, newStudentHandler(t, empty).Today(), "/api/students/today")
	if none.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body = %q)", none.Code, none.Body.String())
	}
	if body := decodeBody(t, none); body["next_exam"] != nil || body["classes_today"] != float64(0) {
		t.Errorf("payload = %v, want no exam and no class", body)
	}
	failing := newStudentStore()
	failing.err = errors.New("database is down")
	broken := studentCall(t, newStudentHandler(t, failing).Today(), "/api/students/today")
	if broken.Code != http.StatusInternalServerError {
		t.Errorf("failing store status = %d, want 500", broken.Code)
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

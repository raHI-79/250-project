package handler

import (
	"github.com/raHI-79/250-project/backend/internal/auth"
	"github.com/raHI-79/250-project/backend/internal/store"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func dayItems(t *testing.T, rec *httptest.ResponseRecorder) []map[string]any {
	t.Helper()
	raw, ok := decodeBody(t, rec)["items"].([]any)
	if !ok {
		t.Fatalf("the response %q carries no items", rec.Body.String())
	}
	items := make([]map[string]any, 0, len(raw))
	for _, entry := range raw {
		item, ok := entry.(map[string]any)
		if !ok {
			t.Fatalf("an item of %q is not an object", rec.Body.String())
		}
		items = append(items, item)
	}
	return items
}

func TestTeacherUpcomingAsksForOneDay(t *testing.T) {
	silenceLog(t)
	items := newFakeScheduleStore()
	items.items = []store.ScheduleItem{itemOn("2026-11-06", store.ItemAwaiting, 4)}
	teacher := newTeacherHandler(t, items, &fakeNotifier{})
	request := httptest.NewRequest(http.MethodGet, "/api/teacher/upcoming?date=2026-11-06", nil)
	rec := serve(teacher.Upcoming(), withCurrentUser(request, teacherUserID, auth.RoleTeacher))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body = %q)", rec.Code, rec.Body.String())
	}
	body := decodeBody(t, rec)
	if body["date"] != "2026-11-06" {
		t.Errorf("date = %v, want the day that was asked for", body["date"])
	}
	if body["past"] != false || body["read_only"] != false {
		t.Errorf("a future day came back as read only: %q", rec.Body.String())
	}
	if len(items.askedDates) == 0 && len(items.items) == 0 {
		t.Error("the store was not asked for the day")
	}
	marks := dayItems(t, rec)
	if len(marks) != 1 {
		t.Fatalf("items = %d, want the one class of that day", len(marks))
	}
	actions, ok := marks[0]["allowed_actions"].([]any)
	if !ok || len(actions) == 0 {
		t.Errorf("a future day lost its actions: %q", rec.Body.String())
	}
}

func TestTeacherUpcomingReadsAPastDayWithItsFinalStatus(t *testing.T) {
	silenceLog(t)
	items := newFakeScheduleStore()
	items.items = []store.ScheduleItem{
		itemOn("2020-01-02", store.ItemConfirmed, 4),
		itemOn("2020-01-02", store.ItemCancelled, 5),
	}
	teacher := newTeacherHandler(t, items, &fakeNotifier{})
	request := httptest.NewRequest(http.MethodGet, "/api/teacher/upcoming?date=2020-01-02", nil)
	rec := serve(teacher.Upcoming(), withCurrentUser(request, teacherUserID, auth.RoleTeacher))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body = %q)", rec.Code, rec.Body.String())
	}
	body := decodeBody(t, rec)
	if body["past"] != true || body["read_only"] != true {
		t.Errorf("a past day is not read only: %q", rec.Body.String())
	}
	marks := dayItems(t, rec)
	if len(marks) != 2 {
		t.Fatalf("items = %d, want the kept rows of that day including the cancelled one", len(marks))
	}
	if marks[0]["final_status"] != StatusHeld {
		t.Errorf("final status = %v, want %q", marks[0]["final_status"], StatusHeld)
	}
	if marks[1]["final_status"] != StatusCancelledAt {
		t.Errorf("final status = %v, want %q", marks[1]["final_status"], StatusCancelledAt)
	}
	for _, mark := range marks {
		if _, ok := mark["allowed_actions"]; ok {
			t.Errorf("a past day still offers actions: %q", rec.Body.String())
		}
	}
}

func TestTeacherUpcomingPagesOneDayAtATime(t *testing.T) {
	silenceLog(t)
	items := newFakeScheduleStore()
	items.items = []store.ScheduleItem{
		itemOn("2026-11-07", store.ItemAwaiting, 4),
		itemOn("2026-11-08", store.ItemAwaiting, 5),
		itemOn("2026-11-09", store.ItemAwaiting, 6),
	}
	items.horizon = "2026-11-30"
	teacher := newTeacherHandler(t, items, &fakeNotifier{})
	request := httptest.NewRequest(http.MethodGet,
		"/api/teacher/upcoming?from=2026-11-07&to=2026-11-30&limit_days=1", nil)
	rec := serve(teacher.Upcoming(), withCurrentUser(request, teacherUserID, auth.RoleTeacher))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body = %q)", rec.Code, rec.Body.String())
	}
	body := decodeBody(t, rec)
	days, ok := body["days"].([]any)
	if !ok || len(days) != 1 {
		t.Fatalf("days = %v, want exactly one group (body = %q)", body["days"], rec.Body.String())
	}
	if body["has_more"] != true {
		t.Error("has_more = false, want the page to say that more days are waiting")
	}
	if body["horizon"] != "2026-11-30" {
		t.Errorf("horizon = %v, want the generated horizon", body["horizon"])
	}
}

func TestTeacherFinalRoutineReadsADay(t *testing.T) {
	silenceLog(t)
	items := newFakeScheduleStore()
	items.items = []store.ScheduleItem{itemOn("2026-11-06", store.ItemConfirmed, 4)}
	teacher := newTeacherHandler(t, items, &fakeNotifier{})
	request := httptest.NewRequest(http.MethodGet,
		"/api/teacher/upcoming/final?date=2026-11-06", nil)
	rec := serve(teacher.FinalRoutine(), withCurrentUser(request, teacherUserID, auth.RoleTeacher))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body = %q)", rec.Code, rec.Body.String())
	}
	if got := decodeBody(t, rec)["date"]; got != "2026-11-06" {
		t.Errorf("date = %v, want the day that was asked for", got)
	}
}

func TestTeacherUpcomingRefusesABrokenDate(t *testing.T) {
	silenceLog(t)
	items := newFakeScheduleStore()
	teacher := newTeacherHandler(t, items, &fakeNotifier{})
	request := httptest.NewRequest(http.MethodGet, "/api/teacher/upcoming?date=yesterday", nil)
	rec := serve(teacher.Upcoming(), withCurrentUser(request, teacherUserID, auth.RoleTeacher))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 (body = %q)", rec.Code, rec.Body.String())
	}
}

func TestTeacherEditsTheSlotAndTellsEverybody(t *testing.T) {
	silenceLog(t)
	tests := []struct {
		name       string
		body       string
		storeErr   error
		wantStatus int
		wantError  string
	}{
		{
			name:       "a new day and time",
			body:       `{"item_date":"2026-11-20","start_time":"09:00","end_time":"09:45"}`,
			wantStatus: http.StatusOK,
		},
		{
			name:       "a new room",
			body:       `{"mode":"room","room":"1004"}`,
			wantStatus: http.StatusOK,
		},
		{
			name:       "online with a link",
			body:       `{"mode":"online","meeting_link":"https://meet.example.com/a"}`,
			wantStatus: http.StatusOK,
		},
		{
			name:       "one time without the other",
			body:       `{"start_time":"09:00"}`,
			wantStatus: http.StatusBadRequest,
			wantError:  MessageTimesTogether,
		},
		{
			name:       "a day that is gone",
			body:       `{"item_date":"2020-01-01"}`,
			wantStatus: http.StatusBadRequest,
			wantError:  MessagePastDate,
		},
		{
			name:       "a room that is taken",
			body:       `{"mode":"room","room":"1004"}`,
			storeErr:   &store.SlotConflict{Reason: store.ConflictRoom, Room: "1004", Batch: "CSE-23", Section: "B", StartTime: "08:00", EndTime: "09:30"},
			wantStatus: http.StatusConflict,
		},
		{
			name:       "a class that already started",
			body:       `{"mode":"room","room":"1004"}`,
			storeErr:   store.ErrItemStarted,
			wantStatus: http.StatusConflict,
			wantError:  MessageItemStarted,
		},
		{
			name:       "not this teacher",
			body:       `{"mode":"room","room":"1004"}`,
			storeErr:   store.ErrItemNotFound,
			wantStatus: http.StatusNotFound,
			wantError:  MessageItemNotFound,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			items := newFakeScheduleStore()
			items.err = test.storeErr
			items.crs = []int64{crUserID}
			items.item = plannedItem()
			items.item.Snapshot.Exists = true
			items.periods = testPeriods()
			alerts := &fakeNotifier{}
			teacher := newTeacherHandler(t, items, alerts)
			rec := scheduleCall(t, teacher.UpdateItem(), http.MethodPatch, "/api/teacher/items/4",
				"4", test.body, teacherUserID, auth.RoleTeacher)
			if rec.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d (body = %q)", rec.Code, test.wantStatus, rec.Body.String())
			}
			if test.wantStatus != http.StatusOK {
				if test.wantError != "" {
					if got := errorMessage(t, rec); got != test.wantError {
						t.Errorf("error = %q, want %q", got, test.wantError)
					}
				}
				if alerts.calls != 0 || alerts.broadcasts != 0 {
					t.Error("a refused edit still told somebody")
				}
				return
			}
			if alerts.calls != 1 || alerts.userID != crUserID {
				t.Errorf("notifications = %d to user %d, want one to the class representative",
					alerts.calls, alerts.userID)
			}
			if alerts.broadcasts != 1 {
				t.Errorf("broadcasts = %d, want one to the students of the published class",
					alerts.broadcasts)
			}
		})
	}
}

func TestTeacherEditKeepsTheTimesWithinThePeriods(t *testing.T) {
	silenceLog(t)
	items := newFakeScheduleStore()
	items.periods = testPeriods()
	teacher := newTeacherHandler(t, items, &fakeNotifier{})
	rec := scheduleCall(t, teacher.UpdateItem(), http.MethodPatch, "/api/teacher/items/4", "4",
		`{"start_time":"05:00","end_time":"05:45"}`, teacherUserID, auth.RoleTeacher)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 (body = %q)", rec.Code, rec.Body.String())
	}
}

func TestCRItemPicksTheTeacherOfTheCourse(t *testing.T) {
	silenceLog(t)
	tests := []struct {
		name       string
		teachers   []store.CourseTeacher
		body       string
		wantStatus int
		wantTeache int64
	}{
		{
			name:       "one teacher is chosen by itself",
			teachers:   []store.CourseTeacher{{ID: 3, Name: "Only Teacher"}},
			body:       `{"kind":"class","course_id":7,"item_date":"2026-11-20","start_time":"09:00","end_time":"09:45","mode":"room","room":"1003"}`,
			wantStatus: http.StatusCreated,
			wantTeache: 3,
		},
		{
			name:     "the chosen one of several is kept",
			teachers: []store.CourseTeacher{{ID: 3, Name: "First"}, {ID: 5, Name: "Second"}},
			body: `{"kind":"class","course_id":7,"teacher_id":5,"item_date":"2026-11-20",` +
				`"start_time":"09:00","end_time":"09:45","mode":"room","room":"1003"}`,
			wantStatus: http.StatusCreated,
			wantTeache: 5,
		},
		{
			name:     "a teacher off the course is refused",
			teachers: []store.CourseTeacher{{ID: 3, Name: "First"}},
			body: `{"kind":"class","course_id":7,"teacher_id":9,"item_date":"2026-11-20",` +
				`"start_time":"09:00","end_time":"09:45","mode":"room","room":"1003"}`,
			wantStatus: http.StatusBadRequest,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			items := newFakeScheduleStore()
			items.courseTeachers = test.teachers
			items.periods = testPeriods()
			cr := newCRHandler(t, items, &fakeNotifier{})
			request := httptest.NewRequest(http.MethodPost, "/api/cr/items", strings.NewReader(test.body))
			request.Header.Set("Content-Type", "application/json")
			rec := serve(cr.CreateItem(), withCurrentUser(request, crUserID, auth.RoleCR))
			if rec.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d (body = %q)", rec.Code, test.wantStatus, rec.Body.String())
			}
			if test.wantStatus != http.StatusCreated {
				if got := errorMessage(t, rec); got != MessageTeacherNotOnCourse {
					t.Errorf("error = %q, want %q", got, MessageTeacherNotOnCourse)
				}
				if len(items.created) != 0 {
					t.Error("a refused item was still created")
				}
				return
			}
			if len(items.created) != 1 {
				t.Fatalf("created = %d items, want one", len(items.created))
			}
			if items.created[0].TeacherID != test.wantTeache {
				t.Errorf("teacher = %d, want %d", items.created[0].TeacherID, test.wantTeache)
			}
		})
	}
}

package handler

import (
	"github.com/raHI-79/250-project/backend/internal/store"
	"net/http"
	"testing"
)

func crPlannerCall(t *testing.T, classes *fakeCRStore, build func(*CR) http.HandlerFunc, target string) (int, map[string]any) {
	t.Helper()
	handler := newCRHandler(t, classes, &fakeNotifier{})
	rec := crCall(t, build(handler), http.MethodGet, target, "")
	if rec.Code != http.StatusOK {
		return rec.Code, nil
	}
	return rec.Code, decodeBody(t, rec)
}

func TestCRCourses(t *testing.T) {
	silenceLog(t)
	classes := newCRStore()
	classes.slots = []store.CourseSlotRow{
		{CourseID: 4, CourseCode: "CSE-233", CourseTitle: "Digital Logic Design", TeacherID: 6, TeacherUserID: 44, TeacherName: "First Teacher", DayOfWeek: 1, StartTime: "10:30", EndTime: "12:45", Room: "1003", Type: "lab"},
		{CourseID: 4, CourseCode: "CSE-233", CourseTitle: "Digital Logic Design", TeacherID: 6, TeacherUserID: 44, TeacherName: "First Teacher", DayOfWeek: 3, StartTime: "09:00", EndTime: "10:00", Room: "1003", Type: "class"},
		{CourseID: 7, CourseCode: "CSE-250", CourseTitle: "Software Engineering", TeacherID: 8, TeacherUserID: 45, TeacherName: "Second Teacher", DayOfWeek: 2, StartTime: "08:00", EndTime: "09:00", Room: "1004", Type: "class"},
	}
	code, payload := crPlannerCall(t, classes, (*CR).Courses, "/api/cr/courses")
	if code != http.StatusOK {
		t.Fatalf("status = %d, want 200", code)
	}
	if classes.askedBatch != crBatch || classes.askedSection != crSection {
		t.Errorf("the store was asked for %q %q, want the own batch and section", classes.askedBatch, classes.askedSection)
	}
	courses, ok := payload["items"].([]any)
	if !ok || len(courses) != 2 {
		t.Fatalf("items = %v, want two courses", payload["items"])
	}
	first := courses[0].(map[string]any)
	if first["code"] != "CSE-233" || first["title"] != "Digital Logic Design" {
		t.Errorf("first course = %v, want CSE-233", first)
	}
	teacher, ok := first["teacher"].(map[string]any)
	if !ok || teacher["id"] != float64(6) || teacher["user_id"] != float64(44) || teacher["name"] != "First Teacher" {
		t.Errorf("teacher = %v, want the teacher ids and name", first["teacher"])
	}
	slots, ok := first["slots"].([]any)
	if !ok || len(slots) != 2 {
		t.Fatalf("slots = %v, want the two weekly slots", first["slots"])
	}
	monday := slots[0].(map[string]any)
	if monday["day_of_week"] != float64(1) || monday["day"] != "Monday" || monday["room"] != "1003" || monday["type"] != "lab" {
		t.Errorf("slot = %v, want the Monday lab", monday)
	}
	if second := courses[1].(map[string]any); second["code"] != "CSE-250" || len(second["slots"].([]any)) != 1 {
		t.Errorf("second course = %v, want CSE-250 with one slot", second)
	}
}

func TestCRDay(t *testing.T) {
	silenceLog(t)
	tests := []struct {
		name       string
		query      string
		wantStatus int
		wantError  string
	}{
		{name: "a chosen date", query: "?date=2026-10-20", wantStatus: http.StatusOK},
		{name: "without a date", query: "", wantStatus: http.StatusBadRequest, wantError: MessageDateRequired},
		{name: "a broken date", query: "?date=soon", wantStatus: http.StatusBadRequest, wantError: "date must look like 2026-10-17."},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			classes := newCRStore()
			classes.day = []store.DayEntryRow{
				{Source: "fixed", Section: "A", CourseCode: "CSE-233", CourseTitle: "Digital Logic Design", TeacherName: "First Teacher", StartTime: "10:30", EndTime: "12:45", Room: "1003", Kind: "lab", Status: StatusScheduled},
				{Source: "event", Section: "ALL", CourseCode: "CSE-250", CourseTitle: "Software Engineering", TeacherName: "Second Teacher", StartTime: "13:00", EndTime: "14:00", Room: "2001", Kind: "exam", Status: store.EventStatusPending},
			}
			handler := newCRHandler(t, classes, &fakeNotifier{})
			rec := crCall(t, handler.Day(), http.MethodGet, "/api/cr/day"+test.query, "")
			if rec.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d (body = %q)", rec.Code, test.wantStatus, rec.Body.String())
			}
			if test.wantStatus != http.StatusOK {
				if got := errorMessage(t, rec); got != test.wantError {
					t.Errorf("error = %q, want %q", got, test.wantError)
				}
				return
			}
			payload := decodeBody(t, rec)
			if payload["date"] != "2026-10-20" || payload["weekday"] != "Tuesday" || payload["batch"] != crBatch {
				t.Errorf("payload = %v, want the date, the weekday and the batch", payload)
			}
			if classes.askedBatch != crBatch || classes.askedDate != "2026-10-20" {
				t.Errorf("the store was asked for %q on %q, want the own batch on the chosen date", classes.askedBatch, classes.askedDate)
			}
			list := items(t, rec)
			if len(list) != 2 {
				t.Fatalf("the day has %d entries, want 2", len(list))
			}
			if list[0]["source"] != "fixed" || list[0]["status"] != StatusScheduled || list[0]["kind"] != "lab" {
				t.Errorf("first entry = %v, want the fixed lab", list[0])
			}
			if list[1]["source"] != "event" || list[1]["status"] != store.EventStatusPending {
				t.Errorf("second entry = %v, want the pending event", list[1])
			}
		})
	}
}

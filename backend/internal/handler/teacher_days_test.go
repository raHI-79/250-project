package handler

import (
	"encoding/json"
	"github.com/raHI-79/250-project/backend/internal/auth"
	"github.com/raHI-79/250-project/backend/internal/config"
	"github.com/raHI-79/250-project/backend/internal/notify"
	"github.com/raHI-79/250-project/backend/internal/store"
	"github.com/raHI-79/250-project/backend/internal/validate"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"
)

type dayPayload struct {
	Date    string           `json:"date"`
	Weekday string           `json:"weekday"`
	Items   []map[string]any `json:"items"`
}

func daysConfig(t *testing.T, daysAhead int) *config.Config {
	t.Helper()
	cfg := accountConfig(t, "test")
	cfg.DaysAhead = daysAhead
	return cfg
}

func dayAfter(cfg *config.Config, days int) string {
	return validate.DateString(cfg.Now().AddDate(0, 0, days))
}

func itemOn(date, status string, id int64) store.ScheduleItem {
	item := plannedItem()
	item.ID = id
	item.ItemDate = date
	item.Status = status
	return item
}

func responseDays(t *testing.T, rec *httptest.ResponseRecorder) []dayPayload {
	t.Helper()
	var payload struct {
		Days []dayPayload `json:"days"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("body is not valid JSON: %v (body = %q)", err, rec.Body.String())
	}
	return payload.Days
}

func TestTeacherUpcomingGroupsEveryDayOfTheHorizon(t *testing.T) {
	silenceLog(t)
	cfg := daysConfig(t, 7)
	items := newFakeScheduleStore()
	exam := itemOn(dayAfter(cfg, 3), store.ItemScheduled, 9)
	exam.Origin = store.OriginTeacherExam
	items.items = []store.ScheduleItem{
		itemOn(dayAfter(cfg, 1), store.ItemAwaiting, 5),
		itemOn(dayAfter(cfg, 1), store.ItemConfirmed, 6),
		itemOn(dayAfter(cfg, 4), store.ItemNoResponse, 7),
		itemOn(dayAfter(cfg, 2), store.ItemCancelled, 8),
		exam,
	}
	teacher := NewTeacher(cfg, items, &fakeNotifier{})
	rec := scheduleCall(t, teacher.Upcoming(), http.MethodGet, "/api/teacher/upcoming",
		"", "", teacherUserID, auth.RoleTeacher)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body = %q)", rec.Code, rec.Body.String())
	}
	payload := decodeBody(t, rec)
	if payload["from"] != dayAfter(cfg, 0) || payload["to"] != dayAfter(cfg, 7) {
		t.Errorf("range = %v to %v, want today through the horizon", payload["from"], payload["to"])
	}
	days := responseDays(t, rec)
	if len(days) != 2 {
		t.Fatalf("days = %d, want tomorrow and the fourth day (body = %q)", len(days), rec.Body.String())
	}
	if days[0].Date != dayAfter(cfg, 1) || len(days[0].Items) != 2 {
		t.Errorf("the first day = %q with %d items, want tomorrow with two",
			days[0].Date, len(days[0].Items))
	}
	if days[0].Weekday == "" {
		t.Error("a day carries no weekday name")
	}
	if days[1].Date != dayAfter(cfg, 4) || len(days[1].Items) != 1 {
		t.Errorf("the second day = %q with %d items, want the fourth day with one",
			days[1].Date, len(days[1].Items))
	}
}

func TestTeacherUpcomingKeepsTodayUndecidedAndNotStarted(t *testing.T) {
	cfg := daysConfig(t, 7)
	now := time.Date(2026, 11, 1, 10, 0, 0, 0, cfg.Location)
	today := validate.DateString(now)
	started := itemOn(today, store.ItemAwaiting, 1)
	started.StartTime, started.EndTime = "08:00", "09:30"
	open := itemOn(today, store.ItemAwaiting, 2)
	open.StartTime, open.EndTime = "14:00", "15:30"
	decided := itemOn(today, store.ItemConfirmed, 3)
	decided.StartTime, decided.EndTime = "16:00", "17:30"
	tomorrow := itemOn(validate.DateString(now.AddDate(0, 0, 1)), store.ItemConfirmed, 4)
	kept := decidableItems([]store.ScheduleItem{started, open, decided, tomorrow}, now, today)
	if len(kept) != 2 {
		t.Fatalf("kept = %d items, want the open class of today and tomorrow", len(kept))
	}
	if kept[0].ID != 2 || kept[1].ID != 4 {
		t.Errorf("kept = %d and %d, want 2 and 4", kept[0].ID, kept[1].ID)
	}
}

func TestTeacherUpcomingRefusesABackwardsRange(t *testing.T) {
	silenceLog(t)
	teacher := NewTeacher(daysConfig(t, 7), newFakeScheduleStore(), &fakeNotifier{})
	rec := scheduleCall(t, teacher.Upcoming(), http.MethodGet,
		"/api/teacher/upcoming?from=2026-11-05&to=2026-11-01", "", "", teacherUserID, auth.RoleTeacher)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 (body = %q)", rec.Code, rec.Body.String())
	}
	if got := errorMessage(t, rec); got != MessageRangeOrder {
		t.Errorf("error = %q, want %q", got, MessageRangeOrder)
	}
}

func TestTeacherDecidesADayAfterTomorrow(t *testing.T) {
	silenceLog(t)
	cfg := daysConfig(t, 7)
	later := dayAfter(cfg, 3)
	items := newFakeScheduleStore()
	items.item = itemOn(later, store.ItemAwaiting, 12)
	items.items = []store.ScheduleItem{items.item}
	items.crs = []int64{crUserID}
	alerts := &fakeNotifier{}
	teacher := NewTeacher(cfg, items, alerts)
	listed := scheduleCall(t, teacher.Upcoming(), http.MethodGet, "/api/teacher/upcoming",
		"", "", teacherUserID, auth.RoleTeacher)
	days := responseDays(t, listed)
	if len(days) != 1 || days[0].Date != later {
		t.Fatalf("days = %+v, want only the fourth day", days)
	}
	id := strconv.FormatInt(12, 10)
	rec := scheduleCall(t, teacher.Confirm(), http.MethodPost, "/api/teacher/items/"+id+"/confirm",
		id, `{"mode":"room","room":"1003"}`, teacherUserID, auth.RoleTeacher)
	if rec.Code != http.StatusOK {
		t.Fatalf("confirm status = %d, want 200 (body = %q)", rec.Code, rec.Body.String())
	}
	if len(items.confirmed) != 1 || items.confirmed[0] != "room:1003:" {
		t.Errorf("the store received %v, want the confirmed room", items.confirmed)
	}
	if alerts.calls != 1 || alerts.userID != crUserID {
		t.Errorf("notifications = %d to user %d, want one to the class representative",
			alerts.calls, alerts.userID)
	}
}

func TestTeacherFinalRoutineGroupsDays(t *testing.T) {
	silenceLog(t)
	cfg := daysConfig(t, 7)
	items := newFakeScheduleStore()
	exam := itemOn(dayAfter(cfg, 2), store.ItemScheduled, 21)
	exam.Origin = store.OriginTeacherExam
	exam.Kind = store.KindExam
	byCR := itemOn(dayAfter(cfg, 2), store.ItemScheduled, 22)
	byCR.Origin = store.OriginCR
	items.items = []store.ScheduleItem{
		itemOn(dayAfter(cfg, 0), store.ItemConfirmed, 20),
		exam,
		byCR,
	}
	teacher := NewTeacher(cfg, items, &fakeNotifier{})
	rec := scheduleCall(t, teacher.FinalRoutine(), http.MethodGet, "/api/teacher/upcoming/final",
		"", "", teacherUserID, auth.RoleTeacher)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body = %q)", rec.Code, rec.Body.String())
	}
	days := responseDays(t, rec)
	if len(days) != 2 {
		t.Fatalf("days = %d, want today and the second day (body = %q)", len(days), rec.Body.String())
	}
	if days[0].Date != dayAfter(cfg, 0) || len(days[0].Items) != 1 {
		t.Errorf("the first day = %q with %d items, want today with one", days[0].Date, len(days[0].Items))
	}
	if len(days[1].Items) != 2 {
		t.Errorf("the second day carries %d items, want the exam and the item of the class representative",
			len(days[1].Items))
	}
}

func TestTeacherCancelFromFinalReachesEverybody(t *testing.T) {
	silenceLog(t)
	tests := []struct {
		name   string
		cancel func(*Teacher) http.HandlerFunc
		path   string
	}{
		{name: "a class", cancel: (*Teacher).Cancel, path: "/api/teacher/items/4/cancel"},
		{name: "an exam", cancel: (*Teacher).CancelExam, path: "/api/teacher/exams/4/cancel"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			items := newFakeScheduleStore()
			items.crs = []int64{crUserID}
			items.dropUsers = []int64{77, 78}
			items.item.Snapshot = store.ItemSnapshot{Exists: true, Status: store.ItemConfirmed}
			alerts := &fakeNotifier{}
			teacher := newTeacherHandler(t, items, alerts)
			rec := scheduleCall(t, test.cancel(teacher), http.MethodPost, test.path, "4",
				`{"reason":""}`, teacherUserID, auth.RoleTeacher)
			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200 (body = %q)", rec.Code, rec.Body.String())
			}
			if len(items.cancelled) != 1 || items.cancelled[0] != "" {
				t.Errorf("the store received %v, want one cancellation without a reason", items.cancelled)
			}
			if alerts.broadcasts != 1 {
				t.Errorf("broadcasts = %d, want one to the batch of the published class", alerts.broadcasts)
			}
			if len(items.dropCourses) != 1 {
				t.Fatalf("the store was asked %d times for the drop course students, want once",
					len(items.dropCourses))
			}
			reached := make(map[int64]bool)
			for _, call := range alerts.direct {
				reached[call.userID] = true
				if call.userID == 77 && call.link.Kind != notify.LinkRoutine {
					t.Errorf("the drop course link = %+v, want the student routine", call.link)
				}
			}
			for _, userID := range []int64{crUserID, 77, 78} {
				if !reached[userID] {
					t.Errorf("the user %d heard nothing about the cancelled class", userID)
				}
			}
		})
	}
}

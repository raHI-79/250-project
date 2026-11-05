package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/raHI-79/250-project/backend/internal/config"
	"github.com/raHI-79/250-project/backend/internal/middleware"
	"github.com/raHI-79/250-project/backend/internal/settings"
	"github.com/raHI-79/250-project/backend/internal/store"
)

type fakeCalendar struct {
	entries []store.Holiday
	saved   store.Holiday
	removed []store.ScheduleItem
	deleted string
	crs     []int64
	saveErr error
	delErr  error
	listErr error
}

func (f *fakeCalendar) ListHolidays(ctx context.Context, from string) ([]store.Holiday, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	kept := make([]store.Holiday, 0, len(f.entries))
	for _, entry := range f.entries {
		if entry.Date >= from {
			kept = append(kept, entry)
		}
	}
	return kept, nil
}

func (f *fakeCalendar) SaveHoliday(ctx context.Context, holiday store.Holiday) ([]store.ScheduleItem, error) {
	if f.saveErr != nil {
		return nil, f.saveErr
	}
	f.saved = holiday
	return f.removed, nil
}

func (f *fakeCalendar) DeleteHoliday(ctx context.Context, date string) error {
	if f.delErr != nil {
		return f.delErr
	}
	f.deleted = date
	return nil
}

func (f *fakeCalendar) ListActiveCRs(ctx context.Context, batch, section string) ([]int64, error) {
	return f.crs, nil
}

func holidayConfig(t *testing.T) *config.Config {
	t.Helper()
	loc, err := time.LoadLocation("Asia/Dhaka")
	if err != nil {
		t.Fatalf("could not load the timezone: %v", err)
	}
	cfg := &config.Config{Timezone: "Asia/Dhaka", Location: loc}
	cfg.Freeze(time.Date(2026, 11, 5, 23, 57, 0, 0, loc))
	return cfg
}

type fakeSettings struct {
	values map[string]string
	saved  map[string]string
	by     int64
}

func (f *fakeSettings) AllSettings(ctx context.Context) (map[string]string, error) {
	return f.values, nil
}

func (f *fakeSettings) SaveSettings(ctx context.Context, values map[string]string, by int64) error {
	if f.saved == nil {
		f.saved = make(map[string]string, len(values))
	}
	for key, value := range values {
		f.saved[key] = value
	}
	f.by = by
	return nil
}

func newHolidays(t *testing.T, calendar *fakeCalendar, alerts Notifier) *Holidays {
	t.Helper()
	return newHolidaysWith(t, calendar, alerts, &fakeSettings{})
}

func newHolidaysWith(t *testing.T, calendar *fakeCalendar, alerts Notifier,
	keys *fakeSettings) *Holidays {
	t.Helper()
	values := settings.New(keys, map[string]string{settings.KeyOffDays: settings.DefaultOffDays})
	return NewHolidays(holidayConfig(t), calendar, values, alerts)
}

func calendarCall(t *testing.T, handlerFunc http.HandlerFunc, method, target, date,
	body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, target, nil)
	if body != "" {
		req = httptest.NewRequest(method, target, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	}
	if date != "" {
		req.SetPathValue("date", date)
	}
	req = req.WithContext(middleware.WithUser(req.Context(), middleware.User{ID: 9, Role: "admin"}))
	rec := httptest.NewRecorder()
	handlerFunc.ServeHTTP(rec, req)
	return rec
}

func TestCalendarListsTheOffDaysAndTheHolidays(t *testing.T) {
	calendar := &fakeCalendar{entries: []store.Holiday{
		{Date: "2026-11-08", Name: "Victory Day"},
		{Date: "2026-11-14", Name: "Make-up Saturday", WorkingDay: true, Follows: 0},
	}}
	rec := calendarCall(t, newHolidays(t, calendar, &fakeNotifier{}).Calendar(),
		http.MethodGet, "/api/admin/calendar", "", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body = %q)", rec.Code, rec.Body.String())
	}
	payload := decodeBody(t, rec)
	offDays, ok := payload["off_days"].([]any)
	if !ok || len(offDays) != 2 || offDays[0] != float64(5) || offDays[1] != float64(6) {
		t.Errorf("off_days = %v, want Friday and Saturday", payload["off_days"])
	}
	names, ok := payload["off_day_names"].([]any)
	if !ok || names[0] != "Friday" || names[1] != "Saturday" {
		t.Errorf("off_day_names = %v, want Friday and Saturday", payload["off_day_names"])
	}
	holidays, ok := payload["holidays"].([]any)
	if !ok || len(holidays) != 2 {
		t.Fatalf("holidays = %v, want both entries", payload["holidays"])
	}
	first := holidays[0].(map[string]any)
	if first["label"] != "Sun 8 Nov 2026" {
		t.Errorf("the label = %v, want \"Sun 8 Nov 2026\"", first["label"])
	}
	if first["working_day"] != false || first["name"] != "Victory Day" {
		t.Errorf("the first entry = %v, want the plain holiday", first)
	}
	second := holidays[1].(map[string]any)
	if second["working_day"] != true || second["follows_weekday_name"] != "Sunday" {
		t.Errorf("the second entry = %v, want a working day that follows Sunday", second)
	}
}

func TestSavingACalendarEntryChecksWhatItNeeds(t *testing.T) {
	silenceLog(t)
	tests := []struct {
		name       string
		body       string
		wantStatus int
		wantError  string
	}{
		{
			name:       "a holiday with a name",
			body:       `{"date":"2026-11-08","name":"Victory Day"}`,
			wantStatus: http.StatusOK,
		},
		{
			name:       "a holiday without a name",
			body:       `{"date":"2026-11-08","name":"  "}`,
			wantStatus: http.StatusBadRequest,
			wantError:  MessageHolidayName,
		},
		{
			name:       "a working day with the weekday it follows",
			body:       `{"date":"2026-11-14","working_day":true,"follows_weekday":0}`,
			wantStatus: http.StatusOK,
		},
		{
			name:       "a working day without a weekday",
			body:       `{"date":"2026-11-14","working_day":true}`,
			wantStatus: http.StatusBadRequest,
			wantError:  MessageHolidayFollows,
		},
		{
			name:       "a working day with a weekday out of range",
			body:       `{"date":"2026-11-14","working_day":true,"follows_weekday":7}`,
			wantStatus: http.StatusBadRequest,
			wantError:  MessageHolidayFollows,
		},
		{
			name:       "a month-first date",
			body:       `{"date":"11/08/2026","name":"Victory Day"}`,
			wantStatus: http.StatusBadRequest,
			wantError:  MessageDateFormat,
		},
		{
			name:       "no date at all",
			body:       `{"name":"Victory Day"}`,
			wantStatus: http.StatusBadRequest,
			wantError:  MessageDateFormat,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			calendar := &fakeCalendar{}
			rec := calendarCall(t, newHolidays(t, calendar, &fakeNotifier{}).Save(),
				http.MethodPut, "/api/admin/calendar/holidays", "", test.body)
			if rec.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d (body = %q)", rec.Code, test.wantStatus, rec.Body.String())
			}
			if test.wantError != "" {
				if got := decodeBody(t, rec)["error"]; got != test.wantError {
					t.Errorf("error = %v, want %q", got, test.wantError)
				}
				return
			}
			if calendar.saved.Date == "" {
				t.Error("the store received no entry")
			}
			if calendar.saved.CreatedBy != 9 {
				t.Errorf("the entry was saved by %d, want the signed in admin", calendar.saved.CreatedBy)
			}
		})
	}
}

func TestAHolidayTellsTheTeacherAndTheClassRepresentatives(t *testing.T) {
	silenceLog(t)
	calendar := &fakeCalendar{
		crs: []int64{41, 42},
		removed: []store.ScheduleItem{{
			ID:            7,
			Batch:         "CSE-23",
			Section:       "A",
			CourseCode:    "CSE 2243",
			CourseTitle:   "Compilers",
			ItemDate:      "2026-11-08",
			TeacherUserID: 31,
		}},
	}
	alerts := &fakeNotifier{}
	rec := calendarCall(t, newHolidays(t, calendar, alerts).Save(), http.MethodPut,
		"/api/admin/calendar/holidays", "", `{"date":"2026-11-08","name":"Strike"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body = %q)", rec.Code, rec.Body.String())
	}
	if got := decodeBody(t, rec)["removed_items"]; got != float64(1) {
		t.Errorf("removed_items = %v, want 1", got)
	}
	if len(alerts.sent) != 3 {
		t.Fatalf("sent %d notifications, want the teacher and both representatives", len(alerts.sent))
	}
	wanted := map[int64]bool{31: false, 41: false, 42: false}
	for _, sent := range alerts.sent {
		if _, ok := wanted[sent.userID]; !ok {
			t.Errorf("notified %d, want only the teacher and the representatives", sent.userID)
			continue
		}
		wanted[sent.userID] = true
		if sent.title != TitleHolidayCleared {
			t.Errorf("the title = %q, want %q", sent.title, TitleHolidayCleared)
		}
		if !strings.Contains(sent.body, "Sun 8 Nov 2026") {
			t.Errorf("the body = %q, want the readable date in it", sent.body)
		}
		if !strings.Contains(sent.body, "Strike") {
			t.Errorf("the body = %q, want the holiday name in it", sent.body)
		}
	}
	for userID, told := range wanted {
		if !told {
			t.Errorf("nobody told %d about the holiday", userID)
		}
	}
}

func TestAWorkingDayTellsNobody(t *testing.T) {
	calendar := &fakeCalendar{crs: []int64{41}}
	alerts := &fakeNotifier{}
	rec := calendarCall(t, newHolidays(t, calendar, alerts).Save(), http.MethodPut,
		"/api/admin/calendar/holidays", "",
		`{"date":"2026-11-14","working_day":true,"follows_weekday":0}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body = %q)", rec.Code, rec.Body.String())
	}
	if len(alerts.sent) != 0 {
		t.Errorf("sent %d notifications, want none for a working day", len(alerts.sent))
	}
	if calendar.saved.Follows != 0 || !calendar.saved.WorkingDay {
		t.Errorf("the entry = %+v, want a working day that follows Sunday", calendar.saved)
	}
	if calendar.saved.Name != "Working day" {
		t.Errorf("the name = %q, want the standing name of a working day", calendar.saved.Name)
	}
}

func TestDeletingACalendarEntry(t *testing.T) {
	silenceLog(t)
	calendar := &fakeCalendar{}
	rec := calendarCall(t, newHolidays(t, calendar, &fakeNotifier{}).Delete(),
		http.MethodDelete, "/api/admin/calendar/holidays/2026-11-08", "2026-11-08", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body = %q)", rec.Code, rec.Body.String())
	}
	if calendar.deleted != "2026-11-08" {
		t.Errorf("deleted %q, want 2026-11-08", calendar.deleted)
	}
	missing := &fakeCalendar{delErr: store.ErrNotAHoliday}
	gone := calendarCall(t, newHolidays(t, missing, &fakeNotifier{}).Delete(),
		http.MethodDelete, "/api/admin/calendar/holidays/2026-11-08", "2026-11-08", "")
	if gone.Code != http.StatusNotFound {
		t.Errorf("status for an unknown entry = %d, want 404", gone.Code)
	}
	wrong := calendarCall(t, newHolidays(t, &fakeCalendar{}, &fakeNotifier{}).Delete(),
		http.MethodDelete, "/api/admin/calendar/holidays/11-08-2026", "11-08-2026", "")
	if wrong.Code != http.StatusBadRequest {
		t.Errorf("status for a date in the wrong order = %d, want 400", wrong.Code)
	}
}

func TestSavingTheWeeklyOffDays(t *testing.T) {
	silenceLog(t)
	tests := []struct {
		name       string
		body       string
		wantStatus int
		wantSaved  string
	}{
		{name: "Friday and Saturday", body: `{"off_days":[5,6]}`, wantStatus: http.StatusOK, wantSaved: "5,6"},
		{name: "only Friday", body: `{"off_days":[5]}`, wantStatus: http.StatusOK, wantSaved: "5"},
		{name: "no off day at all", body: `{"off_days":[]}`, wantStatus: http.StatusOK, wantSaved: ""},
		{name: "a repeated day", body: `{"off_days":[5,5,6]}`, wantStatus: http.StatusOK, wantSaved: "5,6"},
		{name: "a day out of range", body: `{"off_days":[7]}`, wantStatus: http.StatusBadRequest},
		{name: "a negative day", body: `{"off_days":[-1]}`, wantStatus: http.StatusBadRequest},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			keys := &fakeSettings{}
			holidays := newHolidaysWith(t, &fakeCalendar{}, &fakeNotifier{}, keys)
			rec := calendarCall(t, holidays.SaveOffDays(),
				http.MethodPut, "/api/admin/calendar/off-days", "", test.body)
			if rec.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d (body = %q)", rec.Code, test.wantStatus, rec.Body.String())
			}
			if test.wantStatus == http.StatusBadRequest {
				if got := decodeBody(t, rec)["error"]; got != MessageOffDays {
					t.Errorf("error = %v, want %q", got, MessageOffDays)
				}
				if len(keys.saved) != 0 {
					t.Errorf("the store received %v, want nothing after a refusal", keys.saved)
				}
				return
			}
			if got := keys.saved[settings.KeyOffDays]; got != test.wantSaved {
				t.Errorf("saved off days = %q, want %q", got, test.wantSaved)
			}
			if keys.by != 9 {
				t.Errorf("the change was saved by %d, want the signed in admin", keys.by)
			}
		})
	}
}

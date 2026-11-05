package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/raHI-79/250-project/backend/internal/auth"
	"github.com/raHI-79/250-project/backend/internal/store"
)

const (
	crThursday = "2026-11-05"
	crFriday   = "2026-11-06"
)

func crDayRequest(t *testing.T, handler *CR, query string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, "/api/cr/upcoming"+query, nil)
	return serve(handler.Upcoming(), withCurrentUser(request, crUserID, auth.RoleCR))
}

// The hub asked for a date and the handler read only "from", so Tomorrow and Today came
// back with the very same list: every item from today to the end of the horizon.
func TestCRUpcomingAnswersTheDayThatWasAskedFor(t *testing.T) {
	silenceLog(t)
	items := newFakeScheduleStore()
	items.byDate = map[string][]store.ScheduleItem{
		crThursday: {itemOn(crThursday, store.ItemAwaiting, 1)},
		crFriday:   {itemOn(crFriday, store.ItemScheduled, 2)},
	}
	cr := newCRHandler(t, items, &fakeNotifier{})
	thursday := crDayRequest(t, cr, "?date="+crThursday)
	if thursday.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body = %q)", thursday.Code, thursday.Body.String())
	}
	body := decodeBody(t, thursday)
	if body["date"] != crThursday {
		t.Errorf("date = %v, want %q", body["date"], crThursday)
	}
	if body["date_label"] != "Thu 5 Nov 2026" {
		t.Errorf("date_label = %v, want \"Thu 5 Nov 2026\"", body["date_label"])
	}
	one := dayItems(t, thursday)
	if len(one) != 1 || one[0]["id"] != float64(1) {
		t.Fatalf("Thursday carries %v, want only the item of that day", one)
	}
	friday := crDayRequest(t, cr, "?date="+crFriday)
	two := dayItems(t, friday)
	if len(two) != 1 || two[0]["id"] != float64(2) {
		t.Fatalf("Friday carries %v, want only the item of that day", two)
	}
	if one[0]["id"] == two[0]["id"] {
		t.Error("the two days came back with the same item, so the toggle changes nothing")
	}
}

func TestCRUpcomingRefusesADateInTheWrongOrder(t *testing.T) {
	silenceLog(t)
	cr := newCRHandler(t, newFakeScheduleStore(), &fakeNotifier{})
	rec := crDayRequest(t, cr, "?date=11/06/2026")
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400 (body = %q)", rec.Code, rec.Body.String())
	}
}

func TestCRUpcomingShowsTheHolidayChipAndKeepsTheItemsBelowIt(t *testing.T) {
	silenceLog(t)
	items := newFakeScheduleStore()
	// Friday is a weekly off day, and the class representative created an exam on it.
	items.byDate = map[string][]store.ScheduleItem{
		crFriday: {itemOn(crFriday, store.ItemScheduled, 7)},
	}
	items.kinds = map[string]store.DayKind{
		crFriday: {Date: crFriday, Holiday: true, Name: "Friday"},
	}
	rec := crDayRequest(t, newCRHandler(t, items, &fakeNotifier{}), "?date="+crFriday)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body = %q)", rec.Code, rec.Body.String())
	}
	body := decodeBody(t, rec)
	kind, ok := body["day_kind"].(map[string]any)
	if !ok {
		t.Fatalf("the day carries no chip: %q", rec.Body.String())
	}
	if kind["holiday"] != true || kind["label"] != "Holiday · Friday" {
		t.Errorf("the chip = %v, want the Friday off day", kind)
	}
	listed := dayItems(t, rec)
	if len(listed) != 1 || listed[0]["id"] != float64(7) {
		t.Fatalf("the holiday hid the items: %v", listed)
	}
	if actions := actionsOf(t, listed[0]); len(actions) == 0 {
		t.Error("the exam on the off day lost its actions")
	}
}

func TestCRUpcomingNamesAWorkingDayAndTheWeekdayItFollows(t *testing.T) {
	silenceLog(t)
	items := newFakeScheduleStore()
	items.byDate = map[string][]store.ScheduleItem{"2026-11-14": {}}
	items.kinds = map[string]store.DayKind{
		"2026-11-14": {Date: "2026-11-14", WorkingDay: true, Name: "Make-up Saturday", Follows: 0},
	}
	rec := crDayRequest(t, newCRHandler(t, items, &fakeNotifier{}), "?date=2026-11-14")
	kind, ok := decodeBody(t, rec)["day_kind"].(map[string]any)
	if !ok {
		t.Fatalf("the day carries no chip: %q", rec.Body.String())
	}
	if kind["working_day"] != true {
		t.Errorf("the chip = %v, want a working day", kind)
	}
	if kind["label"] != "Working day · follows Sunday" {
		t.Errorf("the label = %v, want \"Working day · follows Sunday\"", kind["label"])
	}
}

func TestCRUpcomingStillAnswersARangeForShowMoreDays(t *testing.T) {
	silenceLog(t)
	items := newFakeScheduleStore()
	items.items = []store.ScheduleItem{
		itemOn(crThursday, store.ItemAwaiting, 1),
		itemOn(crFriday, store.ItemScheduled, 2),
	}
	rec := crDayRequest(t, newCRHandler(t, items, &fakeNotifier{}), "?from="+crThursday)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body = %q)", rec.Code, rec.Body.String())
	}
	days := responseDays(t, rec)
	if len(days) != 2 {
		t.Fatalf("days = %v, want one group per day", days)
	}
	if days[0].Date != crThursday || days[1].Date != crFriday {
		t.Errorf("days = %v, want them in order", days)
	}
	if decodeBody(t, rec)["date"] != nil {
		t.Error("the range answer carried a single date")
	}
}

func TestCRUpcomingLeavesAPastDayReadOnly(t *testing.T) {
	silenceLog(t)
	items := newFakeScheduleStore()
	past := "2020-01-02"
	items.byDate = map[string][]store.ScheduleItem{
		past: {itemOn(past, store.ItemConfirmed, 3)},
	}
	rec := crDayRequest(t, newCRHandler(t, items, &fakeNotifier{}), "?date="+past)
	body := decodeBody(t, rec)
	if body["past"] != true || body["read_only"] != true {
		t.Errorf("a past day came back changeable: %q", rec.Body.String())
	}
	listed := dayItems(t, rec)
	if len(listed) != 1 {
		t.Fatalf("items = %v, want the one class that was held", listed)
	}
	if listed[0]["final_status"] != StatusHeld {
		t.Errorf("final_status = %v, want %q", listed[0]["final_status"], StatusHeld)
	}
	if actions := actionsOf(t, listed[0]); len(actions) != 0 {
		t.Errorf("a past day offers %v, want nothing", actions)
	}
}

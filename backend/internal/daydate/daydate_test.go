package daydate

import (
	"testing"
	"time"
)

func dhaka(t *testing.T) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation("Asia/Dhaka")
	if err != nil {
		t.Fatalf("could not load Asia/Dhaka: %v", err)
	}
	return loc
}

func TestTodayAndTomorrowFollowDhakaAcrossMidnight(t *testing.T) {
	loc := dhaka(t)
	cases := []struct {
		name     string
		at       time.Time
		today    string
		tomorrow string
		weekday  string
	}{
		{
			name:     "Thursday 23:57 is still Thursday",
			at:       time.Date(2026, 11, 5, 23, 57, 0, 0, loc),
			today:    "2026-11-05",
			tomorrow: "2026-11-06",
			weekday:  "Thursday",
		},
		{
			name:     "Friday 00:03 has already turned over",
			at:       time.Date(2026, 11, 6, 0, 3, 0, 0, loc),
			today:    "2026-11-06",
			tomorrow: "2026-11-07",
			weekday:  "Friday",
		},
		{
			name:     "Friday 05:59 is the same Friday",
			at:       time.Date(2026, 11, 6, 5, 59, 0, 0, loc),
			today:    "2026-11-06",
			tomorrow: "2026-11-07",
			weekday:  "Friday",
		},
		{
			name:     "Friday 06:01 is the same Friday",
			at:       time.Date(2026, 11, 6, 6, 1, 0, 0, loc),
			today:    "2026-11-06",
			tomorrow: "2026-11-07",
			weekday:  "Friday",
		},
		{
			name:     "Saturday 00:03 is Saturday",
			at:       time.Date(2026, 11, 7, 0, 3, 0, 0, loc),
			today:    "2026-11-07",
			tomorrow: "2026-11-08",
			weekday:  "Saturday",
		},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			if got := Today(test.at, loc); got != test.today {
				t.Errorf("Today() = %q, want %q", got, test.today)
			}
			if got := Tomorrow(test.at, loc); got != test.tomorrow {
				t.Errorf("Tomorrow() = %q, want %q", got, test.tomorrow)
			}
			if got := WeekdayName(test.today); got != test.weekday {
				t.Errorf("WeekdayName(%q) = %q, want %q", test.today, got, test.weekday)
			}
			if got := Today(test.at.UTC(), loc); got != test.today {
				t.Errorf("Today() on a UTC stamp = %q, want %q", got, test.today)
			}
		})
	}
}

func TestOfNeverReadsTheStampZone(t *testing.T) {
	loc := dhaka(t)
	at := time.Date(2026, 11, 5, 18, 30, 0, 0, time.UTC)
	if got := Of(at, loc); got != "2026-11-06" {
		t.Errorf("Of() = %q, want the Dhaka date 2026-11-06", got)
	}
	if got := Of(at, time.UTC); got != "2026-11-05" {
		t.Errorf("Of() in UTC = %q, want 2026-11-05", got)
	}
}

func TestDayBoundariesAreDhakaMidnights(t *testing.T) {
	loc := dhaka(t)
	start, err := Start("2026-11-06", loc)
	if err != nil {
		t.Fatalf("Start() returned an unexpected error: %v", err)
	}
	if got := start.UTC().Format(time.RFC3339); got != "2026-11-05T18:00:00Z" {
		t.Errorf("the day start = %s, want 2026-11-05T18:00:00Z", got)
	}
	end, err := End("2026-11-06", loc)
	if err != nil {
		t.Fatalf("End() returned an unexpected error: %v", err)
	}
	if got := end.UTC().Format(time.RFC3339); got != "2026-11-06T18:00:00Z" {
		t.Errorf("the day end = %s, want 2026-11-06T18:00:00Z", got)
	}
	if end.Sub(start) != 24*time.Hour {
		t.Errorf("the day lasted %s, want 24h", end.Sub(start))
	}
}

func TestStartOfTodayUsesDhakaMidnight(t *testing.T) {
	loc := dhaka(t)
	at := time.Date(2026, 11, 6, 0, 3, 0, 0, loc)
	start := StartOf(at, loc)
	if got := Of(start, loc); got != "2026-11-06" {
		t.Errorf("StartOf() landed on %q, want 2026-11-06", got)
	}
	if start.Hour() != 0 || start.Minute() != 0 {
		t.Errorf("StartOf() = %s, want midnight", start)
	}
}

func TestShiftAndRangeWalkWholeDays(t *testing.T) {
	if got := Shift("2026-11-05", 1); got != "2026-11-06" {
		t.Errorf("Shift(+1) = %q, want 2026-11-06", got)
	}
	if got := Shift("2026-11-06", -1); got != "2026-11-05" {
		t.Errorf("Shift(-1) = %q, want 2026-11-05", got)
	}
	if got := Shift("2026-11-30", 1); got != "2026-12-01" {
		t.Errorf("Shift over a month end = %q, want 2026-12-01", got)
	}
	days := Range("2026-11-05", "2026-11-08")
	want := []string{"2026-11-05", "2026-11-06", "2026-11-07", "2026-11-08"}
	if len(days) != len(want) {
		t.Fatalf("Range() = %v, want %v", days, want)
	}
	for index := range want {
		if days[index] != want[index] {
			t.Errorf("Range()[%d] = %q, want %q", index, days[index], want[index])
		}
	}
	if got := Between("2026-11-05", "2026-11-08"); got != 3 {
		t.Errorf("Between() = %d, want 3", got)
	}
}

func TestWeekdayNumbersMatchTheNames(t *testing.T) {
	cases := map[string]int{
		"2026-11-05": 4,
		"2026-11-06": 5,
		"2026-11-07": 6,
		"2026-11-08": 0,
	}
	for date, want := range cases {
		if got := Weekday(date); got != want {
			t.Errorf("Weekday(%q) = %d, want %d", date, got, want)
		}
		if got := DayName(want); got != WeekdayName(date) {
			t.Errorf("DayName(%d) = %q, want %q", want, got, WeekdayName(date))
		}
	}
	if day, ok := WeekdayNumber("Sunday"); !ok || day != 0 {
		t.Errorf("WeekdayNumber(Sunday) = %d, %v, want 0, true", day, ok)
	}
	if day, ok := WeekdayNumber("sat"); !ok || day != 6 {
		t.Errorf("WeekdayNumber(sat) = %d, %v, want 6, true", day, ok)
	}
	if _, ok := WeekdayNumber("Funday"); ok {
		t.Error("WeekdayNumber accepted a name that is not a weekday")
	}
}

func TestWeekNumberFollowsTheIsoWeek(t *testing.T) {
	year, week := WeekNumber("2026-11-06")
	if year != 2026 || week != 45 {
		t.Errorf("WeekNumber() = %d week %d, want 2026 week 45", year, week)
	}
	thursday, thursdayWeek := WeekNumber("2026-11-05")
	if thursday != year || thursdayWeek != week {
		t.Errorf("Thursday and Friday fell in different weeks: %d/%d and %d/%d",
			thursday, thursdayWeek, year, week)
	}
	_, sunday := WeekNumber("2026-11-08")
	if sunday != week {
		t.Errorf("Sunday 8 Nov is in week %d, want the same week %d as the Friday", sunday, week)
	}
}

func TestReadableWritesTheDayFirst(t *testing.T) {
	if got := Readable("2026-11-06"); got != "Fri 6 Nov 2026" {
		t.Errorf("Readable() = %q, want \"Fri 6 Nov 2026\"", got)
	}
	if got := Readable("2026-01-01"); got != "Thu 1 Jan 2026" {
		t.Errorf("Readable() = %q, want \"Thu 1 Jan 2026\"", got)
	}
	if got := Readable("not a date"); got != "not a date" {
		t.Errorf("Readable() changed a value it could not parse: %q", got)
	}
}

func TestParseRejectsOtherOrders(t *testing.T) {
	loc := dhaka(t)
	if _, err := Parse("06/11/2026", loc); err == nil {
		t.Error("Parse accepted a slash date")
	}
	if _, err := Parse("11/06/2026", loc); err == nil {
		t.Error("Parse accepted a month-first date")
	}
	if _, err := Parse("", loc); err == nil {
		t.Error("Parse accepted an empty date")
	}
	parsed, err := Parse("2026-11-06", loc)
	if err != nil {
		t.Fatalf("Parse() returned an unexpected error: %v", err)
	}
	if parsed.Location() != loc {
		t.Errorf("Parse() used the zone %s, want Asia/Dhaka", parsed.Location())
	}
	if !Valid("2026-11-06") || Valid("2026-13-01") {
		t.Error("Valid() disagreed with Parse()")
	}
}

func TestClockAndMinutes(t *testing.T) {
	loc := dhaka(t)
	at := time.Date(2026, 11, 5, 23, 57, 0, 0, loc)
	if got := Clock(at, loc); got != "23:57" {
		t.Errorf("Clock() = %q, want 23:57", got)
	}
	if got := Clock(at.UTC(), loc); got != "23:57" {
		t.Errorf("Clock() on a UTC stamp = %q, want 23:57", got)
	}
	minutes, ok := Minutes("16:00")
	if !ok || minutes != 960 {
		t.Errorf("Minutes(16:00) = %d, %v, want 960, true", minutes, ok)
	}
	if _, ok := Minutes("nope"); ok {
		t.Error("Minutes accepted a value that is not a clock time")
	}
}

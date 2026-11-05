package config

import (
	"testing"
	"time"
)

func dhakaConfig(t *testing.T) *Config {
	t.Helper()
	loc, err := time.LoadLocation("Asia/Dhaka")
	if err != nil {
		t.Fatalf("could not load Asia/Dhaka: %v", err)
	}
	return &Config{Timezone: "Asia/Dhaka", Location: loc}
}

func TestNowFollowsTheFrozenClock(t *testing.T) {
	cfg := dhakaConfig(t)
	at := time.Date(2026, 11, 5, 23, 57, 0, 0, time.UTC)
	cfg.Freeze(at)
	if got := cfg.Now(); !got.Equal(at) {
		t.Errorf("Now() = %s, want the frozen instant %s", got, at)
	}
	if got := cfg.Now().Location().String(); got != "Asia/Dhaka" {
		t.Errorf("Now() reads in %s, want Asia/Dhaka", got)
	}
}

func TestNowFallsBackToTheRealClock(t *testing.T) {
	cfg := dhakaConfig(t)
	before := time.Now()
	got := cfg.Now()
	if got.Before(before.Add(-time.Minute)) || got.After(time.Now().Add(time.Minute)) {
		t.Errorf("Now() = %s, want a stamp from around now", got)
	}
	if _, offset := got.Zone(); offset != int((6 * time.Hour).Seconds()) {
		t.Errorf("Now() offset = %d seconds, want the Asia/Dhaka offset", offset)
	}
}

func TestTheDateHelpersReadTheDhakaDay(t *testing.T) {
	cfg := dhakaConfig(t)
	cases := []struct {
		name      string
		at        time.Time
		today     string
		tomorrow  string
		yesterday string
	}{
		{
			name:      "Thursday 23:57 Dhaka",
			at:        time.Date(2026, 11, 5, 23, 57, 0, 0, cfg.Location),
			today:     "2026-11-05",
			tomorrow:  "2026-11-06",
			yesterday: "2026-11-04",
		},
		{
			name:      "the same instant as a UTC stamp",
			at:        time.Date(2026, 11, 5, 17, 57, 0, 0, time.UTC),
			today:     "2026-11-05",
			tomorrow:  "2026-11-06",
			yesterday: "2026-11-04",
		},
		{
			name:      "Friday 00:03 Dhaka",
			at:        time.Date(2026, 11, 6, 0, 3, 0, 0, cfg.Location),
			today:     "2026-11-06",
			tomorrow:  "2026-11-07",
			yesterday: "2026-11-05",
		},
		{
			name:      "the same instant as a UTC stamp",
			at:        time.Date(2026, 11, 5, 18, 3, 0, 0, time.UTC),
			today:     "2026-11-06",
			tomorrow:  "2026-11-07",
			yesterday: "2026-11-05",
		},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			cfg.Freeze(test.at)
			if got := cfg.Today(); got != test.today {
				t.Errorf("Today() = %q, want %q", got, test.today)
			}
			if got := cfg.Tomorrow(); got != test.tomorrow {
				t.Errorf("Tomorrow() = %q, want %q", got, test.tomorrow)
			}
			if got := cfg.Yesterday(); got != test.yesterday {
				t.Errorf("Yesterday() = %q, want %q", got, test.yesterday)
			}
			if got := cfg.DateOf(test.at.UTC()); got != test.today {
				t.Errorf("DateOf() on a UTC stamp = %q, want %q", got, test.today)
			}
		})
	}
}

func TestDayBoundariesAreDhakaMidnights(t *testing.T) {
	cfg := dhakaConfig(t)
	start, err := cfg.DayStart("2026-11-06")
	if err != nil {
		t.Fatalf("DayStart() returned an unexpected error: %v", err)
	}
	if got := start.UTC().Format(time.RFC3339); got != "2026-11-05T18:00:00Z" {
		t.Errorf("DayStart() = %s, want 2026-11-05T18:00:00Z", got)
	}
	end, err := cfg.DayEnd("2026-11-06")
	if err != nil {
		t.Fatalf("DayEnd() returned an unexpected error: %v", err)
	}
	if got := end.UTC().Format(time.RFC3339); got != "2026-11-06T18:00:00Z" {
		t.Errorf("DayEnd() = %s, want 2026-11-06T18:00:00Z", got)
	}
	cfg.Freeze(time.Date(2026, 11, 6, 0, 3, 0, 0, cfg.Location))
	if got := cfg.StartOfToday(); !got.Equal(start) {
		t.Errorf("StartOfToday() = %s, want %s", got, start)
	}
	if _, err := cfg.DayStart("06/11/2026"); err == nil {
		t.Error("DayStart() accepted a slash date")
	}
}

func TestShiftAndParseStayOnWholeDays(t *testing.T) {
	cfg := dhakaConfig(t)
	if got := cfg.Shift("2026-11-05", 1); got != "2026-11-06" {
		t.Errorf("Shift(+1) = %q, want 2026-11-06", got)
	}
	if got := cfg.Shift("2026-11-06", -1); got != "2026-11-05" {
		t.Errorf("Shift(-1) = %q, want 2026-11-05", got)
	}
	parsed, err := cfg.ParseDate("2026-11-06")
	if err != nil {
		t.Fatalf("ParseDate() returned an unexpected error: %v", err)
	}
	if got := cfg.DateOf(parsed); got != "2026-11-06" {
		t.Errorf("ParseDate() then DateOf() = %q, want 2026-11-06", got)
	}
	if _, err := cfg.ParseDate("11/06/2026"); err == nil {
		t.Error("ParseDate() accepted a month-first date")
	}
}

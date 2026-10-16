package validate

import (
	"testing"
	"time"
)

func dhaka(t *testing.T) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation("Asia/Dhaka")
	if err != nil {
		t.Fatalf("could not load the timezone: %v", err)
	}
	return loc
}

func TestEventType(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		want      string
		wantError string
	}{
		{name: "class", value: "Class", want: EventClass},
		{name: "exam", value: "EXAM", want: EventExam},
		{name: "extra", value: " extra ", want: EventExtra},
		{name: "empty", value: "", wantError: "Type is required."},
		{name: "lab", value: "lab", wantError: "Type must be one of class, exam, extra."},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := EventType("Type", test.value)
			if test.wantError != "" {
				if err == nil || err.Error() != test.wantError {
					t.Fatalf("error = %v, want %q", err, test.wantError)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != test.want {
				t.Errorf("type = %q, want %q", got, test.want)
			}
		})
	}
}

func TestDate(t *testing.T) {
	loc := dhaka(t)
	tests := []struct {
		name      string
		value     string
		wantError string
	}{
		{name: "valid", value: "2026-10-17"},
		{name: "empty", value: " ", wantError: "date is required."},
		{name: "other order", value: "17-10-2026", wantError: "date must look like 2026-10-17."},
		{name: "with a time", value: "2026-10-17T10:30:00Z", wantError: "date must look like 2026-10-17."},
		{name: "impossible day", value: "2026-02-30", wantError: "date must look like 2026-10-17."},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := Date("date", test.value, loc)
			if test.wantError != "" {
				if err == nil || err.Error() != test.wantError {
					t.Fatalf("error = %v, want %q", err, test.wantError)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if DateString(got) != test.value {
				t.Errorf("date = %q, want %q", DateString(got), test.value)
			}
		})
	}
}

func TestFutureDate(t *testing.T) {
	loc := dhaka(t)
	now := time.Date(2026, 10, 16, 23, 30, 0, 0, loc)
	tests := []struct {
		name      string
		value     string
		wantError string
	}{
		{name: "tomorrow", value: "2026-10-17"},
		{name: "next week", value: "2026-10-23"},
		{name: "today", value: "2026-10-16", wantError: "Date must be tomorrow or later."},
		{name: "yesterday", value: "2026-10-15", wantError: "Date must be tomorrow or later."},
		{name: "last year", value: "2025-10-20", wantError: "Date must be tomorrow or later."},
		{name: "not a date", value: "tomorrow", wantError: "Date must look like 2026-10-17."},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := FutureDate("Date", test.value, now, loc)
			if test.wantError != "" {
				if err == nil || err.Error() != test.wantError {
					t.Fatalf("error = %v, want %q", err, test.wantError)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if DateString(got) != test.value {
				t.Errorf("date = %q, want %q", DateString(got), test.value)
			}
		})
	}
}

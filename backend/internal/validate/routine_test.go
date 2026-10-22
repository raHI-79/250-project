package validate

import "testing"

func TestRoutineSection(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		want      string
		wantError string
	}{
		{name: "single letter", value: "a", want: "A"},
		{name: "all", value: "all", want: "ALL"},
		{name: "already upper", value: "B", want: "B"},
		{name: "empty", value: "  ", wantError: "Section is required."},
		{name: "two letters", value: "AB", wantError: "Section must be A, B or ALL."},
		{name: "number", value: "1", wantError: "Section must be A, B or ALL."},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := RoutineSection("Section", test.value)
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
				t.Errorf("section = %q, want %q", got, test.want)
			}
		})
	}
}

func TestClockTimeAndMinutes(t *testing.T) {
	tests := []struct {
		name        string
		value       string
		want        string
		wantMinutes int
		wantError   string
	}{
		{name: "morning", value: "08:00", want: "08:00", wantMinutes: 480},
		{name: "afternoon", value: "16:45", want: "16:45", wantMinutes: 1005},
		{name: "midnight", value: "00:00", want: "00:00", wantMinutes: 0},
		{name: "missing", value: "", wantError: "Start time is required."},
		{name: "no colon", value: "0800", wantError: "Start time must look like 09:30."},
		{name: "hour out of range", value: "24:00", wantError: "Start time must look like 09:30."},
		{name: "minute out of range", value: "10:60", wantError: "Start time must look like 09:30."},
		{name: "with seconds", value: "10:30:00", wantError: "Start time must look like 09:30."},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := ClockTime("Start time", test.value)
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
				t.Errorf("clock = %q, want %q", got, test.want)
			}
			if minutes := Minutes(got); minutes != test.wantMinutes {
				t.Errorf("minutes = %d, want %d", minutes, test.wantMinutes)
			}
		})
	}
}

func TestDayOfWeekAndName(t *testing.T) {
	tests := []struct {
		name      string
		value     int
		wantName  string
		wantError string
	}{
		{name: "sunday", value: 0, wantName: "Sunday"},
		{name: "monday", value: 1, wantName: "Monday"},
		{name: "saturday", value: 6, wantName: "Saturday"},
		{name: "negative", value: -1, wantError: "Day of week must be a number from 0 (Sunday) to 6 (Saturday)."},
		{name: "seven", value: 7, wantError: "Day of week must be a number from 0 (Sunday) to 6 (Saturday)."},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := DayOfWeek("Day of week", test.value)
			if test.wantError != "" {
				if err == nil || err.Error() != test.wantError {
					t.Fatalf("error = %v, want %q", err, test.wantError)
				}
				if DayName(test.value) != "" {
					t.Errorf("DayName(%d) = %q, want an empty name", test.value, DayName(test.value))
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != test.value {
				t.Errorf("day = %d, want %d", got, test.value)
			}
			if name := DayName(got); name != test.wantName {
				t.Errorf("DayName(%d) = %q, want %q", got, name, test.wantName)
			}
		})
	}
}

func TestRoutineType(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		want      string
		wantError string
	}{
		{name: "class", value: "Class", want: "class"},
		{name: "lab", value: "LAB", want: "lab"},
		{name: "empty", value: "", wantError: "Type is required."},
		{name: "exam", value: "exam", wantError: "Type must be class or lab."},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := RoutineType("Type", test.value)
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

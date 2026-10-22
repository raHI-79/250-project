package validate

import "testing"

func dayPeriods() []Period {
	return []Period{
		{Label: "Period 1", Start: "09:00", End: "09:45"},
		{Label: "Period 2", Start: "09:45", End: "10:30"},
		{Label: "Period 3", Start: "10:30", End: "11:15"},
		{Label: "Period 4", Start: "11:15", End: "12:00"},
		{Label: "Period 5", Start: "12:00", End: "12:45"},
		{Label: "Break", Start: "12:45", End: "13:45", IsBreak: true},
		{Label: "Period 6", Start: "13:45", End: "14:30"},
		{Label: "Period 7", Start: "14:30", End: "15:15"},
		{Label: "Period 8", Start: "15:15", End: "16:00"},
		{Label: "Period 9", Start: "16:00", End: "16:45"},
	}
}

func TestPeriodLabel(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		want      string
		wantError string
	}{
		{name: "plain", value: "Period 1", want: "Period 1"},
		{name: "spaces are trimmed", value: "  Break  ", want: "Break"},
		{name: "empty", value: "   ", wantError: "Label is required."},
		{name: "too long", value: string(make([]rune, MaxPeriodLabel+1)),
			wantError: "Label must be at most 40 characters long."},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := PeriodLabel("Label", test.value)
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
				t.Errorf("label = %q, want %q", got, test.want)
			}
		})
	}
}

func TestPeriodFits(t *testing.T) {
	tests := []struct {
		name      string
		candidate Period
		wantError string
	}{
		{name: "after the last period", candidate: Period{Label: "Period 10", Start: "16:45", End: "17:30"}},
		{name: "before the first period", candidate: Period{Label: "Period 0", Start: "08:00", End: "09:00"}},
		{name: "starts inside another", candidate: Period{Label: "Clash", Start: "09:30", End: "10:00"},
			wantError: "This period overlaps Period 1 (09:00–09:45)."},
		{name: "swallows another", candidate: Period{Label: "Clash", Start: "08:00", End: "10:00"},
			wantError: "This period overlaps Period 1 (09:00–09:45)."},
		{name: "ends inside the break", candidate: Period{Label: "Clash", Start: "12:30", End: "13:00"},
			wantError: "This period overlaps Period 5 (12:00–12:45)."},
		{name: "end before start", candidate: Period{Label: "Backwards", Start: "18:00", End: "17:00"},
			wantError: "Start time must be earlier than end time."},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := PeriodFits(dayPeriods(), test.candidate)
			if test.wantError == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil || err.Error() != test.wantError {
				t.Fatalf("error = %v, want %q", err, test.wantError)
			}
		})
	}
}

func TestDayWindow(t *testing.T) {
	opening, closing := DayWindow(dayPeriods())
	if Clock(opening) != "09:00" || Clock(closing) != "16:45" {
		t.Errorf("window = %s to %s, want 09:00 to 16:45", Clock(opening), Clock(closing))
	}
	if from, to := DayWindow(nil); from != -1 || to != -1 {
		t.Errorf("empty window = %d to %d, want -1 to -1", from, to)
	}
}

func TestClassWindow(t *testing.T) {
	tests := []struct {
		name      string
		periods   []Period
		start     string
		end       string
		wantError string
	}{
		{name: "one period", periods: dayPeriods(), start: "09:00", end: "09:45"},
		{name: "a lab over three periods", periods: dayPeriods(), start: "10:30", end: "12:45"},
		{name: "the whole day", periods: dayPeriods(), start: "09:00", end: "12:45"},
		{name: "after the break", periods: dayPeriods(), start: "13:45", end: "15:15"},
		{name: "before the first period", periods: dayPeriods(), start: "08:00", end: "09:00",
			wantError: "Classes must be between 09:00 and 16:45."},
		{name: "after the last period", periods: dayPeriods(), start: "16:00", end: "17:30",
			wantError: "Classes must be between 09:00 and 16:45."},
		{name: "inside the break", periods: dayPeriods(), start: "13:00", end: "13:30",
			wantError: "Classes cannot run during Break (12:45–13:45)."},
		{name: "across the break", periods: dayPeriods(), start: "12:00", end: "14:30",
			wantError: "Classes cannot run during Break (12:45–13:45)."},
		{name: "end before start", periods: dayPeriods(), start: "11:00", end: "10:00",
			wantError: "Start time must be earlier than end time."},
		{name: "a broken clock", periods: dayPeriods(), start: "half nine", end: "10:00",
			wantError: "Start time and end time must look like 09:30."},
		{name: "no periods at all", periods: nil, start: "09:00", end: "10:00",
			wantError: "The class periods are not set up yet. Ask the department office to add them."},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := ClassWindow(test.periods, test.start, test.end)
			if test.wantError == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil || err.Error() != test.wantError {
				t.Fatalf("error = %v, want %q", err, test.wantError)
			}
		})
	}
}

func TestSortPeriods(t *testing.T) {
	shuffled := []Period{
		{Label: "Late", Start: "13:45", End: "14:30"},
		{Label: "Early", Start: "09:00", End: "09:45"},
		{Label: "Middle", Start: "11:15", End: "12:00"},
	}
	sorted := SortPeriods(shuffled)
	want := []string{"Early", "Middle", "Late"}
	for index, label := range want {
		if sorted[index].Label != label {
			t.Errorf("period %d = %q, want %q", index, sorted[index].Label, label)
		}
	}
	if shuffled[0].Label != "Late" {
		t.Error("SortPeriods changed the list it was given")
	}
}

func TestCredit(t *testing.T) {
	tests := []struct {
		name      string
		value     float64
		want      float64
		wantError string
	}{
		{name: "three", value: 3, want: 3},
		{name: "the smallest", value: 0.5, want: 0.5},
		{name: "the largest", value: 6, want: 6},
		{name: "a quarter step", value: 1.75, want: 1.75},
		{name: "missing", value: 0, wantError: "Credit is required."},
		{name: "off the step", value: 1.3, wantError: "Credit must move in steps of 0.25."},
		{name: "too small", value: 0.25, wantError: "Credit must be between 0.5 and 6.0."},
		{name: "too large", value: 6.25, wantError: "Credit must be between 0.5 and 6.0."},
		{name: "negative", value: -3, wantError: "Credit must be between 0.5 and 6.0."},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := Credit("Credit", test.value)
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
				t.Errorf("credit = %v, want %v", got, test.want)
			}
		})
	}
}

func TestCanStartDirect(t *testing.T) {
	tests := []struct {
		name   string
		viewer string
		target string
		want   bool
	}{
		{name: "a student writes to a teacher", viewer: RoleStudent, target: RoleTeacher, want: true},
		{name: "a student writes to a class representative", viewer: RoleStudent, target: RoleCR, want: true},
		{name: "a student writes to the office", viewer: RoleStudent, target: RoleAdmin, want: true},
		{name: "a student writes to another student", viewer: RoleStudent, target: RoleStudent},
		{name: "a class representative writes to a student", viewer: RoleCR, target: RoleStudent, want: true},
		{name: "a teacher writes to a student", viewer: RoleTeacher, target: RoleStudent, want: true},
		{name: "the office writes to a student", viewer: RoleAdmin, target: RoleStudent, want: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := CanStartDirect(test.viewer, test.target); got != test.want {
				t.Errorf("CanStartDirect(%q, %q) = %v, want %v", test.viewer, test.target, got, test.want)
			}
		})
	}
	if ListsPlainStudents(RoleStudent) {
		t.Error("a plain student should not see plain students in the contacts")
	}
	for _, role := range []string{RoleCR, RoleTeacher, RoleAdmin} {
		if !ListsPlainStudents(role) {
			t.Errorf("%s should see the students in the contacts", role)
		}
	}
}

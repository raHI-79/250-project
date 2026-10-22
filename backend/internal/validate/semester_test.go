package validate

import "testing"

func TestSemester(t *testing.T) {
	const message = "Semester must be one of 1-1, 1-2, 2-1, 2-2, 3-1, 3-2, 4-1, 4-2."
	tests := []struct {
		name      string
		value     string
		want      string
		wantError string
	}{
		{name: "the first one", value: "1-1", want: "1-1"},
		{name: "the last one", value: "4-2", want: "4-2"},
		{name: "spaces are trimmed", value: " 2-1 ", want: "2-1"},
		{name: "empty", value: "  ", wantError: "Semester is required."},
		{name: "a fifth year", value: "5-1", wantError: message},
		{name: "a third half", value: "2-3", wantError: message},
		{name: "a plain number", value: "2", wantError: message},
		{name: "words", value: "second", wantError: message},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := Semester("Semester", test.value)
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
				t.Errorf("semester = %q, want %q", got, test.want)
			}
		})
	}
}

func TestSemesterLabel(t *testing.T) {
	tests := []struct {
		value string
		want  string
	}{
		{value: "1-1", want: "1st year 1st semester"},
		{value: "1-2", want: "1st year 2nd semester"},
		{value: "2-1", want: "2nd year 1st semester"},
		{value: "3-2", want: "3rd year 2nd semester"},
		{value: "4-2", want: "4th year 2nd semester"},
		{value: "5-1", want: "5-1"},
		{value: "", want: ""},
	}
	for _, test := range tests {
		t.Run(test.value, func(t *testing.T) {
			if got := SemesterLabel(test.value); got != test.want {
				t.Errorf("SemesterLabel(%q) = %q, want %q", test.value, got, test.want)
			}
		})
	}
}

func TestSemestersAreTheEightTerms(t *testing.T) {
	list := Semesters()
	if len(list) != 8 {
		t.Fatalf("got %d semesters, want 8", len(list))
	}
	list[0] = "changed"
	if Semesters()[0] != "1-1" {
		t.Error("Semesters() handed out the list it keeps")
	}
}

package handler

import (
	"testing"
	"time"
)

func TestSemesterWeekCountsFromTheRunningSemester(t *testing.T) {
	start := time.Date(2026, time.January, 5, 9, 0, 0, 0, time.UTC)
	tests := []struct {
		name  string
		since time.Time
		now   time.Time
		want  int
	}{
		{name: "the first day", since: start, now: start, want: 1},
		{name: "the sixth day", since: start, now: start.AddDate(0, 0, 6), want: 1},
		{name: "the eighth day", since: start, now: start.AddDate(0, 0, 7), want: 2},
		{name: "the fourth week", since: start, now: start.AddDate(0, 0, 22), want: 4},
		{name: "before the semester", since: start, now: start.AddDate(0, 0, -3), want: 1},
		{name: "without a stored date", now: start, want: 1},
		{name: "far beyond the semester", since: start, now: start.AddDate(1, 0, 0), want: maxSemesterWeeks},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := semesterWeek(test.since, test.now); got != test.want {
				t.Errorf("semesterWeek = %d, want %d", got, test.want)
			}
		})
	}
}

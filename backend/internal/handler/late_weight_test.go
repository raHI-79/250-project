package handler

import "testing"

func TestTheLateWeightIsANumberBetweenZeroAndOne(t *testing.T) {
	tests := []struct {
		raw  string
		want float64
		ok   bool
	}{
		{raw: "0.5", want: 0.5, ok: true},
		{raw: " 0 ", want: 0, ok: true},
		{raw: "1", want: 1, ok: true},
		{raw: "0.25", want: 0.25, ok: true},
		{raw: "1.01"},
		{raw: "-0.1"},
		{raw: "half"},
		{raw: ""},
		{raw: "NaN"},
	}
	for _, tc := range tests {
		got, ok := ParseLateWeight(tc.raw)
		if ok != tc.ok || got != tc.want {
			t.Errorf("ParseLateWeight(%q) = %v, %t, want %v, %t", tc.raw, got, ok, tc.want, tc.ok)
		}
	}
}

func TestClassHoursArePrefilledFromTheRoutineItem(t *testing.T) {
	tests := []struct {
		start, end string
		want       int
	}{
		{start: "09:00", end: "09:50", want: 1},
		{start: "09:00", end: "10:00", want: 1},
		{start: "09:00", end: "10:01", want: 2},
		{start: "09:00:00", end: "11:00:00", want: 2},
		{start: "09:00", end: "11:30", want: 3},
		{start: "09:00", end: "12:30", want: 4},
		{start: "09:00", end: "15:00", want: 4},
		{start: "", end: "", want: 1},
		{start: "10:00", end: "09:00", want: 1},
	}
	for _, tc := range tests {
		if got := classHours(tc.start, tc.end); got != tc.want {
			t.Errorf("classHours(%q, %q) = %d, want %d", tc.start, tc.end, got, tc.want)
		}
	}
}

package attendance

import (
	"math"
	"strings"
	"testing"
	"time"
)

const (
	testKey   = "7a1f0c9d5e3b8a62f40d1c7b9e2a5f3081d6c4b7a9e2f501c3d8b6a4e7f90213"
	otherKey  = "0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20"
	sessionID = int64(42)
)

var at = time.Date(2026, 10, 30, 10, 15, 0, 0, time.UTC)

func TestTokenMatchesTheCurrentAndPreviousSlice(t *testing.T) {
	slice := Slice(at, TokenSliceSeconds)
	tests := []struct {
		name  string
		token string
		want  bool
	}{
		{name: "current slice", token: Token(testKey, sessionID, slice), want: true},
		{name: "previous slice", token: Token(testKey, sessionID, slice-1), want: true},
		{name: "next slice", token: Token(testKey, sessionID, slice+1)},
		{name: "two slices back", token: Token(testKey, sessionID, slice-2)},
		{name: "wrong key", token: Token(otherKey, sessionID, slice)},
		{name: "another session", token: Token(testKey, sessionID+1, slice)},
		{name: "upper case current slice",
			token: strings.ToUpper(Token(testKey, sessionID, slice)), want: true},
		{name: "empty token"},
		{name: "short token", token: "abcd"},
		{name: "not hex", token: "zzzzzzzzzzzzzzzz"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := TokenMatches(testKey, sessionID, at, tc.token); got != tc.want {
				t.Errorf("TokenMatches(%q) = %t, want %t", tc.token, got, tc.want)
			}
		})
	}
}

func TestTokenIsEightBytes(t *testing.T) {
	token := Token(testKey, sessionID, Slice(at, TokenSliceSeconds))
	if len(token) != TokenBytes*2 {
		t.Errorf("Token() returned %d characters, want %d", len(token), TokenBytes*2)
	}
	if token == Token(testKey, sessionID, Slice(at, TokenSliceSeconds)+1) {
		t.Error("two slices of the same session produced the same token")
	}
}

func TestCodeMatchesTheCurrentAndPreviousSlice(t *testing.T) {
	slice := Slice(at, CodeSliceSeconds)
	tests := []struct {
		name string
		code string
		want bool
	}{
		{name: "current slice", code: Code(testKey, sessionID, slice), want: true},
		{name: "previous slice", code: Code(testKey, sessionID, slice-1), want: true},
		{name: "next slice", code: Code(testKey, sessionID, slice+1)},
		{name: "expired slice", code: Code(testKey, sessionID, slice-3)},
		{name: "wrong key", code: Code(otherKey, sessionID, slice)},
		{name: "another session", code: Code(testKey, sessionID+7, slice)},
		{name: "spaces around the code",
			code: " " + Code(testKey, sessionID, slice) + " ", want: true},
		{name: "empty code"},
		{name: "three digits", code: "123"},
		{name: "five digits", code: "12345"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := CodeMatches(testKey, sessionID, at, tc.code); got != tc.want {
				t.Errorf("CodeMatches(%q) = %t, want %t", tc.code, got, tc.want)
			}
		})
	}
}

func TestCodeIsFourDigits(t *testing.T) {
	for offset := int64(0); offset < 200; offset++ {
		code := Code(testKey, sessionID, Slice(at, CodeSliceSeconds)+offset)
		if len(code) != CodeDigits {
			t.Fatalf("Code() returned %q, want %d digits", code, CodeDigits)
		}
		for _, digit := range code {
			if digit < '0' || digit > '9' {
				t.Fatalf("Code() returned %q, want only digits", code)
			}
		}
	}
}

func TestSliceWalksForwardWithTime(t *testing.T) {
	tests := []struct {
		name    string
		seconds int
		shift   time.Duration
		want    int64
	}{
		{name: "same ten second slice", seconds: TokenSliceSeconds, shift: 9 * time.Second},
		{name: "next ten second slice", seconds: TokenSliceSeconds, shift: 10 * time.Second, want: 1},
		{name: "same twenty second slice", seconds: CodeSliceSeconds, shift: 19 * time.Second},
		{name: "next twenty second slice", seconds: CodeSliceSeconds, shift: 20 * time.Second, want: 1},
		{name: "no slice length", seconds: 0, shift: time.Hour},
	}
	base := time.Unix(1762000000, 0).UTC()
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := Slice(base.Add(tc.shift), tc.seconds) - Slice(base, tc.seconds)
			if got != tc.want {
				t.Errorf("slice moved by %d, want %d", got, tc.want)
			}
		})
	}
}

func TestDistanceMatchesKnownPairs(t *testing.T) {
	tests := []struct {
		name                           string
		fromLat, fromLng, toLat, toLng float64
		want                           float64
		tolerance                      float64
	}{
		{name: "same point", fromLat: 24.9169, fromLng: 91.8316,
			toLat: 24.9169, toLng: 91.8316, want: 0, tolerance: 0.5},
		{name: "ten metres north", fromLat: 24.9169, fromLng: 91.8316,
			toLat: 24.916990, toLng: 91.8316, want: 10, tolerance: 1.5},
		{name: "one hundred metres east", fromLat: 24.9169, fromLng: 91.8316,
			toLat: 24.9169, toLng: 91.832591, want: 100, tolerance: 3},
		{name: "one kilometre south", fromLat: 24.9169, fromLng: 91.8316,
			toLat: 24.907905, toLng: 91.8316, want: 1000, tolerance: 10},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := DistanceM(tc.fromLat, tc.fromLng, tc.toLat, tc.toLng)
			if math.Abs(got-tc.want) > tc.tolerance {
				t.Errorf("DistanceM() = %.2f m, want %.2f m within %.2f", got, tc.want, tc.tolerance)
			}
		})
	}
}

func TestCheckPositionAppliesEveryRule(t *testing.T) {
	teacherLat, teacherLng := 24.9169, 91.8316
	fresh := Anchor{Lat: teacherLat, Lng: teacherLng, UpdatedAt: at.Add(-5 * time.Second), Known: true}
	tests := []struct {
		name    string
		student Position
		teacher Anchor
		want    string
	}{
		{
			name:    "inside the room",
			student: Position{Lat: 24.916990, Lng: 91.8316, AccuracyM: 12},
			teacher: fresh,
		},
		{
			name:    "right at thirty metres",
			student: Position{Lat: 24.917165, Lng: 91.8316, AccuracyM: 20},
			teacher: fresh,
		},
		{
			name:    "too far away",
			student: Position{Lat: 24.918000, Lng: 91.8316, AccuracyM: 10},
			teacher: fresh,
			want:    MessageTooFar,
		},
		{
			name:    "weak signal",
			student: Position{Lat: 24.916990, Lng: 91.8316, AccuracyM: 51},
			teacher: fresh,
			want:    MessageWeakSignal,
		},
		{
			name:    "accuracy right at fifty metres",
			student: Position{Lat: 24.916990, Lng: 91.8316, AccuracyM: 50},
			teacher: fresh,
		},
		{
			name:    "no accuracy reported",
			student: Position{Lat: 24.916990, Lng: 91.8316},
			teacher: fresh,
			want:    MessageWeakSignal,
		},
		{
			name:    "mock provider",
			student: Position{Lat: 24.916990, Lng: 91.8316, AccuracyM: 8, IsMock: true},
			teacher: fresh,
			want:    MessageMockLocation,
		},
		{
			name:    "no position at all",
			student: Position{AccuracyM: 8},
			teacher: fresh,
			want:    MessageNoPosition,
		},
		{
			name:    "the teacher never sent a position",
			student: Position{Lat: 24.916990, Lng: 91.8316, AccuracyM: 8},
			teacher: Anchor{},
			want:    MessageTeacherStale,
		},
		{
			name:    "the teacher position is a minute old",
			student: Position{Lat: 24.916990, Lng: 91.8316, AccuracyM: 8},
			teacher: Anchor{Lat: teacherLat, Lng: teacherLng, UpdatedAt: at.Add(-61 * time.Second), Known: true},
			want:    MessageTeacherStale,
		},
		{
			name:    "the teacher position is just fresh enough",
			student: Position{Lat: 24.916990, Lng: 91.8316, AccuracyM: 8},
			teacher: Anchor{Lat: teacherLat, Lng: teacherLng, UpdatedAt: at.Add(-59 * time.Second), Known: true},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			distance, refusal := CheckPosition(tc.student, tc.teacher, at)
			if refusal != tc.want {
				t.Fatalf("CheckPosition() refusal = %q, want %q", refusal, tc.want)
			}
			if refusal == "" && distance > MaxDistanceM {
				t.Errorf("an accepted mark reported %.2f m, want at most %.2f m", distance, MaxDistanceM)
			}
		})
	}
}

func TestValidDuration(t *testing.T) {
	tests := []struct {
		minutes int
		want    bool
	}{
		{minutes: 0},
		{minutes: 4},
		{minutes: 5, want: true},
		{minutes: 15, want: true},
		{minutes: 30, want: true},
		{minutes: 31},
		{minutes: 120},
		{minutes: -10},
	}
	for _, tc := range tests {
		if got := ValidDuration(tc.minutes); got != tc.want {
			t.Errorf("ValidDuration(%d) = %t, want %t", tc.minutes, got, tc.want)
		}
	}
}

func TestPercentageAndThreshold(t *testing.T) {
	tests := []struct {
		name    string
		present int
		held    int
		percent float64
		below   bool
	}{
		{name: "nothing held"},
		{name: "one of two is too few sessions", present: 1, held: 2, percent: 50},
		{name: "one of three", present: 1, held: 3, percent: 100.0 / 3, below: true},
		{name: "two of four", present: 2, held: 4, percent: 50},
		{name: "three of seven", present: 3, held: 7, percent: 300.0 / 7, below: true},
		{name: "all present", present: 8, held: 8, percent: 100},
		{name: "none present", present: 0, held: 5, below: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := Percentage(tc.present, tc.held); math.Abs(got-tc.percent) > 0.01 {
				t.Errorf("Percentage(%d, %d) = %.2f, want %.2f", tc.present, tc.held, got, tc.percent)
			}
			if got := BelowThreshold(tc.present, tc.held); got != tc.below {
				t.Errorf("BelowThreshold(%d, %d) = %t, want %t", tc.present, tc.held, got, tc.below)
			}
		})
	}
}

func TestAlertDueOnceThenWeekly(t *testing.T) {
	tests := []struct {
		name     string
		present  int
		held     int
		lastSent time.Time
		want     bool
	}{
		{name: "first time below fifty", present: 1, held: 4, want: true},
		{name: "already told today", present: 1, held: 4, lastSent: at.Add(-2 * time.Hour)},
		{name: "told six days ago", present: 1, held: 4, lastSent: at.Add(-6 * 24 * time.Hour)},
		{name: "told a week ago", present: 1, held: 4, lastSent: at.Add(-7 * 24 * time.Hour), want: true},
		{name: "told nine days ago", present: 1, held: 4, lastSent: at.Add(-9 * 24 * time.Hour), want: true},
		{name: "not below fifty", present: 3, held: 4, lastSent: at.Add(-30 * 24 * time.Hour)},
		{name: "too few sessions", present: 0, held: 2},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := AlertDue(tc.present, tc.held, tc.lastSent, at); got != tc.want {
				t.Errorf("AlertDue() = %t, want %t", got, tc.want)
			}
		})
	}
}

func TestNoteRequiredBeyondTheSemesterLimit(t *testing.T) {
	tests := []struct {
		changes int
		want    bool
	}{
		{changes: 0},
		{changes: 1},
		{changes: 2, want: true},
		{changes: 5, want: true},
	}
	for _, tc := range tests {
		if got := NoteRequired(tc.changes); got != tc.want {
			t.Errorf("NoteRequired(%d) = %t, want %t", tc.changes, got, tc.want)
		}
	}
}

func TestCoversSection(t *testing.T) {
	tests := []struct {
		session string
		student string
		want    bool
	}{
		{session: "A", student: "A", want: true},
		{session: "A", student: "B"},
		{session: "B", student: "B", want: true},
		{session: "ALL", student: "A", want: true},
		{session: "ALL", student: "B", want: true},
		{session: "all", student: "b", want: true},
		{session: " A ", student: "a", want: true},
		{session: "", student: "A"},
	}
	for _, tc := range tests {
		if got := CoversSection(tc.session, tc.student); got != tc.want {
			t.Errorf("CoversSection(%q, %q) = %t, want %t", tc.session, tc.student, got, tc.want)
		}
	}
}

func TestNewKeyIsFreshEveryTime(t *testing.T) {
	seen := make(map[string]bool)
	for attempt := 0; attempt < 32; attempt++ {
		key, err := NewKey()
		if err != nil {
			t.Fatalf("NewKey() returned an unexpected error: %v", err)
		}
		if len(key) != keyBytes*2 {
			t.Fatalf("NewKey() returned %d characters, want %d", len(key), keyBytes*2)
		}
		if seen[key] {
			t.Fatalf("NewKey() returned the same key twice: %s", key)
		}
		seen[key] = true
	}
}

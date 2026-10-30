package jobs

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/raHI-79/250-project/backend/internal/notify"
	"github.com/raHI-79/250-project/backend/internal/store"
)

func lowRow(studentID int64, present, held int, lastSent time.Time) store.AttendanceAlert {
	return store.AttendanceAlert{
		StudentID:   studentID,
		UserID:      studentID + 1000,
		CourseID:    5,
		CourseTitle: "Databases",
		Held:        held,
		Present:     present,
		LastSentAt:  lastSent,
	}
}

func TestWarnLowAttendanceTellsEachStudentOnce(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name  string
		rows  []store.AttendanceAlert
		want  int
		title string
	}{
		{
			name: "nothing below fifty",
			rows: []store.AttendanceAlert{lowRow(1, 3, 4, time.Time{})},
		},
		{
			name:  "one of four classes",
			rows:  []store.AttendanceAlert{lowRow(1, 1, 4, time.Time{})},
			want:  1,
			title: "Your attendance in Databases is 25 % — below 50 %",
		},
		{
			name: "too few sessions to judge",
			rows: []store.AttendanceAlert{lowRow(1, 0, 2, time.Time{})},
		},
		{
			name: "already told two days ago",
			rows: []store.AttendanceAlert{lowRow(1, 1, 4, now.Add(-2*24*time.Hour))},
		},
		{
			name:  "told eight days ago",
			rows:  []store.AttendanceAlert{lowRow(1, 1, 4, now.Add(-8*24*time.Hour))},
			want:  1,
			title: "Your attendance in Databases is 25 % — below 50 %",
		},
		{
			name: "two students, one of them fine",
			rows: []store.AttendanceAlert{
				lowRow(1, 1, 4, time.Time{}),
				lowRow(2, 4, 4, time.Time{}),
				lowRow(3, 2, 7, time.Time{}),
			},
			want:  2,
			title: "Your attendance in Databases is 25 % — below 50 %",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			items := &fakeItems{low: tc.rows}
			alerts := &fakeAlerts{}
			scheduler := NewScheduler(testConfig(t), items, alerts)
			sent, err := scheduler.WarnLowAttendance(context.Background())
			if err != nil {
				t.Fatalf("WarnLowAttendance() returned an unexpected error: %v", err)
			}
			if sent != tc.want {
				t.Fatalf("WarnLowAttendance() = %d, want %d", sent, tc.want)
			}
			if len(alerts.sent) != tc.want {
				t.Fatalf("the students heard %d times, want %d", len(alerts.sent), tc.want)
			}
			if tc.want == 0 {
				return
			}
			first := alerts.sent[0]
			if first.title != tc.title {
				t.Errorf("title = %q, want %q", first.title, tc.title)
			}
			if first.kind != notify.TypeAttend {
				t.Errorf("kind = %q, want %q", first.kind, notify.TypeAttend)
			}
			if first.link.Kind != notify.LinkAttendMine || first.link.Ref != 5 {
				t.Errorf("link = %+v, want the attendance of course 5", first.link)
			}
		})
	}
}

func TestWarnLowAttendanceClaimsBeforeItSends(t *testing.T) {
	items := &fakeItems{low: []store.AttendanceAlert{lowRow(1, 1, 4, time.Time{})}}
	alerts := &fakeAlerts{}
	scheduler := NewScheduler(testConfig(t), items, alerts)
	if _, err := scheduler.WarnLowAttendance(context.Background()); err != nil {
		t.Fatalf("WarnLowAttendance() returned an unexpected error: %v", err)
	}
	if len(items.warned) != 1 {
		t.Fatalf("the job claimed %d alerts, want 1", len(items.warned))
	}
	again, err := scheduler.WarnLowAttendance(context.Background())
	if err != nil {
		t.Fatalf("WarnLowAttendance() returned an unexpected error on the second run: %v", err)
	}
	if again != 0 {
		t.Errorf("the second run sent %d alerts, want none", again)
	}
	if len(alerts.sent) != 1 {
		t.Errorf("the student heard %d times, want once", len(alerts.sent))
	}
}

func TestLowAttendanceTitleRoundsThePercentage(t *testing.T) {
	tests := []struct {
		name    string
		course  string
		percent float64
		want    string
	}{
		{name: "a third", course: "Databases", percent: 100.0 / 3,
			want: "Your attendance in Databases is 33 % — below 50 %"},
		{name: "rounds up", course: "Algorithms", percent: 42.6,
			want: "Your attendance in Algorithms is 43 % — below 50 %"},
		{name: "nothing attended", course: "Compilers", percent: 0,
			want: "Your attendance in Compilers is 0 % — below 50 %"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := LowAttendanceTitle(tc.course, tc.percent)
			if got != tc.want {
				t.Errorf("LowAttendanceTitle() = %q, want %q", got, tc.want)
			}
			if !strings.Contains(got, tc.course) {
				t.Errorf("LowAttendanceTitle() = %q, want it to name the course", got)
			}
		})
	}
}

package jobs

import (
	"context"
	"fmt"
	"github.com/raHI-79/250-project/backend/internal/attendance"
	"github.com/raHI-79/250-project/backend/internal/notify"
	"log"
	"math"
)

const (
	lowAttendanceTitle = "Your attendance in %s is %d %% — below 50 %%"
	lowAttendanceBody  = "Open your attendance to see the classes you missed."
)

func LowAttendanceTitle(courseTitle string, percentage float64) string {
	return fmt.Sprintf(lowAttendanceTitle, courseTitle, int(math.Round(percentage)))
}

func (s *Scheduler) WarnLowAttendance(ctx context.Context) (int, error) {
	rows, err := s.items.ListLowAttendance(ctx, attendance.AlertMinSessions, attendance.AlertPercentage)
	if err != nil {
		return 0, err
	}
	now := s.cfg.Now()
	before := now.Add(-attendance.AlertCooldown)
	sent := 0
	for _, row := range rows {
		if !attendance.AlertDue(row.Present, row.Held, row.LastSentAt, now) {
			continue
		}
		fresh, err := s.items.ClaimAttendanceAlert(ctx, row.StudentID, row.CourseID, before)
		if err != nil {
			log.Printf("jobs: could not claim a low attendance alert: %v", err)
			continue
		}
		if !fresh {
			continue
		}
		title := LowAttendanceTitle(row.CourseTitle, attendance.Percentage(row.Present, row.Held))
		err = s.alerts.NotifyUser(ctx, row.UserID, notify.TypeAttend, title, lowAttendanceBody,
			notify.Ref(notify.LinkAttendMine, row.CourseID))
		if err != nil {
			log.Printf("jobs: could not warn the student %d about low attendance: %v", row.UserID, err)
			continue
		}
		sent++
	}
	return sent, nil
}

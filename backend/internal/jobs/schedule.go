package jobs

import (
	"context"
	"fmt"
	"github.com/raHI-79/250-project/backend/internal/config"
	"github.com/raHI-79/250-project/backend/internal/notify"
	"github.com/raHI-79/250-project/backend/internal/store"
	"log"
	"time"
)

const (
	Interval           = 5 * time.Minute
	TitleNoResponse    = "No answer from the teacher"
	TitleExamInFinal   = "Exam added to the final routine"
	dateLayout         = "2006-01-02"
	noResponseSentence = "%s on %s at %s got no answer by the deadline. The room stays booked, so decide for your batch."
	examFinalSentence  = "%s on %s is in the final routine. Set the time and the room, then publish."
)

type Store interface {
	GenerateFixedItems(ctx context.Context, from, to string) (int, error)
	MarkNoResponse(ctx context.Context, date string) ([]store.ScheduleItem, error)
	PromoteExamsToFinal(ctx context.Context, date string) ([]store.ScheduleItem, error)
	ListActiveCRs(ctx context.Context, batch, section string) ([]int64, error)
	ListUndecidedTeachers(ctx context.Context, date string) ([]store.UndecidedTeacher, error)
	ListUndecidedTeachersFrom(ctx context.Context, date, after string) ([]store.UndecidedTeacher, error)
	ClaimReminder(ctx context.Context, userID int64, slot, date string) (bool, error)
}

type Notifier interface {
	NotifyUser(ctx context.Context, userID int64, kind, title, body string, link store.NotificationLink) error
}

type Scheduler struct {
	cfg    *config.Config
	items  Store
	alerts Notifier
}

func NewScheduler(cfg *config.Config, items Store, alerts Notifier) *Scheduler {
	return &Scheduler{cfg: cfg, items: items, alerts: alerts}
}

func (s *Scheduler) Run(ctx context.Context, interval time.Duration) {
	s.runOnce(ctx)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.runOnce(ctx)
		}
	}
}

func (s *Scheduler) runOnce(ctx context.Context) {
	now := s.cfg.Now()
	if created, err := s.Generate(ctx, now); err != nil {
		log.Printf("jobs: could not generate the schedule items: %v", err)
	} else if created > 0 {
		log.Printf("jobs: generated %d schedule items", created)
	}
	if closed, err := s.CloseDecisions(ctx, now); err != nil {
		log.Printf("jobs: could not close the decision window: %v", err)
	} else if closed > 0 {
		log.Printf("jobs: %d items got no answer from their teacher", closed)
	}
	if promoted, err := s.PromoteExams(ctx, now); err != nil {
		log.Printf("jobs: could not move the exams into the final routine: %v", err)
	} else if promoted > 0 {
		log.Printf("jobs: moved %d exams into the final routine", promoted)
	}
	if reminded, err := s.RemindTeachers(ctx, now); err != nil {
		log.Printf("jobs: could not remind the teachers: %v", err)
	} else if reminded > 0 {
		log.Printf("jobs: reminded %d teachers about undecided classes", reminded)
	}
}

func (s *Scheduler) Generate(ctx context.Context, now time.Time) (int, error) {
	from := now.Format(dateLayout)
	to := now.AddDate(0, 0, s.cfg.DaysAhead).Format(dateLayout)
	return s.items.GenerateFixedItems(ctx, from, to)
}

func (s *Scheduler) CloseDecisions(ctx context.Context, now time.Time) (int, error) {
	if !s.cfg.DecisionClosed(now) {
		return 0, nil
	}
	tomorrow := now.AddDate(0, 0, 1).Format(dateLayout)
	items, err := s.items.MarkNoResponse(ctx, tomorrow)
	if err != nil {
		return 0, err
	}
	for _, item := range items {
		body := fmt.Sprintf(noResponseSentence, item.CourseCode, item.ItemDate, item.StartTime)
		s.tellCRs(ctx, item, notify.TypeRoutine, TitleNoResponse, body)
	}
	return len(items), nil
}

func (s *Scheduler) PromoteExams(ctx context.Context, now time.Time) (int, error) {
	tomorrow := now.AddDate(0, 0, 1).Format(dateLayout)
	items, err := s.items.PromoteExamsToFinal(ctx, tomorrow)
	if err != nil {
		return 0, err
	}
	for _, item := range items {
		body := fmt.Sprintf(examFinalSentence, item.CourseCode, item.ItemDate)
		s.tellCRs(ctx, item, notify.TypeExam, TitleExamInFinal, body)
	}
	return len(items), nil
}

func (s *Scheduler) tellCRs(ctx context.Context, item store.ScheduleItem, kind, title, body string) {
	crs, err := s.items.ListActiveCRs(ctx, item.Batch, item.Section)
	if err != nil {
		log.Printf("jobs: could not list the class representatives of %s: %v", item.Batch, err)
		return
	}
	link := notify.Ref(notify.LinkCRUpcoming, item.ID)
	if kind == notify.TypeExam {
		link = notify.Page(notify.LinkCRFinal)
	}
	for _, userID := range crs {
		if err := s.alerts.NotifyUser(ctx, userID, kind, title, body, link); err != nil {
			log.Printf("jobs: could not notify the class representative %d: %v", userID, err)
		}
	}
}

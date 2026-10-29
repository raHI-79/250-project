package jobs

import (
	"context"
	"fmt"
	"github.com/raHI-79/250-project/backend/internal/notify"
	"github.com/raHI-79/250-project/backend/internal/store"
	"log"
	"strings"
	"time"
)

const (
	SlotMorning       = "morning"
	TitleDecideNeeded = "Tomorrow's classes need your decision"
	TitleStillWaiting = "Reminder: tomorrow's classes are undecided"
	TitleTodayPending = "Today's classes still need your decision"
	clockLayout       = "15:04"
	decideSentence    = "Tomorrow's classes need your decision · %d %s"
	remindSentence    = "Reminder: %d %s tomorrow are still undecided"
	todaySentence     = "%d %s today have not started and still need your decision"
	slotWindow        = 2 * Interval
)

func classWord(count int) string {
	if count == 1 {
		return "class"
	}
	return "classes"
}

func slotKey(prefix, clock string) string {
	return prefix + ":" + clock
}

func dueSlot(now time.Time, clock string) bool {
	parsed, err := time.Parse(clockLayout, strings.TrimSpace(clock))
	if err != nil {
		return false
	}
	due := time.Date(now.Year(), now.Month(), now.Day(), parsed.Hour(), parsed.Minute(), 0, 0,
		now.Location())
	if now.Before(due) {
		return false
	}
	return now.Sub(due) < slotWindow
}

func (s *Scheduler) RemindTeachers(ctx context.Context, now time.Time) (int, error) {
	sent := 0
	today := now.Format(dateLayout)
	tomorrow := now.AddDate(0, 0, 1).Format(dateLayout)
	for index, clock := range s.cfg.ReminderTimes {
		if !dueSlot(now, clock) {
			continue
		}
		teachers, err := s.items.ListUndecidedTeachers(ctx, tomorrow)
		if err != nil {
			return sent, err
		}
		title, sentence := TitleDecideNeeded, decideSentence
		if index > 0 {
			title, sentence = TitleStillWaiting, remindSentence
		}
		sent += s.remind(ctx, teachers, slotKey("evening", clock), today, title, sentence)
	}
	if dueSlot(now, s.cfg.MorningReminder) {
		teachers, err := s.items.ListUndecidedTeachersFrom(ctx, today, now.Format(clockLayout))
		if err != nil {
			return sent, err
		}
		sent += s.remind(ctx, teachers, slotKey(SlotMorning, s.cfg.MorningReminder), today,
			TitleTodayPending, todaySentence)
	}
	return sent, nil
}

func (s *Scheduler) remind(ctx context.Context, teachers []store.UndecidedTeacher,
	slot, date, title, sentence string) int {
	sent := 0
	for _, teacher := range teachers {
		fresh, err := s.items.ClaimReminder(ctx, teacher.UserID, slot, date)
		if err != nil {
			log.Printf("jobs: could not claim a teacher reminder: %v", err)
			continue
		}
		if !fresh {
			continue
		}
		body := fmt.Sprintf(sentence, teacher.Count, classWord(teacher.Count))
		err = s.alerts.NotifyUser(ctx, teacher.UserID, notify.TypeRoutine, title, body,
			notify.Page(notify.LinkTeacherNextDay))
		if err != nil {
			log.Printf("jobs: could not remind the teacher %d: %v", teacher.UserID, err)
			continue
		}
		sent++
	}
	return sent
}

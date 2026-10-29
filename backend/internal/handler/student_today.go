package handler

import (
	"github.com/raHI-79/250-project/backend/internal/httpx"
	"github.com/raHI-79/250-project/backend/internal/middleware"
	"github.com/raHI-79/250-project/backend/internal/store"
	"github.com/raHI-79/250-project/backend/internal/validate"
	"log"
	"net/http"
	"time"
)

const (
	maxSemesterWeeks = 26
	nextExamCount    = 3
)

type nextExam struct {
	Date   string `json:"date"`
	Course string `json:"course"`
}

type todayResponse struct {
	Date           string       `json:"date"`
	Semester       string       `json:"semester"`
	SemesterLabel  string       `json:"semester_label"`
	Week           int          `json:"week"`
	ClassesToday   int          `json:"classes_today"`
	UpcomingExams  int          `json:"upcoming_exams"`
	NextExam       *nextExam    `json:"next_exam"`
	NextExams      []examItem   `json:"next_exams"`
	UnreadMessages int          `json:"unread_messages"`
	OpenPolls      int          `json:"open_polls"`
	LatestNotices  []noticeItem `json:"latest_notices"`
}

func (s *Student) Today() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profile, ok := s.profile(w, r)
		if !ok {
			return
		}
		now := s.cfg.Now()
		day := validate.DateString(now)
		groups, ok := s.dayGroups(w, r, profile, day)
		if !ok {
			return
		}
		codes, ok := s.dropCourseCodes(w, r, profile.StudentID)
		if !ok {
			return
		}
		exams, err := s.students.ListStudentExams(r.Context(), profile.Batch, profile.Section, now, codes)
		if err != nil {
			log.Printf("handler: could not count the exams of a student: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		polls, err := s.students.ListStudentPolls(r.Context(), profile.UserID, profile.Batch, profile.Section)
		if err != nil {
			log.Printf("handler: could not count the polls of a student: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		notices, err := s.students.ListStudentNotices(r.Context(), profile.Batch, profile.Section, store.NoticeAll, "")
		if err != nil {
			log.Printf("handler: could not list the latest notices of a student: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		unread, ok := s.unreadMessages(w, r, profile.UserID)
		if !ok {
			return
		}
		semester, since, err := s.students.RunningSemesterSince(r.Context(), profile.Batch)
		if err != nil {
			log.Printf("handler: could not read the running semester of a student: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, todayResponse{
			Date:           day,
			Semester:       semester,
			SemesterLabel:  validate.SemesterLabel(semester),
			Week:           semesterWeek(since, now),
			ClassesToday:   scheduledCount(groups),
			UpcomingExams:  len(exams),
			NextExam:       firstExam(exams),
			NextExams:      nextExams(exams),
			UnreadMessages: unread,
			OpenPolls:      openPollCount(polls),
			LatestNotices:  latestNotices(notices),
		})
	}
}

func (s *Student) unreadMessages(w http.ResponseWriter, r *http.Request, userID int64) (int, bool) {
	viewer, err := s.students.GetMessageViewer(r.Context(), userID)
	if err != nil {
		log.Printf("handler: could not load the message viewer of a student: %v", err)
		httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
		return 0, false
	}
	count, err := s.students.CountUnreadMessages(r.Context(), viewer)
	if err != nil {
		log.Printf("handler: could not count the unread messages of a student: %v", err)
		httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
		return 0, false
	}
	return count, true
}

func scheduledCount(groups []planGroup) int {
	count := 0
	for _, group := range groups {
		for _, item := range group.Items {
			if item.Status != StatusCancelled {
				count++
			}
		}
	}
	return count
}

func semesterWeek(since, now time.Time) int {
	if since.IsZero() || now.Before(since) {
		return 1
	}
	days := int(now.Sub(since).Hours() / 24)
	week := days/7 + 1
	if week > maxSemesterWeeks {
		return maxSemesterWeeks
	}
	return week
}

func nextExams(exams []store.StudentExam) []examItem {
	items := make([]examItem, 0, nextExamCount)
	for _, exam := range exams {
		if len(items) == nextExamCount {
			break
		}
		items = append(items, newExamItem(exam))
	}
	return items
}

func firstExam(exams []store.StudentExam) *nextExam {
	if len(exams) == 0 {
		return nil
	}
	return &nextExam{Date: exams[0].ItemDate, Course: exams[0].CourseCode}
}

func openPollCount(polls []store.PollRow) int {
	count := 0
	for _, poll := range polls {
		if poll.Status == store.PollOpen {
			count++
		}
	}
	return count
}

func latestNotices(notices []store.NoticeRow) []noticeItem {
	items := make([]noticeItem, 0, latestNoticeCount)
	for _, notice := range notices {
		if len(items) == latestNoticeCount {
			break
		}
		items = append(items, newNoticeItem(notice))
	}
	return items
}

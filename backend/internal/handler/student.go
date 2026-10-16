package handler

import (
	"context"
	"errors"
	"github.com/raHI-79/250-project/backend/internal/config"
	"github.com/raHI-79/250-project/backend/internal/httpx"
	"github.com/raHI-79/250-project/backend/internal/middleware"
	"github.com/raHI-79/250-project/backend/internal/store"
	"github.com/raHI-79/250-project/backend/internal/validate"
	"log"
	"net/http"
	"sort"
	"strings"
	"time"
)

const (
	MessageNotAStudent = "No student profile found for this account."
	StatusScheduled    = "scheduled"
	StatusCancelled    = "cancelled"
	dateLabelLayout    = "02.01.06"
)

type StudentStore interface {
	GetStudentProfile(ctx context.Context, userID int64) (store.StudentProfile, error)
	ListDayPlan(ctx context.Context, batch, date string) ([]store.PlanItem, error)
	ListDropCourseSchedule(ctx context.Context, courseCode, ownBatch string, now time.Time) ([]store.DropCourseItem, error)
	ListExams(ctx context.Context, batch, section string, now time.Time) ([]store.ExamRow, error)
	ListFilteredRoutines(ctx context.Context, filter store.RoutineFilter) ([]store.RoutineRow, error)
}

type Student struct {
	cfg      *config.Config
	students StudentStore
}

func NewStudent(cfg *config.Config, students StudentStore) *Student {
	return &Student{cfg: cfg, students: students}
}

type planItem struct {
	StartTime   string `json:"start_time"`
	EndTime     string `json:"end_time"`
	CourseCode  string `json:"course_code"`
	CourseTitle string `json:"course_title"`
	Kind        string `json:"kind"`
	Room        string `json:"room"`
	TeacherName string `json:"teacher_name"`
	Status      string `json:"status"`
}

type planGroup struct {
	Section      string     `json:"section"`
	SectionLabel string     `json:"section_label"`
	Items        []planItem `json:"items"`
}

type upcomingRoutineResponse struct {
	Date      string      `json:"date"`
	DateLabel string      `json:"date_label"`
	Weekday   string      `json:"weekday"`
	Batch     string      `json:"batch"`
	Groups    []planGroup `json:"groups"`
}

func (s *Student) profile(w http.ResponseWriter, r *http.Request) (store.StudentProfile, bool) {
	current, ok := middleware.CurrentUser(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, middleware.MessageUnauthorized)
		return store.StudentProfile{}, false
	}
	profile, err := s.students.GetStudentProfile(r.Context(), current.ID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			httpx.WriteError(w, http.StatusForbidden, MessageNotAStudent)
			return store.StudentProfile{}, false
		}
		log.Printf("handler: could not load the profile of a student: %v", err)
		httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
		return store.StudentProfile{}, false
	}
	return profile, true
}

func (s *Student) UpcomingRoutine() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profile, ok := s.profile(w, r)
		if !ok {
			return
		}
		date := s.cfg.Now().AddDate(0, 0, 1)
		if raw := strings.TrimSpace(r.URL.Query().Get("date")); raw != "" {
			parsed, err := validate.Date("date", raw, s.cfg.Location)
			if err != nil {
				httpx.WriteError(w, http.StatusBadRequest, err.Error())
				return
			}
			date = parsed
		}
		day := validate.DateString(date)
		items, err := s.students.ListDayPlan(r.Context(), profile.Batch, day)
		if err != nil {
			log.Printf("handler: could not list the upcoming routine of a student: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, upcomingRoutineResponse{
			Date:      day,
			DateLabel: date.Format(dateLabelLayout),
			Weekday:   validate.DayName(int(date.Weekday())),
			Batch:     profile.Batch,
			Groups:    groupPlan(items),
		})
	}
}

func groupPlan(items []store.PlanItem) []planGroup {
	bySection := make(map[string][]planItem)
	sections := make([]string, 0)
	for _, item := range items {
		if _, seen := bySection[item.Section]; !seen {
			sections = append(sections, item.Section)
		}
		bySection[item.Section] = append(bySection[item.Section], planItem{
			StartTime:   item.StartTime,
			EndTime:     item.EndTime,
			CourseCode:  item.CourseCode,
			CourseTitle: item.CourseTitle,
			Kind:        item.Kind,
			Room:        item.Room,
			TeacherName: item.TeacherName,
			Status:      planStatus(item.Cancelled),
		})
	}
	sort.Slice(sections, func(i, j int) bool {
		left, right := sections[i], sections[j]
		if (left == validate.SectionAll) != (right == validate.SectionAll) {
			return right == validate.SectionAll
		}
		return left < right
	})
	groups := make([]planGroup, 0, len(sections))
	for _, section := range sections {
		entries := bySection[section]
		sort.SliceStable(entries, func(i, j int) bool { return entries[i].StartTime < entries[j].StartTime })
		groups = append(groups, planGroup{
			Section:      section,
			SectionLabel: sectionLabel(section),
			Items:        entries,
		})
	}
	return groups
}

func planStatus(cancelled bool) string {
	if cancelled {
		return StatusCancelled
	}
	return StatusScheduled
}

func sectionLabel(section string) string {
	if section == validate.SectionAll {
		return "Both Sections"
	}
	return "Section " + section
}

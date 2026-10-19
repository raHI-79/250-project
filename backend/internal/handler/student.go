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
	GroupDropCourses   = "drop_courses"
	LabelDropCourses   = "My drop courses"
	dateLabelLayout    = "02.01.06"
	latestNoticeCount  = 3
)

type StudentStore interface {
	GetStudentProfile(ctx context.Context, userID int64) (store.StudentProfile, error)
	ListStudentDropCourses(ctx context.Context, studentID int64) ([]store.Course, error)
	ListDayPlan(ctx context.Context, batch, date string, dropCourseCodes []string) ([]store.PlanItem, error)
	ListExams(ctx context.Context, batch, section string, now time.Time) ([]store.ExamRow, error)
	ListFilteredRoutines(ctx context.Context, filter store.RoutineFilter) ([]store.RoutineRow, error)
	ListStudentNotices(ctx context.Context, batch, section, source, scope string) ([]store.NoticeRow, error)
	ListStudentPolls(ctx context.Context, viewerUserID int64, batch, section string) ([]store.PollRow, error)
	GetMessageViewer(ctx context.Context, userID int64) (store.MessageViewer, error)
	CountUnreadMessages(ctx context.Context, viewer store.MessageViewer) (int, error)
}

type Student struct {
	cfg      *config.Config
	students StudentStore
}

func NewStudent(cfg *config.Config, students StudentStore) *Student {
	return &Student{cfg: cfg, students: students}
}

type planItem struct {
	Batch       string `json:"batch"`
	StartTime   string `json:"start_time"`
	EndTime     string `json:"end_time"`
	CourseCode  string `json:"course_code"`
	CourseTitle string `json:"course_title"`
	Kind        string `json:"kind"`
	Room        string `json:"room"`
	TeacherName string `json:"teacher_name"`
	Status      string `json:"status"`
	Note        string `json:"note,omitempty"`
}

type planGroup struct {
	Key          string     `json:"key"`
	Batch        string     `json:"batch"`
	Section      string     `json:"section"`
	SectionLabel string     `json:"section_label"`
	DropCourse   bool       `json:"drop_course"`
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
		date, ok := s.requestedDate(w, r)
		if !ok {
			return
		}
		groups, ok := s.dayGroups(w, r, profile, validate.DateString(date))
		if !ok {
			return
		}
		httpx.WriteJSON(w, http.StatusOK, upcomingRoutineResponse{
			Date:      validate.DateString(date),
			DateLabel: date.Format(dateLabelLayout),
			Weekday:   validate.DayName(int(date.Weekday())),
			Batch:     profile.Batch,
			Groups:    groups,
		})
	}
}

func (s *Student) requestedDate(w http.ResponseWriter, r *http.Request) (time.Time, bool) {
	date := s.cfg.Now()
	raw := strings.TrimSpace(r.URL.Query().Get("date"))
	if raw == "" {
		return date, true
	}
	parsed, err := validate.Date("date", raw, s.cfg.Location)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return time.Time{}, false
	}
	return parsed, true
}

func (s *Student) dayGroups(w http.ResponseWriter, r *http.Request, profile store.StudentProfile, day string) ([]planGroup, bool) {
	codes, ok := s.dropCourseCodes(w, r, profile.StudentID)
	if !ok {
		return nil, false
	}
	items, err := s.students.ListDayPlan(r.Context(), profile.Batch, day, codes)
	if err != nil {
		log.Printf("handler: could not list the upcoming routine of a student: %v", err)
		httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
		return nil, false
	}
	return groupPlan(items, profile.Batch), true
}

func (s *Student) dropCourseCodes(w http.ResponseWriter, r *http.Request, studentID int64) ([]string, bool) {
	courses, err := s.students.ListStudentDropCourses(r.Context(), studentID)
	if err != nil {
		log.Printf("handler: could not list the drop courses of a student: %v", err)
		httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
		return nil, false
	}
	codes := make([]string, 0, len(courses))
	for _, course := range courses {
		codes = append(codes, course.Code)
	}
	return codes, true
}

func groupPlan(items []store.PlanItem, ownBatch string) []planGroup {
	byKey := make(map[string][]planItem)
	sections := make([]string, 0)
	for _, item := range items {
		key := GroupDropCourses
		if item.Batch == ownBatch {
			key = item.Section
			if _, seen := byKey[key]; !seen {
				sections = append(sections, key)
			}
		}
		byKey[key] = append(byKey[key], newPlanItem(item))
	}
	sort.Slice(sections, func(i, j int) bool {
		if (sections[i] == validate.SectionAll) != (sections[j] == validate.SectionAll) {
			return sections[j] == validate.SectionAll
		}
		return sections[i] < sections[j]
	})
	groups := make([]planGroup, 0, len(sections)+1)
	for _, section := range sections {
		groups = append(groups, planGroup{
			Key:          section,
			Batch:        ownBatch,
			Section:      section,
			SectionLabel: sectionLabel(section),
			Items:        sortedPlan(byKey[section]),
		})
	}
	if dropped, ok := byKey[GroupDropCourses]; ok {
		groups = append(groups, planGroup{
			Key:          GroupDropCourses,
			SectionLabel: LabelDropCourses,
			DropCourse:   true,
			Items:        sortedPlan(dropped),
		})
	}
	return groups
}

func newPlanItem(item store.PlanItem) planItem {
	return planItem{
		Batch:       item.Batch,
		StartTime:   item.StartTime,
		EndTime:     item.EndTime,
		CourseCode:  item.CourseCode,
		CourseTitle: item.CourseTitle,
		Kind:        item.Kind,
		Room:        item.Room,
		TeacherName: item.TeacherName,
		Status:      planStatus(item.Cancelled),
		Note:        item.Note,
	}
}

func sortedPlan(entries []planItem) []planItem {
	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].StartTime != entries[j].StartTime {
			return entries[i].StartTime < entries[j].StartTime
		}
		return entries[i].Batch < entries[j].Batch
	})
	return entries
}

func planStatus(cancelled bool) string {
	if cancelled {
		return StatusCancelled
	}
	return StatusScheduled
}

func sectionLabel(section string) string {
	if section == validate.SectionAll {
		return "Both sections"
	}
	return "Section " + section
}

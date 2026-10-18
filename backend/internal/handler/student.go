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
	ListStudentDropCourses(ctx context.Context, studentID int64) ([]store.Course, error)
	ListDayPlan(ctx context.Context, batch, date string, dropCourseCodes []string) ([]store.PlanItem, error)
	ListDropCourseSchedule(ctx context.Context, courseCodes []string, ownBatch string, now time.Time) ([]store.DropCourseItem, error)
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
	Batch       string `json:"batch"`
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
		codes, ok := s.dropCourseCodes(w, r, profile.StudentID)
		if !ok {
			return
		}
		items, err := s.students.ListDayPlan(r.Context(), profile.Batch, day, codes)
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
			Groups:    groupPlan(items, profile.Batch),
		})
	}
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
	keys := make([]string, 0)
	owners := make(map[string]store.PlanItem)
	for _, item := range items {
		key := item.Section
		if item.Batch != ownBatch {
			key = item.Batch + " " + item.Section
		}
		if _, seen := byKey[key]; !seen {
			keys = append(keys, key)
			owners[key] = item
		}
		byKey[key] = append(byKey[key], planItem{
			Batch:       item.Batch,
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
	sort.Slice(keys, func(i, j int) bool {
		left, right := owners[keys[i]], owners[keys[j]]
		if (left.Batch == ownBatch) != (right.Batch == ownBatch) {
			return left.Batch == ownBatch
		}
		if left.Batch != right.Batch {
			return left.Batch < right.Batch
		}
		if (left.Section == validate.SectionAll) != (right.Section == validate.SectionAll) {
			return right.Section == validate.SectionAll
		}
		return left.Section < right.Section
	})
	groups := make([]planGroup, 0, len(keys))
	for _, key := range keys {
		entries := byKey[key]
		sort.SliceStable(entries, func(i, j int) bool { return entries[i].StartTime < entries[j].StartTime })
		owner := owners[key]
		dropCourse := owner.Batch != ownBatch
		groups = append(groups, planGroup{
			Batch:        owner.Batch,
			Section:      owner.Section,
			SectionLabel: planSectionLabel(owner.Batch, owner.Section, dropCourse),
			DropCourse:   dropCourse,
			Items:        entries,
		})
	}
	return groups
}

func planSectionLabel(batch, section string, dropCourse bool) string {
	if !dropCourse {
		return sectionLabel(section)
	}
	return batch + " " + sectionLabel(section) + " (drop course)"
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

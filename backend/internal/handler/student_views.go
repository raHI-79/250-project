package handler

import (
	"errors"
	"github.com/raHI-79/250-project/backend/internal/httpx"
	"github.com/raHI-79/250-project/backend/internal/middleware"
	"github.com/raHI-79/250-project/backend/internal/store"
	"github.com/raHI-79/250-project/backend/internal/validate"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

type dropCourseItem struct {
	Batch       string `json:"batch"`
	Section     string `json:"section"`
	Source      string `json:"source"`
	Kind        string `json:"kind"`
	DayOfWeek   int    `json:"day_of_week"`
	Day         string `json:"day"`
	EventDate   string `json:"event_date,omitempty"`
	StartTime   string `json:"start_time"`
	EndTime     string `json:"end_time"`
	Room        string `json:"room"`
	TeacherName string `json:"teacher_name"`
	CourseCode  string `json:"course_code"`
	CourseTitle string `json:"course_title"`
}

type dropCourseGroup struct {
	Batch string           `json:"batch"`
	Items []dropCourseItem `json:"items"`
}

type dropCourseResponse struct {
	HasDropCourse bool              `json:"has_drop_course"`
	Courses       []profileCourse   `json:"courses"`
	Items         []dropCourseItem  `json:"items"`
	Groups        []dropCourseGroup `json:"groups"`
}

type examItem struct {
	ID          int64  `json:"id"`
	EventDate   string `json:"event_date"`
	StartTime   string `json:"start_time"`
	EndTime     string `json:"end_time"`
	CourseCode  string `json:"course_code"`
	CourseTitle string `json:"course_title"`
	Batch       string `json:"batch"`
	Section     string `json:"section"`
	Room        string `json:"room"`
}

type examsResponse struct {
	Items []examItem `json:"items"`
}

func (s *Student) DropCourseRoutine() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profile, ok := s.profile(w, r)
		if !ok {
			return
		}
		courses, err := s.students.ListStudentDropCourses(r.Context(), profile.StudentID)
		if err != nil {
			log.Printf("handler: could not list the drop courses of a student: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		if len(courses) == 0 {
			httpx.WriteJSON(w, http.StatusOK, dropCourseResponse{
				Courses: make([]profileCourse, 0),
				Items:   make([]dropCourseItem, 0),
				Groups:  make([]dropCourseGroup, 0),
			})
			return
		}
		codes := make([]string, 0, len(courses))
		for _, course := range courses {
			codes = append(codes, course.Code)
		}
		entries, err := s.students.ListDropCourseSchedule(r.Context(), codes, profile.Batch, s.cfg.Now())
		if err != nil {
			log.Printf("handler: could not list the drop course routine of a student: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		items := make([]dropCourseItem, 0, len(entries))
		groups := make([]dropCourseGroup, 0)
		for _, entry := range entries {
			item := dropCourseItem{
				Batch:       entry.Batch,
				Section:     entry.Section,
				Source:      entry.Source,
				Kind:        entry.Kind,
				DayOfWeek:   entry.DayOfWeek,
				Day:         validate.DayName(entry.DayOfWeek),
				EventDate:   entry.EventDate,
				StartTime:   entry.StartTime,
				EndTime:     entry.EndTime,
				Room:        entry.Room,
				TeacherName: entry.TeacherName,
				CourseCode:  entry.CourseCode,
				CourseTitle: entry.CourseTitle,
			}
			items = append(items, item)
			if len(groups) == 0 || groups[len(groups)-1].Batch != item.Batch {
				groups = append(groups, dropCourseGroup{Batch: item.Batch, Items: make([]dropCourseItem, 0, 1)})
			}
			groups[len(groups)-1].Items = append(groups[len(groups)-1].Items, item)
		}
		httpx.WriteJSON(w, http.StatusOK, dropCourseResponse{
			HasDropCourse: true,
			Courses:       profileCourses(courses),
			Items:         items,
			Groups:        groups,
		})
	}
}

func (s *Student) Exams() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profile, ok := s.profile(w, r)
		if !ok {
			return
		}
		exams, err := s.students.ListExams(r.Context(), profile.Batch, profile.Section, s.cfg.Now())
		if err != nil {
			log.Printf("handler: could not list the exams of a student: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		items := make([]examItem, 0, len(exams))
		for _, exam := range exams {
			items = append(items, examItem{
				ID:          exam.ID,
				EventDate:   exam.EventDate,
				StartTime:   exam.StartTime,
				EndTime:     exam.EndTime,
				CourseCode:  exam.CourseCode,
				CourseTitle: exam.CourseTitle,
				Batch:       exam.Batch,
				Section:     exam.Section,
				Room:        exam.Room,
			})
		}
		httpx.WriteJSON(w, http.StatusOK, examsResponse{Items: items})
	}
}

func (s *Student) Routine() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profile, ok := s.profile(w, r)
		if !ok {
			return
		}
		filter, err := routineBrowserFilter(r.URL.Query(), profile)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		entries, err := s.students.ListFilteredRoutines(r.Context(), filter)
		if err != nil {
			log.Printf("handler: could not list the routine for a student: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		items := make([]routineItem, 0, len(entries))
		for _, entry := range entries {
			items = append(items, newRoutineItem(entry))
		}
		httpx.WriteJSON(w, http.StatusOK, routinesResponse{Items: items})
	}
}

func routineBrowserFilter(query url.Values, profile store.StudentProfile) (store.RoutineFilter, error) {
	filter := store.RoutineFilter{
		Batch:     profile.Batch,
		Section:   profile.Section,
		DayOfWeek: store.NoDayFilter,
	}
	if raw := strings.TrimSpace(query.Get("batch")); raw != "" {
		batch, err := validate.Batch("batch", raw)
		if err != nil {
			return store.RoutineFilter{}, err
		}
		filter.Batch = batch
	}
	if raw := strings.TrimSpace(query.Get("section")); raw != "" {
		section, err := validate.RoutineSection("section", raw)
		if err != nil {
			return store.RoutineFilter{}, err
		}
		filter.Section = section
	}
	if raw := strings.TrimSpace(query.Get("day")); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil {
			return store.RoutineFilter{}, errors.New("day must be a number from 0 (Sunday) to 6 (Saturday).")
		}
		day, err := validate.DayOfWeek("day", value)
		if err != nil {
			return store.RoutineFilter{}, err
		}
		filter.DayOfWeek = day
	}
	if raw := strings.TrimSpace(query.Get("course")); raw != "" {
		code, err := validate.CourseCode("course", raw)
		if err != nil {
			return store.RoutineFilter{}, err
		}
		filter.CourseCode = code
	}
	return filter, nil
}

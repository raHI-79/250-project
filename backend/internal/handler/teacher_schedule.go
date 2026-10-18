package handler

import (
	"github.com/raHI-79/250-project/backend/internal/httpx"
	"github.com/raHI-79/250-project/backend/internal/middleware"
	"github.com/raHI-79/250-project/backend/internal/validate"
	"log"
	"net/http"
	"strings"
)

type scheduleItem struct {
	Kind        string `json:"kind"`
	ID          int64  `json:"id"`
	Batch       string `json:"batch"`
	Section     string `json:"section"`
	CourseCode  string `json:"course_code"`
	CourseTitle string `json:"course_title"`
	StartTime   string `json:"start_time"`
	EndTime     string `json:"end_time"`
	Room        string `json:"room"`
	Cancelled   bool   `json:"cancelled"`
	Note        string `json:"note,omitempty"`
}

type scheduleResponse struct {
	Date    string         `json:"date"`
	Weekday string         `json:"weekday"`
	Items   []scheduleItem `json:"items"`
}

type classItem struct {
	ID          int64  `json:"id"`
	EventType   string `json:"event_type"`
	Batch       string `json:"batch"`
	Section     string `json:"section"`
	CourseCode  string `json:"course_code"`
	CourseTitle string `json:"course_title"`
	EventDate   string `json:"event_date"`
	StartTime   string `json:"start_time"`
	EndTime     string `json:"end_time"`
	Room        string `json:"room"`
	Status      string `json:"status"`
	Note        string `json:"note,omitempty"`
}

type classesResponse struct {
	Items []classItem `json:"items"`
}

func (t *Teacher) Schedule() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		teacher, ok := t.profile(w, r)
		if !ok {
			return
		}
		raw := strings.TrimSpace(r.URL.Query().Get("date"))
		if raw == "" {
			httpx.WriteError(w, http.StatusBadRequest, MessageDateRequired)
			return
		}
		date, err := validate.Date("date", raw, t.cfg.Location)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		day := validate.DateString(date)
		entries, err := t.teachers.ListTeacherSchedule(r.Context(), teacher.ID, day)
		if err != nil {
			log.Printf("handler: could not list the schedule of a teacher: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		items := make([]scheduleItem, 0, len(entries))
		for _, entry := range entries {
			items = append(items, scheduleItem{
				Kind:        entry.Kind,
				ID:          entry.ID,
				Batch:       entry.Batch,
				Section:     entry.Section,
				CourseCode:  entry.CourseCode,
				CourseTitle: entry.CourseTitle,
				StartTime:   entry.StartTime,
				EndTime:     entry.EndTime,
				Room:        entry.Room,
				Cancelled:   entry.Cancelled,
				Note:        entry.Note,
			})
		}
		httpx.WriteJSON(w, http.StatusOK, scheduleResponse{
			Date:    day,
			Weekday: validate.DayName(int(date.Weekday())),
			Items:   items,
		})
	}
}

func (t *Teacher) UpcomingClasses() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		teacher, ok := t.profile(w, r)
		if !ok {
			return
		}
		var day string
		if raw := strings.TrimSpace(r.URL.Query().Get("date")); raw != "" {
			date, err := validate.Date("date", raw, t.cfg.Location)
			if err != nil {
				httpx.WriteError(w, http.StatusBadRequest, err.Error())
				return
			}
			day = validate.DateString(date)
		}
		classes, err := t.teachers.ListTeacherClasses(r.Context(), teacher.ID, day, t.cfg.Now())
		if err != nil {
			log.Printf("handler: could not list the classes of a teacher: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		items := make([]classItem, 0, len(classes))
		for _, class := range classes {
			items = append(items, classItem{
				ID:          class.ID,
				EventType:   class.EventType,
				Batch:       class.Batch,
				Section:     class.Section,
				CourseCode:  class.CourseCode,
				CourseTitle: class.CourseTitle,
				EventDate:   class.EventDate,
				StartTime:   class.StartTime,
				EndTime:     class.EndTime,
				Room:        class.Room,
				Note:        class.Note,
				Status:      class.Status,
			})
		}
		httpx.WriteJSON(w, http.StatusOK, classesResponse{Items: items})
	}
}

package handler

import (
	"github.com/raHI-79/250-project/backend/internal/httpx"
	"github.com/raHI-79/250-project/backend/internal/middleware"
	"github.com/raHI-79/250-project/backend/internal/store"
	"github.com/raHI-79/250-project/backend/internal/validate"
	"log"
	"net/http"
	"strings"
)

type courseTeacher struct {
	ID     int64  `json:"id"`
	UserID int64  `json:"user_id"`
	Name   string `json:"name"`
}

type courseSlot struct {
	DayOfWeek int    `json:"day_of_week"`
	Day       string `json:"day"`
	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`
	Room      string `json:"room"`
	Type      string `json:"type"`
}

type crCourseItem struct {
	ID      int64         `json:"id"`
	Code    string        `json:"code"`
	Title   string        `json:"title"`
	Teacher courseTeacher `json:"teacher"`
	Slots   []courseSlot  `json:"slots"`
}

type crCoursesResponse struct {
	Items []crCourseItem `json:"items"`
}

type dayEntryItem struct {
	Source      string `json:"source"`
	Section     string `json:"section"`
	CourseCode  string `json:"course_code"`
	CourseTitle string `json:"course_title"`
	TeacherName string `json:"teacher_name"`
	StartTime   string `json:"start_time"`
	EndTime     string `json:"end_time"`
	Room        string `json:"room"`
	Kind        string `json:"kind"`
	Status      string `json:"status"`
}

type crDayResponse struct {
	Date    string         `json:"date"`
	Weekday string         `json:"weekday"`
	Batch   string         `json:"batch"`
	Items   []dayEntryItem `json:"items"`
}

func (c *CR) Courses() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profile, ok := c.profile(w, r)
		if !ok {
			return
		}
		slots, err := c.classes.ListBatchCourseSlots(r.Context(), profile.Batch, profile.Section)
		if err != nil {
			log.Printf("handler: could not list the courses of a class representative: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, crCoursesResponse{Items: groupCourseSlots(slots)})
	}
}

func (c *CR) Day() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profile, ok := c.profile(w, r)
		if !ok {
			return
		}
		raw := strings.TrimSpace(r.URL.Query().Get("date"))
		if raw == "" {
			httpx.WriteError(w, http.StatusBadRequest, MessageDateRequired)
			return
		}
		date, err := validate.Date("date", raw, c.cfg.Location)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		day := validate.DateString(date)
		entries, err := c.classes.ListBatchDay(r.Context(), profile.Batch, day)
		if err != nil {
			log.Printf("handler: could not list the day of a class representative: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		items := make([]dayEntryItem, 0, len(entries))
		for _, entry := range entries {
			items = append(items, dayEntryItem{
				Source:      entry.Source,
				Section:     entry.Section,
				CourseCode:  entry.CourseCode,
				CourseTitle: entry.CourseTitle,
				TeacherName: entry.TeacherName,
				StartTime:   entry.StartTime,
				EndTime:     entry.EndTime,
				Room:        entry.Room,
				Kind:        entry.Kind,
				Status:      entry.Status,
			})
		}
		httpx.WriteJSON(w, http.StatusOK, crDayResponse{
			Date:    day,
			Weekday: validate.DayName(int(date.Weekday())),
			Batch:   profile.Batch,
			Items:   items,
		})
	}
}

func groupCourseSlots(slots []store.CourseSlotRow) []crCourseItem {
	items := make([]crCourseItem, 0)
	index := make(map[int64]int)
	for _, slot := range slots {
		position, seen := index[slot.CourseID]
		if !seen {
			position = len(items)
			index[slot.CourseID] = position
			items = append(items, crCourseItem{
				ID:    slot.CourseID,
				Code:  slot.CourseCode,
				Title: slot.CourseTitle,
				Teacher: courseTeacher{
					ID:     slot.TeacherID,
					UserID: slot.TeacherUserID,
					Name:   slot.TeacherName,
				},
				Slots: make([]courseSlot, 0, 1),
			})
		}
		items[position].Slots = append(items[position].Slots, courseSlot{
			DayOfWeek: slot.DayOfWeek,
			Day:       validate.DayName(slot.DayOfWeek),
			StartTime: slot.StartTime,
			EndTime:   slot.EndTime,
			Room:      slot.Room,
			Type:      slot.Type,
		})
	}
	return items
}

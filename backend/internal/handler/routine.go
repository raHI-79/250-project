package handler

import (
	"errors"
	"fmt"
	"github.com/raHI-79/250-project/backend/internal/httpx"
	"github.com/raHI-79/250-project/backend/internal/middleware"
	"github.com/raHI-79/250-project/backend/internal/store"
	"github.com/raHI-79/250-project/backend/internal/validate"
	"log"
	"net/http"
	"strings"
)

const (
	MessageRoutineCreated  = "Routine entry added."
	MessageRoutineNotFound = "No routine entry found with this ID."
	MessageCourseRequired  = "Course is required."
	MessageTeacherRequired = "Teacher is required."
)

type createRoutineRequest struct {
	Batch     string `json:"batch"`
	Section   string `json:"section"`
	CourseID  int64  `json:"course_id"`
	TeacherID int64  `json:"teacher_id"`
	DayOfWeek int    `json:"day_of_week"`
	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`
	Room      string `json:"room"`
	Type      string `json:"type"`
}

type routineItem struct {
	ID          int64  `json:"id"`
	Batch       string `json:"batch"`
	Section     string `json:"section"`
	CourseID    int64  `json:"course_id"`
	CourseCode  string `json:"course_code"`
	CourseTitle string `json:"course_title"`
	TeacherID   int64  `json:"teacher_id"`
	TeacherName string `json:"teacher_name"`
	DayOfWeek   int    `json:"day_of_week"`
	Day         string `json:"day"`
	StartTime   string `json:"start_time"`
	EndTime     string `json:"end_time"`
	Room        string `json:"room"`
	Type        string `json:"type"`
}

type routinesResponse struct {
	Items []routineItem `json:"items"`
}

type routineResponse struct {
	routineItem
	Message string `json:"message"`
}

func (c *Catalog) CreateRoutine() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var request createRoutineRequest
		if !httpx.DecodeJSON(w, r, &request) {
			return
		}
		entry, err := request.toNewRoutine()
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		created, err := c.catalog.CreateRoutine(r.Context(), entry)
		var booked *store.RoomBookedError
		switch {
		case errors.As(err, &booked):
			httpx.WriteError(w, http.StatusConflict, roomBookedMessage(booked.Booking))
		case errors.Is(err, store.ErrCourseNotFound):
			httpx.WriteError(w, http.StatusBadRequest, MessageCourseUnknown)
		case errors.Is(err, store.ErrTeacherNotFound):
			httpx.WriteError(w, http.StatusBadRequest, MessageTeacherNotFound)
		case errors.Is(err, store.ErrTeacherInactive):
			httpx.WriteError(w, http.StatusBadRequest, MessageTeacherNotActive)
		case err != nil:
			log.Printf("handler: could not create a routine entry: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
		default:
			httpx.WriteJSON(w, http.StatusCreated, routineResponse{
				routineItem: newRoutineItem(created),
				Message:     MessageRoutineCreated,
			})
		}
	}
}

func (request createRoutineRequest) toNewRoutine() (store.NewRoutine, error) {
	var entry store.NewRoutine
	batch, err := validate.Batch("Batch", request.Batch)
	if err != nil {
		return store.NewRoutine{}, err
	}
	entry.Batch = batch
	section, err := validate.RoutineSection("Section", request.Section)
	if err != nil {
		return store.NewRoutine{}, err
	}
	entry.Section = section
	if request.CourseID <= 0 {
		return store.NewRoutine{}, errors.New(MessageCourseRequired)
	}
	entry.CourseID = request.CourseID
	if request.TeacherID <= 0 {
		return store.NewRoutine{}, errors.New(MessageTeacherRequired)
	}
	entry.TeacherID = request.TeacherID
	day, err := validate.DayOfWeek("Day of week", request.DayOfWeek)
	if err != nil {
		return store.NewRoutine{}, err
	}
	entry.DayOfWeek = day
	start, err := validate.ClockTime("Start time", request.StartTime)
	if err != nil {
		return store.NewRoutine{}, err
	}
	end, err := validate.ClockTime("End time", request.EndTime)
	if err != nil {
		return store.NewRoutine{}, err
	}
	if err := validate.ClassWindow(start, end); err != nil {
		return store.NewRoutine{}, err
	}
	entry.StartTime, entry.EndTime = start, end
	room, err := validate.Required("Room", request.Room)
	if err != nil {
		return store.NewRoutine{}, err
	}
	entry.Room = room
	entryType, err := validate.RoutineType("Type", request.Type)
	if err != nil {
		return store.NewRoutine{}, err
	}
	entry.Type = entryType
	return entry, nil
}

func roomBookedMessage(booking store.RoomBooking) string {
	audience := "section " + booking.Section
	if booking.Section == validate.SectionAll {
		audience = "both sections"
	}
	return fmt.Sprintf("Room %s is already booked on %s %s–%s by %s %s (%s).",
		booking.Room,
		validate.DayName(booking.DayOfWeek),
		booking.StartTime,
		booking.EndTime,
		booking.Batch,
		audience,
		booking.CourseCode,
	)
}

func newRoutineItem(entry store.RoutineRow) routineItem {
	return routineItem{
		ID:          entry.ID,
		Batch:       entry.Batch,
		Section:     entry.Section,
		CourseID:    entry.CourseID,
		CourseCode:  entry.CourseCode,
		CourseTitle: entry.CourseTitle,
		TeacherID:   entry.TeacherID,
		TeacherName: entry.TeacherName,
		DayOfWeek:   entry.DayOfWeek,
		Day:         validate.DayName(entry.DayOfWeek),
		StartTime:   entry.StartTime,
		EndTime:     entry.EndTime,
		Room:        entry.Room,
		Type:        entry.Type,
	}
}

func (c *Catalog) Routines() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		var batch, section string
		if raw := strings.TrimSpace(query.Get("batch")); raw != "" {
			value, err := validate.Batch("batch", raw)
			if err != nil {
				httpx.WriteError(w, http.StatusBadRequest, err.Error())
				return
			}
			batch = value
		}
		if raw := strings.TrimSpace(query.Get("section")); raw != "" {
			value, err := validate.RoutineSection("section", raw)
			if err != nil {
				httpx.WriteError(w, http.StatusBadRequest, err.Error())
				return
			}
			section = value
		}
		entries, err := c.catalog.ListRoutines(r.Context(), batch, section)
		if err != nil {
			log.Printf("handler: could not list the routine entries: %v", err)
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

func (c *Catalog) DeleteRoutine() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		routineID, ok := pathID(w, r)
		if !ok {
			return
		}
		err := c.catalog.DeleteRoutine(r.Context(), routineID)
		switch {
		case errors.Is(err, store.ErrNotFound):
			httpx.WriteError(w, http.StatusNotFound, MessageRoutineNotFound)
		case err != nil:
			log.Printf("handler: could not delete a routine entry: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
		default:
			httpx.WriteJSON(w, http.StatusNoContent, nil)
		}
	}
}

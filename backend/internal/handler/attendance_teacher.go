package handler

import (
	"errors"
	"github.com/raHI-79/250-project/backend/internal/attendance"
	"github.com/raHI-79/250-project/backend/internal/httpx"
	"github.com/raHI-79/250-project/backend/internal/middleware"
	"github.com/raHI-79/250-project/backend/internal/store"
	"github.com/raHI-79/250-project/backend/internal/validate"
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type startRequest struct {
	CourseID       int64  `json:"course_id"`
	Batch          string `json:"batch"`
	Section        string `json:"section"`
	ScheduleItemID int64  `json:"schedule_item_id"`
	Minutes        int    `json:"minutes"`
}

type extendRequest struct {
	Minutes int `json:"minutes"`
}

type locationRequest struct {
	Lat       float64 `json:"lat"`
	Lng       float64 `json:"lng"`
	AccuracyM float64 `json:"accuracy_m"`
}

type editMarkRequest struct {
	Status string `json:"status"`
	Note   string `json:"note"`
}

type todayClassBody struct {
	ScheduleItemID int64  `json:"schedule_item_id"`
	CourseID       int64  `json:"course_id"`
	CourseCode     string `json:"course_code"`
	CourseTitle    string `json:"course_title"`
	Batch          string `json:"batch"`
	Section        string `json:"section"`
	StartTime      string `json:"start_time"`
	EndTime        string `json:"end_time"`
	Room           string `json:"room"`
}

type optionsResponse struct {
	Today   []todayClassBody `json:"today"`
	Classes []todayClassBody `json:"classes"`
	Open    *sessionBody     `json:"open,omitempty"`
}

func (a *Attendance) Options() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		teacher, ok := a.teacher(w, r)
		if !ok {
			return
		}
		now := a.cfg.Now()
		items, err := a.store.ListTeacherDayItems(r.Context(), teacher.ID, validate.DateString(now))
		if err != nil {
			log.Printf("handler: could not list the classes of a teacher for attendance: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		rows, err := a.store.ListTeacherRoutine(r.Context(), teacher.ID)
		if err != nil {
			log.Printf("handler: could not list the routine of a teacher for attendance: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		response := optionsResponse{Today: make([]todayClassBody, 0), Classes: make([]todayClassBody, 0)}
		for _, item := range items {
			if item.Kind == "exam" || item.Status == "cancelled" {
				continue
			}
			response.Today = append(response.Today, todayClassBody{
				ScheduleItemID: item.ID,
				CourseID:       item.CourseID,
				CourseCode:     item.CourseCode,
				CourseTitle:    item.CourseTitle,
				Batch:          item.Batch,
				Section:        item.Section,
				StartTime:      item.StartTime,
				EndTime:        item.EndTime,
				Room:           item.Room,
			})
		}
		seen := make(map[string]bool)
		for _, row := range rows {
			key := strconv.FormatInt(row.CourseID, 10) + "|" + row.Batch + "|" + row.Section
			if seen[key] {
				continue
			}
			seen[key] = true
			response.Classes = append(response.Classes, todayClassBody{
				CourseID:    row.CourseID,
				CourseCode:  row.CourseCode,
				CourseTitle: row.CourseTitle,
				Batch:       row.Batch,
				Section:     row.Section,
			})
		}
		from := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		sessions, err := a.store.ListTeacherAttendance(r.Context(), teacher.ID, from, now.Add(time.Minute))
		if err != nil {
			log.Printf("handler: could not look for an open attendance session: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		for _, session := range sessions {
			if session.Status != attendance.StatusOpen || !session.EndsAt.After(now) {
				continue
			}
			full, err := a.store.AttendanceSession(r.Context(), session.ID)
			if err != nil {
				log.Printf("handler: could not read the open attendance session: %v", err)
				break
			}
			body := newSessionBody(full, now, true)
			response.Open = &body
			break
		}
		httpx.WriteJSON(w, http.StatusOK, response)
	}
}

func (a *Attendance) Start() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		teacher, ok := a.teacher(w, r)
		if !ok {
			return
		}
		var request startRequest
		if !httpx.DecodeJSON(w, r, &request) {
			return
		}
		batch, err := validate.Batch("Batch", request.Batch)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		section, err := validate.RoutineSection("Section", request.Section)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		if request.CourseID <= 0 {
			httpx.WriteError(w, http.StatusBadRequest, "Pick the course of this class.")
			return
		}
		if !attendance.ValidDuration(request.Minutes) {
			httpx.WriteError(w, http.StatusBadRequest, attendance.MessageBadDuration)
			return
		}
		mine, err := a.teaches(r.Context(), teacher.ID, request.CourseID, batch, section)
		if err != nil {
			log.Printf("handler: could not check the classes of a teacher: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		if !mine {
			httpx.WriteError(w, http.StatusForbidden, MessageNotYourClass)
			return
		}
		key, err := attendance.NewKey()
		if err != nil {
			log.Printf("handler: could not draw an attendance key: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		now := a.cfg.Now()
		session, err := a.store.OpenAttendanceSession(r.Context(), store.NewAttendanceSession{
			CourseID:       request.CourseID,
			Batch:          batch,
			Section:        section,
			ScheduleItemID: request.ScheduleItemID,
			TeacherID:      teacher.ID,
			EndsAt:         now.Add(time.Duration(request.Minutes) * time.Minute),
			TokenKey:       key,
		})
		if err != nil {
			log.Printf("handler: could not open an attendance session: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		if session.TokenKey == key {
			a.announce(r.Context(), session)
		}
		httpx.WriteJSON(w, http.StatusCreated, newSessionBody(session, now, true))
	}
}

func (a *Attendance) Session() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		teacher, ok := a.teacher(w, r)
		if !ok {
			return
		}
		session, ok := a.ownSession(w, r, teacher)
		if !ok {
			return
		}
		records, err := a.store.ListSessionRecords(r.Context(), session.ID)
		if err != nil {
			log.Printf("handler: could not list the marks of an attendance session: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, sessionDetailResponse{
			Session: newSessionBody(session, a.cfg.Now(), true),
			Marks:   newMarkBodies(records),
		})
	}
}

func (a *Attendance) Location() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		teacher, ok := a.teacher(w, r)
		if !ok {
			return
		}
		session, ok := a.ownSession(w, r, teacher)
		if !ok {
			return
		}
		var request locationRequest
		if !httpx.DecodeJSON(w, r, &request) {
			return
		}
		if request.Lat == 0 && request.Lng == 0 {
			httpx.WriteError(w, http.StatusBadRequest, attendance.MessageNoPosition)
			return
		}
		err := a.store.SetAttendanceLocation(r.Context(), session.ID, teacher.ID,
			request.Lat, request.Lng, request.AccuracyM)
		if errors.Is(err, store.ErrAttendanceClosed) {
			httpx.WriteError(w, http.StatusConflict, MessageAttendanceOver)
			return
		}
		if err != nil {
			log.Printf("handler: could not save the position of an attendance session: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		fresh, err := a.store.AttendanceSession(r.Context(), session.ID)
		if err != nil {
			log.Printf("handler: could not reload an attendance session: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, newSessionBody(fresh, a.cfg.Now(), true))
	}
}

func (a *Attendance) Extend() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		teacher, ok := a.teacher(w, r)
		if !ok {
			return
		}
		session, ok := a.ownSession(w, r, teacher)
		if !ok {
			return
		}
		var request extendRequest
		if !httpx.DecodeJSON(w, r, &request) {
			return
		}
		if !attendance.ValidDuration(request.Minutes) {
			httpx.WriteError(w, http.StatusBadRequest, attendance.MessageBadDuration)
			return
		}
		longer, err := a.store.ExtendAttendanceSession(r.Context(), session.ID, teacher.ID, request.Minutes)
		if errors.Is(err, store.ErrAttendanceClosed) {
			httpx.WriteError(w, http.StatusConflict, MessageAttendanceOver)
			return
		}
		if err != nil {
			log.Printf("handler: could not extend an attendance session: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, newSessionBody(longer, a.cfg.Now(), true))
	}
}

func (a *Attendance) Close() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		teacher, ok := a.teacher(w, r)
		if !ok {
			return
		}
		session, ok := a.ownSession(w, r, teacher)
		if !ok {
			return
		}
		closed, err := a.store.CloseAttendanceSession(r.Context(), session.ID, teacher.ID)
		if errors.Is(err, store.ErrAttendanceClosed) {
			httpx.WriteError(w, http.StatusConflict, MessageAttendanceOver)
			return
		}
		if err != nil {
			log.Printf("handler: could not close an attendance session: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, newSessionBody(closed, a.cfg.Now(), false))
	}
}

func (a *Attendance) History() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		teacher, ok := a.teacher(w, r)
		if !ok {
			return
		}
		from, to := a.window(r)
		sessions, err := a.store.ListTeacherAttendance(r.Context(), teacher.ID, from, to)
		if err != nil {
			log.Printf("handler: could not list the attendance history of a teacher: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		now := a.cfg.Now()
		items := make([]sessionBody, 0, len(sessions))
		for _, session := range sessions {
			items = append(items, newSessionBody(session, now, false))
		}
		httpx.WriteJSON(w, http.StatusOK, sessionsResponse{Items: items})
	}
}

func (a *Attendance) EditMark() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		teacher, ok := a.teacher(w, r)
		if !ok {
			return
		}
		session, ok := a.ownSession(w, r, teacher)
		if !ok {
			return
		}
		studentID, err := strconv.ParseInt(strings.TrimSpace(r.PathValue("student_id")), 10, 64)
		if err != nil || studentID <= 0 {
			httpx.WriteError(w, http.StatusBadRequest, MessageInvalidID)
			return
		}
		var request editMarkRequest
		if !httpx.DecodeJSON(w, r, &request) {
			return
		}
		status := strings.ToLower(strings.TrimSpace(request.Status))
		if status != attendance.Present && status != attendance.Absent {
			httpx.WriteError(w, http.StatusBadRequest, MessageWrongMarkStatus)
			return
		}
		note, err := validate.Description("Note", request.Note)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		roster, err := a.store.ListRoster(r.Context(), session.ID)
		if err != nil {
			log.Printf("handler: could not list the roster for a manual mark: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		listed := false
		for _, entry := range roster {
			if entry.StudentID == studentID {
				listed = true
				break
			}
		}
		if !listed {
			httpx.WriteError(w, http.StatusNotFound, attendance.MessageNotOnRoster)
			return
		}
		current, _ := middleware.CurrentUser(r.Context())
		err = a.store.EditAttendanceRecord(r.Context(), session.ID, studentID, current.ID, status, note)
		if err != nil {
			log.Printf("handler: could not edit an attendance mark: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		records, err := a.store.ListSessionRecords(r.Context(), session.ID)
		if err != nil {
			log.Printf("handler: could not reload the marks of an attendance session: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		fresh, err := a.store.AttendanceSession(r.Context(), session.ID)
		if err != nil {
			log.Printf("handler: could not reload an attendance session: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, sessionDetailResponse{
			Session: newSessionBody(fresh, a.cfg.Now(), false),
			Marks:   newMarkBodies(records),
		})
	}
}

func (a *Attendance) Classes() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		teacher, ok := a.teacher(w, r)
		if !ok {
			return
		}
		from, to := a.window(r)
		classes, err := a.store.ListTeacherClasses(r.Context(), teacher.ID, from, to)
		if err != nil {
			log.Printf("handler: could not list the attendance classes of a teacher: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		items := make([]classBody, 0, len(classes))
		for _, class := range classes {
			rates, err := a.store.ClassRates(r.Context(), teacher.ID, class.CourseID,
				class.Batch, class.Section, from, to)
			if err != nil {
				log.Printf("handler: could not count the attendance of a class: %v", err)
				httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
				return
			}
			total, below := 0.0, 0
			for _, rate := range rates {
				share := attendance.Percentage(rate.Present, rate.Held)
				total += share
				if share < attendance.AlertPercentage {
					below++
				}
			}
			average := 0
			if len(rates) > 0 {
				average = int(math.Round(total / float64(len(rates))))
			}
			items = append(items, classBody{
				CourseID:    class.CourseID,
				CourseCode:  class.CourseCode,
				CourseTitle: class.CourseTitle,
				Batch:       class.Batch,
				Section:     class.Section,
				Sessions:    class.Sessions,
				AverageRate: average,
				BelowFifty:  below,
			})
		}
		httpx.WriteJSON(w, http.StatusOK, classesResponse{Items: items})
	}
}

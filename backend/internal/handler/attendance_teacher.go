package handler

import (
	"errors"
	"fmt"
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
	Until          string `json:"until"`
	LateAfter      int    `json:"late_after_minutes"`
	Hours          int    `json:"hours"`
	Method         string `json:"method"`
	CodeOnly       bool   `json:"code_only"`
}

type sheetRequest struct {
	CourseID int64  `json:"course_id"`
	Batch    string `json:"batch"`
	Section  string `json:"section"`
}

type emptyCloseResponse struct {
	ID      int64  `json:"id"`
	Status  string `json:"status"`
	Removed bool   `json:"removed"`
	Message string `json:"message"`
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

type noteRequest struct {
	Note string `json:"note"`
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
	Hours          int    `json:"hours"`
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
				Hours:          classHours(item.StartTime, item.EndTime),
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
				Hours:       attendance.MinHours,
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
		method := strings.ToLower(strings.TrimSpace(request.Method))
		if method == "" {
			method = attendance.TakeBluetooth
		}
		if !attendance.ValidTakeMethod(method) {
			httpx.WriteError(w, http.StatusBadRequest, attendance.MessageBadTakeMethod)
			return
		}
		now := a.cfg.Now()
		minutes := request.Minutes
		if raw := strings.TrimSpace(request.Until); raw != "" {
			clock, err := time.ParseInLocation("15:04", raw, now.Location())
			if err != nil {
				httpx.WriteError(w, http.StatusBadRequest, attendance.MessageBadUntil)
				return
			}
			until := time.Date(now.Year(), now.Month(), now.Day(), clock.Hour(), clock.Minute(), 0, 0,
				now.Location())
			minutes = attendance.MinutesUntil(now, until)
			if minutes <= 0 {
				httpx.WriteError(w, http.StatusBadRequest, attendance.MessageBadUntil)
				return
			}
		}
		if !attendance.ValidDuration(minutes) {
			httpx.WriteError(w, http.StatusBadRequest, attendance.MessageBadDuration)
			return
		}
		if !attendance.ValidLateAfter(request.LateAfter, minutes) {
			httpx.WriteError(w, http.StatusBadRequest, attendance.MessageBadLateAfter)
			return
		}
		hours := request.Hours
		if hours == 0 {
			hours = attendance.MinHours
		}
		if !attendance.ValidHours(hours) {
			httpx.WriteError(w, http.StatusBadRequest, attendance.MessageBadHours)
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
		session, err := a.store.OpenAttendanceSession(r.Context(), store.NewAttendanceSession{
			CourseID:       request.CourseID,
			Batch:          batch,
			Section:        section,
			ScheduleItemID: request.ScheduleItemID,
			TeacherID:      teacher.ID,
			Minutes:        minutes,
			Hours:          hours,
			LateAfter:      request.LateAfter,
			TokenKey:       key,
			Method:         method,
			CodeOnly:       request.CodeOnly && method != attendance.TakeRollCall,
		})
		if err != nil {
			log.Printf("handler: could not open an attendance session: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		if session.TokenKey == key {
			log.Printf("handler: attendance %d opened by teacher %d for %s %s-%s by %s, code only %t, until %s",
				session.ID, teacher.ID, session.CourseCode, session.Batch, session.Section,
				session.Method, session.CodeOnly, session.EndsAt.Format(time.TimeOnly))
			if session.Method != attendance.TakeRollCall {
				a.announce(r.Context(), session)
			}
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
		log.Printf("handler: attendance %d has a position from teacher %d, accuracy %.0f m",
			session.ID, teacher.ID, request.AccuracyM)
		httpx.WriteJSON(w, http.StatusOK, newSessionBody(fresh, a.cfg.Now(), true))
	}
}

func (a *Attendance) CodeOnly() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		teacher, ok := a.teacher(w, r)
		if !ok {
			return
		}
		session, ok := a.ownSession(w, r, teacher)
		if !ok {
			return
		}
		if session.Method == attendance.TakeRollCall {
			httpx.WriteError(w, http.StatusConflict, attendance.MessageRollCallOnly)
			return
		}
		switched, err := a.store.SwitchAttendanceToCodeOnly(r.Context(), session.ID, teacher.ID)
		if errors.Is(err, store.ErrAttendanceClosed) {
			httpx.WriteError(w, http.StatusConflict, MessageAttendanceOver)
			return
		}
		if err != nil {
			log.Printf("handler: could not switch an attendance session to code only: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		log.Printf("handler: attendance %d switched to code only by teacher %d", session.ID, teacher.ID)
		httpx.WriteJSON(w, http.StatusOK, newSessionBody(switched, a.cfg.Now(), true))
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
		if !attendance.WithinWindow(session.StartedAt,
			session.EndsAt.Add(time.Duration(request.Minutes)*time.Minute)) {
			httpx.WriteError(w, http.StatusBadRequest, attendance.MessageTooLong)
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
		if errors.Is(err, store.ErrAttendanceEmpty) {
			log.Printf("handler: attendance %d closed by teacher %d with no marks, so it was removed",
				session.ID, teacher.ID)
			httpx.WriteJSON(w, http.StatusOK, emptyCloseResponse{
				ID:      session.ID,
				Status:  attendance.StatusClosed,
				Removed: true,
				Message: MessageEmptyRemoved,
			})
			return
		}
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
		studentID, ok := studentPathID(w, r)
		if !ok {
			return
		}
		var request editMarkRequest
		if !httpx.DecodeJSON(w, r, &request) {
			return
		}
		status := strings.ToLower(strings.TrimSpace(request.Status))
		if !attendance.ValidStatus(status) {
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
		a.writeSession(w, r, session.ID)
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
				share := attendance.Rate(rate.Credit, rate.Held)
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

func (a *Attendance) MarkAll() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		teacher, ok := a.teacher(w, r)
		if !ok {
			return
		}
		session, ok := a.ownSession(w, r, teacher)
		if !ok {
			return
		}
		current, _ := middleware.CurrentUser(r.Context())
		if _, err := a.store.MarkAllPresent(r.Context(), session.ID, current.ID); err != nil {
			log.Printf("handler: could not mark a whole roster present: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		a.writeSession(w, r, session.ID)
	}
}

func (a *Attendance) SaveNote() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		teacher, ok := a.teacher(w, r)
		if !ok {
			return
		}
		session, ok := a.ownSession(w, r, teacher)
		if !ok {
			return
		}
		studentID, ok := studentPathID(w, r)
		if !ok {
			return
		}
		var request noteRequest
		if !httpx.DecodeJSON(w, r, &request) {
			return
		}
		note, err := validate.Description("Note", request.Note)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		current, _ := middleware.CurrentUser(r.Context())
		err = a.store.UpdateAttendanceNote(r.Context(), session.ID, studentID, current.ID, note)
		if errors.Is(err, store.ErrNotFound) {
			httpx.WriteError(w, http.StatusConflict, MessageMarkMissing)
			return
		}
		if err != nil {
			log.Printf("handler: could not save an attendance note: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		a.writeSession(w, r, session.ID)
	}
}

func (a *Attendance) DeleteSession() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		teacher, ok := a.teacher(w, r)
		if !ok {
			return
		}
		session, ok := a.ownSession(w, r, teacher)
		if !ok {
			return
		}
		current, _ := middleware.CurrentUser(r.Context())
		a.removeSession(w, r, session, current.ID)
	}
}

func (a *Attendance) DeleteSheet() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		teacher, ok := a.teacher(w, r)
		if !ok {
			return
		}
		request, ok := decodeSheet(w, r)
		if !ok {
			return
		}
		mine, err := a.teaches(r.Context(), teacher.ID, request.CourseID, request.Batch, request.Section)
		if err != nil {
			log.Printf("handler: could not check the classes of a teacher: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		if !mine {
			httpx.WriteError(w, http.StatusForbidden, MessageNotYourClass)
			return
		}
		current, _ := middleware.CurrentUser(r.Context())
		a.removeSheet(w, r, request, current.ID)
	}
}

func decodeSheet(w http.ResponseWriter, r *http.Request) (sheetRequest, bool) {
	var request sheetRequest
	if !httpx.DecodeJSON(w, r, &request) {
		return sheetRequest{}, false
	}
	batch, err := validate.Batch("Batch", request.Batch)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return sheetRequest{}, false
	}
	section, err := validate.RoutineSection("Section", request.Section)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return sheetRequest{}, false
	}
	if request.CourseID <= 0 {
		httpx.WriteError(w, http.StatusBadRequest, MessageSheetFields)
		return sheetRequest{}, false
	}
	return sheetRequest{CourseID: request.CourseID, Batch: batch, Section: section}, true
}

func (a *Attendance) removeSession(w http.ResponseWriter, r *http.Request,
	session store.AttendanceSession, actorID int64) {
	if err := a.store.DeleteAttendanceSession(r.Context(), session.ID); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			httpx.WriteError(w, http.StatusNotFound, MessageSessionGone)
			return
		}
		log.Printf("handler: could not delete an attendance session: %v", err)
		httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
		return
	}
	summary := sheetSummary(session.CourseCode, session.CourseTitle, session.Batch, session.Section) +
		" · " + session.StartedAt.Format(time.RFC3339)
	if err := a.store.WriteAudit(r.Context(), actorID, auditSessionDeleted, summary); err != nil {
		log.Printf("handler: could not write the attendance delete audit: %v", err)
	}
	httpx.WriteJSON(w, http.StatusOK, messageResponse{Message: MessageSessionDeleted})
}

func (a *Attendance) removeSheet(w http.ResponseWriter, r *http.Request,
	request sheetRequest, actorID int64) {
	deleted, err := a.store.DeleteAttendanceSheet(r.Context(), request.CourseID,
		request.Batch, request.Section)
	if errors.Is(err, store.ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, MessageSessionGone)
		return
	}
	if err != nil {
		log.Printf("handler: could not delete an attendance sheet: %v", err)
		httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
		return
	}
	summary := fmt.Sprintf("course %d · %s %s · %d sessions", request.CourseID,
		request.Batch, request.Section, deleted)
	if err := a.store.WriteAudit(r.Context(), actorID, auditSheetDeleted, summary); err != nil {
		log.Printf("handler: could not write the attendance delete audit: %v", err)
	}
	httpx.WriteJSON(w, http.StatusOK, messageResponse{Message: MessageSheetDeleted})
}

func (a *Attendance) writeSession(w http.ResponseWriter, r *http.Request, sessionID int64) {
	records, err := a.store.ListSessionRecords(r.Context(), sessionID)
	if err != nil {
		log.Printf("handler: could not reload the marks of an attendance session: %v", err)
		httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
		return
	}
	fresh, err := a.store.AttendanceSession(r.Context(), sessionID)
	if err != nil {
		log.Printf("handler: could not reload an attendance session: %v", err)
		httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, sessionDetailResponse{
		Session: newSessionBody(fresh, a.cfg.Now(), true),
		Marks:   newMarkBodies(records),
	})
}

func studentPathID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	studentID, err := strconv.ParseInt(strings.TrimSpace(r.PathValue("student_id")), 10, 64)
	if err != nil || studentID <= 0 {
		httpx.WriteError(w, http.StatusBadRequest, MessageInvalidID)
		return 0, false
	}
	return studentID, true
}

package handler

import (
	"context"
	"errors"
	"fmt"
	"github.com/raHI-79/250-project/backend/internal/attendance"
	"github.com/raHI-79/250-project/backend/internal/config"
	"github.com/raHI-79/250-project/backend/internal/httpx"
	"github.com/raHI-79/250-project/backend/internal/middleware"
	"github.com/raHI-79/250-project/backend/internal/notify"
	"github.com/raHI-79/250-project/backend/internal/store"
	"github.com/raHI-79/250-project/backend/internal/validate"
	"log"
	"net/http"
	"strings"
)

const (
	MessagePhoneRegistered = "This phone is now your attendance phone."
	MessagePhoneRequested  = "Your teachers were asked. You can mark attendance once one of them approves."
	MessagePhoneSame       = "This phone is already your attendance phone."
	MessagePhoneTaken      = "This phone is already registered to another account."
	MessageRequestWaiting  = "Your teachers already have a request from you. Wait for their answer."
	MessageReasonNeeded    = "Tell your teachers why you need another phone."
	MessageMarkNeedsProof  = "Send the Bluetooth token or the code on the board."

	titlePhoneRequest = " wants to use a new phone"
	titlePhoneAllowed = "Your new phone is approved"
	titlePhoneRefused = "Your phone request was refused"
	bodyPhoneAllowed  = " approved your new attendance phone."
	maxReasonLength   = 500
)

type StudentAttendanceStore interface {
	GetStudentProfile(ctx context.Context, userID int64) (store.StudentProfile, error)
	FindAccount(ctx context.Context, userID int64) (store.AccountRef, error)
	RunningSemester(ctx context.Context, batch string) (string, error)
	OpenSessionForStudent(ctx context.Context, studentID int64) (store.AttendanceSession, error)
	AttendanceSession(ctx context.Context, sessionID int64) (store.AttendanceSession, error)
	OnRoster(ctx context.Context, sessionID, studentID int64) (bool, error)
	MarkAttendance(ctx context.Context, mark store.MarkInput) error
	StudentCourseAttendance(ctx context.Context, studentID int64) ([]store.CourseAttendance, error)
	StudentAttendanceSessions(ctx context.Context, studentID int64) ([]store.StudentSessionRow, error)
	DeviceBinding(ctx context.Context, userID int64) (store.DeviceBinding, error)
	BindDevice(ctx context.Context, userID int64, deviceID string) (store.DeviceBinding, bool, error)
	TouchDevice(ctx context.Context, userID int64, deviceID string) error
	CreateDeviceChangeRequest(ctx context.Context, userID int64,
		deviceID, reason, semester string) (store.DeviceChangeRequest, error)
	LatestDeviceRequest(ctx context.Context, userID int64) (store.DeviceChangeRequest, error)
	ApprovedDeviceChanges(ctx context.Context, userID int64, semester string) (int, error)
	ListBatchTeachers(ctx context.Context, batch, section string) ([]int64, error)
}

type StudentAttendance struct {
	cfg      *config.Config
	students StudentAttendanceStore
	alerts   BroadcastNotifier
}

func NewStudentAttendance(cfg *config.Config, students StudentAttendanceStore,
	alerts BroadcastNotifier) *StudentAttendance {
	return &StudentAttendance{cfg: cfg, students: students, alerts: alerts}
}

type markRequest struct {
	SessionID int64   `json:"session_id"`
	Token     string  `json:"token"`
	Code      string  `json:"code"`
	Lat       float64 `json:"lat"`
	Lng       float64 `json:"lng"`
	AccuracyM float64 `json:"accuracy_m"`
	IsMock    bool    `json:"is_mock"`
	DeviceID  string  `json:"device_id"`
}

type deviceRequest struct {
	DeviceID string `json:"device_id"`
	Reason   string `json:"reason"`
}

type phoneBody struct {
	Registered    bool   `json:"registered"`
	ThisPhone     bool   `json:"this_phone"`
	BoundAt       string `json:"bound_at,omitempty"`
	RequestStatus string `json:"request_status,omitempty"`
	RequestedAt   string `json:"requested_at,omitempty"`
	DeciderName   string `json:"decider_name,omitempty"`
	DecisionNote  string `json:"decision_note,omitempty"`
	Changes       int    `json:"changes"`
	Limit         int    `json:"limit"`
}

type openSessionBody struct {
	ID          int64  `json:"id"`
	CourseCode  string `json:"course_code"`
	CourseTitle string `json:"course_title"`
	TeacherName string `json:"teacher_name"`
	Batch       string `json:"batch"`
	Section     string `json:"section"`
	EndsAt      string `json:"ends_at"`
	Present     bool   `json:"present"`
}

type studentOpenResponse struct {
	Session *openSessionBody `json:"session,omitempty"`
	Phone   phoneBody        `json:"phone"`
	Service string           `json:"service_uuid"`
	Message string           `json:"message,omitempty"`
}

type myCourseBody struct {
	CourseID    int64  `json:"course_id"`
	CourseCode  string `json:"course_code"`
	CourseTitle string `json:"course_title"`
	TeacherName string `json:"teacher_name"`
	Held        int    `json:"held"`
	Present     int    `json:"present"`
	Percentage  int    `json:"percentage"`
}

type myClassBody struct {
	SessionID   int64  `json:"session_id"`
	CourseID    int64  `json:"course_id"`
	CourseCode  string `json:"course_code"`
	CourseTitle string `json:"course_title"`
	TeacherName string `json:"teacher_name"`
	Date        string `json:"date"`
	Status      string `json:"status"`
	Method      string `json:"method"`
	MarkedAt    string `json:"marked_at,omitempty"`
	Note        string `json:"note,omitempty"`
}

type myAttendanceResponse struct {
	Courses []myCourseBody `json:"courses"`
	Classes []myClassBody  `json:"classes"`
}

func (s *StudentAttendance) profile(w http.ResponseWriter, r *http.Request) (store.StudentProfile, bool) {
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
		log.Printf("handler: could not load the student of an attendance request: %v", err)
		httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
		return store.StudentProfile{}, false
	}
	return profile, true
}

func (s *StudentAttendance) phone(ctx context.Context, profile store.StudentProfile,
	deviceID string) phoneBody {
	body := phoneBody{Limit: attendance.SemesterChangeLimit}
	binding, err := s.students.DeviceBinding(ctx, profile.UserID)
	if err == nil {
		body.Registered = true
		body.BoundAt = timestamp(binding.BoundAt)
		body.ThisPhone = deviceID != "" && binding.DeviceID == deviceID
	} else if !errors.Is(err, store.ErrNotFound) {
		log.Printf("handler: could not read the phone of a student: %v", err)
	}
	request, err := s.students.LatestDeviceRequest(ctx, profile.UserID)
	if err == nil {
		body.RequestStatus = request.Status
		body.RequestedAt = timestamp(request.CreatedAt)
		body.DeciderName = request.DeciderName
		body.DecisionNote = request.DecisionNote
	} else if !errors.Is(err, store.ErrNotFound) {
		log.Printf("handler: could not read the phone request of a student: %v", err)
	}
	semester, err := s.students.RunningSemester(ctx, profile.Batch)
	if err == nil {
		if changes, err := s.students.ApprovedDeviceChanges(ctx, profile.UserID, semester); err == nil {
			body.Changes = changes
		}
	}
	return body
}

func (s *StudentAttendance) Open() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profile, ok := s.profile(w, r)
		if !ok {
			return
		}
		deviceID := strings.TrimSpace(r.URL.Query().Get("device_id"))
		response := studentOpenResponse{
			Phone:   s.phone(r.Context(), profile, deviceID),
			Service: attendance.ServiceUUID,
		}
		if !middleware.FromAndroidApp(r) {
			response.Message = attendance.MessageWebsiteOnly
			httpx.WriteJSON(w, http.StatusOK, response)
			return
		}
		session, err := s.students.OpenSessionForStudent(r.Context(), profile.StudentID)
		if errors.Is(err, store.ErrNotFound) {
			response.Message = MessageNoSession
			httpx.WriteJSON(w, http.StatusOK, response)
			return
		}
		if err != nil {
			log.Printf("handler: could not find the open attendance of a student: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		records, err := s.students.StudentAttendanceSessions(r.Context(), profile.StudentID)
		if err != nil {
			log.Printf("handler: could not read the attendance rows of a student: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		marked := false
		for _, record := range records {
			if record.SessionID == session.ID {
				marked = true
				break
			}
		}
		response.Session = &openSessionBody{
			ID:          session.ID,
			CourseCode:  session.CourseCode,
			CourseTitle: session.CourseTitle,
			TeacherName: session.TeacherName,
			Batch:       session.Batch,
			Section:     session.Section,
			EndsAt:      timestamp(session.EndsAt),
			Present:     marked,
		}
		if !response.Phone.Registered {
			response.Message = ""
		} else if !response.Phone.ThisPhone && deviceID != "" {
			response.Message = attendance.MessageWrongPhone
		}
		httpx.WriteJSON(w, http.StatusOK, response)
	}
}

func (s *StudentAttendance) Mark() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profile, ok := s.profile(w, r)
		if !ok {
			return
		}
		if !middleware.FromAndroidApp(r) {
			httpx.WriteError(w, http.StatusForbidden, attendance.MessageWebsiteOnly)
			return
		}
		var request markRequest
		if !httpx.DecodeJSON(w, r, &request) {
			return
		}
		deviceID := strings.TrimSpace(request.DeviceID)
		if deviceID == "" {
			httpx.WriteError(w, http.StatusBadRequest, MessageDeviceIDRequired)
			return
		}
		binding, err := s.students.DeviceBinding(r.Context(), profile.UserID)
		if errors.Is(err, store.ErrNotFound) {
			httpx.WriteError(w, http.StatusForbidden, attendance.MessageWrongPhone)
			return
		}
		if err != nil {
			log.Printf("handler: could not read the phone of a marking student: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		if binding.DeviceID != deviceID {
			refuse(w, profile.StudentID, request.SessionID, http.StatusForbidden,
				attendance.MessageWrongPhone, "another phone")
			return
		}
		session, err := s.students.AttendanceSession(r.Context(), request.SessionID)
		if errors.Is(err, store.ErrNotFound) {
			httpx.WriteError(w, http.StatusNotFound, MessageSessionGone)
			return
		}
		if err != nil {
			log.Printf("handler: could not load the attendance session of a mark: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		now := s.cfg.Now()
		if session.Status != attendance.StatusOpen || !session.EndsAt.After(now) {
			refuse(w, profile.StudentID, session.ID, http.StatusConflict,
				attendance.MessageSessionClosed, "the session is closed")
			return
		}
		listed, err := s.students.OnRoster(r.Context(), session.ID, profile.StudentID)
		if err != nil {
			log.Printf("handler: could not check the roster of a mark: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		if !listed {
			refuse(w, profile.StudentID, session.ID, http.StatusForbidden,
				attendance.MessageNotOnRoster, "not on the roster")
			return
		}
		method := attendance.MethodApp
		switch {
		case strings.TrimSpace(request.Token) != "":
			if !attendance.TokenMatches(session.TokenKey, session.ID, now, request.Token) {
				refuse(w, profile.StudentID, session.ID, http.StatusForbidden,
					attendance.MessageWrongToken, "a token from no current slice")
				return
			}
		case strings.TrimSpace(request.Code) != "":
			if !attendance.CodeMatches(session.TokenKey, session.ID, now, request.Code) {
				refuse(w, profile.StudentID, session.ID, http.StatusForbidden,
					attendance.MessageWrongToken, "a code from no current slice")
				return
			}
			method = attendance.MethodCode
		default:
			refuse(w, profile.StudentID, session.ID, http.StatusBadRequest,
				MessageMarkNeedsProof, "neither a token nor a code")
			return
		}
		distance, refusal := attendance.CheckPosition(
			attendance.Position{
				Lat:       request.Lat,
				Lng:       request.Lng,
				AccuracyM: request.AccuracyM,
				IsMock:    request.IsMock,
			},
			attendance.Anchor{
				Lat:       session.TeacherLat,
				Lng:       session.TeacherLng,
				UpdatedAt: session.LocationAt,
				Known:     session.HasLocation,
			},
			now,
		)
		if refusal != "" {
			refuse(w, profile.StudentID, session.ID, http.StatusForbidden, refusal,
				fmt.Sprintf("position %.0f m away, accuracy %.0f m, mock %t",
					distance, request.AccuracyM, request.IsMock))
			return
		}
		err = s.students.MarkAttendance(r.Context(), store.MarkInput{
			SessionID: session.ID,
			StudentID: profile.StudentID,
			Method:    method,
			DeviceID:  deviceID,
			Lat:       request.Lat,
			Lng:       request.Lng,
			AccuracyM: request.AccuracyM,
			DistanceM: distance,
		})
		switch {
		case errors.Is(err, store.ErrAlreadyMarked):
			refuse(w, profile.StudentID, session.ID, http.StatusConflict,
				attendance.MessageAlreadyMarked, "already marked")
			return
		case errors.Is(err, store.ErrDeviceUsed):
			refuse(w, profile.StudentID, session.ID, http.StatusConflict,
				attendance.MessagePhoneUsed, "this phone already marked somebody")
			return
		case errors.Is(err, store.ErrAttendanceClosed):
			refuse(w, profile.StudentID, session.ID, http.StatusConflict,
				attendance.MessageSessionClosed, "the session closed meanwhile")
			return
		case err != nil:
			log.Printf("handler: could not save an attendance mark: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		if err := s.students.TouchDevice(r.Context(), profile.UserID, deviceID); err != nil {
			log.Printf("handler: could not touch the phone of a marking student: %v", err)
		}
		httpx.WriteJSON(w, http.StatusCreated, messageResponse{Message: MessageMarkSaved})
	}
}

func (s *StudentAttendance) Mine() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profile, ok := s.profile(w, r)
		if !ok {
			return
		}
		courses, err := s.students.StudentCourseAttendance(r.Context(), profile.StudentID)
		if err != nil {
			log.Printf("handler: could not list the attendance of a student: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		classes, err := s.students.StudentAttendanceSessions(r.Context(), profile.StudentID)
		if err != nil {
			log.Printf("handler: could not list the classes of a student: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		response := myAttendanceResponse{
			Courses: make([]myCourseBody, 0, len(courses)),
			Classes: make([]myClassBody, 0, len(classes)),
		}
		for _, course := range courses {
			response.Courses = append(response.Courses, myCourseBody{
				CourseID:    course.CourseID,
				CourseCode:  course.CourseCode,
				CourseTitle: course.CourseTitle,
				TeacherName: course.TeacherName,
				Held:        course.Held,
				Present:     course.Present,
				Percentage:  percentText(course.Present, course.Held),
			})
		}
		for _, class := range classes {
			body := myClassBody{
				SessionID:   class.SessionID,
				CourseID:    class.CourseID,
				CourseCode:  class.CourseCode,
				CourseTitle: class.CourseTitle,
				TeacherName: class.TeacherName,
				Date:        validate.DateString(class.StartedAt),
				Status:      class.Status,
				Method:      class.Method,
				Note:        class.Note,
			}
			if class.HasMark {
				body.MarkedAt = timestamp(class.MarkedAt)
			}
			response.Classes = append(response.Classes, body)
		}
		httpx.WriteJSON(w, http.StatusOK, response)
	}
}

func (s *StudentAttendance) RegisterPhone() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profile, ok := s.profile(w, r)
		if !ok {
			return
		}
		if !middleware.FromAndroidApp(r) {
			httpx.WriteError(w, http.StatusForbidden, attendance.MessageWebsiteOnly)
			return
		}
		var request deviceRequest
		if !httpx.DecodeJSON(w, r, &request) {
			return
		}
		deviceID := strings.TrimSpace(request.DeviceID)
		if deviceID == "" {
			httpx.WriteError(w, http.StatusBadRequest, MessageDeviceIDRequired)
			return
		}
		binding, fresh, err := s.students.BindDevice(r.Context(), profile.UserID, deviceID)
		if err != nil {
			log.Printf("handler: could not register the phone of a student: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		message := MessagePhoneSame
		if fresh {
			message = MessagePhoneRegistered
		} else if binding.DeviceID != deviceID {
			message = attendance.MessageWrongPhone
		} else if err := s.students.TouchDevice(r.Context(), profile.UserID, deviceID); err != nil {
			log.Printf("handler: could not touch the phone of a student: %v", err)
		}
		httpx.WriteJSON(w, http.StatusOK, struct {
			Message string    `json:"message"`
			Phone   phoneBody `json:"phone"`
		}{Message: message, Phone: s.phone(r.Context(), profile, deviceID)})
	}
}

func (s *StudentAttendance) RequestPhone() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profile, ok := s.profile(w, r)
		if !ok {
			return
		}
		if !middleware.FromAndroidApp(r) {
			httpx.WriteError(w, http.StatusForbidden, attendance.MessageWebsiteOnly)
			return
		}
		var request deviceRequest
		if !httpx.DecodeJSON(w, r, &request) {
			return
		}
		deviceID := strings.TrimSpace(request.DeviceID)
		if deviceID == "" {
			httpx.WriteError(w, http.StatusBadRequest, MessageDeviceIDRequired)
			return
		}
		reason := strings.TrimSpace(request.Reason)
		if reason == "" {
			httpx.WriteError(w, http.StatusBadRequest, MessageReasonNeeded)
			return
		}
		if len(reason) > maxReasonLength {
			httpx.WriteError(w, http.StatusBadRequest, MessageReasonNeeded)
			return
		}
		semester, err := s.students.RunningSemester(r.Context(), profile.Batch)
		if err != nil && !errors.Is(err, store.ErrNotFound) {
			log.Printf("handler: could not read the running semester of a phone request: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		saved, err := s.students.CreateDeviceChangeRequest(r.Context(), profile.UserID,
			deviceID, reason, semester)
		switch {
		case errors.Is(err, store.ErrRequestPending):
			httpx.WriteError(w, http.StatusConflict, MessageRequestWaiting)
			return
		case errors.Is(err, store.ErrSameDeviceBound):
			httpx.WriteError(w, http.StatusConflict, MessagePhoneSame)
			return
		case errors.Is(err, store.ErrDeviceTaken):
			httpx.WriteError(w, http.StatusConflict, MessagePhoneTaken)
			return
		case err != nil:
			log.Printf("handler: could not save a phone request: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		s.tellTeachers(r.Context(), profile, saved)
		httpx.WriteJSON(w, http.StatusCreated, struct {
			Message string    `json:"message"`
			Phone   phoneBody `json:"phone"`
		}{Message: MessagePhoneRequested, Phone: s.phone(r.Context(), profile, deviceID)})
	}
}

func refuse(w http.ResponseWriter, studentID, sessionID int64, status int, message, why string) {
	log.Printf("handler: attendance %d refused the student %d: %s", sessionID, studentID, why)
	httpx.WriteError(w, status, message)
}

func (s *StudentAttendance) tellTeachers(ctx context.Context, profile store.StudentProfile,
	request store.DeviceChangeRequest) {
	account, err := s.students.FindAccount(ctx, profile.UserID)
	if err != nil {
		log.Printf("handler: could not read the account behind a phone request: %v", err)
		return
	}
	teachers, err := s.students.ListBatchTeachers(ctx, profile.Batch, profile.Section)
	if err != nil {
		log.Printf("handler: could not list the teachers of a phone request: %v", err)
		return
	}
	title := account.Name + " (" + account.Identifier + ")" + titlePhoneRequest
	link := notify.Ref(notify.LinkPhoneRequests, request.ID)
	for _, userID := range teachers {
		if err := s.alerts.NotifyUser(ctx, userID, notify.TypeRequest, title, request.Reason, link); err != nil {
			log.Printf("handler: could not tell the teacher %d about a phone request: %v", userID, err)
		}
	}
}

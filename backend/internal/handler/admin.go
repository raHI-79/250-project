package handler

import (
	"context"
	"errors"
	"github.com/raHI-79/250-project/backend/internal/httpx"
	"github.com/raHI-79/250-project/backend/internal/middleware"
	"github.com/raHI-79/250-project/backend/internal/store"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	MessageInvalidID              = "The id in the path must be a positive number."
	MessageStudentNotFound        = "No student found with this ID."
	MessageRegistrationNotPending = "This registration has already been decided."
	MessageRegistrationApproved   = "Registration approved."
	MessageRegistrationRejected   = "Registration rejected."
	MessageStudentRemoved         = "Student removed."
	MessageAlreadyRemoved         = "This account is already removed."
	MessageCRCreated              = "Class representative added."
	MessageCRNotFound             = "No class representative found with this ID."
	MessageNotCR                  = "This student is not a class representative."
	MessageCRDemoted              = "Class representative demoted."
	MessageCRDeactivated          = "Class representative deactivated."
	MessageTeacherCreated         = "Teacher added."
	MessageTeacherNotFound        = "No teacher found with this ID."
	MessageTeacherDeactivated     = "Teacher deactivated."
	MessageDuplicateTeacher       = "An account with this teacher ID or email already exists."
	MessageInvalidStatus          = "status must be one of active, pending, rejected, removed, all."
	MessageInvalidLimit           = "limit must be a number between 1 and 200."
	MessageInvalidOffset          = "offset must be a number of 0 or more."
)

type AdminStore interface {
	ListPendingRegistrations(ctx context.Context) ([]store.PendingRegistration, error)
	SetRegistrationStatus(ctx context.Context, userID int64, status string) error
	ListStudents(ctx context.Context, filter store.StudentFilter) ([]store.StudentRow, int, error)
	RemoveStudent(ctx context.Context, userID int64) error
	CreateCR(ctx context.Context, cr store.NewCR) (int64, error)
	ListCRs(ctx context.Context, filter store.CRFilter) ([]store.StudentRow, error)
	DemoteCR(ctx context.Context, userID int64) error
	DeactivateCR(ctx context.Context, userID int64) error
	CreateTeacher(ctx context.Context, teacher store.NewTeacher) (int64, error)
	ListTeachers(ctx context.Context, filter store.TeacherFilter) ([]store.TeacherRow, error)
	DeactivateTeacher(ctx context.Context, userID int64) error
	Stats(ctx context.Context) (store.Stats, error)
}

type Admin struct {
	people AdminStore
}

func NewAdmin(people AdminStore) *Admin {
	return &Admin{people: people}
}

type registrationItem struct {
	ID             int64  `json:"id"`
	FullName       string `json:"full_name"`
	RegNo          string `json:"reg_no"`
	Batch          string `json:"batch"`
	Section        string `json:"section"`
	Email          string `json:"email"`
	Phone          string `json:"phone"`
	BloodGroup     string `json:"blood_group"`
	HasDropCourse  bool   `json:"has_drop_course"`
	DropCourseCode string `json:"drop_course_code"`
	CreatedAt      string `json:"created_at"`
}

type registrationsResponse struct {
	Items []registrationItem `json:"items"`
}

func (a *Admin) Registrations() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		registrations, err := a.people.ListPendingRegistrations(r.Context())
		if err != nil {
			log.Printf("handler: could not list the pending registrations: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		items := make([]registrationItem, 0, len(registrations))
		for _, registration := range registrations {
			items = append(items, registrationItem{
				ID:             registration.ID,
				FullName:       registration.FullName,
				RegNo:          registration.RegNo,
				Batch:          registration.Batch,
				Section:        registration.Section,
				Email:          registration.Email,
				Phone:          registration.Phone,
				BloodGroup:     registration.BloodGroup,
				HasDropCourse:  registration.HasDropCourse,
				DropCourseCode: registration.DropCourseCode,
				CreatedAt:      timestamp(registration.CreatedAt),
			})
		}
		httpx.WriteJSON(w, http.StatusOK, registrationsResponse{Items: items})
	}
}

func (a *Admin) ApproveRegistration() http.HandlerFunc {
	return a.decideRegistration(store.StatusActive, MessageRegistrationApproved)
}

func (a *Admin) RejectRegistration() http.HandlerFunc {
	return a.decideRegistration(store.StatusRejected, MessageRegistrationRejected)
}

func (a *Admin) decideRegistration(status, message string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := pathID(w, r)
		if !ok {
			return
		}
		err := a.people.SetRegistrationStatus(r.Context(), userID, status)
		switch {
		case errors.Is(err, store.ErrNotFound):
			httpx.WriteError(w, http.StatusNotFound, MessageStudentNotFound)
		case errors.Is(err, store.ErrNotPending):
			httpx.WriteError(w, http.StatusConflict, MessageRegistrationNotPending)
		case err != nil:
			log.Printf("handler: could not decide a registration: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
		default:
			httpx.WriteJSON(w, http.StatusOK, messageResponse{Message: message})
		}
	}
}

func pathID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	userID, err := strconv.ParseInt(strings.TrimSpace(r.PathValue("id")), 10, 64)
	if err != nil || userID <= 0 {
		httpx.WriteError(w, http.StatusBadRequest, MessageInvalidID)
		return 0, false
	}
	return userID, true
}

func timestamp(at time.Time) string {
	return at.Format(time.RFC3339)
}

package handler

import (
	"context"
	"errors"
	"github.com/raHI-79/250-project/backend/internal/config"
	"github.com/raHI-79/250-project/backend/internal/httpx"
	"github.com/raHI-79/250-project/backend/internal/mailer"
	"github.com/raHI-79/250-project/backend/internal/middleware"
	"github.com/raHI-79/250-project/backend/internal/store"
	"github.com/raHI-79/250-project/backend/internal/validate"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	MessageInvalidID              = "The id in the path must be a positive number."
	MessageInvalidUserID          = "user_id must be a positive number."
	MessageStudentNotFound        = "No student found with this ID."
	MessageActiveStudentNotFound  = "No active student found with this ID."
	MessageRegistrationNotPending = "This registration has already been decided."
	MessageRegistrationApproved   = "Registration approved."
	MessageRegistrationRejected   = "Registration rejected."
	MessageStudentRemoved         = "Student removed."
	MessageAlreadyRemoved         = "This account is already removed."
	MessageAlreadyCR              = "This student is already a class representative."
	MessageCRPromoted             = "Class representative added."
	MessageCRNotFound             = "No class representative found with this ID."
	MessageNotCR                  = "This student is not a class representative."
	MessageCRDemoted              = "Class representative demoted."
	MessageCRDeactivated          = "Class representative deactivated."
	MessageTeacherInvited         = "Teacher invited. They finish the account from the invitation email."
	MessageTeacherUpdated         = "Teacher updated."
	MessageInviteResent           = "Invitation sent again. The older link no longer works."
	MessageTeacherNotInvited      = "This teacher has already activated their account."
	MessageTeacherNotFound        = "No teacher found with this ID."
	MessageTeacherDeactivated     = "Teacher deactivated."
	MessageDuplicateTeacher       = "An account with this teacher ID or email already exists."
	MessageInvalidStatus          = "status must be one of active, pending, rejected, removed, unverified, invited, all."
	MessageInvalidLimit           = "limit must be a number between 1 and 200."
	MessageInvalidOffset          = "offset must be a number of 0 or more."
	fieldJoiningDate              = "joining_date"
)

type AdminStore interface {
	ListPendingRegistrations(ctx context.Context) ([]store.PendingRegistration, error)
	SetRegistrationStatus(ctx context.Context, userID int64, status string) error
	ListStudents(ctx context.Context, filter store.StudentFilter) ([]store.StudentRow, int, error)
	RemoveStudent(ctx context.Context, userID int64) error
	PromoteCR(ctx context.Context, userID int64) (store.PromotedCR, error)
	ListCRs(ctx context.Context, filter store.CRFilter) ([]store.StudentRow, error)
	DemoteCR(ctx context.Context, userID int64) error
	DeactivateCR(ctx context.Context, userID int64) error
	CreateTeacher(ctx context.Context, teacher store.NewTeacher) (int64, error)
	UpdateTeacher(ctx context.Context, userID int64, update store.TeacherUpdate) error
	ResetTeacherInvite(ctx context.Context, userID int64, inviteHash string, expiresAt time.Time) (store.TeacherInvite, error)
	ListTeachers(ctx context.Context, filter store.TeacherFilter) ([]store.TeacherRow, error)
	DeactivateTeacher(ctx context.Context, userID int64) error
	Stats(ctx context.Context) (store.Stats, error)
}

type Admin struct {
	cfg    *config.Config
	people AdminStore
	alerts Notifier
	mail   mailer.Sender
	photos PhotoStore
}

func NewAdmin(cfg *config.Config, people AdminStore, alerts Notifier, mail mailer.Sender, photos PhotoStore) *Admin {
	return &Admin{cfg: cfg, people: people, alerts: alerts, mail: mail, photos: photos}
}

type registrationItem struct {
	ID          int64           `json:"id"`
	UserID      int64           `json:"user_id"`
	FullName    string          `json:"full_name"`
	RegNo       string          `json:"reg_no"`
	Batch       string          `json:"batch"`
	Section     string          `json:"section"`
	Email       string          `json:"email"`
	Phone       string          `json:"phone"`
	BloodGroup  string          `json:"blood_group"`
	DateOfBirth string          `json:"date_of_birth,omitempty"`
	PhotoURL    string          `json:"photo_url,omitempty"`
	DropCourses []profileCourse `json:"drop_courses"`
	CreatedAt   string          `json:"created_at"`
}

type registrationsResponse struct {
	Items []registrationItem `json:"items"`
}

type updateTeacherRequest struct {
	Name        *string `json:"name"`
	TeacherID   *string `json:"teacher_id"`
	Designation *string `json:"designation"`
	Email       *string `json:"email"`
	DateOfBirth *string `json:"date_of_birth"`
	BloodGroup  *string `json:"blood_group"`
	JoiningDate *string `json:"joining_date"`
	OfficeRoom  *string `json:"office_room"`
}

func (request updateTeacherRequest) toUpdate(cfg *config.Config) (store.TeacherUpdate, error) {
	var update store.TeacherUpdate
	texts := []struct {
		source *string
		target **string
		label  string
		check  func(string, string) (string, error)
	}{
		{source: request.Name, target: &update.Name, label: "Name", check: validate.Required},
		{source: request.TeacherID, target: &update.TeacherID, label: "Teacher ID", check: validate.TeacherID},
		{source: request.Designation, target: &update.Designation, label: "Designation", check: validate.Required},
		{source: request.BloodGroup, target: &update.BloodGroup, label: "Blood group", check: validate.BloodGroup},
		{source: request.OfficeRoom, target: &update.OfficeRoom, label: "Office room", check: validate.Required},
	}
	for _, text := range texts {
		if text.source == nil {
			continue
		}
		value, err := text.check(text.label, *text.source)
		if err != nil {
			return store.TeacherUpdate{}, err
		}
		*text.target = &value
	}
	if request.Email != nil {
		email, err := validate.EmailInDomain("Email", *request.Email, cfg.TeacherDomain)
		if err != nil {
			return store.TeacherUpdate{}, err
		}
		update.Email = &email
	}
	if request.DateOfBirth != nil {
		birthDate, err := validate.BirthDate("Date of birth", *request.DateOfBirth, cfg.Now())
		if err != nil {
			return store.TeacherUpdate{}, err
		}
		update.DateOfBirth = &birthDate
	}
	if request.JoiningDate != nil {
		joiningDate, err := validate.PlainDate("Joining date", *request.JoiningDate)
		if err != nil {
			return store.TeacherUpdate{}, err
		}
		update.JoiningDate = &joiningDate
	}
	if update == (store.TeacherUpdate{}) {
		return store.TeacherUpdate{}, errors.New(MessageNothingToUpdate)
	}
	return update, nil
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
				ID:          registration.ID,
				UserID:      registration.ID,
				FullName:    registration.FullName,
				RegNo:       registration.RegNo,
				Batch:       registration.Batch,
				Section:     registration.Section,
				Email:       registration.Email,
				Phone:       registration.Phone,
				BloodGroup:  registration.BloodGroup,
				DateOfBirth: registration.DateOfBirth,
				PhotoURL:    registration.PhotoPath,
				DropCourses: profileCourses(registration.DropCourses),
				CreatedAt:   timestamp(registration.CreatedAt),
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

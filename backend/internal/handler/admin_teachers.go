package handler

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"github.com/raHI-79/250-project/backend/internal/httpx"
	"github.com/raHI-79/250-project/backend/internal/middleware"
	"github.com/raHI-79/250-project/backend/internal/store"
	"github.com/raHI-79/250-project/backend/internal/validate"
	"log"
	"net/http"
	"strings"
	"time"
)

const (
	InviteEmailSubject = "Your teacher account invitation"
	inviteTokenBytes   = 32
	inviteLifetime     = 72 * time.Hour
)

type teacherItem struct {
	ID          int64  `json:"id"`
	UserID      int64  `json:"user_id"`
	TeacherID   string `json:"teacher_id"`
	Name        string `json:"name"`
	Designation string `json:"designation"`
	Email       string `json:"email"`
	Phone       string `json:"phone"`
	BloodGroup  string `json:"blood_group,omitempty"`
	DateOfBirth string `json:"date_of_birth,omitempty"`
	OfficeRoom  string `json:"office_room,omitempty"`
	JoiningDate string `json:"joining_date,omitempty"`
	Bio         string `json:"bio,omitempty"`
	PhotoURL    string `json:"photo_url,omitempty"`
	Status      string `json:"status"`
	CreatedAt   string `json:"created_at"`
}

type teachersResponse struct {
	Items []teacherItem `json:"items"`
}

type inviteCreatedResponse struct {
	ID         int64  `json:"id"`
	Message    string `json:"message"`
	InviteLink string `json:"invite_link,omitempty"`
	ExpiresAt  string `json:"expires_at"`
}

func (a *Admin) CreateTeacher() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := a.photos.Parse(r); err != nil {
			status, message := photoError(err)
			httpx.WriteError(w, status, message)
			return
		}
		teacher, err := a.teacherFrom(r)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		photoPath, err := a.photos.Save(r)
		if err != nil {
			status, message := photoError(err)
			httpx.WriteError(w, status, message)
			return
		}
		teacher.PhotoPath = photoPath
		token, hash, err := newInviteToken()
		if err != nil {
			a.removePhoto(photoPath)
			log.Printf("handler: could not generate an invitation token: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		expiresAt := a.cfg.Now().Add(inviteLifetime)
		teacher.InviteHash = hash
		teacher.InviteExpiresAt = expiresAt
		userID, err := a.people.CreateTeacher(r.Context(), teacher)
		if err != nil {
			a.removePhoto(photoPath)
			if errors.Is(err, store.ErrDuplicateAccount) {
				httpx.WriteError(w, http.StatusConflict, MessageDuplicateTeacher)
				return
			}
			log.Printf("handler: could not create a teacher: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		body := inviteCreatedResponse{
			ID:        userID,
			Message:   MessageTeacherInvited,
			ExpiresAt: expiresAt.Format(time.RFC3339),
		}
		a.sendInvite(r, teacher.Email, token, &body)
		httpx.WriteJSON(w, http.StatusCreated, body)
	}
}

func (a *Admin) teacherFrom(r *http.Request) (store.NewTeacher, error) {
	var teacher store.NewTeacher
	fields := []struct {
		target *string
		field  string
		label  string
		check  func(string, string) (string, error)
	}{
		{target: &teacher.Name, field: "name", label: "Name", check: validate.Required},
		{target: &teacher.TeacherID, field: fieldTeacherID, label: "Teacher ID", check: validate.TeacherID},
		{target: &teacher.Designation, field: "designation", label: "Designation", check: validate.Required},
		{target: &teacher.Phone, field: fieldPhone, label: "Phone", check: validate.Phone},
		{target: &teacher.BloodGroup, field: fieldBloodGroup, label: "Blood group", check: validate.BloodGroup},
		{target: &teacher.OfficeRoom, field: fieldOfficeRoom, label: "Office room", check: validate.Required},
	}
	for _, field := range fields {
		value, err := field.check(field.label, r.FormValue(field.field))
		if err != nil {
			return store.NewTeacher{}, err
		}
		*field.target = value
	}
	email, err := validate.EmailInDomain("Email", r.FormValue(fieldEmail), a.cfg.TeacherDomain)
	if err != nil {
		return store.NewTeacher{}, err
	}
	teacher.Email = email
	birthDate, err := validate.BirthDate("Date of birth", r.FormValue(fieldDateOfBirth), a.cfg.Now())
	if err != nil {
		return store.NewTeacher{}, err
	}
	teacher.DateOfBirth = birthDate
	if raw := strings.TrimSpace(r.FormValue(fieldJoiningDate)); raw != "" {
		joiningDate, err := validate.PlainDate("Joining date", raw)
		if err != nil {
			return store.NewTeacher{}, err
		}
		teacher.JoiningDate = joiningDate
	}
	return teacher, nil
}

func (a *Admin) UpdateTeacher() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := pathID(w, r)
		if !ok {
			return
		}
		var request updateTeacherRequest
		if !httpx.DecodeJSON(w, r, &request) {
			return
		}
		update, err := request.toUpdate(a.cfg)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		err = a.people.UpdateTeacher(r.Context(), userID, update)
		switch {
		case errors.Is(err, store.ErrNotFound):
			httpx.WriteError(w, http.StatusNotFound, MessageTeacherNotFound)
		case errors.Is(err, store.ErrDuplicateAccount):
			httpx.WriteError(w, http.StatusConflict, MessageDuplicateTeacher)
		case err != nil:
			log.Printf("handler: could not update a teacher: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
		default:
			httpx.WriteJSON(w, http.StatusOK, messageResponse{Message: MessageTeacherUpdated})
		}
	}
}

func (a *Admin) ResendInvite() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := pathID(w, r)
		if !ok {
			return
		}
		token, hash, err := newInviteToken()
		if err != nil {
			log.Printf("handler: could not generate an invitation token: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		expiresAt := a.cfg.Now().Add(inviteLifetime)
		invite, err := a.people.ResetTeacherInvite(r.Context(), userID, hash, expiresAt)
		switch {
		case errors.Is(err, store.ErrNotFound):
			httpx.WriteError(w, http.StatusNotFound, MessageTeacherNotFound)
			return
		case errors.Is(err, store.ErrNotInvited):
			httpx.WriteError(w, http.StatusConflict, MessageTeacherNotInvited)
			return
		case err != nil:
			log.Printf("handler: could not refresh a teacher invitation: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		body := inviteCreatedResponse{
			ID:        invite.UserID,
			Message:   MessageInviteResent,
			ExpiresAt: expiresAt.Format(time.RFC3339),
		}
		a.sendInvite(r, invite.Email, token, &body)
		httpx.WriteJSON(w, http.StatusOK, body)
	}
}

func (a *Admin) sendInvite(r *http.Request, email, token string, body *inviteCreatedResponse) {
	link := a.cfg.FrontendOrigin + "/teacher/activate?token=" + token
	if a.mailerReady() {
		if err := a.mail.Send(r.Context(), email, InviteEmailSubject, inviteEmailBody(link)); err != nil {
			log.Printf("handler: could not email a teacher invitation: %v", err)
		}
		return
	}
	if !a.cfg.IsProduction() {
		body.InviteLink = link
	}
}

func (a *Admin) mailerReady() bool {
	return a.mail != nil && a.mail.Configured()
}

func (a *Admin) removePhoto(photoPath string) {
	if photoPath == "" {
		return
	}
	if err := a.photos.Delete(photoPath); err != nil {
		log.Printf("handler: could not delete a profile photo: %v", err)
	}
}

func (a *Admin) Teachers() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		status, err := statusFilter(query.Get("status"))
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		filter := store.TeacherFilter{Search: strings.TrimSpace(query.Get("search")), Status: status}
		teachers, err := a.people.ListTeachers(r.Context(), filter)
		if err != nil {
			log.Printf("handler: could not list the teachers: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		items := make([]teacherItem, 0, len(teachers))
		for _, teacher := range teachers {
			items = append(items, teacherItem{
				ID:          teacher.ID,
				UserID:      teacher.ID,
				TeacherID:   teacher.TeacherID,
				Name:        teacher.Name,
				Designation: teacher.Designation,
				Email:       teacher.Email,
				Phone:       teacher.Phone,
				BloodGroup:  teacher.BloodGroup,
				DateOfBirth: teacher.DateOfBirth,
				OfficeRoom:  teacher.OfficeRoom,
				JoiningDate: teacher.JoiningDate,
				Bio:         teacher.Bio,
				PhotoURL:    teacher.PhotoPath,
				Status:      teacher.Status,
				CreatedAt:   timestamp(teacher.CreatedAt),
			})
		}
		httpx.WriteJSON(w, http.StatusOK, teachersResponse{Items: items})
	}
}

func (a *Admin) DeactivateTeacher() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := pathID(w, r)
		if !ok {
			return
		}
		err := a.people.DeactivateTeacher(r.Context(), userID)
		switch {
		case errors.Is(err, store.ErrNotFound):
			httpx.WriteError(w, http.StatusNotFound, MessageTeacherNotFound)
		case errors.Is(err, store.ErrAlreadyRemoved):
			httpx.WriteError(w, http.StatusConflict, MessageAlreadyRemoved)
		case err != nil:
			log.Printf("handler: could not deactivate a teacher: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
		default:
			httpx.WriteJSON(w, http.StatusOK, messageResponse{Message: MessageTeacherDeactivated})
		}
	}
}

func inviteEmailBody(link string) string {
	return "You have been invited to the SUST CSE coordination system.\n\n" +
		"Open this link within 72 hours to set your password and activate your account:\n" + link
}

func newInviteToken() (string, string, error) {
	raw := make([]byte, inviteTokenBytes)
	if _, err := rand.Read(raw); err != nil {
		return "", "", err
	}
	token := hex.EncodeToString(raw)
	return token, hashResetSecret(token), nil
}

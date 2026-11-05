package handler

import (
	"context"
	"errors"
	"github.com/raHI-79/250-project/backend/internal/auth"
	"github.com/raHI-79/250-project/backend/internal/config"
	"github.com/raHI-79/250-project/backend/internal/httpx"
	"github.com/raHI-79/250-project/backend/internal/mailer"
	"github.com/raHI-79/250-project/backend/internal/middleware"
	"github.com/raHI-79/250-project/backend/internal/store"
	"github.com/raHI-79/250-project/backend/internal/validate"
	"log"
	"net/http"
	"strings"
	"time"
)

const (
	MessageAdminInvited     = "Invitation sent. The link works for 72 hours."
	MessageAdminInviteGone  = "This invitation is no longer waiting."
	MessageAdminNotFound    = "Office admin not found."
	MessageAdminNotAdmin    = "This account is not an office admin."
	MessageAdminProtected   = "The super admin account cannot be changed here."
	MessageAdminSuspended   = "Office admin suspended."
	MessageAdminRestored    = "Office admin restored."
	MessageAdminRemoved     = "Office admin removed."
	MessageAdminConfirm     = "Type the username to confirm."
	MessageAdminDuplicate   = "An account with this email already exists."
	MessageAdminInviteSent  = "A fresh invitation is on its way."
	MessageInviteCancelled  = "Invitation cancelled."
	MessageTeacherNoOffice  = "Only an active teacher can be given office access."
	MessageOfficeGranted    = "Office access granted."
	MessageOfficeRevoked    = "Office access revoked."
	AdminInviteEmailSubject = "Your office admin invitation"
	adminInvitePath         = "/office-portal/accept-invite?token="
)

type OfficeStore interface {
	ListOfficeAdmins(ctx context.Context) ([]store.OfficeAdmin, error)
	CreateAdminInvite(ctx context.Context, email, inviteHash string, expiresAt time.Time,
		invitedBy int64) (int64, error)
	ResendAdminInvite(ctx context.Context, userID int64, inviteHash string, expiresAt time.Time) (string, error)
	CancelAdminInvite(ctx context.Context, userID int64) (string, error)
	SuspendAdmin(ctx context.Context, userID int64) (string, error)
	RestoreAdmin(ctx context.Context, userID int64) (string, error)
	RemoveAdmin(ctx context.Context, userID int64) (string, error)
	OfficeAccount(ctx context.Context, userID int64) (string, string, string, error)
	SetTeacherOfficeAccess(ctx context.Context, userID int64, granted bool) (string, error)
	ListOfficeTeachers(ctx context.Context, search string, limit int) ([]store.OfficeTeacher, error)
}

type SuperAdmin struct {
	cfg    *config.Config
	office OfficeStore
	mail   mailer.Sender
	events *Recorder
}

func NewSuperAdmin(cfg *config.Config, office OfficeStore, mail mailer.Sender, events *Recorder) *SuperAdmin {
	return &SuperAdmin{cfg: cfg, office: office, mail: mail, events: events}
}

type officeAdminItem struct {
	ID          int64  `json:"id"`
	Username    string `json:"username"`
	Name        string `json:"name,omitempty"`
	Email       string `json:"email,omitempty"`
	Role        string `json:"role"`
	Status      string `json:"status"`
	Source      string `json:"source"`
	LastSignIn  string `json:"last_sign_in,omitempty"`
	InviteEnds  string `json:"invite_expires_at,omitempty"`
	InvitedBy   string `json:"invited_by,omitempty"`
	ActionCount int    `json:"action_count"`
	CreatedAt   string `json:"created_at"`
}

type officeAdminsResponse struct {
	Items []officeAdminItem `json:"items"`
}

type officeTeacherItem struct {
	ID           int64  `json:"id"`
	TeacherID    string `json:"teacher_id,omitempty"`
	Name         string `json:"name"`
	Designation  string `json:"designation,omitempty"`
	Email        string `json:"email,omitempty"`
	OfficeAccess bool   `json:"office_access"`
	LastSignIn   string `json:"last_sign_in,omitempty"`
}

type officeTeachersResponse struct {
	Items []officeTeacherItem `json:"items"`
}

type inviteAdminRequest struct {
	Email string `json:"email"`
}

type confirmNameRequest struct {
	Confirm string `json:"confirm"`
}

func (s *SuperAdmin) Admins() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		admins, err := s.office.ListOfficeAdmins(r.Context())
		if err != nil {
			log.Printf("handler: could not list the office admins: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		items := make([]officeAdminItem, 0, len(admins))
		for _, admin := range admins {
			items = append(items, officeAdminItem{
				ID:          admin.ID,
				Username:    admin.Identifier,
				Name:        admin.Name,
				Email:       admin.Email,
				Role:        admin.Role,
				Status:      admin.Status,
				Source:      admin.Source,
				LastSignIn:  optionalTime(admin.LastSignIn),
				InviteEnds:  optionalTime(admin.InviteEnds),
				InvitedBy:   admin.InvitedBy,
				ActionCount: admin.ActionCount,
				CreatedAt:   timestamp(admin.CreatedAt),
			})
		}
		httpx.WriteJSON(w, http.StatusOK, officeAdminsResponse{Items: items})
	}
}

func (s *SuperAdmin) Invite() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var request inviteAdminRequest
		if !httpx.DecodeJSON(w, r, &request) {
			return
		}
		email, err := validate.Email("Email", request.Email)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		current, _ := middleware.CurrentUser(r.Context())
		token, hash, err := newInviteToken()
		if err != nil {
			log.Printf("handler: could not generate an admin invitation token: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		expiresAt := s.cfg.Now().Add(inviteLifetime)
		userID, err := s.office.CreateAdminInvite(r.Context(), email, hash, expiresAt, current.ID)
		if errors.Is(err, store.ErrDuplicateAccount) {
			httpx.WriteError(w, http.StatusConflict, MessageAdminDuplicate)
			return
		}
		if err != nil {
			log.Printf("handler: could not invite an office admin: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		body := inviteCreatedResponse{
			ID:        userID,
			Message:   MessageAdminInvited,
			ExpiresAt: expiresAt.Format(time.RFC3339),
		}
		s.sendInvite(r, email, token, &body)
		s.events.Record(r, store.ActionInviteAdmin, "office_admin", idText(userID),
			"Invited "+email+" as an office admin")
		httpx.WriteJSON(w, http.StatusCreated, body)
	}
}

func (s *SuperAdmin) ResendInvite() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := pathID(w, r)
		if !ok {
			return
		}
		token, hash, err := newInviteToken()
		if err != nil {
			log.Printf("handler: could not generate an admin invitation token: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		expiresAt := s.cfg.Now().Add(inviteLifetime)
		email, err := s.office.ResendAdminInvite(r.Context(), userID, hash, expiresAt)
		if errors.Is(err, store.ErrNotFound) {
			httpx.WriteError(w, http.StatusNotFound, MessageAdminInviteGone)
			return
		}
		if err != nil {
			log.Printf("handler: could not refresh an admin invitation: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		body := inviteCreatedResponse{
			ID:        userID,
			Message:   MessageAdminInviteSent,
			ExpiresAt: expiresAt.Format(time.RFC3339),
		}
		s.sendInvite(r, email, token, &body)
		s.events.Record(r, store.ActionResendInvite, "office_admin", idText(userID),
			"Sent "+email+" a fresh office admin invitation")
		httpx.WriteJSON(w, http.StatusOK, body)
	}
}

func (s *SuperAdmin) CancelInvite() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := pathID(w, r)
		if !ok {
			return
		}
		email, err := s.office.CancelAdminInvite(r.Context(), userID)
		if errors.Is(err, store.ErrNotFound) {
			httpx.WriteError(w, http.StatusNotFound, MessageAdminInviteGone)
			return
		}
		if err != nil {
			log.Printf("handler: could not cancel an admin invitation: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		s.events.Record(r, store.ActionCancelInvite, "office_admin", idText(userID),
			"Cancelled the office admin invitation for "+email)
		httpx.WriteJSON(w, http.StatusOK, messageResponse{Message: MessageInviteCancelled})
	}
}

func (s *SuperAdmin) Suspend() http.HandlerFunc {
	return s.change(store.ActionSuspendAdmin, MessageAdminSuspended, false,
		func(ctx context.Context, userID int64) (string, error) {
			return s.office.SuspendAdmin(ctx, userID)
		})
}

func (s *SuperAdmin) Restore() http.HandlerFunc {
	return s.change(store.ActionRestoreAdmin, MessageAdminRestored, false,
		func(ctx context.Context, userID int64) (string, error) {
			return s.office.RestoreAdmin(ctx, userID)
		})
}

func (s *SuperAdmin) Remove() http.HandlerFunc {
	return s.change(store.ActionRemoveAdmin, MessageAdminRemoved, true,
		func(ctx context.Context, userID int64) (string, error) {
			return s.office.RemoveAdmin(ctx, userID)
		})
}

func (s *SuperAdmin) change(action, message string, confirmed bool,
	apply func(ctx context.Context, userID int64) (string, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := pathID(w, r)
		if !ok {
			return
		}
		if confirmed && !s.confirmUsername(w, r, userID) {
			return
		}
		username, err := apply(r.Context(), userID)
		switch {
		case errors.Is(err, store.ErrNotFound):
			httpx.WriteError(w, http.StatusNotFound, MessageAdminNotFound)
			return
		case errors.Is(err, store.ErrSuperAdminLocked):
			httpx.WriteError(w, http.StatusForbidden, MessageAdminProtected)
			return
		case errors.Is(err, store.ErrNotAnOfficeAdmin):
			httpx.WriteError(w, http.StatusConflict, MessageAdminNotAdmin)
			return
		case err != nil:
			log.Printf("handler: could not change an office admin: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		s.events.Record(r, action, "office_admin", idText(userID), message+" "+username)
		httpx.WriteJSON(w, http.StatusOK, messageResponse{Message: message})
	}
}

func (s *SuperAdmin) confirmUsername(w http.ResponseWriter, r *http.Request, userID int64) bool {
	var request confirmNameRequest
	if !httpx.DecodeJSON(w, r, &request) {
		return false
	}
	username, role, _, err := s.office.OfficeAccount(r.Context(), userID)
	if errors.Is(err, store.ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, MessageAdminNotFound)
		return false
	}
	if err != nil {
		log.Printf("handler: could not load an office admin: %v", err)
		httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
		return false
	}
	if role == auth.RoleSuperAdmin {
		httpx.WriteError(w, http.StatusForbidden, MessageAdminProtected)
		return false
	}
	if !strings.EqualFold(strings.TrimSpace(request.Confirm), username) {
		httpx.WriteError(w, http.StatusBadRequest, MessageAdminConfirm)
		return false
	}
	return true
}

func (s *SuperAdmin) Teachers() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		search := strings.TrimSpace(r.URL.Query().Get("search"))
		teachers, err := s.office.ListOfficeTeachers(r.Context(), search, store.DefaultStudentPage)
		if err != nil {
			log.Printf("handler: could not list the teachers for office access: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		items := make([]officeTeacherItem, 0, len(teachers))
		for _, teacher := range teachers {
			items = append(items, officeTeacherItem{
				ID:           teacher.ID,
				TeacherID:    teacher.TeacherID,
				Name:         teacher.Name,
				Designation:  teacher.Designation,
				Email:        teacher.Email,
				OfficeAccess: teacher.OfficeAccess,
				LastSignIn:   optionalTime(teacher.LastSignIn),
			})
		}
		httpx.WriteJSON(w, http.StatusOK, officeTeachersResponse{Items: items})
	}
}

func (s *SuperAdmin) SetOfficeAccess(granted bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := pathID(w, r)
		if !ok {
			return
		}
		name, err := s.office.SetTeacherOfficeAccess(r.Context(), userID, granted)
		if errors.Is(err, store.ErrNotATeacher) {
			httpx.WriteError(w, http.StatusConflict, MessageTeacherNoOffice)
			return
		}
		if err != nil {
			log.Printf("handler: could not change the office access of a teacher: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		action := store.ActionRevokeOffice
		message := MessageOfficeRevoked
		if granted {
			action = store.ActionGrantOffice
			message = MessageOfficeGranted
		}
		s.events.Record(r, action, "teacher", idText(userID), message+" for "+name)
		httpx.WriteJSON(w, http.StatusOK, messageResponse{Message: message})
	}
}

func (s *SuperAdmin) sendInvite(r *http.Request, email, token string, body *inviteCreatedResponse) {
	link := s.cfg.FrontendOrigin + adminInvitePath + token
	if s.mail != nil && s.mail.Configured() {
		if err := s.mail.Send(r.Context(), email, AdminInviteEmailSubject, adminInviteEmailBody(link)); err != nil {
			log.Printf("handler: could not email an office admin invitation: %v", err)
		}
		return
	}
	if !s.cfg.IsProduction() {
		body.InviteLink = link
	}
}

func adminInviteEmailBody(link string) string {
	return "You have been invited to help run the SUST CSE portal office pages.\n\n" +
		"Open this link within 72 hours to choose your username and password:\n" + link
}

func optionalTime(at *time.Time) string {
	if at == nil {
		return ""
	}
	return at.Format(time.RFC3339)
}

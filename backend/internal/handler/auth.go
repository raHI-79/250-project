package handler

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"github.com/raHI-79/250-project/backend/internal/auth"
	"github.com/raHI-79/250-project/backend/internal/config"
	"github.com/raHI-79/250-project/backend/internal/httpx"
	"github.com/raHI-79/250-project/backend/internal/middleware"
	"github.com/raHI-79/250-project/backend/internal/store"
	"github.com/raHI-79/250-project/backend/internal/validate"
	"log"
	"net/http"
	"time"
)

const (
	MessageInvalidLogin      = "Invalid ID or password."
	MessagePendingAccount    = "Your registration is pending approval by the administration."
	MessageRejectedAccount   = "Your registration was rejected. Please contact the department office."
	MessageRemovedAccount    = "Your account has been removed. Please contact the department office."
	MessageDuplicateAccount  = "An account with this registration number or email already exists."
	MessageRegistrationDone  = "Registration submitted. You can log in after the administration approves your account."
	MessageLoggedOut         = "Logged out."
	MessageNoActiveAccount   = "No active account found for this ID."
	MessageResetTokenIssued  = "Use this token to reset your password. In production it would be sent by email."
	MessageInvalidResetToken = "Invalid or expired reset token."
	MessagePasswordUpdated   = "Password updated. Please log in with your new password."
	resetTokenBytes          = 32
	resetTokenLifetime       = 15 * time.Minute
)

type AuthStore interface {
	CreateRegistration(ctx context.Context, registration store.Registration) (int64, error)
	FindLoginUser(ctx context.Context, identifier string) (store.LoginUser, error)
	IncrementTokenVersion(ctx context.Context, userID int64) (int, error)
	SetResetToken(ctx context.Context, userID int64, tokenHash string, expiresAt time.Time) error
	ResetPassword(ctx context.Context, identifier, tokenHash, passwordHash string) error
}

type TokenIssuer interface {
	IssueToken(userID int64, effectiveRole string, tokenVersion int) (string, error)
}

type Auth struct {
	cfg    *config.Config
	users  AuthStore
	tokens TokenIssuer
}

func NewAuth(cfg *config.Config, users AuthStore, tokens TokenIssuer) *Auth {
	return &Auth{cfg: cfg, users: users, tokens: tokens}
}

type registerRequest struct {
	FullName        string `json:"full_name"`
	RegNo           string `json:"reg_no"`
	Batch           string `json:"batch"`
	Section         string `json:"section"`
	Email           string `json:"email"`
	Phone           string `json:"phone"`
	BloodGroup      string `json:"blood_group"`
	Password        string `json:"password"`
	ConfirmPassword string `json:"confirm_password"`
	HasDropCourse   bool   `json:"has_drop_course"`
	DropCourseCode  string `json:"drop_course_code"`
}

type messageResponse struct {
	Message string `json:"message"`
}

type loginRequest struct {
	Identifier string `json:"identifier"`
	Password   string `json:"password"`
}

type loginUser struct {
	ID         int64  `json:"id"`
	Identifier string `json:"identifier"`
	Role       string `json:"role"`
	Name       string `json:"name,omitempty"`
	Batch      string `json:"batch,omitempty"`
	Section    string `json:"section,omitempty"`
}

type loginResponse struct {
	Token string    `json:"token"`
	User  loginUser `json:"user"`
}

type forgotPasswordRequest struct {
	Identifier string `json:"identifier"`
}

type forgotPasswordResponse struct {
	Message    string `json:"message"`
	ResetToken string `json:"reset_token"`
	ExpiresAt  string `json:"expires_at"`
}

type resetPasswordRequest struct {
	Identifier      string `json:"identifier"`
	Token           string `json:"token"`
	NewPassword     string `json:"new_password"`
	ConfirmPassword string `json:"confirm_password"`
}

func (a *Auth) Register() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var request registerRequest
		if !httpx.DecodeJSON(w, r, &request) {
			return
		}
		registration, err := request.toRegistration()
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		hash, err := auth.HashPassword(request.Password)
		if err != nil {
			log.Printf("handler: could not hash a registration password: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		registration.PasswordHash = hash
		if _, err := a.users.CreateRegistration(r.Context(), registration); err != nil {
			if errors.Is(err, store.ErrDuplicateAccount) {
				httpx.WriteError(w, http.StatusConflict, MessageDuplicateAccount)
				return
			}
			log.Printf("handler: could not create a registration: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		httpx.WriteJSON(w, http.StatusCreated, messageResponse{Message: MessageRegistrationDone})
	}
}

func (request registerRequest) toRegistration() (store.Registration, error) {
	registration := store.Registration{HasDropCourse: request.HasDropCourse}
	fields := []struct {
		target *string
		value  string
		label  string
		check  func(string, string) (string, error)
	}{
		{target: &registration.FullName, value: request.FullName, label: "Full name", check: validate.Required},
		{target: &registration.RegNo, value: request.RegNo, label: "Registration number", check: validate.Required},
		{target: &registration.Batch, value: request.Batch, label: "Batch", check: validate.Batch},
		{target: &registration.Section, value: request.Section, label: "Section", check: validate.Section},
		{target: &registration.Email, value: request.Email, label: "Email", check: validate.Email},
		{target: &registration.Phone, value: request.Phone, label: "Phone", check: validate.Phone},
		{target: &registration.BloodGroup, value: request.BloodGroup, label: "Blood group", check: validate.BloodGroup},
	}
	for _, field := range fields {
		value, err := field.check(field.label, field.value)
		if err != nil {
			return store.Registration{}, err
		}
		*field.target = value
	}
	if err := validate.Password("Password", request.Password, request.ConfirmPassword); err != nil {
		return store.Registration{}, err
	}
	if request.HasDropCourse {
		code, err := validate.CourseCode("Drop course code", request.DropCourseCode)
		if err != nil {
			return store.Registration{}, err
		}
		registration.DropCourseCode = code
	}
	return registration, nil
}

func (a *Auth) Login() http.HandlerFunc {
	return a.login(false)
}

func (a *Auth) AdminLogin() http.HandlerFunc {
	return a.login(true)
}

func (a *Auth) login(adminOnly bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var request loginRequest
		if !httpx.DecodeJSON(w, r, &request) {
			return
		}
		identifier, err := validate.Required("ID", request.Identifier)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		if request.Password == "" {
			httpx.WriteError(w, http.StatusBadRequest, "Password is required.")
			return
		}
		user, err := a.users.FindLoginUser(r.Context(), identifier)
		if errors.Is(err, store.ErrNotFound) {
			httpx.WriteError(w, http.StatusUnauthorized, MessageInvalidLogin)
			return
		}
		if err != nil {
			log.Printf("handler: could not load a login user: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		if auth.CheckPassword(user.PasswordHash, request.Password) != nil {
			httpx.WriteError(w, http.StatusUnauthorized, MessageInvalidLogin)
			return
		}
		isAdmin := user.Role == auth.RoleAdmin
		if isAdmin != adminOnly {
			httpx.WriteError(w, http.StatusUnauthorized, MessageInvalidLogin)
			return
		}
		if adminOnly {
			if !user.IsActive() {
				httpx.WriteError(w, http.StatusUnauthorized, MessageInvalidLogin)
				return
			}
		} else if message, blocked := statusMessage(user.Status); blocked {
			httpx.WriteError(w, http.StatusForbidden, message)
			return
		}
		role := auth.EffectiveRole(user.Role, user.IsCR)
		token, err := a.tokens.IssueToken(user.ID, role, user.TokenVersion)
		if err != nil {
			log.Printf("handler: could not issue a token: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		body := loginResponse{
			Token: token,
			User: loginUser{
				ID:         user.ID,
				Identifier: user.Identifier,
				Role:       role,
				Name:       user.Name,
			},
		}
		if role == auth.RoleStudent || role == auth.RoleCR {
			body.User.Batch = user.Batch
			body.User.Section = user.Section
		}
		httpx.WriteJSON(w, http.StatusOK, body)
	}
}

func statusMessage(status string) (string, bool) {
	switch status {
	case store.StatusPending:
		return MessagePendingAccount, true
	case store.StatusRejected:
		return MessageRejectedAccount, true
	case store.StatusRemoved:
		return MessageRemovedAccount, true
	case store.StatusActive:
		return "", false
	default:
		return MessageRemovedAccount, true
	}
}

func (a *Auth) Logout() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		current, ok := middleware.CurrentUser(r.Context())
		if !ok {
			httpx.WriteError(w, http.StatusUnauthorized, middleware.MessageUnauthorized)
			return
		}
		if _, err := a.users.IncrementTokenVersion(r.Context(), current.ID); err != nil {
			if errors.Is(err, store.ErrNotFound) {
				httpx.WriteError(w, http.StatusUnauthorized, middleware.MessageSessionExpired)
				return
			}
			log.Printf("handler: could not invalidate the tokens of a user: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, messageResponse{Message: MessageLoggedOut})
	}
}

func (a *Auth) ForgotPassword() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var request forgotPasswordRequest
		if !httpx.DecodeJSON(w, r, &request) {
			return
		}
		identifier, err := validate.Required("ID", request.Identifier)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		user, err := a.users.FindLoginUser(r.Context(), identifier)
		if errors.Is(err, store.ErrNotFound) {
			httpx.WriteError(w, http.StatusNotFound, MessageNoActiveAccount)
			return
		}
		if err != nil {
			log.Printf("handler: could not load a user for a password reset: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		if !user.IsActive() {
			httpx.WriteError(w, http.StatusNotFound, MessageNoActiveAccount)
			return
		}
		token, hash, err := newResetToken()
		if err != nil {
			log.Printf("handler: could not generate a reset token: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		expiresAt := a.cfg.Now().Add(resetTokenLifetime)
		if err := a.users.SetResetToken(r.Context(), user.ID, hash, expiresAt); err != nil {
			if errors.Is(err, store.ErrNotFound) {
				httpx.WriteError(w, http.StatusNotFound, MessageNoActiveAccount)
				return
			}
			log.Printf("handler: could not store a reset token: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, forgotPasswordResponse{
			Message:    MessageResetTokenIssued,
			ResetToken: token,
			ExpiresAt:  expiresAt.Format(time.RFC3339),
		})
	}
}

func (a *Auth) ResetPassword() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var request resetPasswordRequest
		if !httpx.DecodeJSON(w, r, &request) {
			return
		}
		identifier, err := validate.Required("ID", request.Identifier)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		token, err := validate.Required("Reset token", request.Token)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		if err := validate.Password("New password", request.NewPassword, request.ConfirmPassword); err != nil {
			httpx.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		hash, err := auth.HashPassword(request.NewPassword)
		if err != nil {
			log.Printf("handler: could not hash a new password: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		err = a.users.ResetPassword(r.Context(), identifier, hashResetToken(token), hash)
		if errors.Is(err, store.ErrInvalidResetToken) || errors.Is(err, store.ErrNotFound) {
			httpx.WriteError(w, http.StatusBadRequest, MessageInvalidResetToken)
			return
		}
		if err != nil {
			log.Printf("handler: could not reset a password: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, messageResponse{Message: MessagePasswordUpdated})
	}
}

func newResetToken() (string, string, error) {
	raw := make([]byte, resetTokenBytes)
	if _, err := rand.Read(raw); err != nil {
		return "", "", err
	}
	token := hex.EncodeToString(raw)
	return token, hashResetToken(token), nil
}

func hashResetToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

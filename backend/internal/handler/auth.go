package handler

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/raHI-79/250-project/backend/internal/auth"
	"github.com/raHI-79/250-project/backend/internal/config"
	"github.com/raHI-79/250-project/backend/internal/httpx"
	"github.com/raHI-79/250-project/backend/internal/mailer"
	"github.com/raHI-79/250-project/backend/internal/middleware"
	"github.com/raHI-79/250-project/backend/internal/store"
	"github.com/raHI-79/250-project/backend/internal/validate"
	"log"
	"math/big"
	"net/http"
	"strings"
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
	MessageResetMismatch     = "The ID and phone number do not match an active account."
	MessageResetCooldown     = "Please wait a minute before requesting a new code."
	MessageResetCodeSent     = "We sent a 6-digit code to the email address registered with this account."
	MessageInvalidResetCode  = "Invalid or expired code."
	MessageTooManyResetCodes = "Too many wrong codes. Please request a new code."
	MessageInvalidResetToken = "Invalid or expired reset token."
	MessagePasswordUpdated   = "Password updated. Please log in with your new password."
	ResetEmailSubject        = "Password reset code"
	resetTokenBytes          = 32
	resetCodeDigits          = 6
	resetCodeLifetime        = 10 * time.Minute
	resetTokenLifetime       = 10 * time.Minute
	resetResendCooldown      = time.Minute
	maxResetAttempts         = 5
)

type AuthStore interface {
	CreateRegistration(ctx context.Context, registration store.Registration) (int64, error)
	FindLoginUser(ctx context.Context, identifier string) (store.LoginUser, error)
	IncrementTokenVersion(ctx context.Context, userID int64) (int, error)
	FindResetTarget(ctx context.Context, identifier string) (store.ResetTarget, error)
	FindResetChallenge(ctx context.Context, identifier string) (store.ResetChallenge, error)
	StartPasswordReset(ctx context.Context, userID int64, codeHash string, expiresAt, sentAt time.Time) error
	CountResetAttempt(ctx context.Context, userID int64) (int, error)
	ClearPasswordReset(ctx context.Context, userID int64) error
	VerifyResetCode(ctx context.Context, userID int64, tokenHash string, expiresAt time.Time) error
	FinishPasswordReset(ctx context.Context, identifier, tokenHash, passwordHash string) error
}

type TokenIssuer interface {
	IssueToken(userID int64, effectiveRole string, tokenVersion int) (string, error)
}

type Auth struct {
	cfg    *config.Config
	users  AuthStore
	tokens TokenIssuer
	mail   mailer.Sender
}

func NewAuth(cfg *config.Config, users AuthStore, tokens TokenIssuer, mail mailer.Sender) *Auth {
	return &Auth{cfg: cfg, users: users, tokens: tokens, mail: mail}
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
	Phone      string `json:"phone"`
}

type forgotPasswordResponse struct {
	Message   string `json:"message"`
	EmailHint string `json:"email_hint"`
	ExpiresAt string `json:"expires_at"`
	DevCode   string `json:"dev_code,omitempty"`
}

type verifyResetCodeRequest struct {
	Identifier string `json:"identifier"`
	Code       string `json:"code"`
}

type verifyResetCodeResponse struct {
	ResetToken string `json:"reset_token"`
}

type resetPasswordRequest struct {
	Identifier      string `json:"identifier"`
	ResetToken      string `json:"reset_token"`
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
		phone, err := validate.Required("Phone number", request.Phone)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		target, err := a.users.FindResetTarget(r.Context(), identifier)
		if errors.Is(err, store.ErrNotFound) {
			httpx.WriteError(w, http.StatusBadRequest, MessageResetMismatch)
			return
		}
		if err != nil {
			log.Printf("handler: could not load a user for a password reset: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		if !target.IsActive() || !validate.SamePhone(phone, target.Phone) {
			httpx.WriteError(w, http.StatusBadRequest, MessageResetMismatch)
			return
		}
		now := a.cfg.Now()
		if target.ResetSentAt != nil && now.Sub(*target.ResetSentAt) < resetResendCooldown {
			httpx.WriteError(w, http.StatusTooManyRequests, MessageResetCooldown)
			return
		}
		code, err := newResetCode()
		if err != nil {
			log.Printf("handler: could not generate a reset code: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		expiresAt := now.Add(resetCodeLifetime)
		if err := a.users.StartPasswordReset(r.Context(), target.ID, hashResetSecret(code), expiresAt, now); err != nil {
			if errors.Is(err, store.ErrNotFound) {
				httpx.WriteError(w, http.StatusBadRequest, MessageResetMismatch)
				return
			}
			log.Printf("handler: could not store a reset code: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		body := forgotPasswordResponse{
			Message:   MessageResetCodeSent,
			EmailHint: emailHint(target.Email),
			ExpiresAt: expiresAt.Format(time.RFC3339),
		}
		if a.mailerReady() {
			if err := a.mail.Send(r.Context(), target.Email, ResetEmailSubject, resetEmailBody(code)); err != nil {
				log.Printf("handler: could not email a reset code to user %d: %v", target.ID, err)
				if clearErr := a.users.ClearPasswordReset(r.Context(), target.ID); clearErr != nil {
					log.Printf("handler: could not clear the reset fields of user %d: %v", target.ID, clearErr)
				}
				httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
				return
			}
		} else if !a.cfg.IsProduction() {
			body.DevCode = code
		}
		httpx.WriteJSON(w, http.StatusOK, body)
	}
}

func (a *Auth) VerifyResetCode() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var request verifyResetCodeRequest
		if !httpx.DecodeJSON(w, r, &request) {
			return
		}
		identifier, err := validate.Required("ID", request.Identifier)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		code := strings.TrimSpace(request.Code)
		if !isResetCode(code) {
			httpx.WriteError(w, http.StatusBadRequest, MessageInvalidResetCode)
			return
		}
		challenge, err := a.users.FindResetChallenge(r.Context(), identifier)
		if errors.Is(err, store.ErrNotFound) {
			httpx.WriteError(w, http.StatusBadRequest, MessageInvalidResetCode)
			return
		}
		if err != nil {
			log.Printf("handler: could not load a reset challenge: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		now := a.cfg.Now()
		if challenge.Status != store.StatusActive || challenge.Stage != store.ResetStageCode ||
			challenge.TokenHash == "" || challenge.ExpiresAt == nil || !challenge.ExpiresAt.After(now) {
			httpx.WriteError(w, http.StatusBadRequest, MessageInvalidResetCode)
			return
		}
		if challenge.TokenHash != hashResetSecret(code) {
			attempts, err := a.users.CountResetAttempt(r.Context(), challenge.UserID)
			if err != nil {
				log.Printf("handler: could not count a reset attempt: %v", err)
				httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
				return
			}
			if attempts >= maxResetAttempts {
				if err := a.users.ClearPasswordReset(r.Context(), challenge.UserID); err != nil {
					log.Printf("handler: could not clear the reset fields of user %d: %v", challenge.UserID, err)
				}
				httpx.WriteError(w, http.StatusBadRequest, MessageTooManyResetCodes)
				return
			}
			httpx.WriteError(w, http.StatusBadRequest, MessageInvalidResetCode)
			return
		}
		token, hash, err := newResetToken()
		if err != nil {
			log.Printf("handler: could not generate a reset token: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		err = a.users.VerifyResetCode(r.Context(), challenge.UserID, hash, now.Add(resetTokenLifetime))
		if errors.Is(err, store.ErrInvalidResetToken) || errors.Is(err, store.ErrNotFound) {
			httpx.WriteError(w, http.StatusBadRequest, MessageInvalidResetCode)
			return
		}
		if err != nil {
			log.Printf("handler: could not store a verified reset token: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, verifyResetCodeResponse{ResetToken: token})
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
		token, err := validate.Required("Reset token", request.ResetToken)
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
		err = a.users.FinishPasswordReset(r.Context(), identifier, hashResetSecret(token), hash)
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

func (a *Auth) mailerReady() bool {
	return a.mail != nil && a.mail.Configured()
}

func resetEmailBody(code string) string {
	return fmt.Sprintf("Your password reset code is %s. It expires in 10 minutes.", code)
}

func emailHint(email string) string {
	at := strings.Index(email, "@")
	if at <= 0 {
		return ""
	}
	return email[:1] + "***" + email[at:]
}

func isResetCode(code string) bool {
	if len(code) != resetCodeDigits {
		return false
	}
	for _, symbol := range code {
		if symbol < '0' || symbol > '9' {
			return false
		}
	}
	return true
}

func newResetCode() (string, error) {
	limit := big.NewInt(1)
	for i := 0; i < resetCodeDigits; i++ {
		limit.Mul(limit, big.NewInt(10))
	}
	drawn, err := rand.Int(rand.Reader, limit)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%0*d", resetCodeDigits, drawn), nil
}

func newResetToken() (string, string, error) {
	raw := make([]byte, resetTokenBytes)
	if _, err := rand.Read(raw); err != nil {
		return "", "", err
	}
	token := hex.EncodeToString(raw)
	return token, hashResetSecret(token), nil
}

func hashResetSecret(secret string) string {
	sum := sha256.Sum256([]byte(secret))
	return hex.EncodeToString(sum[:])
}

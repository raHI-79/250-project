package handler

import (
	"errors"
	"fmt"
	"github.com/raHI-79/250-project/backend/internal/auth"
	"github.com/raHI-79/250-project/backend/internal/httpx"
	"github.com/raHI-79/250-project/backend/internal/middleware"
	"github.com/raHI-79/250-project/backend/internal/store"
	"github.com/raHI-79/250-project/backend/internal/upload"
	"github.com/raHI-79/250-project/backend/internal/validate"
	"log"
	"net/http"
	"strings"
	"time"
)

const (
	MessageRegistrationDone      = "Registration submitted. You can log in after the administration approves your account."
	MessageVerifyEmailSent       = "We sent a 6-digit code to your university email address. Enter it to finish your registration."
	MessagePhotoRequired         = "A profile photo is required."
	MessagePhotoNotImage         = "The profile photo must be a JPG, PNG or WEBP image."
	MessagePhotoTooLarge         = "The profile photo must be 2 MB or smaller."
	MessagePhotoTooSmall         = "The profile photo must be at least 200 by 200 pixels."
	MessageInvalidEmailProof     = "The Google verification has expired. Please verify your email again."
	MessageInvalidEmailCode      = "Invalid or expired code."
	MessageTooManyEmailCodes     = "Too many wrong codes. Request a new code."
	MessageEmailVerified         = "Email verified. You can log in after the administration approves your account."
	MessageVerificationCooldown  = "Please wait a minute before requesting a new code."
	MessageAlreadyVerified       = "This registration number is not waiting for an email code."
	MessagePasswordTooLong       = "Password must be at most 72 characters long."
	VerifyEmailSubject           = "Confirm your email address"
	fieldFullName                = "full_name"
	fieldRegNo                   = "reg_no"
	fieldEmail                   = "email"
	fieldPhone                   = "phone"
	fieldBloodGroup              = "blood_group"
	fieldDateOfBirth             = "date_of_birth"
	fieldPassword                = "password"
	fieldConfirmPassword         = "confirm_password"
	fieldEmailProof              = "email_proof"
	emailCodeLifetime            = 15 * time.Minute
	emailCodeResendCooldown      = time.Minute
	maxEmailVerificationAttempts = 5
)

type registerResponse struct {
	Message   string `json:"message"`
	EmailHint string `json:"email_hint"`
	ExpiresAt string `json:"expires_at"`
	DevCode   string `json:"dev_code,omitempty"`
}

type verifyEmailRequest struct {
	RegNo string `json:"reg_no"`
	Code  string `json:"code"`
}

type resendVerificationRequest struct {
	RegNo string `json:"reg_no"`
}

func (a *Auth) Register() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := a.photos.Parse(r); err != nil {
			status, message := photoError(err)
			httpx.WriteError(w, status, message)
			return
		}
		registration, err := a.registrationFrom(r)
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
		registration.PhotoPath = photoPath
		code := ""
		now := a.cfg.Now()
		proof := strings.TrimSpace(r.FormValue(fieldEmailProof))
		if proof != "" {
			if err := a.proofs.CheckEmailProof(proof, registration.Email); err != nil {
				a.removePhoto(photoPath)
				httpx.WriteError(w, http.StatusBadRequest, MessageInvalidEmailProof)
				return
			}
			verifiedAt := now
			registration.Status = store.StatusPending
			registration.EmailVerifiedAt = &verifiedAt
		} else {
			registration.Status = store.StatusUnverified
			code, err = newResetCode()
			if err != nil {
				a.removePhoto(photoPath)
				log.Printf("handler: could not generate an email code: %v", err)
				httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
				return
			}
			expiresAt := now.Add(emailCodeLifetime)
			sentAt := now
			registration.CodeHash = hashResetSecret(code)
			registration.CodeExpiresAt = &expiresAt
			registration.CodeSentAt = &sentAt
		}
		result, err := a.users.CreateRegistration(r.Context(), registration)
		if err != nil {
			a.removePhoto(photoPath)
			if errors.Is(err, store.ErrDuplicateAccount) {
				httpx.WriteError(w, http.StatusConflict, MessageDuplicateAccount)
				return
			}
			log.Printf("handler: could not create a registration: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		for _, replaced := range result.ReplacedPhotos {
			a.removePhoto(replaced)
		}
		if registration.Status == store.StatusPending {
			httpx.WriteJSON(w, http.StatusCreated, registerResponse{
				Message:   MessageRegistrationDone,
				EmailHint: emailHint(registration.Email),
			})
			return
		}
		body := registerResponse{
			Message:   MessageVerifyEmailSent,
			EmailHint: emailHint(registration.Email),
			ExpiresAt: registration.CodeExpiresAt.Format(time.RFC3339),
		}
		a.sendVerificationCode(r, registration.Email, code, &body)
		httpx.WriteJSON(w, http.StatusCreated, body)
	}
}

func (a *Auth) registrationFrom(r *http.Request) (store.Registration, error) {
	var registration store.Registration
	fields := []struct {
		target *string
		field  string
		label  string
		check  func(string, string) (string, error)
	}{
		{target: &registration.FullName, field: fieldFullName, label: "Full name", check: validate.Required},
		{target: &registration.RegNo, field: fieldRegNo, label: "Registration number", check: validate.Required},
		{target: &registration.Batch, field: fieldBatch, label: "Batch", check: validate.Batch},
		{target: &registration.Section, field: fieldSection, label: "Section", check: validate.Section},
		{target: &registration.Phone, field: fieldPhone, label: "Phone", check: validate.Phone},
		{target: &registration.BloodGroup, field: fieldBloodGroup, label: "Blood group", check: validate.BloodGroup},
	}
	for _, field := range fields {
		value, err := field.check(field.label, r.FormValue(field.field))
		if err != nil {
			return store.Registration{}, err
		}
		*field.target = value
	}
	email, err := validate.EmailInDomain("Email", r.FormValue(fieldEmail), a.cfg.StudentDomain)
	if err != nil {
		return store.Registration{}, err
	}
	registration.Email = email
	birthDate, err := validate.BirthDate("Date of birth", r.FormValue(fieldDateOfBirth), a.cfg.Now())
	if err != nil {
		return store.Registration{}, err
	}
	registration.DateOfBirth = birthDate
	password := r.FormValue(fieldPassword)
	if err := validate.Password("Password", password, r.FormValue(fieldConfirmPassword)); err != nil {
		return store.Registration{}, err
	}
	hash, err := auth.HashPassword(password)
	if err != nil {
		return store.Registration{}, errors.New(MessagePasswordTooLong)
	}
	registration.PasswordHash = hash
	return registration, nil
}

func (a *Auth) sendVerificationCode(r *http.Request, email, code string, body *registerResponse) {
	if a.mailerReady() {
		if err := a.mail.Send(r.Context(), email, VerifyEmailSubject, verifyEmailBody(code)); err != nil {
			log.Printf("handler: could not email a verification code: %v", err)
		}
		return
	}
	if !a.cfg.IsProduction() {
		body.DevCode = code
	}
}

func (a *Auth) removePhoto(photoPath string) {
	if photoPath == "" {
		return
	}
	if err := a.photos.Delete(photoPath); err != nil {
		log.Printf("handler: could not delete a profile photo: %v", err)
	}
}

func (a *Auth) VerifyEmail() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var request verifyEmailRequest
		if !httpx.DecodeJSON(w, r, &request) {
			return
		}
		regNo, err := validate.Required("Registration number", request.RegNo)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		code := strings.TrimSpace(request.Code)
		if !isSixDigitCode(code) {
			httpx.WriteError(w, http.StatusBadRequest, MessageInvalidEmailCode)
			return
		}
		challenge, err := a.users.FindEmailChallenge(r.Context(), regNo)
		if errors.Is(err, store.ErrNotFound) {
			httpx.WriteError(w, http.StatusBadRequest, MessageInvalidEmailCode)
			return
		}
		if err != nil {
			log.Printf("handler: could not load an email challenge: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		now := a.cfg.Now()
		if challenge.Status != store.StatusUnverified || challenge.CodeHash == "" ||
			challenge.ExpiresAt == nil || !challenge.ExpiresAt.After(now) {
			httpx.WriteError(w, http.StatusBadRequest, MessageInvalidEmailCode)
			return
		}
		if challenge.CodeHash != hashResetSecret(code) {
			attempts, err := a.users.CountEmailAttempt(r.Context(), challenge.UserID)
			if err != nil {
				log.Printf("handler: could not count an email verification attempt: %v", err)
				httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
				return
			}
			if attempts >= maxEmailVerificationAttempts {
				if err := a.users.ClearEmailCode(r.Context(), challenge.UserID); err != nil {
					log.Printf("handler: could not clear the email code of user %d: %v", challenge.UserID, err)
				}
				httpx.WriteError(w, http.StatusBadRequest, MessageTooManyEmailCodes)
				return
			}
			httpx.WriteError(w, http.StatusBadRequest, MessageInvalidEmailCode)
			return
		}
		if err := a.users.ConfirmEmail(r.Context(), challenge.UserID, now); err != nil {
			if errors.Is(err, store.ErrNotFound) {
				httpx.WriteError(w, http.StatusBadRequest, MessageInvalidEmailCode)
				return
			}
			log.Printf("handler: could not confirm an email address: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, messageResponse{Message: MessageEmailVerified})
	}
}

func (a *Auth) ResendVerification() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var request resendVerificationRequest
		if !httpx.DecodeJSON(w, r, &request) {
			return
		}
		regNo, err := validate.Required("Registration number", request.RegNo)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		challenge, err := a.users.FindEmailChallenge(r.Context(), regNo)
		if errors.Is(err, store.ErrNotFound) {
			httpx.WriteError(w, http.StatusBadRequest, MessageAlreadyVerified)
			return
		}
		if err != nil {
			log.Printf("handler: could not load an email challenge: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		if challenge.Status != store.StatusUnverified {
			httpx.WriteError(w, http.StatusBadRequest, MessageAlreadyVerified)
			return
		}
		now := a.cfg.Now()
		if challenge.SentAt != nil && now.Sub(*challenge.SentAt) < emailCodeResendCooldown {
			httpx.WriteError(w, http.StatusTooManyRequests, MessageVerificationCooldown)
			return
		}
		code, err := newResetCode()
		if err != nil {
			log.Printf("handler: could not generate an email code: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		expiresAt := now.Add(emailCodeLifetime)
		err = a.users.StartEmailVerification(r.Context(), challenge.UserID, hashResetSecret(code), expiresAt, now)
		if err != nil {
			if errors.Is(err, store.ErrNotFound) {
				httpx.WriteError(w, http.StatusBadRequest, MessageAlreadyVerified)
				return
			}
			log.Printf("handler: could not store an email code: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		body := registerResponse{
			Message:   MessageVerifyEmailSent,
			EmailHint: emailHint(challenge.Email),
			ExpiresAt: expiresAt.Format(time.RFC3339),
		}
		a.sendVerificationCode(r, challenge.Email, code, &body)
		httpx.WriteJSON(w, http.StatusOK, body)
	}
}

func verifyEmailBody(code string) string {
	return fmt.Sprintf("Your email confirmation code is %s. It expires in 15 minutes.", code)
}

func photoError(err error) (int, string) {
	switch {
	case errors.Is(err, upload.ErrPhotoTooLarge):
		return http.StatusRequestEntityTooLarge, MessagePhotoTooLarge
	case errors.Is(err, upload.ErrPhotoTooSmall):
		return http.StatusBadRequest, MessagePhotoTooSmall
	case errors.Is(err, upload.ErrNotPhoto):
		return http.StatusBadRequest, MessagePhotoNotImage
	case errors.Is(err, upload.ErrMissingPhoto):
		return http.StatusBadRequest, MessagePhotoRequired
	default:
		return http.StatusBadRequest, httpx.MessageInvalidBody
	}
}

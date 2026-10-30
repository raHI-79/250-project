package handler

import (
	"context"
	"errors"
	"fmt"
	"github.com/raHI-79/250-project/backend/internal/httpx"
	"github.com/raHI-79/250-project/backend/internal/middleware"
	"github.com/raHI-79/250-project/backend/internal/store"
	"log"
	"net/http"
	"strings"
	"time"
)

const (
	WelcomeEmailSubject  = "Welcome to SUST CSE Portal"
	RejectedEmailSubject = "Your SUST CSE Portal registration was not approved"

	MessageReasonRequired = "Write why this registration is refused, at least 5 characters."
	MinRejectionReason    = 5
	MaxRejectionReason    = 300
	emailTimeout          = 30 * time.Second
)

var QuickRejectionReasons = []string{
	"Registration number does not match",
	"Not a CSE student",
	"Photo is not clear",
	"Name does not match the registration number",
}

type decideRegistrationRequest struct {
	Reason string `json:"reason"`
}

type rejectionReasonsResponse struct {
	Items []string `json:"items"`
	Min   int      `json:"min"`
	Max   int      `json:"max"`
}

func (a *Admin) RejectionReasons() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		httpx.WriteJSON(w, http.StatusOK, rejectionReasonsResponse{
			Items: QuickRejectionReasons,
			Min:   MinRejectionReason,
			Max:   MaxRejectionReason,
		})
	}
}

func rejectionReason(body string) (string, error) {
	reason := strings.Join(strings.Fields(body), " ")
	if len([]rune(reason)) < MinRejectionReason {
		return "", errors.New(MessageReasonRequired)
	}
	if len([]rune(reason)) > MaxRejectionReason {
		return "", fmt.Errorf("The reason must be at most %d characters long.", MaxRejectionReason)
	}
	return reason, nil
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
		reason := ""
		if status == store.StatusRejected {
			var request decideRegistrationRequest
			if r.ContentLength > 0 && !httpx.DecodeJSON(w, r, &request) {
				return
			}
			cleaned, err := rejectionReason(request.Reason)
			if err != nil {
				httpx.WriteError(w, http.StatusBadRequest, err.Error())
				return
			}
			reason = cleaned
		}
		decided, err := a.people.DecideRegistration(r.Context(), userID, status, reason)
		switch {
		case errors.Is(err, store.ErrNotFound):
			httpx.WriteError(w, http.StatusNotFound, MessageStudentNotFound)
			return
		case errors.Is(err, store.ErrNotPending):
			httpx.WriteError(w, http.StatusConflict, MessageRegistrationNotPending)
			return
		case err != nil:
			log.Printf("handler: could not decide a registration: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, middleware.MessageServerError)
			return
		}
		a.mailDecision(decided, status, reason)
		httpx.WriteJSON(w, http.StatusOK, messageResponse{Message: message})
	}
}

func (a *Admin) mailDecision(decided store.DecidedRegistration, status, reason string) {
	if !a.mailerReady() || decided.Email == "" {
		return
	}
	subject := WelcomeEmailSubject
	body := welcomeEmailBody(decided.FullName, decided.RegNo, a.cfg.FrontendOrigin)
	if status == store.StatusRejected {
		subject = RejectedEmailSubject
		body = rejectedEmailBody(decided.FullName, reason, a.cfg.FrontendOrigin)
	}
	go a.deliver(decided.Email, subject, body)
}

func (a *Admin) deliver(to, subject, body string) {
	ctx, cancel := context.WithTimeout(context.Background(), emailTimeout)
	defer cancel()
	if err := a.mail.Send(ctx, to, subject, body); err != nil {
		log.Printf("handler: could not email a registration decision: %v", err)
	}
}

func firstName(name string) string {
	parts := strings.Fields(name)
	if len(parts) == 0 {
		return "there"
	}
	return parts[0]
}

func welcomeEmailBody(name, regNo, origin string) string {
	site := strings.TrimSuffix(strings.TrimSpace(origin), "/")
	if site == "" {
		site = "the portal"
	}
	return strings.Join([]string{
		"Hello " + firstName(name) + ",",
		"",
		"Your SUST CSE Portal account is approved. Welcome aboard.",
		"",
		"Sign in with your registration number " + regNo + " and the password you chose while",
		"registering. From there you get your routine, your exams, the notices of your batch,",
		"your attendance and the messages your teachers and class representatives send.",
		"",
		"The website: " + site,
		"The Android app: " + site + DownloadAndroidPath,
		"",
		"Attendance is marked from the Android app on the first phone you sign in with, so",
		"install it before your next class.",
		"",
		"SUST CSE Portal",
		"Department of Computer Science and Engineering, Shahjalal University of Science and Technology",
	}, "\n")
}

func rejectedEmailBody(name, reason, origin string) string {
	site := strings.TrimSuffix(strings.TrimSpace(origin), "/")
	if site == "" {
		site = "the portal"
	}
	return strings.Join([]string{
		"Hello " + firstName(name) + ",",
		"",
		"Your SUST CSE Portal registration was not approved.",
		"",
		"Why: " + reason,
		"",
		"What to do next: if this is something you can correct, register again at",
		site + "/student/register with the right details and a clear photo. If you believe the",
		"decision is a mistake, visit the department office with your student ID and they will",
		"check it for you.",
		"",
		"SUST CSE Portal",
		"Department of Computer Science and Engineering, Shahjalal University of Science and Technology",
	}, "\n")
}

package handler

import (
	"errors"
	"github.com/raHI-79/250-project/backend/internal/auth"
	"github.com/raHI-79/250-project/backend/internal/googleauth"
	"github.com/raHI-79/250-project/backend/internal/store"
	"net/http"
	"strings"
	"testing"
)

func TestAuthConfig(t *testing.T) {
	silenceLog(t)
	tests := []struct {
		name        string
		clientID    string
		wantGoogle  bool
		wantStudent string
		wantTeacher string
	}{
		{
			name:        "google configured",
			clientID:    googleClientID,
			wantGoogle:  true,
			wantStudent: studentDomain,
			wantTeacher: teacherDomain,
		},
		{
			name:        "google missing",
			clientID:    "",
			wantGoogle:  false,
			wantStudent: studentDomain,
			wantTeacher: teacherDomain,
		},
		{
			name:        "google set to spaces",
			clientID:    "   ",
			wantGoogle:  false,
			wantStudent: studentDomain,
			wantTeacher: teacherDomain,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cfg := accountConfig(t, "test")
			cfg.GoogleClientID = test.clientID
			handler := NewAuth(cfg, newAccountStore(), &fakeIssuer{token: issuedToken}, &fakeMailer{},
				testProofs(t, auth.ProofTTL), &fakeGoogle{}, testPhotos(t))
			rec := serve(handler.Config(), jsonRequest(t, http.MethodGet, "/api/auth/config", "", 0, ""))
			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200 (body = %q)", rec.Code, rec.Body.String())
			}
			payload := decodeBody(t, rec)
			if payload["student_email_domain"] != test.wantStudent {
				t.Errorf("student_email_domain = %v, want %q", payload["student_email_domain"], test.wantStudent)
			}
			if payload["teacher_email_domain"] != test.wantTeacher {
				t.Errorf("teacher_email_domain = %v, want %q", payload["teacher_email_domain"], test.wantTeacher)
			}
			if payload["google_enabled"] != test.wantGoogle {
				t.Errorf("google_enabled = %v, want %v", payload["google_enabled"], test.wantGoogle)
			}
			if strings.Contains(rec.Body.String(), test.clientID) && test.clientID != "" {
				t.Error("the public config leaks the Google client id")
			}
		})
	}
}

func TestGoogleVerifyEmail(t *testing.T) {
	silenceLog(t)
	const studentEmail = "2023331079@student.sust.edu"
	tests := []struct {
		name       string
		body       string
		identity   googleauth.Identity
		googleErr  error
		clientID   string
		wantStatus int
		wantError  string
	}{
		{
			name:       "verified student account",
			body:       `{"credential":"a-google-credential"}`,
			identity:   googleauth.Identity{Email: studentEmail, EmailVerified: true, HostedDomain: studentDomain},
			clientID:   googleClientID,
			wantStatus: http.StatusOK,
		},
		{
			name:       "upper case address",
			body:       `{"credential":"a-google-credential"}`,
			identity:   googleauth.Identity{Email: "2023331079@Student.SUST.edu", EmailVerified: true, HostedDomain: studentDomain},
			clientID:   googleClientID,
			wantStatus: http.StatusOK,
		},
		{
			name:       "unverified email",
			body:       `{"credential":"a-google-credential"}`,
			identity:   googleauth.Identity{Email: studentEmail, EmailVerified: false, HostedDomain: studentDomain},
			clientID:   googleClientID,
			wantStatus: http.StatusBadRequest,
			wantError:  "Use your @student.sust.edu Google account.",
		},
		{
			name:       "another hosted domain",
			body:       `{"credential":"a-google-credential"}`,
			identity:   googleauth.Identity{Email: studentEmail, EmailVerified: true, HostedDomain: "sust.edu"},
			clientID:   googleClientID,
			wantStatus: http.StatusBadRequest,
			wantError:  "Use your @student.sust.edu Google account.",
		},
		{
			name:       "no hosted domain",
			body:       `{"credential":"a-google-credential"}`,
			identity:   googleauth.Identity{Email: studentEmail, EmailVerified: true},
			clientID:   googleClientID,
			wantStatus: http.StatusBadRequest,
			wantError:  "Use your @student.sust.edu Google account.",
		},
		{
			name:       "address outside the student domain",
			body:       `{"credential":"a-google-credential"}`,
			identity:   googleauth.Identity{Email: "ratul@gmail.com", EmailVerified: true, HostedDomain: studentDomain},
			clientID:   googleClientID,
			wantStatus: http.StatusBadRequest,
			wantError:  "Use your @student.sust.edu Google account.",
		},
		{
			name:       "wrong audience",
			body:       `{"credential":"a-google-credential"}`,
			identity:   googleauth.Identity{Email: studentEmail, EmailVerified: true, HostedDomain: studentDomain},
			clientID:   "another-client-id.apps.googleusercontent.com",
			wantStatus: http.StatusBadRequest,
			wantError:  "Use your @student.sust.edu Google account.",
		},
		{
			name:       "google refuses the credential",
			body:       `{"credential":"a-google-credential"}`,
			googleErr:  googleauth.ErrInvalidToken,
			clientID:   googleClientID,
			wantStatus: http.StatusBadRequest,
			wantError:  "Use your @student.sust.edu Google account.",
		},
		{
			name:       "google sign-in is off",
			body:       `{"credential":"a-google-credential"}`,
			clientID:   "",
			wantStatus: http.StatusBadRequest,
			wantError:  MessageGoogleDisabled,
		},
		{
			name:       "missing credential",
			body:       `{"credential":"  "}`,
			clientID:   googleClientID,
			wantStatus: http.StatusBadRequest,
			wantError:  MessageGoogleCredential,
		},
		{
			name:       "invalid body",
			body:       `{"credential":}`,
			clientID:   googleClientID,
			wantStatus: http.StatusBadRequest,
			wantError:  "Invalid request body",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cfg := accountConfig(t, "test")
			cfg.GoogleClientID = test.clientID
			google := &fakeGoogle{identity: test.identity, err: test.googleErr}
			proofs := testProofs(t, auth.ProofTTL)
			handler := NewAuth(cfg, newAccountStore(), &fakeIssuer{token: issuedToken}, &fakeMailer{},
				proofs, google, testPhotos(t))
			request := jsonRequest(t, http.MethodPost, "/api/auth/google/verify-email", test.body, 0, "")
			rec := serve(handler.GoogleVerifyEmail(), request)
			if rec.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d (body = %q)", rec.Code, test.wantStatus, rec.Body.String())
			}
			if test.wantError != "" {
				if got := errorMessage(t, rec); got != test.wantError {
					t.Errorf("error = %q, want %q", got, test.wantError)
				}
				if strings.Contains(rec.Body.String(), "email_proof") {
					t.Error("the refused response carries an email proof")
				}
				return
			}
			if google.calls != 1 || google.audience != test.clientID {
				t.Errorf("the validator was called %d times with audience %q, want once with %q",
					google.calls, google.audience, test.clientID)
			}
			if got := successField(t, rec, "email"); got != studentEmail {
				t.Errorf("email = %q, want %q", got, studentEmail)
			}
			proof := successField(t, rec, "email_proof")
			if err := proofs.CheckEmailProof(proof, studentEmail); err != nil {
				t.Errorf("the issued proof does not check out: %v", err)
			}
			if err := proofs.CheckEmailProof(proof, "2023331080@student.sust.edu"); err == nil {
				t.Error("the proof is accepted for another email address")
			}
		})
	}
}

func TestDropCourseOptions(t *testing.T) {
	silenceLog(t)
	tests := []struct {
		name       string
		query      string
		courses    []store.Course
		searchErr  error
		wantStatus int
		wantCount  int
	}{
		{
			name:       "two matches",
			query:      "?search=cse",
			courses:    []store.Course{{ID: 4, Code: "CSE-233", Title: "Algorithms"}, {ID: 9, Code: "CSE-245", Title: "Databases"}},
			wantStatus: http.StatusOK,
			wantCount:  2,
		},
		{name: "no matches", query: "?search=zzz", wantStatus: http.StatusOK, wantCount: 0},
		{name: "no search at all", query: "", wantStatus: http.StatusOK, wantCount: 0},
		{
			name:       "store failure",
			query:      "?search=cse",
			searchErr:  errors.New("database is down"),
			wantStatus: http.StatusInternalServerError,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			users := newAccountStore()
			users.dropCourses = test.courses
			users.searchErr = test.searchErr
			handler := newAccountAuth(t, users, testPhotos(t), &fakeGoogle{}, &fakeMailer{})
			request := jsonRequest(t, http.MethodGet, "/api/auth/drop-course-options"+test.query, "", 0, "")
			rec := serve(handler.DropCourseOptions(), request)
			if rec.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d (body = %q)", rec.Code, test.wantStatus, rec.Body.String())
			}
			if test.wantStatus != http.StatusOK {
				return
			}
			list := items(t, rec)
			if len(list) != test.wantCount {
				t.Fatalf("items = %d, want %d", len(list), test.wantCount)
			}
			for _, item := range list {
				if item["code"] == nil || item["title"] == nil || item["id"] == nil {
					t.Errorf("item = %v, want an id, a code and a title", item)
				}
			}
		})
	}
}

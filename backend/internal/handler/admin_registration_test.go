package handler

import (
	"net/http"
	"strings"
	"testing"

	"github.com/raHI-79/250-project/backend/internal/store"
)

func registrationHandler(t *testing.T, people *fakeAdminStore) (*Admin, *fakeMailer) {
	t.Helper()
	mail := &fakeMailer{configured: true, notify: make(chan sentMail, 4)}
	cfg := accountConfig(t, "test")
	cfg.FrontendOrigin = "https://portal.test"
	return NewAdmin(cfg, people, &fakeNotifier{}, mail, testPhotos(t)), mail
}

func TestApprovingARegistrationWelcomesTheStudent(t *testing.T) {
	silenceLog(t)
	people := &fakeAdminStore{}
	admin, mail := registrationHandler(t, people)
	rec := adminCall(t, admin.ApproveRegistration(), http.MethodPost,
		"/api/admin/registrations/7/approve", "7", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body = %q)", rec.Code, rec.Body.String())
	}
	if people.statuses[7] != store.StatusActive {
		t.Errorf("status = %q, want active", people.statuses[7])
	}
	sent := mail.waitForMail(t)
	if sent.to != "2023331099@student.sust.edu" {
		t.Errorf("to = %q, want the registered address", sent.to)
	}
	if sent.subject != WelcomeEmailSubject {
		t.Errorf("subject = %q, want %q", sent.subject, WelcomeEmailSubject)
	}
	wants := []string{
		"Pending",
		"approved",
		"2023331099",
		"https://portal.test",
		"https://portal.test" + DownloadAndroidPath,
	}
	for _, want := range wants {
		if !strings.Contains(sent.body, want) {
			t.Errorf("the welcome email does not mention %q: %q", want, sent.body)
		}
	}
}

func TestRejectingARegistrationNeedsAReason(t *testing.T) {
	silenceLog(t)
	tests := []struct {
		name   string
		body   string
		status int
	}{
		{name: "no body at all", status: http.StatusBadRequest},
		{name: "empty reason", body: `{"reason":""}`, status: http.StatusBadRequest},
		{name: "four characters", body: `{"reason":"abcd"}`, status: http.StatusBadRequest},
		{name: "only spaces", body: `{"reason":"        "}`, status: http.StatusBadRequest},
		{name: "five characters", body: `{"reason":"blurry"}`, status: http.StatusOK},
		{name: "a quick pick", body: `{"reason":"Not a CSE student"}`, status: http.StatusOK},
		{
			name:   "far too long",
			body:   `{"reason":"` + strings.Repeat("x", MaxRejectionReason+1) + `"}`,
			status: http.StatusBadRequest,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			people := &fakeAdminStore{}
			admin, _ := registrationHandler(t, people)
			rec := adminCall(t, admin.RejectRegistration(), http.MethodPost,
				"/api/admin/registrations/7/reject", "7", tc.body)
			if rec.Code != tc.status {
				t.Fatalf("status = %d, want %d (body = %q)", rec.Code, tc.status, rec.Body.String())
			}
			if tc.status != http.StatusOK {
				if got := errorOf(t, rec); got == "" {
					t.Error("the refusal carries no message")
				}
				if _, decided := people.statuses[7]; decided {
					t.Error("the registration was decided although the reason was refused")
				}
				return
			}
			if people.statuses[7] != store.StatusRejected {
				t.Errorf("status = %q, want rejected", people.statuses[7])
			}
		})
	}
}

func TestRejectingARegistrationEmailsTheReason(t *testing.T) {
	silenceLog(t)
	people := &fakeAdminStore{}
	admin, mail := registrationHandler(t, people)
	rec := adminCall(t, admin.RejectRegistration(), http.MethodPost,
		"/api/admin/registrations/7/reject", "7",
		`{"reason":"Registration number does not match"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body = %q)", rec.Code, rec.Body.String())
	}
	if people.reasons[7] != "Registration number does not match" {
		t.Errorf("the stored reason = %q", people.reasons[7])
	}
	sent := mail.waitForMail(t)
	if sent.subject != RejectedEmailSubject {
		t.Errorf("subject = %q, want %q", sent.subject, RejectedEmailSubject)
	}
	wants := []string{
		"Registration number does not match",
		"register again",
		"department office",
		"https://portal.test/student/register",
	}
	for _, want := range wants {
		if !strings.Contains(sent.body, want) {
			t.Errorf("the rejection email does not mention %q: %q", want, sent.body)
		}
	}
}

func TestADecisionSurvivesAMailerThatFails(t *testing.T) {
	silenceLog(t)
	people := &fakeAdminStore{}
	mail := &fakeMailer{configured: false}
	cfg := accountConfig(t, "test")
	admin := NewAdmin(cfg, people, &fakeNotifier{}, mail, testPhotos(t))
	rec := adminCall(t, admin.ApproveRegistration(), http.MethodPost,
		"/api/admin/registrations/7/approve", "7", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 even without a mailer (body = %q)", rec.Code, rec.Body.String())
	}
	if people.statuses[7] != store.StatusActive {
		t.Errorf("status = %q, want active", people.statuses[7])
	}
	if len(mail.delivered()) != 0 {
		t.Errorf("an unconfigured mailer still sent %v", mail.delivered())
	}
}

func TestRejectionReasonsAreOffered(t *testing.T) {
	people := &fakeAdminStore{}
	admin, _ := registrationHandler(t, people)
	rec := adminCall(t, admin.RejectionReasons(), http.MethodGet,
		"/api/admin/rejection-reasons", "", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range QuickRejectionReasons {
		if !strings.Contains(body, want) {
			t.Errorf("the quick picks do not carry %q: %q", want, body)
		}
	}
	if !strings.Contains(body, `"min":5`) {
		t.Errorf("the answer does not carry the minimum length: %q", body)
	}
}

func TestRejectionReasonIsTidiedUp(t *testing.T) {
	tests := []struct {
		name  string
		given string
		want  string
		fails bool
	}{
		{name: "plain", given: "Not a CSE student", want: "Not a CSE student"},
		{name: "extra spaces", given: "  Photo   is not clear  ", want: "Photo is not clear"},
		{name: "newlines", given: "Photo\nis\nnot\nclear", want: "Photo is not clear"},
		{name: "too short", given: "bad", fails: true},
		{name: "blank", given: "   ", fails: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := rejectionReason(tc.given)
			if tc.fails {
				if err == nil {
					t.Fatalf("rejectionReason(%q) = %q, want a refusal", tc.given, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("rejectionReason(%q) returned %v", tc.given, err)
			}
			if got != tc.want {
				t.Errorf("rejectionReason(%q) = %q, want %q", tc.given, got, tc.want)
			}
		})
	}
}

package handler

import (
	"github.com/raHI-79/250-project/backend/internal/store"
	"net/http"
	"strings"
	"testing"
)

const (
	inviteToken     = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	inviteTeacherID = "T-100"
	inviteBirthDate = "1988-03-09"
)

func plannedInvite(users *accountStore, attempts int) {
	users.invites[hashResetSecret(inviteToken)] = store.TeacherInvite{
		UserID:      6,
		TeacherID:   inviteTeacherID,
		Name:        "Invited Teacher",
		Designation: "Assistant Professor",
		Email:       "invited@sust.edu",
		DateOfBirth: inviteBirthDate,
		Attempts:    attempts,
	}
}

func TestInviteLookup(t *testing.T) {
	silenceLog(t)
	tests := []struct {
		name       string
		query      string
		wantStatus int
		wantError  string
	}{
		{name: "valid token", query: "?token=" + inviteToken, wantStatus: http.StatusOK},
		{name: "unknown token", query: "?token=deadbeef", wantStatus: http.StatusBadRequest, wantError: MessageInviteInvalid},
		{name: "no token", query: "", wantStatus: http.StatusBadRequest, wantError: MessageInviteInvalid},
		{name: "blank token", query: "?token=%20", wantStatus: http.StatusBadRequest, wantError: MessageInviteInvalid},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			users := newAccountStore()
			plannedInvite(users, 0)
			handler := newAccountAuth(t, users, testPhotos(t), &fakeGoogle{}, &fakeMailer{})
			rec := serve(handler.Invite(), jsonRequest(t, http.MethodGet, "/api/auth/invite"+test.query, "", 0, ""))
			if rec.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d (body = %q)", rec.Code, test.wantStatus, rec.Body.String())
			}
			if test.wantError != "" {
				if got := errorMessage(t, rec); got != test.wantError {
					t.Errorf("error = %q, want %q", got, test.wantError)
				}
				return
			}
			payload := decodeBody(t, rec)
			if payload["name"] != "Invited Teacher" || payload["designation"] != "Assistant Professor" {
				t.Errorf("payload = %v, want the name and designation", payload)
			}
			if payload["masked_email"] != "i***@sust.edu" {
				t.Errorf("masked_email = %v, want i***@sust.edu", payload["masked_email"])
			}
			if strings.Contains(rec.Body.String(), inviteBirthDate) {
				t.Error("the invitation lookup leaks the birth date")
			}
			if strings.Contains(rec.Body.String(), inviteTeacherID) {
				t.Error("the invitation lookup leaks the teacher id")
			}
			if strings.Contains(rec.Body.String(), "invited@sust.edu") {
				t.Error("the invitation lookup leaks the full email address")
			}
		})
	}
}

func TestActivateTeacher(t *testing.T) {
	silenceLog(t)
	activation := func() map[string][]string {
		return map[string][]string{
			"token":            {inviteToken},
			"teacher_id":       {inviteTeacherID},
			"date_of_birth":    {inviteBirthDate},
			"password":         {"sust-250-project"},
			"confirm_password": {"sust-250-project"},
		}
	}
	tests := []struct {
		name        string
		values      map[string][]string
		attempts    int
		withPhoto   bool
		wantStatus  int
		wantError   string
		wantActive  bool
		wantCleared bool
	}{
		{name: "right answers", values: activation(), wantStatus: http.StatusOK, wantActive: true},
		{name: "with a photo", values: activation(), withPhoto: true, wantStatus: http.StatusOK, wantActive: true},
		{
			name:       "wrong teacher id",
			values:     withField(activation(), "teacher_id", "T-999"),
			wantStatus: http.StatusBadRequest,
			wantError:  MessageInviteMismatch,
		},
		{
			name:       "wrong birth date",
			values:     withField(activation(), "date_of_birth", "1990-01-01"),
			wantStatus: http.StatusBadRequest,
			wantError:  MessageInviteMismatch,
		},
		{
			name:       "missing birth date",
			values:     withField(activation(), "date_of_birth"),
			wantStatus: http.StatusBadRequest,
			wantError:  MessageInviteMismatch,
		},
		{
			name:        "the fifth wrong answer locks the invitation",
			values:      withField(activation(), "teacher_id", "T-999"),
			attempts:    4,
			wantStatus:  http.StatusBadRequest,
			wantError:   MessageInviteLocked,
			wantCleared: true,
		},
		{
			name:       "unknown token",
			values:     withField(activation(), "token", "deadbeef"),
			wantStatus: http.StatusBadRequest,
			wantError:  MessageInviteInvalid,
		},
		{
			name:       "no token",
			values:     withField(activation(), "token"),
			wantStatus: http.StatusBadRequest,
			wantError:  MessageInviteInvalid,
		},
		{
			name:       "passwords do not match",
			values:     withField(activation(), "confirm_password", "another-password"),
			wantStatus: http.StatusBadRequest,
			wantError:  "Password and the confirmation do not match.",
		},
		{
			name:       "short password",
			values:     withField(withField(activation(), "password", "short"), "confirm_password", "short"),
			wantStatus: http.StatusBadRequest,
			wantError:  "Password must be at least 8 characters long.",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			users := newAccountStore()
			plannedInvite(users, test.attempts)
			users.inviteTries = test.attempts
			photos := testPhotos(t)
			handler := newAccountAuth(t, users, photos, &fakeGoogle{}, &fakeMailer{})
			files := []formFile{}
			if test.withPhoto {
				files = append(files, formFile{field: "photo", name: "teacher.jpg", data: jpegPhoto(t, 300, 300)})
			}
			request := multipartRequest(t, http.MethodPost, "/api/auth/activate", test.values, files...)
			rec := serve(handler.Activate(), request)
			if rec.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d (body = %q)", rec.Code, test.wantStatus, rec.Body.String())
			}
			if test.wantError != "" {
				if got := errorMessage(t, rec); got != test.wantError {
					t.Errorf("error = %q, want %q", got, test.wantError)
				}
			}
			if active := len(users.activated) == 1; active != test.wantActive {
				t.Errorf("activated = %v, want %v", active, test.wantActive)
			}
			if cleared := len(users.clearedInvite) == 1; cleared != test.wantCleared {
				t.Errorf("invitation cleared = %v, want %v", cleared, test.wantCleared)
			}
			if !test.wantActive {
				if savedPhotos(t, photos) != 0 {
					t.Error("a photo stayed on disk although the activation failed")
				}
				return
			}
			if got := successMessage(t, rec); got != MessageTeacherActivated {
				t.Errorf("message = %q, want %q", got, MessageTeacherActivated)
			}
			if strings.Contains(rec.Body.String(), "sust-250-project") {
				t.Error("the activation response leaks the password")
			}
			wantPhotos := 0
			if test.withPhoto {
				wantPhotos = 1
			}
			if savedPhotos(t, photos) != wantPhotos {
				t.Errorf("%d photos are on disk, want %d", savedPhotos(t, photos), wantPhotos)
			}
		})
	}
}

func TestActivateTeacherRemovesThePhotoWhenTheStoreFails(t *testing.T) {
	silenceLog(t)
	users := newAccountStore()
	plannedInvite(users, 0)
	users.activateErr = store.ErrNotFound
	photos := testPhotos(t)
	handler := newAccountAuth(t, users, photos, &fakeGoogle{}, &fakeMailer{})
	values := map[string][]string{
		"token":            {inviteToken},
		"teacher_id":       {inviteTeacherID},
		"date_of_birth":    {inviteBirthDate},
		"password":         {"sust-250-project"},
		"confirm_password": {"sust-250-project"},
	}
	request := multipartRequest(t, http.MethodPost, "/api/auth/activate", values,
		formFile{field: "photo", name: "teacher.png", data: pngPhoto(t, 220, 220)})
	rec := serve(handler.Activate(), request)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 (body = %q)", rec.Code, rec.Body.String())
	}
	if got := errorMessage(t, rec); got != MessageInviteInvalid {
		t.Errorf("error = %q, want %q", got, MessageInviteInvalid)
	}
	if savedPhotos(t, photos) != 0 {
		t.Error("the photo stayed on disk although the activation failed")
	}
}

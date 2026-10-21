package handler

import (
	"github.com/raHI-79/250-project/backend/internal/store"
	"github.com/raHI-79/250-project/backend/internal/validate"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

const (
	inviteToken     = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	inviteBirthDate = "1988-03-09"
	chosenUserID    = "karim.cse"
)

func plannedInvite(users *accountStore, attempts int) {
	users.invites[hashResetSecret(inviteToken)] = store.TeacherInvite{
		UserID:      6,
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
			if strings.Contains(rec.Body.String(), "invited@sust.edu") {
				t.Error("the invitation lookup leaks the full email address")
			}
		})
	}
}

func TestUserIDAvailable(t *testing.T) {
	silenceLog(t)
	tests := []struct {
		name      string
		userID    string
		taken     string
		wantFree  bool
		wantWhyNo string
	}{
		{name: "free", userID: chosenUserID, wantFree: true},
		{name: "letters digits and marks", userID: "k_rahman-2", wantFree: true},
		{name: "mixed case of a free id", userID: "Karim.CSE", wantFree: true},
		{name: "missing", userID: "", wantWhyNo: validate.MessageUserIDRequired},
		{name: "too short", userID: "ab1", wantWhyNo: validate.MessageUserIDLength},
		{name: "too long", userID: "karim" + strings.Repeat("x", 16), wantWhyNo: validate.MessageUserIDLength},
		{name: "a space inside", userID: "karim cse", wantWhyNo: validate.MessageUserIDCharacters},
		{name: "an at sign", userID: "karim@cse", wantWhyNo: validate.MessageUserIDCharacters},
		{name: "digits only", userID: "20233310", wantWhyNo: validate.MessageUserIDDigitsOnly},
		{name: "starts with a digit", userID: "1karim", wantWhyNo: validate.MessageUserIDStart},
		{name: "starts with a dot", userID: ".karim", wantWhyNo: validate.MessageUserIDStart},
		{name: "reserved", userID: "admin", wantWhyNo: validate.MessageUserIDReserved},
		{name: "reserved in another case", userID: "Student", wantWhyNo: validate.MessageUserIDReserved},
		{name: "taken", userID: chosenUserID, taken: chosenUserID, wantWhyNo: MessageUserIDTaken},
		{name: "taken in another case", userID: "KARIM.CSE", taken: chosenUserID, wantWhyNo: MessageUserIDTaken},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			users := newAccountStore()
			if test.taken != "" {
				users.takenIDs[strings.ToLower(test.taken)] = true
			}
			handler := newAccountAuth(t, users, testPhotos(t), &fakeGoogle{}, &fakeMailer{})
			target := "/api/auth/user-id-available?user_id=" + url.QueryEscape(test.userID)
			rec := serve(handler.UserIDAvailable(), jsonRequest(t, http.MethodGet, target, "", 0, ""))
			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200 (body = %q)", rec.Code, rec.Body.String())
			}
			payload := decodeBody(t, rec)
			if payload["available"] != test.wantFree {
				t.Errorf("available = %v, want %v", payload["available"], test.wantFree)
			}
			if test.wantFree {
				if _, present := payload["reason"]; present {
					t.Errorf("a free user id carries the reason %v", payload["reason"])
				}
				return
			}
			if got := payload["reason"]; got != test.wantWhyNo {
				t.Errorf("reason = %v, want %q", got, test.wantWhyNo)
			}
		})
	}
}

func TestActivateTeacher(t *testing.T) {
	silenceLog(t)
	activation := func() map[string][]string {
		return map[string][]string{
			"token":            {inviteToken},
			"user_id":          {chosenUserID},
			"date_of_birth":    {inviteBirthDate},
			"password":         {"sust-250-project"},
			"confirm_password": {"sust-250-project"},
		}
	}
	tests := []struct {
		name        string
		values      map[string][]string
		attempts    int
		taken       string
		withPhoto   bool
		wantStatus  int
		wantError   string
		wantActive  bool
		wantCleared bool
	}{
		{name: "right answers", values: activation(), wantStatus: http.StatusOK, wantActive: true},
		{name: "with a photo", values: activation(), withPhoto: true, wantStatus: http.StatusOK, wantActive: true},
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
			values:      withField(activation(), "date_of_birth", "1990-01-01"),
			attempts:    4,
			wantStatus:  http.StatusBadRequest,
			wantError:   MessageInviteLocked,
			wantCleared: true,
		},
		{
			name:       "no user id",
			values:     withField(activation(), "user_id"),
			wantStatus: http.StatusBadRequest,
			wantError:  validate.MessageUserIDRequired,
		},
		{
			name:       "a user id against the rules",
			values:     withField(activation(), "user_id", "1karim"),
			wantStatus: http.StatusBadRequest,
			wantError:  validate.MessageUserIDStart,
		},
		{
			name:       "a reserved user id",
			values:     withField(activation(), "user_id", "office"),
			wantStatus: http.StatusBadRequest,
			wantError:  validate.MessageUserIDReserved,
		},
		{
			name:       "a taken user id",
			values:     activation(),
			taken:      chosenUserID,
			wantStatus: http.StatusConflict,
			wantError:  MessageUserIDTaken,
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
			if test.taken != "" {
				users.takenIDs[strings.ToLower(test.taken)] = true
			}
			photos := testPhotos(t)
			mail := &fakeMailer{configured: true}
			handler := newAccountAuth(t, users, photos, &fakeGoogle{}, mail)
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
			if got := users.chosenIDs[0]; got != chosenUserID {
				t.Errorf("stored user id = %q, want %q", got, chosenUserID)
			}
			payload := decodeBody(t, rec)
			if payload["user_id"] != chosenUserID {
				t.Errorf("user_id = %v, want %q", payload["user_id"], chosenUserID)
			}
			if payload["name"] != "Invited Teacher" {
				t.Errorf("name = %v, want the invited teacher", payload["name"])
			}
			if payload["message"] != MessageTeacherActivated {
				t.Errorf("message = %v, want %q", payload["message"], MessageTeacherActivated)
			}
			if strings.Contains(rec.Body.String(), "sust-250-project") {
				t.Error("the activation response leaks the password")
			}
			if len(mail.sent) != 1 || mail.sent[0].to != "invited@sust.edu" {
				t.Fatalf("sent mail = %+v, want one message to the teacher", mail.sent)
			}
			if !strings.Contains(mail.sent[0].body, chosenUserID) {
				t.Errorf("mail body = %q, want the chosen user id", mail.sent[0].body)
			}
			if strings.Contains(mail.sent[0].body, "sust-250-project") {
				t.Error("the confirmation email leaks the password")
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
		"user_id":          {chosenUserID},
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

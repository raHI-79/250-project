package handler

import (
	"github.com/raHI-79/250-project/backend/internal/auth"
	"github.com/raHI-79/250-project/backend/internal/store"
	"github.com/raHI-79/250-project/backend/internal/upload"
	"github.com/raHI-79/250-project/backend/internal/validate"
	"net/http"
	"strings"
	"testing"
	"time"
)

func registrationForm() map[string][]string {
	return map[string][]string{
		"full_name":        {"Ratul Islam"},
		"reg_no":           {"2023331079"},
		"batch":            {"CSE-23"},
		"section":          {"A"},
		"email":            {"2023331079@student.sust.edu"},
		"phone":            {"01712345678"},
		"blood_group":      {"O+"},
		"date_of_birth":    {"2003-05-14"},
		"password":         {"sust-250-project"},
		"confirm_password": {"sust-250-project"},
	}
}

func withField(values map[string][]string, name string, replacement ...string) map[string][]string {
	copied := make(map[string][]string, len(values))
	for key, value := range values {
		copied[key] = value
	}
	if len(replacement) == 0 {
		delete(copied, name)
		return copied
	}
	copied[name] = replacement
	return copied
}

func TestRegisterValidation(t *testing.T) {
	silenceLog(t)
	tests := []struct {
		name      string
		values    map[string][]string
		photo     []byte
		photoName string
		noPhoto   bool
		wantCode  int
		wantError string
	}{
		{name: "complete form", values: registrationForm(), wantCode: http.StatusCreated},
		{
			name:      "email outside the student domain",
			values:    withField(registrationForm(), "email", "ratul@gmail.com"),
			wantCode:  http.StatusBadRequest,
			wantError: "Email must end with @student.sust.edu.",
		},
		{
			name:      "teacher domain is refused",
			values:    withField(registrationForm(), "email", "ratul@sust.edu"),
			wantCode:  http.StatusBadRequest,
			wantError: "Email must end with @student.sust.edu.",
		},
		{
			name:      "phone with ten digits",
			values:    withField(registrationForm(), "phone", "1712345678"),
			wantCode:  http.StatusBadRequest,
			wantError: validate.MessagePhone,
		},
		{
			name:      "phone with a country code",
			values:    withField(registrationForm(), "phone", "+8801712345678"),
			wantCode:  http.StatusBadRequest,
			wantError: validate.MessagePhone,
		},
		{
			name:      "too young",
			values:    withField(registrationForm(), "date_of_birth", "2020-01-01"),
			wantCode:  http.StatusBadRequest,
			wantError: "Date of birth must belong to a person between 15 and 60 years old.",
		},
		{
			name:      "birth date in the wrong format",
			values:    withField(registrationForm(), "date_of_birth", "14-05-2003"),
			wantCode:  http.StatusBadRequest,
			wantError: "Date of birth must be a date in the YYYY-MM-DD format.",
		},
		{
			name:      "missing birth date",
			values:    withField(registrationForm(), "date_of_birth"),
			wantCode:  http.StatusBadRequest,
			wantError: "Date of birth is required.",
		},
		{
			name:      "passwords do not match",
			values:    withField(registrationForm(), "confirm_password", "another-password"),
			wantCode:  http.StatusBadRequest,
			wantError: "Password and the confirmation do not match.",
		},
		{
			name:      "bad batch",
			values:    withField(registrationForm(), "batch", "CSE23"),
			wantCode:  http.StatusBadRequest,
			wantError: "Batch must look like CSE-23.",
		},
		{
			name:      "bad blood group",
			values:    withField(registrationForm(), "blood_group", "C+"),
			wantCode:  http.StatusBadRequest,
			wantError: "Blood group must be one of A+, A-, B+, B-, AB+, AB-, O+, O-.",
		},
		{
			name:      "no photo",
			values:    registrationForm(),
			noPhoto:   true,
			wantCode:  http.StatusBadRequest,
			wantError: MessagePhotoRequired,
		},
		{
			name:      "the photo is a pdf",
			values:    registrationForm(),
			photo:     []byte("%PDF-1.7 not a photo"),
			photoName: "photo.png",
			wantCode:  http.StatusBadRequest,
			wantError: MessagePhotoNotImage,
		},
		{
			name:      "the photo is too small",
			values:    registrationForm(),
			photo:     pngPhoto(t, 120, 120),
			photoName: "photo.png",
			wantCode:  http.StatusBadRequest,
			wantError: MessagePhotoTooSmall,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			users := newAccountStore()
			photos := testPhotos(t)
			handler := newAccountAuth(t, users, photos, &fakeGoogle{}, &fakeMailer{})
			files := []formFile{}
			if !test.noPhoto {
				data := test.photo
				name := test.photoName
				if data == nil {
					data, name = jpegPhoto(t, 320, 320), "photo.jpg"
				}
				files = append(files, formFile{field: "photo", name: name, data: data})
			}
			request := multipartRequest(t, http.MethodPost, "/api/auth/register", test.values, files...)
			rec := serve(handler.Register(), request)
			if rec.Code != test.wantCode {
				t.Fatalf("status = %d, want %d (body = %q)", rec.Code, test.wantCode, rec.Body.String())
			}
			if test.wantError != "" {
				if got := errorMessage(t, rec); got != test.wantError {
					t.Errorf("error = %q, want %q", got, test.wantError)
				}
				if len(users.registrations) != 0 {
					t.Error("the store received a registration although the form was refused")
				}
				if savedPhotos(t, photos) != 0 {
					t.Error("a photo stayed on disk although the form was refused")
				}
				return
			}
			if len(users.registrations) != 1 {
				t.Fatalf("the store received %d registrations, want 1", len(users.registrations))
			}
			saved := users.registrations[0]
			if saved.Status != store.StatusUnverified {
				t.Errorf("status = %q, want %q", saved.Status, store.StatusUnverified)
			}
			if saved.PhotoPath == "" || savedPhotos(t, photos) != 1 {
				t.Error("the registration did not keep the uploaded photo")
			}
			if saved.PasswordHash == "" || strings.Contains(saved.PasswordHash, "sust-250-project") {
				t.Error("the store did not receive a hashed password")
			}
			if saved.CodeHash == "" || saved.CodeExpiresAt == nil || saved.CodeSentAt == nil {
				t.Error("the registration did not store an email code")
			}
			if strings.Contains(rec.Body.String(), "sust-250-project") ||
				strings.Contains(rec.Body.String(), saved.CodeHash) {
				t.Error("the response leaks the password or the stored code hash")
			}
			code := successField(t, rec, "dev_code")
			if !isSixDigitCode(code) {
				t.Errorf("dev_code = %q, want six digits", code)
			}
			if hashResetSecret(code) != saved.CodeHash {
				t.Error("the stored hash does not match the code in the response")
			}
		})
	}
}

func TestRegisterWithAnEmailProof(t *testing.T) {
	silenceLog(t)
	proofs := testProofs(t, 15*time.Minute)
	valid, _, err := proofs.IssueEmailProof("2023331079@student.sust.edu")
	if err != nil {
		t.Fatalf("IssueEmailProof() returned an unexpected error: %v", err)
	}
	other, _, err := proofs.IssueEmailProof("2023331080@student.sust.edu")
	if err != nil {
		t.Fatalf("IssueEmailProof() returned an unexpected error: %v", err)
	}
	foreign, err := auth.NewProofs("fedcba9876543210fedcba9876543210", 15*time.Minute)
	if err != nil {
		t.Fatalf("auth.NewProofs() returned an unexpected error: %v", err)
	}
	signedElsewhere, _, err := foreign.IssueEmailProof("2023331079@student.sust.edu")
	if err != nil {
		t.Fatalf("IssueEmailProof() returned an unexpected error: %v", err)
	}
	tests := []struct {
		name            string
		proof           string
		wantStatus      int
		wantStatusValue string
		wantError       string
	}{
		{
			name:            "valid proof skips the code",
			proof:           valid,
			wantStatus:      http.StatusCreated,
			wantStatusValue: store.StatusPending,
		},
		{
			name:       "proof for another email",
			proof:      other,
			wantStatus: http.StatusBadRequest,
			wantError:  MessageInvalidEmailProof,
		},
		{
			name:       "nonsense proof",
			proof:      "not-a-proof",
			wantStatus: http.StatusBadRequest,
			wantError:  MessageInvalidEmailProof,
		},
		{
			name:            "no proof falls back to the code",
			proof:           "",
			wantStatus:      http.StatusCreated,
			wantStatusValue: store.StatusUnverified,
		},
		{
			name:       "proof signed with another secret",
			proof:      signedElsewhere,
			wantStatus: http.StatusBadRequest,
			wantError:  MessageInvalidEmailProof,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			users := newAccountStore()
			photos := testPhotos(t)
			handler := newAccountAuth(t, users, photos, &fakeGoogle{}, &fakeMailer{})
			values := registrationForm()
			if test.proof != "" {
				values["email_proof"] = []string{test.proof}
			}
			request := multipartRequest(t, http.MethodPost, "/api/auth/register", values,
				formFile{field: "photo", name: "photo.png", data: pngPhoto(t, 240, 240)})
			rec := serve(handler.Register(), request)
			if rec.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d (body = %q)", rec.Code, test.wantStatus, rec.Body.String())
			}
			if test.wantError != "" {
				if got := errorMessage(t, rec); got != test.wantError {
					t.Errorf("error = %q, want %q", got, test.wantError)
				}
				if savedPhotos(t, photos) != 0 {
					t.Error("a photo stayed on disk although the proof was refused")
				}
				return
			}
			saved := users.registrations[0]
			if saved.Status != test.wantStatusValue {
				t.Errorf("status = %q, want %q", saved.Status, test.wantStatusValue)
			}
			if test.wantStatusValue == store.StatusPending {
				if saved.EmailVerifiedAt == nil {
					t.Error("the verified registration has no email_verified_at")
				}
				if saved.CodeHash != "" {
					t.Error("the verified registration still stores an email code")
				}
				if got := successMessage(t, rec); got != MessageRegistrationDone {
					t.Errorf("message = %q, want %q", got, MessageRegistrationDone)
				}
				return
			}
			if saved.EmailVerifiedAt != nil {
				t.Error("the unverified registration carries an email_verified_at")
			}
		})
	}
}

func TestRegisterDropCourses(t *testing.T) {
	silenceLog(t)
	tests := []struct {
		name       string
		ids        []string
		allowed    []int64
		wantStatus int
		wantIDs    []int64
	}{
		{name: "no drop course", ids: nil, wantStatus: http.StatusCreated},
		{name: "one allowed course", ids: []string{"4"}, allowed: []int64{4}, wantStatus: http.StatusCreated, wantIDs: []int64{4}},
		{
			name:       "two allowed courses",
			ids:        []string{"4", "9"},
			allowed:    []int64{4, 9},
			wantStatus: http.StatusCreated,
			wantIDs:    []int64{4, 9},
		},
		{
			name:       "the same course twice",
			ids:        []string{"4", "4"},
			allowed:    []int64{4},
			wantStatus: http.StatusCreated,
			wantIDs:    []int64{4},
		},
		{name: "a course outside the options", ids: []string{"4", "77"}, allowed: []int64{4}, wantStatus: http.StatusBadRequest},
		{name: "not a number", ids: []string{"abc"}, allowed: []int64{4}, wantStatus: http.StatusBadRequest},
		{name: "a negative id", ids: []string{"-4"}, allowed: []int64{4}, wantStatus: http.StatusBadRequest},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			users := newAccountStore()
			for _, courseID := range test.allowed {
				users.allowedIDs[courseID] = true
			}
			photos := testPhotos(t)
			handler := newAccountAuth(t, users, photos, &fakeGoogle{}, &fakeMailer{})
			values := registrationForm()
			if test.ids != nil {
				values["drop_course_ids"] = test.ids
			}
			request := multipartRequest(t, http.MethodPost, "/api/auth/register", values,
				formFile{field: "photo", name: "photo.png", data: pngPhoto(t, 240, 240)})
			rec := serve(handler.Register(), request)
			if rec.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d (body = %q)", rec.Code, test.wantStatus, rec.Body.String())
			}
			if test.wantStatus != http.StatusCreated {
				if got := errorMessage(t, rec); got != MessageUnknownDropCourse {
					t.Errorf("error = %q, want %q", got, MessageUnknownDropCourse)
				}
				if savedPhotos(t, photos) != 0 {
					t.Error("a photo stayed on disk although the drop course was refused")
				}
				return
			}
			saved := users.registrations[0].DropCourseIDs
			if len(saved) != len(test.wantIDs) {
				t.Fatalf("drop courses = %v, want %v", saved, test.wantIDs)
			}
			for i, want := range test.wantIDs {
				if saved[i] != want {
					t.Errorf("drop course %d = %d, want %d", i, saved[i], want)
				}
			}
		})
	}
}

func TestRegisterRemovesThePhotoWhenTheStoreFails(t *testing.T) {
	silenceLog(t)
	users := newAccountStore()
	users.createErr = store.ErrDuplicateAccount
	photos := testPhotos(t)
	handler := newAccountAuth(t, users, photos, &fakeGoogle{}, &fakeMailer{})
	request := multipartRequest(t, http.MethodPost, "/api/auth/register", registrationForm(),
		formFile{field: "photo", name: "photo.png", data: pngPhoto(t, 240, 240)})
	rec := serve(handler.Register(), request)
	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409 (body = %q)", rec.Code, rec.Body.String())
	}
	if got := errorMessage(t, rec); got != MessageDuplicateAccount {
		t.Errorf("error = %q, want %q", got, MessageDuplicateAccount)
	}
	if savedPhotos(t, photos) != 0 {
		t.Error("the photo of the refused registration stayed on disk")
	}
}

func TestRegisterDeletesTheReplacedPhoto(t *testing.T) {
	silenceLog(t)
	users := newAccountStore()
	photos := testPhotos(t)
	handler := newAccountAuth(t, users, photos, &fakeGoogle{}, &fakeMailer{})
	first := multipartRequest(t, http.MethodPost, "/api/auth/register", registrationForm(),
		formFile{field: "photo", name: "photo.png", data: pngPhoto(t, 240, 240)})
	if rec := serve(handler.Register(), first); rec.Code != http.StatusCreated {
		t.Fatalf("first register status = %d, want 201 (body = %q)", rec.Code, rec.Body.String())
	}
	users.result = store.RegistrationResult{UserID: 2, ReplacedPhotos: []string{users.registrations[0].PhotoPath}}
	second := multipartRequest(t, http.MethodPost, "/api/auth/register", registrationForm(),
		formFile{field: "photo", name: "photo.png", data: pngPhoto(t, 240, 240)})
	if rec := serve(handler.Register(), second); rec.Code != http.StatusCreated {
		t.Fatalf("second register status = %d, want 201 (body = %q)", rec.Code, rec.Body.String())
	}
	if savedPhotos(t, photos) != 1 {
		t.Errorf("%d photos are on disk, want only the newest one", savedPhotos(t, photos))
	}
}

func TestRegisterMailsTheCodeInProduction(t *testing.T) {
	silenceLog(t)
	users := newAccountStore()
	photos := testPhotos(t)
	mail := &fakeMailer{configured: true}
	handler := NewAuth(accountConfig(t, "production"), users, &fakeIssuer{token: issuedToken}, mail,
		testProofs(t, 15*time.Minute), &fakeGoogle{}, photos)
	request := multipartRequest(t, http.MethodPost, "/api/auth/register", registrationForm(),
		formFile{field: "photo", name: "photo.png", data: pngPhoto(t, 240, 240)})
	rec := serve(handler.Register(), request)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201 (body = %q)", rec.Code, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "dev_code") {
		t.Error("the production response carries a dev_code")
	}
	if len(mail.sent) != 1 || mail.sent[0].to != "2023331079@student.sust.edu" {
		t.Fatalf("sent mail = %+v, want one message to the student", mail.sent)
	}
	if !strings.Contains(mail.sent[0].body, "expires in 15 minutes") {
		t.Errorf("mail body = %q, want the lifetime of the code", mail.sent[0].body)
	}
}

func TestVerifyEmail(t *testing.T) {
	silenceLog(t)
	const regNo = "2023331079"
	tests := []struct {
		name        string
		body        string
		code        string
		status      string
		expiresIn   time.Duration
		attempts    int
		wantStatus  int
		wantError   string
		wantConfirm bool
		wantCleared bool
	}{
		{
			name:        "right code",
			body:        `{"reg_no":"` + regNo + `","code":"123456"}`,
			code:        "123456",
			status:      store.StatusUnverified,
			expiresIn:   10 * time.Minute,
			wantStatus:  http.StatusOK,
			wantConfirm: true,
		},
		{
			name:       "wrong code",
			body:       `{"reg_no":"` + regNo + `","code":"000000"}`,
			code:       "123456",
			status:     store.StatusUnverified,
			expiresIn:  10 * time.Minute,
			wantStatus: http.StatusBadRequest,
			wantError:  MessageInvalidEmailCode,
		},
		{
			name:        "the fifth wrong code clears the challenge",
			body:        `{"reg_no":"` + regNo + `","code":"000000"}`,
			code:        "123456",
			status:      store.StatusUnverified,
			expiresIn:   10 * time.Minute,
			attempts:    4,
			wantStatus:  http.StatusBadRequest,
			wantError:   MessageTooManyEmailCodes,
			wantCleared: true,
		},
		{
			name:       "expired code",
			body:       `{"reg_no":"` + regNo + `","code":"123456"}`,
			code:       "123456",
			status:     store.StatusUnverified,
			expiresIn:  -time.Minute,
			wantStatus: http.StatusBadRequest,
			wantError:  MessageInvalidEmailCode,
		},
		{
			name:       "already pending",
			body:       `{"reg_no":"` + regNo + `","code":"123456"}`,
			code:       "123456",
			status:     store.StatusPending,
			expiresIn:  10 * time.Minute,
			wantStatus: http.StatusBadRequest,
			wantError:  MessageInvalidEmailCode,
		},
		{
			name:       "unknown registration number",
			body:       `{"reg_no":"2023339999","code":"123456"}`,
			code:       "123456",
			status:     store.StatusUnverified,
			expiresIn:  10 * time.Minute,
			wantStatus: http.StatusBadRequest,
			wantError:  MessageInvalidEmailCode,
		},
		{
			name:       "letters instead of digits",
			body:       `{"reg_no":"` + regNo + `","code":"abcdef"}`,
			code:       "123456",
			status:     store.StatusUnverified,
			expiresIn:  10 * time.Minute,
			wantStatus: http.StatusBadRequest,
			wantError:  MessageInvalidEmailCode,
		},
		{
			name:       "five digits",
			body:       `{"reg_no":"` + regNo + `","code":"12345"}`,
			code:       "123456",
			status:     store.StatusUnverified,
			expiresIn:  10 * time.Minute,
			wantStatus: http.StatusBadRequest,
			wantError:  MessageInvalidEmailCode,
		},
		{
			name:       "missing registration number",
			body:       `{"reg_no":"  ","code":"123456"}`,
			code:       "123456",
			status:     store.StatusUnverified,
			expiresIn:  10 * time.Minute,
			wantStatus: http.StatusBadRequest,
			wantError:  "Registration number is required.",
		},
		{
			name:       "invalid body",
			body:       `{"reg_no":}`,
			code:       "123456",
			status:     store.StatusUnverified,
			expiresIn:  10 * time.Minute,
			wantStatus: http.StatusBadRequest,
			wantError:  "Invalid request body",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			users := newAccountStore()
			expiresAt := time.Now().Add(test.expiresIn)
			sentAt := time.Now().Add(-5 * time.Minute)
			users.challenges[regNo] = store.EmailChallenge{
				UserID:    11,
				Status:    test.status,
				Email:     "2023331079@student.sust.edu",
				CodeHash:  hashResetSecret(test.code),
				Attempts:  test.attempts,
				ExpiresAt: &expiresAt,
				SentAt:    &sentAt,
			}
			users.emailAttempts = test.attempts
			handler := newAccountAuth(t, users, testPhotos(t), &fakeGoogle{}, &fakeMailer{})
			rec := serve(handler.VerifyEmail(), jsonRequest(t, http.MethodPost, "/api/auth/verify-email", test.body, 0, ""))
			if rec.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d (body = %q)", rec.Code, test.wantStatus, rec.Body.String())
			}
			if test.wantError != "" {
				if got := errorMessage(t, rec); got != test.wantError {
					t.Errorf("error = %q, want %q", got, test.wantError)
				}
			}
			if confirmed := len(users.confirmed) == 1; confirmed != test.wantConfirm {
				t.Errorf("confirmed = %v, want %v", confirmed, test.wantConfirm)
			}
			if cleared := len(users.clearedCodes) == 1; cleared != test.wantCleared {
				t.Errorf("cleared = %v, want %v", cleared, test.wantCleared)
			}
			if test.wantConfirm {
				if got := successMessage(t, rec); got != MessageEmailVerified {
					t.Errorf("message = %q, want %q", got, MessageEmailVerified)
				}
			}
		})
	}
}

func TestResendVerification(t *testing.T) {
	silenceLog(t)
	const regNo = "2023331079"
	tests := []struct {
		name       string
		status     string
		sentAgo    time.Duration
		known      bool
		wantStatus int
		wantError  string
	}{
		{name: "after the cooldown", status: store.StatusUnverified, sentAgo: 2 * time.Minute, known: true, wantStatus: http.StatusOK},
		{
			name:       "inside the cooldown",
			status:     store.StatusUnverified,
			sentAgo:    20 * time.Second,
			known:      true,
			wantStatus: http.StatusTooManyRequests,
			wantError:  MessageVerificationCooldown,
		},
		{
			name:       "already pending",
			status:     store.StatusPending,
			sentAgo:    2 * time.Minute,
			known:      true,
			wantStatus: http.StatusBadRequest,
			wantError:  MessageAlreadyVerified,
		},
		{
			name:       "unknown registration number",
			status:     store.StatusUnverified,
			sentAgo:    2 * time.Minute,
			wantStatus: http.StatusBadRequest,
			wantError:  MessageAlreadyVerified,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			users := newAccountStore()
			if test.known {
				sentAt := time.Now().Add(-test.sentAgo)
				expiresAt := sentAt.Add(15 * time.Minute)
				users.challenges[regNo] = store.EmailChallenge{
					UserID:    11,
					Status:    test.status,
					Email:     "2023331079@student.sust.edu",
					CodeHash:  hashResetSecret("123456"),
					ExpiresAt: &expiresAt,
					SentAt:    &sentAt,
				}
			}
			handler := newAccountAuth(t, users, testPhotos(t), &fakeGoogle{}, &fakeMailer{})
			body := `{"reg_no":"` + regNo + `"}`
			rec := serve(handler.ResendVerification(), jsonRequest(t, http.MethodPost, "/api/auth/resend-verification", body, 0, ""))
			if rec.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d (body = %q)", rec.Code, test.wantStatus, rec.Body.String())
			}
			if test.wantError != "" {
				if got := errorMessage(t, rec); got != test.wantError {
					t.Errorf("error = %q, want %q", got, test.wantError)
				}
				if len(users.startedCodes) != 0 {
					t.Error("the store stored a new code although the request was refused")
				}
				return
			}
			if len(users.startedCodes) != 1 {
				t.Fatalf("the store received %d codes, want 1", len(users.startedCodes))
			}
			code := successField(t, rec, "dev_code")
			if hashResetSecret(code) != users.startedCodes[0].codeHash {
				t.Error("the stored hash does not match the code in the response")
			}
		})
	}
}

func TestPhotoErrorMapping(t *testing.T) {
	tests := []struct {
		name        string
		err         error
		wantStatus  int
		wantMessage string
	}{
		{name: "too large", err: upload.ErrPhotoTooLarge, wantStatus: http.StatusRequestEntityTooLarge, wantMessage: MessagePhotoTooLarge},
		{name: "too small", err: upload.ErrPhotoTooSmall, wantStatus: http.StatusBadRequest, wantMessage: MessagePhotoTooSmall},
		{name: "not an image", err: upload.ErrNotPhoto, wantStatus: http.StatusBadRequest, wantMessage: MessagePhotoNotImage},
		{name: "missing", err: upload.ErrMissingPhoto, wantStatus: http.StatusBadRequest, wantMessage: MessagePhotoRequired},
		{name: "broken form", err: upload.ErrBadForm, wantStatus: http.StatusBadRequest, wantMessage: "Invalid request body"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			status, message := photoError(test.err)
			if status != test.wantStatus || message != test.wantMessage {
				t.Errorf("photoError() = %d, %q, want %d, %q", status, message, test.wantStatus, test.wantMessage)
			}
		})
	}
}

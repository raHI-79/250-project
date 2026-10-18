package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/raHI-79/250-project/backend/internal/auth"
	"github.com/raHI-79/250-project/backend/internal/config"
	"github.com/raHI-79/250-project/backend/internal/googleauth"
	"github.com/raHI-79/250-project/backend/internal/middleware"
	"github.com/raHI-79/250-project/backend/internal/store"
	"github.com/raHI-79/250-project/backend/internal/upload"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io/fs"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

const (
	testSecret     = "0123456789abcdef0123456789abcdef"
	studentDomain  = "student.sust.edu"
	teacherDomain  = "sust.edu"
	googleClientID = "client-id.apps.googleusercontent.com"
)

type fakeGoogle struct {
	identity googleauth.Identity
	err      error
	audience string
	calls    int
}

func (f *fakeGoogle) Validate(ctx context.Context, credential, audience string) (googleauth.Identity, error) {
	f.calls++
	f.audience = audience
	if f.err != nil {
		return googleauth.Identity{}, f.err
	}
	if audience != googleClientID {
		return googleauth.Identity{}, googleauth.ErrInvalidToken
	}
	return f.identity, nil
}

func accountConfig(t *testing.T, appEnv string) *config.Config {
	t.Helper()
	loc, err := time.LoadLocation("Asia/Dhaka")
	if err != nil {
		t.Fatalf("could not load timezone: %v", err)
	}
	return &config.Config{
		AppEnv:         appEnv,
		Timezone:       "Asia/Dhaka",
		Location:       loc,
		JWTSecret:      testSecret,
		FrontendOrigin: "http://localhost:5173",
		StudentDomain:  studentDomain,
		TeacherDomain:  teacherDomain,
		GoogleClientID: googleClientID,
	}
}

func testProofs(t *testing.T, ttl time.Duration) *auth.Proofs {
	t.Helper()
	proofs, err := auth.NewProofs(testSecret, ttl)
	if err != nil {
		t.Fatalf("auth.NewProofs() returned an unexpected error: %v", err)
	}
	return proofs
}

func testPhotos(t *testing.T) *upload.Photos {
	t.Helper()
	return upload.NewPhotos(t.TempDir())
}

func pngPhoto(t *testing.T, width, height int) []byte {
	t.Helper()
	var buffer bytes.Buffer
	if err := png.Encode(&buffer, solidImage(width, height)); err != nil {
		t.Fatalf("could not encode a test png: %v", err)
	}
	return buffer.Bytes()
}

func jpegPhoto(t *testing.T, width, height int) []byte {
	t.Helper()
	var buffer bytes.Buffer
	if err := jpeg.Encode(&buffer, solidImage(width, height), nil); err != nil {
		t.Fatalf("could not encode a test jpeg: %v", err)
	}
	return buffer.Bytes()
}

func solidImage(width, height int) image.Image {
	canvas := image.NewRGBA(image.Rect(0, 0, width, height))
	for x := 0; x < width; x++ {
		for y := 0; y < height; y++ {
			canvas.Set(x, y, color.RGBA{R: uint8(x % 251), G: uint8(y % 241), B: 90, A: 255})
		}
	}
	return canvas
}

type formFile struct {
	field string
	name  string
	data  []byte
}

func multipartRequest(t *testing.T, method, target string, values map[string][]string, files ...formFile) *http.Request {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	for name, entries := range values {
		for _, value := range entries {
			if err := writer.WriteField(name, value); err != nil {
				t.Fatalf("could not write the field %q: %v", name, err)
			}
		}
	}
	for _, file := range files {
		part, err := writer.CreateFormFile(file.field, file.name)
		if err != nil {
			t.Fatalf("could not create the file part %q: %v", file.field, err)
		}
		if _, err := part.Write(file.data); err != nil {
			t.Fatalf("could not write the file part %q: %v", file.field, err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("could not close the multipart writer: %v", err)
	}
	request := httptest.NewRequest(method, target, bytes.NewReader(body.Bytes()))
	request.Header.Set("Content-Type", writer.FormDataContentType())
	return request
}

func serve(handlerFunc http.HandlerFunc, request *http.Request) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	handlerFunc.ServeHTTP(rec, request)
	return rec
}

func savedPhotos(t *testing.T, photos *upload.Photos) int {
	t.Helper()
	entries, err := os.ReadDir(photos.Dir())
	if errors.Is(err, fs.ErrNotExist) {
		return 0
	}
	if err != nil {
		t.Fatalf("could not read the photo folder: %v", err)
	}
	return len(entries)
}

type accountStore struct {
	registrations []store.Registration
	result        store.RegistrationResult
	createErr     error
	challenges    map[string]store.EmailChallenge
	invites       map[string]store.TeacherInvite
	dropCourses   []store.Course
	allowedIDs    map[int64]bool
	confirmed     []int64
	clearedCodes  []int64
	clearedInvite []int64
	activated     []int64
	startedCodes  []startResetCall
	emailAttempts int
	inviteTries   int
	activateErr   error
	searchErr     error
}

func newAccountStore() *accountStore {
	return &accountStore{
		challenges: make(map[string]store.EmailChallenge),
		invites:    make(map[string]store.TeacherInvite),
		allowedIDs: make(map[int64]bool),
	}
}

func (f *accountStore) CreateRegistration(ctx context.Context, registration store.Registration) (store.RegistrationResult, error) {
	if f.createErr != nil {
		return store.RegistrationResult{}, f.createErr
	}
	f.registrations = append(f.registrations, registration)
	if f.result.UserID == 0 {
		f.result.UserID = int64(len(f.registrations))
	}
	return f.result, nil
}

func (f *accountStore) FindLoginUser(ctx context.Context, identifier string) (store.LoginUser, error) {
	return store.LoginUser{}, store.ErrNotFound
}

func (f *accountStore) IncrementTokenVersion(ctx context.Context, userID int64) (int, error) {
	return 1, nil
}

func (f *accountStore) FindResetTarget(ctx context.Context, identifier string) (store.ResetTarget, error) {
	return store.ResetTarget{}, store.ErrNotFound
}

func (f *accountStore) FindResetChallenge(ctx context.Context, identifier string) (store.ResetChallenge, error) {
	return store.ResetChallenge{}, store.ErrNotFound
}

func (f *accountStore) StartPasswordReset(ctx context.Context, userID int64, codeHash string, expiresAt, sentAt time.Time) error {
	return nil
}

func (f *accountStore) CountResetAttempt(ctx context.Context, userID int64) (int, error) {
	return 1, nil
}

func (f *accountStore) ClearPasswordReset(ctx context.Context, userID int64) error {
	return nil
}

func (f *accountStore) VerifyResetCode(ctx context.Context, userID int64, tokenHash string, expiresAt time.Time) error {
	return nil
}

func (f *accountStore) FinishPasswordReset(ctx context.Context, identifier, tokenHash, passwordHash string) error {
	return nil
}

func (f *accountStore) FindEmailChallenge(ctx context.Context, regNo string) (store.EmailChallenge, error) {
	challenge, ok := f.challenges[regNo]
	if !ok {
		return store.EmailChallenge{}, store.ErrNotFound
	}
	return challenge, nil
}

func (f *accountStore) StartEmailVerification(ctx context.Context, userID int64, codeHash string, expiresAt, sentAt time.Time) error {
	f.startedCodes = append(f.startedCodes, startResetCall{
		userID:    userID,
		codeHash:  codeHash,
		expiresAt: expiresAt,
		sentAt:    sentAt,
	})
	return nil
}

func (f *accountStore) CountEmailAttempt(ctx context.Context, userID int64) (int, error) {
	f.emailAttempts++
	return f.emailAttempts, nil
}

func (f *accountStore) ClearEmailCode(ctx context.Context, userID int64) error {
	f.clearedCodes = append(f.clearedCodes, userID)
	return nil
}

func (f *accountStore) ConfirmEmail(ctx context.Context, userID int64, verifiedAt time.Time) error {
	f.confirmed = append(f.confirmed, userID)
	return nil
}

func (f *accountStore) SearchDropCourseOptions(ctx context.Context, search string) ([]store.Course, error) {
	if f.searchErr != nil {
		return nil, f.searchErr
	}
	return f.dropCourses, nil
}

func (f *accountStore) FilterDropCourseIDs(ctx context.Context, courseIDs []int64) ([]int64, error) {
	allowed := make([]int64, 0, len(courseIDs))
	for _, courseID := range courseIDs {
		if f.allowedIDs[courseID] {
			allowed = append(allowed, courseID)
		}
	}
	return allowed, nil
}

func (f *accountStore) FindInvite(ctx context.Context, inviteHash string) (store.TeacherInvite, error) {
	invite, ok := f.invites[inviteHash]
	if !ok {
		return store.TeacherInvite{}, store.ErrNotFound
	}
	return invite, nil
}

func (f *accountStore) CountInviteAttempt(ctx context.Context, userID int64) (int, error) {
	f.inviteTries++
	return f.inviteTries, nil
}

func (f *accountStore) ClearInvite(ctx context.Context, userID int64) error {
	f.clearedInvite = append(f.clearedInvite, userID)
	return nil
}

func (f *accountStore) ActivateTeacher(ctx context.Context, userID int64, passwordHash, photoPath string) error {
	if f.activateErr != nil {
		return f.activateErr
	}
	if passwordHash == "" {
		return errors.New("the activation did not receive a password hash")
	}
	f.activated = append(f.activated, userID)
	return nil
}

func newAccountAuth(t *testing.T, users AuthStore, photos PhotoStore, google googleauth.Validator, mail *fakeMailer) *Auth {
	t.Helper()
	return NewAuth(accountConfig(t, "test"), users, &fakeIssuer{token: issuedToken}, mail,
		testProofs(t, auth.ProofTTL), google, photos)
}

func successField(t *testing.T, rec *httptest.ResponseRecorder, name string) string {
	t.Helper()
	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("body is not valid JSON: %v (body = %q)", err, rec.Body.String())
	}
	value, ok := payload[name].(string)
	if !ok {
		t.Fatalf("the response has no %q string field (body = %q)", name, rec.Body.String())
	}
	return value
}

func withCurrentUser(request *http.Request, userID int64, role string) *http.Request {
	current := middleware.User{ID: userID, Role: role}
	return request.WithContext(middleware.WithUser(request.Context(), current))
}

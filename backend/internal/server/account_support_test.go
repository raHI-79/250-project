package server

import (
	"bytes"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

const (
	flowBirthDate        = "2003-05-14"
	teacherFlowBirthDate = "1988-03-09"
	flowJoiningDate      = "2015-01-15"
)

func flowPhoto(t *testing.T) []byte {
	t.Helper()
	canvas := image.NewRGBA(image.Rect(0, 0, 220, 220))
	for x := 0; x < 220; x++ {
		for y := 0; y < 220; y++ {
			canvas.Set(x, y, color.RGBA{R: uint8(x), G: uint8(y), B: 120, A: 255})
		}
	}
	var buffer bytes.Buffer
	if err := png.Encode(&buffer, canvas); err != nil {
		t.Fatalf("could not encode a test photo: %v", err)
	}
	return buffer.Bytes()
}

func formCall(t *testing.T, router http.Handler, method, path, token string,
	values map[string][]string, photo []byte) *httptest.ResponseRecorder {
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
	if photo != nil {
		part, err := writer.CreateFormFile("photo", "photo.png")
		if err != nil {
			t.Fatalf("could not create the photo part: %v", err)
		}
		if _, err := part.Write(photo); err != nil {
			t.Fatalf("could not write the photo part: %v", err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("could not close the multipart writer: %v", err)
	}
	request := httptest.NewRequest(method, path, bytes.NewReader(body.Bytes()))
	request.Header.Set("Content-Type", writer.FormDataContentType())
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, request)
	return rec
}

func formValues(t *testing.T, body string, defaults map[string]string) map[string][]string {
	t.Helper()
	var decoded map[string]any
	if err := json.Unmarshal([]byte(body), &decoded); err != nil {
		t.Fatalf("the fixture is not valid JSON: %v (body = %q)", err, body)
	}
	values := make(map[string][]string, len(decoded)+len(defaults))
	for name, raw := range decoded {
		text, ok := raw.(string)
		if !ok {
			continue
		}
		values[name] = []string{text}
	}
	for name, value := range defaults {
		if _, present := values[name]; !present {
			values[name] = []string{value}
		}
	}
	delete(values, "drop_course_code")
	return values
}

func registerStudent(t *testing.T, router http.Handler, body string, dropCourseIDs ...int64) *httptest.ResponseRecorder {
	t.Helper()
	values := formValues(t, body, map[string]string{"date_of_birth": flowBirthDate})
	for _, courseID := range dropCourseIDs {
		values["drop_course_ids"] = append(values["drop_course_ids"], strconv.FormatInt(courseID, 10))
	}
	return formCall(t, router, http.MethodPost, "/api/auth/register", "", values, flowPhoto(t))
}

func profileBody(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("body is not valid JSON: %v (body = %q)", err, rec.Body.String())
	}
	return payload
}

func chooseDropCourses(t *testing.T, router http.Handler, regNo string, courseIDs []int64) {
	t.Helper()
	values := map[string][]string{}
	for _, courseID := range courseIDs {
		values["drop_course_ids"] = append(values["drop_course_ids"], strconv.FormatInt(courseID, 10))
	}
	saved := formCall(t, router, http.MethodPatch, "/api/profile",
		loginToken(t, router, regNo), values, nil)
	if saved.Code != http.StatusOK {
		t.Fatalf("drop course patch status = %d, want 200 (body = %q)", saved.Code, saved.Body.String())
	}
}

func verifyStudentEmail(t *testing.T, router http.Handler, registered *httptest.ResponseRecorder, regNo string) {
	t.Helper()
	code := field(t, registered, "dev_code")
	verified := call(t, router, http.MethodPost, "/api/auth/verify-email",
		`{"reg_no":"`+regNo+`","code":"`+code+`"}`, "")
	if verified.Code != http.StatusOK {
		t.Fatalf("verify email status = %d, want 200 (body = %q)", verified.Code, verified.Body.String())
	}
}

func onboardStudent(t *testing.T, router http.Handler, adminToken, body string, dropCourseIDs ...int64) int64 {
	t.Helper()
	registered := registerStudent(t, router, body)
	if registered.Code != http.StatusCreated {
		t.Fatalf("register status = %d, want 201 (body = %q)", registered.Code, registered.Body.String())
	}
	regNo := regNoOf(t, body)
	verifyStudentEmail(t, router, registered, regNo)
	userID := approveStudent(t, router, adminToken, regNo)
	if len(dropCourseIDs) > 0 {
		chooseDropCourses(t, router, regNo, dropCourseIDs)
	}
	return userID
}

func onboardCR(t *testing.T, router http.Handler, adminToken, body string) int64 {
	t.Helper()
	userID := onboardStudent(t, router, adminToken, body)
	promoteCR(t, router, adminToken, userID)
	return userID
}

func promoteCR(t *testing.T, router http.Handler, adminToken string, userID int64) {
	t.Helper()
	promoted := call(t, router, http.MethodPost, "/api/admin/crs/promote",
		`{"user_id":`+strconv.FormatInt(userID, 10)+`}`, adminToken)
	if promoted.Code != http.StatusOK {
		t.Fatalf("promote status = %d, want 200 (body = %q)", promoted.Code, promoted.Body.String())
	}
}

func inviteTeacherCall(t *testing.T, router http.Handler, adminToken, body string) *httptest.ResponseRecorder {
	t.Helper()
	values := formValues(t, body, map[string]string{
		"designation":   "Assistant Professor",
		"blood_group":   "B+",
		"office_room":   "AB-401",
		"date_of_birth": teacherFlowBirthDate,
		"joining_date":  flowJoiningDate,
	})
	delete(values, "password")
	delete(values, "confirm_password")
	return formCall(t, router, http.MethodPost, "/api/admin/teachers", adminToken, values, nil)
}

func teacherInviteCall(t *testing.T, router http.Handler, adminToken, body string) *httptest.ResponseRecorder {
	t.Helper()
	invited := inviteTeacherCall(t, router, adminToken, body)
	if invited.Code != http.StatusCreated {
		return invited
	}
	activated := activateTeacher(t, router, field(t, invited, "invite_link"), chosenUserIDOf(t, body), teacherFlowBirthDate)
	if activated.Code != http.StatusOK {
		t.Fatalf("activate teacher status = %d, want 200 (body = %q)", activated.Code, activated.Body.String())
	}
	return invited
}

func activateTeacher(t *testing.T, router http.Handler, link, chosenID, birthDate string) *httptest.ResponseRecorder {
	t.Helper()
	values := map[string][]string{
		"token":            {inviteTokenOf(t, link)},
		"user_id":          {chosenID},
		"date_of_birth":    {birthDate},
		"password":         {flowPassword},
		"confirm_password": {flowPassword},
	}
	return formCall(t, router, http.MethodPost, "/api/auth/activate", "", values, nil)
}

func inviteTokenOf(t *testing.T, link string) string {
	t.Helper()
	_, token, found := strings.Cut(link, "token=")
	if !found || token == "" {
		t.Fatalf("the invitation link %q carries no token", link)
	}
	return token
}

func regNoOf(t *testing.T, body string) string {
	t.Helper()
	return stringField(t, body, "reg_no")
}

func chosenUserIDOf(t *testing.T, body string) string {
	t.Helper()
	return stringField(t, body, "user_id")
}

func stringField(t *testing.T, body, name string) string {
	t.Helper()
	var decoded map[string]any
	if err := json.Unmarshal([]byte(body), &decoded); err != nil {
		t.Fatalf("the fixture is not valid JSON: %v", err)
	}
	value, ok := decoded[name].(string)
	if !ok {
		t.Fatalf("the fixture has no %q field", name)
	}
	return value
}

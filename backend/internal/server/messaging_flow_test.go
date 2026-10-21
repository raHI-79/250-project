package server

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/raHI-79/250-project/backend/internal/handler"
	"github.com/raHI-79/250-project/backend/internal/upload"
)

const (
	chatFlowCR = `{"full_name":"Chat CR","reg_no":"2023331960","batch":"CSE-23","section":"A",
"email":"chatcr@student.sust.edu","phone":"01712345960","blood_group":"O+",
"password":"sust-250-project","confirm_password":"sust-250-project"}`
	chatFlowStudent = `{"full_name":"Chat Student","reg_no":"2023331961","batch":"CSE-23","section":"A",
"email":"chatstudent@student.sust.edu","phone":"01712345961","blood_group":"A+",
"password":"sust-250-project","confirm_password":"sust-250-project"}`
	chatFlowOutsider = `{"full_name":"Chat Outsider","reg_no":"2022331962","batch":"CSE-22","section":"A",
"email":"chatoutsider@student.sust.edu","phone":"01712345962","blood_group":"B+",
"password":"sust-250-project","confirm_password":"sust-250-project"}`
	chatFlowTeacher = `{"name":"Chat Teacher","user_id":"T-960","email":"chatteacher@sust.edu",
"phone":"01712345963"}`
)

func sendMessage(t *testing.T, router http.Handler, conversationID int64, token string,
	fields map[string]string, filename string, data []byte) *httptest.ResponseRecorder {
	t.Helper()
	var buffer bytes.Buffer
	writer := multipart.NewWriter(&buffer)
	for name, value := range fields {
		if err := writer.WriteField(name, value); err != nil {
			t.Fatalf("could not write the field %q: %v", name, err)
		}
	}
	if filename != "" {
		part, err := writer.CreateFormFile(upload.AttachmentField, filename)
		if err != nil {
			t.Fatalf("could not create the file part: %v", err)
		}
		if _, err := part.Write(data); err != nil {
			t.Fatalf("could not write the file part: %v", err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("could not close the multipart writer: %v", err)
	}
	path := "/api/messages/conversations/" + strconv.FormatInt(conversationID, 10) + "/messages"
	request := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(buffer.Bytes()))
	request.Header.Set("Content-Type", writer.FormDataContentType())
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, request)
	return rec
}

func TestMessagingAndNoticeFlow(t *testing.T) {
	ctx, pool, router := newFlowRouter(t)
	adminToken := seedAdmin(t, ctx, pool, router)
	teacherUserID := onboardTeacherForMessaging(t, router, adminToken)
	crUserID := onboardCR(t, router, adminToken, chatFlowCR)
	studentUserID := onboardStudent(t, router, adminToken, chatFlowStudent)
	onboardStudent(t, router, adminToken, chatFlowOutsider)
	crToken := loginToken(t, router, "2023331960")
	studentToken := loginToken(t, router, "2023331961")
	outsiderToken := loginToken(t, router, "2022331962")
	teacherToken := loginToken(t, router, "T-960")

	contacts := listItems(t, call(t, router,
		http.MethodGet, "/api/messages/contacts?role=teacher", "", studentToken))
	if len(contacts) == 0 {
		t.Fatal("the student cannot find a teacher to message")
	}
	opened := call(t, router, http.MethodPost, "/api/messages/direct",
		`{"user_id":`+strconv.FormatInt(teacherUserID, 10)+`}`, studentToken)
	if opened.Code != http.StatusOK {
		t.Fatalf("open direct status = %d, want 200 (body = %q)", opened.Code, opened.Body.String())
	}
	directID := int64(number(t, opened, "id"))
	again := call(t, router, http.MethodPost, "/api/messages/direct",
		`{"user_id":`+strconv.FormatInt(teacherUserID, 10)+`}`, studentToken)
	if int64(number(t, again, "id")) != directID {
		t.Error("opening the same direct conversation twice made a second one")
	}
	sent := sendMessage(t, router, directID, studentToken,
		map[string]string{"body": "Sir, the routine sheet is attached."},
		"routine.xlsx", append([]byte("PK\x03\x04"), bytes.Repeat([]byte{0}, 64)...))
	if sent.Code != http.StatusCreated {
		t.Fatalf("send status = %d, want 201 (body = %q)", sent.Code, sent.Body.String())
	}
	fileURL := field(t, sent, "file_url")
	if fileURL == "" {
		t.Fatal("the sent message carries no file url")
	}
	download := call(t, router, http.MethodGet, fileURL, "", "")
	if download.Code != http.StatusOK {
		t.Fatalf("download status = %d, want 200", download.Code)
	}
	if got := download.Header().Get("Content-Disposition"); got != `attachment; filename="routine.xlsx"` {
		t.Errorf("Content-Disposition = %q, want the original name", got)
	}
	if download.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Error("the attachment is served without the nosniff header")
	}
	teacherUnread := number(t, call(t, router, http.MethodGet, "/api/messages/unread-count", "", teacherToken), "count")
	if teacherUnread != 1 {
		t.Errorf("teacher unread = %v, want 1", teacherUnread)
	}
	if own := number(t, call(t, router, http.MethodGet, "/api/messages/unread-count", "", studentToken), "count"); own != 0 {
		t.Errorf("sender unread = %v, want 0", own)
	}
	teacherAlerts := listItems(t, call(t, router, http.MethodGet, "/api/notifications", "", teacherToken))
	if len(teacherAlerts) == 0 || teacherAlerts[0]["title"] != "New message from Chat Student" {
		t.Fatalf("teacher notifications = %v, want one naming the sender", teacherAlerts)
	}
	teacherChats := listItems(t, call(t, router, http.MethodGet, "/api/messages/conversations", "", teacherToken))
	if len(teacherChats) != 1 || teacherChats[0]["title"] != "Chat Student" {
		t.Fatalf("teacher conversations = %v, want the direct chat with the student", teacherChats)
	}
	if teacherChats[0]["unread_count"] != float64(1) {
		t.Errorf("unread on the row = %v, want 1", teacherChats[0]["unread_count"])
	}
	read := call(t, router, http.MethodGet,
		"/api/messages/conversations/"+strconv.FormatInt(directID, 10)+"/messages", "", teacherToken)
	if read.Code != http.StatusOK {
		t.Fatalf("read status = %d, want 200 (body = %q)", read.Code, read.Body.String())
	}
	messages := listItems(t, read)
	if len(messages) != 1 || messages[0]["file_name"] != "routine.xlsx" {
		t.Fatalf("messages = %v, want the one with the attachment", messages)
	}
	if after := number(t, call(t, router, http.MethodGet, "/api/messages/unread-count", "", teacherToken), "count"); after != 0 {
		t.Errorf("teacher unread after reading = %v, want 0", after)
	}
	hidden := call(t, router, http.MethodGet,
		"/api/messages/conversations/"+strconv.FormatInt(directID, 10)+"/messages", "", outsiderToken)
	if hidden.Code != http.StatusNotFound {
		t.Errorf("an outsider reading the chat status = %d, want 404", hidden.Code)
	}

	group := call(t, router, http.MethodPost, "/api/messages/groups",
		`{"scope":"section","batch":"CSE-23","section":"A"}`, teacherToken)
	if group.Code != http.StatusOK {
		t.Fatalf("open group status = %d, want 200 (body = %q)", group.Code, group.Body.String())
	}
	groupID := int64(number(t, group, "id"))
	if got := field(t, group, "title"); got != "CSE-23 · Section A" {
		t.Errorf("group title = %q, want the section label", got)
	}
	posted := sendMessage(t, router, groupID, teacherToken,
		map[string]string{"body": "Extra class tomorrow at 10:30."}, "", nil)
	if posted.Code != http.StatusCreated {
		t.Fatalf("group post status = %d, want 201 (body = %q)", posted.Code, posted.Body.String())
	}
	refused := sendMessage(t, router, groupID, studentToken, map[string]string{"body": "me too"}, "", nil)
	if refused.Code != http.StatusForbidden {
		t.Fatalf("a student posting in the group status = %d, want 403", refused.Code)
	}
	if got := field(t, refused, "error"); got != handler.MessageCannotPost {
		t.Errorf("error = %q, want %q", got, handler.MessageCannotPost)
	}
	studentChats := listItems(t, call(t, router, http.MethodGet, "/api/messages/conversations", "", studentToken))
	if len(studentChats) != 2 {
		t.Fatalf("the student sees %d conversations, want the direct chat and the group", len(studentChats))
	}
	outsiderChats := listItems(t, call(t, router, http.MethodGet, "/api/messages/conversations", "", outsiderToken))
	if len(outsiderChats) != 0 {
		t.Errorf("another batch sees %d conversations, want none", len(outsiderChats))
	}
	crChats := listItems(t, call(t, router, http.MethodGet, "/api/messages/conversations", "", crToken))
	if len(crChats) != 1 || crChats[0]["kind"] != "group" {
		t.Fatalf("the class representative sees %v, want only the section group", crChats)
	}
	if crChats[0]["can_post"] != false {
		t.Error("the class representative should not be able to post in the teacher's group")
	}

	noticeFields := map[string]string{
		"title":       "Quiz on Monday",
		"description": "Chapters 1 to 3",
		"batch":       "CSE-23",
		"section":     "A",
		"link":        "https://drive.google.com/file/quiz",
	}
	notice := postNoticeForm(t, router, "/api/teacher/notices", noticeFields, teacherToken,
		"syllabus.pdf", []byte("%PDF-1.7 syllabus"))
	if notice.Code != http.StatusCreated {
		t.Fatalf("teacher notice status = %d, want 201 (body = %q)", notice.Code, notice.Body.String())
	}
	mine := listItems(t, call(t, router, http.MethodGet, "/api/teacher/notices", "", teacherToken))
	if len(mine) != 1 || mine[0]["section"] != "A" {
		t.Fatalf("teacher notices = %v, want the one just posted", mine)
	}
	noticeID := int64(mine[0]["id"].(float64))
	forStudent := listItems(t, call(t, router,
		http.MethodGet, "/api/students/notices?source=teacher", "", studentToken))
	if len(forStudent) != 1 || forStudent[0]["title"] != "Quiz on Monday" {
		t.Fatalf("student notices = %v, want the teacher notice", forStudent)
	}
	if forStudent[0]["link_url"] != "https://drive.google.com/file/quiz" {
		t.Errorf("link = %v, want the stored link", forStudent[0]["link_url"])
	}
	if forStudent[0]["file_name"] != "syllabus.pdf" {
		t.Errorf("file_name = %v, want syllabus.pdf", forStudent[0]["file_name"])
	}
	forOutsider := listItems(t, call(t, router,
		http.MethodGet, "/api/students/notices?source=teacher", "", outsiderToken))
	if len(forOutsider) != 0 {
		t.Errorf("another batch sees %d teacher notices, want none", len(forOutsider))
	}
	everything := listItems(t, call(t, router, http.MethodGet, "/api/students/notices", "", studentToken))
	if len(everything) != 1 {
		t.Errorf("the unified list shows %d notices, want 1", len(everything))
	}
	studentAlerts := listItems(t, call(t, router, http.MethodGet, "/api/notifications", "", studentToken))
	if len(studentAlerts) == 0 || studentAlerts[0]["title"] != "Quiz on Monday" {
		t.Fatalf("student notifications = %v, want the notice on top", studentAlerts)
	}
	otherTeacher := onboardSecondTeacher(t, router, adminToken)
	strangerToken := loginToken(t, router, otherTeacher)
	stranger := call(t, router, http.MethodDelete,
		"/api/teacher/notices/"+strconv.FormatInt(noticeID, 10), "", strangerToken)
	if stranger.Code != http.StatusNotFound {
		t.Errorf("another teacher deleting the notice status = %d, want 404", stranger.Code)
	}
	removed := call(t, router, http.MethodDelete,
		"/api/teacher/notices/"+strconv.FormatInt(noticeID, 10), "", teacherToken)
	if removed.Code != http.StatusNoContent {
		t.Fatalf("delete status = %d, want 204 (body = %q)", removed.Code, removed.Body.String())
	}
	if left := listItems(t, call(t, router, http.MethodGet, "/api/teacher/notices", "", teacherToken)); len(left) != 0 {
		t.Errorf("the teacher still has %d notices, want none", len(left))
	}
	if crUserID == 0 || studentUserID == 0 {
		t.Error("the flow did not create the expected accounts")
	}
}

func postNoticeForm(t *testing.T, router http.Handler, path string, fields map[string]string,
	token, filename string, data []byte) *httptest.ResponseRecorder {
	t.Helper()
	var buffer bytes.Buffer
	writer := multipart.NewWriter(&buffer)
	for name, value := range fields {
		if err := writer.WriteField(name, value); err != nil {
			t.Fatalf("could not write the field %q: %v", name, err)
		}
	}
	if filename != "" {
		part, err := writer.CreateFormFile(upload.AttachmentField, filename)
		if err != nil {
			t.Fatalf("could not create the file part: %v", err)
		}
		if _, err := part.Write(data); err != nil {
			t.Fatalf("could not write the file part: %v", err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("could not close the multipart writer: %v", err)
	}
	request := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(buffer.Bytes()))
	request.Header.Set("Content-Type", writer.FormDataContentType())
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, request)
	return rec
}

func onboardTeacherForMessaging(t *testing.T, router http.Handler, adminToken string) int64 {
	t.Helper()
	invited := teacherInviteCall(t, router, adminToken, chatFlowTeacher)
	if invited.Code != http.StatusCreated {
		t.Fatalf("invite status = %d, want 201 (body = %q)", invited.Code, invited.Body.String())
	}
	return int64(number(t, invited, "id"))
}

func onboardSecondTeacher(t *testing.T, router http.Handler, adminToken string) string {
	t.Helper()
	body := `{"name":"Other Teacher","user_id":"T-961","email":"otherteacher@sust.edu",
"phone":"01712345964"}`
	invited := teacherInviteCall(t, router, adminToken, body)
	if invited.Code != http.StatusCreated {
		t.Fatalf("second invite status = %d, want 201 (body = %q)", invited.Code, invited.Body.String())
	}
	return "T-961"
}

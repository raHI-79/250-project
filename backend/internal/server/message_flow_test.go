package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
)

const (
	flowCRAccount = `{"full_name":"Flow CR","reg_no":"2023331920","batch":"CSE-23","section":"A",
"email":"flowcr@student.sust.edu","phone":"01712345920","blood_group":"O+",
"password":"sust-250-project","confirm_password":"sust-250-project"}`
	flowSectionStudent = `{"full_name":"Flow Student A","reg_no":"2023331921","batch":"CSE-23","section":"A",
"email":"flowstudenta@student.sust.edu","phone":"01712345921","blood_group":"A+",
"password":"sust-250-project","confirm_password":"sust-250-project","has_drop_course":false}`
	flowOtherStudent = `{"full_name":"Flow Student B","reg_no":"2023331922","batch":"CSE-23","section":"B",
"email":"flowstudentb@student.sust.edu","phone":"01712345922","blood_group":"B+",
"password":"sust-250-project","confirm_password":"sust-250-project","has_drop_course":false}`
	flowTeacherAccount = `{"name":"Flow Teacher","teacher_id":"T-920","email":"flowteacher@sust.edu",
"phone":"01712345923","password":"sust-250-project","confirm_password":"sust-250-project"}`
)

type pollPayload struct {
	ID         int64  `json:"id"`
	IsClosed   bool   `json:"is_closed"`
	TotalVotes int    `json:"total_votes"`
	MyOptionID *int64 `json:"my_option_id"`
	Options    []struct {
		ID    int64  `json:"id"`
		Text  string `json:"text"`
		Votes int    `json:"votes"`
	} `json:"options"`
}

func pollOf(t *testing.T, rec *httptest.ResponseRecorder) pollPayload {
	t.Helper()
	var payload pollPayload
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("body is not valid JSON: %v (body = %q)", err, rec.Body.String())
	}
	return payload
}

func pollsOf(t *testing.T, rec *httptest.ResponseRecorder) []pollPayload {
	t.Helper()
	var payload struct {
		Items []pollPayload `json:"items"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("body is not valid JSON: %v (body = %q)", err, rec.Body.String())
	}
	return payload.Items
}

func TestMessageAndPollFlow(t *testing.T) {
	ctx, pool, router := newFlowRouter(t)
	adminToken := seedAdmin(t, ctx, pool, router)
	createdTeacher := call(t, router, http.MethodPost, "/api/admin/teachers", flowTeacherAccount, adminToken)
	if createdTeacher.Code != http.StatusCreated {
		t.Fatalf("create teacher status = %d, want 201 (body = %q)", createdTeacher.Code, createdTeacher.Body.String())
	}
	teacherUser := int64(number(t, createdTeacher, "id"))
	var teacherRow int64
	for _, item := range listItems(t, call(t, router, http.MethodGet, "/api/lookups/teachers", "", adminToken)) {
		if int64(item["user_id"].(float64)) == teacherUser {
			teacherRow = int64(item["id"].(float64))
		}
	}
	if teacherRow == 0 {
		t.Fatal("the new teacher is missing from the teacher lookup")
	}
	if created := call(t, router, http.MethodPost, "/api/admin/crs", flowCRAccount, adminToken); created.Code != http.StatusCreated {
		t.Fatalf("create class representative status = %d, want 201 (body = %q)", created.Code, created.Body.String())
	}
	for _, registration := range []string{flowSectionStudent, flowOtherStudent} {
		if registered := call(t, router, http.MethodPost, "/api/auth/register", registration, ""); registered.Code != http.StatusCreated {
			t.Fatalf("register status = %d, want 201 (body = %q)", registered.Code, registered.Body.String())
		}
	}
	approveStudent(t, router, adminToken, "2023331921")
	approveStudent(t, router, adminToken, "2023331922")
	crToken := loginToken(t, router, "2023331920")
	teacherToken := loginToken(t, router, "T-920")
	sectionToken := loginToken(t, router, "2023331921")
	otherToken := loginToken(t, router, "2023331922")
	if teacherRow != teacherUser {
		wrong := postForm(t, router, "/api/cr/messages", map[string]string{
			"target":          "teacher",
			"teacher_user_id": strconv.FormatInt(teacherRow, 10),
			"content":         "Sent with the wrong id.",
		}, "", "", crToken)
		if wrong.Code != http.StatusBadRequest {
			t.Fatalf("the teachers table id returned %d, want 400 (body = %q)", wrong.Code, wrong.Body.String())
		}
	}
	asked := postForm(t, router, "/api/cr/messages", map[string]string{
		"target":          "teacher",
		"teacher_user_id": strconv.FormatInt(teacherUser, 10),
		"content":         "May we move Sunday's class?",
	}, "ask.pdf", pdfBody, crToken)
	if asked.Code != http.StatusCreated {
		t.Fatalf("message to the teacher status = %d, want 201 (body = %q)", asked.Code, asked.Body.String())
	}
	if link := fileURL(t, asked); link == "" {
		t.Error("the message has no file link")
	}
	inbox := listItems(t, call(t, router, http.MethodGet, "/api/teacher/messages/inbox", "", teacherToken))
	if len(inbox) != 1 {
		t.Fatalf("the teacher inbox holds %d messages, want 1", len(inbox))
	}
	if inbox[0]["sender_name"] != "Flow CR" || inbox[0]["sender_batch"] != "CSE-23" || inbox[0]["sender_section"] != "A" {
		t.Errorf("inbox message = %v, want the class representative with a batch and section", inbox[0])
	}
	askedID := int64(inbox[0]["id"].(float64))
	replied := postForm(t, router, "/api/teacher/messages", map[string]string{
		"reply_to_id": strconv.FormatInt(askedID, 10),
		"content":     "Yes, Sunday at ten works.",
	}, "", "", teacherToken)
	if replied.Code != http.StatusCreated {
		t.Fatalf("reply status = %d, want 201 (body = %q)", replied.Code, replied.Body.String())
	}
	crInbox := listItems(t, call(t, router, http.MethodGet, "/api/cr/messages/inbox", "", crToken))
	if len(crInbox) != 1 {
		t.Fatalf("the class representative inbox holds %d messages, want 1", len(crInbox))
	}
	if crInbox[0]["sender_name"] != "Flow Teacher" || crInbox[0]["reply_to_id"] != float64(askedID) {
		t.Errorf("reply = %v, want the teacher answering message %d", crInbox[0], askedID)
	}
	if sent := listItems(t, call(t, router, http.MethodGet, "/api/cr/messages/sent", "", crToken)); len(sent) != 1 {
		t.Errorf("the class representative sent %d messages, want 1", len(sent))
	}
	if sent := listItems(t, call(t, router, http.MethodGet, "/api/teacher/messages/sent", "", teacherToken)); len(sent) != 1 {
		t.Errorf("the teacher sent %d messages, want 1", len(sent))
	}
	broadcast := postForm(t, router, "/api/teacher/messages", map[string]string{
		"target":  "batch",
		"batch":   "CSE-23",
		"section": "A",
		"content": "Section A meets in 1101.",
	}, "", "", teacherToken)
	if broadcast.Code != http.StatusCreated {
		t.Fatalf("section message status = %d, want 201 (body = %q)", broadcast.Code, broadcast.Body.String())
	}
	sectionInbox := listItems(t, call(t, router, http.MethodGet, "/api/students/messages", "", sectionToken))
	if len(sectionInbox) != 1 || sectionInbox[0]["target_type"] != "section" {
		t.Fatalf("section A sees %v, want the one section message", sectionInbox)
	}
	if otherInbox := listItems(t, call(t, router, http.MethodGet, "/api/students/messages", "", otherToken)); len(otherInbox) != 0 {
		t.Errorf("section B sees %v, want nothing", otherInbox)
	}
	createdPoll := call(t, router, http.MethodPost, "/api/cr/polls",
		`{"question":"Which day suits the extra class?","options":["Sunday","Monday"],"scope":"section"}`, crToken)
	if createdPoll.Code != http.StatusCreated {
		t.Fatalf("create poll status = %d, want 201 (body = %q)", createdPoll.Code, createdPoll.Body.String())
	}
	poll := pollOf(t, createdPoll)
	if len(poll.Options) != 2 {
		t.Fatalf("the poll has %d options, want 2", len(poll.Options))
	}
	pollPath := "/api/students/polls/" + strconv.FormatInt(poll.ID, 10) + "/vote"
	vote := `{"option_id":` + strconv.FormatInt(poll.Options[1].ID, 10) + `}`
	if voted := call(t, router, http.MethodPost, pollPath, vote, sectionToken); voted.Code != http.StatusCreated {
		t.Fatalf("vote status = %d, want 201 (body = %q)", voted.Code, voted.Body.String())
	}
	again := call(t, router, http.MethodPost, pollPath, vote, sectionToken)
	if again.Code != http.StatusConflict {
		t.Fatalf("second vote status = %d, want 409 (body = %q)", again.Code, again.Body.String())
	}
	if got := errorField(t, again); got != "You have already voted in this poll." {
		t.Errorf("error = %q, want the already voted message", got)
	}
	if hidden := pollsOf(t, call(t, router, http.MethodGet, "/api/students/polls", "", otherToken)); len(hidden) != 0 {
		t.Errorf("section B sees %v polls, want none", hidden)
	}
	if blocked := call(t, router, http.MethodPost, pollPath, vote, otherToken); blocked.Code != http.StatusNotFound {
		t.Errorf("a vote from another section returned %d, want 404", blocked.Code)
	}
	visible := pollsOf(t, call(t, router, http.MethodGet, "/api/students/polls", "", sectionToken))
	if len(visible) != 1 || visible[0].TotalVotes != 1 || visible[0].MyOptionID == nil {
		t.Fatalf("section A sees %v, want one poll with the own vote", visible)
	}
	if *visible[0].MyOptionID != poll.Options[1].ID {
		t.Errorf("my_option_id = %d, want %d", *visible[0].MyOptionID, poll.Options[1].ID)
	}
	closePath := "/api/cr/polls/" + strconv.FormatInt(poll.ID, 10) + "/close"
	if closed := call(t, router, http.MethodPost, closePath, "", crToken); closed.Code != http.StatusOK {
		t.Fatalf("close status = %d, want 200 (body = %q)", closed.Code, closed.Body.String())
	}
	twice := call(t, router, http.MethodPost, closePath, "", crToken)
	if twice.Code != http.StatusConflict {
		t.Fatalf("closing twice returned %d, want 409 (body = %q)", twice.Code, twice.Body.String())
	}
	late := call(t, router, http.MethodPost, pollPath, vote, sectionToken)
	if late.Code != http.StatusConflict {
		t.Fatalf("a vote in a closed poll returned %d, want 409 (body = %q)", late.Code, late.Body.String())
	}
	if got := errorField(t, late); got != "This poll is closed." {
		t.Errorf("error = %q, want the closed poll message", got)
	}
	own := pollsOf(t, call(t, router, http.MethodGet, "/api/cr/polls", "", crToken))
	if len(own) != 1 || !own[0].IsClosed || own[0].TotalVotes != 1 {
		t.Fatalf("the class representative sees %v, want one closed poll with one vote", own)
	}
	if own[0].Options[1].Votes != 1 {
		t.Errorf("the chosen option has %d votes, want 1", own[0].Options[1].Votes)
	}
}

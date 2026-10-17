package server

import (
	"bytes"
	"encoding/json"
	"github.com/raHI-79/250-project/backend/internal/store"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

const (
	boardCR = `{"full_name":"Board CR","reg_no":"2023331910","batch":"CSE-23","section":"A",
"email":"boardcr@student.sust.edu","phone":"01712345910","blood_group":"O+",
"password":"sust-250-project","confirm_password":"sust-250-project"}`
	boardStudent = `{"full_name":"Board Student","reg_no":"2023331911","batch":"CSE-23","section":"A",
"email":"boardstudent@student.sust.edu","phone":"01712345911","blood_group":"B+",
"password":"sust-250-project","confirm_password":"sust-250-project","has_drop_course":false}`
	boardOutsider = `{"full_name":"Board Outsider","reg_no":"2022331912","batch":"CSE-22","section":"A",
"email":"boardoutsider@student.sust.edu","phone":"01712345912","blood_group":"A+",
"password":"sust-250-project","confirm_password":"sust-250-project","has_drop_course":false}`
	boardTeacher = `{"name":"Board Teacher","teacher_id":"T-910","email":"boardteacher@sust.edu",
"phone":"01712345913","password":"sust-250-project","confirm_password":"sust-250-project"}`
	boardLink = "https://drive.google.com/drive/folders/board"
	pdfBody   = "%PDF-1.7 board notice"
)

func postForm(t *testing.T, router http.Handler, path string, fields map[string]string, filename, content, token string) *httptest.ResponseRecorder {
	t.Helper()
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	for name, value := range fields {
		if err := writer.WriteField(name, value); err != nil {
			t.Fatalf("could not write the field %q: %v", name, err)
		}
	}
	if filename != "" {
		part, err := writer.CreateFormFile("file", filename)
		if err != nil {
			t.Fatalf("could not create the file part: %v", err)
		}
		if _, err := part.Write([]byte(content)); err != nil {
			t.Fatalf("could not write the file part: %v", err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("could not close the multipart writer: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, path, body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func fileURL(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("body is not valid JSON: %v (body = %q)", err, rec.Body.String())
	}
	link, ok := payload["file_url"].(string)
	if !ok {
		t.Fatalf("file_url is not a link (body = %q)", rec.Body.String())
	}
	return link
}

func titlesOf(t *testing.T, rec *httptest.ResponseRecorder) []string {
	t.Helper()
	items := listItems(t, rec)
	titles := make([]string, 0, len(items))
	for _, item := range items {
		title, _ := item["title"].(string)
		titles = append(titles, title)
	}
	return titles
}

func hasTitle(titles []string, want string) bool {
	for _, title := range titles {
		if title == want {
			return true
		}
	}
	return false
}

func TestBoardFlow(t *testing.T) {
	ctx, pool, router := newFlowRouter(t)
	adminToken := seedAdmin(t, ctx, pool, router)
	if created := call(t, router, http.MethodPost, "/api/admin/crs", boardCR, adminToken); created.Code != http.StatusCreated {
		t.Fatalf("create class representative status = %d, want 201 (body = %q)", created.Code, created.Body.String())
	}
	if created := call(t, router, http.MethodPost, "/api/admin/teachers", boardTeacher, adminToken); created.Code != http.StatusCreated {
		t.Fatalf("create teacher status = %d, want 201 (body = %q)", created.Code, created.Body.String())
	}
	for _, registration := range []string{boardStudent, boardOutsider} {
		if registered := call(t, router, http.MethodPost, "/api/auth/register", registration, ""); registered.Code != http.StatusCreated {
			t.Fatalf("register status = %d, want 201 (body = %q)", registered.Code, registered.Body.String())
		}
	}
	approveStudent(t, router, adminToken, "2023331911")
	approveStudent(t, router, adminToken, "2022331912")
	crToken := loginToken(t, router, "2023331910")
	studentToken := loginToken(t, router, "2023331911")
	outsiderToken := loginToken(t, router, "2022331912")
	teacherToken := loginToken(t, router, "T-910")
	withFile := postForm(t, router, "/api/admin/notices",
		map[string]string{"title": "Exam routine", "description": "Final exams start on Sunday."},
		"routine.pdf", pdfBody, adminToken)
	if withFile.Code != http.StatusCreated {
		t.Fatalf("admin notice status = %d, want 201 (body = %q)", withFile.Code, withFile.Body.String())
	}
	noticeID := int64(number(t, withFile, "id"))
	link := fileURL(t, withFile)
	if !strings.HasPrefix(link, "/uploads/") {
		t.Fatalf("file_url = %q, want a link under /uploads/", link)
	}
	var savedPath string
	if err := pool.QueryRow(ctx, "SELECT file_path FROM notices WHERE id = $1", noticeID).Scan(&savedPath); err != nil {
		t.Fatalf("could not read the saved file path: %v", err)
	}
	if savedPath != link {
		t.Errorf("the database holds %q, want %q", savedPath, link)
	}
	served := call(t, router, http.MethodGet, link, "", "")
	if served.Code != http.StatusOK {
		t.Fatalf("serving the file returned %d, want 200", served.Code)
	}
	if got := served.Header().Get("Content-Type"); got != "application/pdf" {
		t.Errorf("Content-Type = %q, want application/pdf", got)
	}
	if served.Body.String() != pdfBody {
		t.Error("the served file does not match the uploaded one")
	}
	forOtherBatch := postForm(t, router, "/api/admin/notices",
		map[string]string{"title": "CSE-22 only", "description": "Not for every batch.", "batch": "CSE-22"},
		"", "", adminToken)
	if forOtherBatch.Code != http.StatusCreated {
		t.Fatalf("admin notice for one batch status = %d, want 201 (body = %q)", forOtherBatch.Code, forOtherBatch.Body.String())
	}
	ownNotice := postForm(t, router, "/api/cr/notices",
		map[string]string{"title": "Class cancelled", "description": "No class tomorrow.", "target": "own"}, "", "", crToken)
	if ownNotice.Code != http.StatusCreated {
		t.Fatalf("own batch notice status = %d, want 201 (body = %q)", ownNotice.Code, ownNotice.Body.String())
	}
	otherNotice := postForm(t, router, "/api/cr/notices",
		map[string]string{"title": "Shared lab", "description": "Lab moved.", "target": "CSE-22"}, "", "", crToken)
	if otherNotice.Code != http.StatusCreated {
		t.Fatalf("other batch notice status = %d, want 201 (body = %q)", otherNotice.Code, otherNotice.Body.String())
	}
	if refused := postForm(t, router, "/api/cr/notices",
		map[string]string{"title": "Wrong target", "target": "CSE-23"}, "", "", crToken); refused.Code != http.StatusBadRequest {
		t.Fatalf("own batch code as a target returned %d, want 400 (body = %q)", refused.Code, refused.Body.String())
	}
	adminForStudent := titlesOf(t, call(t, router, http.MethodGet, "/api/students/notices/admin", "", studentToken))
	if !hasTitle(adminForStudent, "Exam routine") || hasTitle(adminForStudent, "CSE-22 only") {
		t.Errorf("the student sees %v, want the notice for every batch and not the CSE-22 one", adminForStudent)
	}
	adminForOutsider := titlesOf(t, call(t, router, http.MethodGet, "/api/students/notices/admin", "", outsiderToken))
	if !hasTitle(adminForOutsider, "Exam routine") || !hasTitle(adminForOutsider, "CSE-22 only") {
		t.Errorf("the CSE-22 student sees %v, want both administration notices", adminForOutsider)
	}
	mine := titlesOf(t, call(t, router, http.MethodGet, "/api/students/notices/cr?scope=my", "", studentToken))
	if !hasTitle(mine, "Class cancelled") || hasTitle(mine, "Shared lab") {
		t.Errorf("my batch shows %v, want only the own batch notice", mine)
	}
	otherScope := call(t, router, http.MethodGet, "/api/students/notices/cr?scope=other", "", studentToken)
	others := listItems(t, otherScope)
	if len(others) != 1 || others[0]["title"] != "Shared lab" || others[0]["batch"] != "CSE-22" {
		t.Errorf("the other batches show %v, want the CSE-22 notice with its badge", others)
	}
	crOwn := titlesOf(t, call(t, router, http.MethodGet, "/api/cr/notices", "", crToken))
	if len(crOwn) != 2 {
		t.Errorf("the class representative sees %v, want the two notices they posted", crOwn)
	}
	alerts := titlesOf(t, call(t, router, http.MethodGet, "/api/notifications", "", studentToken))
	if !hasTitle(alerts, "Exam routine") || !hasTitle(alerts, "Class cancelled") || hasTitle(alerts, "CSE-22 only") {
		t.Errorf("the notifications of the student are %v, want the ones aimed at this batch", alerts)
	}
	deletePath := "/api/admin/notices/" + strconv.FormatInt(noticeID, 10)
	if removed := call(t, router, http.MethodDelete, deletePath, "", adminToken); removed.Code != http.StatusNoContent {
		t.Fatalf("delete status = %d, want 204 (body = %q)", removed.Code, removed.Body.String())
	}
	var left int
	if err := pool.QueryRow(ctx, "SELECT COUNT(*) FROM notices WHERE id = $1", noticeID).Scan(&left); err != nil {
		t.Fatalf("could not count the notice: %v", err)
	}
	if left != 0 {
		t.Errorf("the notice row is still there")
	}
	if gone := call(t, router, http.MethodGet, link, "", ""); gone.Code != http.StatusNotFound {
		t.Errorf("serving the deleted file returned %d, want 404", gone.Code)
	}
	if again := call(t, router, http.MethodDelete, deletePath, "", adminToken); again.Code != http.StatusNotFound {
		t.Errorf("deleting the notice twice returned %d, want 404", again.Code)
	}
	adminMaterial := `{"batch":"ALL","title":"Handbook","link":"` + boardLink + `"}`
	if added := call(t, router, http.MethodPost, "/api/admin/materials", adminMaterial, adminToken); added.Code != http.StatusCreated {
		t.Fatalf("admin material status = %d, want 201 (body = %q)", added.Code, added.Body.String())
	}
	mineMaterial := call(t, router, http.MethodPost, "/api/cr/materials",
		`{"title":"Own slides","link":"`+boardLink+`","visibility":"mine"}`, crToken)
	if mineMaterial.Code != http.StatusCreated {
		t.Fatalf("own batch material status = %d, want 201 (body = %q)", mineMaterial.Code, mineMaterial.Body.String())
	}
	if got := firstItem(t, call(t, router, http.MethodGet, "/api/cr/materials", "", crToken))["batch"]; got != "CSE-23" {
		t.Errorf("the own batch material is stored for %v, want CSE-23", got)
	}
	batchMaterial := call(t, router, http.MethodPost, "/api/cr/materials",
		`{"title":"Shared notes","link":"`+boardLink+`","visibility":"batch","batch":"CSE-22"}`, crToken)
	if batchMaterial.Code != http.StatusCreated {
		t.Fatalf("other batch material status = %d, want 201 (body = %q)", batchMaterial.Code, batchMaterial.Body.String())
	}
	if refused := call(t, router, http.MethodPost, "/api/cr/materials",
		`{"title":"Broken","link":"drive.google.com","visibility":"all"}`, crToken); refused.Code != http.StatusBadRequest {
		t.Fatalf("a link without a scheme returned %d, want 400 (body = %q)", refused.Code, refused.Body.String())
	}
	studentMaterials := titlesOf(t, call(t, router, http.MethodGet, "/api/students/materials", "", studentToken))
	if !hasTitle(studentMaterials, "Handbook") || !hasTitle(studentMaterials, "Own slides") || hasTitle(studentMaterials, "Shared notes") {
		t.Errorf("the student sees %v, want the own batch materials and the ALL ones", studentMaterials)
	}
	materialID := int64(number(t, batchMaterial, "id"))
	materialPath := "/api/cr/materials/" + strconv.FormatInt(materialID, 10)
	if removed := call(t, router, http.MethodDelete, materialPath, "", crToken); removed.Code != http.StatusNoContent {
		t.Fatalf("delete material status = %d, want 204 (body = %q)", removed.Code, removed.Body.String())
	}
	if again := call(t, router, http.MethodDelete, materialPath, "", crToken); again.Code != http.StatusNotFound {
		t.Errorf("deleting the material twice returned %d, want 404", again.Code)
	}
	people := listItems(t, call(t, router, http.MethodGet, "/api/students/directory", "", studentToken))
	if len(people) != 3 {
		t.Fatalf("the directory lists %d people, want 3", len(people))
	}
	if people[0]["batch"] != "CSE-22" {
		t.Errorf("the directory starts with %v, want the CSE-22 row first", people[0])
	}
	filtered := listItems(t, call(t, router, http.MethodGet, "/api/students/directory?batch=CSE-23&blood_group=O%2B", "", studentToken))
	if len(filtered) != 1 || filtered[0]["reg_no"] != "2023331910" || filtered[0]["is_cr"] != true {
		t.Errorf("the filtered directory is %v, want the class representative only", filtered)
	}
	searched := listItems(t, call(t, router, http.MethodGet, "/api/students/directory?search=board+stud", "", studentToken))
	if len(searched) != 1 || searched[0]["reg_no"] != "2023331911" {
		t.Errorf("the search found %v, want the one student", searched)
	}
	if _, err := pool.Exec(ctx, "UPDATE users SET status = $1 WHERE identifier = $2", store.StatusRemoved, "2022331912"); err != nil {
		t.Fatalf("could not deactivate a student: %v", err)
	}
	active := listItems(t, call(t, router, http.MethodGet, "/api/students/directory", "", studentToken))
	if len(active) != 2 {
		t.Errorf("the directory lists %d people after a removal, want 2", len(active))
	}
	teachers := listItems(t, call(t, router, http.MethodGet, "/api/teachers-info?search=board", "", teacherToken))
	if len(teachers) != 1 || teachers[0]["name"] != "Board Teacher" || teachers[0]["phone"] != "01712345913" {
		t.Errorf("teachers-info returned %v, want the one matching teacher", teachers)
	}
	if empty := listItems(t, call(t, router, http.MethodGet, "/api/teachers-info?search=nobody", "", studentToken)); len(empty) != 0 {
		t.Errorf("teachers-info returned %v for an unmatched search, want nothing", empty)
	}
}

package server

import (
	"context"
	"encoding/json"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/raHI-79/250-project/backend/internal/auth"
	"github.com/raHI-79/250-project/backend/internal/handler"
	"github.com/raHI-79/250-project/backend/internal/middleware"
	"github.com/raHI-79/250-project/backend/internal/store"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
)

const (
	adminFlowIdentifier   = "admin"
	adminFlowPassword     = "admin-flow-password"
	adminFlowRegNo        = "2023331600"
	adminFlowCRRegNo      = "2023331601"
	adminFlowTeacherID    = "T-900"
	adminFlowPersonPass   = "sust-250-project"
	adminFlowRegistration = `{"full_name":"Approval Student","reg_no":"2023331600","batch":"CSE-23","section":"A",
"email":"approval@student.sust.edu","phone":"01712345600","blood_group":"O+",
"password":"sust-250-project","confirm_password":"sust-250-project","has_drop_course":true,
"drop_course_code":"CSE-233"}`
	adminFlowCR = `{"full_name":"Flow CR","reg_no":"2023331601","batch":"CSE-23","section":"B",
"email":"flowcr@student.sust.edu","phone":"01712345601","blood_group":"A+",
"password":"sust-250-project","confirm_password":"sust-250-project"}`
	adminFlowTeacher = `{"name":"Flow Teacher","teacher_id":"T-900","email":"flowteacher@sust.edu",
"phone":"01712345602","password":"sust-250-project","confirm_password":"sust-250-project"}`
)

func seedAdmin(t *testing.T, ctx context.Context, pool *pgxpool.Pool, router http.Handler) string {
	t.Helper()
	hash, err := auth.HashPassword(adminFlowPassword)
	if err != nil {
		t.Fatalf("HashPassword() returned an unexpected error: %v", err)
	}
	if _, err := store.New(pool).EnsureAdmin(ctx, adminFlowIdentifier, hash); err != nil {
		t.Fatalf("EnsureAdmin() returned an unexpected error: %v", err)
	}
	body := `{"identifier":"` + adminFlowIdentifier + `","password":"` + adminFlowPassword + `"}`
	rec := call(t, router, http.MethodPost, "/api/auth/admin/login", body, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("admin login status = %d, want 200 (body = %q)", rec.Code, rec.Body.String())
	}
	return field(t, rec, "token")
}

func number(t *testing.T, rec *httptest.ResponseRecorder, name string) float64 {
	t.Helper()
	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("body is not valid JSON: %v (body = %q)", err, rec.Body.String())
	}
	value, ok := payload[name].(float64)
	if !ok {
		t.Fatalf("the %q field is not a number (body = %q)", name, rec.Body.String())
	}
	return value
}

func firstItem(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var payload struct {
		Items []map[string]any `json:"items"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("body is not valid JSON: %v (body = %q)", err, rec.Body.String())
	}
	if len(payload.Items) == 0 {
		t.Fatalf("the response has no items (body = %q)", rec.Body.String())
	}
	return payload.Items[0]
}

func itemCount(t *testing.T, rec *httptest.ResponseRecorder) int {
	t.Helper()
	var payload struct {
		Items []map[string]any `json:"items"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("body is not valid JSON: %v (body = %q)", err, rec.Body.String())
	}
	return len(payload.Items)
}

func checkStats(t *testing.T, router http.Handler, token string, students, crs, teachers, pending float64) {
	t.Helper()
	rec := call(t, router, http.MethodGet, "/api/admin/stats", "", token)
	if rec.Code != http.StatusOK {
		t.Fatalf("stats status = %d, want 200 (body = %q)", rec.Code, rec.Body.String())
	}
	wants := map[string]float64{
		"total_students":    students,
		"total_crs":         crs,
		"total_teachers":    teachers,
		"pending_approvals": pending,
	}
	for name, want := range wants {
		if got := number(t, rec, name); got != want {
			t.Errorf("%s = %v, want %v", name, got, want)
		}
	}
}

func TestAdminPeopleFlow(t *testing.T) {
	ctx, pool, router := newFlowRouter(t)
	adminToken := seedAdmin(t, ctx, pool, router)
	registered := call(t, router, http.MethodPost, "/api/auth/register", adminFlowRegistration, "")
	if registered.Code != http.StatusCreated {
		t.Fatalf("register status = %d, want 201 (body = %q)", registered.Code, registered.Body.String())
	}
	checkStats(t, router, adminToken, 0, 0, 0, 1)
	pending := call(t, router, http.MethodGet, "/api/admin/registrations", "", adminToken)
	if pending.Code != http.StatusOK {
		t.Fatalf("registrations status = %d, want 200 (body = %q)", pending.Code, pending.Body.String())
	}
	entry := firstItem(t, pending)
	if entry["reg_no"] != adminFlowRegNo {
		t.Fatalf("registration reg_no = %v, want %q", entry["reg_no"], adminFlowRegNo)
	}
	if entry["drop_course_code"] != "CSE-233" || entry["has_drop_course"] != true {
		t.Errorf("the pending registration hides the drop course (item = %v)", entry)
	}
	if _, leaked := entry["password_hash"]; leaked {
		t.Error("the pending registration carries a password hash")
	}
	studentID := int64(entry["id"].(float64))
	studentPath := "/api/admin/registrations/" + strconv.FormatInt(studentID, 10)
	approved := call(t, router, http.MethodPost, studentPath+"/approve", "", adminToken)
	if approved.Code != http.StatusOK {
		t.Fatalf("approve status = %d, want 200 (body = %q)", approved.Code, approved.Body.String())
	}
	again := call(t, router, http.MethodPost, studentPath+"/approve", "", adminToken)
	if again.Code != http.StatusConflict {
		t.Fatalf("second approve status = %d, want 409 (body = %q)", again.Code, again.Body.String())
	}
	if got := field(t, again, "error"); got != handler.MessageRegistrationNotPending {
		t.Errorf("second approve error = %q, want %q", got, handler.MessageRegistrationNotPending)
	}
	missing := call(t, router, http.MethodPost, "/api/admin/registrations/999999/approve", "", adminToken)
	if missing.Code != http.StatusNotFound {
		t.Errorf("approve of an unknown id status = %d, want 404", missing.Code)
	}
	studentLogin := `{"identifier":"` + adminFlowRegNo + `","password":"` + adminFlowPersonPass + `"}`
	loggedIn := call(t, router, http.MethodPost, "/api/auth/login", studentLogin, "")
	if loggedIn.Code != http.StatusOK {
		t.Fatalf("student login status = %d, want 200 (body = %q)", loggedIn.Code, loggedIn.Body.String())
	}
	createdCR := call(t, router, http.MethodPost, "/api/admin/crs", adminFlowCR, adminToken)
	if createdCR.Code != http.StatusCreated {
		t.Fatalf("create class representative status = %d, want 201 (body = %q)", createdCR.Code, createdCR.Body.String())
	}
	crID := int64(number(t, createdCR, "id"))
	duplicateCR := call(t, router, http.MethodPost, "/api/admin/crs", adminFlowCR, adminToken)
	if duplicateCR.Code != http.StatusConflict {
		t.Errorf("duplicate class representative status = %d, want 409 (body = %q)", duplicateCR.Code, duplicateCR.Body.String())
	}
	crLogin := `{"identifier":"` + adminFlowCRRegNo + `","password":"` + adminFlowPersonPass + `"}`
	crLoggedIn := call(t, router, http.MethodPost, "/api/auth/login", crLogin, "")
	if crLoggedIn.Code != http.StatusOK {
		t.Fatalf("class representative login status = %d, want 200 (body = %q)", crLoggedIn.Code, crLoggedIn.Body.String())
	}
	var crBody struct {
		Token string `json:"token"`
		User  struct {
			Role string `json:"role"`
		} `json:"user"`
	}
	if err := json.Unmarshal(crLoggedIn.Body.Bytes(), &crBody); err != nil {
		t.Fatalf("the class representative login body is not valid JSON: %v", err)
	}
	if crBody.User.Role != auth.RoleCR {
		t.Errorf("class representative role = %q, want %q", crBody.User.Role, auth.RoleCR)
	}
	checkStats(t, router, adminToken, 2, 1, 0, 0)
	crs := call(t, router, http.MethodGet, "/api/admin/crs?batch=CSE-23", "", adminToken)
	if crs.Code != http.StatusOK || itemCount(t, crs) != 1 {
		t.Fatalf("class representative list status = %d with %d items, want 200 with 1 (body = %q)", crs.Code, itemCount(t, crs), crs.Body.String())
	}
	crPath := "/api/admin/crs/" + strconv.FormatInt(crID, 10)
	demoted := call(t, router, http.MethodPost, crPath+"/demote", "", adminToken)
	if demoted.Code != http.StatusOK {
		t.Fatalf("demote status = %d, want 200 (body = %q)", demoted.Code, demoted.Body.String())
	}
	afterDemote := call(t, router, http.MethodGet, "/api/me", "", crBody.Token)
	if afterDemote.Code != http.StatusUnauthorized {
		t.Fatalf("me with the old class representative token status = %d, want 401 (body = %q)", afterDemote.Code, afterDemote.Body.String())
	}
	if got := field(t, afterDemote, "error"); got != middleware.MessageSessionExpired {
		t.Errorf("me after demotion error = %q, want %q", got, middleware.MessageSessionExpired)
	}
	demotedAgain := call(t, router, http.MethodPost, crPath+"/demote", "", adminToken)
	if demotedAgain.Code != http.StatusConflict {
		t.Errorf("second demote status = %d, want 409 (body = %q)", demotedAgain.Code, demotedAgain.Body.String())
	}
	createdTeacher := call(t, router, http.MethodPost, "/api/admin/teachers", adminFlowTeacher, adminToken)
	if createdTeacher.Code != http.StatusCreated {
		t.Fatalf("create teacher status = %d, want 201 (body = %q)", createdTeacher.Code, createdTeacher.Body.String())
	}
	teacherID := int64(number(t, createdTeacher, "id"))
	duplicateTeacher := call(t, router, http.MethodPost, "/api/admin/teachers", adminFlowTeacher, adminToken)
	if duplicateTeacher.Code != http.StatusConflict {
		t.Errorf("duplicate teacher status = %d, want 409 (body = %q)", duplicateTeacher.Code, duplicateTeacher.Body.String())
	}
	teacherLogin := `{"identifier":"` + adminFlowTeacherID + `","password":"` + adminFlowPersonPass + `"}`
	teacherLoggedIn := call(t, router, http.MethodPost, "/api/auth/login", teacherLogin, "")
	if teacherLoggedIn.Code != http.StatusOK {
		t.Fatalf("teacher login status = %d, want 200 (body = %q)", teacherLoggedIn.Code, teacherLoggedIn.Body.String())
	}
	var teacherBody struct {
		User struct {
			Role string `json:"role"`
		} `json:"user"`
	}
	if err := json.Unmarshal(teacherLoggedIn.Body.Bytes(), &teacherBody); err != nil {
		t.Fatalf("the teacher login body is not valid JSON: %v", err)
	}
	if teacherBody.User.Role != auth.RoleTeacher {
		t.Errorf("teacher role = %q, want %q", teacherBody.User.Role, auth.RoleTeacher)
	}
	searched := call(t, router, http.MethodGet, "/api/admin/students?search=approval", "", adminToken)
	if searched.Code != http.StatusOK {
		t.Fatalf("student search status = %d, want 200 (body = %q)", searched.Code, searched.Body.String())
	}
	if got := number(t, searched, "total"); got != 1 {
		t.Errorf("student search total = %v, want 1 (body = %q)", got, searched.Body.String())
	}
	if found := firstItem(t, searched); found["reg_no"] != adminFlowRegNo {
		t.Errorf("student search reg_no = %v, want %q", found["reg_no"], adminFlowRegNo)
	}
	removePath := "/api/admin/students/" + strconv.FormatInt(studentID, 10) + "/remove"
	removed := call(t, router, http.MethodPost, removePath, "", adminToken)
	if removed.Code != http.StatusOK {
		t.Fatalf("remove status = %d, want 200 (body = %q)", removed.Code, removed.Body.String())
	}
	removedAgain := call(t, router, http.MethodPost, removePath, "", adminToken)
	if removedAgain.Code != http.StatusConflict {
		t.Errorf("second remove status = %d, want 409 (body = %q)", removedAgain.Code, removedAgain.Body.String())
	}
	blocked := call(t, router, http.MethodPost, "/api/auth/login", studentLogin, "")
	if blocked.Code != http.StatusForbidden {
		t.Fatalf("login after removal status = %d, want 403 (body = %q)", blocked.Code, blocked.Body.String())
	}
	if got := field(t, blocked, "error"); got != handler.MessageRemovedAccount {
		t.Errorf("login after removal error = %q, want %q", got, handler.MessageRemovedAccount)
	}
	teacherPath := "/api/admin/teachers/" + strconv.FormatInt(teacherID, 10) + "/deactivate"
	deactivated := call(t, router, http.MethodPost, teacherPath, "", adminToken)
	if deactivated.Code != http.StatusOK {
		t.Fatalf("deactivate teacher status = %d, want 200 (body = %q)", deactivated.Code, deactivated.Body.String())
	}
	checkStats(t, router, adminToken, 1, 0, 0, 0)
	var hashes int
	if err := pool.QueryRow(ctx, "SELECT COUNT(*) FROM users WHERE password_hash = $1", adminFlowPersonPass).Scan(&hashes); err != nil {
		t.Fatalf("could not count the plain text passwords: %v", err)
	}
	if hashes != 0 {
		t.Errorf("%d accounts store their password as plain text, want 0", hashes)
	}
}

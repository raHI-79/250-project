package server

import (
	"github.com/raHI-79/250-project/backend/internal/handler"
	"github.com/raHI-79/250-project/backend/internal/middleware"
	"github.com/raHI-79/250-project/backend/internal/validate"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
)

const (
	accountStudent = `{"full_name":"Account Student","reg_no":"2023331950","batch":"CSE-23","section":"A",
"email":"accountstudent@student.sust.edu","phone":"01712345950","blood_group":"O+",
"password":"sust-250-project","confirm_password":"sust-250-project"}`
	accountTeacher = `{"name":"Account Teacher","user_id":"T-950","email":"accountteacher@sust.edu",
"phone":"01712345951"}`
	accountCourse = `{"code":"cse245","title":"Databases"}`
)

func TestAccountFlowFromRegistrationToLogin(t *testing.T) {
	ctx, pool, router := newFlowRouter(t)
	adminToken := seedAdmin(t, ctx, pool, router)
	config := call(t, router, http.MethodGet, "/api/auth/config", "", "")
	if config.Code != http.StatusOK {
		t.Fatalf("auth config status = %d, want 200 (body = %q)", config.Code, config.Body.String())
	}
	if got := field(t, config, "student_email_domain"); got != "student.sust.edu" {
		t.Errorf("student_email_domain = %q, want student.sust.edu", got)
	}
	courseID := int64(number(t, mustCreate(t, router, adminToken, "/api/admin/courses", accountCourse), "id"))
	teacherUserID := onboardTeacherForAccounts(t, router, adminToken)
	teacherRowID := teacherRowOf(t, router, adminToken, teacherUserID)
	weekday := 1
	routine := routineBody("A", courseID, teacherRowID, weekday, "08:00", "09:00", "2104", "class")
	if rec := mustCreate(t, router, adminToken, "/api/admin/routines", routine); rec.Code != http.StatusCreated {
		t.Fatalf("routine status = %d, want 201", rec.Code)
	}
	options := call(t, router, http.MethodGet, "/api/auth/drop-course-options?search=cse", "", "")
	if options.Code != http.StatusOK {
		t.Fatalf("drop course options status = %d, want 200 (body = %q)", options.Code, options.Body.String())
	}
	found := listItems(t, options)
	if len(found) != 1 || found[0]["code"] != "CSE245" {
		t.Fatalf("drop course options = %v, want the routine course", found)
	}
	registered := registerStudent(t, router, accountStudent, courseID)
	if registered.Code != http.StatusCreated {
		t.Fatalf("register status = %d, want 201 (body = %q)", registered.Code, registered.Body.String())
	}
	if got := field(t, registered, "message"); got != handler.MessageVerifyEmailSent {
		t.Errorf("register message = %q, want %q", got, handler.MessageVerifyEmailSent)
	}
	blocked := call(t, router, http.MethodPost, "/api/auth/login",
		`{"identifier":"2023331950","password":"sust-250-project","portal":"student"}`, "")
	if blocked.Code != http.StatusForbidden {
		t.Fatalf("unverified login status = %d, want 403 (body = %q)", blocked.Code, blocked.Body.String())
	}
	if got := field(t, blocked, "error"); got != handler.MessageUnverifiedAccount {
		t.Errorf("unverified login error = %q, want %q", got, handler.MessageUnverifiedAccount)
	}
	wrongCode := call(t, router, http.MethodPost, "/api/auth/verify-email",
		`{"reg_no":"2023331950","code":"000000"}`, "")
	if wrongCode.Code != http.StatusBadRequest {
		t.Fatalf("wrong code status = %d, want 400 (body = %q)", wrongCode.Code, wrongCode.Body.String())
	}
	verifyStudentEmail(t, router, registered, "2023331950")
	pendingLogin := call(t, router, http.MethodPost, "/api/auth/login",
		`{"identifier":"2023331950","password":"sust-250-project","portal":"student"}`, "")
	if pendingLogin.Code != http.StatusForbidden {
		t.Fatalf("pending login status = %d, want 403 (body = %q)", pendingLogin.Code, pendingLogin.Body.String())
	}
	if got := field(t, pendingLogin, "error"); got != handler.MessagePendingAccount {
		t.Errorf("pending login error = %q, want %q", got, handler.MessagePendingAccount)
	}
	pending := call(t, router, http.MethodGet, "/api/admin/registrations", "", adminToken)
	entry := firstItem(t, pending)
	if entry["reg_no"] != "2023331950" {
		t.Fatalf("pending registration = %v, want the new student", entry)
	}
	dropCourses, ok := entry["drop_courses"].([]any)
	if !ok || len(dropCourses) != 1 {
		t.Fatalf("drop_courses = %v, want the chosen course", entry["drop_courses"])
	}
	photoURL, _ := entry["photo_url"].(string)
	if photoURL == "" {
		t.Fatal("the pending registration carries no photo")
	}
	photo := call(t, router, http.MethodGet, photoURL, "", "")
	if photo.Code != http.StatusOK {
		t.Fatalf("photo status = %d, want 200", photo.Code)
	}
	if got := photo.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Errorf("photo nosniff header = %q, want nosniff", got)
	}
	studentUserID := int64(entry["id"].(float64))
	approved := call(t, router, http.MethodPost,
		"/api/admin/registrations/"+strconv.FormatInt(studentUserID, 10)+"/approve", "", adminToken)
	if approved.Code != http.StatusOK {
		t.Fatalf("approve status = %d, want 200 (body = %q)", approved.Code, approved.Body.String())
	}
	wrongPortal := call(t, router, http.MethodPost, "/api/auth/login",
		`{"identifier":"2023331950","password":"sust-250-project","portal":"teacher"}`, "")
	if wrongPortal.Code != http.StatusUnauthorized {
		t.Fatalf("student on the teacher portal status = %d, want 401 (body = %q)", wrongPortal.Code, wrongPortal.Body.String())
	}
	if got := field(t, wrongPortal, "error"); got != handler.MessageStudentPortal {
		t.Errorf("portal error = %q, want %q", got, handler.MessageStudentPortal)
	}
	loggedIn := call(t, router, http.MethodPost, "/api/auth/login",
		`{"identifier":"2023331950","password":"sust-250-project","portal":"student"}`, "")
	if loggedIn.Code != http.StatusOK {
		t.Fatalf("student login status = %d, want 200 (body = %q)", loggedIn.Code, loggedIn.Body.String())
	}
	studentToken := field(t, loggedIn, "token")
	profile := call(t, router, http.MethodGet, "/api/profile", "", studentToken)
	if profile.Code != http.StatusOK {
		t.Fatalf("profile status = %d, want 200 (body = %q)", profile.Code, profile.Body.String())
	}
	if got := field(t, profile, "reg_no"); got != "2023331950" {
		t.Errorf("profile reg_no = %q, want 2023331950", got)
	}
	if got := field(t, profile, "date_of_birth"); got != flowBirthDate {
		t.Errorf("profile date_of_birth = %q, want %q", got, flowBirthDate)
	}
	cleared := formCall(t, router, http.MethodPatch, "/api/profile", studentToken,
		map[string][]string{"phone": {"01799999950"}, "drop_course_ids": {""}}, nil)
	if cleared.Code != http.StatusOK {
		t.Fatalf("profile patch status = %d, want 200 (body = %q)", cleared.Code, cleared.Body.String())
	}
	if got := field(t, cleared, "phone"); got != "01799999950" {
		t.Errorf("profile phone = %q, want the new number", got)
	}
	refused := formCall(t, router, http.MethodPatch, "/api/profile", studentToken,
		map[string][]string{"batch": {"CSE-22"}}, nil)
	if refused.Code != http.StatusBadRequest {
		t.Fatalf("batch change status = %d, want 400 (body = %q)", refused.Code, refused.Body.String())
	}
	card := call(t, router, http.MethodGet, "/api/people/"+strconv.FormatInt(teacherUserID, 10), "", studentToken)
	if card.Code != http.StatusOK {
		t.Fatalf("people card status = %d, want 200 (body = %q)", card.Code, card.Body.String())
	}
	if got := field(t, card, "role"); got != "teacher" {
		t.Errorf("people card role = %q, want teacher", got)
	}
	missing := call(t, router, http.MethodGet, "/api/people/999999", "", studentToken)
	if missing.Code != http.StatusNotFound {
		t.Errorf("unknown people card status = %d, want 404", missing.Code)
	}
	changed := call(t, router, http.MethodPost, "/api/profile/password",
		`{"current_password":"sust-250-project","new_password":"second-password","confirm_password":"second-password"}`,
		studentToken)
	if changed.Code != http.StatusOK {
		t.Fatalf("password change status = %d, want 200 (body = %q)", changed.Code, changed.Body.String())
	}
	stale := call(t, router, http.MethodGet, "/api/profile", "", studentToken)
	if stale.Code != http.StatusUnauthorized {
		t.Fatalf("profile with the old token status = %d, want 401", stale.Code)
	}
	if got := field(t, stale, "error"); got != middleware.MessageSessionExpired {
		t.Errorf("stale token error = %q, want %q", got, middleware.MessageSessionExpired)
	}
	again := call(t, router, http.MethodPost, "/api/auth/login",
		`{"identifier":"2023331950","password":"second-password","portal":"student"}`, "")
	if again.Code != http.StatusOK {
		t.Fatalf("login with the new password status = %d, want 200 (body = %q)", again.Code, again.Body.String())
	}
	promoteCR(t, router, adminToken, studentUserID)
	asCR := call(t, router, http.MethodPost, "/api/auth/login",
		`{"identifier":"2023331950","password":"second-password","portal":"student"}`, "")
	if asCR.Code != http.StatusOK {
		t.Fatalf("class representative login status = %d, want 200 (body = %q)", asCR.Code, asCR.Body.String())
	}
	crToken := field(t, asCR, "token")
	alerts := listItems(t, call(t, router, http.MethodGet, "/api/notifications", "", crToken))
	if len(alerts) != 1 {
		t.Fatalf("notifications = %v, want the promotion notice", alerts)
	}
	want := "You are now the Class Representative of CSE-23 section A"
	if alerts[0]["title"] != want {
		t.Errorf("notification title = %v, want %q", alerts[0]["title"], want)
	}
}

func TestTeacherInvitationFlow(t *testing.T) {
	ctx, pool, router := newFlowRouter(t)
	adminToken := seedAdmin(t, ctx, pool, router)
	invited := inviteTeacherCall(t, router, adminToken, accountTeacher)
	if invited.Code != http.StatusCreated {
		t.Fatalf("invite status = %d, want 201 (body = %q)", invited.Code, invited.Body.String())
	}
	teacherUserID := int64(number(t, invited, "id"))
	link := field(t, invited, "invite_link")
	token := inviteTokenOf(t, link)
	lookup := call(t, router, http.MethodGet, "/api/auth/invite?token="+token, "", "")
	if lookup.Code != http.StatusOK {
		t.Fatalf("invite lookup status = %d, want 200 (body = %q)", lookup.Code, lookup.Body.String())
	}
	if got := field(t, lookup, "name"); got != "Account Teacher" {
		t.Errorf("invite name = %q, want Account Teacher", got)
	}
	if got := field(t, lookup, "masked_email"); got != "a***@sust.edu" {
		t.Errorf("masked_email = %q, want a***@sust.edu", got)
	}
	withoutUserID := call(t, router, http.MethodPost, "/api/auth/login",
		`{"identifier":"T-950","password":"sust-250-project","portal":"teacher"}`, "")
	if withoutUserID.Code != http.StatusUnauthorized {
		t.Fatalf("login before a user id exists status = %d, want 401 (body = %q)",
			withoutUserID.Code, withoutUserID.Body.String())
	}
	beforeActivation := call(t, router, http.MethodPost, "/api/auth/login",
		`{"identifier":"accountteacher@sust.edu","password":"sust-250-project","portal":"teacher"}`, "")
	if beforeActivation.Code != http.StatusForbidden {
		t.Fatalf("invited login status = %d, want 403 (body = %q)", beforeActivation.Code, beforeActivation.Body.String())
	}
	if got := field(t, beforeActivation, "error"); got != handler.MessageInvitedAccount {
		t.Errorf("invited login error = %q, want %q", got, handler.MessageInvitedAccount)
	}
	wrongDate := activateTeacher(t, router, link, "T-950", "1990-01-01")
	if wrongDate.Code != http.StatusBadRequest {
		t.Fatalf("activation with a wrong birth date status = %d, want 400", wrongDate.Code)
	}
	if got := field(t, wrongDate, "error"); got != handler.MessageInviteMismatch {
		t.Errorf("activation error = %q, want %q", got, handler.MessageInviteMismatch)
	}
	brokenID := activateTeacher(t, router, link, "1teacher", teacherFlowBirthDate)
	if brokenID.Code != http.StatusBadRequest {
		t.Fatalf("activation with a broken user id status = %d, want 400", brokenID.Code)
	}
	if got := field(t, brokenID, "error"); got != validate.MessageUserIDStart {
		t.Errorf("user id error = %q, want %q", got, validate.MessageUserIDStart)
	}
	free := call(t, router, http.MethodGet, "/api/auth/user-id-available?user_id=T-950", "", "")
	if got := boolean(t, free, "available"); !got {
		t.Error("the chosen user id is reported as taken before the activation")
	}
	resent := call(t, router, http.MethodPost,
		"/api/admin/teachers/"+strconv.FormatInt(teacherUserID, 10)+"/resend-invite", "", adminToken)
	if resent.Code != http.StatusOK {
		t.Fatalf("resend status = %d, want 200 (body = %q)", resent.Code, resent.Body.String())
	}
	newLink := field(t, resent, "invite_link")
	if newLink == link {
		t.Error("the resent invitation repeats the old link")
	}
	if stale := activateTeacher(t, router, link, "T-950", teacherFlowBirthDate); stale.Code != http.StatusBadRequest {
		t.Fatalf("activation with the old link status = %d, want 400 (body = %q)", stale.Code, stale.Body.String())
	}
	activated := activateTeacher(t, router, newLink, "T-950", teacherFlowBirthDate)
	if activated.Code != http.StatusOK {
		t.Fatalf("activation status = %d, want 200 (body = %q)", activated.Code, activated.Body.String())
	}
	if got := field(t, activated, "user_id"); got != "T-950" {
		t.Errorf("activation user_id = %q, want T-950", got)
	}
	reused := activateTeacher(t, router, newLink, "T-950", teacherFlowBirthDate)
	if reused.Code != http.StatusBadRequest {
		t.Errorf("reusing the activation link status = %d, want 400", reused.Code)
	}
	taken := call(t, router, http.MethodGet, "/api/auth/user-id-available?user_id=t-950", "", "")
	if boolean(t, taken, "available") {
		t.Error("the stored user id is still reported as available")
	}
	if got := field(t, taken, "reason"); got != handler.MessageUserIDTaken {
		t.Errorf("reason = %q, want %q", got, handler.MessageUserIDTaken)
	}
	wrongPortal := call(t, router, http.MethodPost, "/api/auth/login",
		`{"identifier":"T-950","password":"`+flowPassword+`","portal":"student"}`, "")
	if wrongPortal.Code != http.StatusUnauthorized {
		t.Fatalf("teacher on the student portal status = %d, want 401", wrongPortal.Code)
	}
	if got := field(t, wrongPortal, "error"); got != handler.MessageTeacherPortal {
		t.Errorf("portal error = %q, want %q", got, handler.MessageTeacherPortal)
	}
	loggedIn := call(t, router, http.MethodPost, "/api/auth/login",
		`{"identifier":"t-950","password":"`+flowPassword+`","portal":"teacher"}`, "")
	if loggedIn.Code != http.StatusOK {
		t.Fatalf("teacher login status = %d, want 200 (body = %q)", loggedIn.Code, loggedIn.Body.String())
	}
	byEmail := call(t, router, http.MethodPost, "/api/auth/login",
		`{"identifier":"AccountTeacher@SUST.edu","password":"`+flowPassword+`","portal":"teacher"}`, "")
	if byEmail.Code != http.StatusOK {
		t.Fatalf("teacher login by email status = %d, want 200 (body = %q)", byEmail.Code, byEmail.Body.String())
	}
	resetByEmail := call(t, router, http.MethodPost, "/api/auth/forgot-password",
		`{"identifier":"accountteacher@sust.edu","date_of_birth":"`+teacherFlowBirthDate+`"}`, "")
	if resetByEmail.Code != http.StatusOK {
		t.Fatalf("teacher reset by email status = %d, want 200 (body = %q)", resetByEmail.Code, resetByEmail.Body.String())
	}
	teacherToken := field(t, loggedIn, "token")
	profile := call(t, router, http.MethodGet, "/api/profile", "", teacherToken)
	if profile.Code != http.StatusOK {
		t.Fatalf("teacher profile status = %d, want 200 (body = %q)", profile.Code, profile.Body.String())
	}
	if got := field(t, profile, "designation"); got != "Assistant Professor" {
		t.Errorf("designation = %q, want Assistant Professor", got)
	}
	if got := field(t, profile, "office_room"); got != "AB-401" {
		t.Errorf("office_room = %q, want AB-401", got)
	}
	patched := formCall(t, router, http.MethodPatch, "/api/profile", teacherToken,
		map[string][]string{"office_room": {"AB-502"}, "bio": {"Teaches databases."}}, nil)
	if patched.Code != http.StatusOK {
		t.Fatalf("teacher profile patch status = %d, want 200 (body = %q)", patched.Code, patched.Body.String())
	}
	if got := field(t, patched, "bio"); got != "Teaches databases." {
		t.Errorf("bio = %q, want the new bio", got)
	}
	edited := call(t, router, http.MethodPatch, "/api/admin/teachers/"+strconv.FormatInt(teacherUserID, 10),
		`{"designation":"Professor"}`, adminToken)
	if edited.Code != http.StatusOK {
		t.Fatalf("admin edit status = %d, want 200 (body = %q)", edited.Code, edited.Body.String())
	}
	afterEdit := call(t, router, http.MethodGet, "/api/profile", "", teacherToken)
	if got := field(t, afterEdit, "designation"); got != "Professor" {
		t.Errorf("designation after the admin edit = %q, want Professor", got)
	}
	noResend := call(t, router, http.MethodPost,
		"/api/admin/teachers/"+strconv.FormatInt(teacherUserID, 10)+"/resend-invite", "", adminToken)
	if noResend.Code != http.StatusConflict {
		t.Errorf("resend for an active teacher status = %d, want 409 (body = %q)", noResend.Code, noResend.Body.String())
	}
}

func mustCreate(t *testing.T, router http.Handler, adminToken, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	rec := call(t, router, http.MethodPost, path, body, adminToken)
	if rec.Code != http.StatusCreated {
		t.Fatalf("POST %s status = %d, want 201 (body = %q)", path, rec.Code, rec.Body.String())
	}
	return rec
}

func onboardTeacherForAccounts(t *testing.T, router http.Handler, adminToken string) int64 {
	t.Helper()
	invited := teacherInviteCall(t, router, adminToken, accountTeacher)
	if invited.Code != http.StatusCreated {
		t.Fatalf("invite status = %d, want 201 (body = %q)", invited.Code, invited.Body.String())
	}
	return int64(number(t, invited, "id"))
}

func teacherRowOf(t *testing.T, router http.Handler, adminToken string, teacherUserID int64) int64 {
	t.Helper()
	for _, item := range listItems(t, call(t, router, http.MethodGet, "/api/lookups/teachers", "", adminToken)) {
		if int64(item["user_id"].(float64)) == teacherUserID {
			return int64(item["id"].(float64))
		}
	}
	t.Fatalf("the teacher %d is missing from the teacher lookup", teacherUserID)
	return 0
}

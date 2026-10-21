package server

import (
	"github.com/raHI-79/250-project/backend/internal/handler"
	"github.com/raHI-79/250-project/backend/internal/store"
	"net/http"
	"strconv"
	"testing"
)

const (
	retentionRegNo   = "2023331700"
	retentionStudent = `{"full_name":"Retention Student","reg_no":"2023331700","batch":"CSE-23","section":"A",
"email":"retention@student.sust.edu","phone":"01712345700","blood_group":"O+",
"password":"sust-250-project","confirm_password":"sust-250-project"}`
)

func TestRecordRetentionFlow(t *testing.T) {
	ctx, pool, router := newFlowRouter(t)
	adminToken := seedAdmin(t, ctx, pool, router)
	userID := onboardStudent(t, router, adminToken, retentionStudent)
	path := "/api/admin/users/" + strconv.FormatInt(userID, 10)
	early := call(t, router, http.MethodPost, path+"/restore", "", adminToken)
	if early.Code != http.StatusConflict {
		t.Fatalf("restore of an active account status = %d, want 409 (body = %q)", early.Code, early.Body.String())
	}
	if got := field(t, early, "error"); got != handler.MessageNotRestorable {
		t.Errorf("restore error = %q, want %q", got, handler.MessageNotRestorable)
	}
	confirm := `{"confirm":"` + retentionRegNo + `"}`
	tooEarly := call(t, router, http.MethodDelete, path+"/permanent", confirm, adminToken)
	if tooEarly.Code != http.StatusConflict {
		t.Fatalf("delete of an active account status = %d, want 409 (body = %q)", tooEarly.Code, tooEarly.Body.String())
	}
	removed := call(t, router, http.MethodPost,
		"/api/admin/students/"+strconv.FormatInt(userID, 10)+"/remove", "", adminToken)
	if removed.Code != http.StatusOK {
		t.Fatalf("remove status = %d, want 200 (body = %q)", removed.Code, removed.Body.String())
	}
	if !listsStudent(t, router, adminToken, "removed", retentionRegNo) {
		t.Error("the removed student is missing from the removed filter")
	}
	restored := call(t, router, http.MethodPost, path+"/restore", "", adminToken)
	if restored.Code != http.StatusOK {
		t.Fatalf("restore status = %d, want 200 (body = %q)", restored.Code, restored.Body.String())
	}
	if !listsStudent(t, router, adminToken, "active", retentionRegNo) {
		t.Error("the restored student is not active again")
	}
	login := `{"identifier":"` + retentionRegNo + `","password":"` + adminFlowPersonPass + `"}`
	back := call(t, router, http.MethodPost, "/api/auth/login", login, "")
	if back.Code != http.StatusOK {
		t.Fatalf("login after the restore status = %d, want 200 (body = %q)", back.Code, back.Body.String())
	}
	if again := call(t, router, http.MethodPost,
		"/api/admin/students/"+strconv.FormatInt(userID, 10)+"/remove", "", adminToken); again.Code != http.StatusOK {
		t.Fatalf("second remove status = %d, want 200 (body = %q)", again.Code, again.Body.String())
	}
	wrongConfirm := call(t, router, http.MethodDelete, path+"/permanent", `{"confirm":"2023331799"}`, adminToken)
	if wrongConfirm.Code != http.StatusBadRequest {
		t.Fatalf("delete with a wrong confirmation status = %d, want 400 (body = %q)",
			wrongConfirm.Code, wrongConfirm.Body.String())
	}
	if got := field(t, wrongConfirm, "error"); got != handler.MessageConfirmMismatch {
		t.Errorf("confirmation error = %q, want %q", got, handler.MessageConfirmMismatch)
	}
	deleted := call(t, router, http.MethodDelete, path+"/permanent", confirm, adminToken)
	if deleted.Code != http.StatusOK {
		t.Fatalf("delete status = %d, want 200 (body = %q)", deleted.Code, deleted.Body.String())
	}
	if listsStudent(t, router, adminToken, "all", retentionRegNo) {
		t.Error("the deleted student is still listed")
	}
	gone := call(t, router, http.MethodPost, "/api/auth/login", login, "")
	if gone.Code != http.StatusUnauthorized {
		t.Errorf("login after the deletion status = %d, want 401 (body = %q)", gone.Code, gone.Body.String())
	}
	missing := call(t, router, http.MethodDelete, path+"/permanent", confirm, adminToken)
	if missing.Code != http.StatusNotFound {
		t.Errorf("second delete status = %d, want 404 (body = %q)", missing.Code, missing.Body.String())
	}
	audit := call(t, router, http.MethodGet, "/api/admin/audit", "", adminToken)
	if audit.Code != http.StatusOK {
		t.Fatalf("audit status = %d, want 200 (body = %q)", audit.Code, audit.Body.String())
	}
	entries := listItems(t, audit)
	if len(entries) != 4 {
		t.Fatalf("got %d audit entries, want 4 (body = %q)", len(entries), audit.Body.String())
	}
	wantActions := []string{store.ActionDelete, store.ActionRemove, store.ActionRestore, store.ActionRemove}
	for index, want := range wantActions {
		if entries[index]["action"] != want {
			t.Errorf("audit entry %d action = %v, want %q", index, entries[index]["action"], want)
		}
	}
	if entries[0]["admin_name"] != adminFlowIdentifier {
		t.Errorf("audit admin = %v, want %q", entries[0]["admin_name"], adminFlowIdentifier)
	}
}

func listsStudent(t *testing.T, router http.Handler, adminToken, status, regNo string) bool {
	t.Helper()
	rec := call(t, router, http.MethodGet, "/api/admin/students?status="+status, "", adminToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("students status = %d, want 200 (body = %q)", rec.Code, rec.Body.String())
	}
	for _, item := range listItems(t, rec) {
		if item["reg_no"] == regNo {
			return true
		}
	}
	return false
}

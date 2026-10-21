package handler

import (
	"github.com/raHI-79/250-project/backend/internal/auth"
	"github.com/raHI-79/250-project/backend/internal/store"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const actingAdminID = int64(1)

func retentionAccounts() map[int64]store.AccountRef {
	return map[int64]store.AccountRef{
		actingAdminID: {ID: actingAdminID, Role: store.RoleAdmin, Status: store.StatusActive, Identifier: "admin"},
		2: {
			ID: 2, Role: store.RoleStudent, Status: store.StatusRemoved, Identifier: "2023331001",
			Email: "student@student.sust.edu", Name: "Removed Student", PhotoPath: "/uploads/photos/a.png",
		},
		3: {
			ID: 3, Role: store.RoleStudent, Status: store.StatusRejected, Identifier: "2023331002",
			Email: "rejected@student.sust.edu", Name: "Rejected Student",
		},
		4: {
			ID: 4, Role: store.RoleTeacher, Status: store.StatusActive, Identifier: "karim.cse",
			Email: "karim@sust.edu", Name: "Active Teacher",
		},
		5: {
			ID: 5, Role: store.RoleAdmin, Status: store.StatusRemoved, Identifier: "office-admin",
			Email: "office@sust.edu", Name: "Second Admin",
		},
	}
}

func retentionCall(t *testing.T, handlerFunc http.HandlerFunc, method, target, id, body string) *httptest.ResponseRecorder {
	t.Helper()
	var request *http.Request
	if body == "" {
		request = httptest.NewRequest(method, target, nil)
	} else {
		request = httptest.NewRequest(method, target, strings.NewReader(body))
		request.Header.Set("Content-Type", "application/json")
	}
	if id != "" {
		request.SetPathValue("id", id)
	}
	return serve(handlerFunc, withCurrentUser(request, actingAdminID, auth.RoleAdmin))
}

func TestAdminRestore(t *testing.T) {
	silenceLog(t)
	tests := []struct {
		name        string
		id          string
		wantStatus  int
		wantError   string
		wantAccount string
		wantAudit   bool
	}{
		{name: "a removed account becomes active", id: "2", wantStatus: http.StatusOK,
			wantAccount: store.StatusActive, wantAudit: true},
		{name: "a rejected account becomes pending", id: "3", wantStatus: http.StatusOK,
			wantAccount: store.StatusPending, wantAudit: true},
		{name: "an active account is refused", id: "4", wantStatus: http.StatusConflict, wantError: MessageNotRestorable},
		{name: "an unknown account", id: "99", wantStatus: http.StatusNotFound, wantError: MessageAccountNotFound},
		{name: "a broken id", id: "two", wantStatus: http.StatusBadRequest, wantError: MessageInvalidID},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			people := &fakeAdminStore{accounts: retentionAccounts()}
			admin := newAdminHandler(t, people)
			rec := retentionCall(t, admin.Restore(), http.MethodPost, "/api/admin/users/"+test.id+"/restore", test.id, "")
			if rec.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d (body = %q)", rec.Code, test.wantStatus, rec.Body.String())
			}
			if test.wantError != "" {
				if got := errorMessage(t, rec); got != test.wantError {
					t.Errorf("error = %q, want %q", got, test.wantError)
				}
			}
			if test.wantAccount != "" {
				if got := people.accounts[2].Status; test.id == "2" && got != test.wantAccount {
					t.Errorf("status = %q, want %q", got, test.wantAccount)
				}
				if got := people.accounts[3].Status; test.id == "3" && got != test.wantAccount {
					t.Errorf("status = %q, want %q", got, test.wantAccount)
				}
			}
			if audited := len(people.auditEntries) == 1; audited != test.wantAudit {
				t.Fatalf("audit entries = %+v, want one entry = %v", people.auditEntries, test.wantAudit)
			}
			if !test.wantAudit {
				return
			}
			entry := people.auditEntries[0]
			if entry.Action != store.ActionRestore || entry.AdminID != actingAdminID {
				t.Errorf("audit entry = %+v, want a restore by the acting admin", entry)
			}
			if !strings.Contains(entry.TargetSummary, test.wantAccount) {
				t.Errorf("target summary = %q, want the new status", entry.TargetSummary)
			}
		})
	}
}

func TestAdminPermanentDelete(t *testing.T) {
	silenceLog(t)
	tests := []struct {
		name       string
		id         string
		body       string
		wantStatus int
		wantError  string
		wantGone   bool
	}{
		{name: "a removed account confirmed by user id", id: "2", body: `{"confirm":"2023331001"}`,
			wantStatus: http.StatusOK, wantGone: true},
		{name: "a rejected account confirmed by email", id: "3", body: `{"confirm":"rejected@student.sust.edu"}`,
			wantStatus: http.StatusOK, wantGone: true},
		{name: "the confirmation ignores case", id: "2", body: `{"confirm":"  2023331001  "}`,
			wantStatus: http.StatusOK, wantGone: true},
		{name: "a wrong confirmation", id: "2", body: `{"confirm":"2023331009"}`,
			wantStatus: http.StatusBadRequest, wantError: MessageConfirmMismatch},
		{name: "an empty confirmation", id: "2", body: `{"confirm":""}`,
			wantStatus: http.StatusBadRequest, wantError: MessageConfirmMismatch},
		{name: "an active account", id: "4", body: `{"confirm":"karim.cse"}`,
			wantStatus: http.StatusConflict, wantError: MessageNotDeletable},
		{name: "an admin account", id: "5", body: `{"confirm":"office-admin"}`,
			wantStatus: http.StatusForbidden, wantError: MessageProtectedAccount},
		{name: "the caller", id: "1", body: `{"confirm":"admin"}`,
			wantStatus: http.StatusForbidden, wantError: MessageProtectedAccount},
		{name: "an unknown account", id: "99", body: `{"confirm":"anything"}`,
			wantStatus: http.StatusNotFound, wantError: MessageAccountNotFound},
		{name: "a broken id", id: "two", body: `{"confirm":"anything"}`,
			wantStatus: http.StatusBadRequest, wantError: MessageInvalidID},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			people := &fakeAdminStore{accounts: retentionAccounts()}
			admin := newAdminHandler(t, people)
			target := "/api/admin/users/" + test.id + "/permanent"
			rec := retentionCall(t, admin.PermanentDelete(), http.MethodDelete, target, test.id, test.body)
			if rec.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d (body = %q)", rec.Code, test.wantStatus, rec.Body.String())
			}
			if test.wantError != "" {
				if got := errorMessage(t, rec); got != test.wantError {
					t.Errorf("error = %q, want %q", got, test.wantError)
				}
			}
			if gone := len(people.deleted) == 1; gone != test.wantGone {
				t.Fatalf("deleted = %v, want %v", people.deleted, test.wantGone)
			}
			if audited := len(people.auditEntries) == 1; audited != test.wantGone {
				t.Fatalf("audit entries = %+v, want one entry = %v", people.auditEntries, test.wantGone)
			}
			if !test.wantGone {
				return
			}
			if got := successMessage(t, rec); got != MessageAccountDeleted {
				t.Errorf("message = %q, want %q", got, MessageAccountDeleted)
			}
			entry := people.auditEntries[0]
			if entry.Action != store.ActionDelete || entry.AdminID != actingAdminID {
				t.Errorf("audit entry = %+v, want a permanent delete by the acting admin", entry)
			}
			if entry.TargetSummary == "" {
				t.Error("the audit entry has no target summary")
			}
		})
	}
}

func TestAdminAudit(t *testing.T) {
	silenceLog(t)
	people := &fakeAdminStore{auditEntries: []store.AuditRow{
		{ID: 3, AdminID: actingAdminID, AdminName: "admin", Action: store.ActionDelete,
			TargetSummary: "student Removed Student (2023331001)", CreatedAt: adminNow},
	}}
	admin := newAdminHandler(t, people)
	rec := adminCall(t, admin.Audit(), http.MethodGet, "/api/admin/audit", "", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body = %q)", rec.Code, rec.Body.String())
	}
	if people.auditLimit != store.DefaultAuditLimit {
		t.Errorf("limit = %d, want %d", people.auditLimit, store.DefaultAuditLimit)
	}
	list := items(t, rec)
	if len(list) != 1 {
		t.Fatalf("got %d items, want 1", len(list))
	}
	if got := list[0]["action"]; got != store.ActionDelete {
		t.Errorf("action = %v, want %q", got, store.ActionDelete)
	}
	if got := list[0]["target_summary"]; got != "student Removed Student (2023331001)" {
		t.Errorf("target_summary = %v, want the stored summary", got)
	}
	if got := list[0]["admin_name"]; got != "admin" {
		t.Errorf("admin_name = %v, want admin", got)
	}
}

func TestAdminRemovalIsAudited(t *testing.T) {
	silenceLog(t)
	tests := []struct {
		name        string
		handlerFor  func(*Admin) http.HandlerFunc
		path        string
		wantSummary string
	}{
		{
			name:        "a removed student",
			handlerFor:  func(admin *Admin) http.HandlerFunc { return admin.RemoveStudent() },
			path:        "/api/admin/students/2/remove",
			wantSummary: "student Removed Student (2023331001)",
		},
		{
			name:        "a deactivated class representative",
			handlerFor:  func(admin *Admin) http.HandlerFunc { return admin.DeactivateCR() },
			path:        "/api/admin/crs/2/deactivate",
			wantSummary: "student Removed Student (2023331001)",
		},
		{
			name:        "a deactivated teacher",
			handlerFor:  func(admin *Admin) http.HandlerFunc { return admin.DeactivateTeacher() },
			path:        "/api/admin/teachers/4/deactivate",
			wantSummary: "teacher Active Teacher (karim.cse)",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			people := &fakeAdminStore{accounts: retentionAccounts()}
			admin := newAdminHandler(t, people)
			id := "2"
			if strings.Contains(test.path, "teachers") {
				id = "4"
			}
			rec := retentionCall(t, test.handlerFor(admin), http.MethodPost, test.path, id, "")
			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200 (body = %q)", rec.Code, rec.Body.String())
			}
			if len(people.auditEntries) != 1 {
				t.Fatalf("audit entries = %+v, want one entry", people.auditEntries)
			}
			entry := people.auditEntries[0]
			if entry.Action != store.ActionRemove {
				t.Errorf("action = %q, want %q", entry.Action, store.ActionRemove)
			}
			if entry.TargetSummary != test.wantSummary {
				t.Errorf("target summary = %q, want %q", entry.TargetSummary, test.wantSummary)
			}
		})
	}
}

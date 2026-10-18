package handler

import (
	"github.com/raHI-79/250-project/backend/internal/auth"
	"github.com/raHI-79/250-project/backend/internal/store"
	"net/http"
	"testing"
)

func portalStore(t *testing.T) *fakeAuthStore {
	t.Helper()
	users := newAuthStore(t)
	studentHash := cachedHash(t, studentPassword)
	users.users["2023331010"] = store.LoginUser{
		AuthUser:     store.AuthUser{ID: 10, Identifier: "2023331010", Role: "student", Status: store.StatusUnverified},
		PasswordHash: studentHash,
		Name:         "Unverified Student",
		Batch:        "CSE-23",
		Section:      "A",
	}
	users.users["T-010"] = store.LoginUser{
		AuthUser:     store.AuthUser{ID: 12, Identifier: "T-010", Role: "teacher", Status: store.StatusInvited},
		PasswordHash: "",
		Name:         "Invited Teacher",
	}
	users.users["2023331011"] = store.LoginUser{
		AuthUser: store.AuthUser{
			ID: 13, Identifier: "2023331011", Role: "student", Status: store.StatusActive, PhotoPath: "/uploads/photos/cccc.png",
		},
		PasswordHash: studentHash,
		Name:         "Photo Student",
		Batch:        "CSE-23",
		Section:      "A",
	}
	return users
}

func TestLoginPortals(t *testing.T) {
	silenceLog(t)
	tests := []struct {
		name       string
		identifier string
		password   string
		portal     string
		wantStatus int
		wantError  string
		wantRole   string
	}{
		{
			name:       "student on the student portal",
			identifier: "2023331001",
			password:   studentPassword,
			portal:     PortalStudent,
			wantStatus: http.StatusOK,
			wantRole:   auth.RoleStudent,
		},
		{
			name:       "class representative on the student portal",
			identifier: "2023331002",
			password:   studentPassword,
			portal:     PortalStudent,
			wantStatus: http.StatusOK,
			wantRole:   auth.RoleCR,
		},
		{
			name:       "teacher on the teacher portal",
			identifier: "T-001",
			password:   teacherPassword,
			portal:     PortalTeacher,
			wantStatus: http.StatusOK,
			wantRole:   auth.RoleTeacher,
		},
		{
			name:       "student without a portal",
			identifier: "2023331001",
			password:   studentPassword,
			wantStatus: http.StatusOK,
			wantRole:   auth.RoleStudent,
		},
		{
			name:       "teacher without a portal",
			identifier: "T-001",
			password:   teacherPassword,
			wantStatus: http.StatusOK,
			wantRole:   auth.RoleTeacher,
		},
		{
			name:       "student on the teacher portal",
			identifier: "2023331001",
			password:   studentPassword,
			portal:     PortalTeacher,
			wantStatus: http.StatusUnauthorized,
			wantError:  MessageStudentPortal,
		},
		{
			name:       "class representative on the teacher portal",
			identifier: "2023331002",
			password:   studentPassword,
			portal:     PortalTeacher,
			wantStatus: http.StatusUnauthorized,
			wantError:  MessageStudentPortal,
		},
		{
			name:       "teacher on the student portal",
			identifier: "T-001",
			password:   teacherPassword,
			portal:     PortalStudent,
			wantStatus: http.StatusUnauthorized,
			wantError:  MessageTeacherPortal,
		},
		{
			name:       "admin on the student portal",
			identifier: "admin",
			password:   adminPassword,
			portal:     PortalStudent,
			wantStatus: http.StatusUnauthorized,
			wantError:  MessageInvalidLogin,
		},
		{
			name:       "unknown portal",
			identifier: "2023331001",
			password:   studentPassword,
			portal:     "office",
			wantStatus: http.StatusBadRequest,
			wantError:  MessageUnknownLoginPortal,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			users := portalStore(t)
			handler := newAuthHandler(t, users, &fakeIssuer{token: issuedToken})
			body := `{"identifier":"` + test.identifier + `","password":"` + test.password + `"}`
			if test.portal != "" {
				body = `{"identifier":"` + test.identifier + `","password":"` + test.password +
					`","portal":"` + test.portal + `"}`
			}
			rec := serve(handler.Login(), jsonRequest(t, http.MethodPost, "/api/auth/login", body, 0, ""))
			if rec.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d (body = %q)", rec.Code, test.wantStatus, rec.Body.String())
			}
			if test.wantError != "" {
				if got := errorMessage(t, rec); got != test.wantError {
					t.Errorf("error = %q, want %q", got, test.wantError)
				}
				return
			}
			user, ok := decodeBody(t, rec)["user"].(map[string]any)
			if !ok {
				t.Fatalf("the login body has no user object (body = %q)", rec.Body.String())
			}
			if user["role"] != test.wantRole {
				t.Errorf("role = %v, want %q", user["role"], test.wantRole)
			}
		})
	}
}

func TestLoginStatusMessages(t *testing.T) {
	silenceLog(t)
	tests := []struct {
		name       string
		identifier string
		password   string
		wantStatus int
		wantError  string
	}{
		{
			name:       "pending",
			identifier: "2023331003",
			password:   studentPassword,
			wantStatus: http.StatusForbidden,
			wantError:  MessagePendingAccount,
		},
		{
			name:       "rejected",
			identifier: "2023331004",
			password:   studentPassword,
			wantStatus: http.StatusForbidden,
			wantError:  MessageRejectedAccount,
		},
		{
			name:       "removed",
			identifier: "2023331005",
			password:   studentPassword,
			wantStatus: http.StatusForbidden,
			wantError:  MessageRemovedAccount,
		},
		{
			name:       "unverified",
			identifier: "2023331010",
			password:   studentPassword,
			wantStatus: http.StatusForbidden,
			wantError:  MessageUnverifiedAccount,
		},
		{
			name:       "invited teacher with any password",
			identifier: "T-010",
			password:   "anything-at-all",
			wantStatus: http.StatusForbidden,
			wantError:  MessageInvitedAccount,
		},
		{
			name:       "unverified with the wrong password",
			identifier: "2023331010",
			password:   "wrong-password",
			wantStatus: http.StatusUnauthorized,
			wantError:  MessageInvalidLogin,
		},
		{
			name:       "unknown account",
			identifier: "2023339999",
			password:   studentPassword,
			wantStatus: http.StatusUnauthorized,
			wantError:  MessageInvalidLogin,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			users := portalStore(t)
			handler := newAuthHandler(t, users, &fakeIssuer{token: issuedToken})
			body := `{"identifier":"` + test.identifier + `","password":"` + test.password + `"}`
			rec := serve(handler.Login(), jsonRequest(t, http.MethodPost, "/api/auth/login", body, 0, ""))
			if rec.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d (body = %q)", rec.Code, test.wantStatus, rec.Body.String())
			}
			if got := errorMessage(t, rec); got != test.wantError {
				t.Errorf("error = %q, want %q", got, test.wantError)
			}
		})
	}
}

func TestLoginCarriesThePhoto(t *testing.T) {
	silenceLog(t)
	users := portalStore(t)
	handler := newAuthHandler(t, users, &fakeIssuer{token: issuedToken})
	body := `{"identifier":"2023331011","password":"` + studentPassword + `"}`
	rec := serve(handler.Login(), jsonRequest(t, http.MethodPost, "/api/auth/login", body, 0, ""))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body = %q)", rec.Code, rec.Body.String())
	}
	user, ok := decodeBody(t, rec)["user"].(map[string]any)
	if !ok {
		t.Fatalf("the login body has no user object (body = %q)", rec.Body.String())
	}
	if user["photo_url"] != "/uploads/photos/cccc.png" {
		t.Errorf("photo_url = %v, want the stored photo", user["photo_url"])
	}
	if user["user_id"] != float64(13) {
		t.Errorf("user_id = %v, want 13", user["user_id"])
	}
}

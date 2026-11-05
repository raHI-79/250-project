package store

import (
	"errors"
	"testing"
	"time"
)

const inviteHash = "hashed-invite-token"

func TestEnsureSuperAdminPromotesTheSeededAccount(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	if _, err := store.EnsureAdmin(ctx, "chief", "stored-hash", "chief@sust.edu", "01700000000"); err != nil {
		t.Fatalf("EnsureAdmin() returned an unexpected error: %v", err)
	}
	promoted, err := store.EnsureSuperAdmin(ctx, "chief")
	if err != nil {
		t.Fatalf("EnsureSuperAdmin() returned an unexpected error: %v", err)
	}
	if !promoted {
		t.Fatal("EnsureSuperAdmin() reported no promotion, want true")
	}
	var role string
	if err := pool.QueryRow(ctx, "SELECT role FROM users WHERE identifier = 'chief'").Scan(&role); err != nil {
		t.Fatalf("could not read back the role: %v", err)
	}
	if role != RoleSuperAdmin {
		t.Errorf("role = %q, want %q", role, RoleSuperAdmin)
	}
	again, err := store.EnsureSuperAdmin(ctx, "chief")
	if err != nil {
		t.Fatalf("EnsureSuperAdmin() returned an unexpected error on the second call: %v", err)
	}
	if again {
		t.Error("EnsureSuperAdmin() promoted the account twice, want false the second time")
	}
}

func TestAdminInviteBecomesAnActiveAdmin(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	chief := insertUser(t, ctx, pool, RoleAdmin, "chief", StatusActive, 0)
	expires := time.Now().Add(72 * time.Hour)
	userID, err := store.CreateAdminInvite(ctx, "office@sust.edu", inviteHash, expires, chief)
	if err != nil {
		t.Fatalf("CreateAdminInvite() returned an unexpected error: %v", err)
	}
	if _, err := store.CreateAdminInvite(ctx, "office@sust.edu", "other", expires, chief); !errors.Is(err, ErrDuplicateAccount) {
		t.Errorf("a second invite for the same email returned %v, want ErrDuplicateAccount", err)
	}
	invite, err := store.FindAdminInvite(ctx, inviteHash)
	if err != nil {
		t.Fatalf("FindAdminInvite() returned an unexpected error: %v", err)
	}
	if invite.UserID != userID || invite.Email != "office@sust.edu" {
		t.Errorf("invite = %+v, want the invited account", invite)
	}
	if err := store.ActivateAdmin(ctx, userID, "office.rakib", "new-hash"); err != nil {
		t.Fatalf("ActivateAdmin() returned an unexpected error: %v", err)
	}
	var status, identifier string
	query := "SELECT status, COALESCE(identifier, '') FROM users WHERE id = $1"
	if err := pool.QueryRow(ctx, query, userID).Scan(&status, &identifier); err != nil {
		t.Fatalf("could not read back the activated admin: %v", err)
	}
	if status != StatusActive || identifier != "office.rakib" {
		t.Errorf("status = %q and identifier = %q, want an active office.rakib", status, identifier)
	}
	if _, err := store.FindAdminInvite(ctx, inviteHash); !errors.Is(err, ErrNotFound) {
		t.Errorf("the invite still resolves after activation, got %v", err)
	}
}

func TestAdminInviteCanBeResentAndCancelled(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	chief := insertUser(t, ctx, pool, RoleAdmin, "chief", StatusActive, 0)
	expires := time.Now().Add(72 * time.Hour)
	userID, err := store.CreateAdminInvite(ctx, "second@sust.edu", inviteHash, expires, chief)
	if err != nil {
		t.Fatalf("CreateAdminInvite() returned an unexpected error: %v", err)
	}
	email, err := store.ResendAdminInvite(ctx, userID, "fresh-hash", expires)
	if err != nil {
		t.Fatalf("ResendAdminInvite() returned an unexpected error: %v", err)
	}
	if email != "second@sust.edu" {
		t.Errorf("email = %q, want the invited address", email)
	}
	if _, err := store.FindAdminInvite(ctx, inviteHash); !errors.Is(err, ErrNotFound) {
		t.Errorf("the old invitation token still works, got %v", err)
	}
	if _, err := store.FindAdminInvite(ctx, "fresh-hash"); err != nil {
		t.Errorf("the fresh invitation token does not work: %v", err)
	}
	if _, err := store.CancelAdminInvite(ctx, userID); err != nil {
		t.Fatalf("CancelAdminInvite() returned an unexpected error: %v", err)
	}
	var count int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM users WHERE id = $1", userID).Scan(&count); err != nil {
		t.Fatalf("could not count the cancelled invite: %v", err)
	}
	if count != 0 {
		t.Errorf("found %d rows for the cancelled invite, want 0", count)
	}
}

func TestOfficeAdminCanBeSuspendedRestoredAndRemoved(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	userID := insertUser(t, ctx, pool, RoleAdmin, "office.one", StatusActive, 0)
	if _, err := store.SuspendAdmin(ctx, userID); err != nil {
		t.Fatalf("SuspendAdmin() returned an unexpected error: %v", err)
	}
	var status string
	if err := pool.QueryRow(ctx, "SELECT status FROM users WHERE id = $1", userID).Scan(&status); err != nil {
		t.Fatalf("could not read back the suspended admin: %v", err)
	}
	if status != StatusRemoved {
		t.Errorf("status = %q, want %q", status, StatusRemoved)
	}
	if _, err := store.RestoreAdmin(ctx, userID); err != nil {
		t.Fatalf("RestoreAdmin() returned an unexpected error: %v", err)
	}
	name, err := store.RemoveAdmin(ctx, userID)
	if err != nil {
		t.Fatalf("RemoveAdmin() returned an unexpected error: %v", err)
	}
	if name != "office.one" {
		t.Errorf("removed name = %q, want office.one", name)
	}
}

func TestTheSuperAdminAccountStaysProtected(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	chief := insertUser(t, ctx, pool, RoleSuperAdmin, "chief", StatusActive, 0)
	if _, err := store.SuspendAdmin(ctx, chief); !errors.Is(err, ErrSuperAdminLocked) {
		t.Errorf("SuspendAdmin() on the super admin returned %v, want ErrSuperAdminLocked", err)
	}
	if _, err := store.RemoveAdmin(ctx, chief); !errors.Is(err, ErrSuperAdminLocked) {
		t.Errorf("RemoveAdmin() on the super admin returned %v, want ErrSuperAdminLocked", err)
	}
}

func TestTeacherOfficeAccessCanBeGrantedAndRevoked(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	userID := insertUser(t, ctx, pool, RoleTeacher, "t.islam", StatusActive, 0)
	insertTeacherRow(t, ctx, pool, userID, "T-101", "Ratul Islam")
	name, err := store.SetTeacherOfficeAccess(ctx, userID, true)
	if err != nil {
		t.Fatalf("SetTeacherOfficeAccess() returned an unexpected error: %v", err)
	}
	if name != "Ratul Islam" {
		t.Errorf("name = %q, want the teacher name", name)
	}
	user, err := store.GetAuthUser(ctx, userID)
	if err != nil {
		t.Fatalf("GetAuthUser() returned an unexpected error: %v", err)
	}
	if !user.OfficeAccess || user.Role != RoleTeacher || !user.IsActive() {
		t.Errorf("user = %+v, want an active teacher with office access", user)
	}
	teachers, err := store.ListOfficeTeachers(ctx, "Ratul", 10)
	if err != nil {
		t.Fatalf("ListOfficeTeachers() returned an unexpected error: %v", err)
	}
	if len(teachers) != 1 || !teachers[0].OfficeAccess {
		t.Fatalf("teachers = %+v, want one teacher with office access", teachers)
	}
	if _, err := store.SetTeacherOfficeAccess(ctx, userID, false); err != nil {
		t.Fatalf("SetTeacherOfficeAccess(false) returned an unexpected error: %v", err)
	}
	user, err = store.GetAuthUser(ctx, userID)
	if err != nil {
		t.Fatalf("GetAuthUser() returned an unexpected error: %v", err)
	}
	if user.OfficeAccess {
		t.Error("the teacher still has office access after revoking it")
	}
	if user.Role != RoleTeacher || !user.IsActive() {
		t.Errorf("user = %+v, want the teacher account untouched", user)
	}
}

func TestOnlyAnActiveTeacherCanBeGivenOfficeAccess(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	studentID := insertUser(t, ctx, pool, RoleStudent, "2023331009", StatusActive, 0)
	if _, err := store.SetTeacherOfficeAccess(ctx, studentID, true); !errors.Is(err, ErrNotATeacher) {
		t.Errorf("SetTeacherOfficeAccess() on a student returned %v, want ErrNotATeacher", err)
	}
}

func TestOfficeAdminListCarriesTheSignInAndActionCount(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	chief := insertUser(t, ctx, pool, RoleSuperAdmin, "chief", StatusActive, 0)
	adminID := insertUser(t, ctx, pool, RoleAdmin, "office.two", StatusActive, 0)
	if err := store.TouchSignIn(ctx, adminID); err != nil {
		t.Fatalf("TouchSignIn() returned an unexpected error: %v", err)
	}
	for i := 0; i < 3; i++ {
		event := AuditEvent{ActorID: adminID, ActorRole: RoleAdmin, Action: ActionCreated, Client: ClientWeb}
		if err := store.WriteEvent(ctx, event); err != nil {
			t.Fatalf("WriteEvent() returned an unexpected error: %v", err)
		}
	}
	admins, err := store.ListOfficeAdmins(ctx)
	if err != nil {
		t.Fatalf("ListOfficeAdmins() returned an unexpected error: %v", err)
	}
	if len(admins) != 2 {
		t.Fatalf("got %d office admins, want 2", len(admins))
	}
	if admins[0].ID != chief || admins[0].Role != RoleSuperAdmin {
		t.Errorf("the first row is %+v, want the super admin first", admins[0])
	}
	if admins[1].ActionCount != 3 {
		t.Errorf("action count = %d, want 3", admins[1].ActionCount)
	}
	if admins[1].LastSignIn == nil {
		t.Error("the office admin has no last sign-in, want the touched time")
	}
	if admins[1].Source != OfficeSourceAccount {
		t.Errorf("source = %q, want %q", admins[1].Source, OfficeSourceAccount)
	}
}

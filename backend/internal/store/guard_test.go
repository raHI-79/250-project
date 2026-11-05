package store

import (
	"errors"
	"testing"
	"time"
)

func TestSettingsAreSavedAndRead(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	chief := insertUser(t, ctx, pool, RoleSuperAdmin, "chief", StatusActive, 0)
	values := map[string]string{"maintenance_mode": "on", "maintenance_message": "Back soon"}
	if err := store.SaveSettings(ctx, values, chief); err != nil {
		t.Fatalf("SaveSettings() returned an unexpected error: %v", err)
	}
	stored, err := store.AllSettings(ctx)
	if err != nil {
		t.Fatalf("AllSettings() returned an unexpected error: %v", err)
	}
	if stored["maintenance_mode"] != "on" || stored["maintenance_message"] != "Back soon" {
		t.Errorf("settings = %+v, want the saved pair", stored)
	}
	again := map[string]string{"maintenance_mode": "off"}
	if err := store.SaveSettings(ctx, again, chief); err != nil {
		t.Fatalf("SaveSettings() is not repeatable: %v", err)
	}
	stored, err = store.AllSettings(ctx)
	if err != nil {
		t.Fatalf("AllSettings() returned an unexpected error: %v", err)
	}
	if stored["maintenance_mode"] != "off" {
		t.Errorf("maintenance = %q, want off", stored["maintenance_mode"])
	}
}

func TestAddressesAreBlockedAndFreed(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	chief := insertUser(t, ctx, pool, RoleSuperAdmin, "chief", StatusActive, 0)
	until := time.Now().Add(time.Hour)
	id, err := store.BlockIP(ctx, "203.0.113.0/24", "A flood", chief, &until)
	if err != nil {
		t.Fatalf("BlockIP() returned an unexpected error: %v", err)
	}
	ranges, err := store.ActiveIPBlocks(ctx)
	if err != nil {
		t.Fatalf("ActiveIPBlocks() returned an unexpected error: %v", err)
	}
	if len(ranges) != 1 || ranges[0] != "203.0.113.0/24" {
		t.Fatalf("blocks = %v, want the one range", ranges)
	}
	blocks, err := store.ListIPBlocks(ctx)
	if err != nil {
		t.Fatalf("ListIPBlocks() returned an unexpected error: %v", err)
	}
	if len(blocks) != 1 || blocks[0].CreatedBy != "chief" {
		t.Errorf("blocks = %+v, want one named block", blocks)
	}
	if _, err := store.UnblockIP(ctx, id); err != nil {
		t.Fatalf("UnblockIP() returned an unexpected error: %v", err)
	}
	if _, err := store.UnblockIP(ctx, id); !errors.Is(err, ErrNotFound) {
		t.Errorf("unblocking twice returned %v, want ErrNotFound", err)
	}
	ranges, err = store.ActiveIPBlocks(ctx)
	if err != nil {
		t.Fatalf("ActiveIPBlocks() returned an unexpected error: %v", err)
	}
	if len(ranges) != 0 {
		t.Errorf("blocks = %v, want none left", ranges)
	}
}

func TestAnExpiredBlockIsIgnored(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	chief := insertUser(t, ctx, pool, RoleSuperAdmin, "chief", StatusActive, 0)
	past := time.Now().Add(-time.Hour)
	if _, err := store.BlockIP(ctx, "198.51.100.7", "Old", chief, &past); err != nil {
		t.Fatalf("BlockIP() returned an unexpected error: %v", err)
	}
	ranges, err := store.ActiveIPBlocks(ctx)
	if err != nil {
		t.Fatalf("ActiveIPBlocks() returned an unexpected error: %v", err)
	}
	if len(ranges) != 0 {
		t.Errorf("blocks = %v, want the expired one gone", ranges)
	}
}

func TestAccountsLockAfterTooManyFailuresAndUnlock(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	userID := insertUser(t, ctx, pool, RoleAdmin, "office.one", StatusActive, 0)
	for attempt := 1; attempt < MaxFailedSignIns; attempt++ {
		locked, err := store.RegisterFailure(ctx, userID)
		if err != nil {
			t.Fatalf("RegisterFailure() returned an unexpected error: %v", err)
		}
		if locked {
			t.Fatalf("the account locked after %d tries, want %d", attempt, MaxFailedSignIns)
		}
	}
	locked, err := store.RegisterFailure(ctx, userID)
	if err != nil {
		t.Fatalf("RegisterFailure() returned an unexpected error: %v", err)
	}
	if !locked {
		t.Fatalf("the account did not lock after %d tries", MaxFailedSignIns)
	}
	accounts, err := store.LockedAccounts(ctx)
	if err != nil {
		t.Fatalf("LockedAccounts() returned an unexpected error: %v", err)
	}
	if len(accounts) != 1 || accounts[0].ID != userID {
		t.Fatalf("locked accounts = %+v, want the one account", accounts)
	}
	if err := store.ClearFailures(ctx, userID); err != nil {
		t.Fatalf("ClearFailures() returned an unexpected error: %v", err)
	}
	accounts, err = store.LockedAccounts(ctx)
	if err != nil {
		t.Fatalf("LockedAccounts() returned an unexpected error: %v", err)
	}
	if len(accounts) != 0 {
		t.Errorf("locked accounts = %+v, want none after unlocking", accounts)
	}
}

func TestTwoPersonRequestsWaitForSomebodyElse(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	asker := insertUser(t, ctx, pool, RoleAdmin, "office.one", StatusActive, 0)
	other := insertUser(t, ctx, pool, RoleAdmin, "office.two", StatusActive, 0)
	others, err := store.OtherOfficeAdmins(ctx, asker)
	if err != nil {
		t.Fatalf("OtherOfficeAdmins() returned an unexpected error: %v", err)
	}
	if others != 1 {
		t.Fatalf("other office admins = %d, want 1", others)
	}
	id, err := store.CreateApproval(ctx, Approval{
		Action:      "delete_user",
		TargetType:  "account",
		TargetID:    "7",
		Summary:     "Delete a student for good",
		RequestedBy: asker,
		ExpiresAt:   time.Now().Add(24 * time.Hour),
	})
	if err != nil {
		t.Fatalf("CreateApproval() returned an unexpected error: %v", err)
	}
	if _, err := store.DecideApproval(ctx, id, asker, ApprovalApproved); !errors.Is(err, ErrOwnApproval) {
		t.Errorf("the asker approved their own request: %v", err)
	}
	approval, err := store.DecideApproval(ctx, id, other, ApprovalApproved)
	if err != nil {
		t.Fatalf("DecideApproval() returned an unexpected error: %v", err)
	}
	if approval.TargetID != "7" {
		t.Errorf("approval = %+v, want the stored target", approval)
	}
	if _, err := store.DecideApproval(ctx, id, other, ApprovalApproved); !errors.Is(err, ErrApprovalGone) {
		t.Errorf("deciding twice returned %v, want ErrApprovalGone", err)
	}
}

func TestAnOldRequestExpires(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	asker := insertUser(t, ctx, pool, RoleAdmin, "office.one", StatusActive, 0)
	other := insertUser(t, ctx, pool, RoleAdmin, "office.two", StatusActive, 0)
	id, err := store.CreateApproval(ctx, Approval{
		Action:      "delete_sheet",
		Summary:     "Delete a sheet",
		RequestedBy: asker,
		ExpiresAt:   time.Now().Add(-time.Minute),
	})
	if err != nil {
		t.Fatalf("CreateApproval() returned an unexpected error: %v", err)
	}
	if _, err := store.DecideApproval(ctx, id, other, ApprovalApproved); !errors.Is(err, ErrApprovalGone) {
		t.Errorf("an expired request was decided: %v", err)
	}
	approvals, err := store.ListApprovals(ctx)
	if err != nil {
		t.Fatalf("ListApprovals() returned an unexpected error: %v", err)
	}
	if len(approvals) != 1 || approvals[0].Status != ApprovalExpired {
		t.Errorf("approvals = %+v, want the expired one marked", approvals)
	}
}

func TestRetentionCleanupDeletesOldRows(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	userID := insertUser(t, ctx, pool, RoleStudent, "2023331001", StatusActive, 0)
	insertStudent(t, ctx, pool, userID, "2023331001", false)
	notice := `INSERT INTO notifications (recipient_user_id, type, title, body, created_at)
VALUES ($1, 'notice', 'Old notice', 'Body', now() - INTERVAL '200 days')`
	if _, err := pool.Exec(ctx, notice, userID); err != nil {
		t.Fatalf("could not insert the old notification: %v", err)
	}
	if err := store.RecordViews(ctx, []PageView{{VisitorID: "v1", Path: "/", Client: ClientWeb}}); err != nil {
		t.Fatalf("RecordViews() returned an unexpected error: %v", err)
	}
	if _, err := pool.Exec(ctx, "UPDATE page_views SET created_at = now() - INTERVAL '200 days'"); err != nil {
		t.Fatalf("could not age the page view: %v", err)
	}
	event := AuditEvent{ActorID: userID, ActorRole: RoleStudent, Action: ActionSignIn, Client: ClientWeb}
	if err := store.WriteEvent(ctx, event); err != nil {
		t.Fatalf("WriteEvent() returned an unexpected error: %v", err)
	}
	if _, err := pool.Exec(ctx, "UPDATE audit_events SET created_at = now() - INTERVAL '400 days'"); err != nil {
		t.Fatalf("could not age the audit event: %v", err)
	}
	sessionID, err := store.CreateSession(ctx, NewSession{
		UserID:    userID,
		Client:    ClientWeb,
		ExpiresAt: time.Now().Add(time.Hour),
	})
	if err != nil {
		t.Fatalf("CreateSession() returned an unexpected error: %v", err)
	}
	if err := store.RevokeSession(ctx, userID, sessionID); err != nil {
		t.Fatalf("RevokeSession() returned an unexpected error: %v", err)
	}
	if _, err := pool.Exec(ctx,
		"UPDATE user_sessions SET revoked_at = now() - INTERVAL '90 days'"); err != nil {
		t.Fatalf("could not age the session: %v", err)
	}
	before := time.Now().AddDate(0, 0, -90)
	if dropped, err := store.DeleteOldViews(ctx, before); err != nil || dropped != 1 {
		t.Errorf("DeleteOldViews() = %d, %v, want 1 and no error", dropped, err)
	}
	if dropped, err := store.DeleteOldEvents(ctx, time.Now().AddDate(0, 0, -365)); err != nil || dropped != 1 {
		t.Errorf("DeleteOldEvents() = %d, %v, want 1 and no error", dropped, err)
	}
	if dropped, err := store.DeleteOldNotifications(ctx, time.Now().AddDate(0, 0, -120)); err != nil ||
		dropped != 1 {
		t.Errorf("DeleteOldNotifications() = %d, %v, want 1 and no error", dropped, err)
	}
	if dropped, err := store.DeleteClosedSessions(ctx, time.Now().AddDate(0, 0, -30)); err != nil ||
		dropped != 1 {
		t.Errorf("DeleteClosedSessions() = %d, %v, want 1 and no error", dropped, err)
	}
	counts, err := store.DataCounts(ctx)
	if err != nil {
		t.Fatalf("DataCounts() returned an unexpected error: %v", err)
	}
	if counts.Views != 0 || counts.Events != 0 || counts.Notifications != 0 || counts.Sessions != 0 {
		t.Errorf("counts = %+v, want everything swept", counts)
	}
}

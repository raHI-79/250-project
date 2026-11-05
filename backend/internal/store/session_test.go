package store

import (
	"errors"
	"testing"
	"time"
)

func TestSessionsAreRevokedOneByOneAndTogether(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	userID := insertUser(t, ctx, pool, RoleAdmin, "office.one", StatusActive, 0)
	ids := make([]int64, 0, 3)
	for i := 0; i < 3; i++ {
		id, err := store.CreateSession(ctx, NewSession{
			UserID:    userID,
			Device:    "Windows",
			Client:    ClientWeb,
			ExpiresAt: time.Now().Add(time.Hour),
		})
		if err != nil {
			t.Fatalf("CreateSession() returned an unexpected error: %v", err)
		}
		ids = append(ids, id)
	}
	live, err := store.ListSessions(ctx, userID)
	if err != nil {
		t.Fatalf("ListSessions() returned an unexpected error: %v", err)
	}
	if len(live) != 3 {
		t.Fatalf("got %d sessions, want 3", len(live))
	}
	if err := store.RevokeSession(ctx, userID, ids[0]); err != nil {
		t.Fatalf("RevokeSession() returned an unexpected error: %v", err)
	}
	if err := store.RevokeSession(ctx, userID, ids[0]); !errors.Is(err, ErrNotFound) {
		t.Errorf("revoking twice returned %v, want ErrNotFound", err)
	}
	gone, err := store.LoadSession(ctx, ids[0])
	if err != nil {
		t.Fatalf("LoadSession() returned an unexpected error: %v", err)
	}
	if gone.Live(time.Now()) {
		t.Error("the revoked session is still live")
	}
	dropped, err := store.RevokeOtherSessions(ctx, userID, ids[1])
	if err != nil {
		t.Fatalf("RevokeOtherSessions() returned an unexpected error: %v", err)
	}
	if dropped != 1 {
		t.Errorf("revoked %d other sessions, want 1", dropped)
	}
	left, err := store.ListSessions(ctx, userID)
	if err != nil {
		t.Fatalf("ListSessions() returned an unexpected error: %v", err)
	}
	if len(left) != 1 || left[0].ID != ids[1] {
		t.Errorf("sessions left = %+v, want only the kept one", left)
	}
}

func TestAnExpiredSessionIsNotLive(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	userID := insertUser(t, ctx, pool, RoleAdmin, "office.one", StatusActive, 0)
	id, err := store.CreateSession(ctx, NewSession{
		UserID:    userID,
		Client:    ClientWeb,
		ExpiresAt: time.Now().Add(-time.Minute),
	})
	if err != nil {
		t.Fatalf("CreateSession() returned an unexpected error: %v", err)
	}
	session, err := store.LoadSession(ctx, id)
	if err != nil {
		t.Fatalf("LoadSession() returned an unexpected error: %v", err)
	}
	if session.Live(time.Now()) {
		t.Error("an expired session reports itself live")
	}
	live, err := store.ListSessions(ctx, userID)
	if err != nil {
		t.Fatalf("ListSessions() returned an unexpected error: %v", err)
	}
	if len(live) != 0 {
		t.Errorf("got %d live sessions, want none", len(live))
	}
}

func TestTheConfirmationWindowOpensAndCloses(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	userID := insertUser(t, ctx, pool, RoleSuperAdmin, "chief", StatusActive, 0)
	id, err := store.CreateSession(ctx, NewSession{
		UserID:    userID,
		Client:    ClientWeb,
		ExpiresAt: time.Now().Add(time.Hour),
	})
	if err != nil {
		t.Fatalf("CreateSession() returned an unexpected error: %v", err)
	}
	until := time.Now().Add(5 * time.Minute)
	if err := store.GrantSudo(ctx, id, until); err != nil {
		t.Fatalf("GrantSudo() returned an unexpected error: %v", err)
	}
	session, err := store.LoadSession(ctx, id)
	if err != nil {
		t.Fatalf("LoadSession() returned an unexpected error: %v", err)
	}
	if session.SudoUntil == nil || !session.SudoUntil.After(time.Now()) {
		t.Fatalf("sudo window = %v, want a future time", session.SudoUntil)
	}
	if err := store.DropSudo(ctx, userID); err != nil {
		t.Fatalf("DropSudo() returned an unexpected error: %v", err)
	}
	session, err = store.LoadSession(ctx, id)
	if err != nil {
		t.Fatalf("LoadSession() returned an unexpected error: %v", err)
	}
	if session.SudoUntil != nil {
		t.Error("the confirmation window is still open")
	}
}

func TestTwoStepCodesExpireAndCount(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	userID := insertUser(t, ctx, pool, RoleAdmin, "office.one", StatusActive, 0)
	if err := store.SetTwoStep(ctx, userID, true); err != nil {
		t.Fatalf("SetTwoStep() returned an unexpected error: %v", err)
	}
	now := time.Now()
	if err := store.StartTwoStep(ctx, userID, "hashed-code", now.Add(10*time.Minute), now); err != nil {
		t.Fatalf("StartTwoStep() returned an unexpected error: %v", err)
	}
	challenge, err := store.TwoStepChallenge(ctx, "office.one")
	if err != nil {
		t.Fatalf("TwoStepChallenge() returned an unexpected error: %v", err)
	}
	if !challenge.Enabled || challenge.CodeHash != "hashed-code" {
		t.Fatalf("challenge = %+v, want the stored code", challenge)
	}
	for attempt := 1; attempt <= 3; attempt++ {
		total, err := store.CountTwoStepAttempt(ctx, userID)
		if err != nil {
			t.Fatalf("CountTwoStepAttempt() returned an unexpected error: %v", err)
		}
		if total != attempt {
			t.Errorf("attempt %d counted as %d", attempt, total)
		}
	}
	if err := store.ClearTwoStep(ctx, userID); err != nil {
		t.Fatalf("ClearTwoStep() returned an unexpected error: %v", err)
	}
	challenge, err = store.TwoStepChallenge(ctx, "office.one")
	if err != nil {
		t.Fatalf("TwoStepChallenge() returned an unexpected error: %v", err)
	}
	if challenge.CodeHash != "" || challenge.Attempts != 0 {
		t.Errorf("challenge = %+v, want a cleared code", challenge)
	}
	if !challenge.Enabled {
		t.Error("clearing the code also turned two-step off")
	}
}

func TestTheEmergencyResetChangesTheHashAndClosesSessions(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	userID := insertUser(t, ctx, pool, RoleSuperAdmin, "chief", StatusActive, 0)
	if _, err := store.CreateSession(ctx, NewSession{
		UserID:    userID,
		Client:    ClientWeb,
		ExpiresAt: time.Now().Add(time.Hour),
	}); err != nil {
		t.Fatalf("CreateSession() returned an unexpected error: %v", err)
	}
	identifier, err := store.ResetSuperAdminPassword(ctx, "fresh-hash")
	if err != nil {
		t.Fatalf("ResetSuperAdminPassword() returned an unexpected error: %v", err)
	}
	if identifier != "chief" {
		t.Errorf("identifier = %q, want chief", identifier)
	}
	if storedPasswordHash(t, ctx, pool, "chief") != "fresh-hash" {
		t.Error("the password hash was not replaced")
	}
	live, err := store.ListSessions(ctx, userID)
	if err != nil {
		t.Fatalf("ListSessions() returned an unexpected error: %v", err)
	}
	if len(live) != 0 {
		t.Errorf("got %d live sessions after the reset, want none", len(live))
	}
	var version int
	if err := pool.QueryRow(ctx, "SELECT token_version FROM users WHERE id = $1", userID).
		Scan(&version); err != nil {
		t.Fatalf("could not read the token version: %v", err)
	}
	if version != 1 {
		t.Errorf("token version = %d, want 1", version)
	}
}

func TestOfficeContactDetailsChange(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	userID := insertUser(t, ctx, pool, RoleAdmin, "office.one", StatusActive, 0)
	if err := store.UpdateOfficeContact(ctx, userID, "office@sust.edu", "01700000000"); err != nil {
		t.Fatalf("UpdateOfficeContact() returned an unexpected error: %v", err)
	}
	var email, phone string
	query := "SELECT COALESCE(contact_email, ''), COALESCE(contact_phone, '') FROM users WHERE id = $1"
	if err := pool.QueryRow(ctx, query, userID).Scan(&email, &phone); err != nil {
		t.Fatalf("could not read back the contact details: %v", err)
	}
	if email != "office@sust.edu" || phone != "01700000000" {
		t.Errorf("contacts = %q and %q, want the new ones", email, phone)
	}
	studentID := insertUser(t, ctx, pool, RoleStudent, "2023331001", StatusActive, 0)
	if err := store.UpdateOfficeContact(ctx, studentID, "s@sust.edu", ""); !errors.Is(err, ErrNotFound) {
		t.Errorf("UpdateOfficeContact() on a student returned %v, want ErrNotFound", err)
	}
}

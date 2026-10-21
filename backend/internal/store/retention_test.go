package store

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5/pgxpool"
	"testing"
	"time"
)

func seedCR(t *testing.T, ctx context.Context, pool *pgxpool.Pool, regNo string) int64 {
	t.Helper()
	userID := insertUser(t, ctx, pool, "student", regNo, StatusActive, 0)
	insertStudent(t, ctx, pool, userID, regNo, true)
	return userID
}

func removeAccount(t *testing.T, ctx context.Context, pool *pgxpool.Pool, userID int64, status string) {
	t.Helper()
	if _, err := pool.Exec(ctx, "UPDATE users SET status = $2 WHERE id = $1", userID, status); err != nil {
		t.Fatalf("could not set the status of the account %d: %v", userID, err)
	}
}

func TestUserIDsAreUniqueWithoutCase(t *testing.T) {
	ctx, _, pool := newTestStore(t)
	insertUser(t, ctx, pool, "teacher", "Karim.CSE", StatusActive, 0)
	query := `INSERT INTO users (role, identifier, password_hash, status) VALUES ('teacher', $1, '', 'active')`
	if _, err := pool.Exec(ctx, query, "karim.cse"); !isUniqueViolation(err) {
		t.Errorf("inserting the same user id in another case gave %v, want a unique violation", err)
	}
	if _, err := pool.Exec(ctx, query, "karim.cse2"); err != nil {
		t.Errorf("inserting a different user id returned an unexpected error: %v", err)
	}
}

func TestInvitedTeachersShareTheEmptyUserID(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	first := NewTeacher{
		Name: "First Invited", Designation: "Lecturer", Email: "first@sust.edu", Phone: "01812345671",
		BloodGroup: "B+", DateOfBirth: "1988-03-09", OfficeRoom: "AB-401",
		InviteHash: "first-hash", InviteExpiresAt: time.Now().Add(time.Hour),
	}
	second := first
	second.Name = "Second Invited"
	second.Email = "second@sust.edu"
	second.InviteHash = "second-hash"
	firstID, err := store.CreateTeacher(ctx, first)
	if err != nil {
		t.Fatalf("CreateTeacher() returned an unexpected error: %v", err)
	}
	secondID, err := store.CreateTeacher(ctx, second)
	if err != nil {
		t.Fatalf("the second invitation returned an unexpected error: %v", err)
	}
	if storedIdentifier(t, ctx, pool, firstID) != "" || storedIdentifier(t, ctx, pool, secondID) != "" {
		t.Error("an invited teacher already carries a user id")
	}
	if err := store.ActivateTeacher(ctx, firstID, "karim.cse", "first-hash-value", ""); err != nil {
		t.Fatalf("ActivateTeacher() returned an unexpected error: %v", err)
	}
	err = store.ActivateTeacher(ctx, secondID, "Karim.CSE", "second-hash-value", "")
	if !errors.Is(err, ErrDuplicateIdentifier) {
		t.Errorf("activating with a taken user id error = %v, want ErrDuplicateIdentifier", err)
	}
	taken, err := store.IdentifierTaken(ctx, "KARIM.cse")
	if err != nil || !taken {
		t.Errorf("IdentifierTaken() = %v, %v, want true and no error", taken, err)
	}
	free, err := store.IdentifierTaken(ctx, "karim.cse2")
	if err != nil || free {
		t.Errorf("IdentifierTaken() for a free id = %v, %v, want false and no error", free, err)
	}
}

func TestRestoreAccount(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	tests := []struct {
		name       string
		status     string
		wantStatus string
		wantErr    error
		wantTokens int
	}{
		{name: "a removed account becomes active", status: StatusRemoved, wantStatus: StatusActive, wantTokens: 3},
		{name: "a rejected account becomes pending", status: StatusRejected, wantStatus: StatusPending, wantTokens: 2},
		{name: "an active account is refused", status: StatusActive, wantErr: ErrNotRestorable},
		{name: "a pending account is refused", status: StatusPending, wantErr: ErrNotRestorable},
	}
	for index, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			regNo := "202333200" + string(rune('0'+index))
			userID := insertUser(t, ctx, pool, "student", regNo, test.status, 2)
			insertStudent(t, ctx, pool, userID, regNo, false)
			status, err := store.RestoreAccount(ctx, userID)
			if test.wantErr != nil {
				if !errors.Is(err, test.wantErr) {
					t.Fatalf("RestoreAccount() error = %v, want %v", err, test.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("RestoreAccount() returned an unexpected error: %v", err)
			}
			if status != test.wantStatus {
				t.Errorf("status = %q, want %q", status, test.wantStatus)
			}
			user, err := store.GetAuthUser(ctx, userID)
			if err != nil {
				t.Fatalf("GetAuthUser() returned an unexpected error: %v", err)
			}
			if user.Status != test.wantStatus {
				t.Errorf("stored status = %q, want %q", user.Status, test.wantStatus)
			}
			if user.TokenVersion != test.wantTokens {
				t.Errorf("token version = %d, want %d", user.TokenVersion, test.wantTokens)
			}
		})
	}
	if _, err := store.RestoreAccount(ctx, 999999); !errors.Is(err, ErrNotFound) {
		t.Errorf("RestoreAccount() for an unknown account error = %v, want ErrNotFound", err)
	}
}

func TestDeleteAccountRefusesProtectedAccounts(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	adminID := insertUser(t, ctx, pool, "admin", "office-admin", StatusRemoved, 0)
	activeID := insertUser(t, ctx, pool, "student", "2023332100", StatusActive, 0)
	insertStudent(t, ctx, pool, activeID, "2023332100", false)
	if _, err := store.DeleteAccount(ctx, adminID); !errors.Is(err, ErrProtected) {
		t.Errorf("deleting an admin error = %v, want ErrProtected", err)
	}
	if _, err := store.DeleteAccount(ctx, activeID); !errors.Is(err, ErrNotDeletable) {
		t.Errorf("deleting an active account error = %v, want ErrNotDeletable", err)
	}
	if _, err := store.DeleteAccount(ctx, 999999); !errors.Is(err, ErrNotFound) {
		t.Errorf("deleting an unknown account error = %v, want ErrNotFound", err)
	}
}

func TestDeleteAccountKeepsTheContentOthersRelyOn(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	crID := seedCR(t, ctx, pool, "2023332200")
	readerID := seedCR(t, ctx, pool, "2023332201")
	if _, err := pool.Exec(ctx, "UPDATE users SET photo_path = '/uploads/photos/cr.png' WHERE id = $1", crID); err != nil {
		t.Fatalf("could not set the photo of the class representative: %v", err)
	}
	notice, err := store.CreateNotice(ctx, NewNotice{
		Source: SourceCR, Title: "Lab moved", Description: "To room 1003",
		Batch: "CSE-23", CreatedBy: crID,
	})
	if err != nil {
		t.Fatalf("CreateNotice() returned an unexpected error: %v", err)
	}
	poll, err := store.CreatePoll(ctx, NewPoll{
		Question: "Which day suits the extra class?", Batch: "CSE-23", Section: "A",
		Options: []string{"Monday", "Tuesday"}, CreatedBy: crID,
	})
	if err != nil {
		t.Fatalf("CreatePoll() returned an unexpected error: %v", err)
	}
	conversation, err := store.GetOrCreateDirect(ctx, crID, readerID)
	if err != nil {
		t.Fatalf("GetOrCreateDirect() returned an unexpected error: %v", err)
	}
	if _, err := store.CreateChatMessage(ctx, NewChatMessage{
		ConversationID: conversation.ID, SenderID: crID, Body: "The lab moved to 1003.",
	}); err != nil {
		t.Fatalf("CreateChatMessage() returned an unexpected error: %v", err)
	}
	if err := store.VoteInPoll(ctx, poll.ID, poll.Options[0].ID, crID, "CSE-23", "A"); err != nil {
		t.Fatalf("VoteInPoll() returned an unexpected error: %v", err)
	}
	removeAccount(t, ctx, pool, crID, StatusRemoved)
	photoPath, err := store.DeleteAccount(ctx, crID)
	if err != nil {
		t.Fatalf("DeleteAccount() returned an unexpected error: %v", err)
	}
	if photoPath != "/uploads/photos/cr.png" {
		t.Errorf("photo path = %q, want the stored photo", photoPath)
	}
	if _, err := store.GetAuthUser(ctx, crID); !errors.Is(err, ErrNotFound) {
		t.Errorf("GetAuthUser() after the deletion error = %v, want ErrNotFound", err)
	}
	notices, err := store.ListBatchCRNotices(ctx, "CSE-23", false)
	if err != nil {
		t.Fatalf("ListBatchCRNotices() returned an unexpected error: %v", err)
	}
	if len(notices) != 1 || notices[0].ID != notice.ID {
		t.Fatalf("notices = %+v, want the notice of the deleted account", notices)
	}
	if notices[0].CreatedByName != "Deleted user" || notices[0].CreatedByPhoto != "" {
		t.Errorf("notice author = %q with photo %q, want a deleted user without a photo",
			notices[0].CreatedByName, notices[0].CreatedByPhoto)
	}
	polls, err := store.ListStudentPolls(ctx, readerID, "CSE-23", "A")
	if err != nil {
		t.Fatalf("ListStudentPolls() returned an unexpected error: %v", err)
	}
	if len(polls) != 1 || polls[0].CreatedByName != "Deleted user" {
		t.Fatalf("polls = %+v, want one poll by a deleted user", polls)
	}
	if polls[0].TotalVotes != 0 {
		t.Errorf("total votes = %d, want 0 once the voter is deleted", polls[0].TotalVotes)
	}
	messages, err := store.ListChatMessages(ctx, conversation.ID, nil, 20)
	if err != nil {
		t.Fatalf("ListChatMessages() returned an unexpected error: %v", err)
	}
	if len(messages) != 1 || messages[0].SenderName != "Deleted user" {
		t.Fatalf("messages = %+v, want one message from a deleted user", messages)
	}
	if messages[0].SenderID != 0 || messages[0].SenderPhotoPath != "" {
		t.Errorf("message sender = %d with photo %q, want no sender and no photo",
			messages[0].SenderID, messages[0].SenderPhotoPath)
	}
	var members int
	if err := pool.QueryRow(ctx,
		"SELECT count(*) FROM conversation_members WHERE user_id = $1", crID).Scan(&members); err != nil {
		t.Fatalf("could not count the conversation members: %v", err)
	}
	if members != 0 {
		t.Errorf("the deleted account still has %d conversation memberships", members)
	}
}

func TestAuditEntries(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	adminID := insertUser(t, ctx, pool, "admin", "admin", StatusActive, 0)
	entries := []struct {
		action  string
		summary string
	}{
		{action: ActionRemove, summary: "student Removed Student (2023332300)"},
		{action: ActionRestore, summary: "student Removed Student (2023332300) is now active"},
		{action: ActionDelete, summary: "teacher Gone Teacher (gone.cse)"},
	}
	for _, entry := range entries {
		if err := store.WriteAudit(ctx, adminID, entry.action, entry.summary); err != nil {
			t.Fatalf("WriteAudit() returned an unexpected error: %v", err)
		}
	}
	rows, err := store.ListAudit(ctx, DefaultAuditLimit)
	if err != nil {
		t.Fatalf("ListAudit() returned an unexpected error: %v", err)
	}
	if len(rows) != len(entries) {
		t.Fatalf("got %d audit rows, want %d", len(rows), len(entries))
	}
	if rows[0].Action != ActionDelete || rows[0].TargetSummary != entries[2].summary {
		t.Errorf("newest row = %+v, want the permanent delete", rows[0])
	}
	if rows[0].AdminID != adminID || rows[0].AdminName != "admin" {
		t.Errorf("audit admin = %d %q, want the seeded admin", rows[0].AdminID, rows[0].AdminName)
	}
	if rows[0].CreatedAt.IsZero() {
		t.Error("the audit row has no timestamp")
	}
}

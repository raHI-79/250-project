package store

import (
	"errors"
	"testing"
	"time"
)

func TestEditAndDeleteStayInsideTheWindow(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	people := seedChatPeople(t, ctx, pool)
	direct, err := store.GetOrCreateDirect(ctx, people["cr"], "cr", people["teacher"])
	if err != nil {
		t.Fatalf("GetOrCreateDirect() returned an unexpected error: %v", err)
	}
	sent, err := store.CreateChatMessage(ctx, NewChatMessage{
		ConversationID: direct.ID,
		SenderID:       people["cr"],
		Body:           "Sir, is the lab on?",
		LinkURL:        "https://example.com/lab",
	})
	if err != nil {
		t.Fatalf("CreateChatMessage() returned an unexpected error: %v", err)
	}
	own, err := store.FindOwnMessage(ctx, sent.ID, people["cr"])
	if err != nil {
		t.Fatalf("FindOwnMessage() returned an unexpected error: %v", err)
	}
	if own.ConversationID != direct.ID || !own.HasAttachment || own.Deleted {
		t.Errorf("own message = %+v, want the open conversation with a link", own)
	}
	if _, err := store.FindOwnMessage(ctx, sent.ID, people["teacher"]); !errors.Is(err, ErrNotFound) {
		t.Errorf("another account reading the message: error = %v, want ErrNotFound", err)
	}
	now := time.Now()
	edited, err := store.EditChatMessage(ctx, sent.ID, people["cr"], "Sir, is the lab on today?", now)
	if err != nil {
		t.Fatalf("EditChatMessage() returned an unexpected error: %v", err)
	}
	if edited.Body != "Sir, is the lab on today?" {
		t.Errorf("body = %q, want the edited one", edited.Body)
	}
	if edited.EditedAt == nil {
		t.Error("the edited message carries no edited_at stamp")
	}
	late := now.Add(EditWindow + time.Second)
	if _, err := store.EditChatMessage(ctx, sent.ID, people["cr"], "Too late", late); !errors.Is(err, ErrEditWindowOver) {
		t.Fatalf("a late edit: error = %v, want ErrEditWindowOver", err)
	}
	if _, err := store.DeleteChatMessage(ctx, sent.ID, people["cr"], late); !errors.Is(err, ErrEditWindowOver) {
		t.Fatalf("a late delete: error = %v, want ErrEditWindowOver", err)
	}
	if _, err := store.EditChatMessage(ctx, sent.ID, people["teacher"], "Not mine", now); !errors.Is(err, ErrEditWindowOver) {
		t.Fatalf("another account editing: error = %v, want ErrEditWindowOver", err)
	}
}

func TestDeleteChatMessageClearsTheContent(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	people := seedChatPeople(t, ctx, pool)
	direct, err := store.GetOrCreateDirect(ctx, people["cr"], "cr", people["teacher"])
	if err != nil {
		t.Fatalf("GetOrCreateDirect() returned an unexpected error: %v", err)
	}
	sent, err := store.CreateChatMessage(ctx, NewChatMessage{
		ConversationID: direct.ID,
		SenderID:       people["cr"],
		Body:           "Here is the manual",
		LinkURL:        "https://example.com/manual",
		FilePath:       "attachments/manual.pdf",
		FileName:       "manual.pdf",
		FileMime:       "application/pdf",
		FileSize:       1024,
	})
	if err != nil {
		t.Fatalf("CreateChatMessage() returned an unexpected error: %v", err)
	}
	conversationID, err := store.DeleteChatMessage(ctx, sent.ID, people["cr"], time.Now())
	if err != nil {
		t.Fatalf("DeleteChatMessage() returned an unexpected error: %v", err)
	}
	if conversationID != direct.ID {
		t.Errorf("conversation = %d, want %d", conversationID, direct.ID)
	}
	messages, err := store.ListChatMessages(ctx, direct.ID, nil, MessagePageSize, people["cr"])
	if err != nil {
		t.Fatalf("ListChatMessages() returned an unexpected error: %v", err)
	}
	if len(messages) != 1 {
		t.Fatalf("got %d messages, want the deleted one still in the thread", len(messages))
	}
	deleted := messages[0]
	if deleted.DeletedAt == nil {
		t.Error("the message carries no deleted_at stamp")
	}
	if deleted.Body != "" || deleted.LinkURL != "" || deleted.FilePath != "" || deleted.FileName != "" {
		t.Errorf("deleted message still carries content: %+v", deleted)
	}
	if _, err := store.DeleteChatMessage(ctx, sent.ID, people["cr"], time.Now()); !errors.Is(err, ErrEditWindowOver) {
		t.Fatalf("deleting twice: error = %v, want ErrEditWindowOver", err)
	}
}

func TestDirectConversationsFollowTheMessagingRules(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	people := seedChatPeople(t, ctx, pool)
	tests := []struct {
		name       string
		viewer     int64
		viewerRole string
		target     int64
		wantError  error
	}{
		{name: "a student writes to a teacher", viewer: people["student"], viewerRole: "student",
			target: people["teacher"]},
		{name: "a student writes to a class representative", viewer: people["student"],
			viewerRole: "student", target: people["cr"]},
		{name: "a student writes to the office", viewer: people["student"], viewerRole: "student",
			target: people["admin"]},
		{name: "a student writes to another student", viewer: people["student"], viewerRole: "student",
			target: people["other"], wantError: ErrContactBlocked},
		{name: "a class representative writes to a student", viewer: people["cr"], viewerRole: "cr",
			target: people["student"]},
		{name: "a teacher writes to a student", viewer: people["teacher"], viewerRole: "teacher",
			target: people["student"]},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := store.GetOrCreateDirect(ctx, test.viewer, test.viewerRole, test.target)
			if test.wantError != nil {
				if !errors.Is(err, test.wantError) {
					t.Fatalf("error = %v, want %v", err, test.wantError)
				}
				return
			}
			if err != nil {
				t.Fatalf("GetOrCreateDirect() returned an unexpected error: %v", err)
			}
		})
	}
}

func TestClearConversationHidesOnlyTheCallersHistory(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	people := seedChatPeople(t, ctx, pool)
	direct, err := store.GetOrCreateDirect(ctx, people["cr"], "cr", people["teacher"])
	if err != nil {
		t.Fatalf("GetOrCreateDirect() returned an unexpected error: %v", err)
	}
	for _, body := range []string{"First", "Second"} {
		_, err := store.CreateChatMessage(ctx, NewChatMessage{
			ConversationID: direct.ID, SenderID: people["teacher"], Body: body,
		})
		if err != nil {
			t.Fatalf("CreateChatMessage() returned an unexpected error: %v", err)
		}
	}
	crViewer := viewerOf(t, ctx, store, people["cr"])
	teacherViewer := viewerOf(t, ctx, store, people["teacher"])
	before, err := store.CountUnreadMessages(ctx, crViewer)
	if err != nil {
		t.Fatalf("CountUnreadMessages() returned an unexpected error: %v", err)
	}
	if before != 2 {
		t.Fatalf("unread = %d, want 2 before the chat is cleared", before)
	}
	if err := store.ClearConversation(ctx, people["cr"], direct.ID, time.Now()); err != nil {
		t.Fatalf("ClearConversation() returned an unexpected error: %v", err)
	}
	cleared, err := store.ListChatMessages(ctx, direct.ID, nil, MessagePageSize, people["cr"])
	if err != nil {
		t.Fatalf("ListChatMessages() returned an unexpected error: %v", err)
	}
	if len(cleared) != 0 {
		t.Errorf("the caller still sees %d messages, want none", len(cleared))
	}
	kept, err := store.ListChatMessages(ctx, direct.ID, nil, MessagePageSize, people["teacher"])
	if err != nil {
		t.Fatalf("ListChatMessages() returned an unexpected error: %v", err)
	}
	if len(kept) != 2 {
		t.Errorf("the other member sees %d messages, want the whole history", len(kept))
	}
	unread, err := store.CountUnreadMessages(ctx, crViewer)
	if err != nil {
		t.Fatalf("CountUnreadMessages() returned an unexpected error: %v", err)
	}
	if unread != 0 {
		t.Errorf("unread = %d, want none once the chat is cleared", unread)
	}
	list, err := store.ListConversations(ctx, crViewer, MessagePageSize)
	if err != nil {
		t.Fatalf("ListConversations() returned an unexpected error: %v", err)
	}
	for _, row := range list {
		if row.ID == direct.ID {
			t.Error("the cleared chat is still in the caller's list")
		}
	}
	others, err := store.ListConversations(ctx, teacherViewer, MessagePageSize)
	if err != nil {
		t.Fatalf("ListConversations() returned an unexpected error: %v", err)
	}
	found := false
	for _, row := range others {
		if row.ID == direct.ID {
			found = true
		}
	}
	if !found {
		t.Error("clearing a chat removed it from the other member's list too")
	}
}

func TestClearedConversationComesBackWithTheNewMessages(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	people := seedChatPeople(t, ctx, pool)
	group, err := store.CreateGroup(ctx, NewGroup{
		Title:     "CSE-23 section A",
		CreatedBy: people["teacher"],
		Members:   []int64{people["cr"]},
	})
	if err != nil {
		t.Fatalf("CreateGroup() returned an unexpected error: %v", err)
	}
	if _, err := store.CreateChatMessage(ctx, NewChatMessage{
		ConversationID: group.ID, SenderID: people["teacher"], Body: "Old news",
	}); err != nil {
		t.Fatalf("CreateChatMessage() returned an unexpected error: %v", err)
	}
	if err := store.ClearConversation(ctx, people["cr"], group.ID, time.Now()); err != nil {
		t.Fatalf("ClearConversation() returned an unexpected error: %v", err)
	}
	members, err := store.ListConversationMembers(ctx, group.ID)
	if err != nil {
		t.Fatalf("ListConversationMembers() returned an unexpected error: %v", err)
	}
	if len(members) != 2 {
		t.Errorf("members = %d, want the caller still in the group", len(members))
	}
	if _, err := store.CreateChatMessage(ctx, NewChatMessage{
		ConversationID: group.ID, SenderID: people["teacher"], Body: "Fresh news",
	}); err != nil {
		t.Fatalf("CreateChatMessage() returned an unexpected error: %v", err)
	}
	crViewer := viewerOf(t, ctx, store, people["cr"])
	back, err := store.ListConversations(ctx, crViewer, MessagePageSize)
	if err != nil {
		t.Fatalf("ListConversations() returned an unexpected error: %v", err)
	}
	found := false
	for _, row := range back {
		if row.ID == group.ID {
			found = true
			if row.LastBody != "Fresh news" {
				t.Errorf("preview = %q, want the message sent after the clear", row.LastBody)
			}
			if row.UnreadCount != 1 {
				t.Errorf("unread = %d, want only the new message", row.UnreadCount)
			}
		}
	}
	if !found {
		t.Fatal("the cleared chat did not come back with the new message")
	}
	messages, err := store.ListChatMessages(ctx, group.ID, nil, MessagePageSize, people["cr"])
	if err != nil {
		t.Fatalf("ListChatMessages() returned an unexpected error: %v", err)
	}
	if len(messages) != 1 || messages[0].Body != "Fresh news" {
		t.Errorf("messages = %+v, want only the newer one", messages)
	}
}

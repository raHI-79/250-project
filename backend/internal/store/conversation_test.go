package store

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5/pgxpool"
	"testing"
	"time"
)

func seedChatPeople(t *testing.T, ctx context.Context, pool *pgxpool.Pool) map[string]int64 {
	t.Helper()
	people := make(map[string]int64)
	studentID := insertUser(t, ctx, pool, "student", "2023331001", StatusActive, 0)
	insertStudent(t, ctx, pool, studentID, "2023331001", false)
	people["student"] = studentID
	crID := insertUser(t, ctx, pool, "student", "2023331002", StatusActive, 0)
	insertStudent(t, ctx, pool, crID, "2023331002", true)
	people["cr"] = crID
	otherID := insertUser(t, ctx, pool, "student", "2022331003", StatusActive, 0)
	if _, err := pool.Exec(ctx,
		`INSERT INTO students (user_id, full_name, reg_no, batch, section, email, phone, blood_group, is_cr)
VALUES ($1, 'Other Batch Student', '2022331003', 'CSE-22', 'B', 'other@student.sust.edu', '01700000003', 'O+', FALSE)`,
		otherID); err != nil {
		t.Fatalf("could not insert the other batch student: %v", err)
	}
	people["other"] = otherID
	sectionID := insertUser(t, ctx, pool, "student", "2023331004", StatusActive, 0)
	if _, err := pool.Exec(ctx,
		`INSERT INTO students (user_id, full_name, reg_no, batch, section, email, phone, blood_group, is_cr)
VALUES ($1, 'Section B Student', '2023331004', 'CSE-23', 'B', 'sectionb@student.sust.edu', '01700000004', 'O+', FALSE)`,
		sectionID); err != nil {
		t.Fatalf("could not insert the section B student: %v", err)
	}
	people["sectionB"] = sectionID
	teacherID := insertUser(t, ctx, pool, "teacher", "T-001", StatusActive, 0)
	insertTeacher(t, ctx, pool, teacherID, "T-001")
	people["teacher"] = teacherID
	secondTeacher := insertUser(t, ctx, pool, "teacher", "T-002", StatusActive, 0)
	if _, err := pool.Exec(ctx,
		`INSERT INTO teachers (user_id, teacher_id, name, email, phone)
VALUES ($1, 'T-002', 'Second Teacher', 'second@sust.edu', '01800000002')`,
		secondTeacher); err != nil {
		t.Fatalf("could not insert the second teacher: %v", err)
	}
	people["teacher2"] = secondTeacher
	people["admin"] = insertUser(t, ctx, pool, "admin", "admin", StatusActive, 0)
	return people
}

func viewerOf(t *testing.T, ctx context.Context, store *Store, userID int64) MessageViewer {
	t.Helper()
	viewer, err := store.GetMessageViewer(ctx, userID)
	if err != nil {
		t.Fatalf("GetMessageViewer() returned an unexpected error: %v", err)
	}
	return viewer
}

func TestGetMessageViewer(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	people := seedChatPeople(t, ctx, pool)
	removed := insertUser(t, ctx, pool, "student", "2023331099", StatusRemoved, 0)
	insertStudent(t, ctx, pool, removed, "2023331099", false)
	tests := []struct {
		name        string
		userID      int64
		wantRole    string
		wantBatch   string
		wantSection string
		wantErr     error
	}{
		{name: "student", userID: people["student"], wantRole: "student", wantBatch: "CSE-23", wantSection: "A"},
		{name: "class representative", userID: people["cr"], wantRole: "cr", wantBatch: "CSE-23", wantSection: "A"},
		{name: "teacher", userID: people["teacher"], wantRole: "teacher"},
		{name: "admin", userID: people["admin"], wantRole: "admin"},
		{name: "a removed account", userID: removed, wantErr: ErrNotFound},
		{name: "an unknown account", userID: 999999, wantErr: ErrNotFound},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			viewer, err := store.GetMessageViewer(ctx, tc.userID)
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("GetMessageViewer() error = %v, want %v", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("GetMessageViewer() returned an unexpected error: %v", err)
			}
			if viewer.Role != tc.wantRole || viewer.Batch != tc.wantBatch || viewer.Section != tc.wantSection {
				t.Errorf("viewer = %+v, want %q/%q/%q", viewer, tc.wantRole, tc.wantBatch, tc.wantSection)
			}
			if viewer.Name == "" {
				t.Error("the viewer has no name")
			}
		})
	}
}

func TestSearchContacts(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	people := seedChatPeople(t, ctx, pool)
	removed := insertUser(t, ctx, pool, "student", "2023331098", StatusRemoved, 0)
	insertStudent(t, ctx, pool, removed, "2023331098", false)
	tests := []struct {
		name       string
		viewerRole string
		search     string
		role       string
		wantFound  bool
		wantMiss   int64
	}{
		{name: "everybody else", viewerRole: "cr", wantFound: true, wantMiss: people["student"]},
		{name: "only teachers", viewerRole: "cr", role: "teacher", wantFound: true},
		{name: "only class representatives", viewerRole: "cr", role: "cr", wantFound: true},
		{name: "only admins", viewerRole: "cr", role: "admin", wantFound: true},
		{name: "by registration number", viewerRole: "cr", search: "2023331002", wantFound: true},
		{name: "no match", viewerRole: "cr", search: "nobody-at-all", wantFound: false},
		{name: "a plain student sees no plain student", viewerRole: "student", role: "student",
			wantFound: false},
		{name: "a plain student still sees the teachers", viewerRole: "student", role: "teacher",
			wantFound: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			contacts, err := store.SearchContacts(ctx, people["student"], tc.viewerRole, tc.search, tc.role)
			if err != nil {
				t.Fatalf("SearchContacts() returned an unexpected error: %v", err)
			}
			if tc.wantFound && len(contacts) == 0 {
				t.Fatal("SearchContacts() found nobody, want at least one contact")
			}
			if !tc.wantFound && len(contacts) != 0 {
				t.Fatalf("SearchContacts() found %d contacts, want none", len(contacts))
			}
			for _, contact := range contacts {
				if contact.UserID == people["student"] {
					t.Error("the search returned the caller")
				}
				if contact.UserID == removed {
					t.Error("the search returned a removed account")
				}
				if tc.role != "" && contact.Role != tc.role {
					t.Errorf("role = %q, want %q", contact.Role, tc.role)
				}
			}
		})
	}
	if _, err := store.SearchContacts(ctx, people["student"], "cr", "", ""); err != nil {
		t.Fatalf("SearchContacts() returned an unexpected error: %v", err)
	}
}

func TestGetOrCreateDirect(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	people := seedChatPeople(t, ctx, pool)
	first, err := store.GetOrCreateDirect(ctx, people["student"], "cr", people["teacher"])
	if err != nil {
		t.Fatalf("GetOrCreateDirect() returned an unexpected error: %v", err)
	}
	if first.Kind != ConversationDirect {
		t.Errorf("kind = %q, want %q", first.Kind, ConversationDirect)
	}
	again, err := store.GetOrCreateDirect(ctx, people["student"], "cr", people["teacher"])
	if err != nil {
		t.Fatalf("GetOrCreateDirect() returned an unexpected error: %v", err)
	}
	if again.ID != first.ID {
		t.Errorf("second call made conversation %d, want %d", again.ID, first.ID)
	}
	swapped, err := store.GetOrCreateDirect(ctx, people["teacher"], "cr", people["student"])
	if err != nil {
		t.Fatalf("GetOrCreateDirect() returned an unexpected error: %v", err)
	}
	if swapped.ID != first.ID {
		t.Errorf("the reversed pair made conversation %d, want %d", swapped.ID, first.ID)
	}
	if _, err := store.GetOrCreateDirect(ctx, people["student"], "cr", people["student"]); !errors.Is(err, ErrSameUser) {
		t.Errorf("a conversation with yourself error = %v, want ErrSameUser", err)
	}
	if _, err := store.GetOrCreateDirect(ctx, people["student"], "cr", 999999); !errors.Is(err, ErrInactiveTarget) {
		t.Errorf("an unknown target error = %v, want ErrInactiveTarget", err)
	}
	recipients, err := store.ConversationRecipients(ctx, first.ID, people["student"])
	if err != nil {
		t.Fatalf("ConversationRecipients() returned an unexpected error: %v", err)
	}
	if len(recipients) != 1 || recipients[0] != people["teacher"] {
		t.Errorf("recipients = %v, want only the teacher", recipients)
	}
}

func TestGroupMembershipRules(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	people := seedChatPeople(t, ctx, pool)
	group, err := store.CreateGroup(ctx, NewGroup{
		Title:     "CSE-23 section A",
		CreatedBy: people["teacher"],
		Members:   []int64{people["student"], people["cr"]},
	})
	if err != nil {
		t.Fatalf("CreateGroup() returned an unexpected error: %v", err)
	}
	if group.Title != "CSE-23 section A" || group.Kind != ConversationGroup {
		t.Fatalf("group = %+v, want a titled group", group)
	}
	members, err := store.ListConversationMembers(ctx, group.ID)
	if err != nil {
		t.Fatalf("ListConversationMembers() returned an unexpected error: %v", err)
	}
	if len(members) != 3 {
		t.Fatalf("the group holds %d members, want 3", len(members))
	}
	roles := make(map[int64]string)
	for _, member := range members {
		roles[member.UserID] = member.Role
	}
	if roles[people["teacher"]] != MemberAdmin {
		t.Errorf("the creator is %q, want the group admin", roles[people["teacher"]])
	}
	if roles[people["student"]] != MemberRegular {
		t.Errorf("an added student is %q, want a member", roles[people["student"]])
	}
	for _, viewerName := range []string{"teacher", "student", "cr"} {
		viewer := viewerOf(t, ctx, store, people[viewerName])
		access, err := store.FindConversationAccess(ctx, group.ID, viewer)
		if err != nil {
			t.Fatalf("FindConversationAccess() for %s returned an unexpected error: %v", viewerName, err)
		}
		if access.MemberCount != 3 {
			t.Errorf("%s sees %d members, want 3", viewerName, access.MemberCount)
		}
	}
	outsider := viewerOf(t, ctx, store, people["other"])
	if _, err := store.FindConversationAccess(ctx, group.ID, outsider); !errors.Is(err, ErrNotFound) {
		t.Errorf("an outsider access error = %v, want ErrNotFound", err)
	}
	added, err := store.AddConversationMembers(ctx, group.ID, people["teacher"],
		[]int64{people["other"], people["student"]})
	if err != nil {
		t.Fatalf("AddConversationMembers() returned an unexpected error: %v", err)
	}
	if added != 1 {
		t.Errorf("added %d members, want only the new one", added)
	}
	if err := store.RemoveConversationMember(ctx, group.ID, people["other"]); err != nil {
		t.Fatalf("RemoveConversationMember() returned an unexpected error: %v", err)
	}
	if err := store.RemoveConversationMember(ctx, group.ID, people["other"]); !errors.Is(err, ErrNotAMember) {
		t.Errorf("removing a stranger error = %v, want ErrNotAMember", err)
	}
	if err := store.RemoveConversationMember(ctx, group.ID, people["teacher"]); !errors.Is(err, ErrLastAdmin) {
		t.Errorf("removing the last admin error = %v, want ErrLastAdmin", err)
	}
	if err := store.SetConversationMemberAdmin(ctx, group.ID, people["teacher"], false); !errors.Is(err, ErrLastAdmin) {
		t.Errorf("demoting the last admin error = %v, want ErrLastAdmin", err)
	}
	if err := store.SetConversationMemberAdmin(ctx, group.ID, people["cr"], true); err != nil {
		t.Fatalf("SetConversationMemberAdmin() returned an unexpected error: %v", err)
	}
	if err := store.SetConversationMemberAdmin(ctx, group.ID, people["teacher"], false); err != nil {
		t.Fatalf("demoting the second admin returned an unexpected error: %v", err)
	}
	if err := store.RenameConversation(ctx, group.ID, "CSE-23 lab group"); err != nil {
		t.Fatalf("RenameConversation() returned an unexpected error: %v", err)
	}
	viewer := viewerOf(t, ctx, store, people["cr"])
	access, err := store.FindConversationAccess(ctx, group.ID, viewer)
	if err != nil {
		t.Fatalf("FindConversationAccess() returned an unexpected error: %v", err)
	}
	if access.Title != "CSE-23 lab group" || !access.IsAdmin() {
		t.Errorf("access = %+v, want the new title and the promoted admin", access)
	}
	if err := store.LeaveConversation(ctx, group.ID, people["cr"]); err != nil {
		t.Fatalf("LeaveConversation() returned an unexpected error: %v", err)
	}
	after, err := store.ListConversationMembers(ctx, group.ID)
	if err != nil {
		t.Fatalf("ListConversationMembers() returned an unexpected error: %v", err)
	}
	admins := 0
	for _, member := range after {
		if member.Role == MemberAdmin {
			admins++
		}
	}
	if len(after) != 2 || admins != 1 {
		t.Errorf("after the last admin left: %d members with %d admins, want 2 and 1", len(after), admins)
	}
	if err := store.LeaveConversation(ctx, group.ID, people["other"]); !errors.Is(err, ErrNotAMember) {
		t.Errorf("leaving a group you are not in error = %v, want ErrNotAMember", err)
	}
}

func TestConversationListAndUnreadCounts(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	people := seedChatPeople(t, ctx, pool)
	direct, err := store.GetOrCreateDirect(ctx, people["teacher"], "cr", people["student"])
	if err != nil {
		t.Fatalf("GetOrCreateDirect() returned an unexpected error: %v", err)
	}
	group, err := store.CreateGroup(ctx, NewGroup{
		Title:     "CSE-23 section A",
		CreatedBy: people["teacher"],
		Members:   []int64{people["student"]},
	})
	if err != nil {
		t.Fatalf("CreateGroup() returned an unexpected error: %v", err)
	}
	for _, message := range []NewChatMessage{
		{ConversationID: direct.ID, SenderID: people["teacher"], Body: "Are you coming?"},
		{ConversationID: direct.ID, SenderID: people["teacher"], Body: "The lab moved"},
		{ConversationID: group.ID, SenderID: people["teacher"], Body: "Extra class tomorrow"},
	} {
		if _, err := store.CreateChatMessage(ctx, message); err != nil {
			t.Fatalf("CreateChatMessage() returned an unexpected error: %v", err)
		}
	}
	studentViewer := viewerOf(t, ctx, store, people["student"])
	teacherViewer := viewerOf(t, ctx, store, people["teacher"])
	unread, err := store.CountUnreadMessages(ctx, studentViewer)
	if err != nil {
		t.Fatalf("CountUnreadMessages() returned an unexpected error: %v", err)
	}
	if unread != 3 {
		t.Errorf("student unread = %d, want 3", unread)
	}
	senderUnread, err := store.CountUnreadMessages(ctx, teacherViewer)
	if err != nil {
		t.Fatalf("CountUnreadMessages() returned an unexpected error: %v", err)
	}
	if senderUnread != 0 {
		t.Errorf("sender unread = %d, want 0", senderUnread)
	}
	rows, err := store.ListConversations(ctx, studentViewer, MaxMessagePage)
	if err != nil {
		t.Fatalf("ListConversations() returned an unexpected error: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("the student sees %d conversations, want 2", len(rows))
	}
	for _, row := range rows {
		if row.Kind == ConversationDirect {
			if row.OtherUserID != people["teacher"] || row.OtherName == "" {
				t.Errorf("direct row = %+v, want the teacher as the other person", row)
			}
			if row.UnreadCount != 2 {
				t.Errorf("direct unread = %d, want 2", row.UnreadCount)
			}
			if row.LastBody != "The lab moved" {
				t.Errorf("preview = %q, want the newest message", row.LastBody)
			}
		}
		if row.Kind == ConversationGroup && row.UnreadCount != 1 {
			t.Errorf("group unread = %d, want 1", row.UnreadCount)
		}
	}
	if err := store.MarkConversationRead(ctx, people["student"], direct.ID, time.Now()); err != nil {
		t.Fatalf("MarkConversationRead() returned an unexpected error: %v", err)
	}
	afterRead, err := store.CountUnreadMessages(ctx, studentViewer)
	if err != nil {
		t.Fatalf("CountUnreadMessages() returned an unexpected error: %v", err)
	}
	if afterRead != 1 {
		t.Errorf("unread after reading the direct chat = %d, want 1", afterRead)
	}
	outsider := viewerOf(t, ctx, store, people["other"])
	hidden, err := store.ListConversations(ctx, outsider, MaxMessagePage)
	if err != nil {
		t.Fatalf("ListConversations() returned an unexpected error: %v", err)
	}
	if len(hidden) != 0 {
		t.Errorf("another batch sees %d conversations, want none", len(hidden))
	}
	if count, err := store.CountUnreadMessages(ctx, outsider); err != nil || count != 0 {
		t.Errorf("another batch unread = %d, %v, want 0 and no error", count, err)
	}
}

func TestListChatMessagesPaging(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	people := seedChatPeople(t, ctx, pool)
	direct, err := store.GetOrCreateDirect(ctx, people["teacher"], "cr", people["student"])
	if err != nil {
		t.Fatalf("GetOrCreateDirect() returned an unexpected error: %v", err)
	}
	for _, body := range []string{"first", "second", "third"} {
		if _, err := store.CreateChatMessage(ctx, NewChatMessage{
			ConversationID: direct.ID, SenderID: people["teacher"], Body: body,
		}); err != nil {
			t.Fatalf("CreateChatMessage() returned an unexpected error: %v", err)
		}
	}
	newest, err := store.ListChatMessages(ctx, direct.ID, nil, 2, people["cr"])
	if err != nil {
		t.Fatalf("ListChatMessages() returned an unexpected error: %v", err)
	}
	if len(newest) != 2 || newest[0].Body != "third" {
		t.Fatalf("messages = %+v, want the two newest with third first", newest)
	}
	if newest[0].SenderName == "" || newest[0].SenderRole != "teacher" {
		t.Errorf("message = %+v, want the sender name and role", newest[0])
	}
	older, err := store.ListChatMessages(ctx, direct.ID, &newest[1].CreatedAt, 10, people["cr"])
	if err != nil {
		t.Fatalf("ListChatMessages() returned an unexpected error: %v", err)
	}
	if len(older) != 1 || older[0].Body != "first" {
		t.Errorf("older messages = %+v, want only the first one", older)
	}
}

func TestCreateChatMessageKeepsTheAttachment(t *testing.T) {
	ctx, store, pool := newTestStore(t)
	people := seedChatPeople(t, ctx, pool)
	direct, err := store.GetOrCreateDirect(ctx, people["cr"], "cr", people["teacher"])
	if err != nil {
		t.Fatalf("GetOrCreateDirect() returned an unexpected error: %v", err)
	}
	created, err := store.CreateChatMessage(ctx, NewChatMessage{
		ConversationID: direct.ID,
		SenderID:       people["cr"],
		Body:           "The routine is attached",
		LinkURL:        "https://drive.google.com/file/x",
		FilePath:       "/uploads/files/0123456789abcdef0123456789abcdef.xlsx",
		FileName:       "routine.xlsx",
		FileMime:       "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		FileSize:       4096,
	})
	if err != nil {
		t.Fatalf("CreateChatMessage() returned an unexpected error: %v", err)
	}
	if created.ID == 0 || created.CreatedAt.IsZero() {
		t.Fatalf("created message = %+v, want an id and a timestamp", created)
	}
	messages, err := store.ListChatMessages(ctx, direct.ID, nil, 10, people["cr"])
	if err != nil {
		t.Fatalf("ListChatMessages() returned an unexpected error: %v", err)
	}
	if len(messages) != 1 {
		t.Fatalf("got %d messages, want 1", len(messages))
	}
	stored := messages[0]
	if stored.FileName != "routine.xlsx" || stored.FileSize != 4096 {
		t.Errorf("stored file = %+v, want the name and size kept", stored)
	}
	if stored.LinkURL != "https://drive.google.com/file/x" {
		t.Errorf("link = %q, want the stored link", stored.LinkURL)
	}
	viewer := viewerOf(t, ctx, store, people["cr"])
	rows, err := store.ListConversations(ctx, viewer, MaxMessagePage)
	if err != nil {
		t.Fatalf("ListConversations() returned an unexpected error: %v", err)
	}
	if len(rows) != 1 || rows[0].LastMessageAt == nil {
		t.Fatalf("conversations = %+v, want one with an activity time", rows)
	}
}

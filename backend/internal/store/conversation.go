package store

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"time"
)

const (
	ConversationDirect = "direct"
	ConversationGroup  = "group"
	ScopeBatch         = "batch"
	ScopeSection       = "section"
	ScopeAllStudents   = "all_students"
	ScopeAllTeachers   = "all_teachers"
	ContactsLimit      = 20
	MessagePageSize    = 50
	MaxMessagePage     = 100
)

var (
	ErrNotAMember     = errors.New("store: the account is not part of this conversation")
	ErrNoRecipient    = errors.New("store: the conversation has no other member")
	ErrSameUser       = errors.New("store: a direct conversation needs two different people")
	ErrInactiveTarget = errors.New("store: the other account is not active")
)

type MessageViewer struct {
	UserID  int64
	Name    string
	Role    string
	Batch   string
	Section string
}

func (v MessageViewer) learner() bool {
	return v.Role == "student" || v.Role == "cr"
}

func (v MessageViewer) teacher() bool {
	return v.Role == "teacher"
}

type Conversation struct {
	ID            int64
	Kind          string
	Scope         string
	Batch         string
	Section       string
	CreatedBy     int64
	CreatedAt     time.Time
	LastMessageAt *time.Time
}

type ConversationRow struct {
	Conversation
	OtherUserID    int64
	OtherName      string
	OtherPhotoPath string
	LastSenderName string
	LastBody       string
	LastFileName   string
	UnreadCount    int
}

type ConversationAccess struct {
	Conversation
	CanPost bool
}

type NewGroup struct {
	Scope     string
	Batch     string
	Section   string
	CreatedBy int64
}

type NewChatMessage struct {
	ConversationID int64
	SenderID       int64
	Body           string
	LinkURL        string
	FilePath       string
	FileName       string
	FileMime       string
	FileSize       int64
}

type ChatMessageRow struct {
	ID              int64
	SenderID        int64
	SenderName      string
	SenderRole      string
	SenderPhotoPath string
	Body            string
	LinkURL         string
	FilePath        string
	FileName        string
	FileMime        string
	FileSize        int64
	CreatedAt       time.Time
}

type ContactRow struct {
	UserID      int64
	Name        string
	Role        string
	Batch       string
	Section     string
	Designation string
	PhotoPath   string
}

const visibleConversation = `(
    c.created_by = $1
    OR EXISTS (SELECT 1 FROM conversation_members cm WHERE cm.conversation_id = c.id AND cm.user_id = $1)
    OR (c.kind = 'group' AND (
        (c.scope = 'batch' AND $2::bool AND c.batch = $3::text)
        OR (c.scope = 'section' AND $2::bool AND c.batch = $3::text AND c.section = $4::text)
        OR (c.scope = 'all_students' AND $2::bool)
        OR (c.scope = 'all_teachers' AND $5::bool)
    ))
)`

const (
	messageViewerSQL = `SELECT u.id,
COALESCE(s.full_name, t.name, u.identifier),
CASE WHEN u.role = 'student' AND COALESCE(s.is_cr, FALSE) THEN 'cr' ELSE u.role END,
COALESCE(s.batch, ''), COALESCE(s.section, '')
FROM users u
LEFT JOIN students s ON s.user_id = u.id
LEFT JOIN teachers t ON t.user_id = u.id
WHERE u.id = $1 AND u.status = 'active'`
	searchContactsSQL = `SELECT u.id,
COALESCE(s.full_name, t.name, u.identifier),
CASE WHEN u.role = 'student' AND COALESCE(s.is_cr, FALSE) THEN 'cr' ELSE u.role END AS effective_role,
COALESCE(s.batch, ''), COALESCE(s.section, ''), COALESCE(t.designation, ''),
COALESCE(u.photo_path, '')
FROM users u
LEFT JOIN students s ON s.user_id = u.id
LEFT JOIN teachers t ON t.user_id = u.id
WHERE u.status = 'active' AND u.id <> $1
  AND ($2::text = '' OR COALESCE(s.full_name, t.name, u.identifier) ILIKE '%' || $2::text || '%'
       OR u.identifier ILIKE '%' || $2::text || '%')
  AND ($3::text = '' OR $3::text =
       CASE WHEN u.role = 'student' AND COALESCE(s.is_cr, FALSE) THEN 'cr' ELSE u.role END)
ORDER BY effective_role, COALESCE(s.full_name, t.name, u.identifier), u.id
LIMIT $4`
	activeUserRoleSQL = `SELECT u.role FROM users u WHERE u.id = $1 AND u.status = 'active'`
	lockDirectPairSQL = `SELECT pg_advisory_xact_lock(hashtext('direct:' || LEAST($1::bigint, $2::bigint) || ':' || GREATEST($1::bigint, $2::bigint)))`
	findDirectSQL     = `SELECT c.id, c.kind, COALESCE(c.scope, ''), COALESCE(c.batch, ''), COALESCE(c.section, ''),
c.created_by, c.created_at, c.last_message_at
FROM conversations c
JOIN conversation_members a ON a.conversation_id = c.id AND a.user_id = $1
JOIN conversation_members b ON b.conversation_id = c.id AND b.user_id = $2
WHERE c.kind = 'direct'
LIMIT 1`
	insertDirectSQL = `INSERT INTO conversations (kind, created_by)
VALUES ('direct', $1)
RETURNING id, kind, COALESCE(scope, ''), COALESCE(batch, ''), COALESCE(section, ''),
created_by, created_at, last_message_at`
	insertMemberSQL = `INSERT INTO conversation_members (conversation_id, user_id)
VALUES ($1, $2)
ON CONFLICT DO NOTHING`
	findGroupSQL = `SELECT c.id, c.kind, COALESCE(c.scope, ''), COALESCE(c.batch, ''), COALESCE(c.section, ''),
c.created_by, c.created_at, c.last_message_at
FROM conversations c
WHERE c.kind = 'group' AND c.created_by = $1 AND c.scope = $2
  AND COALESCE(c.batch, '') = $3::text AND COALESCE(c.section, '') = $4::text`
	insertGroupSQL = `INSERT INTO conversations (kind, scope, batch, section, created_by)
VALUES ('group', $2, NULLIF($3::text, ''), NULLIF($4::text, ''), $1)
RETURNING id, kind, COALESCE(scope, ''), COALESCE(batch, ''), COALESCE(section, ''),
created_by, created_at, last_message_at`
	listConversationsSQL = `SELECT c.id, c.kind, COALESCE(c.scope, ''), COALESCE(c.batch, ''), COALESCE(c.section, ''),
c.created_by, c.created_at, c.last_message_at,
COALESCE(other.user_id, 0), COALESCE(other.name, ''), COALESCE(other.photo_path, ''),
COALESCE(last.sender_name, ''), COALESCE(last.body, ''), COALESCE(last.file_name, ''),
(SELECT COUNT(*) FROM messages m2
 WHERE m2.conversation_id = c.id AND m2.sender_id <> $1
   AND (r.last_read_at IS NULL OR m2.created_at > r.last_read_at))
FROM conversations c
LEFT JOIN conversation_reads r ON r.conversation_id = c.id AND r.user_id = $1
LEFT JOIN LATERAL (
    SELECT cm.user_id,
           COALESCE(s.full_name, t.name, u.identifier) AS name,
           COALESCE(u.photo_path, '') AS photo_path
    FROM conversation_members cm
    JOIN users u ON u.id = cm.user_id
    LEFT JOIN students s ON s.user_id = u.id
    LEFT JOIN teachers t ON t.user_id = u.id
    WHERE cm.conversation_id = c.id AND cm.user_id <> $1
    LIMIT 1
) other ON c.kind = 'direct'
LEFT JOIN LATERAL (
    SELECT m.body, m.file_name,
           COALESCE(s.full_name, t.name, u.identifier) AS sender_name
    FROM messages m
    JOIN users u ON u.id = m.sender_id
    LEFT JOIN students s ON s.user_id = u.id
    LEFT JOIN teachers t ON t.user_id = u.id
    WHERE m.conversation_id = c.id
    ORDER BY m.created_at DESC, m.id DESC
    LIMIT 1
) last ON TRUE
WHERE ` + visibleConversation + `
ORDER BY COALESCE(c.last_message_at, c.created_at) DESC, c.id DESC
LIMIT $6`
	findConversationAccessSQL = `SELECT c.id, c.kind, COALESCE(c.scope, ''), COALESCE(c.batch, ''), COALESCE(c.section, ''),
c.created_by, c.created_at, c.last_message_at,
CASE WHEN c.kind = 'direct'
     THEN EXISTS (SELECT 1 FROM conversation_members cm WHERE cm.conversation_id = c.id AND cm.user_id = $1)
     ELSE c.created_by = $1 END
FROM conversations c
WHERE c.id = $6 AND ` + visibleConversation
	listChatMessagesSQL = `SELECT m.id, m.sender_id,
COALESCE(s.full_name, t.name, u.identifier),
CASE WHEN u.role = 'student' AND COALESCE(s.is_cr, FALSE) THEN 'cr' ELSE u.role END,
COALESCE(u.photo_path, ''),
m.body, COALESCE(m.link_url, ''), COALESCE(m.file_path, ''), COALESCE(m.file_name, ''),
COALESCE(m.file_mime, ''), COALESCE(m.file_size, 0), m.created_at
FROM messages m
JOIN users u ON u.id = m.sender_id
LEFT JOIN students s ON s.user_id = u.id
LEFT JOIN teachers t ON t.user_id = u.id
WHERE m.conversation_id = $1 AND ($2::timestamptz IS NULL OR m.created_at < $2::timestamptz)
ORDER BY m.created_at DESC, m.id DESC
LIMIT $3`
	insertChatMessageSQL = `INSERT INTO messages
(conversation_id, sender_id, body, link_url, file_path, file_name, file_mime, file_size)
VALUES ($1, $2, $3, NULLIF($4::text, ''), NULLIF($5::text, ''), NULLIF($6::text, ''),
NULLIF($7::text, ''), NULLIF($8::bigint, 0))
RETURNING id, created_at`
	touchConversationSQL = `UPDATE conversations SET last_message_at = $2 WHERE id = $1`
	markReadSQL          = `INSERT INTO conversation_reads (user_id, conversation_id, last_read_at)
VALUES ($1, $2, $3)
ON CONFLICT (user_id, conversation_id) DO UPDATE SET last_read_at = EXCLUDED.last_read_at`
	unreadCountSQL = `SELECT COUNT(*)
FROM messages m
JOIN conversations c ON c.id = m.conversation_id
LEFT JOIN conversation_reads r ON r.conversation_id = c.id AND r.user_id = $1
WHERE m.sender_id <> $1
  AND (r.last_read_at IS NULL OR m.created_at > r.last_read_at)
  AND ` + visibleConversation
	directRecipientSQL = `SELECT cm.user_id
FROM conversation_members cm
WHERE cm.conversation_id = $1 AND cm.user_id <> $2
LIMIT 1`
)

func (s *Store) GetMessageViewer(ctx context.Context, userID int64) (MessageViewer, error) {
	var viewer MessageViewer
	err := s.pool.QueryRow(ctx, messageViewerSQL, userID).Scan(
		&viewer.UserID,
		&viewer.Name,
		&viewer.Role,
		&viewer.Batch,
		&viewer.Section,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return MessageViewer{}, ErrNotFound
	}
	if err != nil {
		return MessageViewer{}, fmt.Errorf("store: could not load the messaging viewer %d: %w", userID, err)
	}
	return viewer, nil
}

func (s *Store) SearchContacts(ctx context.Context, viewerID int64, search, role string) ([]ContactRow, error) {
	rows, err := s.pool.Query(ctx, searchContactsSQL, viewerID, search, role, ContactsLimit)
	if err != nil {
		return nil, fmt.Errorf("store: could not search the contacts: %w", err)
	}
	defer rows.Close()
	contacts := make([]ContactRow, 0)
	for rows.Next() {
		var contact ContactRow
		err := rows.Scan(
			&contact.UserID,
			&contact.Name,
			&contact.Role,
			&contact.Batch,
			&contact.Section,
			&contact.Designation,
			&contact.PhotoPath,
		)
		if err != nil {
			return nil, fmt.Errorf("store: could not scan a contact: %w", err)
		}
		contacts = append(contacts, contact)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: could not read the contacts: %w", err)
	}
	return contacts, nil
}

func (s *Store) GetOrCreateDirect(ctx context.Context, viewerID, otherID int64) (Conversation, error) {
	if viewerID == otherID {
		return Conversation{}, ErrSameUser
	}
	var role string
	err := s.pool.QueryRow(ctx, activeUserRoleSQL, otherID).Scan(&role)
	if errors.Is(err, pgx.ErrNoRows) {
		return Conversation{}, ErrInactiveTarget
	}
	if err != nil {
		return Conversation{}, fmt.Errorf("store: could not load the other account %d: %w", otherID, err)
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Conversation{}, fmt.Errorf("store: could not start the conversation transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, lockDirectPairSQL, viewerID, otherID); err != nil {
		return Conversation{}, fmt.Errorf("store: could not lock the conversation pair: %w", err)
	}
	conversation, err := scanConversation(tx.QueryRow(ctx, findDirectSQL, viewerID, otherID))
	if err == nil {
		return conversation, tx.Commit(ctx)
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return Conversation{}, fmt.Errorf("store: could not look for a direct conversation: %w", err)
	}
	conversation, err = scanConversation(tx.QueryRow(ctx, insertDirectSQL, viewerID))
	if err != nil {
		return Conversation{}, fmt.Errorf("store: could not create the direct conversation: %w", err)
	}
	for _, member := range []int64{viewerID, otherID} {
		if _, err := tx.Exec(ctx, insertMemberSQL, conversation.ID, member); err != nil {
			return Conversation{}, fmt.Errorf("store: could not add the member %d: %w", member, err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return Conversation{}, fmt.Errorf("store: could not commit the direct conversation: %w", err)
	}
	return conversation, nil
}

func (s *Store) GetOrCreateGroup(ctx context.Context, group NewGroup) (Conversation, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Conversation{}, fmt.Errorf("store: could not start the group transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	conversation, err := scanConversation(
		tx.QueryRow(ctx, findGroupSQL, group.CreatedBy, group.Scope, group.Batch, group.Section),
	)
	if err == nil {
		return conversation, tx.Commit(ctx)
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return Conversation{}, fmt.Errorf("store: could not look for a group conversation: %w", err)
	}
	conversation, err = scanConversation(
		tx.QueryRow(ctx, insertGroupSQL, group.CreatedBy, group.Scope, group.Batch, group.Section),
	)
	if err != nil {
		return Conversation{}, fmt.Errorf("store: could not create the group conversation: %w", err)
	}
	if _, err := tx.Exec(ctx, insertMemberSQL, conversation.ID, group.CreatedBy); err != nil {
		return Conversation{}, fmt.Errorf("store: could not add the group creator: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return Conversation{}, fmt.Errorf("store: could not commit the group conversation: %w", err)
	}
	return conversation, nil
}

func (s *Store) ListConversations(ctx context.Context, viewer MessageViewer, limit int) ([]ConversationRow, error) {
	if limit <= 0 || limit > MaxMessagePage {
		limit = MaxMessagePage
	}
	args := append(viewerArgs(viewer), limit)
	rows, err := s.pool.Query(ctx, listConversationsSQL, args...)
	if err != nil {
		return nil, fmt.Errorf("store: could not list the conversations: %w", err)
	}
	defer rows.Close()
	conversations := make([]ConversationRow, 0)
	for rows.Next() {
		var row ConversationRow
		err := rows.Scan(
			&row.ID,
			&row.Kind,
			&row.Scope,
			&row.Batch,
			&row.Section,
			&row.CreatedBy,
			&row.CreatedAt,
			&row.LastMessageAt,
			&row.OtherUserID,
			&row.OtherName,
			&row.OtherPhotoPath,
			&row.LastSenderName,
			&row.LastBody,
			&row.LastFileName,
			&row.UnreadCount,
		)
		if err != nil {
			return nil, fmt.Errorf("store: could not scan a conversation: %w", err)
		}
		conversations = append(conversations, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: could not read the conversations: %w", err)
	}
	return conversations, nil
}

func (s *Store) FindConversationAccess(ctx context.Context, conversationID int64, viewer MessageViewer) (ConversationAccess, error) {
	var access ConversationAccess
	args := append(viewerArgs(viewer), conversationID)
	err := s.pool.QueryRow(ctx, findConversationAccessSQL, args...).Scan(
		&access.ID,
		&access.Kind,
		&access.Scope,
		&access.Batch,
		&access.Section,
		&access.CreatedBy,
		&access.CreatedAt,
		&access.LastMessageAt,
		&access.CanPost,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return ConversationAccess{}, ErrNotFound
	}
	if err != nil {
		return ConversationAccess{}, fmt.Errorf("store: could not load the conversation %d: %w", conversationID, err)
	}
	return access, nil
}

func (s *Store) ListChatMessages(ctx context.Context, conversationID int64, before *time.Time, limit int) ([]ChatMessageRow, error) {
	if limit <= 0 || limit > MaxMessagePage {
		limit = MessagePageSize
	}
	rows, err := s.pool.Query(ctx, listChatMessagesSQL, conversationID, before, limit)
	if err != nil {
		return nil, fmt.Errorf("store: could not list the messages of the conversation %d: %w", conversationID, err)
	}
	defer rows.Close()
	messages := make([]ChatMessageRow, 0)
	for rows.Next() {
		var row ChatMessageRow
		err := rows.Scan(
			&row.ID,
			&row.SenderID,
			&row.SenderName,
			&row.SenderRole,
			&row.SenderPhotoPath,
			&row.Body,
			&row.LinkURL,
			&row.FilePath,
			&row.FileName,
			&row.FileMime,
			&row.FileSize,
			&row.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("store: could not scan a message: %w", err)
		}
		messages = append(messages, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: could not read the messages: %w", err)
	}
	return messages, nil
}

func (s *Store) CreateChatMessage(ctx context.Context, message NewChatMessage) (ChatMessageRow, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return ChatMessageRow{}, fmt.Errorf("store: could not start the message transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var id int64
	var createdAt time.Time
	err = tx.QueryRow(ctx, insertChatMessageSQL,
		message.ConversationID,
		message.SenderID,
		message.Body,
		message.LinkURL,
		message.FilePath,
		message.FileName,
		message.FileMime,
		message.FileSize,
	).Scan(&id, &createdAt)
	if err != nil {
		return ChatMessageRow{}, fmt.Errorf("store: could not create the message: %w", err)
	}
	if _, err := tx.Exec(ctx, touchConversationSQL, message.ConversationID, createdAt); err != nil {
		return ChatMessageRow{}, fmt.Errorf("store: could not update the conversation activity: %w", err)
	}
	if _, err := tx.Exec(ctx, markReadSQL, message.SenderID, message.ConversationID, createdAt); err != nil {
		return ChatMessageRow{}, fmt.Errorf("store: could not mark the conversation read: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return ChatMessageRow{}, fmt.Errorf("store: could not commit the message: %w", err)
	}
	return ChatMessageRow{
		ID:        id,
		SenderID:  message.SenderID,
		Body:      message.Body,
		LinkURL:   message.LinkURL,
		FilePath:  message.FilePath,
		FileName:  message.FileName,
		FileMime:  message.FileMime,
		FileSize:  message.FileSize,
		CreatedAt: createdAt,
	}, nil
}

func (s *Store) MarkConversationRead(ctx context.Context, userID, conversationID int64, at time.Time) error {
	if _, err := s.pool.Exec(ctx, markReadSQL, userID, conversationID, at); err != nil {
		return fmt.Errorf("store: could not mark the conversation %d read: %w", conversationID, err)
	}
	return nil
}

func (s *Store) CountUnreadMessages(ctx context.Context, viewer MessageViewer) (int, error) {
	var count int
	if err := s.pool.QueryRow(ctx, unreadCountSQL, viewerArgs(viewer)...).Scan(&count); err != nil {
		return 0, fmt.Errorf("store: could not count the unread messages: %w", err)
	}
	return count, nil
}

func (s *Store) DirectRecipient(ctx context.Context, conversationID, senderID int64) (int64, error) {
	var userID int64
	err := s.pool.QueryRow(ctx, directRecipientSQL, conversationID, senderID).Scan(&userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrNoRecipient
	}
	if err != nil {
		return 0, fmt.Errorf("store: could not load the recipient of the conversation %d: %w", conversationID, err)
	}
	return userID, nil
}

func viewerArgs(viewer MessageViewer) []any {
	return []any{viewer.UserID, viewer.learner(), viewer.Batch, viewer.Section, viewer.teacher()}
}

func scanConversation(row pgx.Row) (Conversation, error) {
	var conversation Conversation
	err := row.Scan(
		&conversation.ID,
		&conversation.Kind,
		&conversation.Scope,
		&conversation.Batch,
		&conversation.Section,
		&conversation.CreatedBy,
		&conversation.CreatedAt,
		&conversation.LastMessageAt,
	)
	return conversation, err
}

package store

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/raHI-79/250-project/backend/internal/validate"
	"time"
)

const (
	ConversationDirect = "direct"
	ConversationGroup  = "group"
	MemberAdmin        = "admin"
	MemberRegular      = "member"
	ContactsLimit      = 20
	MessagePageSize    = 50
	MaxMessagePage     = 100
	MaxGroupMembers    = 500
	EditWindow         = 2 * time.Minute
)

var (
	ErrNotAMember     = errors.New("store: the account is not part of this conversation")
	ErrNoRecipient    = errors.New("store: the conversation has no other member")
	ErrSameUser       = errors.New("store: a direct conversation needs two different people")
	ErrInactiveTarget = errors.New("store: the other account is not active")
	ErrLastAdmin      = errors.New("store: a group keeps at least one admin")
	ErrNotAGroup      = errors.New("store: this conversation is not a group")
	ErrContactBlocked = errors.New("store: this account may not start a chat with that one")
	ErrEditWindowOver = errors.New("store: this message can no longer be edited or deleted")
)

type MessageViewer struct {
	UserID  int64
	Name    string
	Role    string
	Batch   string
	Section string
}

type Conversation struct {
	ID            int64
	Kind          string
	Title         string
	CreatedBy     int64
	CreatedAt     time.Time
	LastMessageAt *time.Time
}

type ConversationRow struct {
	Conversation
	OtherUserID    int64
	OtherName      string
	OtherPhotoPath string
	OtherRole      string
	LastSenderName string
	LastBody       string
	LastFileName   string
	UnreadCount    int
	MemberRole     string
	MemberCount    int
}

type ConversationAccess struct {
	Conversation
	MemberRole  string
	MemberCount int
}

func (a ConversationAccess) IsAdmin() bool {
	return a.MemberRole == MemberAdmin
}

type NewGroup struct {
	Title     string
	CreatedBy int64
	Members   []int64
}

type MemberRow struct {
	UserID      int64
	Name        string
	Role        string
	AccountRole string
	Batch       string
	Section     string
	Designation string
	PhotoPath   string
	AddedAt     time.Time
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
	EditedAt        *time.Time
	DeletedAt       *time.Time
}

type OwnMessage struct {
	ConversationID int64
	CreatedAt      time.Time
	Deleted        bool
	HasAttachment  bool
	FilePath       string
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

const memberOfConversation = `EXISTS (
    SELECT 1 FROM conversation_members cm WHERE cm.conversation_id = c.id AND cm.user_id = $1
)`

const conversationColumns = `c.id, c.kind, COALESCE(c.title, ''), COALESCE(c.created_by, 0),
c.created_at, c.last_message_at`

const personName = `COALESCE(s.full_name, t.name, u.identifier, '')`

const effectiveRole = `CASE WHEN u.role = 'student' AND COALESCE(s.is_cr, FALSE) THEN 'cr' ELSE u.role END`

const (
	messageViewerSQL = `SELECT u.id, ` + personName + `, ` + effectiveRole + `,
COALESCE(s.batch, ''), COALESCE(s.section, '')
FROM users u
LEFT JOIN students s ON s.user_id = u.id
LEFT JOIN teachers t ON t.user_id = u.id
WHERE u.id = $1 AND u.status = 'active'`
	searchContactsSQL = `SELECT u.id, ` + personName + `, ` + effectiveRole + ` AS role,
COALESCE(s.batch, ''), COALESCE(s.section, ''), COALESCE(t.designation, ''),
COALESCE(u.photo_path, '')
FROM users u
LEFT JOIN students s ON s.user_id = u.id
LEFT JOIN teachers t ON t.user_id = u.id
WHERE u.status = 'active' AND u.id <> $1
  AND ($2::text = '' OR ` + personName + ` ILIKE '%' || $2::text || '%'
       OR u.identifier ILIKE '%' || $2::text || '%')
  AND ($3::text = '' OR $3::text = ` + effectiveRole + `)
  AND ($5::text <> 'student' OR ` + effectiveRole + ` <> 'student')
ORDER BY ` + effectiveRole + `, ` + personName + `, u.id
LIMIT $4`
	activeUserRoleSQL = `SELECT ` + effectiveRole + `
FROM users u
LEFT JOIN students s ON s.user_id = u.id
WHERE u.id = $1 AND u.status = 'active'`
	activeUserIDsSQL = `SELECT u.id FROM users u WHERE u.status = 'active' AND u.id = ANY($1::bigint[])`
	batchMembersSQL  = `SELECT u.id
FROM users u
JOIN students s ON s.user_id = u.id
WHERE u.status = 'active' AND s.batch = $1::text
  AND ($2::text = '' OR s.section = $2::text)
ORDER BY u.id`
	lockDirectPairSQL = `SELECT pg_advisory_xact_lock(hashtext('direct:' || LEAST($1::bigint, $2::bigint) || ':' || GREATEST($1::bigint, $2::bigint)))`
	findDirectSQL     = `SELECT ` + conversationColumns + `
FROM conversations c
JOIN conversation_members a ON a.conversation_id = c.id AND a.user_id = $1
JOIN conversation_members b ON b.conversation_id = c.id AND b.user_id = $2
WHERE c.kind = 'direct'
LIMIT 1`
	insertDirectSQL = `INSERT INTO conversations (kind, created_by)
VALUES ('direct', $1)
RETURNING id, kind, COALESCE(title, ''), COALESCE(created_by, 0), created_at, last_message_at`
	insertMemberSQL = `INSERT INTO conversation_members (conversation_id, user_id, role, added_by)
VALUES ($1, $2, $3, NULLIF($4::bigint, 0))
ON CONFLICT (conversation_id, user_id) DO NOTHING`
	insertGroupSQL = `INSERT INTO conversations (kind, title, created_by)
VALUES ('group', $1, $2)
RETURNING id, kind, COALESCE(title, ''), COALESCE(created_by, 0), created_at, last_message_at`
	renameGroupSQL       = `UPDATE conversations SET title = $2 WHERE id = $1 AND kind = 'group' RETURNING id`
	listConversationsSQL = `SELECT ` + conversationColumns + `,
COALESCE(other.user_id, 0), COALESCE(other.name, ''), COALESCE(other.photo_path, ''),
COALESCE(other.role, ''),
COALESCE(last.sender_name, ''), COALESCE(last.body, ''), COALESCE(last.file_name, ''),
(SELECT COUNT(*) FROM messages m2
 WHERE m2.conversation_id = c.id AND COALESCE(m2.sender_id, 0) <> $1
   AND (r.last_read_at IS NULL OR m2.created_at > r.last_read_at)
   AND m2.created_at > COALESCE(cleared.cleared_at, '-infinity'::timestamptz)),
COALESCE(mine.role, ''),
(SELECT COUNT(*) FROM conversation_members cm2 WHERE cm2.conversation_id = c.id)
FROM conversations c
JOIN conversation_members mine ON mine.conversation_id = c.id AND mine.user_id = $1
LEFT JOIN conversation_reads r ON r.conversation_id = c.id AND r.user_id = $1
LEFT JOIN conversation_clears cleared ON cleared.conversation_id = c.id AND cleared.user_id = $1
LEFT JOIN LATERAL (
    SELECT cm.user_id, ` + personName + ` AS name, COALESCE(u.photo_path, '') AS photo_path,
           CASE WHEN u.role = 'student' AND COALESCE(s.is_cr, FALSE) THEN 'cr' ELSE u.role END AS role
    FROM conversation_members cm
    JOIN users u ON u.id = cm.user_id
    LEFT JOIN students s ON s.user_id = u.id
    LEFT JOIN teachers t ON t.user_id = u.id
    WHERE cm.conversation_id = c.id AND cm.user_id <> $1
    LIMIT 1
) other ON c.kind = 'direct'
LEFT JOIN LATERAL (
    SELECT m.body, m.file_name, m.created_at,
           COALESCE(s.full_name, t.name, u.identifier, 'Deleted user') AS sender_name
    FROM messages m
    LEFT JOIN users u ON u.id = m.sender_id
    LEFT JOIN students s ON s.user_id = u.id
    LEFT JOIN teachers t ON t.user_id = u.id
    WHERE m.conversation_id = c.id
      AND m.created_at > COALESCE(cleared.cleared_at, '-infinity'::timestamptz)
    ORDER BY m.created_at DESC, m.id DESC
    LIMIT 1
) last ON TRUE
WHERE cleared.cleared_at IS NULL
   OR EXISTS (SELECT 1 FROM messages m3
              WHERE m3.conversation_id = c.id AND m3.created_at > cleared.cleared_at)
ORDER BY COALESCE(last.created_at, c.created_at) DESC, c.id DESC
LIMIT $2`
	findConversationAccessSQL = `SELECT ` + conversationColumns + `,
COALESCE(mine.role, ''),
(SELECT COUNT(*) FROM conversation_members cm2 WHERE cm2.conversation_id = c.id)
FROM conversations c
JOIN conversation_members mine ON mine.conversation_id = c.id AND mine.user_id = $1
WHERE c.id = $2`
	listMembersSQL = `SELECT cm.user_id, ` + personName + `, cm.role, ` + effectiveRole + `,
COALESCE(s.batch, ''), COALESCE(s.section, ''), COALESCE(t.designation, ''),
COALESCE(u.photo_path, ''), cm.added_at
FROM conversation_members cm
JOIN users u ON u.id = cm.user_id
LEFT JOIN students s ON s.user_id = u.id
LEFT JOIN teachers t ON t.user_id = u.id
WHERE cm.conversation_id = $1
ORDER BY cm.role, ` + personName + `, cm.user_id`
	memberRoleSQL    = `SELECT role FROM conversation_members WHERE conversation_id = $1 AND user_id = $2`
	countAdminsSQL   = `SELECT COUNT(*) FROM conversation_members WHERE conversation_id = $1 AND role = 'admin'`
	deleteMemberSQL  = `DELETE FROM conversation_members WHERE conversation_id = $1 AND user_id = $2 RETURNING user_id`
	setMemberRoleSQL = `UPDATE conversation_members SET role = $3
WHERE conversation_id = $1 AND user_id = $2
RETURNING user_id`
	earliestMemberSQL = `SELECT user_id FROM conversation_members
WHERE conversation_id = $1
ORDER BY added_at, user_id
LIMIT 1`
	chatMessageColumns = `m.id, COALESCE(m.sender_id, 0),
COALESCE(s.full_name, t.name, u.identifier, 'Deleted user'),
COALESCE(` + effectiveRole + `, ''),
COALESCE(u.photo_path, ''),
m.body, COALESCE(m.link_url, ''), COALESCE(m.file_path, ''), COALESCE(m.file_name, ''),
COALESCE(m.file_mime, ''), COALESCE(m.file_size, 0), m.created_at, m.edited_at, m.deleted_at`
	chatMessageJoins = `FROM messages m
LEFT JOIN users u ON u.id = m.sender_id
LEFT JOIN students s ON s.user_id = u.id
LEFT JOIN teachers t ON t.user_id = u.id`
	clearedAtSQL = `(SELECT cc.cleared_at FROM conversation_clears cc
    WHERE cc.conversation_id = $1 AND cc.user_id = $4)`
	listChatMessagesSQL = `SELECT ` + chatMessageColumns + `
` + chatMessageJoins + `
WHERE m.conversation_id = $1 AND ($2::timestamptz IS NULL OR m.created_at < $2::timestamptz)
  AND m.created_at > COALESCE(` + clearedAtSQL + `, '-infinity'::timestamptz)
ORDER BY m.created_at DESC, m.id DESC
LIMIT $3`
	clearConversationSQL = `INSERT INTO conversation_clears (user_id, conversation_id, cleared_at)
VALUES ($1, $2, $3)
ON CONFLICT (user_id, conversation_id) DO UPDATE SET cleared_at = EXCLUDED.cleared_at`
	getChatMessageSQL = `SELECT ` + chatMessageColumns + `
` + chatMessageJoins + `
WHERE m.id = $1`
	ownMessageSQL = `SELECT conversation_id, created_at, deleted_at IS NOT NULL,
COALESCE(link_url, '') <> '' OR COALESCE(file_path, '') <> '', COALESCE(file_path, '')
FROM messages
WHERE id = $1 AND sender_id = $2`
	editChatMessageSQL = `UPDATE messages SET body = $2, edited_at = $3
WHERE id = $1 AND sender_id = $4 AND deleted_at IS NULL AND created_at > $5::timestamptz
RETURNING id`
	deleteChatMessageSQL = `UPDATE messages
SET body = '', link_url = NULL, file_path = NULL, file_name = NULL, file_mime = NULL,
    file_size = NULL, deleted_at = $2
WHERE id = $1 AND sender_id = $3 AND deleted_at IS NULL AND created_at > $4::timestamptz
RETURNING conversation_id`
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
JOIN conversation_members cm ON cm.conversation_id = m.conversation_id AND cm.user_id = $1
LEFT JOIN conversation_reads r ON r.conversation_id = m.conversation_id AND r.user_id = $1
LEFT JOIN conversation_clears cc ON cc.conversation_id = m.conversation_id AND cc.user_id = $1
WHERE COALESCE(m.sender_id, 0) <> $1
  AND (r.last_read_at IS NULL OR m.created_at > r.last_read_at)
  AND m.created_at > COALESCE(cc.cleared_at, '-infinity'::timestamptz)`
	recipientsSQL = `SELECT cm.user_id
FROM conversation_members cm
WHERE cm.conversation_id = $1 AND cm.user_id <> $2
ORDER BY cm.user_id`
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

func (s *Store) SearchContacts(ctx context.Context, viewerID int64, viewerRole, search,
	role string) ([]ContactRow, error) {
	rows, err := s.pool.Query(ctx, searchContactsSQL, viewerID, search, role, ContactsLimit, viewerRole)
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

func (s *Store) ActiveUserIDs(ctx context.Context, userIDs []int64) ([]int64, error) {
	if len(userIDs) == 0 {
		return []int64{}, nil
	}
	return s.userIDs(ctx, activeUserIDsSQL, userIDs)
}

func (s *Store) BatchMembers(ctx context.Context, batch, section string) ([]int64, error) {
	return s.userIDs(ctx, batchMembersSQL, batch, section)
}

func (s *Store) ConversationRecipients(ctx context.Context, conversationID, senderID int64) ([]int64, error) {
	return s.userIDs(ctx, recipientsSQL, conversationID, senderID)
}

func (s *Store) GetOrCreateDirect(ctx context.Context, viewerID int64, viewerRole string,
	otherID int64) (Conversation, error) {
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
	if !validate.CanStartDirect(viewerRole, role) {
		return Conversation{}, ErrContactBlocked
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
		if _, err := tx.Exec(ctx, insertMemberSQL, conversation.ID, member, MemberRegular, viewerID); err != nil {
			return Conversation{}, fmt.Errorf("store: could not add the member %d: %w", member, err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return Conversation{}, fmt.Errorf("store: could not commit the direct conversation: %w", err)
	}
	return conversation, nil
}

func (s *Store) CreateGroup(ctx context.Context, group NewGroup) (Conversation, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Conversation{}, fmt.Errorf("store: could not start the group transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	conversation, err := scanConversation(tx.QueryRow(ctx, insertGroupSQL, group.Title, group.CreatedBy))
	if err != nil {
		return Conversation{}, fmt.Errorf("store: could not create the group conversation: %w", err)
	}
	if _, err := tx.Exec(ctx, insertMemberSQL, conversation.ID, group.CreatedBy, MemberAdmin, group.CreatedBy); err != nil {
		return Conversation{}, fmt.Errorf("store: could not add the group admin: %w", err)
	}
	for _, member := range group.Members {
		if member == group.CreatedBy {
			continue
		}
		_, err := tx.Exec(ctx, insertMemberSQL, conversation.ID, member, MemberRegular, group.CreatedBy)
		if err != nil {
			return Conversation{}, fmt.Errorf("store: could not add the member %d: %w", member, err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return Conversation{}, fmt.Errorf("store: could not commit the group conversation: %w", err)
	}
	return conversation, nil
}

func (s *Store) AddConversationMembers(ctx context.Context, conversationID, actorID int64,
	userIDs []int64) (int, error) {
	added := 0
	for _, userID := range userIDs {
		tag, err := s.pool.Exec(ctx, insertMemberSQL, conversationID, userID, MemberRegular, actorID)
		if err != nil {
			return added, fmt.Errorf("store: could not add the member %d: %w", userID, err)
		}
		added += int(tag.RowsAffected())
	}
	return added, nil
}

func (s *Store) ListConversationMembers(ctx context.Context, conversationID int64) ([]MemberRow, error) {
	rows, err := s.pool.Query(ctx, listMembersSQL, conversationID)
	if err != nil {
		return nil, fmt.Errorf("store: could not list the members of the conversation %d: %w", conversationID, err)
	}
	defer rows.Close()
	members := make([]MemberRow, 0)
	for rows.Next() {
		var member MemberRow
		err := rows.Scan(
			&member.UserID,
			&member.Name,
			&member.Role,
			&member.AccountRole,
			&member.Batch,
			&member.Section,
			&member.Designation,
			&member.PhotoPath,
			&member.AddedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("store: could not scan a member: %w", err)
		}
		members = append(members, member)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: could not read the members: %w", err)
	}
	return members, nil
}

func (s *Store) RemoveConversationMember(ctx context.Context, conversationID, userID int64) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("store: could not start the member transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var role string
	err = tx.QueryRow(ctx, memberRoleSQL, conversationID, userID).Scan(&role)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotAMember
	}
	if err != nil {
		return fmt.Errorf("store: could not load the member %d: %w", userID, err)
	}
	if role == MemberAdmin {
		var admins int
		if err := tx.QueryRow(ctx, countAdminsSQL, conversationID).Scan(&admins); err != nil {
			return fmt.Errorf("store: could not count the group admins: %w", err)
		}
		if admins <= 1 {
			return ErrLastAdmin
		}
	}
	var removed int64
	if err := tx.QueryRow(ctx, deleteMemberSQL, conversationID, userID).Scan(&removed); err != nil {
		return fmt.Errorf("store: could not remove the member %d: %w", userID, err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("store: could not commit the member removal: %w", err)
	}
	return nil
}

func (s *Store) SetConversationMemberAdmin(ctx context.Context, conversationID, userID int64, admin bool) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("store: could not start the member transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var role string
	err = tx.QueryRow(ctx, memberRoleSQL, conversationID, userID).Scan(&role)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotAMember
	}
	if err != nil {
		return fmt.Errorf("store: could not load the member %d: %w", userID, err)
	}
	if !admin && role == MemberAdmin {
		var admins int
		if err := tx.QueryRow(ctx, countAdminsSQL, conversationID).Scan(&admins); err != nil {
			return fmt.Errorf("store: could not count the group admins: %w", err)
		}
		if admins <= 1 {
			return ErrLastAdmin
		}
	}
	next := MemberRegular
	if admin {
		next = MemberAdmin
	}
	var changed int64
	if err := tx.QueryRow(ctx, setMemberRoleSQL, conversationID, userID, next).Scan(&changed); err != nil {
		return fmt.Errorf("store: could not change the role of the member %d: %w", userID, err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("store: could not commit the member role: %w", err)
	}
	return nil
}

func (s *Store) LeaveConversation(ctx context.Context, conversationID, userID int64) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("store: could not start the leave transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var role string
	err = tx.QueryRow(ctx, memberRoleSQL, conversationID, userID).Scan(&role)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotAMember
	}
	if err != nil {
		return fmt.Errorf("store: could not load the member %d: %w", userID, err)
	}
	var left int64
	if err := tx.QueryRow(ctx, deleteMemberSQL, conversationID, userID).Scan(&left); err != nil {
		return fmt.Errorf("store: could not remove the member %d: %w", userID, err)
	}
	var admins int
	if err := tx.QueryRow(ctx, countAdminsSQL, conversationID).Scan(&admins); err != nil {
		return fmt.Errorf("store: could not count the group admins: %w", err)
	}
	if admins == 0 {
		var next int64
		err := tx.QueryRow(ctx, earliestMemberSQL, conversationID).Scan(&next)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("store: could not find the next group admin: %w", err)
		}
		if err == nil {
			var promoted int64
			err := tx.QueryRow(ctx, setMemberRoleSQL, conversationID, next, MemberAdmin).Scan(&promoted)
			if err != nil {
				return fmt.Errorf("store: could not promote the next group admin: %w", err)
			}
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("store: could not commit leaving the group: %w", err)
	}
	return nil
}

func (s *Store) RenameConversation(ctx context.Context, conversationID int64, title string) error {
	var id int64
	err := s.pool.QueryRow(ctx, renameGroupSQL, conversationID, title).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotAGroup
	}
	if err != nil {
		return fmt.Errorf("store: could not rename the conversation %d: %w", conversationID, err)
	}
	return nil
}

func (s *Store) ListConversations(ctx context.Context, viewer MessageViewer, limit int) ([]ConversationRow, error) {
	if limit <= 0 || limit > MaxMessagePage {
		limit = MaxMessagePage
	}
	rows, err := s.pool.Query(ctx, listConversationsSQL, viewer.UserID, limit)
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
			&row.Title,
			&row.CreatedBy,
			&row.CreatedAt,
			&row.LastMessageAt,
			&row.OtherUserID,
			&row.OtherName,
			&row.OtherPhotoPath,
			&row.OtherRole,
			&row.LastSenderName,
			&row.LastBody,
			&row.LastFileName,
			&row.UnreadCount,
			&row.MemberRole,
			&row.MemberCount,
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

func (s *Store) FindConversationAccess(ctx context.Context, conversationID int64,
	viewer MessageViewer) (ConversationAccess, error) {
	var access ConversationAccess
	err := s.pool.QueryRow(ctx, findConversationAccessSQL, viewer.UserID, conversationID).Scan(
		&access.ID,
		&access.Kind,
		&access.Title,
		&access.CreatedBy,
		&access.CreatedAt,
		&access.LastMessageAt,
		&access.MemberRole,
		&access.MemberCount,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return ConversationAccess{}, ErrNotFound
	}
	if err != nil {
		return ConversationAccess{}, fmt.Errorf("store: could not load the conversation %d: %w", conversationID, err)
	}
	return access, nil
}

func (s *Store) ListChatMessages(ctx context.Context, conversationID int64, before *time.Time,
	limit int, viewerID int64) ([]ChatMessageRow, error) {
	if limit <= 0 || limit > MaxMessagePage {
		limit = MessagePageSize
	}
	rows, err := s.pool.Query(ctx, listChatMessagesSQL, conversationID, before, limit, viewerID)
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
			&row.EditedAt,
			&row.DeletedAt,
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

func (s *Store) FindOwnMessage(ctx context.Context, messageID, senderID int64) (OwnMessage, error) {
	var own OwnMessage
	err := s.pool.QueryRow(ctx, ownMessageSQL, messageID, senderID).Scan(
		&own.ConversationID,
		&own.CreatedAt,
		&own.Deleted,
		&own.HasAttachment,
		&own.FilePath,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return OwnMessage{}, ErrNotFound
	}
	if err != nil {
		return OwnMessage{}, fmt.Errorf("store: could not load the message %d: %w", messageID, err)
	}
	return own, nil
}

func (s *Store) EditChatMessage(ctx context.Context, messageID, senderID int64, body string,
	now time.Time) (ChatMessageRow, error) {
	var id int64
	err := s.pool.QueryRow(ctx, editChatMessageSQL, messageID, body, now, senderID, now.Add(-EditWindow)).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return ChatMessageRow{}, ErrEditWindowOver
	}
	if err != nil {
		return ChatMessageRow{}, fmt.Errorf("store: could not edit the message %d: %w", messageID, err)
	}
	return s.chatMessage(ctx, messageID)
}

func (s *Store) DeleteChatMessage(ctx context.Context, messageID, senderID int64,
	now time.Time) (int64, error) {
	var conversationID int64
	err := s.pool.QueryRow(ctx, deleteChatMessageSQL, messageID, now, senderID, now.Add(-EditWindow)).
		Scan(&conversationID)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrEditWindowOver
	}
	if err != nil {
		return 0, fmt.Errorf("store: could not delete the message %d: %w", messageID, err)
	}
	return conversationID, nil
}

func (s *Store) chatMessage(ctx context.Context, messageID int64) (ChatMessageRow, error) {
	var row ChatMessageRow
	err := s.pool.QueryRow(ctx, getChatMessageSQL, messageID).Scan(
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
		&row.EditedAt,
		&row.DeletedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return ChatMessageRow{}, ErrNotFound
	}
	if err != nil {
		return ChatMessageRow{}, fmt.Errorf("store: could not load the message %d: %w", messageID, err)
	}
	return row, nil
}

func (s *Store) ClearConversation(ctx context.Context, userID, conversationID int64, at time.Time) error {
	_, err := s.pool.Exec(ctx, clearConversationSQL, userID, conversationID, at)
	if err != nil {
		return fmt.Errorf("store: could not clear the conversation %d: %w", conversationID, err)
	}
	return nil
}

func (s *Store) MarkConversationRead(ctx context.Context, userID, conversationID int64, at time.Time) error {
	if _, err := s.pool.Exec(ctx, markReadSQL, userID, conversationID, at); err != nil {
		return fmt.Errorf("store: could not mark the conversation %d read: %w", conversationID, err)
	}
	return nil
}

func (s *Store) CountUnreadMessages(ctx context.Context, viewer MessageViewer) (int, error) {
	var count int
	if err := s.pool.QueryRow(ctx, unreadCountSQL, viewer.UserID).Scan(&count); err != nil {
		return 0, fmt.Errorf("store: could not count the unread messages: %w", err)
	}
	return count, nil
}

func (s *Store) userIDs(ctx context.Context, query string, args ...any) ([]int64, error) {
	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("store: could not list the accounts: %w", err)
	}
	defer rows.Close()
	ids := make([]int64, 0)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("store: could not scan an account id: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: could not read the accounts: %w", err)
	}
	return ids, nil
}

func scanConversation(row pgx.Row) (Conversation, error) {
	var conversation Conversation
	err := row.Scan(
		&conversation.ID,
		&conversation.Kind,
		&conversation.Title,
		&conversation.CreatedBy,
		&conversation.CreatedAt,
		&conversation.LastMessageAt,
	)
	return conversation, err
}

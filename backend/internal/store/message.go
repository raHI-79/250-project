package store

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"time"
)

const (
	TargetUser      = "user"
	TargetBatch     = "batch"
	TargetSection   = "section"
	KindTeacher     = "teacher"
	KindCR          = "cr"
	KindStudent     = "student"
	StudentsPerHint = 20
)

type Sender struct {
	UserID  int64
	Role    string
	Name    string
	Batch   string
	Section string
	IsCR    bool
}

type MessageTarget struct {
	UserID  int64
	Name    string
	Batch   string
	Section string
	IsCR    bool
}

type NewMessage struct {
	SenderID      int64
	SenderRole    string
	SenderName    string
	TargetType    string
	TargetBatch   string
	TargetSection string
	TargetUserID  int64
	Content       string
	FilePath      string
	ReplyToID     int64
}

type MessageRow struct {
	ID            int64
	SenderID      int64
	SenderRole    string
	SenderName    string
	SenderBatch   string
	SenderSection string
	TargetType    string
	TargetBatch   string
	TargetSection string
	TargetUserID  int64
	Content       string
	FilePath      string
	ReplyToID     int64
	CreatedAt     time.Time
}

type CROption struct {
	UserID   int64
	FullName string
	Batch    string
	Section  string
}

type StudentOption struct {
	UserID   int64
	FullName string
	RegNo    string
	Batch    string
	Section  string
}

const (
	messageColumns = `m.id, m.sender_id, m.sender_role, m.sender_name, COALESCE(ss.batch, ''), COALESCE(ss.section, ''),
m.target_type, COALESCE(m.target_batch, ''), COALESCE(m.target_section, ''), COALESCE(m.target_user_id, 0),
m.content, COALESCE(m.file_path, ''), COALESCE(m.reply_to_id, 0), m.created_at`
	messageJoins = `FROM messages m
LEFT JOIN students ss ON ss.user_id = m.sender_id`
	insertMessageSQL = `WITH inserted AS (
    INSERT INTO messages (sender_id, sender_role, sender_name, target_type, target_batch, target_section,
        target_user_id, content, file_path, reply_to_id)
    VALUES ($1, $2, $3, $4, NULLIF($5::text, ''), NULLIF($6::text, ''), NULLIF($7::bigint, 0), $8,
        NULLIF($9::text, ''), NULLIF($10::bigint, 0))
    RETURNING id, sender_id, sender_role, sender_name, target_type, target_batch, target_section,
        target_user_id, content, file_path, reply_to_id, created_at
)
SELECT ` + messageColumns + `
FROM inserted m
LEFT JOIN students ss ON ss.user_id = m.sender_id`
	listCRInboxSQL = `SELECT ` + messageColumns + `
` + messageJoins + `
WHERE m.target_user_id = $1 AND m.sender_role = 'teacher'
ORDER BY m.created_at DESC, m.id DESC`
	listTeacherInboxSQL = `SELECT ` + messageColumns + `
` + messageJoins + `
WHERE m.target_user_id = $1 AND m.sender_role = 'cr'
ORDER BY m.created_at DESC, m.id DESC`
	listSentMessagesSQL = `SELECT ` + messageColumns + `
` + messageJoins + `
WHERE m.sender_id = $1
ORDER BY m.created_at DESC, m.id DESC`
	listStudentMessagesSQL = `SELECT ` + messageColumns + `
` + messageJoins + `
WHERE m.sender_role IN ('cr', 'teacher')
  AND ((m.target_type = 'user' AND m.target_user_id = $1)
    OR (m.target_type = 'batch' AND m.target_batch = $2::text)
    OR (m.target_type = 'section' AND m.target_batch = $2::text AND m.target_section = $3::text))
ORDER BY m.created_at DESC, m.id DESC`
	messageSenderSQL = `SELECT u.id,
CASE WHEN u.role = 'student' AND COALESCE(s.is_cr, FALSE) THEN 'cr' ELSE u.role END,
COALESCE(s.full_name, t.name, 'Administration'), COALESCE(s.batch, ''), COALESCE(s.section, ''),
COALESCE(s.is_cr, FALSE)
FROM users u
LEFT JOIN students s ON s.user_id = u.id
LEFT JOIN teachers t ON t.user_id = u.id
WHERE u.id = $1 AND u.status = 'active'`
	messageTargetSQL = `SELECT u.id, COALESCE(s.full_name, t.name, ''), COALESCE(s.batch, ''), COALESCE(s.section, ''),
COALESCE(s.is_cr, FALSE)
FROM users u
LEFT JOIN students s ON s.user_id = u.id
LEFT JOIN teachers t ON t.user_id = u.id
WHERE u.id = $1 AND u.status = 'active'
  AND (($2::text = 'teacher' AND u.role = 'teacher')
    OR ($2::text = 'cr' AND u.role = 'student' AND COALESCE(s.is_cr, FALSE))
    OR ($2::text = 'student' AND u.role = 'student'))`
	replyTargetSQL = `SELECT m.sender_id, m.sender_name, COALESCE(ss.batch, ''), COALESCE(ss.section, ''), TRUE
FROM messages m
LEFT JOIN students ss ON ss.user_id = m.sender_id
WHERE m.id = $1 AND m.target_user_id = $2 AND m.sender_role = 'cr'`
	listCROptionsSQL = `SELECT u.id, s.full_name, s.batch, s.section
FROM users u
JOIN students s ON s.user_id = u.id
WHERE u.status = 'active' AND s.is_cr
  AND ($1::text = '' OR s.batch = $1::text)
ORDER BY s.batch, s.section, s.full_name, u.id`
	searchStudentsSQL = `SELECT u.id, s.full_name, s.reg_no, s.batch, s.section
FROM users u
JOIN students s ON s.user_id = u.id
WHERE u.status = 'active'
  AND ($1::text = '' OR s.full_name ILIKE '%' || $1::text || '%' OR s.reg_no ILIKE '%' || $1::text || '%')
ORDER BY s.batch, s.section, s.reg_no
LIMIT $2`
)

func (s *Store) CreateMessage(ctx context.Context, message NewMessage) (MessageRow, error) {
	var row MessageRow
	err := s.pool.QueryRow(ctx, insertMessageSQL,
		message.SenderID,
		message.SenderRole,
		message.SenderName,
		message.TargetType,
		message.TargetBatch,
		message.TargetSection,
		message.TargetUserID,
		message.Content,
		message.FilePath,
		message.ReplyToID,
	).Scan(
		&row.ID,
		&row.SenderID,
		&row.SenderRole,
		&row.SenderName,
		&row.SenderBatch,
		&row.SenderSection,
		&row.TargetType,
		&row.TargetBatch,
		&row.TargetSection,
		&row.TargetUserID,
		&row.Content,
		&row.FilePath,
		&row.ReplyToID,
		&row.CreatedAt,
	)
	if err != nil {
		return MessageRow{}, fmt.Errorf("store: could not create the message: %w", err)
	}
	return row, nil
}

func (s *Store) GetMessageSender(ctx context.Context, userID int64) (Sender, error) {
	var sender Sender
	err := s.pool.QueryRow(ctx, messageSenderSQL, userID).Scan(
		&sender.UserID,
		&sender.Role,
		&sender.Name,
		&sender.Batch,
		&sender.Section,
		&sender.IsCR,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Sender{}, ErrNotFound
	}
	if err != nil {
		return Sender{}, fmt.Errorf("store: could not load the sender %d: %w", userID, err)
	}
	return sender, nil
}

func (s *Store) FindMessageTarget(ctx context.Context, userID int64, kind string) (MessageTarget, error) {
	return s.target(ctx, messageTargetSQL, userID, kind)
}

func (s *Store) FindReplyTarget(ctx context.Context, messageID, teacherUserID int64) (MessageTarget, error) {
	return s.target(ctx, replyTargetSQL, messageID, teacherUserID)
}

func (s *Store) ListCRInbox(ctx context.Context, crUserID int64) ([]MessageRow, error) {
	return s.messages(ctx, listCRInboxSQL, crUserID)
}

func (s *Store) ListTeacherInbox(ctx context.Context, teacherUserID int64) ([]MessageRow, error) {
	return s.messages(ctx, listTeacherInboxSQL, teacherUserID)
}

func (s *Store) ListSentMessages(ctx context.Context, senderID int64) ([]MessageRow, error) {
	return s.messages(ctx, listSentMessagesSQL, senderID)
}

func (s *Store) ListStudentMessages(ctx context.Context, userID int64, batch, section string) ([]MessageRow, error) {
	return s.messages(ctx, listStudentMessagesSQL, userID, batch, section)
}

func (s *Store) ListCROptions(ctx context.Context, batch string) ([]CROption, error) {
	rows, err := s.pool.Query(ctx, listCROptionsSQL, batch)
	if err != nil {
		return nil, fmt.Errorf("store: could not list the class representatives: %w", err)
	}
	defer rows.Close()
	options := make([]CROption, 0)
	for rows.Next() {
		var option CROption
		if err := rows.Scan(&option.UserID, &option.FullName, &option.Batch, &option.Section); err != nil {
			return nil, fmt.Errorf("store: could not scan a class representative: %w", err)
		}
		options = append(options, option)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: could not read the class representatives: %w", err)
	}
	return options, nil
}

func (s *Store) SearchStudents(ctx context.Context, search string) ([]StudentOption, error) {
	rows, err := s.pool.Query(ctx, searchStudentsSQL, search, StudentsPerHint)
	if err != nil {
		return nil, fmt.Errorf("store: could not search the students: %w", err)
	}
	defer rows.Close()
	options := make([]StudentOption, 0)
	for rows.Next() {
		var option StudentOption
		err := rows.Scan(&option.UserID, &option.FullName, &option.RegNo, &option.Batch, &option.Section)
		if err != nil {
			return nil, fmt.Errorf("store: could not scan a student: %w", err)
		}
		options = append(options, option)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: could not read the students: %w", err)
	}
	return options, nil
}

func (s *Store) target(ctx context.Context, query string, args ...any) (MessageTarget, error) {
	var target MessageTarget
	err := s.pool.QueryRow(ctx, query, args...).Scan(
		&target.UserID,
		&target.Name,
		&target.Batch,
		&target.Section,
		&target.IsCR,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return MessageTarget{}, ErrNotFound
	}
	if err != nil {
		return MessageTarget{}, fmt.Errorf("store: could not load the message target: %w", err)
	}
	return target, nil
}

func (s *Store) messages(ctx context.Context, query string, args ...any) ([]MessageRow, error) {
	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("store: could not list the messages: %w", err)
	}
	defer rows.Close()
	messages := make([]MessageRow, 0)
	for rows.Next() {
		var row MessageRow
		err := rows.Scan(
			&row.ID,
			&row.SenderID,
			&row.SenderRole,
			&row.SenderName,
			&row.SenderBatch,
			&row.SenderSection,
			&row.TargetType,
			&row.TargetBatch,
			&row.TargetSection,
			&row.TargetUserID,
			&row.Content,
			&row.FilePath,
			&row.ReplyToID,
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

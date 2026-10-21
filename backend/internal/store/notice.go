package store

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"time"
)

const (
	SourceAdmin   = "admin"
	SourceCR      = "cr"
	SourceTeacher = "teacher"
	NoticeAll     = "all"
	NoticeMine    = "my"
	NoticeOthers  = "other"
)

type NewNotice struct {
	Source      string
	Title       string
	Description string
	Batch       string
	Section     string
	LinkURL     string
	FilePath    string
	FileName    string
	FileMime    string
	FileSize    int64
	CreatedBy   int64
}

type NoticeRow struct {
	ID             int64
	Source         string
	Title          string
	Description    string
	Batch          string
	Section        string
	LinkURL        string
	FilePath       string
	FileName       string
	FileMime       string
	FileSize       int64
	CreatedBy      int64
	CreatedByName  string
	CreatedByPhoto string
	CreatedAt      time.Time
}

const (
	noticeColumns = `n.id, n.source, n.title, n.description, COALESCE(n.batch, ''), COALESCE(n.section, ''),
COALESCE(n.link_url, ''), COALESCE(n.file_path, ''), COALESCE(n.file_name, ''), COALESCE(n.file_mime, ''),
COALESCE(n.file_size, 0), COALESCE(n.created_by, 0),
COALESCE(s.full_name, t.name, CASE WHEN n.created_by IS NULL THEN 'Deleted user' ELSE 'Administration' END),
COALESCE(u.photo_path, ''), n.created_at`
	noticeJoins = `FROM notices n
LEFT JOIN users u ON u.id = n.created_by
LEFT JOIN students s ON s.user_id = u.id
LEFT JOIN teachers t ON t.user_id = u.id`
	insertNoticeSQL = `WITH inserted AS (
    INSERT INTO notices
    (source, title, description, batch, section, link_url, file_path, file_name, file_mime, file_size, created_by)
    VALUES ($1, $2, $3, NULLIF($4::text, ''), NULLIF($5::text, ''), NULLIF($6::text, ''),
    NULLIF($7::text, ''), NULLIF($8::text, ''), NULLIF($9::text, ''), NULLIF($10::bigint, 0), $11)
    RETURNING id, source, title, description, batch, section, link_url, file_path, file_name,
    file_mime, file_size, created_by, created_at
)
SELECT ` + noticeColumns + `
FROM inserted n
LEFT JOIN users u ON u.id = n.created_by
LEFT JOIN students s ON s.user_id = u.id
LEFT JOIN teachers t ON t.user_id = u.id`
	listAdminNoticesSQL = `SELECT ` + noticeColumns + `
` + noticeJoins + `
WHERE n.source = 'admin'
ORDER BY n.created_at DESC, n.id DESC`
	listStudentAdminNoticesSQL = `SELECT ` + noticeColumns + `
` + noticeJoins + `
WHERE n.source = 'admin' AND (n.batch IS NULL OR n.batch = $1::text)
ORDER BY n.created_at DESC, n.id DESC`
	listCRNoticesSQL = `SELECT ` + noticeColumns + `
` + noticeJoins + `
WHERE n.source = 'cr' AND n.created_by = $1
ORDER BY n.created_at DESC, n.id DESC`
	listBatchCRNoticesSQL = `SELECT ` + noticeColumns + `
` + noticeJoins + `
WHERE n.source = 'cr' AND (($2::bool AND n.batch <> $1::text) OR (NOT $2::bool AND n.batch = $1::text))
ORDER BY n.created_at DESC, n.id DESC`
	deleteAdminNoticeSQL = `DELETE FROM notices WHERE id = $1 AND source = 'admin'
RETURNING COALESCE(file_path, '')`
	listTeacherNoticesSQL = `SELECT ` + noticeColumns + `
` + noticeJoins + `
WHERE n.source = 'teacher' AND n.created_by = $1
ORDER BY n.created_at DESC, n.id DESC`
	deleteOwnNoticeSQL = `DELETE FROM notices WHERE id = $1 AND source = $2 AND created_by = $3
RETURNING COALESCE(file_path, '')`
	listStudentNoticesSQL = `SELECT ` + noticeColumns + `
` + noticeJoins + `
WHERE (
    ($3::text IN ('all', 'admin') AND n.source = 'admin'
        AND (n.batch IS NULL OR n.batch = $1::text))
    OR ($3::text IN ('all', 'teacher') AND n.source = 'teacher'
        AND n.batch = $1::text AND (n.section IS NULL OR n.section = $2::text))
    OR ($3::text IN ('all', 'cr') AND n.source = 'cr'
        AND (($4::text = 'other' AND n.batch <> $1::text)
             OR ($4::text <> 'other' AND n.batch = $1::text)))
)
ORDER BY n.created_at DESC, n.id DESC`
)

func (s *Store) CreateNotice(ctx context.Context, notice NewNotice) (NoticeRow, error) {
	var row NoticeRow
	err := s.pool.QueryRow(ctx, insertNoticeSQL,
		notice.Source,
		notice.Title,
		notice.Description,
		notice.Batch,
		notice.Section,
		notice.LinkURL,
		notice.FilePath,
		notice.FileName,
		notice.FileMime,
		notice.FileSize,
		notice.CreatedBy,
	).Scan(
		&row.ID,
		&row.Source,
		&row.Title,
		&row.Description,
		&row.Batch,
		&row.Section,
		&row.LinkURL,
		&row.FilePath,
		&row.FileName,
		&row.FileMime,
		&row.FileSize,
		&row.CreatedBy,
		&row.CreatedByName,
		&row.CreatedByPhoto,
		&row.CreatedAt,
	)
	if err != nil {
		return NoticeRow{}, fmt.Errorf("store: could not create the notice: %w", err)
	}
	return row, nil
}

func (s *Store) ListAdminNotices(ctx context.Context) ([]NoticeRow, error) {
	return s.notices(ctx, listAdminNoticesSQL)
}

func (s *Store) ListStudentAdminNotices(ctx context.Context, batch string) ([]NoticeRow, error) {
	return s.notices(ctx, listStudentAdminNoticesSQL, batch)
}

func (s *Store) ListCRNotices(ctx context.Context, createdBy int64) ([]NoticeRow, error) {
	return s.notices(ctx, listCRNoticesSQL, createdBy)
}

func (s *Store) ListBatchCRNotices(ctx context.Context, batch string, otherBatches bool) ([]NoticeRow, error) {
	return s.notices(ctx, listBatchCRNoticesSQL, batch, otherBatches)
}

func (s *Store) ListTeacherNotices(ctx context.Context, createdBy int64) ([]NoticeRow, error) {
	return s.notices(ctx, listTeacherNoticesSQL, createdBy)
}

func (s *Store) ListStudentNotices(ctx context.Context, batch, section, source, scope string) ([]NoticeRow, error) {
	return s.notices(ctx, listStudentNoticesSQL, batch, section, source, scope)
}

func (s *Store) DeleteOwnNotice(ctx context.Context, noticeID int64, source string, createdBy int64) (string, error) {
	var filePath string
	err := s.pool.QueryRow(ctx, deleteOwnNoticeSQL, noticeID, source, createdBy).Scan(&filePath)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("store: could not delete the notice %d: %w", noticeID, err)
	}
	return filePath, nil
}

func (s *Store) DeleteAdminNotice(ctx context.Context, noticeID int64) (string, error) {
	var filePath string
	err := s.pool.QueryRow(ctx, deleteAdminNoticeSQL, noticeID).Scan(&filePath)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("store: could not delete the notice %d: %w", noticeID, err)
	}
	return filePath, nil
}

func (s *Store) notices(ctx context.Context, query string, args ...any) ([]NoticeRow, error) {
	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("store: could not list the notices: %w", err)
	}
	defer rows.Close()
	notices := make([]NoticeRow, 0)
	for rows.Next() {
		var row NoticeRow
		err := rows.Scan(
			&row.ID,
			&row.Source,
			&row.Title,
			&row.Description,
			&row.Batch,
			&row.Section,
			&row.LinkURL,
			&row.FilePath,
			&row.FileName,
			&row.FileMime,
			&row.FileSize,
			&row.CreatedBy,
			&row.CreatedByName,
			&row.CreatedByPhoto,
			&row.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("store: could not scan a notice: %w", err)
		}
		notices = append(notices, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: could not read the notices: %w", err)
	}
	return notices, nil
}

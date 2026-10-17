package store

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"time"
)

const (
	SourceAdmin = "admin"
	SourceCR    = "cr"
)

type NewNotice struct {
	Source      string
	Title       string
	Description string
	Batch       string
	Section     string
	FilePath    string
	CreatedBy   int64
}

type NoticeRow struct {
	ID            int64
	Source        string
	Title         string
	Description   string
	Batch         string
	Section       string
	FilePath      string
	CreatedByName string
	CreatedAt     time.Time
}

const (
	noticeColumns = `n.id, n.source, n.title, n.description, COALESCE(n.batch, ''), COALESCE(n.section, ''),
COALESCE(n.file_path, ''), COALESCE(s.full_name, t.name, 'Administration'), n.created_at`
	noticeJoins = `FROM notices n
JOIN users u ON u.id = n.created_by
LEFT JOIN students s ON s.user_id = u.id
LEFT JOIN teachers t ON t.user_id = u.id`
	insertNoticeSQL = `WITH inserted AS (
    INSERT INTO notices (source, title, description, batch, section, file_path, created_by)
    VALUES ($1, $2, $3, NULLIF($4::text, ''), NULLIF($5::text, ''), NULLIF($6::text, ''), $7)
    RETURNING id, source, title, description, batch, section, file_path, created_by, created_at
)
SELECT n.id, n.source, n.title, n.description, COALESCE(n.batch, ''), COALESCE(n.section, ''),
COALESCE(n.file_path, ''), COALESCE(s.full_name, t.name, 'Administration'), n.created_at
FROM inserted n
JOIN users u ON u.id = n.created_by
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
)

func (s *Store) CreateNotice(ctx context.Context, notice NewNotice) (NoticeRow, error) {
	var row NoticeRow
	err := s.pool.QueryRow(ctx, insertNoticeSQL,
		notice.Source,
		notice.Title,
		notice.Description,
		notice.Batch,
		notice.Section,
		notice.FilePath,
		notice.CreatedBy,
	).Scan(
		&row.ID,
		&row.Source,
		&row.Title,
		&row.Description,
		&row.Batch,
		&row.Section,
		&row.FilePath,
		&row.CreatedByName,
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
			&row.FilePath,
			&row.CreatedByName,
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

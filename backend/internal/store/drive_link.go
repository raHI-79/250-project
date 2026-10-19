package store

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"time"
)

type NewDriveLink struct {
	Batch     string
	Title     string
	Link      string
	UpdatedBy int64
}

type DriveLinkRow struct {
	Batch     string
	Title     string
	Link      string
	UpdatedAt time.Time
}

const (
	driveLinkColumns  = `batch, title, link, updated_at`
	getDriveLinkSQL   = `SELECT ` + driveLinkColumns + ` FROM batch_drive_links WHERE batch = $1`
	listDriveLinksSQL = `SELECT ` + driveLinkColumns + ` FROM batch_drive_links ORDER BY batch`
	saveDriveLinkSQL  = `INSERT INTO batch_drive_links (batch, title, link, updated_by, updated_at)
VALUES ($1, $2, $3, $4, now())
ON CONFLICT (batch) DO UPDATE
SET title = EXCLUDED.title, link = EXCLUDED.link, updated_by = EXCLUDED.updated_by, updated_at = now()
RETURNING ` + driveLinkColumns
	deleteDriveLinkSQL = `DELETE FROM batch_drive_links WHERE batch = $1 RETURNING batch`
)

func (s *Store) GetBatchDriveLink(ctx context.Context, batch string) (DriveLinkRow, error) {
	var row DriveLinkRow
	err := s.pool.QueryRow(ctx, getDriveLinkSQL, batch).Scan(&row.Batch, &row.Title, &row.Link, &row.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return DriveLinkRow{}, ErrNotFound
	}
	if err != nil {
		return DriveLinkRow{}, fmt.Errorf("store: could not load the drive link of %s: %w", batch, err)
	}
	return row, nil
}

func (s *Store) SaveBatchDriveLink(ctx context.Context, link NewDriveLink) (DriveLinkRow, error) {
	var row DriveLinkRow
	err := s.pool.QueryRow(ctx, saveDriveLinkSQL, link.Batch, link.Title, link.Link, link.UpdatedBy).
		Scan(&row.Batch, &row.Title, &row.Link, &row.UpdatedAt)
	if err != nil {
		return DriveLinkRow{}, fmt.Errorf("store: could not save the drive link of %s: %w", link.Batch, err)
	}
	return row, nil
}

func (s *Store) DeleteBatchDriveLink(ctx context.Context, batch string) error {
	var removed string
	err := s.pool.QueryRow(ctx, deleteDriveLinkSQL, batch).Scan(&removed)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("store: could not remove the drive link of %s: %w", batch, err)
	}
	return nil
}

func (s *Store) ListBatchDriveLinks(ctx context.Context) ([]DriveLinkRow, error) {
	rows, err := s.pool.Query(ctx, listDriveLinksSQL)
	if err != nil {
		return nil, fmt.Errorf("store: could not list the drive links: %w", err)
	}
	defer rows.Close()
	links := make([]DriveLinkRow, 0)
	for rows.Next() {
		var row DriveLinkRow
		if err := rows.Scan(&row.Batch, &row.Title, &row.Link, &row.UpdatedAt); err != nil {
			return nil, fmt.Errorf("store: could not scan a drive link: %w", err)
		}
		links = append(links, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: could not read the drive links: %w", err)
	}
	return links, nil
}

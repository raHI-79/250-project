package store

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"time"
)

const (
	SiteDraft     = "draft"
	SitePublished = "published"

	ActionSitePublish = "site_publish"
	ActionSiteRestore = "site_restore"
)

type SiteVersion struct {
	ID        int64
	Kind      string
	Content   []byte
	SavedBy   int64
	SavedName string
	SavedAt   time.Time
	Note      string
}

const (
	siteVersionColumns = `v.id, v.kind, v.content, COALESCE(v.saved_by, 0),
COALESCE(u.identifier, ''), v.saved_at, v.note`
	saveSiteDraftSQL = `INSERT INTO site_content_versions (kind, content, saved_by, note)
VALUES ('draft', $1::jsonb, NULLIF($2::bigint, 0), '')
ON CONFLICT (kind) WHERE kind = 'draft' DO UPDATE
SET content = EXCLUDED.content, saved_by = EXCLUDED.saved_by, saved_at = now()
RETURNING id`
	siteDraftSQL = `SELECT ` + siteVersionColumns + `
FROM site_content_versions v
LEFT JOIN users u ON u.id = v.saved_by
WHERE v.kind = 'draft'`
	sitePublishedSQL = `SELECT ` + siteVersionColumns + `
FROM site_content_versions v
LEFT JOIN users u ON u.id = v.saved_by
WHERE v.kind = 'published'
ORDER BY v.id DESC
LIMIT 1`
	siteVersionSQL = `SELECT ` + siteVersionColumns + `
FROM site_content_versions v
LEFT JOIN users u ON u.id = v.saved_by
WHERE v.id = $1 AND v.kind = 'published'`
	siteVersionsSQL = `SELECT v.id, v.kind, '{}'::jsonb, COALESCE(v.saved_by, 0),
COALESCE(u.identifier, ''), v.saved_at, v.note
FROM site_content_versions v
LEFT JOIN users u ON u.id = v.saved_by
WHERE v.kind = 'published'
ORDER BY v.id DESC
LIMIT $1`
	publishSiteSQL = `INSERT INTO site_content_versions (kind, content, saved_by, note)
VALUES ('published', $1::jsonb, NULLIF($2::bigint, 0), $3)
RETURNING id`
)

func scanSiteVersion(row rowScanner) (SiteVersion, error) {
	var version SiteVersion
	err := row.Scan(&version.ID, &version.Kind, &version.Content, &version.SavedBy,
		&version.SavedName, &version.SavedAt, &version.Note)
	return version, err
}

func (s *Store) SaveSiteDraft(ctx context.Context, content []byte, userID int64) error {
	var id int64
	if err := s.pool.QueryRow(ctx, saveSiteDraftSQL, string(content), userID).Scan(&id); err != nil {
		return fmt.Errorf("store: could not save the website draft: %w", err)
	}
	return nil
}

func (s *Store) siteVersion(ctx context.Context, query string, args ...any) (SiteVersion, error) {
	version, err := scanSiteVersion(s.pool.QueryRow(ctx, query, args...))
	if errors.Is(err, pgx.ErrNoRows) {
		return SiteVersion{}, ErrNotFound
	}
	if err != nil {
		return SiteVersion{}, fmt.Errorf("store: could not read the website content: %w", err)
	}
	return version, nil
}

func (s *Store) SiteDraft(ctx context.Context) (SiteVersion, error) {
	return s.siteVersion(ctx, siteDraftSQL)
}

func (s *Store) SitePublished(ctx context.Context) (SiteVersion, error) {
	return s.siteVersion(ctx, sitePublishedSQL)
}

func (s *Store) SiteVersion(ctx context.Context, id int64) (SiteVersion, error) {
	return s.siteVersion(ctx, siteVersionSQL, id)
}

func (s *Store) PublishSite(ctx context.Context, content []byte, userID int64, note string) (int64, error) {
	var id int64
	if err := s.pool.QueryRow(ctx, publishSiteSQL, string(content), userID, note).Scan(&id); err != nil {
		return 0, fmt.Errorf("store: could not publish the website: %w", err)
	}
	return id, nil
}

func (s *Store) ListSiteVersions(ctx context.Context, limit int) ([]SiteVersion, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.pool.Query(ctx, siteVersionsSQL, limit)
	if err != nil {
		return nil, fmt.Errorf("store: could not list the website versions: %w", err)
	}
	defer rows.Close()
	versions := make([]SiteVersion, 0)
	for rows.Next() {
		version, err := scanSiteVersion(rows)
		if err != nil {
			return nil, fmt.Errorf("store: could not scan a website version: %w", err)
		}
		versions = append(versions, version)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: could not read the website versions: %w", err)
	}
	return versions, nil
}

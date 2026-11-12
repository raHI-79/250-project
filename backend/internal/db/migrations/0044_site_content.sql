CREATE TABLE IF NOT EXISTS site_content_versions (
    id BIGSERIAL PRIMARY KEY,
    kind TEXT NOT NULL CHECK (kind IN ('draft', 'published')),
    content JSONB NOT NULL,
    saved_by BIGINT REFERENCES users (id) ON DELETE SET NULL,
    saved_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    note TEXT NOT NULL DEFAULT ''
);

CREATE UNIQUE INDEX IF NOT EXISTS site_content_versions_one_draft
    ON site_content_versions (kind) WHERE kind = 'draft';

CREATE INDEX IF NOT EXISTS site_content_versions_published_idx
    ON site_content_versions (id DESC) WHERE kind = 'published';

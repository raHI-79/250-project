CREATE TABLE IF NOT EXISTS batch_drive_links (
    id BIGSERIAL PRIMARY KEY,
    batch TEXT NOT NULL UNIQUE,
    title TEXT NOT NULL,
    link TEXT NOT NULL,
    updated_by BIGINT NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT batch_drive_links_batch_not_blank CHECK (batch <> ''),
    CONSTRAINT batch_drive_links_title_not_blank CHECK (title <> ''),
    CONSTRAINT batch_drive_links_link_not_blank CHECK (link <> '')
);

INSERT INTO batch_drive_links (batch, title, link, updated_by, updated_at)
SELECT DISTINCT ON (m.batch) m.batch, m.batch, m.link, m.created_by, m.created_at
FROM materials m
WHERE m.batch <> 'ALL'
ORDER BY m.batch, m.created_at DESC, m.id DESC
ON CONFLICT (batch) DO NOTHING;

DROP TABLE IF EXISTS materials;

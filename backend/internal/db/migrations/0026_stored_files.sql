CREATE TABLE IF NOT EXISTS stored_files (
    name TEXT PRIMARY KEY,
    kind TEXT NOT NULL,
    mime TEXT NOT NULL DEFAULT '',
    size BIGINT NOT NULL,
    data BYTEA NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT stored_files_name_not_blank CHECK (name <> ''),
    CONSTRAINT stored_files_kind_check CHECK (kind IN ('file', 'photo', 'attachment')),
    CONSTRAINT stored_files_size_not_negative CHECK (size >= 0)
);

CREATE INDEX IF NOT EXISTS stored_files_kind_idx ON stored_files (kind, created_at DESC);

ALTER TABLE messages ADD COLUMN IF NOT EXISTS edited_at TIMESTAMPTZ;

ALTER TABLE messages ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;

ALTER TABLE messages DROP CONSTRAINT IF EXISTS messages_has_content;

ALTER TABLE messages ADD CONSTRAINT messages_has_content CHECK (
    deleted_at IS NOT NULL
    OR body <> ''
    OR link_url IS NOT NULL
    OR file_path IS NOT NULL
);

ALTER TABLE messages DROP CONSTRAINT IF EXISTS messages_deleted_is_empty;

ALTER TABLE messages ADD CONSTRAINT messages_deleted_is_empty CHECK (
    deleted_at IS NULL
    OR (body = '' AND link_url IS NULL AND file_path IS NULL
        AND file_name IS NULL AND file_mime IS NULL AND file_size IS NULL)
);

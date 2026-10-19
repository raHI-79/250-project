ALTER TABLE notices DROP CONSTRAINT IF EXISTS notices_source_check;

ALTER TABLE notices ADD CONSTRAINT notices_source_check CHECK (source IN ('admin', 'cr', 'teacher'));

ALTER TABLE notices ADD COLUMN IF NOT EXISTS link_url TEXT;

ALTER TABLE notices ADD COLUMN IF NOT EXISTS file_name TEXT;

ALTER TABLE notices ADD COLUMN IF NOT EXISTS file_mime TEXT;

ALTER TABLE notices ADD COLUMN IF NOT EXISTS file_size BIGINT;

ALTER TABLE notices DROP CONSTRAINT IF EXISTS notices_cr_needs_batch;

ALTER TABLE notices ADD CONSTRAINT notices_batch_required CHECK (source = 'admin' OR batch IS NOT NULL);

ALTER TABLE notices ALTER COLUMN description SET DEFAULT '';

CREATE INDEX IF NOT EXISTS notices_source_batch_idx ON notices (source, batch, section, created_at DESC);

ALTER TABLE polls ADD COLUMN IF NOT EXISTS scope TEXT NOT NULL DEFAULT 'section';

UPDATE polls SET scope = CASE WHEN section = 'ALL' THEN 'batch' ELSE 'section' END;

ALTER TABLE polls ALTER COLUMN scope DROP DEFAULT;

ALTER TABLE polls DROP CONSTRAINT IF EXISTS polls_scope_check;

ALTER TABLE polls ADD CONSTRAINT polls_scope_check
CHECK (scope IN ('section', 'batch', 'all_batches'));

ALTER TABLE polls DROP CONSTRAINT IF EXISTS polls_scope_targets;

ALTER TABLE polls ADD CONSTRAINT polls_scope_targets CHECK (
    (scope = 'section' AND batch <> 'ALL' AND section <> 'ALL')
    OR (scope = 'batch' AND batch <> 'ALL' AND section = 'ALL')
    OR (scope = 'all_batches' AND batch = 'ALL' AND section = 'ALL')
);

CREATE INDEX IF NOT EXISTS polls_scope_idx ON polls (scope, batch, section, created_at DESC);

ALTER TABLE class_events ADD COLUMN IF NOT EXISTS note TEXT;

ALTER TABLE class_events ADD COLUMN IF NOT EXISTS note_audience TEXT;

ALTER TABLE class_events DROP CONSTRAINT IF EXISTS class_events_note_length;

ALTER TABLE class_events ADD CONSTRAINT class_events_note_length CHECK (
    note IS NULL OR char_length(note) <= 500
);

ALTER TABLE class_events DROP CONSTRAINT IF EXISTS class_events_note_audience;

ALTER TABLE class_events ADD CONSTRAINT class_events_note_audience CHECK (
    (note IS NULL AND note_audience IS NULL)
    OR (note IS NOT NULL AND note_audience IS NOT NULL
        AND note_audience IN ('teacher', 'students', 'both'))
);

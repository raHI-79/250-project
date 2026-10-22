DO $$
DECLARE
    found TEXT;
BEGIN
    SELECT string_agg(DISTINCT quote_literal(value), ', ') INTO found
    FROM (
        SELECT section AS value FROM students
        UNION SELECT section FROM routines
        UNION SELECT section FROM polls
        UNION SELECT section FROM notices
        UNION SELECT section FROM schedule_items
        UNION SELECT section FROM published_items
        UNION SELECT recipient_section FROM notifications
    ) sections
    WHERE value IS NOT NULL AND value NOT IN ('A', 'B', 'ALL');
    IF found IS NOT NULL THEN
        RAISE EXCEPTION 'migration 0021: every section must be A, B or ALL, but the database still holds %', found
            USING HINT = 'Move those people and items to section A or B, then run the migration again.';
    END IF;
    IF EXISTS (SELECT 1 FROM students WHERE section = 'ALL') THEN
        RAISE EXCEPTION 'migration 0021: a student sits in section A or B, never ALL';
    END IF;
END $$;

ALTER TABLE students DROP CONSTRAINT IF EXISTS students_section_check;

ALTER TABLE students ADD CONSTRAINT students_section_check CHECK (section IN ('A', 'B'));

ALTER TABLE routines DROP CONSTRAINT IF EXISTS routines_section_check;

ALTER TABLE routines ADD CONSTRAINT routines_section_check CHECK (section IN ('A', 'B', 'ALL'));

ALTER TABLE polls DROP CONSTRAINT IF EXISTS polls_section_check;

ALTER TABLE polls ADD CONSTRAINT polls_section_check CHECK (section IN ('A', 'B', 'ALL'));

ALTER TABLE notices DROP CONSTRAINT IF EXISTS notices_section_check;

ALTER TABLE notices ADD CONSTRAINT notices_section_check
CHECK (section IS NULL OR section IN ('A', 'B', 'ALL'));

ALTER TABLE schedule_items DROP CONSTRAINT IF EXISTS schedule_items_section_check;

ALTER TABLE schedule_items ADD CONSTRAINT schedule_items_section_check
CHECK (section IN ('A', 'B', 'ALL'));

ALTER TABLE published_items DROP CONSTRAINT IF EXISTS published_items_section_check;

ALTER TABLE published_items ADD CONSTRAINT published_items_section_check
CHECK (section IN ('A', 'B', 'ALL'));

ALTER TABLE notifications DROP CONSTRAINT IF EXISTS notifications_section_check;

ALTER TABLE notifications ADD CONSTRAINT notifications_section_check
CHECK (recipient_section IS NULL OR recipient_section IN ('A', 'B'));

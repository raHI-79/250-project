ALTER TABLE notifications ADD COLUMN IF NOT EXISTS link_kind TEXT;

ALTER TABLE notifications ADD COLUMN IF NOT EXISTS link_ref BIGINT;

ALTER TABLE notifications ADD COLUMN IF NOT EXISTS link_date DATE;

UPDATE notifications
SET link_kind = CASE type
    WHEN 'notice' THEN 'notice'
    WHEN 'message' THEN 'message'
    WHEN 'request' THEN 'teacher_next_day'
    WHEN 'decision' THEN 'cr_upcoming'
    ELSE 'routine'
END
WHERE link_kind IS NULL;

ALTER TABLE notifications DROP CONSTRAINT IF EXISTS notifications_link_kind_check;

ALTER TABLE notifications ADD CONSTRAINT notifications_link_kind_check CHECK (
    link_kind IS NULL OR link_kind IN (
        'teacher_next_day', 'teacher_final', 'cr_upcoming', 'cr_final', 'routine',
        'exam', 'notice', 'message', 'poll', 'cr_role'
    )
);

CREATE INDEX IF NOT EXISTS notifications_link_idx ON notifications (link_kind, link_ref);

DO $$
DECLARE
    link RECORD;
    constraint_name TEXT;
BEGIN
    FOR link IN SELECT * FROM (VALUES
        ('routines', 'teacher_id'),
        ('class_events', 'teacher_id'),
        ('class_events', 'created_by'),
        ('notices', 'created_by'),
        ('polls', 'created_by'),
        ('messages', 'sender_id'),
        ('conversations', 'created_by'),
        ('routine_cancellations', 'cancelled_by'),
        ('batch_drive_links', 'updated_by')
    ) AS pair(child, child_column)
    LOOP
        EXECUTE format('ALTER TABLE %I ALTER COLUMN %I DROP NOT NULL', link.child, link.child_column);
        FOR constraint_name IN
            SELECT c.conname
            FROM pg_constraint c
            JOIN pg_attribute a ON a.attrelid = c.conrelid AND a.attnum = ANY (c.conkey)
            WHERE c.contype = 'f' AND c.conrelid = link.child::regclass AND a.attname = link.child_column
        LOOP
            EXECUTE format('ALTER TABLE %I DROP CONSTRAINT %I', link.child, constraint_name);
        END LOOP;
    END LOOP;
END
$$;

ALTER TABLE routines ADD CONSTRAINT routines_teacher_id_fkey
FOREIGN KEY (teacher_id) REFERENCES teachers (id) ON DELETE SET NULL;

ALTER TABLE class_events ADD CONSTRAINT class_events_teacher_id_fkey
FOREIGN KEY (teacher_id) REFERENCES teachers (id) ON DELETE SET NULL;

ALTER TABLE class_events ADD CONSTRAINT class_events_created_by_fkey
FOREIGN KEY (created_by) REFERENCES users (id) ON DELETE SET NULL;

ALTER TABLE notices ADD CONSTRAINT notices_created_by_fkey
FOREIGN KEY (created_by) REFERENCES users (id) ON DELETE SET NULL;

ALTER TABLE polls ADD CONSTRAINT polls_created_by_fkey
FOREIGN KEY (created_by) REFERENCES users (id) ON DELETE SET NULL;

ALTER TABLE messages ADD CONSTRAINT messages_sender_id_fkey
FOREIGN KEY (sender_id) REFERENCES users (id) ON DELETE SET NULL;

ALTER TABLE conversations ADD CONSTRAINT conversations_created_by_fkey
FOREIGN KEY (created_by) REFERENCES users (id) ON DELETE SET NULL;

ALTER TABLE routine_cancellations ADD CONSTRAINT routine_cancellations_cancelled_by_fkey
FOREIGN KEY (cancelled_by) REFERENCES users (id) ON DELETE SET NULL;

ALTER TABLE batch_drive_links ADD CONSTRAINT batch_drive_links_updated_by_fkey
FOREIGN KEY (updated_by) REFERENCES users (id) ON DELETE SET NULL;

CREATE TABLE IF NOT EXISTS admin_audit (
    id BIGSERIAL PRIMARY KEY,
    admin_id BIGINT REFERENCES users (id) ON DELETE SET NULL,
    action TEXT NOT NULL,
    target_summary TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT admin_audit_action_not_blank CHECK (action <> ''),
    CONSTRAINT admin_audit_target_not_blank CHECK (target_summary <> '')
);

CREATE INDEX IF NOT EXISTS admin_audit_created_idx ON admin_audit (created_at DESC, id DESC);

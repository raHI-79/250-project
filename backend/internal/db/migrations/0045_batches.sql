CREATE TABLE IF NOT EXISTS batches (
    id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    running_semester TEXT NOT NULL DEFAULT '1-1',
    sections TEXT[] NOT NULL DEFAULT ARRAY['A', 'B'],
    status TEXT NOT NULL DEFAULT 'active',
    semester_since TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_by BIGINT REFERENCES users (id) ON DELETE SET NULL,
    updated_by BIGINT REFERENCES users (id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT batches_name_shape CHECK (name ~ '^[A-Z]{2,6}-[0-9]{2}$'),
    CONSTRAINT batches_status_known CHECK (status IN ('active', 'inactive')),
    CONSTRAINT batches_sections_not_empty CHECK (cardinality(sections) > 0),
    CONSTRAINT batches_running_check
    CHECK (running_semester IN ('1-1', '1-2', '2-1', '2-2', '3-1', '3-2', '4-1', '4-2'))
);

CREATE INDEX IF NOT EXISTS batches_status_idx ON batches (status, name);

INSERT INTO batches (name, running_semester, status, semester_since, updated_at)
SELECT used.batch,
    COALESCE(bs.running_semester, '1-1'),
    CASE WHEN EXISTS (
        SELECT 1 FROM students s
        JOIN users u ON u.id = s.user_id
        WHERE s.batch = used.batch AND u.status NOT IN ('removed', 'rejected')
    ) THEN 'active' ELSE 'inactive' END,
    COALESCE(bs.updated_at, now()),
    COALESCE(bs.updated_at, now())
FROM (
    SELECT batch FROM students
    UNION
    SELECT batch FROM routines
    UNION
    SELECT batch FROM schedule_items
    UNION
    SELECT batch FROM batch_semesters
) used
LEFT JOIN batch_semesters bs ON bs.batch = used.batch
WHERE used.batch ~ '^[A-Z]{2,6}-[0-9]{2}$'
ON CONFLICT (name) DO NOTHING;

ALTER TABLE admin_permissions DROP CONSTRAINT IF EXISTS admin_permissions_known;

ALTER TABLE admin_permissions ADD CONSTRAINT admin_permissions_known CHECK (permission IN (
    'registrations', 'students', 'teachers', 'catalog', 'periods',
    'routines', 'notices', 'attendance', 'exports', 'batches'
));

INSERT INTO admin_permissions (user_id, permission, granted_by)
SELECT p.user_id, 'batches', p.granted_by
FROM admin_permissions p
WHERE p.permission = 'catalog'
ON CONFLICT DO NOTHING;

ALTER TABLE notifications DROP CONSTRAINT IF EXISTS notifications_link_kind_check;

ALTER TABLE notifications ADD CONSTRAINT notifications_link_kind_check CHECK (
    link_kind IS NULL OR link_kind IN (
        'teacher_next_day', 'teacher_final', 'cr_upcoming', 'cr_final', 'routine',
        'exam', 'notice', 'message', 'poll', 'cr_role',
        'attendance_mark', 'attendance_mine', 'attendance_teacher', 'phone_requests',
        'batches'
    )
);

CREATE TABLE IF NOT EXISTS admin_permissions (
    user_id BIGINT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    permission TEXT NOT NULL,
    granted_by BIGINT REFERENCES users (id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, permission),
    CONSTRAINT admin_permissions_known CHECK (permission IN (
        'registrations', 'students', 'teachers', 'catalog', 'periods',
        'routines', 'notices', 'attendance', 'exports'
    ))
);

INSERT INTO admin_permissions (user_id, permission)
SELECT u.id, p.permission
FROM users u
CROSS JOIN (VALUES
    ('registrations'), ('students'), ('teachers'), ('catalog'),
    ('periods'), ('routines'), ('notices'), ('attendance')
) AS p (permission)
WHERE u.role = 'admin' OR u.office_access
ON CONFLICT DO NOTHING;

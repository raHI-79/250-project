ALTER TABLE conversations ADD COLUMN IF NOT EXISTS title TEXT;

ALTER TABLE conversation_members ADD COLUMN IF NOT EXISTS role TEXT NOT NULL DEFAULT 'member';

ALTER TABLE conversation_members DROP CONSTRAINT IF EXISTS conversation_members_role_check;

ALTER TABLE conversation_members ADD CONSTRAINT conversation_members_role_check
CHECK (role IN ('admin', 'member'));

ALTER TABLE conversation_members ADD COLUMN IF NOT EXISTS added_by BIGINT
REFERENCES users (id) ON DELETE SET NULL;

ALTER TABLE conversation_members ADD COLUMN IF NOT EXISTS added_at TIMESTAMPTZ NOT NULL DEFAULT now();

UPDATE conversations SET title = CASE
    WHEN scope = 'batch' THEN batch
    WHEN scope = 'section' THEN batch || ' section ' || section
    WHEN scope = 'all_students' THEN 'All students'
    WHEN scope = 'all_teachers' THEN 'All teachers'
    ELSE 'Group chat'
END
WHERE kind = 'group' AND COALESCE(title, '') = '';

INSERT INTO conversation_members (conversation_id, user_id, role, added_by)
SELECT c.id, u.id, 'member', c.created_by
FROM conversations c
JOIN users u ON u.status = 'active'
LEFT JOIN students s ON s.user_id = u.id
WHERE c.kind = 'group' AND (
    (c.scope = 'batch' AND s.batch = c.batch)
    OR (c.scope = 'section' AND s.batch = c.batch AND s.section = c.section)
    OR (c.scope = 'all_students' AND u.role = 'student')
    OR (c.scope = 'all_teachers' AND u.role = 'teacher')
)
ON CONFLICT (conversation_id, user_id) DO NOTHING;

INSERT INTO conversation_members (conversation_id, user_id, role, added_by)
SELECT c.id, c.created_by, 'admin', c.created_by
FROM conversations c
WHERE c.kind = 'group'
ON CONFLICT (conversation_id, user_id) DO UPDATE SET role = 'admin';

ALTER TABLE conversations DROP CONSTRAINT IF EXISTS conversations_kind_shape;

ALTER TABLE conversations DROP CONSTRAINT IF EXISTS conversations_scope_targets;

DROP INDEX IF EXISTS conversations_group_key;

DROP INDEX IF EXISTS conversations_audience_idx;

ALTER TABLE conversations DROP COLUMN IF EXISTS scope;

ALTER TABLE conversations DROP COLUMN IF EXISTS batch;

ALTER TABLE conversations DROP COLUMN IF EXISTS section;

ALTER TABLE conversations ADD CONSTRAINT conversations_kind_shape CHECK (
    (kind = 'direct' AND title IS NULL)
    OR (kind = 'group' AND title IS NOT NULL AND title <> '')
);

CREATE INDEX IF NOT EXISTS conversation_members_role_idx ON conversation_members (conversation_id, role);

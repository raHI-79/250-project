CREATE TABLE IF NOT EXISTS conversations (
    id BIGSERIAL PRIMARY KEY,
    kind TEXT NOT NULL CHECK (kind IN ('direct', 'group')),
    scope TEXT CHECK (scope IN ('batch', 'section', 'all_students', 'all_teachers')),
    batch TEXT,
    section TEXT,
    created_by BIGINT NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_message_at TIMESTAMPTZ,
    legacy_low BIGINT,
    legacy_high BIGINT,
    CONSTRAINT conversations_kind_shape CHECK (
        (kind = 'direct' AND scope IS NULL AND batch IS NULL AND section IS NULL)
        OR (kind = 'group' AND scope IS NOT NULL)
    ),
    CONSTRAINT conversations_scope_targets CHECK (
        scope IS NULL
        OR (scope = 'batch' AND batch IS NOT NULL AND section IS NULL)
        OR (scope = 'section' AND batch IS NOT NULL AND section IS NOT NULL)
        OR (scope IN ('all_students', 'all_teachers') AND batch IS NULL AND section IS NULL)
    )
);

CREATE UNIQUE INDEX IF NOT EXISTS conversations_group_key ON conversations
(created_by, scope, COALESCE(batch, ''), COALESCE(section, '')) WHERE kind = 'group';

CREATE INDEX IF NOT EXISTS conversations_activity_idx ON conversations (last_message_at DESC NULLS LAST);

CREATE INDEX IF NOT EXISTS conversations_audience_idx ON conversations (scope, batch, section) WHERE kind = 'group';

CREATE TABLE IF NOT EXISTS conversation_members (
    conversation_id BIGINT NOT NULL REFERENCES conversations (id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    PRIMARY KEY (conversation_id, user_id)
);

CREATE INDEX IF NOT EXISTS conversation_members_user_idx ON conversation_members (user_id);

ALTER TABLE messages RENAME TO legacy_messages;

CREATE TABLE IF NOT EXISTS messages (
    id BIGSERIAL PRIMARY KEY,
    conversation_id BIGINT NOT NULL REFERENCES conversations (id) ON DELETE CASCADE,
    sender_id BIGINT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    body TEXT NOT NULL DEFAULT '',
    link_url TEXT,
    file_path TEXT,
    file_name TEXT,
    file_mime TEXT,
    file_size BIGINT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT messages_body_length CHECK (char_length(body) <= 4000),
    CONSTRAINT messages_has_content CHECK (body <> '' OR link_url IS NOT NULL OR file_path IS NOT NULL),
    CONSTRAINT messages_file_complete CHECK (
        (file_path IS NULL AND file_name IS NULL AND file_mime IS NULL)
        OR (file_path IS NOT NULL AND file_name IS NOT NULL AND file_mime IS NOT NULL)
    )
);

CREATE INDEX IF NOT EXISTS messages_conversation_idx ON messages (conversation_id, created_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS messages_sender_idx ON messages (sender_id, created_at DESC);

CREATE TABLE IF NOT EXISTS conversation_reads (
    user_id BIGINT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    conversation_id BIGINT NOT NULL REFERENCES conversations (id) ON DELETE CASCADE,
    last_read_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, conversation_id)
);

INSERT INTO conversations (kind, scope, batch, section, created_by, created_at, last_message_at)
SELECT 'group', m.target_type, m.target_batch, m.target_section, m.sender_id,
MIN(m.created_at), MAX(m.created_at)
FROM legacy_messages m
WHERE m.target_type IN ('batch', 'section')
GROUP BY m.target_type, m.target_batch, m.target_section, m.sender_id
ON CONFLICT DO NOTHING;

INSERT INTO conversations (kind, created_by, created_at, last_message_at, legacy_low, legacy_high)
SELECT 'direct', MIN(m.sender_id), MIN(m.created_at), MAX(m.created_at),
LEAST(m.sender_id, m.target_user_id), GREATEST(m.sender_id, m.target_user_id)
FROM legacy_messages m
WHERE m.target_type = 'user' AND m.target_user_id IS NOT NULL
GROUP BY LEAST(m.sender_id, m.target_user_id), GREATEST(m.sender_id, m.target_user_id);

INSERT INTO conversation_members (conversation_id, user_id)
SELECT c.id, c.legacy_low FROM conversations c WHERE c.legacy_low IS NOT NULL
UNION
SELECT c.id, c.legacy_high FROM conversations c WHERE c.legacy_high IS NOT NULL
UNION
SELECT c.id, c.created_by FROM conversations c WHERE c.kind = 'group'
ON CONFLICT DO NOTHING;

INSERT INTO messages (conversation_id, sender_id, body, file_path, file_name, file_mime, created_at)
SELECT c.id, m.sender_id, m.content,
m.file_path,
CASE WHEN m.file_path IS NULL THEN NULL ELSE 'attachment.pdf' END,
CASE WHEN m.file_path IS NULL THEN NULL ELSE 'application/pdf' END,
m.created_at
FROM legacy_messages m
JOIN conversations c
  ON c.kind = 'direct'
 AND c.legacy_low = LEAST(m.sender_id, m.target_user_id)
 AND c.legacy_high = GREATEST(m.sender_id, m.target_user_id)
WHERE m.target_type = 'user' AND m.target_user_id IS NOT NULL
  AND (m.content <> '' OR m.file_path IS NOT NULL);

INSERT INTO messages (conversation_id, sender_id, body, file_path, file_name, file_mime, created_at)
SELECT c.id, m.sender_id, m.content,
m.file_path,
CASE WHEN m.file_path IS NULL THEN NULL ELSE 'attachment.pdf' END,
CASE WHEN m.file_path IS NULL THEN NULL ELSE 'application/pdf' END,
m.created_at
FROM legacy_messages m
JOIN conversations c
  ON c.kind = 'group'
 AND c.created_by = m.sender_id
 AND c.scope = m.target_type
 AND c.batch = m.target_batch
 AND COALESCE(c.section, '') = COALESCE(m.target_section, '')
WHERE m.target_type IN ('batch', 'section')
  AND (m.content <> '' OR m.file_path IS NOT NULL);

ALTER TABLE conversations DROP COLUMN legacy_low;

ALTER TABLE conversations DROP COLUMN legacy_high;

DROP TABLE legacy_messages;

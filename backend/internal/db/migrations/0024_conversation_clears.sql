CREATE TABLE IF NOT EXISTS conversation_clears (
    user_id BIGINT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    conversation_id BIGINT NOT NULL REFERENCES conversations (id) ON DELETE CASCADE,
    cleared_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, conversation_id)
);

CREATE INDEX IF NOT EXISTS conversation_clears_conversation_idx
ON conversation_clears (conversation_id, cleared_at);

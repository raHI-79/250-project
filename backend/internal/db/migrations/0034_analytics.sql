CREATE TABLE IF NOT EXISTS page_views (
    id BIGSERIAL PRIMARY KEY,
    visitor_id TEXT NOT NULL,
    session_id TEXT NOT NULL DEFAULT '',
    path TEXT NOT NULL,
    client TEXT NOT NULL DEFAULT 'web' CHECK (client IN ('web', 'android')),
    device TEXT NOT NULL DEFAULT '',
    app_version TEXT NOT NULL DEFAULT '',
    ip_hash TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT page_views_visitor_not_blank CHECK (visitor_id <> ''),
    CONSTRAINT page_views_path_not_blank CHECK (path <> '')
);

CREATE INDEX IF NOT EXISTS page_views_created_idx ON page_views (created_at DESC);

CREATE INDEX IF NOT EXISTS page_views_visitor_idx ON page_views (visitor_id, created_at DESC);

CREATE INDEX IF NOT EXISTS page_views_path_idx ON page_views (path, created_at DESC);

CREATE TABLE IF NOT EXISTS analytics_daily (
    day DATE NOT NULL,
    client TEXT NOT NULL CHECK (client IN ('web', 'android')),
    views INTEGER NOT NULL DEFAULT 0,
    visitors INTEGER NOT NULL DEFAULT 0,
    sessions INTEGER NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (day, client)
);

CREATE TABLE IF NOT EXISTS system_events (
    id BIGSERIAL PRIMARY KEY,
    kind TEXT NOT NULL,
    ok BOOLEAN NOT NULL DEFAULT TRUE,
    detail TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT system_events_kind_not_blank CHECK (kind <> '')
);

CREATE INDEX IF NOT EXISTS system_events_kind_created_idx ON system_events (kind, created_at DESC);

CREATE TABLE IF NOT EXISTS backup_status (
    id SMALLINT PRIMARY KEY DEFAULT 1,
    last_success_at TIMESTAMPTZ,
    note TEXT NOT NULL DEFAULT '',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT backup_status_single_row CHECK (id = 1)
);

INSERT INTO backup_status (id) VALUES (1) ON CONFLICT (id) DO NOTHING;

CREATE TABLE IF NOT EXISTS device_bindings (
    user_id BIGINT PRIMARY KEY REFERENCES users (id) ON DELETE CASCADE,
    device_id TEXT NOT NULL,
    bound_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_seen_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT device_bindings_device_not_blank CHECK (device_id <> '')
);

CREATE INDEX IF NOT EXISTS device_bindings_device_idx ON device_bindings (device_id);

CREATE TABLE IF NOT EXISTS device_change_requests (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    new_device_id TEXT NOT NULL,
    reason TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'approved', 'rejected')),
    semester TEXT NOT NULL DEFAULT '',
    decided_by BIGINT REFERENCES users (id) ON DELETE SET NULL,
    decision_note TEXT NOT NULL DEFAULT '',
    decided_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT device_change_requests_device_not_blank CHECK (new_device_id <> ''),
    CONSTRAINT device_change_requests_decided_together CHECK ((status = 'pending') = (decided_at IS NULL))
);

CREATE UNIQUE INDEX IF NOT EXISTS device_change_requests_one_pending
ON device_change_requests (user_id) WHERE status = 'pending';

CREATE INDEX IF NOT EXISTS device_change_requests_status_idx
ON device_change_requests (status, created_at DESC);

CREATE INDEX IF NOT EXISTS device_change_requests_user_idx
ON device_change_requests (user_id, created_at DESC);

CREATE TABLE IF NOT EXISTS attendance_sessions (
    id BIGSERIAL PRIMARY KEY,
    course_id BIGINT NOT NULL REFERENCES courses (id) ON DELETE RESTRICT,
    batch TEXT NOT NULL,
    section TEXT NOT NULL CHECK (section IN ('A', 'B', 'ALL')),
    schedule_item_id BIGINT REFERENCES schedule_items (id) ON DELETE SET NULL,
    teacher_id BIGINT NOT NULL REFERENCES teachers (id) ON DELETE RESTRICT,
    started_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    ends_at TIMESTAMPTZ NOT NULL,
    status TEXT NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'closed')),
    teacher_lat DOUBLE PRECISION,
    teacher_lng DOUBLE PRECISION,
    teacher_accuracy_m DOUBLE PRECISION,
    location_updated_at TIMESTAMPTZ,
    token_key TEXT NOT NULL,
    closed_at TIMESTAMPTZ,
    CONSTRAINT attendance_sessions_batch_not_blank CHECK (batch <> ''),
    CONSTRAINT attendance_sessions_token_key_not_blank CHECK (token_key <> ''),
    CONSTRAINT attendance_sessions_window_order CHECK (ends_at > started_at),
    CONSTRAINT attendance_sessions_position_together CHECK ((teacher_lat IS NULL) = (teacher_lng IS NULL)),
    CONSTRAINT attendance_sessions_closed_together CHECK ((status = 'open') = (closed_at IS NULL))
);

CREATE INDEX IF NOT EXISTS attendance_sessions_teacher_idx
ON attendance_sessions (teacher_id, started_at DESC);

CREATE INDEX IF NOT EXISTS attendance_sessions_class_idx
ON attendance_sessions (course_id, batch, section, started_at DESC);

CREATE INDEX IF NOT EXISTS attendance_sessions_open_idx
ON attendance_sessions (batch, section, ends_at) WHERE status = 'open';

CREATE TABLE IF NOT EXISTS attendance_records (
    id BIGSERIAL PRIMARY KEY,
    session_id BIGINT NOT NULL REFERENCES attendance_sessions (id) ON DELETE CASCADE,
    student_id BIGINT NOT NULL REFERENCES students (id) ON DELETE CASCADE,
    status TEXT NOT NULL CHECK (status IN ('present', 'absent')),
    method TEXT NOT NULL CHECK (method IN ('app', 'code', 'manual', 'system')),
    device_id TEXT NOT NULL DEFAULT '',
    lat DOUBLE PRECISION,
    lng DOUBLE PRECISION,
    accuracy_m DOUBLE PRECISION,
    distance_m DOUBLE PRECISION,
    marked_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    note TEXT NOT NULL DEFAULT '',
    edited_by BIGINT REFERENCES users (id) ON DELETE SET NULL,
    edited_at TIMESTAMPTZ,
    CONSTRAINT attendance_records_one_per_student UNIQUE (session_id, student_id),
    CONSTRAINT attendance_records_position_together CHECK ((lat IS NULL) = (lng IS NULL))
);

CREATE UNIQUE INDEX IF NOT EXISTS attendance_records_one_per_device
ON attendance_records (session_id, device_id) WHERE method IN ('app', 'code');

CREATE INDEX IF NOT EXISTS attendance_records_student_idx
ON attendance_records (student_id, marked_at DESC);

CREATE TABLE IF NOT EXISTS attendance_alerts (
    student_id BIGINT NOT NULL REFERENCES students (id) ON DELETE CASCADE,
    course_id BIGINT NOT NULL REFERENCES courses (id) ON DELETE CASCADE,
    last_sent_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (student_id, course_id)
);

ALTER TABLE notifications DROP CONSTRAINT IF EXISTS notifications_link_kind_check;

ALTER TABLE notifications ADD CONSTRAINT notifications_link_kind_check CHECK (
    link_kind IS NULL OR link_kind IN (
        'teacher_next_day', 'teacher_final', 'cr_upcoming', 'cr_final', 'routine',
        'exam', 'notice', 'message', 'poll', 'cr_role',
        'attendance_mark', 'attendance_mine', 'attendance_teacher', 'phone_requests'
    )
);

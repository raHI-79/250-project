CREATE TABLE IF NOT EXISTS users (
    id BIGSERIAL PRIMARY KEY,
    role TEXT NOT NULL CHECK (role IN ('student', 'teacher', 'admin')),
    identifier TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('active', 'pending', 'rejected', 'removed')),
    reset_token TEXT,
    reset_token_expires_at TIMESTAMPTZ,
    token_version INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT users_identifier_not_blank CHECK (identifier <> ''),
    CONSTRAINT users_reset_token_has_expiry CHECK (
        (reset_token IS NULL AND reset_token_expires_at IS NULL)
        OR (reset_token IS NOT NULL AND reset_token_expires_at IS NOT NULL)
    )
);

CREATE INDEX IF NOT EXISTS users_role_status_idx ON users (role, status);

CREATE UNIQUE INDEX IF NOT EXISTS users_reset_token_key ON users (reset_token) WHERE reset_token IS NOT NULL;

CREATE TABLE IF NOT EXISTS students (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL UNIQUE REFERENCES users (id) ON DELETE CASCADE,
    full_name TEXT NOT NULL,
    reg_no TEXT NOT NULL UNIQUE,
    batch TEXT NOT NULL,
    section TEXT NOT NULL,
    email TEXT NOT NULL,
    phone TEXT NOT NULL,
    blood_group TEXT NOT NULL CHECK (blood_group IN ('A+', 'A-', 'B+', 'B-', 'AB+', 'AB-', 'O+', 'O-')),
    has_drop_course BOOLEAN NOT NULL DEFAULT FALSE,
    drop_course_code TEXT,
    is_cr BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT students_full_name_not_blank CHECK (full_name <> ''),
    CONSTRAINT students_batch_not_blank CHECK (batch <> ''),
    CONSTRAINT students_section_not_blank CHECK (section <> ''),
    CONSTRAINT students_drop_course_consistent CHECK (
        (has_drop_course AND drop_course_code IS NOT NULL AND drop_course_code <> '')
        OR (NOT has_drop_course AND drop_course_code IS NULL)
    )
);

CREATE INDEX IF NOT EXISTS students_batch_section_idx ON students (batch, section);

CREATE INDEX IF NOT EXISTS students_blood_group_idx ON students (blood_group);

CREATE INDEX IF NOT EXISTS students_is_cr_idx ON students (batch, section) WHERE is_cr;

CREATE INDEX IF NOT EXISTS students_drop_course_code_idx ON students (drop_course_code) WHERE has_drop_course;

CREATE TABLE IF NOT EXISTS teachers (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL UNIQUE REFERENCES users (id) ON DELETE CASCADE,
    teacher_id TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    email TEXT NOT NULL,
    phone TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT teachers_teacher_id_not_blank CHECK (teacher_id <> ''),
    CONSTRAINT teachers_name_not_blank CHECK (name <> '')
);

CREATE INDEX IF NOT EXISTS teachers_name_idx ON teachers (name);

CREATE TABLE IF NOT EXISTS courses (
    id BIGSERIAL PRIMARY KEY,
    code TEXT NOT NULL UNIQUE,
    title TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT courses_code_not_blank CHECK (code <> ''),
    CONSTRAINT courses_title_not_blank CHECK (title <> '')
);

CREATE TABLE IF NOT EXISTS routines (
    id BIGSERIAL PRIMARY KEY,
    batch TEXT NOT NULL,
    section TEXT NOT NULL,
    course_id BIGINT NOT NULL REFERENCES courses (id) ON DELETE RESTRICT,
    teacher_id BIGINT NOT NULL REFERENCES teachers (id) ON DELETE RESTRICT,
    day_of_week SMALLINT NOT NULL CHECK (day_of_week BETWEEN 0 AND 6),
    start_time TIME NOT NULL,
    end_time TIME NOT NULL,
    room TEXT NOT NULL,
    type TEXT NOT NULL CHECK (type IN ('class', 'lab')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT routines_batch_not_blank CHECK (batch <> ''),
    CONSTRAINT routines_section_not_blank CHECK (section <> ''),
    CONSTRAINT routines_room_not_blank CHECK (room <> ''),
    CONSTRAINT routines_time_order CHECK (start_time < end_time),
    CONSTRAINT routines_class_hours CHECK (start_time >= TIME '08:00' AND end_time <= TIME '17:00')
);

CREATE INDEX IF NOT EXISTS routines_room_day_time_idx ON routines (room, day_of_week, start_time, end_time);

CREATE INDEX IF NOT EXISTS routines_batch_section_day_idx ON routines (batch, section, day_of_week, start_time);

CREATE INDEX IF NOT EXISTS routines_teacher_day_idx ON routines (teacher_id, day_of_week, start_time);

CREATE INDEX IF NOT EXISTS routines_course_idx ON routines (course_id);

CREATE TABLE IF NOT EXISTS class_events (
    id BIGSERIAL PRIMARY KEY,
    batch TEXT NOT NULL,
    section TEXT NOT NULL,
    course_id BIGINT NOT NULL REFERENCES courses (id) ON DELETE RESTRICT,
    teacher_id BIGINT NOT NULL REFERENCES teachers (id) ON DELETE RESTRICT,
    event_date DATE NOT NULL,
    start_time TIME NOT NULL,
    end_time TIME NOT NULL,
    room TEXT NOT NULL,
    event_type TEXT NOT NULL CHECK (event_type IN ('class', 'exam', 'extra')),
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'approved', 'rejected', 'cancelled', 'auto_cancelled')),
    created_by BIGINT NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    decided_at TIMESTAMPTZ,
    CONSTRAINT class_events_batch_not_blank CHECK (batch <> ''),
    CONSTRAINT class_events_section_not_blank CHECK (section <> ''),
    CONSTRAINT class_events_room_not_blank CHECK (room <> ''),
    CONSTRAINT class_events_time_order CHECK (start_time < end_time),
    CONSTRAINT class_events_class_hours CHECK (start_time >= TIME '08:00' AND end_time <= TIME '17:00')
);

CREATE INDEX IF NOT EXISTS class_events_room_date_time_idx ON class_events (room, event_date, start_time, end_time);

CREATE INDEX IF NOT EXISTS class_events_batch_section_date_idx ON class_events (batch, section, event_date);

CREATE INDEX IF NOT EXISTS class_events_teacher_status_idx ON class_events (teacher_id, status, event_date);

CREATE INDEX IF NOT EXISTS class_events_status_date_idx ON class_events (status, event_date);

CREATE INDEX IF NOT EXISTS class_events_created_by_idx ON class_events (created_by, created_at DESC);

CREATE INDEX IF NOT EXISTS class_events_course_idx ON class_events (course_id);

CREATE TABLE IF NOT EXISTS polls (
    id BIGSERIAL PRIMARY KEY,
    question TEXT NOT NULL,
    batch TEXT NOT NULL,
    section TEXT NOT NULL DEFAULT 'ALL',
    status TEXT NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'closed')),
    created_by BIGINT NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    closed_at TIMESTAMPTZ,
    CONSTRAINT polls_question_not_blank CHECK (question <> ''),
    CONSTRAINT polls_batch_not_blank CHECK (batch <> ''),
    CONSTRAINT polls_section_not_blank CHECK (section <> ''),
    CONSTRAINT polls_closed_at_matches_status CHECK (
        (status = 'closed' AND closed_at IS NOT NULL)
        OR (status = 'open' AND closed_at IS NULL)
    )
);

CREATE INDEX IF NOT EXISTS polls_batch_section_status_idx ON polls (batch, section, status, created_at DESC);

CREATE TABLE IF NOT EXISTS poll_options (
    id BIGSERIAL PRIMARY KEY,
    poll_id BIGINT NOT NULL REFERENCES polls (id) ON DELETE CASCADE,
    option_text TEXT NOT NULL,
    position INTEGER NOT NULL DEFAULT 0,
    CONSTRAINT poll_options_text_not_blank CHECK (option_text <> '')
);

CREATE INDEX IF NOT EXISTS poll_options_poll_idx ON poll_options (poll_id, position);

CREATE TABLE IF NOT EXISTS poll_votes (
    id BIGSERIAL PRIMARY KEY,
    poll_id BIGINT NOT NULL REFERENCES polls (id) ON DELETE CASCADE,
    option_id BIGINT NOT NULL REFERENCES poll_options (id) ON DELETE CASCADE,
    student_id BIGINT NOT NULL REFERENCES students (id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT poll_votes_one_per_student UNIQUE (poll_id, student_id)
);

CREATE INDEX IF NOT EXISTS poll_votes_option_idx ON poll_votes (option_id);

CREATE TABLE IF NOT EXISTS notices (
    id BIGSERIAL PRIMARY KEY,
    source TEXT NOT NULL CHECK (source IN ('admin', 'cr')),
    title TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    batch TEXT,
    section TEXT,
    file_path TEXT,
    created_by BIGINT NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT notices_title_not_blank CHECK (title <> ''),
    CONSTRAINT notices_section_needs_batch CHECK (section IS NULL OR batch IS NOT NULL),
    CONSTRAINT notices_cr_needs_batch CHECK (source <> 'cr' OR batch IS NOT NULL)
);

CREATE INDEX IF NOT EXISTS notices_batch_created_idx ON notices (batch, created_at DESC);

CREATE INDEX IF NOT EXISTS notices_source_created_idx ON notices (source, created_at DESC);

CREATE INDEX IF NOT EXISTS notices_created_by_idx ON notices (created_by);

CREATE TABLE IF NOT EXISTS materials (
    id BIGSERIAL PRIMARY KEY,
    batch TEXT NOT NULL DEFAULT 'ALL',
    title TEXT NOT NULL,
    link TEXT NOT NULL,
    created_by BIGINT NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT materials_batch_not_blank CHECK (batch <> ''),
    CONSTRAINT materials_title_not_blank CHECK (title <> ''),
    CONSTRAINT materials_link_not_blank CHECK (link <> '')
);

CREATE INDEX IF NOT EXISTS materials_batch_created_idx ON materials (batch, created_at DESC);

CREATE INDEX IF NOT EXISTS materials_created_by_idx ON materials (created_by, created_at DESC);

CREATE TABLE IF NOT EXISTS messages (
    id BIGSERIAL PRIMARY KEY,
    sender_id BIGINT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    sender_role TEXT NOT NULL CHECK (sender_role IN ('cr', 'teacher', 'admin')),
    sender_name TEXT NOT NULL,
    target_type TEXT NOT NULL CHECK (target_type IN ('user', 'batch', 'section')),
    target_batch TEXT,
    target_section TEXT,
    target_user_id BIGINT REFERENCES users (id) ON DELETE CASCADE,
    content TEXT NOT NULL DEFAULT '',
    file_path TEXT,
    reply_to_id BIGINT REFERENCES messages (id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT messages_sender_name_not_blank CHECK (sender_name <> ''),
    CONSTRAINT messages_has_body CHECK (content <> '' OR file_path IS NOT NULL),
    CONSTRAINT messages_target_matches_type CHECK (
        (target_type = 'user' AND target_user_id IS NOT NULL AND target_batch IS NULL AND target_section IS NULL)
        OR (target_type = 'batch' AND target_user_id IS NULL AND target_batch IS NOT NULL AND target_section IS NULL)
        OR (target_type = 'section' AND target_user_id IS NULL AND target_batch IS NOT NULL AND target_section IS NOT NULL)
    )
);

CREATE INDEX IF NOT EXISTS messages_target_user_idx ON messages (target_user_id, created_at DESC);

CREATE INDEX IF NOT EXISTS messages_target_batch_section_idx ON messages (target_batch, target_section, created_at DESC);

CREATE INDEX IF NOT EXISTS messages_sender_idx ON messages (sender_id, created_at DESC);

CREATE INDEX IF NOT EXISTS messages_reply_to_idx ON messages (reply_to_id);

CREATE TABLE IF NOT EXISTS notifications (
    id BIGSERIAL PRIMARY KEY,
    recipient_user_id BIGINT REFERENCES users (id) ON DELETE CASCADE,
    recipient_role TEXT CHECK (recipient_role IN ('student', 'cr', 'teacher', 'admin')),
    recipient_batch TEXT,
    recipient_section TEXT,
    type TEXT NOT NULL,
    title TEXT NOT NULL,
    body TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT notifications_type_not_blank CHECK (type <> ''),
    CONSTRAINT notifications_title_not_blank CHECK (title <> ''),
    CONSTRAINT notifications_section_needs_batch CHECK (recipient_section IS NULL OR recipient_batch IS NOT NULL),
    CONSTRAINT notifications_has_target CHECK (
        recipient_user_id IS NOT NULL
        OR recipient_role IS NOT NULL
        OR recipient_batch IS NOT NULL
    )
);

CREATE INDEX IF NOT EXISTS notifications_recipient_user_idx ON notifications (recipient_user_id, created_at DESC) WHERE recipient_user_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS notifications_broadcast_idx ON notifications (recipient_role, recipient_batch, recipient_section, created_at DESC) WHERE recipient_user_id IS NULL;

CREATE INDEX IF NOT EXISTS notifications_created_at_idx ON notifications (created_at DESC);

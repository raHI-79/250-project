CREATE TABLE IF NOT EXISTS schedule_items (
    id BIGSERIAL PRIMARY KEY,
    batch TEXT NOT NULL,
    section TEXT NOT NULL,
    course_id BIGINT NOT NULL REFERENCES courses (id) ON DELETE RESTRICT,
    teacher_id BIGINT REFERENCES teachers (id) ON DELETE SET NULL,
    kind TEXT NOT NULL CHECK (kind IN ('class', 'lab', 'exam')),
    origin TEXT NOT NULL CHECK (origin IN ('fixed', 'cr', 'teacher_exam')),
    routine_id BIGINT REFERENCES routines (id) ON DELETE SET NULL,
    item_date DATE NOT NULL,
    start_time TIME,
    end_time TIME,
    mode TEXT NOT NULL DEFAULT 'room' CHECK (mode IN ('room', 'online')),
    room TEXT NOT NULL DEFAULT '',
    meeting_link TEXT NOT NULL DEFAULT '',
    prior_status TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL CHECK (status IN (
        'awaiting_teacher', 'no_response', 'confirmed', 'cancelled',
        'teacher_reschedule_request', 'cr_reschedule_request', 'reschedule_declined',
        'rescheduled', 'scheduled'
    )),
    teacher_message TEXT NOT NULL DEFAULT '',
    cancel_reason TEXT NOT NULL DEFAULT '',
    syllabus TEXT NOT NULL DEFAULT '',
    instructions TEXT NOT NULL DEFAULT '',
    note TEXT NOT NULL DEFAULT '',
    proposed_date DATE,
    proposed_start TIME,
    proposed_end TIME,
    proposed_mode TEXT CHECK (proposed_mode IS NULL OR proposed_mode IN ('room', 'online')),
    proposed_room TEXT NOT NULL DEFAULT '',
    proposed_link TEXT NOT NULL DEFAULT '',
    in_final BOOLEAN NOT NULL DEFAULT FALSE,
    created_by BIGINT REFERENCES users (id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT schedule_items_batch_not_blank CHECK (batch <> ''),
    CONSTRAINT schedule_items_section_not_blank CHECK (section <> ''),
    CONSTRAINT schedule_items_times_together CHECK ((start_time IS NULL) = (end_time IS NULL)),
    CONSTRAINT schedule_items_time_order CHECK (start_time IS NULL OR start_time < end_time),
    CONSTRAINT schedule_items_proposed_times_together CHECK ((proposed_start IS NULL) = (proposed_end IS NULL)),
    CONSTRAINT schedule_items_proposed_time_order CHECK (proposed_start IS NULL OR proposed_start < proposed_end)
);

CREATE UNIQUE INDEX IF NOT EXISTS schedule_items_routine_date_key
ON schedule_items (routine_id, item_date) WHERE routine_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS schedule_items_batch_date_idx ON schedule_items (batch, item_date, start_time);

CREATE INDEX IF NOT EXISTS schedule_items_teacher_date_idx ON schedule_items (teacher_id, item_date, start_time);

CREATE INDEX IF NOT EXISTS schedule_items_date_status_idx ON schedule_items (item_date, status);

CREATE INDEX IF NOT EXISTS schedule_items_room_date_idx ON schedule_items (room, item_date) WHERE mode = 'room';

CREATE INDEX IF NOT EXISTS schedule_items_final_idx ON schedule_items (batch, item_date) WHERE in_final;

CREATE TABLE IF NOT EXISTS publications (
    id BIGSERIAL PRIMARY KEY,
    batch TEXT NOT NULL,
    published_by BIGINT REFERENCES users (id) ON DELETE SET NULL,
    published_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT publications_batch_not_blank CHECK (batch <> '')
);

CREATE INDEX IF NOT EXISTS publications_batch_idx ON publications (batch, published_at DESC);

CREATE TABLE IF NOT EXISTS published_items (
    item_id BIGINT PRIMARY KEY REFERENCES schedule_items (id) ON DELETE CASCADE,
    publication_id BIGINT NOT NULL REFERENCES publications (id) ON DELETE CASCADE,
    batch TEXT NOT NULL,
    section TEXT NOT NULL,
    item_date DATE NOT NULL,
    start_time TIME,
    end_time TIME,
    course_code TEXT NOT NULL,
    course_title TEXT NOT NULL,
    teacher_name TEXT NOT NULL DEFAULT '',
    kind TEXT NOT NULL,
    mode TEXT NOT NULL,
    room TEXT NOT NULL DEFAULT '',
    meeting_link TEXT NOT NULL DEFAULT '',
    note TEXT NOT NULL DEFAULT '',
    syllabus TEXT NOT NULL DEFAULT '',
    instructions TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL,
    published_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS published_items_batch_date_idx ON published_items (batch, item_date, start_time);

CREATE INDEX IF NOT EXISTS published_items_course_idx ON published_items (course_code, item_date);

INSERT INTO schedule_items
(batch, section, course_id, teacher_id, kind, origin, item_date, start_time, end_time,
mode, room, status, note, in_final, created_by, created_at)
SELECT e.batch, e.section, e.course_id, e.teacher_id,
CASE WHEN e.event_type = 'exam' THEN 'exam' ELSE 'class' END,
'cr', e.event_date, e.start_time, e.end_time, 'room', e.room,
CASE WHEN e.status IN ('approved', 'pending') THEN 'scheduled' ELSE 'cancelled' END,
CASE WHEN e.note_audience IN ('students', 'both') THEN COALESCE(e.note, '') ELSE '' END,
e.status = 'approved', e.created_by, e.created_at
FROM class_events e;

INSERT INTO schedule_items
(batch, section, course_id, teacher_id, kind, origin, routine_id, item_date, start_time, end_time,
mode, room, status, cancel_reason, created_by, created_at)
SELECT r.batch, r.section, r.course_id, r.teacher_id, r.type, 'fixed', r.id, rc.cancel_date,
r.start_time, r.end_time, 'room', r.room, 'cancelled', 'Cancelled before the scheduling engine',
rc.cancelled_by, rc.created_at
FROM routine_cancellations rc
JOIN routines r ON r.id = rc.routine_id
ON CONFLICT DO NOTHING;

INSERT INTO publications (batch, published_at)
SELECT DISTINCT i.batch, now()
FROM schedule_items i
WHERE i.in_final;

INSERT INTO published_items
(item_id, publication_id, batch, section, item_date, start_time, end_time, course_code, course_title,
teacher_name, kind, mode, room, meeting_link, note, syllabus, instructions, status, published_at)
SELECT i.id, p.id, i.batch, i.section, i.item_date, i.start_time, i.end_time, c.code, c.title,
COALESCE(t.name, ''), i.kind, i.mode, i.room, i.meeting_link, i.note, i.syllabus, i.instructions,
i.status, now()
FROM schedule_items i
JOIN courses c ON c.id = i.course_id
JOIN publications p ON p.batch = i.batch
LEFT JOIN teachers t ON t.id = i.teacher_id
WHERE i.in_final
ON CONFLICT (item_id) DO NOTHING;

DROP TABLE IF EXISTS routine_cancellations;

DROP TABLE IF EXISTS class_events;

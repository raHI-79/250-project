ALTER TABLE users DROP CONSTRAINT IF EXISTS users_status_check;

ALTER TABLE users ADD CONSTRAINT users_status_check CHECK (
    status IN ('active', 'pending', 'rejected', 'removed', 'unverified', 'invited')
);

ALTER TABLE users ADD COLUMN IF NOT EXISTS photo_path TEXT;

ALTER TABLE users ADD COLUMN IF NOT EXISTS email_code TEXT;

ALTER TABLE users ADD COLUMN IF NOT EXISTS email_code_expires_at TIMESTAMPTZ;

ALTER TABLE users ADD COLUMN IF NOT EXISTS email_code_attempts INTEGER NOT NULL DEFAULT 0;

ALTER TABLE users ADD COLUMN IF NOT EXISTS email_code_sent_at TIMESTAMPTZ;

ALTER TABLE users ADD COLUMN IF NOT EXISTS email_verified_at TIMESTAMPTZ;

ALTER TABLE users ADD COLUMN IF NOT EXISTS invite_token TEXT;

ALTER TABLE users ADD COLUMN IF NOT EXISTS invite_expires_at TIMESTAMPTZ;

ALTER TABLE users ADD COLUMN IF NOT EXISTS invite_attempts INTEGER NOT NULL DEFAULT 0;

ALTER TABLE users DROP CONSTRAINT IF EXISTS users_email_code_attempts_not_negative;

ALTER TABLE users ADD CONSTRAINT users_email_code_attempts_not_negative CHECK (email_code_attempts >= 0);

ALTER TABLE users DROP CONSTRAINT IF EXISTS users_invite_attempts_not_negative;

ALTER TABLE users ADD CONSTRAINT users_invite_attempts_not_negative CHECK (invite_attempts >= 0);

ALTER TABLE users DROP CONSTRAINT IF EXISTS users_email_code_has_expiry;

ALTER TABLE users ADD CONSTRAINT users_email_code_has_expiry CHECK (
    (email_code IS NULL AND email_code_expires_at IS NULL)
    OR (email_code IS NOT NULL AND email_code_expires_at IS NOT NULL)
);

ALTER TABLE users DROP CONSTRAINT IF EXISTS users_invite_has_expiry;

ALTER TABLE users ADD CONSTRAINT users_invite_has_expiry CHECK (
    (invite_token IS NULL AND invite_expires_at IS NULL)
    OR (invite_token IS NOT NULL AND invite_expires_at IS NOT NULL)
);

CREATE INDEX IF NOT EXISTS users_invite_token_idx ON users (invite_token) WHERE invite_token IS NOT NULL;

ALTER TABLE students ADD COLUMN IF NOT EXISTS date_of_birth DATE;

ALTER TABLE teachers ADD COLUMN IF NOT EXISTS designation TEXT NOT NULL DEFAULT '';

ALTER TABLE teachers ADD COLUMN IF NOT EXISTS office_room TEXT NOT NULL DEFAULT '';

ALTER TABLE teachers ADD COLUMN IF NOT EXISTS date_of_birth DATE;

ALTER TABLE teachers ADD COLUMN IF NOT EXISTS blood_group TEXT;

ALTER TABLE teachers ADD COLUMN IF NOT EXISTS joining_date DATE;

ALTER TABLE teachers ADD COLUMN IF NOT EXISTS bio TEXT NOT NULL DEFAULT '';

ALTER TABLE teachers DROP CONSTRAINT IF EXISTS teachers_blood_group_check;

ALTER TABLE teachers ADD CONSTRAINT teachers_blood_group_check CHECK (
    blood_group IS NULL OR blood_group IN ('A+', 'A-', 'B+', 'B-', 'AB+', 'AB-', 'O+', 'O-')
);

CREATE TABLE IF NOT EXISTS student_drop_courses (
    id BIGSERIAL PRIMARY KEY,
    student_id BIGINT NOT NULL REFERENCES students (id) ON DELETE CASCADE,
    course_id BIGINT NOT NULL REFERENCES courses (id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT student_drop_courses_one_per_course UNIQUE (student_id, course_id)
);

CREATE INDEX IF NOT EXISTS student_drop_courses_course_idx ON student_drop_courses (course_id);

INSERT INTO student_drop_courses (student_id, course_id)
SELECT s.id, c.id
FROM students s
JOIN courses c ON c.code = s.drop_course_code
WHERE s.has_drop_course AND s.drop_course_code IS NOT NULL
ON CONFLICT DO NOTHING;

DROP INDEX IF EXISTS students_drop_course_code_idx;

ALTER TABLE students DROP CONSTRAINT IF EXISTS students_drop_course_consistent;

ALTER TABLE students DROP COLUMN IF EXISTS has_drop_course;

ALTER TABLE students DROP COLUMN IF EXISTS drop_course_code;

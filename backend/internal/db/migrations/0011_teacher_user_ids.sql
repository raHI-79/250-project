ALTER TABLE users ALTER COLUMN identifier DROP NOT NULL;

ALTER TABLE users DROP CONSTRAINT IF EXISTS users_identifier_not_blank;

ALTER TABLE users ADD CONSTRAINT users_identifier_not_blank CHECK (identifier IS NULL OR identifier <> '');

CREATE UNIQUE INDEX IF NOT EXISTS users_identifier_lower_key ON users (lower(identifier));

ALTER TABLE teachers ALTER COLUMN teacher_id DROP NOT NULL;

ALTER TABLE teachers DROP CONSTRAINT IF EXISTS teachers_teacher_id_not_blank;

ALTER TABLE teachers ADD CONSTRAINT teachers_teacher_id_not_blank CHECK (teacher_id IS NULL OR teacher_id <> '');

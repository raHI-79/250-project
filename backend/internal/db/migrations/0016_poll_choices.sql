ALTER TABLE polls ADD COLUMN IF NOT EXISTS max_choices INTEGER NOT NULL DEFAULT 1;

ALTER TABLE polls DROP CONSTRAINT IF EXISTS polls_max_choices_positive;

ALTER TABLE polls ADD CONSTRAINT polls_max_choices_positive CHECK (max_choices >= 1);

ALTER TABLE poll_votes DROP CONSTRAINT IF EXISTS poll_votes_one_per_student;

ALTER TABLE poll_votes DROP CONSTRAINT IF EXISTS poll_votes_one_per_option;

ALTER TABLE poll_votes ADD CONSTRAINT poll_votes_one_per_option UNIQUE (poll_id, student_id, option_id);

CREATE INDEX IF NOT EXISTS poll_votes_student_idx ON poll_votes (poll_id, student_id);

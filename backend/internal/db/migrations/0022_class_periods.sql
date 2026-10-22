CREATE TABLE IF NOT EXISTS class_periods (
    id BIGSERIAL PRIMARY KEY,
    label TEXT NOT NULL,
    start_time TIME NOT NULL,
    end_time TIME NOT NULL,
    is_break BOOLEAN NOT NULL DEFAULT FALSE,
    position INTEGER NOT NULL,
    CONSTRAINT class_periods_label_not_blank CHECK (label <> ''),
    CONSTRAINT class_periods_window CHECK (start_time < end_time)
);

CREATE UNIQUE INDEX IF NOT EXISTS class_periods_position_idx ON class_periods (position);

INSERT INTO class_periods (label, start_time, end_time, is_break, position)
SELECT seed.label, seed.start_time::time, seed.end_time::time, seed.is_break, seed.position
FROM (VALUES
    ('Period 1', '09:00', '09:45', FALSE, 1),
    ('Period 2', '09:45', '10:30', FALSE, 2),
    ('Period 3', '10:30', '11:15', FALSE, 3),
    ('Period 4', '11:15', '12:00', FALSE, 4),
    ('Period 5', '12:00', '12:45', FALSE, 5),
    ('Break', '12:45', '13:45', TRUE, 6),
    ('Period 6', '13:45', '14:30', FALSE, 7),
    ('Period 7', '14:30', '15:15', FALSE, 8),
    ('Period 8', '15:15', '16:00', FALSE, 9),
    ('Period 9', '16:00', '16:45', FALSE, 10)
) AS seed (label, start_time, end_time, is_break, position)
WHERE NOT EXISTS (SELECT 1 FROM class_periods);

ALTER TABLE routines DROP CONSTRAINT IF EXISTS routines_class_hours;

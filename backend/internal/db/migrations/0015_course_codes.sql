DO $$
DECLARE
    clashes TEXT;
BEGIN
    SELECT string_agg(normalized, ', ' ORDER BY normalized) INTO clashes
    FROM (
        SELECT btrim(regexp_replace(upper(replace(code, '-', ' ')), '\s+', ' ', 'g')) AS normalized
        FROM courses
        GROUP BY 1
        HAVING count(*) > 1
    ) duplicates;
    IF clashes IS NOT NULL THEN
        RAISE EXCEPTION 'course codes collide after normalisation: %. Merge or rename those courses, then run the migration again.', clashes;
    END IF;
END
$$;

UPDATE courses
SET code = btrim(regexp_replace(upper(replace(code, '-', ' ')), '\s+', ' ', 'g'))
WHERE code <> btrim(regexp_replace(upper(replace(code, '-', ' ')), '\s+', ' ', 'g'));

ALTER TABLE courses DROP CONSTRAINT IF EXISTS courses_code_shape;

ALTER TABLE courses ADD CONSTRAINT courses_code_shape CHECK (
    char_length(code) BETWEEN 3 AND 25 AND code ~ '^[A-Z0-9 ]+$' AND code ~ '[0-9]'
);

CREATE INDEX IF NOT EXISTS courses_code_squeezed_idx ON courses (replace(code, ' ', ''));

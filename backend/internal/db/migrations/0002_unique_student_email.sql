CREATE UNIQUE INDEX IF NOT EXISTS students_email_key ON students (lower(email));

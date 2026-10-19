# SUST CSE Class & Exam Coordination System

A database-driven web application that helps the SUST CSE department coordinate
classes, exams, notices, materials and communication between students, Class
Representatives (CRs), teachers and the department administration. Students see
their upcoming routine and exams, CRs request classes and exams that teachers
approve, teachers manage their daily schedule and messages, and the
administration manages users, courses and the fixed weekly routine from a
hidden admin panel. Built as a course project for **CSE 250**.

The full requirements live in [docs/SPEC.md](docs/SPEC.md). The build order is
tracked in [docs/ROADMAP.md](docs/ROADMAP.md).

## Team

| Member | Name | GitHub |
|---|---|---|
| A | Ratul Islam | [@raHI-79](https://github.com/raHI-79) |
| B | Rakibul Hasan | [@niloySapiens](https://github.com/niloySapiens) |

## Tech stack

| Layer | Technology |
|---|---|
| Backend | Go 1.25+, standard library `net/http` |
| Database | PostgreSQL 14+ via `pgx/v5` with a `pgxpool` connection pool |
| Authentication | bcrypt password hashing + JWT (HS256) via `golang-jwt/jwt/v5` |
| Frontend | React 19 + Vite 7 + React Router + Tailwind CSS v4 |
| File uploads | PDFs in `backend/uploads/` and profile photos in `backend/uploads/photos/` (`UPLOAD_DIR`), path saved in the database |

## Folder structure

```
250-project/
├── backend/                  Go module (the API server)
│   ├── cmd/server/           main.go — program entry point
│   ├── internal/
│   │   ├── auth/             bcrypt hashing, JWT issuing and parsing, roles
│   │   ├── config/           reads backend/.env and environment variables
│   │   ├── db/               pgxpool connection + embedded SQL migrations
│   │   │   └── migrations/   schema files, applied in filename order
│   │   ├── dbtest/           shared helper for the database integration tests
│   │   ├── handler/          one file per group of endpoints
│   │   ├── httpx/            shared JSON response helpers
│   │   ├── middleware/       VerifyToken, RequireRole, Logger, Recoverer, CORS
│   │   ├── server/           router: URL -> handler
│   │   └── store/            database queries used by the API
│   ├── .env.example          copy to .env and fill in your values
│   └── go.mod
├── frontend/                 React + Vite app (the browser client)
│   ├── public/campus/        campus photos used by the banners and heroes
│   └── src/
│       ├── components/ui/    Button, Card, Field, Alert, Toast, Modal, Skeleton
│       ├── components/layout/ sidebar, header, banner, notification bell
│       ├── context/          the signed-in session
│       ├── lib/              paths, API client, navigation, helpers
│       └── pages/            one file per screen
├── docs/
│   ├── SPEC.md               full requirements (do not edit)
│   └── ROADMAP.md            the development steps and their status
├── .gitignore
└── README.md
```

## Requirements

- [Go 1.25 or newer](https://go.dev/dl/) — check with `go version`
- [PostgreSQL 14 or newer](https://www.postgresql.org/download/) — the server
  does not start without it

## Database setup

The backend connects to PostgreSQL at startup and creates every table itself, so
all you have to do is create two empty databases and set your password.

1. **Install PostgreSQL.** On Windows, use the installer from
   [postgresql.org](https://www.postgresql.org/download/windows/) and remember
   the password you set for the `postgres` user.

2. **Create the two databases** — one for the app, one for tests:

   ```powershell
   # Windows PowerShell. It will ask for the postgres password.
   psql -U postgres -c "CREATE DATABASE sust_coord;"
   psql -U postgres -c "CREATE DATABASE sust_coord_test;"
   ```

   If `psql` is not found, add PostgreSQL's `bin` folder to your PATH, for
   example `C:\Program Files\PostgreSQL\17\bin`.

3. **Create your local env file** by copying the example:

   ```powershell
   Copy-Item backend\.env.example backend\.env
   ```

4. **Edit `backend/.env`**:

   - replace `CHANGE_ME` with your own postgres password in both
     `DATABASE_URL` and `TEST_DATABASE_URL`,
   - set `JWT_SECRET` to a long random string of **at least 32 characters**
     (the server refuses to start otherwise). To generate one:

     ```powershell
     -join ((1..64) | ForEach-Object { '{0:x}' -f (Get-Random -Max 16) })
     ```

   - set `ADMIN_IDENTIFIER` and `ADMIN_PASSWORD` if you want the hidden admin
     account created for you at startup,
   - leave `STUDENT_EMAIL_DOMAIN` and `TEACHER_EMAIL_DOMAIN` alone unless your
     university uses other domains, and set `GOOGLE_CLIENT_ID` only if you want
     the Google Sign-In shortcut on the registration page.

   `backend/.env` is listed in `.gitignore`, so your password is never
   committed. Only `backend/.env.example` is tracked.

5. **Start the server once.** It creates every table automatically:

   ```powershell
   cd backend
   go run ./cmd/server
   ```

   ```
   db: connected
   db: applied migration 0001_initial_schema.sql
   server: listening on http://localhost:8080 (env=development, timezone=Asia/Dhaka)
   ```

   The SQL files in `backend/internal/db/migrations/` are embedded in the binary,
   applied in filename order, each inside its own transaction, and recorded in a
   `schema_migrations` table. Starting the server again applies nothing and
   prints `db: schema is up to date`, so running it many times is safe.

   To look at the tables yourself:

   ```powershell
   psql -U postgres -d sust_coord -c "\dt"
   ```

## How to run the backend

```powershell
cd backend
go run ./cmd/server
```

You should see:

```
db: connected
db: schema is up to date
server: listening on http://localhost:8080 (env=development, timezone=Asia/Dhaka)
```

The server stops with a clear error when `DATABASE_URL` is missing or PostgreSQL
is not running, so it never starts with a broken database connection.

Then, in a second terminal, check the health endpoint:

```powershell
Invoke-RestMethod http://localhost:8080/api/health
```

```
status environment database time
------ ----------- -------- ----
ok     development ok       2026-10-16T10:30:00+06:00
```

`database` is `ok` when the server can reach PostgreSQL and `unavailable` when
it cannot.

`Invoke-RestMethod` parses the JSON into an object. To see the raw JSON text
instead, use `Invoke-WebRequest`:

```powershell
(Invoke-WebRequest http://localhost:8080/api/health).Content
```

```json
{"status":"ok","environment":"development","database":"ok","time":"2026-10-16T10:30:00+06:00"}
```

You can also just open <http://localhost:8080/api/health> in a browser, which
is the quickest way to check the server is up.

Press **Ctrl+C** in the first terminal to stop the server. It shuts down
gracefully, finishing any request that is still in progress.

### Changing the port

The server reads its settings from environment variables, so nothing needs to
be edited in the code. On startup it also reads `backend/.env` (when started
from the `backend/` folder) and only fills in variables that are **not** already
set, so a real environment variable always wins:

```powershell
$env:PORT = "9090"
go run ./cmd/server
```

| Variable | Default | Meaning |
|---|---|---|
| `PORT` | `8080` | Port the server listens on |
| `APP_ENV` | `development` | Environment name reported by `/api/health` |
| `APP_TIMEZONE` | `Asia/Dhaka` | Timezone used for all dates and times |
| `DATABASE_URL` | none — **required** | PostgreSQL connection string for the app database |
| `TEST_DATABASE_URL` | none — optional | Connection string used by the database integration tests |
| `JWT_SECRET` | none — **required** | Key that signs the login tokens, at least 32 characters |
| `JWT_TTL` | `24h` | How long a login token stays valid, as a Go duration (`24h`, `90m`) |
| `AUTO_CANCEL_INTERVAL` | `5m` | How often the auto-cancel job looks for requests nobody approved in time |
| `UPLOAD_DIR` | `uploads` | Folder where the uploaded PDFs and profile photos are saved, created when it is missing |
| `MAX_UPLOAD_MB` | `10` | Largest accepted PDF in megabytes; a bigger file gets `413` |
| `STUDENT_EMAIL_DOMAIN` | `student.sust.edu` | Domain every student email address must end with |
| `TEACHER_EMAIL_DOMAIN` | `sust.edu` | Domain every teacher email address must end with |
| `GOOGLE_CLIENT_ID` | none — optional | Google Sign-In client id; empty turns `google_enabled` off |
| `FRONTEND_ORIGIN` | `http://localhost:5173` | The only browser origin allowed to call the API |
| `ADMIN_IDENTIFIER` | none — optional | Login id of the admin account created at startup |
| `ADMIN_PASSWORD` | none — optional | Password of that admin account, at least 8 characters |
| `ADMIN_EMAIL` | none — optional | Email the admin receives password reset codes on |
| `ADMIN_PHONE` | none — optional | Phone number the admin proves a password reset with |
| `SMTP_HOST` | none — optional | SMTP server that sends the reset codes; empty means no mailer is configured |
| `SMTP_PORT` | `587` | Port of that SMTP server |
| `SMTP_USERNAME` | none — optional | SMTP user; when it is empty the server connects without authenticating |
| `SMTP_PASSWORD` | none — optional | SMTP password |
| `MAIL_FROM` | the SMTP username | Address the reset emails come from |

Never commit real values. `backend/.env` is gitignored; only
`backend/.env.example` is tracked.

### The admin account

The admin account is **not** part of the database schema and there is no
registration form for it. When **both** `ADMIN_IDENTIFIER` and
`ADMIN_PASSWORD` are set, the server hashes the password with bcrypt at
startup and inserts one `active` user with role `admin`. The insert uses
`ON CONFLICT (identifier) DO NOTHING`, so:

- the account is created **once**, on the first start with an empty database;
- a later start **never overwrites** the stored hash, even if you change
  `ADMIN_PASSWORD` in `.env` (change the password through the API instead);
- the log line is only `auth: admin account ready` — the identifier and the
  password are never printed or logged.

## How to run the frontend

The browser client lives in `frontend/`. It needs **Node.js 20.19+ or 22.12+**
and talks to the API through a relative `/api` URL, so the backend has to be
running on port `8080` first.

```powershell
cd frontend
npm install
npm run dev
```

Then open <http://localhost:5173>. The dev server proxies `/api` and `/uploads`
to `http://localhost:8080`, so no API base URL is configured anywhere in the
code.

| Command | What it does |
|---|---|
| `npm run dev` | Dev server with hot reload on port `5173` |
| `npm run lint` | ESLint over every `.js` and `.jsx` file |
| `npm run build` | Production bundle in `frontend/dist/` |
| `npm run preview` | Serves the built bundle |

Students, CRs and teachers log in at `/login`. The administration logs in at
`/portal-admin`, which is never linked from any normal page.

### Campus photos

The login heroes and the dashboard banner read their images from
`frontend/public/campus/`. The folder is empty on purpose — drop the real
photos in with exactly these names and nothing else has to change:

| File | Size | Used by |
|---|---|---|
| `hero-monument.jpg`, `hero-building.jpg`, `hero-entrance.jpg` | 1920 × 1080 | the login backgrounds |
| `banner-monument.jpg`, `banner-building.jpg`, `banner-entrance.jpg` | 1600 × 260 | the dashboard banner |

When a photo is missing the banner falls back to its green gradient, so the app
still looks finished without them.

## How to run tests

```powershell
cd backend
go test ./...
```

For more detail, including the name of every test that runs:

```powershell
go test ./... -v
```

Before committing, also run the formatter and the vet checks:

```powershell
gofmt -l .        # prints nothing when all files are formatted
go vet ./...      # prints nothing when there are no suspicious patterns
```

The database integration tests in `backend/internal/db` read
`TEST_DATABASE_URL` from `backend/.env` and skip themselves when it is not set,
so `go test ./...` still passes on a machine without PostgreSQL. When it is set
they refuse to run unless the database name ends with `_test`, and they drop and
recreate the `public` schema before each test — so never point them at
`sust_coord`.

## API endpoints

Every route lives under `/api`. Errors always use the same shape:
`{"error": "message"}`.

| Method | Path | Auth | Description |
|---|---|---|---|
| `GET` | `/api/health` | public | `status`, `environment`, `database` (`ok` / `unavailable`) and the current time |
| `GET` | `/api/auth/config` | public | `student_email_domain`, `teacher_email_domain` and `google_enabled` |
| `POST` | `/api/auth/register` | public | `multipart/form-data` student registration with a required photo; `201` on success |
| `POST` | `/api/auth/verify-email` | public, rate limited | `{reg_no, code}` turns an `unverified` account into `pending` |
| `POST` | `/api/auth/resend-verification` | public, rate limited | `{reg_no}` sends a new 6-digit code; `429` inside the 60 second cooldown |
| `GET` | `/api/auth/drop-course-options` | public, rate limited | Up to 20 courses that appear in a fixed routine, matched by `search` |
| `POST` | `/api/auth/google/verify-email` | public, rate limited | `{credential}` from Google Sign-In; answers `{email, email_proof}` |
| `GET` | `/api/auth/invite` | public | `token=` answers `{name, designation, masked_email}` of a teacher invitation |
| `POST` | `/api/auth/activate` | public, rate limited | `multipart/form-data`; a teacher sets their own password from the invitation |
| `POST` | `/api/auth/login` | public, rate limited | Student, CR and teacher login. Returns a token and the user |
| `POST` | `/api/auth/admin/login` | public, rate limited | Admin login only. Returns a token and the admin user |
| `POST` | `/api/auth/forgot-password` | public, rate limited | Emails a 6-digit reset code valid for 10 minutes |
| `POST` | `/api/auth/verify-reset-code` | public, rate limited | Exchanges that code for a reset token valid for 10 minutes |
| `POST` | `/api/auth/reset-password` | public | Sets a new password with that reset token |
| `POST` | `/api/auth/logout` | any signed-in role | Invalidates **every** token of the account |
| `GET` | `/api/me` | any signed-in role | The signed-in user: `id`, `user_id`, `identifier`, `role`, `photo_url` |
| `GET` | `/api/profile` | any signed-in role | The caller's own profile, shaped by their role |
| `PATCH` | `/api/profile` | any signed-in role | `multipart/form-data`; only the fields that role may change |
| `POST` | `/api/profile/password` | any signed-in role | `{current_password, new_password, confirm_password}`; logs every session out |
| `GET` | `/api/people/{user_id}` | any signed-in role | Public card of an active student, CR or teacher; `404` otherwise |
| `GET` | `/api/admin/registrations` | admin | Pending student registrations, oldest first, with their drop course |
| `POST` | `/api/admin/registrations/{id}/approve` | admin | Turns a pending account into `active`; `404` unknown, `409` already decided |
| `POST` | `/api/admin/registrations/{id}/reject` | admin | Turns a pending account into `rejected`; `404` unknown, `409` already decided |
| `GET` | `/api/admin/students` | admin | `{items, total}` with `search`, `batch`, `section`, `status`, `limit`, `offset` |
| `POST` | `/api/admin/students/{id}/remove` | admin | Sets `removed` and invalidates the tokens; `409` if already removed |
| `POST` | `/api/admin/crs/promote` | admin | `{user_id}` makes an active student a CR; `404` unknown, `409` already a CR |
| `GET` | `/api/admin/crs` | admin | CR list with `search` and `batch` |
| `POST` | `/api/admin/crs/{id}/demote` | admin | Clears `is_cr` and invalidates the tokens; `409` if not a CR |
| `POST` | `/api/admin/crs/{id}/deactivate` | admin | Sets `removed` and invalidates the tokens; `409` if already removed |
| `POST` | `/api/admin/teachers` | admin | `multipart/form-data` or JSON; invites a teacher, no password; `409` on a duplicate |
| `GET` | `/api/admin/teachers` | admin | Teacher list with `search` and `status` |
| `PATCH` | `/api/admin/teachers/{id}` | admin | Changes the admin-owned teacher fields; `409` on a duplicate |
| `POST` | `/api/admin/teachers/{id}/resend-invite` | admin | New invitation link, old one dead; `409` once the teacher is active |
| `POST` | `/api/admin/teachers/{id}/deactivate` | admin | Sets `removed` and invalidates the tokens; `409` if already removed |
| `GET` | `/api/admin/stats` | admin | `total_students`, `total_crs`, `total_teachers`, `pending_approvals` |
| `POST` | `/api/admin/courses` | admin | Adds a course; the code is stored in upper case, `409` on a duplicate |
| `GET` | `/api/admin/courses` | admin | Every course, sorted by code |
| `POST` | `/api/admin/routines` | admin | Adds a fixed routine entry; `400` on a broken rule, `409` on a room clash |
| `GET` | `/api/admin/routines` | admin | Routine entries with the `batch` and `section` filters |
| `DELETE` | `/api/admin/routines/{id}` | admin | Deletes one entry; `204` on success, `404` if it does not exist |
| `GET` | `/api/lookups/batches` | any signed-in role | Batches of the active students and the routine entries |
| `GET` | `/api/lookups/sections` | any signed-in role | Sections of one `batch` (required), without `ALL` |
| `GET` | `/api/lookups/courses` | any signed-in role | `id`, `code` and `title` of every course |
| `GET` | `/api/lookups/teachers` | any signed-in role | `id`, `user_id` and `name` of the active teachers |
| `GET` | `/api/notifications` | student, CR, teacher | Newest first; `limit` defaults to 30 and may not exceed 100 |
| `GET` | `/api/notifications/unread-count` | student, CR, teacher | How many matching notifications arrived after `since` |
| `GET` | `/api/cr/routine` | CR | Fixed routine of the CR's own batch and section, including the `ALL` entries |
| `GET` | `/api/cr/courses` | CR | Running-semester courses of the CR's batch with their teacher and weekly `slots` |
| `GET` | `/api/cr/day` | CR | Everything already on a `date` for the CR's batch: the fixed routine of that weekday and the pending or approved events |
| `POST` | `/api/cr/class-requests` | CR | Asks the chosen teacher for a class, exam or extra class; `201` with a pending request |
| `GET` | `/api/cr/room-availability` | CR | Everything booked on a `date`, optionally narrowed to one `room` |
| `GET` | `/api/cr/requests` | CR | The CR's own requests, newest first, with an optional `status` filter |
| `GET` | `/api/teacher/requests` | teacher | Pending requests naming this teacher, oldest first, with who asked |
| `POST` | `/api/teacher/requests/{id}/approve` | teacher | Approves a pending request dated tomorrow or later |
| `POST` | `/api/teacher/requests/{id}/reject` | teacher | Rejects a pending request |
| `GET` | `/api/teacher/schedule` | teacher | This teacher's fixed routine for a `date` plus the dated classes of that day |
| `GET` | `/api/teacher/classes/upcoming` | teacher | Approved classes from today onward, or only those of one `date` |
| `POST` | `/api/teacher/classes/{id}/cancel` | teacher | Cancels one approved class of this teacher |
| `POST` | `/api/teacher/routines/{id}/cancel` | teacher | Cancels one `date` of this teacher's fixed routine entry |
| `GET` | `/api/students/today` | student, CR | Today's summary: counts, the next exam and the three newest notices |
| `GET` | `/api/students/upcoming-routine` | student, CR | The batch's plan for `date` (today by default), grouped by section plus `drop_courses` |
| `GET` | `/api/students/exams` | student, CR | Approved exams of the own batch and section from today onward |
| `GET` | `/api/students/routine` | student, CR | Fixed routine browser with `batch`, `section`, `day` and `course` filters |
| `POST` | `/api/admin/notices` | admin | `multipart/form-data` with `title`, optional `description`, optional `batch`, optional `link` and optional `file`; an empty `batch` means every batch |
| `GET` | `/api/admin/notices` | admin | Every administration notice, newest first |
| `DELETE` | `/api/admin/notices/{id}` | admin | Deletes one administration notice and its attachment; `204` on success, `404` if it does not exist |
| `POST` | `/api/cr/notices` | CR | `multipart/form-data` with `title`, optional `description`, `target` (`own` or another batch code), optional `link` and optional `file` |
| `GET` | `/api/cr/notices` | CR | The notices this CR posted, newest first |
| `GET` | `/api/cr/drive-link` | CR | The Drive link of the CR's own batch; an empty `link` when there is none yet |
| `PUT` | `/api/cr/drive-link` | CR | `{link}` saves the Drive link of the CR's own batch; the title is always the batch code |
| `DELETE` | `/api/cr/drive-link` | CR | Removes the Drive link of the CR's own batch; `204` on success, `404` if there is none |
| `POST` | `/api/teacher/notices` | teacher | `multipart/form-data` with `title`, optional `description`, `batch`, optional `section`, optional `link` and optional `file` |
| `GET` | `/api/teacher/notices` | teacher | The notices this teacher posted, newest first |
| `DELETE` | `/api/teacher/notices/{id}` | teacher | Deletes one of this teacher's own notices; `204` on success, `404` otherwise |
| `GET` | `/api/students/notices` | student, CR | One list for every source, filtered with `source` (`all`, `admin`, `teacher`, `cr`) and `scope` (`my`, `other`) |
| `GET` | `/api/students/drive-links` | student, CR | Every batch that has a Drive link, sorted by batch, each with `is_mine` |
| `GET` | `/api/students/directory` | student, CR | Active students with the `batch`, `blood_group` and `search` filters; at most 200 rows |
| `GET` | `/api/teachers-info` | student, CR, teacher | Active teachers (`name`, `phone`, `email`) with the optional `search` filter |
| `GET` | `/api/messages/contacts` | any signed-in role | Up to 20 active people matched by `search`, narrowed with `role`; carries `photo_url` |
| `GET` | `/api/messages/conversations` | any signed-in role | Everything visible to the caller, newest activity first, with `title`, `can_post`, a preview and `unread_count` |
| `POST` | `/api/messages/direct` | any signed-in role | `{user_id}` opens or reuses the one-to-one conversation with that person |
| `POST` | `/api/messages/groups` | any signed-in role | `{scope, batch?, section?}` opens or reuses a group the caller's role may start |
| `GET` | `/api/messages/conversations/{id}/messages` | any signed-in role | Newest first with `before` and `limit`; marks the conversation read |
| `POST` | `/api/messages/conversations/{id}/messages` | any signed-in role | `multipart/form-data` with `body`, `link` and `file`; at least one is required |
| `GET` | `/api/messages/unread-count` | any signed-in role | How many unread messages the caller has across every conversation |
| `POST` | `/api/cr/polls` | CR | `{question, options, scope}`; 2 to 10 unique options, `scope` is `section` or `batch` |
| `POST` | `/api/cr/polls/{id}/close` | CR | Closes one of this CR's own polls; `404` otherwise, `409` if it is already closed |
| `GET` | `/api/cr/polls` | CR | The polls this CR made, with the counted results |
| `GET` | `/api/students/polls` | student, CR | Polls for the own batch and section with `options`, `total_votes` and `my_option_id` |
| `POST` | `/api/students/polls/{id}/vote` | student, CR | `{option_id}`; `409` when the poll is closed or the student already voted |
| `GET` | `/uploads/{name}` | public | Serves one older uploaded PDF inline; only the generated file names resolve, anything else is `404` |
| `GET` | `/uploads/files/{name}` | public | Serves one attachment with its real type and `nosniff`; images and PDFs inline, everything else as a download |
| `GET` | `/uploads/photos/{name}` | public | Serves one profile photo inline with `nosniff`; only the generated names resolve |

### Attachments

Every notice and every message takes **one optional `file`** field. The
extension and the content have to agree, so renaming a file does not help, and
only these types are accepted: `pdf`, `jpg`, `jpeg`, `png`, `webp`, `gif`,
`doc`, `docx`, `xls`, `xlsx`, `ppt`, `pptx`, `csv`, `txt` and `zip`. Anything
else answers `400` with `{"error":"This file type is not allowed."}`, and a file
over **10 MB** answers `413`.

The file is written under a random name in `UPLOAD_DIR/files` and the database
keeps the original name, the type and the size, which the API returns as
`file_url`, `file_name`, `file_mime` and `file_size`. `GET
/uploads/files/{name}` serves it with the right `Content-Type` and `nosniff`:
images and PDFs open inline, everything else downloads under its original name.

A `link` may be sent beside or instead of a file. It has to be a full
`http://` or `https://` address.

### Messaging

A conversation is either **direct** between exactly two people or a **group**
with a scope. Groups are reused, so asking twice for the same `batch`, `section`,
`all_students` or `all_teachers` group returns the one that already exists.

| Scope | Who sees it |
|---|---|
| `batch` | The active students and CRs of that batch |
| `section` | That batch and section |
| `all_students` | Every active student and CR |
| `all_teachers` | Every active teacher |

The creator always sees their own group. A new message notifies the other person
for a direct conversation, or the whole audience for a group, with the sender's
name in the title.

### Access rules

- Every `/api/...` route except `/api/health`, the `/api/auth/...` entry points and
  `GET /uploads/{name}` and `GET /uploads/photos/{name}` needs a `Bearer` token; a
  missing or stale token answers `401`.
- **Nobody ever sets a password for somebody else.** A student picks their own
  password while registering and a teacher picks theirs from the invitation link,
  so no endpoint accepts a password for another account.
- A token carries the **effective role**: `student`, `cr`, `teacher` or `admin`. A
  route refuses every other role with `403` and
  `{"error":"403 Access Denied / Unauthorized Access"}`.
- `/api/admin/...` is admin only, `/api/cr/...` is CR only, `/api/teacher/...` is
  teacher only, `/api/students/...` is student and CR, and `/api/notifications`,
  `/api/teachers-info` and `/api/lookups/...` are open to the signed-in panels.
- **Every role messages people.** A student, CR, teacher or admin can open a
  direct conversation with any other active account. Groups are limited by role:
  a teacher reaches a batch or one section, a CR reaches any batch or a section,
  the admin also reaches every student or every teacher, and a student only
  reaches their own batch. **Only the person who created a group posts in it**;
  everybody else replies privately, which opens a direct conversation with the
  creator.
- Batch and section always come from the caller's own student row, never from the
  request, and a CR can only close or delete what they created themselves.

Rate limited routes allow **10 requests per minute per IP address**; the
eleventh gets `429` with `{"error":"Too many attempts. Please try again in a
minute."}`.

### Request and response shapes

`POST /api/auth/register`

It is **`multipart/form-data`**, not JSON, because the profile photo is
required. The fields are `full_name`, `reg_no`, `batch`, `section`, `email`,
`phone`, `blood_group`, `date_of_birth`, `password`, `confirm_password`, the
file field `photo`, and optionally `email_proof` and `drop_course_ids` repeated
once per course.

`batch` must look like `CSE-23`, `section` is a single letter, `blood_group` is
one of `A+ A- B+ B- AB+ AB- O+ O-`, `phone` is exactly 11 digits matching
`^01[3-9][0-9]{8}$`, `date_of_birth` is `YYYY-MM-DD` and belongs to somebody
between 15 and 60 years old, `email` must end with `@` plus
`STUDENT_EMAIL_DOMAIN`, and every `drop_course_ids` entry must be one of the
courses from `GET /api/auth/drop-course-options`. A rejected field answers `400`
with a message naming it, for example `{"error":"Batch must look like
CSE-23."}`, and a rejected phone always answers `{"error":"Phone number must be
11 digits, for example 01712345678."}`.

`photo` must be a JPG, PNG or WEBP **by content** (`http.DetectContentType`, so
renaming a PDF does not help), at most **2 MB** (`413` otherwise) and at least
**200 by 200 pixels**. It is stored under a random name in
`UPLOAD_DIR/photos` and served at `GET /uploads/photos/{name}`. A failed
registration never leaves a photo behind.

With a valid `email_proof` for the same address the account is created as
`pending` with `email_verified_at` set. Without one it is created as
`unverified` and a 6-digit code is emailed; only the **SHA-256 hash** of that
code is stored and it lasts 15 minutes:

```json
{
  "message": "We sent a 6-digit code to your university email address. Enter it to finish your registration.",
  "email_hint": "2***@student.sust.edu",
  "expires_at": "2026-10-18T14:15:00+06:00"
}
```

`POST /api/auth/verify-email` takes `{"reg_no", "code"}` and moves the account
to `pending`. Five wrong codes clear it with `{"error":"Too many wrong codes.
Request a new code."}` and the student has to ask for a new one through
`POST /api/auth/resend-verification`, which keeps a 60 second cooldown.

If the registration number or the email still belongs to an **`unverified`**
account, registering again **replaces** it and deletes its photo. Any other
existing account answers `409`.

`POST /api/auth/google/verify-email` takes `{"credential"}` from Google
Sign-In, checks it with `google.golang.org/api/idtoken` against
`GOOGLE_CLIENT_ID`, and requires `email_verified`, a hosted domain equal to
`STUDENT_EMAIL_DOMAIN` and an address ending with it. Anything else answers
`400` with `{"error":"Use your @student.sust.edu Google account."}`. On success
it answers `{"email", "email_proof"}`, where `email_proof` is a JWT signed with
`JWT_SECRET`, carrying the purpose `email_proof`, and valid for 15 minutes.

`POST /api/auth/login` takes `{"identifier":"2023331079","password":"...",
"portal":"student"}` and answers:

```json
{
  "token": "<jwt>",
  "user": {"id": 1, "user_id": 1, "identifier": "2023331079", "role": "cr", "name": "Ratul Islam", "batch": "CSE-23", "section": "A", "photo_url": "/uploads/photos/<name>.png"}
}
```

`portal` is optional and is `student` or `teacher`. A student or CR on the
teacher portal answers `401` with `{"error":"Use the student login page."}`, and
a teacher on the student portal answers `{"error":"Use the teacher login
page."}`. `role` is the **effective** role (`student`, `cr`, `teacher`), and
`batch` and `section` appear only for students and CRs. Send the token on later
requests as `Authorization: Bearer <token>`.

Teachers are **invited**, never created with a password.
`POST /api/admin/teachers` takes `name`, `teacher_id`, `designation`, `email`
(ending with `@` plus `TEACHER_EMAIL_DOMAIN`), `phone`, `date_of_birth`,
`blood_group`, `office_room`, an optional `joining_date` and an optional
`photo`. The account is stored as `invited` with a random 32-byte invitation
token, hashed, valid for **72 hours**, and the teacher gets an email with
`FRONTEND_ORIGIN` + `/teacher/activate?token=<token>`. The response carries
`invite_link` only when no mailer is configured outside production.

`GET /api/auth/invite?token=` answers `{name, designation, masked_email}` and
never the birth date. `POST /api/auth/activate` takes `token`, `teacher_id`,
`date_of_birth`, `password`, `confirm_password` and an optional `photo`; the
teacher ID and birth date must match the invitation, five wrong answers kill it,
and success makes the account `active`.

The password reset runs in three steps and works for students, CRs, teachers
and the admin.

`POST /api/auth/forgot-password` takes `{"identifier", "date_of_birth"}` for
students, CRs and teachers and `{"identifier", "phone"}` for the admin. The
account must be **active**. A student, CR or teacher whose birth date does not
match answers `{"error":"The ID and date of birth do not match an active
account."}`; the admin phone is compared on digits only, so a leading `+88`
makes no difference. It emails a 6-digit code and answers:

```json
{
  "message": "We sent a 6-digit code to the email address registered with this account.",
  "email_hint": "n***@student.sust.edu",
  "expires_at": "2026-10-16T14:15:00+06:00"
}
```

A fourth field, `dev_code`, carries the code itself **only** when no mailer is
configured (`SMTP_HOST` is empty) and `APP_ENV` is not `production`, so the flow
can be tested without a mail server. Asking for a second code within 60 seconds
returns `429`.

`POST /api/auth/verify-reset-code` takes `{"identifier", "code"}` and answers
`{"reset_token": "<64 hexadecimal characters>"}`. Five wrong codes clear the
reset and the user has to ask for a new one.

`POST /api/auth/reset-password` takes `{"identifier", "reset_token",
"new_password", "confirm_password"}` and only works once the code has been
verified.

Only the **SHA-256 hash** of the code and of the token is stored, so the
database never holds anything that can be used to log in. Both last 10 minutes,
the token works once, and resetting a password invalidates every existing login
token.

### Admin people endpoints

`{id}` is always the **`users.id`** of the account, which every admin list
returns as `id`. No admin response ever carries `password_hash` or the password
reset fields.

`GET /api/admin/students` defaults to every status except `rejected`; pass
`status=active`, `pending`, `rejected`, `removed` or `all` to choose. `search`
matches the full name or the registration number and ignores case. `limit`
defaults to **50** and may not exceed **200**, and the answer looks like
`{"items": [...], "total": 128}` where `total` counts every match, not just
the current page. Each item carries `is_cr` and `status`.

CRs are **promoted**, not created. `POST /api/admin/crs/promote` takes
`{"user_id"}`, sets `is_cr` on an active student, raises `token_version` and
sends that student the notification `You are now the Class Representative of
<batch> section <section>`. It answers `404` when the id is not an active
student and `409` when they already are a CR.

`PATCH /api/admin/teachers/{id}` changes only the fields the administration
owns: `name`, `teacher_id`, `designation`, `email`, `date_of_birth`,
`blood_group`, `joining_date` and `office_room`. Anything else answers `400`.
`POST /api/admin/teachers/{id}/resend-invite` issues a new link and kills the
old one; it answers `409` once the teacher has activated the account.

Removing, deactivating or demoting an account also raises its
`token_version`, so **every token that account already holds stops working at
once** — a demoted CR has to log in again to get a student token.

### Courses, routines and notifications

`POST /api/admin/courses` takes `{"code": "cse233", "title": "Digital Logic
Design"}`. The code may be written as `CSE-233` or `CSE233` and is stored in
upper case.

`POST /api/admin/routines` takes `batch`, `section`, `course_id`, `teacher_id`
(the **`teachers.id`** from the teacher lookup), `day_of_week` (`0` is Sunday,
`6` is Saturday), `start_time` and `end_time` as `HH:MM`, `room` and `type`
(`class` or `lab`). `section` is a single letter or **`ALL`**, which means both
sections. The backend answers `400` unless the start is before the end and both
sides stay inside **08:00 to 17:00**, and unless the course exists and the
teacher exists and is still active.

A second entry for the **same room on the same weekday at an overlapping time**
answers `409` and names the booking it clashes with:

```json
{"error": "Room 1003 is already booked on Monday 10:30–12:45 by CSE-23 section A (CSE-233)."}
```

Two entries that only touch (one ends at 12:45, the next starts at 12:45) do
not clash. The check and the insert run in **one transaction** guarded by
`pg_advisory_xact_lock` on the room and weekday, so two admins saving at the
same moment cannot both win. Listing is sorted by batch, section, weekday and
start time, and a `section` filter also returns the `ALL` entries, because they
apply to that section too.

Notifications are either **direct** (one user id) or a **broadcast** scoped by
role, batch and section. A reader sees their own direct notifications plus the
broadcasts that match their role, batch and section; a broadcast without a
section reaches the whole batch, and a broadcast to role `student` also reaches
the CRs of that batch and section. `since` is an RFC 3339 timestamp, which the
frontend keeps in `localStorage` as the moment the bell was last opened. Admin
tokens get `403` on both notification routes.

### Class requests and room availability

Every `/api/cr/*` route needs role `cr`, and the CR's **batch and section are
read from their own student row**, never from the request body or the query, so
a CR can only act for their own batch and section. The create request body is
`event_type` (`class`, `exam` or `extra`), `course_id`, `teacher_id` (the
**`teachers.id`**), `event_date` as `YYYY-MM-DD`, `start_time`, `end_time` and
`room`; any other field is rejected with `400`.

The date must be **tomorrow or later** in `APP_TIMEZONE`, the start must be
before the end, both must stay inside **08:00 to 17:00**, the course must
exist, and the teacher must exist and still be active.

The check and the insert run in **one transaction** guarded by
`pg_advisory_xact_lock` on the room and the date, so the **first CR to book a
free room wins**. A dated booking (`pending` or `approved`) that overlaps in the
same room answers:

```json
{"error": "This room is not available at the selected time."}
```

A fixed routine entry of **another batch** in that room on that weekday answers
`409` naming it, for example `Room 1003 is reserved for CSE-22 section B
(CSE-211) every Monday 10:30–12:45.` The CR's **own batch's** fixed entries do
**not** block, because that is their own slot.

A new request is stored as `pending` with the CR as `created_by`, and the
chosen teacher gets a `class_request` notification naming the batch, section,
course, date, time and room.

`GET /api/cr/room-availability?date=2026-10-17&room=1003` combines the fixed
routine of that weekday with the `pending` and `approved` dated bookings of
that date, sorted by room and then start time. Each item carries `room`,
`start_time`, `end_time`, `batch`, `section`, `course_code`, `source` (`fixed`,
`class`, `exam` or `extra`), `status` (`fixed`, `pending` or `approved`) and
`label`, which is **`yours`** for the fixed entries of the CR's own batch and
**`unavailable`** for everything else.

`POST /api/cr/class-requests` also takes an optional `note` of at most 500
characters with a `note_audience` of `teacher`, `students` or `both`, which is
required as soon as a note is present. The teacher reads it in
`/api/teacher/requests`, `/api/teacher/schedule` and the upcoming classes when
the audience is `teacher` or `both`. Students read it in
`/api/students/upcoming-routine` and `/api/students/exams`, but only once the
event is **approved** and only when the audience is `students` or `both`. The CR
always sees their own note. A note never creates a message or a notification.

### Teacher decisions, cancellations and the auto-cancel rule

A teacher only ever sees and changes rows where `teacher_id` is their own
`teachers.id`; anything else answers `404`. Approving works only while the
request is still `pending` and its date is **tomorrow or later**: a decided
request answers `409 This request has already been decided.` and one whose date
has arrived answers `409 This request can no longer be approved.` An approval
notifies the CR who asked and broadcasts to the students of that batch and
section, where section `ALL` reaches the whole batch; a rejection notifies only
the CR.

`POST /api/teacher/classes/{id}/cancel` cancels an approved class dated today or
later and notifies the CR and the students.
`POST /api/teacher/routines/{id}/cancel` takes `{"date": "2026-10-20"}` and
cancels **one occurrence** of a fixed routine entry: the date must be today or
later and must fall on that entry's weekday (`400` otherwise), a second attempt
answers `409`, and the students of that batch and section plus every active CR
there are notified. Cancelled occurrences are kept in `routine_cancellations`,
so the routine itself stays untouched.

**Auto-cancel**: a request nobody decided before its date arrives is not left
hanging. A job runs once at startup and then every `AUTO_CANCEL_INTERVAL`
(default `5m`), sets every `pending` request whose date is today or earlier to
`auto_cancelled`, and tells the CR it was not approved in time. The job stops
with the server's graceful shutdown.

### What a student sees

Batch, section and the drop courses come from the caller's own student row.
`GET /api/students/upcoming-routine` defaults to **today** in `APP_TIMEZONE`
and answers `date`, `date_label` (`DD.MM.YY`), `weekday`, `batch` and `groups`:
one group per section of the caller's own batch, the lettered sections first
and `ALL` (`Both sections`) last, each item carrying `start_time`, `end_time`,
`course_code`, `course_title`, `kind` (`class`, `lab`, `exam` or `extra`),
`room`, `teacher_name`, `status`, which is `cancelled` for a cancelled class or
a cancelled routine occurrence and `scheduled` otherwise, and the `note` when
its audience allows it.

A student may carry **several** drop courses, kept in `student_drop_courses`.
When they do, the same answer carries one more group, keyed `drop_courses` and
marked `drop_course: true`, holding the fixed classes and the approved class and
exam events of those courses **in the other batches** on that date. Every item
of that group names its own `batch`.

`GET /api/students/today` answers `date`, `classes_today` (the scheduled items
of that day), `upcoming_exams`, `next_exam` (`date` and `course`, or `null`),
`unread_messages`, `open_polls` and `latest_notices`, which holds the three
newest notices in the same shape as the notices list.

Materials are **one Drive link per batch**. The class representative of a batch
owns that link through `/api/cr/drive-link`, its title is always their batch
code, and every student sees every batch's link through
`GET /api/students/drive-links`, where `is_mine` marks their own batch.

### Account status messages

Login checks the password **before** the account status, so a wrong password
never reveals whether an account exists or what state it is in. An `invited`
teacher has no password at all, so that one case is answered before the
password check.

| Case | Status | Body |
|---|---|---|
| Unknown ID, wrong password, or an admin on the normal login | `401` | `{"error":"Invalid ID or password."}` |
| A normal user on the admin login | `401` | `{"error":"Invalid ID or password."}` |
| Correct password, account still waiting for approval | `403` | `{"error":"Your registration is pending approval by the administration."}` |
| Correct password, registration rejected | `403` | `{"error":"Your registration was rejected. Please contact the department office."}` |
| Correct password, account removed | `403` | `{"error":"Your account has been removed. Please contact the department office."}` |
| Correct password, email not confirmed yet | `403` | `{"error":"Please verify your email first."}` |
| A teacher who has not used the invitation link | `403` | `{"error":"Your account is not activated yet. Use the link in your invitation email."}` |
| A student or CR on the teacher login page | `401` | `{"error":"Use the student login page."}` |
| A teacher on the student login page | `401` | `{"error":"Use the teacher login page."}` |
| Duplicate registration number or email | `409` | `{"error":"An account with this registration number or email already exists."}` |
| Wrong or expired reset token | `400` | `{"error":"Invalid or expired reset token."}` |
| ID and date of birth do not match an active account | `400` | `{"error":"The ID and date of birth do not match an active account."}` |
| Admin ID and phone do not match an active account | `400` | `{"error":"The ID and phone number do not match an active account."}` |
| Wrong or expired email confirmation code | `400` | `{"error":"Invalid or expired code."}` |
| Five wrong email confirmation codes | `400` | `{"error":"Too many wrong codes. Request a new code."}` |
| A dead or expired teacher invitation link | `400` | `{"error":"This invitation link is invalid or has expired."}` |
| Teacher ID or birth date wrong on an invitation | `400` | `{"error":"The teacher ID and date of birth do not match this invitation."}` |
| A phone number that is not 11 Bangladeshi digits | `400` | `{"error":"Phone number must be 11 digits, for example 01712345678."}` |
| A profile photo over 2 MB | `413` | `{"error":"The profile photo must be 2 MB or smaller."}` |
| A new reset code asked for within 60 seconds | `429` | `{"error":"Please wait a minute before requesting a new code."}` |
| Wrong or expired reset code | `400` | `{"error":"Invalid or expired code."}` |
| Five wrong reset codes | `400` | `{"error":"Too many wrong codes. Please request a new code."}` |
| Malformed JSON, unknown field, or a body over 1 MB | `400` | `{"error":"Invalid request body"}` |

Any unknown route returns `404` with `{"error":"Route not found"}`.

### How a request is checked

Every request passes through `Recoverer` → `Logger` → `CORS` → the router. A
protected route then runs `VerifyToken` and, where a role is required,
`RequireRole`:

| Situation | Status | Body |
|---|---|---|
| No `Authorization` header, or not `Bearer <token>` | `401` | `{"error":"Unauthorized"}` |
| Token is malformed, expired, signed with another key, or not HS256 | `401` | `{"error":"Unauthorized"}` |
| Token is valid but the user was deleted or is not `active` (`pending`, `rejected`, `removed`, `unverified`, `invited`) | `401` | `{"error":"Session expired. Please log in again."}` |
| Token is valid but the user logged out everywhere (`token_version` moved on) | `401` | `{"error":"Session expired. Please log in again."}` |
| Token is valid but the user's role changed (a CR was demoted) | `401` | `{"error":"Session expired. Please log in again."}` |
| Signed in, but the route needs another role | `403` | `{"error":"403 Access Denied / Unauthorized Access"}` |

The important part: **`VerifyToken` reads the user from the database on every
single request.** A token alone is never enough, so removing or rejecting an
account takes effect immediately instead of when the token expires.

The token carries the **effective role** (`student`, `cr`, `teacher`,
`admin`), because a CR is stored as a student row with `is_cr = true`. The
middleware recomputes that role from the database and rejects the token if it
no longer matches.

CORS allows `FRONTEND_ORIGIN` only. A preflight `OPTIONS` request answers
`204`, with the CORS headers present only for that one origin.

## Development Guidelines

House rules for this project. Read them before adding code, so both members
write the backend the same way.

### Tech decisions

These are settled. Do not swap them for something else without agreeing first.

**Backend**

- **Go 1.25+**, routing with the **standard library `net/http`** only
  (`http.ServeMux` with method patterns such as `"GET /api/health"`). No
  third-party router: no Gin, Echo, Chi or gorilla/mux.
- **PostgreSQL** through **`github.com/jackc/pgx/v5`** with a **`pgxpool`**
  connection pool, configured from the `DATABASE_URL` environment variable.
- **SQL migrations are embedded** in the binary (`embed.FS`) and run
  automatically at startup, so a teammate only has to create an empty database.
- Use **proper column types**, never `TEXT` for everything: `BOOLEAN` for flags
  such as `is_cr`, `DATE` for dates, `TIME` for class times, `TIMESTAMPTZ` for
  timestamps, and **foreign keys with explicit `ON DELETE` rules**.
- **Integration tests** read **`TEST_DATABASE_URL`** and must call `t.Skip`
  when it is not set, so `go test ./...` still passes on a machine without
  PostgreSQL.

**Security**

- Passwords hashed with **bcrypt**. Never stored or logged in plain text.
- **JWT** via **`github.com/golang-jwt/jwt/v5`**. The token carries the
  **effective role**: `student`, `cr`, `teacher` or `admin`. A CR is stored as
  a student row with `is_cr = true`, so the raw database role is *not* what
  goes into the token — otherwise CR-only routes would reject the CR.
- Middleware **`VerifyToken`** then **`RequireRole`** on **every** protected
  route. The user's **status is re-checked on every request**, so a removed or
  rejected account stops working immediately even if their token is still
  valid.
- The admin API lives under **`/api/admin/*`** and is reachable only with role
  `admin`. A student, CR or teacher token gets **403**; a missing or invalid
  token gets **401**.
- Never rely on hiding a frontend button for security. The backend decides.

**Files and date/time rules**

- Uploaded **PDFs are stored in `backend/uploads/`** and the **`file_path` is
  saved in the database**. `backend/uploads/` is gitignored.
- **All date and time logic uses `APP_TIMEZONE` (Asia/Dhaka)**, never the
  server's local clock. Use `config.Config.Location` or `config.Config.Now()`.
- **`backend/internal/config` imports `time/tzdata` (blank import) so
  Asia/Dhaka works on Windows. Never remove it.**
- Dates are **`YYYY-MM-DD`**, times are **`HH:MM` in 24-hour form**.
- **Class hours are 08:00–17:00**, and start time must be before end time.
  Enforced on the backend, with `min`/`max` on the frontend time inputs as a
  convenience only.

**Frontend**

- **React + Vite + React Router + Tailwind CSS v4**, in `frontend/`, written in
  plain JavaScript with JSX.
- **Every route and every link is built through `src/lib/paths.js`.** No path is
  typed as a string anywhere else, so serving the app under a
  `/:university/:department` prefix later only touches that file and the router.
- **Every request goes through `src/lib/api.js`**, which attaches the bearer
  token, turns `{"error"}` into an `ApiError` and sends the user back to the
  right login page on a `401`.
- **Relative API URLs only.** The dev server proxies `/api` and `/uploads` to
  the backend, so there is no base URL to configure.
- The signed-in session lives in `localStorage` behind `AuthProvider`, and every
  read and write is wrapped so a blocked storage never breaks the app.
- **Destructive actions use `useConfirm()`**, never `window.confirm`.

### Code conventions

- Backend code lives in **`backend/internal/<package>`**, one package per
  concern: `config`, `httpx`, `server`, `handler`, and later `db`, `auth`,
  `middleware` and so on. `backend/cmd/server/main.go` is the only entry point.
- **Handlers must use the `httpx` helpers** — `httpx.WriteJSON(w, status, data)`
  and `httpx.WriteError(w, status, message)`. Never write JSON by hand and
  never use `http.Error`.
- **Errors are always `{"error": "message"}`.** Nothing else.
- **`gofmt -l .`, `go vet ./...` and `go test ./...` must all pass before every
  commit.** The first two print nothing when they are happy.
- Prefer **table-driven tests** where there are several similar cases, and use
  `httptest` for handlers.
- **No comments in Go source code. Keep code self-explanatory with clear
  names.** The one exception is the required `//go:embed` directive in
  `internal/db/migrate.go`, which the compiler needs in order to embed the
  migration files.
- **No unnecessary blank lines.** No blank lines inside function bodies or
  inside struct, const, var and import blocks; exactly one blank line between
  top-level declarations.
- A handler reads the request and replies. Business rules belong in a service
  or store function, so they can be tested without HTTP.

### Git workflow

- **One branch per piece of work**, named `feature/<short-name>`, for example
  `feature/database-setup`. Branch off `main`.
- Each branch is split into **2–6 small, logical commits** — for example data
  layer, handlers and routes, tests, docs. **Every commit must build and pass
  tests on its own** and contain a meaningful change. No empty or padding
  commits.
- **Conventional Commit** messages: `feat:`, `fix:`, `docs:`, `chore:`,
  `test:`, `refactor:`. Subject in the imperative mood, no trailing period.
  Scope the backend ones, for example `feat(backend): add health endpoint`.
- Open a **Pull Request into `main`**, reviewed by the other member. Merge it
  with a **merge commit** so the whole branch stays visible as a group in the
  history.
- The member who owns the work commits it under **their own Git identity**.
  Check with `git config user.name` and `git config user.email` before the
  first commit.
- Update the matching **Status in [docs/ROADMAP.md](docs/ROADMAP.md)** to `done`
  as part of the same branch.

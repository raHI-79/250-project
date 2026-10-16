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
| B | TBD | TBD |

## Tech stack

| Layer | Technology |
|---|---|
| Backend | Go 1.25+, standard library `net/http` |
| Database | PostgreSQL 14+ via `pgx/v5` with a `pgxpool` connection pool |
| Authentication | bcrypt password hashing + JWT (from Set 3) |
| Frontend | React + Vite + React Router + Tailwind CSS (from Set 16) |
| File uploads | PDFs on disk in `backend/uploads/`, path saved in the database |

## Folder structure

```
250-project/
├── backend/                  Go module (the API server)
│   ├── cmd/server/           main.go — program entry point
│   ├── internal/
│   │   ├── config/           reads backend/.env and environment variables
│   │   ├── db/               pgxpool connection + embedded SQL migrations
│   │   │   └── migrations/   schema files, applied in filename order
│   │   ├── handler/          one file per group of endpoints
│   │   ├── httpx/            shared JSON response helpers
│   │   └── server/           router: URL -> handler
│   ├── .env.example          copy to .env and fill in your values
│   └── go.mod
├── frontend/                 React + Vite app (added in Set 16)
├── docs/
│   ├── SPEC.md               full requirements (do not edit)
│   └── ROADMAP.md            the 27 development sets and their status
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

4. **Edit `backend/.env`** and replace `CHANGE_ME` with your own postgres
   password in both `DATABASE_URL` and `TEST_DATABASE_URL`.

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

| Method | Path | Description |
|---|---|---|
| `GET` | `/api/health` | Returns `status`, `environment`, `database` (`ok` / `unavailable`) and the current time |

Any unknown route returns `404` with `{"error":"Route not found"}`.

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

**Frontend (from Set 16)**

- **React + Vite + React Router + Tailwind CSS**, in `frontend/`.

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

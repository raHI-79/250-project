# SUST CSE Class & Exam Coordination System

A database-driven web application that helps the SUST CSE department coordinate
classes, exams, notices, materials and communication between students, Class
Representatives (CRs), teachers and the department administration. Students see
their upcoming routine and exams, CRs request classes and exams that teachers
approve, teachers manage their daily schedule and messages, and the
administration manages users, courses and the fixed weekly routine from a
hidden admin panel. Class attendance is taken by Bluetooth from the Android app,
so a student can only be marked present inside the room, on their own
registered phone. Built as a course project for **CSE 250**.

The full requirements live in [docs/SPEC.md](docs/SPEC.md). The build order is
tracked in [docs/ROADMAP.md](docs/ROADMAP.md). To put the portal online for
free, follow [docs/DEPLOY.md](docs/DEPLOY.md). Every released version is listed
in [CHANGELOG.md](CHANGELOG.md).

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
| Android app | Capacitor 6, package `edu.sust.cse.portal`, with its own Bluetooth, location and biometric plugins |
| Spreadsheets | `.xlsx` attendance workbooks built on the server with `xuri/excelize` |

## Folder structure

```
250-project/
├── backend/                  Go module (the API server)
│   ├── cmd/seed/             go run ./cmd/seed -demo — the demo department
│   ├── cmd/server/           main.go — program entry point
│   ├── internal/
│   │   ├── attendance/       presence tokens, the distance rules, the workbook
│   │   ├── auth/             bcrypt hashing, JWT issuing and parsing, roles
│   │   ├── config/           reads backend/.env and environment variables
│   │   ├── db/               pgxpool connection + embedded SQL migrations
│   │   │   └── migrations/   schema files, applied in filename order
│   │   ├── dbtest/           shared helper for the database integration tests
│   │   ├── handler/          one file per group of endpoints
│   │   ├── httpx/            shared JSON response helpers
│   │   ├── jobs/             the scheduling job: item generation and the deadline
│   │   ├── middleware/       VerifyToken, RequireRole, Logger, Recoverer, CORS
│   │   ├── server/           router: URL -> handler
│   │   └── store/            database queries used by the API
│   ├── .env.example          copy to .env and fill in your values
│   └── go.mod
├── frontend/                 React + Vite app (the browser client)
│   ├── public/brand/         sust-logo.png, the source for every generated icon
│   ├── public/campus/        campus photos used by the banners and heroes
│   └── src/
│       ├── components/ui/    Button, Card, Field, Alert, Toast, Modal, Skeleton
│       ├── components/layout/ sidebar, top bar, bottom tabs, search sheet
│       ├── context/          the signed-in session
│       ├── lib/              paths, API client, navigation, helpers
│       └── pages/            one file per screen
├── Dockerfile                one image: the built pages and the Go server
├── render.yaml               the Render blueprint for the free web service
├── CHANGELOG.md              every released version
├── docs/
│   ├── ANDROID.md            the Capacitor app, notifications and attendance
│   ├── DEPLOY.md             putting the portal online for free
│   ├── SPEC.md               the final specification of the product
│   └── ROADMAP.md            the development steps and their status
├── .github/workflows/        the checks on every push and the nightly backup
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
   example `C:\Program Files\PostgreSQL\18\bin`.

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
   - leave `STUDENT_EMAIL_DOMAIN` alone unless your university uses another
     domain, and set `GOOGLE_CLIENT_ID` only if you want the Google Sign-In
     shortcut on the registration page. Teacher addresses are not tied to a
     domain.

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
| `TEACHER_REMINDER_TIMES` | `16:00,21:00` | When teachers are reminded about tomorrow's undecided classes |
| `TEACHER_MORNING_REMINDER` | `07:30` | When teachers are reminded about today's undecided classes |
| `SCHEDULE_DAYS_AHEAD` | `7` | How many days of dated schedule items the job keeps ahead of today |
| `UPLOAD_DIR` | `uploads` | Folder where the uploaded PDFs and profile photos are saved, created when it is missing |
| `MAX_UPLOAD_MB` | `10` | Largest accepted PDF in megabytes; a bigger file gets `413` |
| `STUDENT_EMAIL_DOMAIN` | `student.sust.edu` | Domain every student email address must end with |
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
| `VAPID_PUBLIC_KEY` | none — optional | Public key the browser subscribes to push with |
| `VAPID_PRIVATE_KEY` | none — optional | Private push key; keep it out of git and out of logs |
| `VAPID_SUBJECT` | none — optional | A `mailto:` address the push services can reach you at |
| `DEMO_PASSWORD` | none — optional | Password every account the demo seeder creates shares |
| `FILE_STORAGE` | `disk` | `disk` keeps uploads under `UPLOAD_DIR`, `database` keeps them in Postgres |
| `STATIC_DIR` | none — optional | When set, the server also serves the built frontend from this folder |
| `DB_MAX_CONNS` | `10` | Largest connection pool; use `5` on a free Neon database |
| `MAIL_PROVIDER` | `smtp` | `smtp`, `brevo` or `appsscript`, the last two send over HTTPS where SMTP ports are blocked |
| `BREVO_API_KEY` | none — optional | The Brevo API key, needed when `MAIL_PROVIDER=brevo` |
| `MAIL_FROM_NAME` | `SUST CSE Portal` | The name beside the sender address |
| `APPS_SCRIPT_URL` | none — optional | The `/exec` address of the Apps Script web app, needed when `MAIL_PROVIDER=appsscript` |
| `APPS_SCRIPT_SECRET` | none — optional | Shared secret the script checks on every request; never logged |
| `TURNSTILE_SITE_KEY` | none — optional | Cloudflare Turnstile site key; with either key empty the human check is skipped |
| `TURNSTILE_SECRET_KEY` | none — optional | Cloudflare Turnstile secret key, used server side only |
| `APP_VERSION` | `0.0.0` | Version reported by `/api/health` |
| `APP_COMMIT` | `RENDER_GIT_COMMIT` or `unknown` | Short commit reported by `/api/health` |
| `FIREBASE_SERVICE_ACCOUNT` | none — optional | The Firebase service account JSON, base64; empty turns the Android notifications off |
| `FCM_CHANNEL_ID` | `sust_cse` | The Android notification channel the messages land in |
| `GITHUB_RELEASES_REPO` | `raHI-79/250-project` | The repository `/download/android` and the in-app updater read releases from |
| `GITHUB_API_URL` | `https://api.github.com` | Override the GitHub API host; only the tests set it |

Never commit real values. `backend/.env` is gitignored; only
`backend/.env.example` is tracked.

### Loading the demo data

With the database migrated, one command fills the department with a believable
term:

```powershell
cd backend
go run ./cmd/seed -demo
```

It creates batches `CSE-21` to `CSE-24` with sections A and B, each on its own
running semester (`4-1`, `3-1`, `2-1` and `1-2`) with an older semester behind
it, courses with real looking codes and credits, five teachers with chosen User
IDs, students and one class representative per batch and section, a fixed
routine built on the class periods, schedule items in every state, a published
final routine, teacher exams, notices, polls, a group chat and messages.

- It refuses to run while `APP_ENV` is `production`.
- It is safe to run twice: a second run changes nothing.
- Every account it creates signs in with `DEMO_PASSWORD` from `backend/.env`,
  so set that before running it. The seeder prints the identifiers it created
  and never prints the password.

The identifiers follow a pattern, so you can sign in without reading the output:

| Role | Sign in with | Example |
|---|---|---|
| Teacher | the User ID on the teacher sign-in page | `mrahman`, `fsultana`, `nhasan` |
| Class representative | `s<year><section>1` | `s23a1` is the CR of CSE-23 section A |
| Student | `s<year><section><2..4>` | `s23a2`, `s23b3`, `s21a4` |
| Office | your own `ADMIN_IDENTIFIER` on the office portal | `admin` |

Teachers sign in on the teacher page, students and class representatives on the
student page, and the office on the hidden office portal.

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

Everyone starts on the public home page at `/`, which carries the campus hero,
the department numbers and a card for the student and the teacher portal.
Students and CRs sign in at `/student/login`, teachers at `/teacher/login`. The
department office signs in at `/office-portal`, a route that is never linked,
labelled or hinted at on any other page; the page also sets a
`noindex,nofollow` robots meta tag while it is open, and the route is kept out
of `robots.txt` on purpose so the file does not advertise it.

### The pages of the scheduling engine

Every route and every link is built in `src/lib/paths.js`, which also holds the
query-string helpers, so no page writes a URL by hand.

**The class representative** works on one page, `Create Class & Exam`, with four
tabs that the URL remembers (`?tab=`, `?date=`, `?item=`):

| Tab | What it does |
|---|---|
| Create | A class or an exam with the course, the section, the date, a From period and a To period, a room or an online link, and the syllabus and instructions of an exam. A refused rule comes back inline. |
| Room availability | Every managed room against the class periods, one table per date: a shaded break column, green free and red booked with the batch, course and status in the tooltip. Clicking a free period fills the create form. |
| Upcoming routine | The batch's items by date with their status, the teacher's message and a proposed slot, and only the actions the API allows on each one. |
| Final routine | Items grouped by date with `Needs time` and `Changed since publish`, inline editing of the time, place, link and note, checkboxes, a timetable preview and Publish All behind a confirm. |

**The teacher** has `Fixed weekly routine`, `Upcoming routine` and
`Create exam`. The upcoming page has two tabs: `Next day`, which opens with the
decision window banner — *Opens at 4:00 PM for tomorrow's classes* before it,
*Open until 11:00 PM · `<h m>` left* inside it, and
*Closed for tomorrow — undecided classes are marked No response* after it — and
offers Confirm, Cancel and Request reschedule per item plus Accept and Decline
on a request from the class representative. Those three buttons are disabled
outside the window for a **first** decision; a later change to a class the
teacher already confirmed stays allowed until it starts, and the class
representative is never limited. The second tab, `Final routine`, lists what the
batches published from today with a `By teacher` or `By CR` badge and the link
editor for an online class. An online class with no link yet reads
`Link coming soon` and offers `Add link` to the teacher and to the CR.

**A student** reads only the published routine. `Today` and `Upcoming routine`
show date chips, an `Online` badge with a `Join` button when a link exists,
cancelled classes struck through, and the notes. `Exams` lists every exam with
its syllabus and instructions, and either the time and the place or
`Time to be announced`. A class representative sees exactly the same pages.

### Messaging and group chats

The conversation list carries a WhatsApp-like search bar with the filter chips
`All · Teachers · CRs · Students · Office`. Typing shows the people who match
with their role and batch or designation; clicking one opens or reuses that
direct conversation. `?c=` opens a conversation straight from a link, which is
how a message notification lands.

A plain student may start a chat with teachers, class representatives and the
department office, never with another plain student: the attempt answers `403`
and the search never lists plain students for them. A CR may message students,
and teachers and the office may message anyone.

Any member may delete a whole chat, direct or group, **for themselves only**.
The messages up to that moment stop showing for them, the chat leaves their list
and their unread count, everybody else keeps the full history, and in a group
they stay a member. The chat comes back carrying only the newer messages as soon
as somebody writes again.

The sender may edit the body of their own message, or delete it, within two
minutes of sending; after that both answer `409`. An edited message shows
`Edited` next to its time and a deleted one reads
`This message was deleted`, with its body, link and file cleared and the file
removed from disk. The Edit and Delete buttons appear only while the two
minutes last.

Group chats have real members. Only a teacher or a class representative starts
one, with a name, a people picker and an `Add whole batch` picker; a CR may add
whole batches only for their own batch, a teacher any batch. The creator becomes
the first group admin. Every member posts, sees the member list and can leave.
Group admins add and remove members, hand the admin role out or take it back and
rename the group; a group always keeps at least one admin, and when the last one
leaves, the earliest member takes over. Nobody outside the member list can read
or post, and each message notifies every member except its sender.

### Semesters and the running semester

A batch studies eight semesters, stored as `1-1` to `4-2` and shown as
*1st year 1st semester* and so on. Every routine entry belongs to one of them,
and each batch has one running semester. The first routine entry the office
enters for a batch sets it; after that the **Running semester** control on the
admin Routines page moves the batch on.

Only the running semester becomes dated classes, fills the class
representative's course helper, offers the drop courses a student can pick in
their profile and is checked for room clashes — so the same room may hold a class in two semesters.
Older entries stay readable: the admin Routines page and the student
`Weekly routine` page both carry a semester picker and show sections A and B
side by side, as tabs on a phone.

`Courses` in the student and CR navigation reads the running semester of a
batch: a heading with the semester, a `<n> courses · <total> credits` summary
and one card per course with its code, title, credit, teachers (each linking to
their person page), the sections it runs for and the weekly class and lab counts.

### Polls

A class representative picks who votes — their own section, their whole batch
or every batch — and how many options each student may pick
(`Each student may choose up to N options`, 1 to the number of options). A vote
carries 1 to `max_choices` distinct options of that poll and is accepted once
per student; a second attempt answers `409`.

Votes are secret. Every response, for the student, the CR, the teacher and the
office alike, carries only the count per option, the number of voters and
`my_option_ids` for the caller. No endpoint and no store method returns who
voted for what, and tests cover both the shape of the responses and the store
rows.

Every notification is a button. `notificationTarget(link, role)` turns the
`link` the API sends into the right page for the reader, so a teacher and a
student following the same kind of notification each land on their own page.
Opening one marks it read, closes the panel and navigates; `Mark all as read`
clears the dot. The target pages honour `?date`, `?item`, `?tab` and
`?highlight`, scrolling the row into view and flashing it for a moment.

### Interface design

The interface is built as a phone app first: calm, card based and quick to
read — neutral near-white surfaces `#F7F8FA`, soft grey borders, ink-black
text and one refined deep blue `#1D4ED8` that carries every action. A warm
amber appears rarely, and the SUST logo is the only brand mark. Maroon is
gone from the whole product.

The tokens live in `frontend/src/index.css`. The palette is declared once as
`--c-*` custom properties and mapped into `@theme`, so every utility resolves
at runtime and the whole app changes with the theme; a page never writes a raw
colour:

| Token group | Values | Used for |
|---|---|---|
| `brand-50` … `brand-950` | deep blue, `brand-600` is `#1D4ED8` (6.6:1 on white) | the primary: links, filled buttons, selected states |
| `gold-50` … `gold-900` | warm amber, `gold-500` is `#F79009` | the rare accent: notes, the syllabus block, highlights |
| `success-*`, `warning-*`, `danger-*` | green, amber, red | the semantic states; `accent-*` still resolves to `danger-*` |
| `ink`, `ink-soft`, `muted`, `faint` | `#101828` to grey | text, in that order of weight |
| `page`, `surface`, `raised`, `wash` | `#F7F8FA`, white, the one elevated card level, the tint behind rows | the page and its panels |
| `line`, `line-strong` | `#E4E7EC` hairlines | rules and borders |
| `glass` | translucent surface | the top app bar and the bottom tab bar |
| `night`, `deep-*`, `sand` | fixed, never themed | the few surfaces that stay dark in both themes, such as a photo overlay |

#### Dark theme

`:root` holds the light values, `:root[data-theme="dark"]` the dark ones, and a
`prefers-color-scheme` block covers the system setting. `src/lib/theme.js`
resolves the choice before the first paint, and `My profile → Appearance` sets
Light, Dark or System. A filled brand surface pairs with `text-page`, so it is
deep blue with near-white text in the light theme and a light blue with
near-black text in the dark one. In the app `src/lib/shell.js` paints the
status bar and the navigation bar with the resolved surface colour and flips
their icons with it, so the system bars read as one surface with the app's own
top and bottom bars.

Two fonts are loaded in `frontend/index.html`: Source Serif 4 for large page
titles and the home greeting (the `font-display` and `font-numeral` utilities),
and Inter for every other piece of interface text.

Body text is 15-16 px, captions are 13 px (`text-xs`, `eyebrow`, `label-caps`),
section titles are 22 px (`text-section`) and large titles 30-34 px
(`text-title`, `text-title-lg`). Cards use a 16 px radius (`rounded-card`) and
sheets a 24 px one (`rounded-sheet`), spacing follows an 8 px grid, icons are
lucide outlines that fill when active, and anything tappable is at least 48 px
(`tap-target`).

Every page is built for 360 px upward and respects `env(safe-area-inset-*)`.
Below 1024 px the chrome is a translucent top app bar whose compact title
appears as the large title scrolls away, plus a translucent bottom tab bar with
five role tabs and unread badges; from 1024 px the same components sit in a
sidebar layout, and the office keeps its desktop layout. Modals are bottom
sheets with a drag handle on a phone and centred dialogs on a desktop, lists
pull to refresh, toasts sit above the tab bar, and transitions respect
`prefers-reduced-motion`.

The shared pieces are `TopBar`, `TabBar`, `Sheet`, `Segmented`, `Chip`,
`ListRow`, `CountBadge`, `Carousel`, `PullToRefresh`, `Skeleton`, `EmptyState`,
`Crest`, `PageHeader` (the large title that the top bar collapses into),
`DateBlock` (the calendar block on notices and exams), `TimelineItem` (the time
rail on every routine) and `IdentityCard` (the digital ID card, with the QR code
of the registration number or User ID).

### Browser push notifications

Students, CRs and teachers can receive notifications on a device even while the
site is closed; the office never does. Every stored notification queues a push
on a small background worker with a 24 hour TTL, and a subscription that answers
`404` or `410` is deleted.

Set three values in `backend/.env` to switch it on:

| Variable | What it holds |
|---|---|
| `VAPID_PUBLIC_KEY` | The public VAPID key the browser subscribes with |
| `VAPID_PRIVATE_KEY` | The private VAPID key; keep it out of git and out of logs |
| `VAPID_SUBJECT` | A `mailto:` address the push services can reach you at |

With any of them missing the feature stays off: `GET /api/push/public-key`
answers `enabled: false`, the frontend never asks for permission and nothing is
queued.

A push carries `{title, body, url, tag}`, goes out with `Urgency: high` and a
24 hour TTL, and its `url` is the real page for the recipient's role, so a
teacher and a student following the same notice each land on their own screen.
A broadcast is resolved to user ids with their effective roles and queued once
per distinct page. Every send is logged with the push service host and the
status code, and each subscribe logs how many subscriptions that account now
has — never a key, never a full endpoint.

The phone is asked for permission inside the sign-in click itself, so the
system dialog appears right after a successful sign-in. When the permission is
already `granted` the app subscribes silently instead, on sign-in and on every
app start, without showing anything. If the dialog could not be shown and the
permission is still `default`, the app asks once more on the next tap inside the
app shell and never again after a decision. There is no in-page banner, switch
or state: the permission dialog is the only thing that asks.
`POST /api/push/test` stays on the backend for debugging; it sends to every
subscription on the account and answers how many went out.

On every start the app registers `/sw.js` at scope `/`, and on start and on
each visibility change it re-syncs the subscription: it subscribes when none
exists, unsubscribes and subscribes again when the stored
`applicationServerKey` no longer matches the server key, and posts the
subscription every time. The server key is trimmed, read as base64url and
checked to be a 65 byte uncompressed point, so a key pasted with a stray
newline fails with a clear message instead of silently never subscribing. Every
failure is reported to `POST /api/push/client-log` with only an error name and
message, so a problem that happens on a phone shows up in the Render log.

`public/sw.js` waits instead of calling `skipWaiting`, so a new version raises
*A new version is ready · Reload*; it takes over only when the page sends it
`SKIP_WAITING`. It claims its clients on activation, always shows a notification
(title, body, icon, the monochrome badge, tag, `renotify`, vibration and
`data.url`) and, on click, focuses an open window and navigates it, or opens a
new one. Signing out removes that device's subscription **row** but keeps the
browser subscription, so signing back in re-registers at once.

The Go static server serves `/sw.js` as `application/javascript` with
`Cache-Control: no-cache` and `Service-Worker-Allowed: /`, and
`manifest.webmanifest` as `application/manifest+json` with `Cache-Control:
no-cache`. Each push failure is logged with the push service host and the
status code only — never a key and never a full endpoint. At startup the server
logs once whether the VAPID keys are configured.

Browsers only allow push on `https://`, with `http://localhost` as the single
exception — see *Showing the site on a phone over HTTPS* below.

### Getting the app, or installing the site

On Android there is one way in: the real app. `GET /download/android` is public
and redirects to the signed APK of the newest GitHub Release, so the link to
share is

```
https://<your host>/download/android
```

Every **Download the Android app** button goes through
`paths.downloadAndroid()` to that route, and on an Android browser one small
bottom banner offers it once; closing or using the banner hides it for thirty
days. Chrome no longer offers to install the website there: `index.html`
attaches `manifest.webmanifest` only on a desktop or an iPhone and refuses
`beforeinstallprompt` on Android, so nobody ends up with a stale web copy.

`frontend/public/manifest.webmanifest` still gives the installed app its name
(`SUST CSE Portal`), its short name (`SUST CSE`), the theme and background
colours, `start_url` `/`, `display: standalone` and the generated icons.
`index.html` carries a light and a dark `theme-color` and paints a branded
splash until the first screen is ready. When the app runs standalone the
browser-only parts of the public page are hidden and a compact sign-in launcher
takes their place.

- **iPhone** — open the site in Safari and choose `Share → Add to Home Screen`.
  Safari only delivers push to an installed site, so this step is required
  before notifications work there.
- **Desktop Chrome** — open the site and choose `Install app` from the address
  bar or the browser menu.
- **Android** — use `/download/android`, not the browser's install option.

### Releasing an update

1. Bump `version` in `frontend/package.json` (it sets `versionName` and the
   version the app reports), and `APP_VERSION` in `render.yaml`.
2. Merge the change into `main`.
3. Tag and push it:

   ```powershell
   git tag v2.2.0
   git push origin v2.2.0
   ```

`.github/workflows/android.yml` builds and signs the APK and attaches it to the
release as `sust-cse-portal-<version>.apk`. From there everything is automatic:
`/download/android` points at the new file within ten minutes, the app picks the
release up on its next start or resume and offers the update, and the server
announces it once to every device token. Put `[required]` anywhere in the
release notes to make the update unskippable.

#### Installing as an Android app

`frontend/public/.well-known/assetlinks.json` carries the statement PWABuilder
generates for the signed APK. Vite copies the folder into `dist/`, and the Go
server answers `/.well-known/assetlinks.json` as `application/json` without a
redirect, which is what Android checks to verify the app. Once it verifies, the
URL bar disappears inside the installed app and the app may show the
notifications the site asks for. If either breaks, open
`https://<your host>/.well-known/assetlinks.json` in a browser first: it has to
return the JSON, not the page shell.

#### Generating the icons

Everything in the manifest, `index.html` and the service worker comes from one
source file, `frontend/public/brand/sust-logo.png`:

```powershell
cd frontend
npm run icons
```

`scripts/icons.mjs` writes `icon-192.png`, `icon-512.png` (purpose `any`),
`icon-192-maskable.png`, `icon-512-maskable.png` (the crest on a white plate
over the primary blue, inside the safe area), `badge-96.png` (the monochrome notification
badge), `apple-touch-icon.png` and `favicon.png`. The generated files are
committed, so the script only has to run again when the logo changes.

### Campus photos and logos

The public hero, the sign-in pages and the crest read their images from
`frontend/public/`. Both folders are empty on purpose — drop the real files in
with exactly these names and nothing else has to change:

| File | Size | Used by |
|---|---|---|
| `campus/hero-oldGate.jpg`, `campus/hero-building.jpg`, `campus/hero-monument.jpg`, `campus/hero-entrance.jpg` | 1920 × 1080 | the sign-in slideshow; the home hero uses `hero-building.jpg` on its own |
| `brand/sust-logo.png` | 512 × 512 | every crest, the home hero, the ID card, and the source for `npm run icons` |

There is no CSE logo; the SUST logo is the only one the app uses. When a campus
photo is missing the hero falls back to its deep gradient, and when the logo
is missing the crest falls back to its `CSE` monogram, so the app still looks
finished without them.

### Showing the site on a phone over HTTPS

Browsers only allow push on `https://`, with `http://localhost` as the single
exception, so `http://192.168.x.x` will not do for a demo. The quickest way to
get a real certificate is a Cloudflare Tunnel:

```powershell
winget install --id Cloudflare.cloudflared
cloudflared tunnel --url http://localhost:5173
```

`cloudflared` prints a `https://<something>.trycloudflare.com` address. Then:

1. Put that address in `FRONTEND_ORIGIN` in `backend/.env` so CORS and the push
   links point at it, and restart the backend.
2. Run a second tunnel for the API, or run the frontend dev server with
   `--host` and proxy `/api` to the backend, whichever your setup already does.
3. Open the printed address on the phone and sign in.

The tunnel address changes every run, so update `FRONTEND_ORIGIN` each time.

### Testing push notifications

1. Make sure `VAPID_PUBLIC_KEY`, `VAPID_PRIVATE_KEY` and `VAPID_SUBJECT` are set
   in `backend/.env`, and that the site is on `https://` or `http://localhost`.
2. Sign in as a student, a CR or a teacher. The phone asks for permission in
   the same click; if it does not, tap anywhere once inside the app.
3. Allow notifications. `My profile → Notifications` now reads `Allowed`, and
   `Send test notification` reaches this device.
4. From another account, do something that notifies that person: post a notice
   for their batch, send them a message, publish a routine, or open a poll.
5. The notification arrives even with the tab closed. Clicking it focuses an
   open tab and navigates to the item, or opens a new window.

To check the cleanup path, remove the site's notification permission in the
browser and send another notification: the push service answers `410`, and the
worker deletes that subscription.

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

The attendance rules are table-driven and need no database at all:
`backend/internal/attendance` covers the Bluetooth token and the fallback code
over the current, previous and expired slices and a wrong key, the great-circle
distance, the accuracy and mock-provider refusals, a stale teacher position, the
percentage, the once-then-weekly warning, the semester change limit and the
contents of the Excel workbook. The phone bindings, the change requests, the
roster, one-phone-one-student, the absentees written on close and the manual
edits are covered against a real database in `backend/internal/store`, and
`backend/internal/server` checks the permissions of every attendance route and
that a student cannot mark from the website.

The frontend has its own three checks:

```powershell
cd frontend
npm run lint
npm test
npm run build
```

### Continuous integration

`.github/workflows/ci.yml` runs on every push to `main` and on every pull
request, in two jobs:

- **Backend** — a PostgreSQL service, then the `gofmt` check, `go vet ./...` and
  `go test ./...` with `TEST_DATABASE_URL` pointing at that service, so the
  integration tests really run.
- **Frontend** — `npm ci`, `npm run lint` and `npm run build`.

## API endpoints

Every route lives under `/api`. Errors always use the same shape:
`{"error": "message"}`.

| Method | Path | Auth | Description |
|---|---|---|---|
| `GET` | `/api/health` | public | `status`, `environment`, `version`, `commit`, `database` (`ok` / `unavailable`) and the current time |
| `POST` | `/api/human/verify` | public | `{token, standalone}` checks a Turnstile token with Cloudflare and returns a signed pass; `503` when the keys are missing |
| `GET` | `/download/android` | public, rate limited | `302` to the signed APK of the newest GitHub Release; a short HTML page when no build is published. The release lookup is cached ten minutes |
| `GET` | `/api/app/android-latest` | public, rate limited | `{version, version_code, apk_url, notes, required}` of the newest release; `required` is true when the notes carry `[required]` |
| `GET` | `/api/public/stats` | public | `students`, `teachers`, `courses` and `batches` of the home page, active accounts only, cached five minutes |
| `GET` | `/api/auth/config` | public | `student_email_domain` and `google_enabled` |
| `GET` | `/api/auth/sections` | public, rate limited | `?batch=CSE-23` returns the sections seen in that batch's routines and active students, out of `A` and `B` |
| `GET` | `/api/push/public-key` | public | `enabled` and the VAPID `public_key` when web push is set up |
| `POST` | `/api/auth/register` | public | `multipart/form-data` student registration with a required photo; `201` on success |
| `POST` | `/api/auth/verify-email` | public, rate limited | `{reg_no, code}` turns an `unverified` account into `pending` |
| `POST` | `/api/auth/resend-verification` | public, rate limited | `{reg_no}` sends a new 6-digit code; `429` inside the 60 second cooldown |
| `GET` | `/api/auth/drop-course-options` | public, rate limited | Up to 20 courses that appear in a fixed routine, matched by `search` |
| `POST` | `/api/auth/google/verify-email` | public, rate limited | `{credential}` from Google Sign-In; answers `{email, email_proof}` |
| `GET` | `/api/auth/invite` | public | `token=` answers `{name, designation, masked_email}` of a teacher invitation |
| `GET` | `/api/auth/user-id-available` | public, rate limited | `user_id=` answers `{available, reason}` for a User ID a teacher is about to choose |
| `POST` | `/api/auth/activate` | public, rate limited | `multipart/form-data`; a teacher picks their own User ID and password from the invitation |
| `POST` | `/api/auth/login` | public, rate limited | Student, CR and teacher login. Returns a token and the user |
| `POST` | `/api/auth/admin/login` | public, rate limited | Admin login only. Returns a token and the admin user |
| `POST` | `/api/auth/forgot-password` | public, rate limited | Emails a 6-digit reset code valid for 10 minutes |
| `POST` | `/api/auth/verify-reset-code` | public, rate limited | Exchanges that code for a reset token valid for 10 minutes |
| `POST` | `/api/auth/reset-password` | public | Sets a new password with that reset token |
| `POST` | `/api/auth/logout` | any signed-in role | Invalidates **every** token of the account |
| `GET` | `/api/me` | any signed-in role | The signed-in user: `id`, `user_id`, `identifier`, `role`, `photo_url` |
| `GET` | `/api/me/badges` | any signed-in role | How many items are new in each of `notices`, `materials`, `polls`, `exams`, `routine` and `courses` |
| `POST` | `/api/me/badges/{section}/seen` | any signed-in role | Clears the badge of one section; the section page calls it as it opens |
| `GET` | `/api/profile` | any signed-in role | The caller's own profile, shaped by their role |
| `PATCH` | `/api/profile` | any signed-in role | `multipart/form-data`; only the fields that role may change |
| `POST` | `/api/profile/password` | any signed-in role | `{current_password, new_password, confirm_password}`; logs every session out |
| `GET` | `/api/people/{user_id}` | any signed-in role | Public card of an active student, CR or teacher; `404` otherwise |
| `GET` | `/api/admin/registrations` | admin | Pending student registrations, oldest first |
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
| `POST` | `/api/admin/users/{id}/restore` | admin | `removed` becomes `active` and `rejected` becomes `pending`; `409` for any other status |
| `DELETE` | `/api/admin/users/{id}/permanent` | admin | `{confirm}` deletes a `removed` or `rejected` account for good; `400` on a wrong confirmation, `403` for an admin or the caller, `409` otherwise |
| `GET` | `/api/admin/audit` | admin | The latest 100 removals, restores and permanent deletions |
| `GET` | `/api/admin/stats` | admin | `total_students`, `total_crs`, `total_teachers`, `pending_approvals` |
| `POST` | `/api/admin/courses` | admin | Adds a course with `code`, `title` and a required `credit` of 0.5 to 6.0 in steps of 0.25; the code is stored in upper case, `409` on a duplicate |
| `GET` | `/api/admin/courses` | admin | Every course with its credit, sorted by code |
| `DELETE` | `/api/admin/courses/{id}` | admin | Deletes a course nothing uses; `409` naming the routine entries, classes, exams and drop course students that still hold it |
| `POST` | `/api/admin/routines` | admin | Adds a fixed routine entry; `400` on a broken rule, `409` on a room clash |
| `GET` | `/api/admin/routines` | admin | Routine entries with the `batch` and `section` filters |
| `DELETE` | `/api/admin/routines/{id}` | admin | Deletes one entry; `204` on success, `404` if it does not exist |
| `GET` | `/api/lookups/batches` | any signed-in role | Batches of the active students and the routine entries |
| `GET` | `/api/lookups/sections` | any signed-in role | Sections of one `batch` (required), without `ALL` |
| `GET` | `/api/lookups/courses` | any signed-in role | `id`, `code` and `title` of every course |
| `GET` | `/api/lookups/teachers` | any signed-in role | `id`, `user_id` and `name` of the active teachers |
| `GET` | `/api/lookups/periods` | any signed-in role | The class periods of the teaching day in order |
| `PATCH` | `/api/admin/routines/{id}` | admin | `{semester}` moves one routine entry to another semester |
| `GET` | `/api/admin/batches/running-semester` | admin | Every batch with the semester it is running now |
| `PUT` | `/api/admin/batches/{batch}/running-semester` | admin | `{semester}` sets what that batch runs now |
| `GET` | `/api/students/drop-course-options` | student, CR | Searchable running-semester courses of the **other** batches, for the drop course picker |
| `GET` | `/api/students/courses` | student, CR | `?batch=` (own batch by default): the running semester, the total credits and one entry per course |
| `GET` | `/api/admin/periods` | admin | The same list for the Class periods page |
| `POST` | `/api/admin/periods` | admin | `{label, start_time, end_time, is_break}`; `409` when it overlaps another period |
| `PATCH` | `/api/admin/periods/{id}` | admin | Changes any of those fields; `409` on an overlap, `404` when unknown |
| `DELETE` | `/api/admin/periods/{id}` | admin | Removes a period; `409` when it is the last one left |
| `POST` | `/api/admin/periods/reorder` | admin | `{ids}` holding every period exactly once, in the new order |
| `GET` | `/api/notifications` | student, CR, teacher | Newest first; `limit` defaults to 30 and may not exceed 100 |
| `GET` | `/api/notifications/unread-count` | student, CR, teacher | How many matching notifications arrived after `since` |
| `GET` | `/api/cr/routine` | CR | Fixed routine of the CR's own batch and section, including the `ALL` entries |
| `GET` | `/api/cr/courses` | CR | Running-semester courses of the CR's batch with their teacher and weekly `slots` |
| `GET` | `/api/cr/room-availability` | CR | Seven days from `from` (today by default): every managed room and date with its booked intervals |
| `GET` | `/api/cr/upcoming` | CR | The own batch's schedule items from `from`, each with its status, the teacher's message, any proposed slot and `allowed_actions` |
| `POST` | `/api/cr/items` | CR | Adds a class or an exam to the own batch; `201` with status `scheduled` and no approval needed |
| `PATCH` | `/api/cr/items/{id}` | CR | Changes an own-batch item as far as its origin and status allow |
| `DELETE` | `/api/cr/items/{id}` | CR | Deletes an item the CR created |
| `POST` | `/api/cr/items/{id}/request-reschedule` | CR | Proposes a new slot for a fixed item; the teacher accepts or declines |
| `POST` | `/api/cr/items/{id}/cancel` | CR | Cancels an own item, a `teacher_reschedule_request`, a `no_response` or a `reschedule_declined` item |
| `POST` | `/api/cr/items/{id}/final` | CR | Puts the item in the final routine |
| `DELETE` | `/api/cr/items/{id}/final` | CR | Takes the item out of the final routine |
| `GET` | `/api/cr/final` | CR | The final routine grouped by date with `needs_time` and `changed_since_publish` |
| `GET` | `/api/cr/final/preview` | CR | The same shape for the `ids` the CR is about to publish |
| `POST` | `/api/cr/final/publish` | CR | `{item_ids}` writes the snapshots students read and notifies the batch |
| `GET` | `/api/teacher/fixed-routine` | teacher | This teacher's weekly fixed routine |
| `GET` | `/api/teacher/upcoming/next-day` | teacher | Tomorrow's items with `decision_open` and the window bounds |
| `GET` | `/api/teacher/upcoming/final` | teacher | This teacher's decided, final and published items from today, with their origin |
| `POST` | `/api/teacher/items/{id}/confirm` | teacher | `{mode, room?, meeting_link?}`; the first decision only inside the window |
| `POST` | `/api/teacher/items/{id}/cancel` | teacher | `{reason?}` cancels the class and frees the room |
| `POST` | `/api/teacher/items/{id}/request-reschedule` | teacher | `{message}` asks the CR for another slot |
| `POST` | `/api/teacher/items/{id}/reschedule-request/accept` | teacher | Takes the CR's proposed slot; the rules are checked again |
| `POST` | `/api/teacher/items/{id}/reschedule-request/decline` | teacher | Keeps the original slot |
| `GET` | `/api/teacher/exams` | teacher | This teacher's own exams from today |
| `POST` | `/api/teacher/exams` | teacher | `{course_id, batch, section?, item_date, syllabus, instructions}`; no approval needed |
| `PATCH` | `/api/teacher/exams/{id}` | teacher | Changes the date, the syllabus or the instructions |
| `POST` | `/api/teacher/exams/{id}/cancel` | teacher | Cancels the exam |
| `DELETE` | `/api/teacher/exams/{id}` | teacher | Deletes the exam |
| `PATCH` | `/api/schedule/items/{id}/link` | item's teacher, batch CR | `{meeting_link}` updates the online link and the published copy |
| `GET` | `/api/students/today` | student, CR | Today's summary: counts, the next exam and the three newest notices |
| `GET` | `/api/students/upcoming-routine` | student, CR | The **published** plan for `date` (today by default), grouped by section plus `drop_courses` |
| `GET` | `/api/students/exams` | student, CR | Teacher exams at once and published CR exams, from today onward |
| `GET` | `/api/students/routine` | student, CR | Fixed routine browser with `batch`, `section`, `day` and `course` filters |
| `POST` | `/api/admin/notices` | admin | `multipart/form-data` with `title`, optional `description`, optional `batch`, optional `link` and optional `file`; an empty `batch` means every batch |
| `GET` | `/api/admin/notices` | admin | Every administration notice, newest first |
| `PATCH` | `/api/admin/notices/{id}` | admin | `{title, description?, link_url?}` rewrites **any** notice and stamps `edited_at` |
| `DELETE` | `/api/admin/notices/{id}` | admin | Deletes **any** notice and its attachment; `204` on success, `404` if it does not exist |
| `POST` | `/api/cr/notices` | CR | `multipart/form-data` with `title`, optional `description`, `target` (`own` or another batch code), optional `link` and optional `file` |
| `GET` | `/api/cr/notices` | CR | The notices this CR posted, newest first |
| `PATCH` | `/api/cr/notices/{id}` | CR | `{title, description?, link_url?}` rewrites one of this CR's own notices |
| `DELETE` | `/api/cr/notices/{id}` | CR | Deletes one of this CR's own notices; `204` on success, `404` otherwise |
| `GET` | `/api/cr/drive-link` | CR | The Drive link of the CR's own batch; an empty `link` when there is none yet |
| `PUT` | `/api/cr/drive-link` | CR | `{link}` saves the Drive link of the CR's own batch; the title is always the batch code |
| `DELETE` | `/api/cr/drive-link` | CR | Removes the Drive link of the CR's own batch; `204` on success, `404` if there is none |
| `POST` | `/api/teacher/notices` | teacher | `multipart/form-data` with `title`, optional `description`, `batch`, optional `section`, optional `link` and optional `file` |
| `GET` | `/api/teacher/notices` | teacher | The notices this teacher posted, newest first |
| `PATCH` | `/api/teacher/notices/{id}` | teacher | `{title, description?, link_url?}` rewrites one of this teacher's own notices |
| `DELETE` | `/api/teacher/notices/{id}` | teacher | Deletes one of this teacher's own notices; `204` on success, `404` otherwise |
| `GET` | `/api/students/notices` | student, CR | One list for every source, filtered with `source` (`all`, `admin`, `teacher`, `cr`) and `scope` (`my`, `other`) |
| `GET` | `/api/students/drive-links` | student, CR | Every batch that has a Drive link, sorted by batch, each with `is_mine` |
| `GET` | `/api/students/directory` | student, CR | Active students with the `batch`, `blood_group` and `search` filters; at most 200 rows |
| `GET` | `/api/teachers-info` | student, CR, teacher | Active teachers (`name`, `phone`, `email`) with the optional `search` filter |
| `GET` | `/api/messages/contacts` | any signed-in role | Up to 20 active people matched by `search`, narrowed with `role`; carries `photo_url` |
| `GET` | `/api/messages/conversations` | any signed-in role | Everything visible to the caller, newest activity first, with `title`, `can_post`, a preview and `unread_count` |
| `POST` | `/api/messages/direct` | any signed-in role | `{user_id}` opens or reuses the one-to-one conversation with that person |
| `POST` | `/api/messages/groups` | teacher, CR | `{title, user_ids[], batches[{batch, section?}]}` makes a group; the caller becomes its group admin, `403` for students and the office |
| `GET` | `/api/messages/conversations/{id}/members` | members | The member list with each one's group role and the time they joined |
| `POST` | `/api/messages/conversations/{id}/members` | group admins | `{user_ids, batches}` adds people; a CR may only add their own batch |
| `DELETE` | `/api/messages/conversations/{id}/members/{user_id}` | group admins | Removes a member; `409` on the last group admin |
| `POST` | `/api/messages/conversations/{id}/members/{user_id}/admin` | group admins | Makes that member a group admin |
| `DELETE` | `/api/messages/conversations/{id}/members/{user_id}/admin` | group admins | Takes the group admin role back; `409` on the last one |
| `PATCH` | `/api/messages/conversations/{id}` | group admins | `{title}` renames the group |
| `POST` | `/api/messages/conversations/{id}/leave` | members | Leaves the group; the earliest member becomes admin if the last admin leaves |
| `DELETE` | `/api/messages/conversations/{id}` | members | Deletes the chat for the caller only: the messages so far stop showing, the chat leaves their list, and membership is untouched |
| `PATCH` | `/api/messages/{id}` | the sender | `{body}` within two minutes of sending; sets `edited_at`, `409` after |
| `DELETE` | `/api/messages/{id}` | the sender | Soft-deletes within two minutes: body, link and file cleared, file removed from disk, `409` after |
| `GET` | `/api/messages/conversations/{id}/messages` | any signed-in role | Newest first with `before` and `limit`; marks the conversation read |
| `POST` | `/api/messages/conversations/{id}/messages` | any signed-in role | `multipart/form-data` with `body`, `link` and `file`; at least one is required |
| `GET` | `/api/messages/unread-count` | any signed-in role | How many unread messages the caller has across every conversation |
| `POST` | `/api/cr/polls` | CR | `{question, options, scope, max_choices}`; 2 to 10 unique options, `scope` is `section` or `batch`, `max_choices` is 1 to the number of options |
| `POST` | `/api/cr/polls/{id}/close` | CR | Closes one of this CR's own polls; `404` otherwise, `409` if it is already closed |
| `POST` | `/api/cr/polls/{id}/reopen` | CR | Reopens a closed poll of this CR; `409` if it is already open |
| `PATCH` | `/api/cr/polls/{id}` | CR | `{question, options, max_choices}` rewrites a poll that has **no votes**; `409` once anyone voted |
| `DELETE` | `/api/cr/polls/{id}` | CR | Deletes one of this CR's own polls at any time |
| `GET` | `/api/cr/polls` | CR | The polls this CR made, with the counted results |
| `GET` | `/api/students/polls` | student, CR | Polls for the own batch and section with `options`, `max_choices`, `total_voters` and `my_option_ids` |
| `POST` | `/api/students/polls/{id}/vote` | student, CR | `{option_ids}`; 1 to `max_choices` distinct options, `409` when the poll is closed or the student already voted |
| `POST` | `/api/push/subscribe` | student, CR, teacher | `{endpoint, keys:{p256dh, auth}, user_agent?}` stores this browser; `503` when the server has no VAPID keys |
| `DELETE` | `/api/push/subscribe` | student, CR, teacher | `{endpoint}` forgets this browser |
| `POST` | `/api/push/client-log` | student, CR, teacher | `{stage, name, message}` records why a device could not subscribe; rate limited per IP |
| `POST` | `/api/push/fcm` | student, CR, teacher | `{token, platform, device_id}` stores the Android device token of the app |
| `DELETE` | `/api/push/fcm` | student, CR, teacher | `{token}` forgets that device, used on sign-out |
| `GET` | `/api/teacher/attendance/options` | teacher | Today's classes to prefill from, every class of the running semester, and the open session if there is one |
| `POST` | `/api/teacher/attendance/sessions` | teacher | `{course_id, batch, section, schedule_item_id?, minutes}` opens attendance for 5 to 30 minutes and pushes every roster student; returns the session with its `token_key` and the current code. Starting again while one is open returns the running one |
| `GET` | `/api/teacher/attendance/sessions` | teacher | This teacher's sessions in `from`/`to`, newest first, each with its present count and roster size |
| `GET` | `/api/teacher/attendance/sessions/{id}` | teacher | One of this teacher's own sessions with every roster student, their status, time, distance, accuracy, method and note; `403` for another teacher's session |
| `POST` | `/api/teacher/attendance/sessions/{id}/location` | teacher | `{lat, lng, accuracy_m}`; the app's foreground service posts this every 15 seconds. `409` once the session is closed |
| `POST` | `/api/teacher/attendance/sessions/{id}/extend` | teacher | `{minutes}` between 5 and 30 |
| `POST` | `/api/teacher/attendance/sessions/{id}/close` | teacher | Closes it and marks every roster student without a record absent with method `system`; `409` if already closed |
| `PATCH` | `/api/teacher/attendance/sessions/{id}/students/{student_id}` | teacher | `{status, note}` sets Present or Absent by hand, audited with who and when |
| `GET` | `/api/teacher/attendance/classes` | teacher | Each course-batch-section with its session count, average attendance and how many students are below 50 % |
| `GET` | `/api/teacher/attendance/export` | teacher | An `.xlsx` workbook; `course_id`, `batch` and `section` narrow it to one class, no filter gives every class plus a Summary sheet. `404` when the range is empty |
| `GET` | `/api/teacher/phone-requests` | teacher | Pending phone requests of the students in the batches this teacher teaches this semester |
| `POST` | `/api/teacher/phone-requests/{id}/approve` | teacher | `{note}` rebinds the phone and tells the student; `400` when the note is missing beyond two changes a semester, `403` for another batch's student, `409` once another teacher answered |
| `POST` | `/api/teacher/phone-requests/{id}/reject` | teacher | `{note}` refuses it and tells the student |
| `GET` | `/api/students/attendance/open` | student, CR | `?device_id=` answers the open session of their courses, the state of their registered phone, the service UUID to scan for, and on the website the message to use the app |
| `POST` | `/api/students/attendance/mark` | student, CR, app only | `{session_id, token or code, lat, lng, accuracy_m, is_mock, device_id}`; `403` from the website, from another phone, on a wrong token, on a mock position, on a weak signal, beyond 30 metres or with a stale teacher position, `409` when already marked or the phone already marked somebody |
| `GET` | `/api/students/attendance/mine` | student, CR | Every course with its held, present and percentage, and every class with its status, method, time and note |
| `POST` | `/api/students/attendance/phone` | student, CR, app only | `{device_id}` registers this phone when the student has none yet |
| `POST` | `/api/students/attendance/phone-request` | student, CR, app only | `{device_id, reason}` asks the batch's teachers for a new phone; `409` when one is pending, when it is already the registered phone, or when another account holds it |
| `GET` | `/api/admin/phone-changes` | admin | Read-only history of every phone request and its answer |
| `GET` | `/api/admin/attendance` | admin | Every course of every batch, or of one `batch`, with its sessions, average and count below 50 % |
| `GET` | `/api/admin/attendance/export` | admin | The same `.xlsx` workbook for one batch or for every batch, with a Summary sheet |
| `GET` | `/uploads/{name}` | public | Serves one older uploaded PDF inline; only the generated file names resolve, anything else is `404` |
| `GET` | `/uploads/files/{name}` | public | Serves one attachment with its real type and `nosniff`; images and PDFs inline, everything else as a download |
| `GET` | `/uploads/photos/{name}` | public | Serves one profile photo inline with `nosniff`; only the generated names resolve |

### Attendance

Attendance is the one feature tied to a single device. A teacher opens a session
for one class of the running semester for 5 to 30 minutes; while it is open the
teacher's Android app advertises a Bluetooth Low Energy token that changes every
10 seconds and keeps posting the teacher's GPS position every 15 seconds, both
from a foreground service, so a locked screen does not stop attendance.

A student's mark is accepted only when the phone heard the current or previous
token, sits within **30 metres** of the teacher's latest position with an
accuracy of **50 metres or better**, reports a position that is not from a mock
provider, passes the phone's own fingerprint, face or screen-lock check, and is
the phone that student registered. The teacher's position has to be under a
minute old. A phone with no Bluetooth, or that hears nothing within ten seconds,
types the rotating 4-digit code the teacher shows instead; the location rules do
not change.

The database does the rest of the work: a unique index per session and student,
and another per session and device for the `app` and `code` methods, so one
phone marks one student and a student marks once, however many requests arrive
at the same moment. Closing the session turns every roster student without a
record into an absence.

The pure rules - the token and code slices, the great-circle distance, the
accuracy and freshness limits, the percentage and the warning cadence - live in
`backend/internal/attendance` with no database behind them, which is also where
the `.xlsx` workbook is built. See
[docs/ANDROID.md](docs/ANDROID.md#how-attendance-works-in-the-app) for the
permissions and the GPS accuracy note, and chapter 12 of
[docs/SPEC.md](docs/SPEC.md) for the full rules.

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

It is **`multipart/form-data`**, not JSON, because of the optional profile
photo. The fields are `full_name`, `reg_no`, `batch`, `section`, `email`,
`phone`, `blood_group`, `date_of_birth`, `password`, `confirm_password`, and
optionally the file field `photo` and `email_proof`. The photo is **optional**:
without one the account shows the student's initials until a photo is added in
**My profile → Change photo**. Registration does not ask about drop courses; a
`drop_course_ids` field is ignored.

`batch` must look like `CSE-23`, `section` is a single letter, `blood_group` is
one of `A+ A- B+ B- AB+ AB- O+ O-`, `phone` is exactly 11 digits matching
`^01[3-9][0-9]{8}$`, `date_of_birth` is `YYYY-MM-DD` and belongs to somebody
between 15 and 60 years old, `email` must end with `@` plus
`STUDENT_EMAIL_DOMAIN`. A rejected field answers `400`
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

#### The two ways to confirm an email address

Registration keeps **both** paths open and the student picks one on the contact
step. The **University email** field comes first with the `@student.sust.edu`
hint, then an `or` divider, then **Continue with Google** below it.

- Typing the address by hand creates the account as `unverified` and the
  6-digit email code step stays required.
- Signing in with Google fills the field, locks it behind a **Verified with
  Google** badge and sends `email_proof` with the registration, so the account
  is created as `pending` and the code step is skipped.
- Clearing a Google-verified address removes the badge and the proof, which puts
  the student back on the code path.

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

Teachers are **invited**, never created with a password, and the administration
never picks their login identifier. `POST /api/admin/teachers` takes `name`,
`designation`, `email` (any valid address, unique across the teachers), `phone`,
`date_of_birth`, `blood_group`, `office_room`, an optional `joining_date` and an
optional `photo`. The account is stored as `invited` with **no identifier at
all** and a random 32-byte invitation token, hashed, valid for **72 hours**, and
the teacher gets an email with `FRONTEND_ORIGIN` +
`/teacher/activate?token=<token>`. The response carries `invite_link` only when
no mailer is configured outside production.

`GET /api/auth/invite?token=` answers `{name, designation, masked_email}` and
never the birth date. `POST /api/auth/activate` takes `token`,
`date_of_birth`, `user_id`, `password`, `confirm_password` and an optional
`photo`. The birth date must match the invitation and five wrong answers kill
the link. The `user_id` is the login identifier the teacher chooses for
themselves and must be **4 to 20 characters**, start with a letter, use only
letters, digits, dots, underscores and hyphens, and may not be digits only or
one of the reserved words `admin`, `office`, `root`, `support`, `system`,
`teacher` and `student`. It is compared without case against every
`users.identifier`, which a unique index on `lower(identifier)` enforces, so a
taken one answers `409` with `{"error":"This User ID is already taken."}`.

`GET /api/auth/user-id-available?user_id=` answers
`{"available": true}` or `{"available": false, "reason": "..."}` with the rule
that failed, so the activation page can check while the teacher types.

On success the account becomes `active`, the chosen ID is stored as both
`users.identifier` and `teachers.teacher_id`, the answer is
`{user_id, name, message}`, and a confirmation email carries the User ID. The
password is never emailed and never returned.

A teacher signs in and resets their password with **either** their User ID or
their email address, compared without case, so
`POST /api/auth/login` and `POST /api/auth/forgot-password` both accept either
one in `identifier`. A student or CR still signs in with their registration
number.

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

`GET /api/admin/students`, `GET /api/admin/crs` and `GET /api/admin/teachers`
all take the same `status` filter and default to every status except
`rejected`; pass `status=active`, `pending`, `invited`, `rejected`, `removed`
or `all` to choose. `search` matches the full name and the registration number
for students and CRs, and the name and the User ID for teachers, ignoring case.
`GET /api/admin/students` also takes `limit`, which defaults to **50** and may
not exceed **200**, and answers `{"items": [...], "total": 128}` where `total`
counts every match, not just the current page. Each student item carries
`is_cr` and `status`; each teacher item carries `identifier`, which is empty
until the teacher activates the account.

CRs are **promoted**, not created. `POST /api/admin/crs/promote` takes
`{"user_id"}`, sets `is_cr` on an active student, raises `token_version` and
sends that student the notification `You are now the Class Representative of
<batch> section <section>`. It answers `404` when the id is not an active
student and `409` when they already are a CR.

`PATCH /api/admin/teachers/{id}` changes only the fields the administration
owns: `name`, `designation`, `email`, `date_of_birth`, `blood_group`,
`joining_date` and `office_room`. The User ID belongs to the teacher, so it is
not an admin field and sending it answers `400`.
`POST /api/admin/teachers/{id}/resend-invite` issues a new link and kills the
old one; it answers `409` once the teacher has activated the account.

Removing, deactivating or demoting an account also raises its
`token_version`, so **every token that account already holds stops working at
once** — a demoted CR has to log in again to get a student token.

### Record retention

A removed or rejected account is kept so the administration can change its
mind. `POST /api/admin/users/{id}/restore` turns `removed` back into `active`
and raises `token_version`, and turns `rejected` back into `pending`. Any other
status answers `409`.

`DELETE /api/admin/users/{id}/permanent` takes `{"confirm": "<identifier or
email>"}` and is the only way to erase an account. It refuses anything that is
not `removed` or `rejected` with `409`, refuses an admin account or the caller
themselves with `403`, and refuses a confirmation that does not match the
account's User ID or email address with `400`. In one transaction it deletes
the user row, the student or teacher profile, the dropped-course rows, the poll
votes, the conversation memberships and the conversation reads, and then
deletes the profile photo from disk.

**What other people rely on stays.** Notices, messages, class requests, polls
and routines keep their row; their author column becomes `NULL` through
`ON DELETE SET NULL` and every API then reports the author as `Deleted user`
with no photo. A routine or class event that named the deleted teacher keeps
its slot and shows `Deleted user` as the teacher.

Every removal, restore and permanent deletion writes a row into `admin_audit`
(`id`, `admin_id`, `action`, `target_summary`, `created_at`), which
`GET /api/admin/audit` returns newest first, at most 100 rows.

### Courses, routines and notifications

#### Course codes

Codes differ from department to department, and the letter prefix is often
missing: `CSE 2243 7637D`, `2243 7637D` and `CSE 1111` are all real. One rule
holds everywhere: **3 to 25 characters, only letters, digits, spaces and
hyphens, and at least one digit.**

Every code is **normalised before it is saved and before it is searched**:
raised to upper case, trimmed, hyphens turned into spaces and repeated spaces
collapsed. So `cse-2243   7637d` is stored as `CSE 2243 7637D`, and a check
constraint keeps the shape. Migration `0015_course_codes.sql` normalises the
codes that already exist and **stops with a clear error** when two of them would
collide, so nothing is merged silently.

`GET /api/admin/courses?search=` and
`GET /api/auth/drop-course-options?search=` match the normalised code **ignoring
spaces** or the title, so `cse22437637d`, `CSE-2243-7637D` and `cse 2243` all
find `CSE 2243 7637D`.

`POST /api/admin/courses` takes `{"code": "cse 2243 7637d", "title":
"Compilers"}` and answers the stored code.

`POST /api/admin/routines` takes `batch`, `section`, `course_id`, `teacher_id`
(the **`teachers.id`** from the teacher lookup), `semester` (`1-1` to `4-2`),
`day_of_week` (`0` is Sunday, `6` is Saturday), `start_time` and `end_time` as
`HH:MM`, `room` and `type` (`class` or `lab`). `section` is **`A`**, **`B`** or
**`ALL`**, which means both sections. The backend answers `400` unless the start is before the end, the
times sit inside the class periods and miss every break, and unless the course
exists and the teacher exists and is still active.

A second entry for the **same room on the same weekday at an overlapping time in
the same semester** answers `409` and names the booking it clashes with:

```json
{"error": "Room 1003 is already booked on Monday 10:30–12:45 by CSE-23 section A (CSE 233)."}
```

Two entries that only touch (one ends at 12:45, the next starts at 12:45) do
not clash. The check and the insert run in **one transaction** guarded by
`pg_advisory_xact_lock` on the room and weekday, so two admins saving at the
same moment cannot both win. Listing is sorted by batch, section, weekday and
start time, and a `section` filter also returns the `ALL` entries, because they
apply to that section too.

#### Notification links

Every notification carries a **link** so the frontend can send the reader to the
page it is about. `GET /api/notifications` answers each item with
`link: {kind, ref?, date?}`:

| Kind | Points at | Carries |
|---|---|---|
| `teacher_next_day` | The teacher's next-day decisions | — |
| `teacher_final` | The teacher's final routine | — |
| `cr_upcoming` | The CR's upcoming list | `ref`, the item |
| `cr_final` | The CR's final routine | — |
| `routine` | The student routine | `date` |
| `exam` | The student exams | `ref`, the item |
| `notice` | The notices | `ref`, the notice |
| `message` | The conversation | `ref`, the conversation |
| `poll` | The polls | `ref`, the poll |
| `cr_role` | The reader's own profile | — |

Every place that creates a notification sets one, and `notify.PageURL` turns a
link into a full address from `FRONTEND_ORIGIN`, which the push notifications of
a later step will reuse.

Notifications are either **direct** (one user id) or a **broadcast** scoped by
role, batch and section. A reader sees their own direct notifications plus the
broadcasts that match their role, batch and section; a broadcast without a
section reaches the whole batch, and a broadcast to role `student` also reaches
the CRs of that batch and section. `since` is an RFC 3339 timestamp, which the
frontend keeps in `localStorage` as the moment the bell was last opened. Admin
tokens get `403` on both notification routes.

### The scheduling engine

Every fixed routine slot becomes a **dated schedule item**. A job runs once at
startup and then every five minutes and creates the missing items for today up
to `SCHEDULE_DAYS_AHEAD`, each one `awaiting_teacher` in the routine's own room.
The job is idempotent: a unique index on `(routine_id, item_date)` means a
second pass creates nothing.

**The teacher decides on the next day.** The teacher confirms, cancels or asks
for a reschedule of an item at any hour until the class starts, on
`awaiting_teacher` and `no_response` items alike. Reminders go out at
`TEACHER_REMINDER_TIMES` for tomorrow and at `TEACHER_MORNING_REMINDER` for
today. After the last reminder slot anything still `awaiting_teacher` for
tomorrow becomes `no_response`, which is only a signal to the class
representative: the room stays booked, the CR may decide, and every active CR
of that batch is notified.

**The CR owns the final routine.** CR-created classes and exams and
teacher-created exams need no approval at all. Nothing reaches students until
the CR adds items to the final routine and presses publish. A cancellation of
an item that is already published reaches the students at once.

These are the statuses an item moves through:

| Status | Meaning |
|---|---|
| `awaiting_teacher` | A generated fixed item the teacher has not decided on yet |
| `no_response` | The window closed with no decision; the room stays booked |
| `confirmed` | The teacher takes the slot |
| `teacher_reschedule_request` | The teacher asked for another slot; the CR sets it |
| `cr_reschedule_request` | The CR proposed a slot; the teacher accepts or declines |
| `reschedule_declined` | The teacher declined the proposal, so the original slot stands |
| `rescheduled` | A proposal was accepted and the item moved |
| `cancelled` | Cancelled by either side; the room is free again |
| `scheduled` | A CR item or a teacher exam, which needs no approval |

An `origin` of `fixed`, `cr` or `teacher_exam` says where the item came from.
Only a teacher exam may arrive without a time: the CR sets it before publishing.

### Room and batch rules

A **managed room** is any room that appears in a fixed routine entry. For
`mode` `room` with a managed room, an overlap with another active item in the
same room on the same date answers `409` naming the other booking:

```json
{"error": "Room 1003 is already booked for CSE-22 section B from 10:30 to 12:00."}
```

Rooms nobody has a routine in and online classes skip that check, so a CR can
book the library hall or go online without fighting the routine.

The **batch rule always applies**: two active items of the same batch may never
overlap in time when their sections overlap, which means the same letter or
either side being `ALL`:

```json
{"error": "CSE-23 section A already has CSE 2211 at 10:00."}
```

An item is *active* unless it is `cancelled`, so cancelling frees both the room
and the batch slot, and an accepted reschedule frees the old one. Times run from
the first class period to the last, never cross a break, and start before they
end. `meeting_link` is optional for both modes but has to be a full `http://` or
`https://` address when it is given.

Both checks and the write run in **one transaction** behind
`pg_advisory_xact_lock` on the batch and date and, for a managed room, on the
room and date, so the first writer of a free slot wins.

### Publishing

`POST /api/cr/final/publish` takes `{"item_ids": [...]}`. Every chosen item has
to be in the final routine and carry a date, a time and either a room or a
meeting link, otherwise the answer is `400` naming the item. The publication
writes one `publications` row and upserts a **snapshot** per item into
`published_items`: the date, times, course code and title, teacher name, kind,
mode, room, link, note, syllabus, instructions and status as they are at that
moment.

Students read **only those snapshots**, so a later edit stays invisible until
the CR publishes again. `GET /api/cr/final` marks such an item with
`changed_since_publish` so the CR can see what is waiting. A publication sends
one `Routine published` notification to the batch plus one per item whose date,
time, room, link or status changed since the previous snapshot.

Cancelling a published item updates the snapshot's status straight away and
notifies the students, because a class that is not happening cannot wait for the
next publication. The same is true for `PATCH /api/schedule/items/{id}/link`,
which the item's teacher or the batch CR may call.

### What the teacher sees

A teacher only ever sees and changes items whose `teacher_id` is their own
`teachers.id`; anything else answers `404`.
`GET /api/teacher/upcoming/next-day` carries `decision_open`, `window_start` and
`window_end` beside tomorrow's items, each with the CR's message and proposed
slot when there is one. Confirming takes `mode` with a `room` or a
`meeting_link`. Every teacher action notifies the CRs of that batch, and an
action on a published item also notifies the students.

A teacher exam needs only `{course_id, batch, section?, item_date, syllabus,
instructions}`. It is `scheduled` at once, students read its date, syllabus and
instructions immediately, and the job moves it into the final routine the day
before so the CR sets the time and the room.

### What a student sees

Batch, section and the drop courses come from the caller's own student row —
drop courses are chosen in **My profile → My drop courses**, saved through
`PATCH /api/profile` — and
a student reads **only published items**.
`GET /api/students/upcoming-routine` defaults to **today** in `APP_TIMEZONE`
and answers `date`, `date_label` (`DD.MM.YY`), `weekday`, `batch` and `groups`:
one group per section of the caller's own batch, the lettered sections first
and `ALL` (`Both sections`) last. Each item carries `start_time`, `end_time`,
`course_code`, `course_title`, `kind` (`class`, `lab` or `exam`), `mode`,
`room` or `meeting_link` for an online class, `teacher_name`, the item `status`
and the `note`. Nothing appears before the CR publishes it.

A student may carry **any number** of drop courses, kept in
`student_drop_courses` and picked in **My profile → My drop courses**: a
searchable multi-select over the running-semester courses of the other batches,
shown as chips, saved in one `PATCH /api/profile` with `drop_course_ids`, and
removed one at a time behind a confirm.

When a student carries drop courses, the day answer carries one more group,
keyed `drop_courses`, labelled **Drop courses** and marked `drop_course: true`,
holding the published classes and labs of those courses **in the other batches**
on that date. Every item of that group names its own `batch`. `Today`, the
upcoming routine and the exam list all include them, their section badges and
notifications count them, and removing a drop course makes all of it disappear
at once.

`GET /api/students/exams` answers two kinds of exam. A **teacher exam** shows up
the moment the teacher creates it, with its date, syllabus and instructions, and
gains its time and room only once the CR publishes it. A **CR exam** appears
once it is published. `published` says which of the two a row is.

`GET /api/students/today` answers `date`, `classes_today` (the published items
of that day that are not cancelled), `upcoming_exams`, `next_exam` (`date` and
`course`, or `null`), `unread_messages`, `open_polls` and `latest_notices`,
which holds the three newest notices in the same shape as the notices list.

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
| Birth date wrong on an invitation | `400` | `{"error":"The date of birth does not match this invitation."}` |
| A User ID somebody already has | `409` | `{"error":"This User ID is already taken."}` |
| Restoring an account that is not removed or rejected | `409` | `{"error":"Only a removed or rejected account can be restored."}` |
| Deleting an account that is not removed or rejected | `409` | `{"error":"Only a removed or rejected account can be deleted permanently."}` |
| Deleting an admin account or your own | `403` | `{"error":"This account cannot be deleted."}` |
| A confirmation that does not match the account | `400` | `{"error":"Type the User ID or the email address of the account exactly to confirm."}` |
| A phone number that is not 11 Bangladeshi digits | `400` | `{"error":"Phone number must be 11 digits, for example 01712345678."}` |
| A profile photo over 2 MB | `413` | `{"error":"The profile photo must be 2 MB or smaller."}` |
| A new reset code asked for within 60 seconds | `429` | `{"error":"Please wait a minute before requesting a new code."}` |
| Wrong or expired reset code | `400` | `{"error":"Invalid or expired code."}` |
| Five wrong reset codes | `400` | `{"error":"Too many wrong codes. Please request a new code."}` |
| A first decision outside the window | `409` | `{"error":"The decision window is closed. The class representative decides from now on."}` |
| A change to a class that already started | `409` | `{"error":"This class has already started."}` |
| An action the item's status does not allow | `409` | `{"error":"This schedule item cannot be changed this way right now."}` |
| An item of another teacher or another batch | `404` | `{"error":"No schedule item found with this ID."}` |
| Publishing an item without a date, a time and a place | `400` | `{"error":"CSE 2243 7637D on 2026-10-23 still needs a date, a time and a room or a meeting link."}` |
| A course code with a character outside the rule | `400` | `{"error":"Course code may only use letters, digits, spaces and hyphens, for example CSE 2243 7637D."}` |
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
- **The teaching day is the class periods table**: a class runs from the first
  period's start to the last period's end, never inside a break, and starts
  before it ends. Enforced on the backend, with period pickers on the frontend
  as a convenience only.
- **Only sections `A` and `B` exist**, plus `ALL` for items that target both.
- **Every routine entry carries a semester** (`1-1` to `4-2`), and each batch has
  one running semester in `batch_semesters`. Only the running semester generates
  dated items, fills the course helper, offers drop courses and is checked for
  room clashes.

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

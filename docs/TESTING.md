# Testing

Every check this repository runs, what it proves, and how to run it. The product
itself is described in [SPEC.md](SPEC.md); setup lives in [../README.md](../README.md).

## 1. What runs where

| Suite | Command | Needs |
|---|---|---|
| Go unit tests | `cd backend && go test ./...` | nothing |
| Go database tests | `cd backend && go test ./...` | `TEST_DATABASE_URL` |
| Frontend unit tests | `cd frontend && npm test` | nothing |
| Browser check | `cd frontend && npm run e2e` | a running backend and website |
| Formatting and vetting | `cd backend && gofmt -l . && go vet ./...` | nothing |
| Lint and build | `cd frontend && npm run lint && npm run build` | nothing |

The database tests skip themselves when `TEST_DATABASE_URL` is unset, so
`go test ./...` passes on a machine with no Postgres. They are not optional in
CI: the workflow always sets the variable.

## 2. The test database

Set `TEST_DATABASE_URL` in `backend/.env` or in the environment. The name has to
end in `_test`, which is a guard against pointing the suite at real data:

```
TEST_DATABASE_URL=postgres://postgres:PASSWORD@localhost:5432/sust_coord_test?sslmode=disable
```

Every database test drops and recreates the `public` schema and runs every
migration, under one advisory lock so that two packages never share a schema.
That makes each test independent and the suite slow: the `store` and `server`
packages take several minutes each, which is why CI passes `-timeout 25m`.

The pool pins every session to `APP_TIMEZONE`, exactly as the server does, so a
test sees the same day boundaries as production.

Run one package or one test:

```
cd backend
go test ./internal/store/
go test -run TestOneWeekAgreesAcrossEveryView ./internal/server/
```

Set `FLOW_LOGS=1` to let the flow tests print the server log, which is the fastest
way to find out why a handler answered 500.

## 3. The clock

Nothing in the suite reads the wall clock where a date matters. `config.Config`
carries an injectable clock:

```go
cfg.Freeze(time.Date(2026, 11, 5, 23, 57, 0, 0, dhaka))
```

The date tests walk the moments the portal used to get wrong: 23:57, 00:03, 05:59
and 06:01 Dhaka time, across Thursday, Friday and Saturday. Between six in the
evening and midnight the UTC date is the day before, which is where the wrong
"Tomorrow" came from.

The flow suite freezes at ten in the morning on a day whose tomorrow is a teaching
day, so the fixed routine it generates always lands and no case depends on the
weekday the suite happens to run on.

## 4. The Go suites

### 4.1 Dates, the clock and the calendar

| Test | What it proves |
|---|---|
| `internal/daydate` | Today, tomorrow, day boundaries, weekdays, ISO week numbers and the `Fri 6 Nov 2026` shape, all in Asia/Dhaka and never from the zone a stamp happens to carry |
| `internal/config/clock_test.go` | `Now`, `Today`, `Tomorrow`, `DayStart`, `DayEnd` and `ParseDate` follow the frozen clock and the app zone |
| `internal/db/timezone_test.go` | The pool pins the session zone, so `CURRENT_DATE`, `date_trunc` and every date-to-stamp cast land on the Dhaka day |
| `internal/store/holiday_test.go` | Off days, named holidays and working-day overrides; the generator skips an off day and a holiday and fills a working Saturday with the Sunday routine; adding a holiday removes only the generated items nobody has decided on |
| `internal/handler/holiday_test.go` | The calendar endpoints, their validation, and the notice that reaches the teacher and the class representatives of every item a holiday cleared |
| `internal/jobs/clock_test.go` | Generation, the decision deadline, the past-day cleanup, the exam promotion and all three reminder slots read the Dhaka day across midnight |

### 4.2 The routine views

| Test | What it proves |
|---|---|
| `internal/handler/actions_test.go` | One table over every status, every origin and both roles: an item outside a final routine only offers *Add to final*, one inside it only *Remove from final*, a cancelled item neither, a past day nothing |
| `internal/handler/cr_days_test.go` | `/api/cr/upcoming` answers the day it was asked for, refuses a month-first date, carries the holiday chip with the items below it, and leaves a past day read only |
| `internal/handler/teacher_days_test.go` | The teacher range, the horizon and the actions of each row |
| `internal/server/week_flow_test.go` | One November week end to end: the fixed routine, a class the representative created, an exam the teacher created, a cancellation, a reschedule onto an off day, a holiday, a working Saturday that follows Sunday, a student with a drop course and a publication, asserting every view agrees for each day |

### 4.3 The whole-app flows

Each file is one flow, with a named section per step.

| Test | Flow |
|---|---|
| `internal/server/auth_flow_test.go` | Registration with an email code, verification, approval, sign-in, the lockout and password reset |
| `internal/server/account_flow_test.go` | Google verification, the profile, the password change |
| `internal/server/admin_flow_test.go` | The office portal: people, courses, semesters, class periods |
| `internal/server/schedule_flow_test.go` | Generation, teacher confirm, cancel, reschedule and edit, the decision deadline, room and batch clashes, the publication snapshot |
| `internal/server/routine_flow_test.go` | The fixed routine and the notification audience |
| `internal/server/today_flow_test.go` | Every Today view and a drop-course student |
| `internal/server/board_flow_test.go` | Notices and the board |
| `internal/server/messaging_flow_test.go` | Direct chats, group chats, attachments, unread counts, editing and clearing |
| `internal/server/attendance_flow_test.go` | A Bluetooth session, the token, the position rules, marking from the app, a stale token, a phone far from the class, mark all, close, a correction with a note, a student's own percentage, and a teacher deleting its own sheet and nobody else's; and the three reminder slots, which stay silent when nothing is undecided |
| `internal/server/push_flow_test.go` | Subscribing a browser and the test notification |
| `internal/server/release_flow_test.go` | `android-latest` and `/download/android` |
| `internal/server/retention_flow_test.go` | Record retention and the nightly sweep |

### 4.4 Access and privacy

| Test | What it proves |
|---|---|
| `internal/server/security_review_test.go` | Every `/api` route the router registers is either public on purpose, named in the role table, or covered by the office permission table — read straight out of the source, so the table cannot fall behind a new route. Also that the public list really answers without a token, that the phone routes only answer the app, and that an office token is refused from the app |
| `internal/server/route_security_test.go` | Every protected route answers 401 without a token and 403 to a role that may not call it |
| `internal/server/control_access_test.go` | Every office route declares a permission, the permission is checked, and the control room writes need a password confirmation |
| `internal/server/android_access_test.go` | The `X-Client: android` rule in both directions |
| `internal/handler/privacy_test.go` | A date of birth only appears in an office body and a request; the attendance token key never reaches a student, a closed session or a roll call; no message constant carries a secret |
| `internal/handler/bounds_test.go` | A caller cannot name a page size outside the allowed range, the standing page is bounded, and every list a caller cannot narrow stops at a number |
| `internal/upload/rules_test.go` | Both uploaders decide from the content, not the name, and stop at their size limit |

### 4.5 Server health

| Test | What it proves |
|---|---|
| `internal/metrics/requests_test.go` | A server error counts only when it came from one of our handlers while the server was running normally: the first minute after a start, the platform probes, the 503 the maintenance gate writes and a request the caller abandoned are left out, while a request that ran out of time still counts. The request counter writes one row per flush |
| `internal/handler/dashboard_test.go` | The alert thresholds, including the minimum request volume behind a 5xx alert and the minimum twenty sends behind a notification alert; that the expired subscriptions the server cleaned up are left out of the failure share; that a dismissal keeps one alert quiet for a day without silencing the others; and that every alert carries a message, a tone and a System health section to link to |
| `internal/push/reason_test.go` | Every push answer maps to one reason; 404, 410 and 403 all mean the subscription is gone; the VAPID pair is checked once and the startup line never prints a key |
| `internal/push/push_test.go` | An answer that means the subscription is gone drops the row, an answer worth retrying keeps it, and the log carries the host and the status but never a key |
| `internal/store/health_test.go` | The request counters add up per bucket, the notification failures group by channel and reason, the failing paths carry their count and last message, a dismissal expires, and the nightly cleanup drops the subscriptions nobody has used for sixty days |

## 5. The frontend unit tests

`npm test` runs `node --test src/lib`. Every file under `src/lib` with a
`.test.js` beside it is covered. The ones that matter most here:

- `time.test.js` — every date is read and written in Asia/Dhaka, never in the
  browser zone; `Fri 6 Nov 2026` everywhere; the day/month/year field parses and
  rebuilds a date without a locale; the turnover at Dhaka midnight.
- `schedule.test.js` — what has started and what may still be edited is decided by
  the Dhaka clock, so a phone set to another zone agrees with the server; only the
  actions the server allowed are rendered.
- `paths.test.js`, `nav.test.js` — addresses and the active menu item.
- `account.test.js`, `humancheck.test.js`, `confirmgate.test.js` — form rules.
- `pushkey.test.js`, `semver.test.js`, `refresh.test.js`, `scrollLock.test.js`,
  `attendanceToken.test.js`, `attendanceView.test.js`, `authpanel.test.js`.

## 6. The browser check

`frontend/e2e` drives a real Chromium against a local backend with the demo seed.
For each of the five roles — student, class representative, teacher, office admin
and super admin — it signs in through the real form, visits every page and opens
whatever dialogs those pages offer, at 390 px and at 1280 px, in light and in dark.
That is ten cases per project and forty in all; a full run takes about twenty
minutes. `specs/fonts.spec.js` adds one more per project: it holds the Google
Fonts stylesheet back for good and expects the sign-in form within ten seconds.
Before measuring a page the suite waits for the font, so every reading is of the
face a reader ends up with.

Before any case runs, a global setup (`e2e/setup/grant.js`) signs in as the super
admin, confirms the password and grants the seeded office admin every
permission, the way a real deployment starts. Without it every office page would
answer 403 to an admin nobody had trusted yet.

The dialogs are found rather than listed: on the pages in `DIALOG_PAGES` the suite
presses each control whose name starts like an opener (*Invite*, *Add*, *Edit*,
*Remove*, *Change*, …) and measures whatever dialog appears, so renaming a button
cannot leave a stale table behind.

A case fails on:

- a console error or an uncaught exception,
- a failing API call: any 5xx, or a 4xx the page did not expect,
- horizontal scrolling of the page,
- an element wider than the viewport,
- a name, id or title cut off with an ellipsis,
- a control that stays under a bar after being scrolled into view,
- a dialog taller than the screen with no inner scrolling,
- a page whose last content cannot be reached by scrolling.

These are not faults, and the suite says so explicitly rather than by accident:

- decoration nobody can touch, such as the background shapes of the sign-in pages;
- a wide table or week grid inside its own horizontal scroller;
- a label only a screen reader reads;
- the controls behind an open dialog, which are meant to be out of reach;
- an endless list, such as the activity log, whose end moves as it loads more;
- a page measured mid-animation: the suite waits for every finite animation to
  finish first, and a page that scrolls smoothly is jumped instantly;
- a connection to another site that drops on the way, such as a font file from
  Google on a poor network. The same failure on the portal's own address still
  fails, and so does another site answering with an error status; every console
  error the suite reports names the address it came from.

The app banner that offers the Android app is closed before a page is measured,
since a reader closes it once and never sees it again.

### 6.1 Running it

Three terminals, or three background processes:

```
# 1. a database and the demo data
export DATABASE_URL=postgres://postgres:PASSWORD@localhost:5432/sust_coord_e2e?sslmode=disable
export TEST_DATABASE_URL=$DATABASE_URL
export DEMO_PASSWORD=sust-250-project
export ADMIN_IDENTIFIER=office
export ADMIN_PASSWORD=sust-250-project
export SUPER_ADMIN_IDENTIFIER=chief
cd backend && go run ./cmd/seed -demo

# 2. the server
cd backend && go run ./cmd/server

# 3. the website and the check
cd frontend && npm run dev
cd frontend && npm run e2e
```

The first run needs the browser:

```
cd frontend && npm run e2e:install
```

### 6.2 The variables it reads

| Variable | Default | What it is |
|---|---|---|
| `E2E_BASE_URL` | `http://localhost:5173` | where the website is |
| `DEMO_PASSWORD` | `sust-250-project` | the password every demo account uses |
| `E2E_STUDENT_ID` | `s23a2` | a demo student |
| `E2E_CR_ID` | `s23a1` | a demo class representative |
| `E2E_TEACHER_ID` | `mrahman` | a demo teacher |
| `E2E_OFFICE_ID` | `office` | `ADMIN_IDENTIFIER` |
| `E2E_OFFICE_PASSWORD` | `DEMO_PASSWORD` | `ADMIN_PASSWORD` |
| `E2E_CONTROL_ID` | `chief` | `SUPER_ADMIN_IDENTIFIER` |
| `E2E_CONTROL_PASSWORD` | the office password | the super admin password |
| `E2E_API_URL` | `http://localhost:8080` | the server the app shell's requests are sent to |

Run one project or one role:

```
npx playwright test --config e2e/playwright.config.js --project phone-light
npx playwright test --config e2e/playwright.config.js --grep "the teacher"
```

The four projects are `phone-light`, `phone-dark`, `desktop-light` and
`desktop-dark`. A failed run leaves its report under `frontend/e2e/.artifacts`,
which is not committed.

### 6.3 The phone suite

`specs/phones.spec.js` runs in four more projects: `phone-web-390` and
`phone-web-360` on the website, and `phone-app-390` and `phone-app-360` in the
app shell. The app shell is the real build with a stand-in for the native
bridge (`helpers/appShell.js`): Capacitor reports Android, every plugin answers
with a harmless value, and the requests the app sends to the Render address are
answered by the local server. The office portal and the control room are
website only, so those two roles skip the app projects. That is eighteen cases.

For every page of the role, the case taps up to two list rows, four buttons
that open something (Open, View, Edit, Add, Invite, Reschedule, Cancel, …) and
two links, plus the menu and the notifications, and for each view it opens it
checks that

- the view has its own title, and a row opens its own item: the view names a
  word of the row, and a row with a date opens the session of that date,
- its top and its title are on screen, below the top bar,
- its body scrolls to its end and its main action is on screen,
- it closes cleanly, the page behind is not left locked and scrolls again,
- the page itself still reaches its bottom.

Controls that change data (Delete, Approve, Start, Save, Send, …) are never
pressed. The demo seed carries four weeks of closed attendance for every class
of the running semester, so the History, Sheets and session views have rows.

A request the browser gave up on answers `499` (client closed request), and the
sign-in limiter of ten requests a minute is waited out, so neither counts as a
failure.

## 7. CI

`.github/workflows/ci.yml` runs three jobs on every pull request:

- **Backend** — `gofmt -l .`, `go vet ./...` and `go test -timeout 25m ./...`
  against a Postgres 18 service, the same major version production runs.
- **Frontend** — `npm ci`, `npm run lint`, `npm run build`.
- **Browser check** — loads the demo data, starts the server and the website, and
  runs the Playwright suite. A failed run uploads the report and both logs.

## 8. Adding a test

- A bug gets a failing test first, in the package that owns the rule.
- A date goes through `internal/daydate` on the server and `src/lib/time.js` on
  the website. Nothing else parses or formats one.
- A test that needs a clock freezes it with `cfg.Freeze`, and names the moment in
  the test name.
- A new `/api` route has to be named in `publicRoutes`, `protectedRoutes` or the
  office permission table, or `TestEveryApiRouteIsAccountedFor` fails and tells
  you which one is missing.
- A new page belongs in `frontend/e2e/fixtures/roles.js` so that the browser check
  walks it.

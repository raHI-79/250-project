# Changelog

Every released version of the SUST CSE Class & Exam Coordination System. The
version in `frontend/package.json` and `render.yaml` is the one the app reports
and the one the Android release is tagged with.

## 2.7.2

### Header art and header size

- **The building is always centred.** Every blue header draws the new wide IICT
  illustration as its background, centred at the bottom at full height, so the
  palm trees fill the width at any screen size. A soft darker overlay (top to
  bottom on phones, left to right on desktop, 55 % at the darkest) keeps the
  greeting, the date, the tabs and the badges readable.
- **A compact home hero.** On phones the top bar and the hero together take about
  a fifth of the screen: a smaller avatar, the greeting and the short date on two
  lines and a slimmer search bar on a dark chip. Page headers on phones show only
  the title and the tabs; desktop keeps a moderate height with the subtitle.

## 2.7.1

### Attendance on the website

- **Location on the website.** A teacher on the website no longer sees a Bluetooth
  card that can never work or an "Open app settings" link that does nothing. The
  page says once that Bluetooth works only in the Android app, the code is the main
  action, and the browser is asked for the teacher's position with an **Allow
  location** button; the position then reaches the server every 20 seconds so the
  students' distance check works from the website.
- **Clear steps when location is blocked.** A blocked browser gets the steps for
  Chrome on Android, Chrome on a computer, Safari on iPhone and Mac or Firefox, and a
  **Try again** button. A location service that is switched off on the device is
  named as such. Students entering the code on the website get the same steps.
- **Code only (no location check).** A teacher can start a session as code only,
  or switch an open one to it when location is not available. Those marks are
  saved with the method Code and flagged **No location** in the teacher's list and
  in a "No location" column and cell note of the Excel sheet. Code with location
  stays the default, and roll call needs no location at all.

### Sign-in in the Android app

- **No more endless Google spinner.** The app no longer offers Google sign-in.
  Registration in the app verifies the university email with the 6-digit code
  alone (Send code, enter the code, continue), counts down to the code's expiry,
  says when a code has expired and explains the resend wait. The website keeps
  **Continue with Google** as before.
- **Removed:** the native Google plugin `@capgo/capacitor-social-login`, its
  Gradle entries and the `MainActivity` hooks it needed, and the app-only Google
  code paths. The Android OAuth client in the Google Cloud Console is no longer
  needed; the backend's Google verify endpoint stays for the website.

### The IICT building in every blue header

- **One shared header-art layer.** A flat tonal illustration of the IICT
  building with palm trees now sits behind the home hero of students, CRs and
  teachers, the blue header of every main page, the office portal and control
  room page headers, and the top panel of the app's sign-in, register, forgot
  password and activation screens. The header keeps its solid colour; the art is
  a decorative layer anchored at the bottom right, smaller and fainter on short
  page headers and on phones, dimmer in dark mode, and loaded once as a static
  file. Header tabs and the home search bar sit on a darker chip and the small
  header text is brighter, so white text keeps its contrast over the art.

## 2.7.0

A full quality pass over the website, the app and the server. Every bug below has
a test that failed before the fix and passes after it.

### Dates, the time zone and the calendar

- **Tomorrow was the wrong day every evening.** The database pool ran in the zone of
  the database server, so `CURRENT_DATE`, `now()`, `date_trunc` and every cast
  between a date and a stamp named the day before between six in the evening and
  midnight in Dhaka. The pool now pins every session to `APP_TIMEZONE`. Two
  dashboard tests that were already failing on the branch pass because of it.
- **Two clocks for one deadline.** The password reset deadlines, the conversation
  read and clear marks, the end of an attendance window, the invitation links of
  teachers and office admins, the two-person approvals and the address blocks were
  written by the server and compared by the database. A skew between the two could
  expire a code early, hide a message that had arrived, or put the end of an
  attendance window before its start, which the database refused. The server now
  passes a lifetime and the database writes every one of them.
- **Classes on Friday and Saturday.** The generator knew nothing about off days.
  Weekly off days now live in the settings (Friday and Saturday by default) and a
  holiday list holds named holidays and working days that follow another weekday,
  editable on the office portal's **Routines** page and in the control room's
  **Site settings**. They only stop the automatic fixed routine: a class, lab,
  makeup class or exam created for such a date is kept, published and announced,
  and the create forms say *This is a holiday* instead of refusing the date.
  Adding a holiday removes only the generated items nobody has decided on yet and
  tells their teacher and class representatives.
- **Dates read 11/06/2026.** The website and the app read every date in the browser
  zone and wrote it in the browser locale. One helper now reads and writes every
  date in Asia/Dhaka, every date reads `Fri 6 Nov 2026`, and a day / month / year
  field replaces the browser's own date input everywhere. What has started and
  what may still be edited is decided by the Dhaka clock, so a phone set to
  another zone agrees with the server.
- The settings cache dereferenced a missing store when it saved.

### The routine views

- **Tomorrow and Today listed the same classes.** The hub sent the chosen date and
  `/api/cr/upcoming` read only `from`, so both tabs answered every item from today
  to the horizon under an empty heading. It now answers the day it is asked for.
- **"Remove from final" on rows that were not in it.** That same list carried rows
  of other days that were already in a final routine. One rule on the server now
  decides every action from the role, the status, the origin and `in_final`, every
  view returns it as `allowed_actions`, and the website draws nothing else: an item
  outside a final routine offers only *Add to final*, one inside only *Remove from
  final*, a cancelled item neither, and a past day nothing.
- Every routine view carries the same chip for a day the routine leaves empty —
  *Holiday · Victory Day*, *Holiday · Friday* or *Working day · follows Sunday* —
  with the day's own classes and exams listed under it.

### Server health

- **"The server answered with 5xx more than usual."** The count took every 5xx,
  including the 503 of a maintenance window, the platform health probes and the
  few errors a restart answers before the pool is warm, and it fired on a flat
  number with no idea how busy the period was. It also counted every request the
  caller gave up on: a phone that left a page before it loaded cancelled the
  database call under the handler, which answered 500 to nobody. The browser check
  saw 450 of them in one run and not one real error. It now counts only our own
  handlers after the first minute, leaves out a request the caller abandoned,
  needs 200 requests behind it, and System health lists the failing addresses with
  their count, last message and time.
- **"More than a fifth of the push notifications failed."** A subscription the
  browser had thrown away answered 410 and was counted as a success, a 403 never
  dropped its row, and the same dead endpoints failed on every send for ever. Every
  answer now maps to a reason, 404, 410 and 403 drop the subscription at once, a
  nightly sweep drops the ones unused for sixty days, and the alert needs twenty
  sends and leaves the cleaned-up ones out. System health breaks the failures down
  by channel and by reason.
- The VAPID pair and the Firebase service account are checked once at startup with
  one log line each that never prints a key.
- Each alert can be dismissed for a day, stored on the server, and links to its
  System health section. The banner drew two icons; it draws one.

### Security

- A route table test reads every `/api` route the router registers out of the
  source and fails on one that names nobody who may call it. It found twenty-five.
- The phone-only attendance routes checked the client rule after reading the
  student from the database; they now refuse the website first.
- Proven by new tests: the public routes really are public, an office token is
  refused from the app, a date of birth never leaves an office body, the attendance
  token key never reaches a student, no message carries a secret, every list is
  bounded, and both uploaders judge a file by its content.

### The browser check

- `npm run e2e` walks every page and the dialogs of the student, the class
  representative, the teacher, the office admin and the super admin at 390 px and
  1280 px, in light and in dark, and fails on console errors, failing API calls,
  sideways scrolling, overflow, cut-off text, controls under a bar, dialogs taller
  than the screen and pages that cannot be read to the end. CI runs it on every
  pull request.
- It found the per-session **Sign out** button on the account pages squeezed until
  its label was cut off on a phone.
- It found every page waiting for Google Fonts. The font stylesheet blocked the
  first paint and the start of the app, so a slow or blocked answer from Google
  left a blank screen, not even the splash, for as long as the network took: a
  stalled answer kept the sign-in page unusable past thirty seconds. The
  stylesheet now loads on the side, the page reads in the system face until
  Nunito lands, and a spec holds the font back for good and expects the sign-in
  form within ten seconds.

### Removed

- Thirty-five exported Go names nothing used, among them `DeleteOldSessions`,
  `HasMailer`, `ConversationScope`, `NoteAudience`, `SectionsWithAll` and
  twenty response messages no handler sent.
- The `/.well-known/assetlinks.json` handling left over from the PWABuilder app.
  The Capacitor app declares no App Links, and no such file is shipped.
- `NoteFields`, a form component nothing rendered, with `noteError` and
  twenty-one other frontend exports nothing imported.
- The `rule-under` and `text-shadow-hero` utilities and three campus banner
  photos (807 KB) that no page shows.
- `go mod tidy`: two modules that were imported directly but listed as indirect,
  and fourteen stale checksum lines.

The built site shrinks from 8,224,107 to 7,416,772 bytes (−807,335, −9.8 %). The
JavaScript and the CSS are the same size as before, because the bundler had already
left the dead code out; the saving is the three photos. Every npm dependency is
still imported.

### Needs doing by hand

- Nothing new on Render: the off days default to Friday and Saturday, and the
  holiday list starts empty. The new migrations run on the next deploy.
- After the deploy, the super admin may want to dismiss any alert left over from
  the old counting; it clears on its own as the period moves on.

## 2.6.1

- **The portal asks for the password instead of refusing.** A `428` from the
  confirmation rule now opens one shared *Confirm it's you* dialog in the API
  client, calls the confirm endpoint and retries the original request once, so
  Remove, Suspend, Resend, Cancel invite, Revoke access, permission changes,
  settings saves, IP blocks and the user actions finish in one go. Several
  requests that need it at the same time share the one dialog and are all
  retried after a single confirmation, a wrong password keeps its error in the
  dialog, cancelling rejects quietly with no toast, a two-step account gets a
  code field with the code mailed to it, and the header shows *Confirmed for 5
  minutes* while the window is open. The office portal uses the same flow.
- **One menu item is active at a time.** The sidebar and the tab bar no longer
  rely on prefix matching, which lit up Dashboard, Site settings and My security
  together: the active item is the longest menu path that matches the current
  route, so Dashboard is active only on `/control-room` and a detail page keeps
  its own list highlighted. The two security entries are now *Security monitor*
  and *My account*, at `/control-room/settings/account` and
  `/office-portal/account`, with the old addresses redirecting.
- **Pages update themselves after an action.** Every successful change
  invalidates the shared resource version, so each mounted list, detail,
  dashboard count and sidebar badge refetches at once instead of waiting for a
  reload, and an empty toast is never shown.
- **The Admins page fits a phone.** The rows wrap instead of scrolling
  sideways: the name, status and counts on one line, the permissions under them
  and the buttons on their own line.

## 2.6.0

- **Sessions you can see and close.** Every sign-in now opens a row in
  `user_sessions` and the token carries its id, so the server can end a session
  on the spot. **Security** in the control room and in the office portal lists
  every device with its client, the country the address resolves to or
  *Unknown*, and when it was last seen, with *Sign out* per row and *Sign out
  everywhere else*. Changing a password signs every other session out.
- **Stronger office passwords and a second step.** An office password must be at
  least 10 characters with a letter and a digit. Optional two-step sign-in sends
  a 6-digit code to the address on the account, good for 10 minutes and five
  tries. Contact email and phone change only after the password is given again.
  `/control-room/forgot-password` resets the super admin password against the
  registered phone number, rate limited and recorded.
- **An emergency reset from the server.** `go run ./cmd/admin -reset-super`
  reads `NEW_SUPER_ADMIN_PASSWORD` from the environment — never from an
  argument — refuses anything weak, replaces the hash, closes every session and
  prints only *Password updated*.
- **Five wrong passwords lock an account.** The lock lasts 15 minutes, the
  control room lists the locked accounts and can unlock any of them, and an
  office sign-in from a device the account has not used before emails its owner.
- **Admins only see what they may use.** Each office admin, and each teacher
  with office access, carries a set of permissions: registrations, students and
  CRs, teachers, courses and semesters, class periods, routines, notices,
  attendance and phone changes, and exports. A new admin gets everything except
  exports. One middleware checks the permission of every office route against a
  table, answering *You do not have permission for this*, the office navigation
  hides what the admin may not open, and a test proves every office route is in
  the table.
- **A users page in the control room.** Search any account by name, registration
  number, User ID or email, filter by role, status and batch, and open a profile
  with its role, status, creation, last sign-in, sessions, phone binding and
  recent activity. Suspend or restore, sign out everywhere, reset the attendance
  phone, switch a student between student and CR, act on a selection in bulk,
  and export the filtered list as CSV.
- **Site settings without a redeploy.** Maintenance mode with its own message
  closes the portal for everybody except the office and the control room — the
  API answers `503` and the pages show a calm curtain. Registration can be
  closed with a message, an announcement banner can be aimed at chosen roles
  with an end date and a severity, attendance, polls, messages, group chats and
  attendance from the website each have a switch, and the reminder times, the
  morning reminder and the number of days the routine is generated ahead now
  live in the settings with the environment as their default.
- **App release control.** A minimum supported Android version makes older apps
  show a blocking *Update required* screen with an Update button. The control
  room shows the current release, its download count from GitHub and the share
  link, and can send the update notification again.
- **A security monitor.** Failed sign-ins by hashed address and by account over
  the last week, the locked accounts, and blocking or unblocking an address or a
  range for a chosen number of hours; a blocked request is refused with `403`
  and recorded.
- **Two heads for the dangerous things.** While another active office admin
  exists, deleting an account for good, deleting a whole attendance sheet and
  removing an office admin become requests that another admin must approve
  within 24 hours. The super admin goes ahead after confirming its password, and
  the override is recorded.
- **Every sensitive action asks for the password again.** A confirmation opens a
  five-minute window on the current session; without it the control room answers
  `428` with *Please confirm your password first.*
- **A data page and real retention.** The last backups, CSV exports of the
  users, the attendance summary and the activity log, the number of rows the
  portal holds, and how long page views, activity events and notifications are
  kept — swept every hour by the cleanup job, with closed sessions dropped after
  30 days.

- **The human check is back on every website form.** The invite-acceptance page
  never had the widget although its route sits behind the human gate, so
  accepting an office invitation always answered *human verification needed*; it
  now carries the same check as the other forms. The loader no longer waits for a
  `load` event that may already have fired: one shared promise resolves as soon
  as `window.turnstile` exists and gives up with a readable message after twelve
  seconds, so a remounted form always draws its widget. Sign-in, register, forgot
  password, teacher activation, the office portal, the control room and its
  forgot password and the invite acceptance all keep the submit button disabled
  until a token exists, send the pass with the request, reset the widget after a
  failed attempt, follow light and dark mode, and show nothing inside the Android
  app, which needs no pass. With no site key the widget and the server check are
  both skipped.
- **The invite button works again.** `useToast()` hands back `{toast, success,
  error, info, dismiss}`, but the newer control-room and office pages called the
  object itself, which is the minified *n is not a function*. Every one of those
  calls now goes through `toast.success` or `toast.error`, so inviting an office
  admin creates the invitation, sends the email and shows the server's own text
  when sending fails.
- **Autofilled fields keep their shape.** The border, radius and background sit
  on the input, select and textarea themselves instead of on a wrapper, and
  `:-webkit-autofill` (with hover, focus and active) repaints Chrome's light-blue
  rectangle with a 1000px inset shadow in the field background, keeps the text
  colour and caret, and parks the background transition far in the future. The
  floating label, the show-password eye and the focus ring stay where they were.

- **Encrypted nightly backups.** The dump is encrypted with
  `gpg --symmetric --cipher-algo AES256` against the new `BACKUP_PASSPHRASE`
  secret before it is uploaded, and only the `.gpg` file leaves the runner, so an
  artifact on a public repository is no longer a readable copy of the database.
  The job fails with a clear message when the secret is missing, the passphrase is
  never printed, retention stays 30 days and the backup still reports itself to
  the portal. `docs/DEPLOY.md` has the download, `gpg --decrypt` and `psql`
  restore steps.
- **The app is served by the portal.** `GET /download/android` no longer
  redirects to GitHub: the server asks the GitHub asset API for
  `sust-cse-portal-<version>.apk` with `GITHUB_TOKEN`, follows the redirect to the
  storage host without passing the token on, caches the file under
  `UPLOAD_DIR/apk/` so GitHub is asked once per release, and answers it as
  `application/vnd.android.package-archive` with an attachment name and a
  `Content-Length`. `GET /download/android/<version>` serves one version and is
  what `apk_url` points at, so the in-app updater downloads through the portal as
  well; downloads keep working if the repository becomes private. Without a token
  or when GitHub refuses, the answer is `503` with a readable message and the log
  holds the status only.
- **The emergency reset runs on Render.** The image now carries
  `/app/sust-admin` next to the server, so the reset needs no Go toolchain and no
  source: `export NEW_SUPER_ADMIN_PASSWORD='…'`, `/app/sust-admin -reset-super`,
  `unset NEW_SUPER_ADMIN_PASSWORD`. `go run ./cmd/admin -reset-super` still works
  on a development machine.

## 2.5.0

- **A sign-in screen built for the phone.** Inside the Android app only, the
  sign-in, register, verify, forgot-password and teacher-activation screens share
  one shape: a solid #0A6FAF panel about 330 px tall that respects the status bar,
  with the real SUST logo on a white 92 px square at a 26 px radius, the screen
  title in Nunito 800 at 26 px and a one-line subtitle, and a white sheet that
  overlaps it by 40 px with 28 px rounded top corners. The sheet carries the
  Student | Teacher segmented control, 50 px labelled inputs that take a 2 px
  primary border and a soft #E3F0FA ring on focus, a full-width 52 px *Sign in*
  button, a centred *Forgot password?*, *New student? Create an account* and the
  line *Department of CSE, SUST · Version <current>*. The form scrolls inside the
  sheet, the keyboard never hides the focused input, touch targets stay at least
  44 px and dark mode reads the dark tokens. The website keeps its own campus
  sign-in, and no sign-in logic or API call changed.
- **A super admin above the office.** The account seeded from
  `ADMIN_IDENTIFIER`, or from `SUPER_ADMIN_IDENTIFIER` when that is set, becomes
  `super_admin` on start. It signs in only at the hidden website route
  `/control-room` — never linked, `noindex`, the same human check as the other
  forms — while `/office-portal` stays for the office admins and each entrance
  refuses the other's accounts. The app reaches neither. From the control room
  the super admin opens every office page read-only: the backend answers any
  write from that role with *The control room opens the office pages read-only.*
- **Office admins are managed, not hard-coded.** The control room invites an
  office admin by email through the existing mailer; the link lives 72 hours and
  lands on a page where the invitee picks a username under the teacher User ID
  rules and a password. The list shows status, last sign-in and action count, and
  an admin can be suspended, restored or removed by typing their username. Any
  active teacher can be given or refused office access at any time: that teacher
  then signs in at `/office-portal` with their own credentials while their
  teacher account, teacher pages and app access stay untouched. Every one of
  those actions is audited.
- **Everything is on the record.** `audit_events` keeps the actor and role, the
  action, the target, a summary, the IP address hashed with `ANALYTICS_SALT`, the
  user agent, whether it came from the website or the app, and the time. One
  middleware behind the token check records every create, update, delete and
  publish of every role that answers 2xx, and the sign-in handlers record every
  sign-in, sign-out and failed sign-in. The activity log filters by person, role,
  action, client and date range, searches, loads more as you scroll and exports
  CSV; *Monitor this admin* opens that account's own timeline with counts per
  action type, and the office keeps its own log.
- **Privacy-friendly analytics.** Every page open sends a beacon with a visitor
  id from `localStorage`, a session id, the path, the client, the device kind and
  the app version — no account id, no readable address — batched to
  `POST /api/analytics/view` at 20 a minute per address. An hourly job rolls the
  rows into daily aggregates and deletes raw rows after 90 days.
- **A dashboard at /control-room.** Desktop-first and responsive, drawn with
  recharts in the Ocean Blue tokens, with skeletons, a Today / 7 days / 30 days /
  Semester / custom range picker, a refresh every 60 seconds and the time it was
  last read. *Overview* has KPI cards with sparklines and the change against the
  previous period, visitors and sign-ins per day, sign-ins by role, a donut of
  website against app, the app versions in use, the top pages and the devices.
  *Academic* has the attendance average per batch and section, how many students
  sit below 50 %, classes held against cancelled against rescheduled per course,
  tomorrow's undecided classes per teacher, pending phone change requests and the
  notices, polls and messages of the period. *System health* has uptime, version
  and commit, database size, uploaded file storage, 4xx and 5xx counts with the
  latest five 5xx paths, push and FCM sends with their failure share and token
  counts, emails sent and failed, and the last successful database backup. An
  alerts banner covers a 5xx spike, an email or push failure share above a fifth,
  a backup older than 36 hours, a failed sign-in spike from one address and a
  deleted attendance sheet. Every number has a label and a tooltip, and each
  panel has an empty state. The dashboard loads as its own chunk, so the app
  bundle is unchanged.
- **The backup reports itself.** The nightly workflow posts the dump name to
  `POST /api/system/backup-status` with `X-Backup-Secret`, checked against
  `BACKUP_STATUS_SECRET`; anything else is refused with `403`. Without the
  secrets the workflow skips the report and the dashboard honestly says *Never*.

## 2.4.0

- **Nothing is cut off any more.** The desktop sidebar is its own fixed 18 rem
  column with wrapping labels, so a page that overflows sideways can no longer
  shift it off-screen or clip its labels, and it stays in place on the teacher
  *Upcoming* and *Final routine* pages. Every dialog and sheet is capped at
  90 dvh, keeps its title bar and its action buttons visible and scrolls its
  body inside, centred on desktop and a bottom sheet on a phone. The sticky top
  bar is opaque, no backdrop blur is left in the build, and the *N students
  still unmarked* notice sits in the page above the *Finish* button.
- **Phones scroll again, everywhere.** The page scroller is the viewport: the
  x-axis clip moved off `html` onto `body` and the shell is a min-height column
  with one scrolling main area instead of a fixed-height box. Dialogs take the
  page scroll lock from a shared counter (`src/lib/scrollLock.js`, with unit
  tests), so a confirm opened over a sheet can no longer leave the page locked
  when the two close in the same render. Pull to refresh is fully passive,
  reads that shared lock and ignores a sideways or upward gesture, and
  `touch-action: none` is left only on the sheet drag handle.
- **The roll call can be read aloud.** Each row carries a large roll number
  (the last three digits of the registration number), the full name over two
  lines when it needs them and the full registration number, sorted by
  registration number, with 44 px *P* and *A* buttons that drop under the name
  below 380 px. The note textarea and *Save note* button are gone: a note icon
  opens a one-line input, and a saved note shows as one muted line you tap to
  edit. The attendance session detail uses the same row. Names, registration
  numbers, course titles and room numbers wrap instead of ending in an
  ellipsis.
- **Ocean Blue.** One token layer carries primary #0A6FAF with #075A8F pressed
  and the #E6F1FC tint under #16599E text, page #F4F7FA, white cards at 16 px
  with the 0 2px 8px and 0 6px 20px shadows, #E8EDF2 borders, inputs on
  #F8FAFC inside #DCE3EA at 14 px, text #1B2533 / #5B6675 / #6B7684, and the
  success, warning, danger and purple tints; dark mode keeps the structure on
  page #0E1620, card #16212D and primary #3A9AD9. Nunito at 400, 600, 700 and
  800 replaces Inter and Source Serif 4. Buttons, inputs, chips, segmented
  controls, tags, sheets, dialogs, toasts, empty states, the tab bar and the
  sidebar all read from the shared tokens.
- **New screens on the same data.** Each main page opens with a solid primary
  header: white title, its tabs as a white pill on translucent white and a
  white round date button where the page reads a date. Home is a blue header
  with the avatar, name and registration number, a white stats card overlapping
  it (classes today, attendance, semester), a 4x2 grid of white tiles with
  coloured line icons on soft tints, the latest notices as cards with a tinted
  tag and a bookmark, and today's classes as horizontal pastel cards. The
  routine is a timeline with the time on top, the dot and line on the left and
  the card on the right, statuses are tinted tags, and the bottom tab bar
  carries a 3 px primary indicator above the active tab with its labels kept.
  The Android status bar matches the header. No feature, permission, API call
  or page structure changed.

## 2.3.0

- **Continue with Google works in the app again.** `MainActivity` now forwards
  the social-login authorisation result back to the plugin, so the sign-in no
  longer waits for a result that never arrives. The screen prints the exact text
  Google or the server returned, and the server classifies a refused credential
  (wrong client id, expired, otherwise invalid) and logs the reason without the
  token. `docs/ANDROID.md` lists everything the Google Cloud Console needs.
- **Registration and the password reset are single pages.** All fields sit in
  Account, Contact, Personal and Photo sections with inline validation next to
  each field, one Submit button, and the email-code input appearing in place
  after *Send code*. The API did not change. The teacher sign-in screen no
  longer offers an activation link; teachers still activate from their
  invitation email.
- **A phone pass over every page and dialog** at 360, 390, 412 and 768 px: no
  horizontal scroll, long text wraps, button rows wrap, touch targets are at
  least 44 px and sheets keep their buttons visible.
- **A class a class representative creates belongs to the course teacher.** The
  teacher of the course for that batch and section is chosen automatically and is
  editable only when the course has several teachers. The class then waits for
  that teacher with Confirm, Cancel and Request reschedule, reaches the next-day
  reminders, and flows into the final, today and published routines like a
  generated class.
- **A teacher can edit a class or exam they own** until it starts: the date, the
  start and end time, the room and the mode, with the meeting link editable
  until the class ends. Every change re-checks the room and batch clashes,
  updates the published copy and tells the class representative and the students.
  `PATCH /api/teacher/items/{id}`.
- **Every routine view reads a date.** Today views show only today. The student
  and class-representative routine, both final routines and the exams take a date
  picker with a *Back to today* link; a past day is read-only and carries each
  item's final status (Held, Cancelled, Rescheduled, Exam held) from the kept
  rows, so the final-routine cleanup no longer hides it. Teacher and class
  representative Upcoming open on tomorrow with a Tomorrow | Today toggle and a
  *Show more days* button that loads the next day group up to the generated
  horizon.
- **The class and exam hub** is ordered Upcoming routine · Final routine ·
  Create class / exam · Room availability and opens on the upcoming routine.
- **One Attendance item per role**, with tabs inside it: Take attendance ·
  History · Sheets · Phone requests for a teacher, Mark attendance · My
  attendance · My phone for a student, History · Sheets · Phone changes for the
  office. The separate phone screens left the navigation and their routes still
  work.
- **Roll call.** A teacher picks Bluetooth or Roll call when starting. Roll call
  shows the roster with photo, reg no, a Present and an Absent button per
  student, a Mark all present shortcut, a note with its own Save button, and a
  Finish that marks the rest absent after a confirm. The same per-student editing
  works during and after a Bluetooth session.
- **The website can mark attendance** with the teacher's rotating 4-digit code
  and browser geolocation within 30 m of the teacher, under the same accuracy and
  freshness rules. Those marks are stored with method `web` and labelled *Web* in
  the teacher's list, so the teacher can change them. The app flow is unchanged,
  and registering a phone stays in the app.
- **One sheet per course, batch and section.** Bluetooth, code, web and roll-call
  sessions share the same sheet and the same Excel export. A teacher or the
  office can delete a single session or a whole sheet after a confirm naming the
  course, the batch and the section; the deletion is audited, students stop
  seeing the removed sessions and every percentage is recounted.

## 2.2.5

- The teacher's upcoming routine opens on tomorrow's full routine instead of only
  the classes that still need a decision, so a settled day is still readable. Every
  non-cancelled class, lab and exam carries its status and only the actions that
  status allows, and the days after tomorrow sit behind *Show later days*.
- *Today* now lists only the confirmed, rescheduled and published items of the day,
  each with its room or Join link and a cancel action, and an empty day says so
  instead of showing *Nothing to decide*.

## 2.2.4

- The teacher's upcoming routine is no longer only tomorrow. It lists every
  undecided or decidable class from today through the generated horizon, grouped
  under a header per day, and Confirm, Cancel and Request reschedule work on any
  of those days. `GET /api/teacher/upcoming?from=&to=` answers the days;
  `/api/teacher/upcoming/next-day` still answers the single day for older apps.
- The teacher's final routine is grouped by day from today onward and carries the
  teacher's confirmed, rescheduled, in-final and published items, the items the
  class representative made for their courses and their own exams. A class or an
  exam can be cancelled from there with an optional reason, which releases the
  room, cancels the published copy and tells the class representative, the batch
  and the students who took the course as a drop course.
- The class representative can remove any item from the final routine, one at a
  time or several at once, and a published item is withdrawn from the batch
  routine when it goes. Items the class representative made can also be deleted
  for good; an exam of a teacher can only be removed, never deleted.
- Both final routines show today and the days ahead only. A job at five past
  midnight, and once on startup, takes the past days out of the final routines
  while keeping the rows for attendance, history and reports.

## 2.2.3

- The app finds an update again. `/api/app/android-latest` was answered with
  `Cache-Control: public, max-age=600`, so a phone that had asked while 2.2.1 was
  the newest release kept reading that same answer out of the WebView cache and
  compared 2.2.1 against itself. The endpoint now answers `no-store`, the app
  fetches it with `cache: "no-store"` and a cache-busting query, and it checks
  again on start, on resume and when the update notification is tapped.
- The installed version is read from the APK with `App.getInfo().version` instead
  of a constant baked in at build time, and the up-to-date message names both
  numbers: *Installed 2.2.3 · Latest 2.2.3*.
- The APK versionCode is now major times ten thousand plus minor times a hundred
  plus patch, the same number the server reports, instead of the CI run number.

## 2.2.2

Everything here came back from real phones.

### Attendance

- The teacher's own session now reaches the app with its Bluetooth key.
  Capacitor reads a JSON number back as an `Integer`, never a `Long`, so
  `session_id` silently became `0` and the plugin refused its own session with
  *The session is missing its key*. The id travels as a string now, and a shared
  test vector pins the eight token bytes in Go and in JavaScript.
- Location no longer waits on Bluetooth. The session screen asks for precise
  location on its own and shows the truth — Allowed, Off, Denied, or Sending with
  the accuracy the server saw — with a button to the app or location settings.
  The foreground service listens on GPS and network and sends the first fix at
  once.
- When Bluetooth cannot broadcast the screen says exactly why and points at the
  4-digit code, which never depended on Bluetooth.
- The duration chips scroll in their own row above Extend and Close, so nothing
  overlaps at 360 px.
- The server logs every session start, every teacher position and the reason
  behind every refused mark.

### Registration and sign-in

- Every date-of-birth picker opens on today instead of fifteen years ago, with
  nothing preselected. The age rule still applies.
- The Google client id comes from `GET /api/auth/config` at runtime, so changing
  it needs no rebuild. `VITE_GOOGLE_CLIENT_ID` is gone.
- Google sign-in works inside the Android app through native sign-in, since
  Google refuses its web flow in a WebView. The release workflow prints the
  signing certificate fingerprints for the Android OAuth client.
- A refused origin, a blocked popup and a cancelled dialog each get their own
  message.

### Approvals

- Approving a registration emails *Welcome to SUST CSE Portal* with the sign-in
  instructions and the app link.
- Rejecting asks for a reason, at least five characters, with quick picks. The
  reason is emailed, kept on the account and shown on the sign-in screen.

### Messaging, sheets and photos

- The message search bar and the role chips stay at the top of the list, and the
  chips now filter the existing conversations as well as the people search.
- Sheets open at the height their content needs up to 92 % of the screen, scroll
  inside, keep Save and Cancel visible and move a focused input clear of the
  keyboard.
- The public home page shows the campus slideshow again; the sign-in, register,
  password and activation screens show one still photo of the IICT building. The
  app keeps no photos on its sign-in screens.

## 2.2.1

- The release lookup survives GitHub errors. It reads a page of releases and
  picks the newest one that is neither a draft nor a prerelease and actually
  carries an APK, sends an optional `GITHUB_TOKEN` and a `User-Agent`, keeps the
  last good answer for six hours, and answers `503` *Update service is busy, try
  again later.* instead of `500` when GitHub cannot be read. The download link
  falls back to the repository releases page.

## 2.2.0

### Class attendance by Bluetooth

- Teachers open attendance for one class of the running semester for 5 to 30
  minutes, can extend it or close it early, and see the live count, the latest
  marks and the rotating 4-digit code for a projector.
- While a session is open the teacher's Android app advertises a Bluetooth Low
  Energy token that changes every 10 seconds and keeps sending the teacher's GPS
  position every 15 seconds, both from a foreground service with an ongoing
  notification, so a locked screen does not stop attendance.
- Students mark themselves present with one tap from the Android app. The phone
  has to hear the teacher's current or previous token, sit within 30 metres of
  the teacher's latest position with an accuracy of 50 metres or better, report
  a position that is not from a mock provider, and pass the phone's own
  fingerprint, face or screen-lock check.
- A phone with no Bluetooth, or that hears nothing within ten seconds, falls
  back to the 4-digit code on the board. The location rules still apply.
- The first Android phone a student signs in with becomes their attendance
  phone. Every other feature keeps working on any device and on the website;
  only attendance is refused elsewhere.
- From a new phone a student sends a request with a reason. It reaches every
  teacher who teaches their batch in the running semester, and any one of them
  can approve or refuse it from "Phone requests". Two approved changes per
  student per semester need no note; beyond that the teacher has to add one.
  The office only reads the history.
- One phone marks one student per session, a student marks once per session,
  and closing a session turns every roster student without a record into an
  absence. The roster is the batch and section of the class, with `ALL` meaning
  both sections, plus everyone who added the course as a drop course.
- Excel export with `excelize`: one sheet per course-batch-section with a column
  per session date, the note as a cell comment, the present count, the total and
  the percentage, below 50 % in red and a frozen header. "Download all my
  classes" adds a Summary sheet, and the office can export every batch.
- A student under 50 % in a course, after at least three sessions, hears about
  it once and then at most once a week while it stays below.

### Android

- New `AttendanceBeacon` and `AttendanceMark` Capacitor plugins, the
  `AttendanceService` foreground service, and the Bluetooth, nearby devices,
  precise location and biometric permissions.
- `androidx.biometric` is now a dependency.

## 2.1.0

- The app updates itself: it reads the newest GitHub Release, offers the update
  and installs the signed APK, with `[required]` in the notes making it
  unskippable.
- `/download/android` always redirects to the newest signed APK.
- One logo in every notification.
- The maroon palette became a calm deep blue one.

## 2.0.1

- Push notifications through Firebase Cloud Messaging for the Android app, web
  push for the browser, and one bell for both.

## 1.1.0

- Messaging, polls, materials and the batch Drive link.

## 1.0.0

- The first release: accounts, the fixed weekly routine, the scheduling engine
  that CRs and teachers share, exams, notices and the department office panel.

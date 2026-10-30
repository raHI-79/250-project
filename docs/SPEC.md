# SUST CSE Class & Exam Coordination System

Final specification of the product as it is built. It describes what the system
does, who may do it, and the rules the backend enforces. The build order lives in
[ROADMAP.md](ROADMAP.md); setup and the API reference live in
[../README.md](../README.md).

## 1. Overview

### 1.1 Purpose

The Department of Computer Science and Engineering at Shahjalal University of
Science and Technology runs its weekly routine on paper and its day to day
changes on group chats. A class moves, an exam is announced, a room is swapped,
and the news reaches some students and misses others.

This system keeps one copy of the truth. The office enters the fixed weekly
routine. The scheduling engine turns that routine into dated classes. Every
teacher decides on their own next day, every class representative builds the
final routine their batch reads, and every student sees one page for today.

### 1.2 Scope

One department, one university: SUST CSE. Batches look like `CSE-23`. Every batch
runs two sections, A and B.

### 1.3 Tech stack

| Part | Choice |
|---|---|
| Backend | Go, standard library `net/http`, `pgx` |
| Database | PostgreSQL, plain SQL migrations embedded in the binary |
| Frontend | React with Vite, Tailwind CSS, React Router |
| Auth | JWT in the `Authorization` header, bcrypt password hashes |
| Push | Web Push with VAPID (`webpush-go`), a service worker |
| Mail | SMTP for verification codes and teacher invitations |

### 1.4 Roles

| Role | Who | How they sign in |
|---|---|---|
| `student` | A registered student of a batch and section | Registration number |
| `cr` | A student the office promoted to class representative | Registration number |
| `teacher` | A teacher of the department | A User ID they choose |
| `admin` | The department office | A hidden office portal |

`cr` is not a separate account. It is a student account with `is_cr` set, and the
backend resolves it to the effective role `cr` on every request.

## 2. Access control

### 2.1 The four parts of the site

- The **public home page** carries the department numbers and the two entrances.
- The **student and CR panel** is for registration numbers.
- The **teacher panel** is for teacher User IDs.
- The **office portal** is hidden. It is never linked from a student, CR or
  teacher page, and it is excluded from search engines.

### 2.2 Backend enforcement

Every rule is enforced in the backend. Hiding a button in the frontend is a
convenience, never a control. Route groups attach the token check and the role
check before the handler runs, so a route that is not in a group is unreachable.

### 2.3 Class representative rules

- Only the office promotes and demotes a class representative.
- A CR acts on their own batch only, and on their own section where the rule
  says so.
- A CR never sees another batch's planning pages.

### 2.4 Messaging rules

- A student or a CR may start a direct chat with teachers, class
  representatives and the department office.
- A plain student may **not** start a direct chat with another plain student.
  The attempt answers `403`, and the contact search of a plain student never
  lists plain students.
- A CR may message students. Teachers and the office may message anyone.
- Group membership is not limited by this rule: a student placed in a group
  posts in it like every other member.

### 2.5 Account security

- Passwords are bcrypt hashes. Nothing stores or logs a password.
- A student registers, verifies their university email, and waits for the
  office to approve the account.
- A teacher is invited by the office, opens the invitation link, chooses a User
  ID and a password, and the account becomes active.
- Password reset runs on a short-lived emailed code.
- A removed or rejected account cannot sign in and its token is rejected.

## 3. Sections, semesters and class periods

### 3.1 Sections

Only two sections exist: `A` and `B`. Wherever an item can target both at once,
the stored value is `ALL` and the interface says **Both sections**. Registration
offers A or B and nothing else. Every section field in the backend goes through
one validator, and the migration that locks the rule in fails loudly, naming the
offending value, if the database still holds a third section.

### 3.2 Semesters

A batch studies eight semesters, stored as `1-1` to `4-2` and shown as
*1st year 1st semester* and so on. Every routine entry belongs to one of them,
and every batch has one **running semester** in `batch_semesters`. The first
routine entry the office enters for a batch sets that batch's running semester;
after that only the office changes it.

The running semester is what the rest of the system reads:

- only its entries become dated schedule items;
- only its courses appear in the class representative's course helper and count
  as drop course options;
- only its rooms and bookings are checked for clashes, so the same room may hold
  a class in two different semesters;
- the teacher's weekly routine shows it alone.

Entries of other semesters stay in the database and can be read on the routine
pages by picking that semester.

### 3.3 Class periods

The teaching day is a list of class periods the office owns, seeded as:

| Period | Time |
|---|---|
| Period 1 | 09:00 – 09:45 |
| Period 2 | 09:45 – 10:30 |
| Period 3 | 10:30 – 11:15 |
| Period 4 | 11:15 – 12:00 |
| Period 5 | 12:00 – 12:45 |
| Break | 12:45 – 13:45 |
| Period 6 | 13:45 – 14:30 |
| Period 7 | 14:30 – 15:15 |
| Period 8 | 15:15 – 16:00 |
| Period 9 | 16:00 – 16:45 |

The office adds, edits, deletes and reorders periods on the **Class periods**
page. A period starts before it ends and never overlaps another one, and the day
keeps at least one period.

The allowed day runs from the first period's start to the last period's end, and
nothing may sit inside a break. Class representatives, teachers and the office
pick a **From period** and a **To period** rather than typing a clock time, so a
lab may run over several periods and the exact times are shown next to the
labels.

## 4. The scheduling workflow

### 4.1 The fixed routine

The office enters the weekly routine for a batch, semester and section: course,
teacher, weekday, period range, room and whether it is a class or a lab. A
routine entry that books a room another batch already holds **in the same
semester** is refused with the clash named. The office moves an entry to another
semester, and sets the running semester of a batch, from the same page.

### 4.2 Dated items

A background job turns the fixed routine into dated schedule items for the days
ahead. Every item carries a status:

| Status | Meaning |
|---|---|
| `awaiting_teacher` | Generated from the routine, no decision yet |
| `no_response` | The decision window closed with no decision |
| `confirmed` | The teacher confirmed it |
| `cancelled` | The teacher or the CR cancelled it |
| `teacher_reschedule_request` | The teacher asked for another slot |
| `cr_reschedule_request` | The CR proposed a new slot |
| `reschedule_declined` | The teacher kept the original slot |
| `rescheduled` | A new slot was agreed |
| `scheduled` | Created by the CR or by a teacher exam |

### 4.3 Teacher decisions and reminders

There is no decision window. Confirm, Cancel and Request reschedule work at any
hour until the class starts, on `awaiting_teacher` and `no_response` items
alike. Once a class has started its item can no longer be changed.

Reminders push and ring the bell instead:

- `TEACHER_REMINDER_TIMES` defaults to `16:00,21:00`. The first slot says
  *Tomorrow's classes need your decision · `<n>` classes*, the later ones
  *Reminder: `<n>` classes tomorrow are still undecided*.
- `TEACHER_MORNING_REMINDER` defaults to `07:30` and covers today's undecided
  classes that have not started yet.
- `reminder_log` keeps one row per teacher, slot and day, so a reminder is
  never sent twice.

After the last slot of `TEACHER_REMINDER_TIMES`, tomorrow's items still
`awaiting_teacher` become `no_response`. That is only a signal to the class
representative: the room stays booked and the CR may decide.

The class representative is never limited either. Creating a class, asking for a
reschedule, setting a new slot, cancelling, adding to the final routine, editing
a link and publishing all work at any hour.

### 4.4 Rooms and clashes

- Two items never share a room at the same time.
- A batch and section never hold two items at the same time, and an item for
  both sections clashes with either one.
- A room nobody's routine uses is not managed, so it is never checked.
- With mode **Online** a room is not required and the meeting link is optional
  at creation. The item reads **Link coming soon** and offers **Add link** to
  the teacher and to the class representative until a link exists.

### 4.5 Room availability

The availability page shows rooms against class periods for each date. A break
column is shaded, a free period is green and opens the create form already
filled in, and a booked period is red with a tooltip naming the batch, the
section, the course and the status.

### 4.6 The final routine

The class representative picks the items their batch should read and publishes
them. Publishing writes a snapshot, so later edits to an item never change what
was published; a new publication replaces it.

## 5. Student pages

| Page | Content |
|---|---|
| Today | Today's published classes, exams and notices in one timeline |
| Upcoming routine | The published routine ahead, by date |
| Fixed routine | Any batch and semester as a week grid, sections A and B side by side |
| Courses | The courses of a batch's running semester with their credits and teachers |
| Exams | Upcoming exams with syllabus and instructions |
| Materials | The batch Drive links the CR keeps |
| Notices | Office and CR notices as date blocks |
| Students | The directory of their batch |
| Teachers | Teacher contact details |
| Polls | Open polls for them, with their own answer marked |
| Messages | Direct chats and group chats |
| My profile | Their ID card, details, photo, drop courses and security settings |

A student with a drop course sees that course's classes from the batch that
actually runs it. Drop courses are chosen in **My profile → My drop courses**,
a searchable picker over the running-semester courses of the other batches, and
are never asked for at registration.

## 6. Class representative pages

Everything a student has, plus:

- **Create class & exam** with the create form, the room availability grid, the
  upcoming routine of the batch and the final routine builder.
- **Notices** for their batch.
- **Materials** — the Drive links their batch reads.
- **Polls** they open and close.

## 7. Teacher pages

- **Next day** — tomorrow's classes with Confirm, Cancel, Request reschedule,
  and accepting or declining a CR's proposal, under the decision window banner.
- **Final routine** — what the class representatives published for their classes.
- **Fixed routine** — their weekly routine.
- **Create exam** — the course, batch, section, date, syllabus and instructions;
  the class representative sets the time and the room.
- **Notices** for a batch or a section.
- **Messages**, including starting group chats.

## 8. The office portal

- Registration approvals.
- Students, class representatives and teachers, with removal, restoration and
  permanent deletion.
- **Courses** — code, title and credit, with deletion when nothing uses the
  course.
- **Routines** — one batch and semester at a time, sections A and B side by
  side, with the running semester of the batch.
- **Class periods** — the teaching day.
- **Fixed routines**.
- **Notices**.
- **Messages**.
- **Activity log** — every removal, restoration and permanent deletion.
- **Overview** — the department numbers.

### 8.1 Courses

A course carries a code such as `CSE 2243 7637D`, a title, and a credit between
0.5 and 6.0 in steps of 0.25. The credit is required for a new course; courses
that existed before the rule carry 3.0. A course is deleted only when no routine
entry, schedule item or drop course row uses it; otherwise the answer is `409`
and names the counts.

### 8.2 The running semester

The Routines page carries a semester picker and a **Running semester** control
per batch. Moving a batch forward changes what the scheduling engine generates
from the next run, while the old entries stay readable under their own semester.

### 8.3 Teacher User IDs

A teacher chooses their own User ID when they accept the invitation. It is 4 to
20 characters, starts with a letter, uses letters, digits, dots, underscores and
hyphens, is never digits only, is unique without regard to case, and a short list
of names such as `admin` is reserved.

### 8.4 Record retention

Removing an account hides it from the site but keeps the rows other people rely
on: notices stay readable, messages stay in their threads, published routines
keep the names they were published with. The office may restore the account or
delete it permanently, and both are written to the activity log.

## 9. Messaging

### 9.1 Direct chats

A direct chat has two members and follows the rules in 2.4.

### 9.2 Group chats

Only teachers and class representatives create groups; a student or the office
asking for one gets `403`.

- A group has a title, members, and member roles of `admin` or `member`.
- The creator starts as a group admin. Every member posts.
- A group admin adds and removes members, promotes and demotes admins, and
  renames the group. A group never loses its last admin.
- Anyone may leave; if the last admin leaves, the earliest member becomes admin.
- A CR adds whole batches only for their own batch; a teacher adds any batch.
  Individuals may be any active student, CR or teacher.
- Only members see a group and post in it, and a message notifies every member
  except the sender.

### 9.3 Deleting a chat

Any member may delete a direct chat or a group chat **for themselves only**. The
moment is recorded, the messages up to it stop showing for that person, and the
chat leaves their list and their unread count. Everybody else keeps the whole
history, and in a group the person stays a member. When a new message arrives,
the chat comes back carrying only the messages sent after the deletion.

### 9.4 Editing and deleting

The sender may edit the body of their message, or delete it, within two minutes
of sending; after that the answer is `409`. An edited message shows **Edited**
next to its time. A deleted message shows **This message was deleted**, and its
body, link and file are cleared with the file removed from disk. Edit and Delete
appear only while the two minutes last.

## 10. Polls

A class representative opens a poll with a question, two to ten options, an
audience and a limit on how many options a student may choose.

- The audience is their own section, their own batch, or every batch.
- "Each student may choose up to N options", where N is between 1 and the number
  of options.
- A student of the audience votes once with between 1 and N distinct options of
  that poll. Voting again answers `409`.
- A response carries the count per option, the number of voters, and the
  caller's own choices. **No endpoint ever returns who voted for what** — not to
  the class representative, not to a teacher, not to the office.

The creator keeps the poll afterwards. `PATCH /api/cr/polls/{id}` rewrites the
question, the options and the limit while the poll has **no votes** and answers
`409` once anyone has voted. `DELETE /api/cr/polls/{id}` removes a poll at any
time, and `POST /api/cr/polls/{id}/close` and `/reopen` flip its status. An
edited poll carries `edited_at` and the pages show *Edited* with the time.

## 11. Registration decisions and notifications

### 11.0 Approval and rejection emails

Approving a student registration emails them a welcome with the registration
number to sign in with, the website and the Android download link. Rejecting
needs a reason of at least 5 characters, offered as quick picks or typed; the
reason is emailed with what to do next, kept on the account and shown on the
sign-in screen. Both mails are sent in the background and a failure is logged
without changing the decision.



### 11.1 In-app

A bell in the masthead carries the unread count and opens the recent list. Every
notification knows the page it belongs to, so opening one lands on the item,
highlighted.

Notifications are created when a notice is posted, a class or exam is created,
changed or cancelled, a reschedule is asked for or decided, a routine is
published, a poll opens, and when an undecided class is marked **No response**.

**Messages never reach the bell.** A new message still pushes, and it raises the
Messages unread badge on the sidebar, on the bottom tab bar and on the installed
app icon through `navigator.setAppBadge`. Every bell row shows a date and a
time: *Today 4:05 PM*, *Yesterday 9:10 AM*, *28 Oct, 9:10 AM*.

### 11.1.1 Section badges

Notices, Materials, Polls, Exams, Routine and Courses each carry a count of the
items added since the reader last opened that section. `section_views` keeps one
row per user and section. `GET /api/me/badges` returns the six counts and
`POST /api/me/badges/{section}/seen` clears one, which every section page calls
as it opens. The counts appear on the sidebar rows, the bottom tabs and the
quick-action tiles.

### 11.1.2 Teacher reminders

Teachers with undecided classes are reminded by push and in the bell:

- at each `TEACHER_REMINDER_TIMES` slot for tomorrow, and
- at `TEACHER_MORNING_REMINDER` for today's classes that have not started.

`reminder_log` holds one row per teacher, slot and day, so no reminder goes out
twice. Each one links to the teacher's next-day page.

### 11.2 Browser push

Students, class representatives and teachers may turn on push notifications; the
office never does.

- `VAPID_PUBLIC_KEY`, `VAPID_PRIVATE_KEY` and `VAPID_SUBJECT` configure it. With
  the keys missing, push is skipped and the rest of the site is unaffected.
- A device subscribes once and the subscription is stored with its endpoint.
- When a notification is stored, a bounded background worker queues a push to
  every subscription of its recipients. The payload carries a title, a body, the
  URL of the page the notification points at and a tag, with a 24 hour TTL.
- A subscription that answers `404` or `410` is deleted.
- Permission is asked inside the sign-in click itself, so the phone shows its
  own dialog. If it could not be shown, the app asks once more on the next tap
  and never again after a decision.
- When permission is already `granted`, the app subscribes silently on sign-in
  and on every app start without asking anything.
- The server key is trimmed, read as base64url and checked to be a 65 byte
  uncompressed point, so a key pasted with a stray newline fails loudly instead
  of silently never subscribing.
- `POST /api/push/client-log` takes `{stage, name, message}` from a signed-in
  device, rate limited per IP, so a failure that only happens on a phone is
  visible in the server log. It never carries a key or an endpoint.
- Signing out removes the subscription row for that account but keeps the
  browser subscription, so signing back in re-registers at once and a sign-out
  can never delete a subscription that was created moments earlier.
- My profile carries no notification section at all; the system dialog is the
  only thing that asks.
- A push carries `{title, body, url, tag}` with `Urgency: high` and a 24 hour
  TTL, and its url is the page for the recipient's own role. Broadcasts resolve
  to user ids with their roles and queue once per distinct page.
- `POST /api/push/test` sends to every subscription of the caller and answers
  how many went out. Each failure is logged with the push service host and the
  status code, never a key and never a full endpoint. The server logs once at
  startup whether the VAPID keys are configured.

### 11.3 Installable site

`/.well-known/assetlinks.json` is served as `application/json` without a
redirect, so Android can verify the installed app, drop the URL bar and accept
the delegated notifications. A web manifest and a service worker make the site
installable. The app registers
`/sw.js` at scope `/` on every start and re-syncs its subscription on start and
on each visibility change, subscribing again whenever the server key changed.
The service worker always shows the notification and, on click, focuses an open
window and navigates it, or opens a new one. The server serves `/sw.js` as
`application/javascript` with `no-cache` and `Service-Worker-Allowed: /`, and
the manifest as `application/manifest+json` with `no-cache`.

### 11.4 Version and updates

`APP_VERSION` and `APP_COMMIT` travel through the Docker build into both the
pages and the Go binary. `GET /api/health` returns them and **My profile →
About** shows them.

On start, when `APP_VERSION` differs from the version in `app_meta`, the server
pushes once to every subscription: *SUST CSE Portal updated to `<version>` —
open the app to get the latest*. The row in `app_meta` makes it once per
version, no matter how often the service restarts.

In the browser the waiting service worker raises *A new version is ready ·
Reload*; the app checks for an update on start and whenever it comes back to the
foreground.

### 11.5 The human check

With `TURNSTILE_SITE_KEY` and `TURNSTILE_SECRET_KEY` both set, a full-screen
glass gate with the SUST logo and a Cloudflare Turnstile widget stands in front
of the app until the visitor holds a valid pass.

- `POST /api/human/verify` checks the token with Cloudflare siteverify and
  returns a signed pass, valid 24 hours in a browser and 30 days in the
  installed app. The app keeps it in `localStorage`.
- Sign-in, registration, email verification, forgot and reset password,
  activation, the user-id check and the public stats all require that pass in
  `X-Human-Pass` and answer `403 Please complete the human check.` without one.
  The app clears the stored pass on that answer and shows the gate again.
- The site key comes from `GET /api/auth/config`. **With either key missing the
  gate is skipped entirely** and every one of those endpoints works as before.

## 12. Class attendance

Attendance is taken by Bluetooth from the Android app and is built so that one
student cannot mark another. Teachers create the session; students only mark
themselves, with one tap, from the phone they registered. There are no QR codes
and no selfies.

### 12.1 The session

A teacher starts attendance for one course, batch and section of the running
semester, prefilled from today's class. Section `ALL` covers both sections. The
session stays open for a duration the teacher picks between **5 and 30 minutes**
and can be extended by the same amount or closed early. Starting one sends every
roster student a push, `Attendance open · <course title>`, that opens the mark
screen.

Each session draws its own random `token_key`, which never reaches a student.

### 12.2 The Bluetooth presence token

The teacher's own app is the only client that receives the session's
`token_key`, and only from `POST` start and `GET` of that teacher's own open
session. It never appears in a list, in a closed session or in anything a
student can read. A shared test vector — a fixed key, session id and slice
against the expected eight bytes — is checked by a Go test and a JavaScript
test, so the server, the client and the native plugin cannot drift apart.


While the session is open the teacher's app advertises a Bluetooth Low Energy
packet on a fixed service UUID carrying an 8-byte token:

```
token = HMAC(token_key, "<session id>:<unix seconds / 10>")[:8]
```

The server accepts the **current or the previous** 10-second slice, so a token
heard a moment ago still works and one from a minute ago does not. The
advertisement is low power, which reaches roughly 10 to 20 metres — about one
classroom.

When a phone has no Bluetooth, or hears nothing within ten seconds, a 4-digit
code derived the same way over **20-second** slices is shown on the teacher's
screen and on the website for a projector, and the student types it. The
location rules are unchanged.

### 12.3 The location rules

Location is asked for and reported on its own, never behind Bluetooth. The
session screen requests precise location when it is missing, offers the app
settings screen once Android stops asking, checks that the device location
switch is on and offers the location settings screen when it is not, and shows
the real state: *Allowed*, *Off*, *Denied* or *Sending · accuracy N m*. When
Bluetooth cannot broadcast the screen names the reason and points at the
4-digit code, which never depended on Bluetooth.


The teacher's phone sends its GPS position when the session starts and every 15
seconds while it is open, from a foreground service, so a locked screen does not
stop it. A mark is accepted only when all of these hold:

- the student is within **30 metres** of the teacher's latest position, by
  great-circle distance;
- the student's accuracy is **50 metres or better**, otherwise *Weak GPS signal
  — move near a window and try again*;
- the position is not from a mock provider, checked natively with
  `Location.isMock()` or `isFromMockProvider()`;
- the teacher's own position is **less than 60 seconds** old.

The distance and the accuracy of every accepted mark are stored, so a session can
be audited afterwards.

### 12.4 One phone, one student

One phone marks one student per session and a student marks once per session,
both enforced by unique indexes rather than by the app. Marks are only taken
while the session is open.

The roster is the active students of the batch and section — `ALL` meaning both —
plus every student who added that course as a drop course. On close, every roster
student without a record becomes **absent** with method `system`.

### 12.5 The registered phone

The first Android phone a student or class representative signs in with becomes
their attendance phone, identified by `Settings.Secure.ANDROID_ID`, which Android
scopes to the app. Every other feature works on any device and on the website;
only attendance is refused elsewhere, with *This is not your registered phone.
Request to use this phone.*

From a new phone the student sends a request with a reason. It reaches every
teacher who teaches that student's batch in the running semester, as a push and a
bell notification reading *<name> (<reg no>) wants to use a new phone*, and any
one of them can approve or refuse it from **Phone requests** in the teacher app
and on the website. An approval rebinds the phone, tells the student who approved
it, and is written to the activity log. Office admins do not approve these; they
only see the history.

At most **two approved changes per student per semester** need no note. Beyond
that the teacher has to add a note to approve.

### 12.6 Marking and editing

The mark screen asks once for Bluetooth, nearby devices and precise location,
offers to turn Bluetooth on, enables one big **I'm present** button only when a
valid token is heard, and then asks for the phone's own biometric or screen-lock
confirmation before sending `{token or code, lat, lng, accuracy, is_mock,
device_id}`. Success and every refusal say exactly what happened.

Students cannot mark from the website, which explains to use the Android app.
Students on an iPhone are marked by hand by their teacher. Every manual change
keeps a note, who made it and when.

A hundred or more students marking at once is normal: the teacher's phone only
broadcasts, and the server handles each mark on its own.

### 12.7 Percentages and the warning

The percentage of a student in a course is `present / sessions held for that
student`. When it falls below **50 %**, after at least three sessions, the student
gets one push and bell notification, *Your attendance in <course title> is <n> %
— below 50 %*, and then at most once a week while it stays below. It links to
that course's attendance.

### 12.8 What each role sees

- **Students** — the Attendance tab, a Home quick action and the push open the
  mark screen. **My attendance** lists each course with its percentage, red below
  50 %, and every session date with Present or Absent, the time, the teacher's
  name and any note, filtered by course and month, in the app and on the website.
- **Teachers** — start, extend and close a session; the live count, the latest
  marks and **Show code**; the session detail with every student's status, time,
  distance and method, each settable to Present or Absent with a note; the
  history; **Phone requests**; and the Excel export. The course list shows each
  course's average attendance and the students below 50 %.
- **The office** — a read-only phone change history and an attendance overview
  per batch and course with the same Excel export across every batch.

### 12.9 The Excel export

Generated on the server with `excelize`. One sheet per course-batch-section
carries the student name, the registration number, one column per session date
with `P` or `A` and the note as a cell comment, the present count, the total and
the percentage, with below 50 % highlighted red and the header frozen.
**Download all my classes** puts every course-batch-section the teacher teaches
into one workbook and adds a Summary sheet with the course, batch, section,
number of sessions, average percentage and how many students are below 50 %. A
date range filter defaults to the running semester.

## 13. Interface

- A frosted glass look on the department's own colours: maroon `#7A1F2B`, paper
  `#FAF9F6` and a muted gold for highlights. A soft background of blurred maroon
  and gold blobs sits behind translucent cards, bars, sheets and the sidebar,
  with a solid fallback wherever `backdrop-filter` is unsupported. Inter for interface text, Source
  Serif 4 for large titles and the home greeting.
- A dark theme that follows the system setting, with Light, Dark and System in
  **My profile → Appearance**.
- A translucent top app bar whose compact title appears as the large title
  scrolls away, a translucent bottom tab bar with five role tabs and unread
  badges, bottom sheets instead of modals on a phone, segmented controls, chips,
  list rows, skeletons, empty states, toasts above the tab bar and
  pull-to-refresh on lists.
- From 1024px the same components sit in a sidebar layout, and the office keeps
  its desktop layout.
- Every route and link goes through one paths module.
- Readable at 360px wide, touch targets of at least 48px, `env(safe-area-inset-*)`
  respected, no horizontal scroll, AA contrast, and transitions that respect
  `prefers-reduced-motion`.
- No admin link or admin wording on any non-admin page.
- The only logo is `frontend/public/brand/sust-logo.png`, with a monogram
  fallback. `npm run icons` generates the app icons, the maskable icons, the
  monochrome badge, the apple-touch icon and the favicon from it.

## 14. Demo data

`go run ./cmd/seed -demo` loads a demonstration department: batches CSE-21 to
CSE-24 with sections A and B, each on its own running semester with an older one
behind it, courses with real-looking codes and credits, teachers with chosen
User IDs, students and class representatives, a fixed routine on the class
periods, schedule items in every state, a published final routine, teacher
exams, notices, polls, group chats and messages.

It refuses to run while `APP_ENV` is `production`, it is safe to run twice, every
account uses `DEMO_PASSWORD` from `backend/.env`, and it prints the identifiers
only — never the password.

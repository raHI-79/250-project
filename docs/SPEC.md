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
| `super_admin` | The one owner of the portal | A hidden control room |

`cr` is not a separate account. It is a student account with `is_cr` set, and the
backend resolves it to the effective role `cr` on every request.

`super_admin` sits above `admin`. There is exactly one: the account seeded from
`ADMIN_IDENTIFIER`, or from `SUPER_ADMIN_IDENTIFIER` when that is set. It is the
only role that reaches `/control-room`, it cannot be suspended or removed from
inside the portal, and it opens every office page read-only.

A teacher can be given office access. That teacher then signs in at
`/office-portal` with their own credentials and holds the effective role `admin`
for that session, while their teacher account, their teacher pages and their app
access stay exactly as they were.

## 2. Access control

### 2.1 The four parts of the site

- The **public home page** carries the department numbers and the two entrances.
- The **student and CR panel** is for registration numbers.
- The **teacher panel** is for teacher User IDs.
- The **office portal** is hidden. It is never linked from a student, CR or
  teacher page, and it is excluded from search engines.
- The **control room** at `/control-room` is hidden the same way: nothing links
  to it, it carries `noindex,nofollow`, its sign-in form passes the same human
  check as the others, and the Android app redirects away from both it and the
  office portal. Office accounts are rejected from the app by the backend as
  well, so a token is not enough.

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
- A student registers on **one page** — Account, Contact, Personal and an
  optional Photo, each field validated next to itself — verifies their
  university email with *Continue with Google* or the 6-digit code that appears
  in place after *Send code*, and waits for the office to approve the account.
- A teacher is invited by the office, opens the invitation link, chooses a User
  ID and a password, and the account becomes active. The teacher sign-in screen
  offers no way to reach that page; the link only arrives in the invitation
  email.
- Password reset is **one page** as well: the ID or email and the date of birth,
  then the emailed code and the new password, on a short-lived code.
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
ahead. It skips the weekly off days and the holidays, and fills a working day
with the routine of the weekday the office chose for it (see 4.7). Every item
carries a status:

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

What a role may do with an item is decided by one rule on the server, from the
role, the status, the origin and whether the item is in a final routine. Every
view returns the answer as `allowed_actions`, and the website and the app draw
only those buttons:

- an item outside a final routine offers **Add to final** and never **Remove
  from final**; an item inside one offers **Remove from final** and never **Add
  to final**;
- a cancelled item offers neither; the class representative takes it out of the
  final routine by selecting it there;
- a day that has passed offers nothing.

### 4.7 Off days, holidays and working days

The weekly off days default to Friday and Saturday. The office portal and the
control room both keep them and a list of dated entries:

| Entry | Effect on the generator |
|---|---|
| A weekly off day | No fixed routine is generated |
| A holiday, with a name | No fixed routine is generated |
| A working day that follows a weekday | The routine of that weekday is generated, such as a make-up Saturday that follows Sunday |

An off day or a holiday only stops the automatic fixed routine. A class, lab,
makeup class or exam that a class representative or a teacher creates for such a
date is kept, published and announced as on any other day; creating one shows a
small *This is a holiday* note and never an error. Adding a holiday removes only
the generated items nobody has decided on yet — still waiting for their teacher,
in no final routine and never published — and tells their teacher and class
representatives. Reminders fire only for undecided items that exist, so a class
created on a holiday is reminded about like any other.

Every routine view puts one chip at the top of such a day: *Holiday · `<name>`*
for a holiday, *Holiday · `<weekday>`* for a weekly off day and *Working day ·
follows `<weekday>`* for a working day. The classes and exams of that date are
listed under the chip with their usual actions; only a day with nothing on it
shows the chip alone. The chip reaches the hub, the final routine, the teacher
views, the student routine and the student Today page.

### 4.8 Dates and the time zone

Every date the server works with is a Dhaka date. The pool pins each database
session to `APP_TIMEZONE`, so `CURRENT_DATE`, `now()` and every cast between a
date and a stamp land on the Dhaka day rather than the day of the database
server; between six in the evening and midnight that used to be the day before.
Stamps compared against each other are written by the same clock: the password
reset deadlines, the conversation read and clear marks and both ends of an
attendance window are all written by the database. Dates travel as
`YYYY-MM-DD`, every date the portal shows reads `Fri 6 Nov 2026`, and every date
field asks for the day, the month and the year in that order.

## 5. Student pages

| Page | Content |
|---|---|
| Today | Today's published classes, exams and notices in one timeline, today only |
| Upcoming routine | The published routine of any date, with a date picker and a *Back to today* link |
| Fixed routine | Any batch and semester as a week grid, sections A and B side by side |
| Courses | The courses of a batch's running semester with their credits and teachers |
| Exams | Every exam ahead with syllabus and instructions, narrowed to one date with the date picker |
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

- **Class & exam hub**, ordered *Upcoming routine · Final routine · Create class
  / exam · Room availability* and opening on the upcoming routine. The upcoming
  tab opens on tomorrow with a `Tomorrow | Today` toggle, a date picker and a
  *Show more days* button; the final routine holds only what the representative
  added with *Add to final* and reads any date too. Creating a class picks the
  teacher of that course for the batch and section by itself, editable only when
  the course has several teachers, and the class then waits for that teacher like
  a generated one.
- **Notices** for their batch.
- **Materials** — the Drive links their batch reads.
- **Polls** they open and close.

## 7. Teacher pages

- **Upcoming** — opens on tomorrow with a `Tomorrow | Today` toggle, a date
  picker and *Show more days*, offering Confirm, Cancel and Request reschedule
  and accepting or declining a CR's proposal, under the decision window banner.
  Today's confirmed classes sit above it on their own. Any class or exam the
  teacher owns can be edited until it starts — the date, the start and end time,
  the room and the mode — and its meeting link until it ends; each change
  re-checks the room and batch clashes, updates the published copy and tells the
  class representative and the students. A past date is read-only and shows
  Held, Cancelled, Rescheduled or Exam held.
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

**Read state lives on the server.** Each notification carries a read flag for the
signed-in person. Opening one from the bell, following its link from anywhere, or
tapping its push marks it read and lowers the bell at once; if the request fails
the count goes back. **Mark all as read** sets a time before which everything is
read. A badge shows the exact number up to 99 and **99+** above. A notice link
opens `/student/notices/<id>`, which loads that notice, scrolls to it and marks
the section seen; a removed notice, exam or poll says it is no longer available
and links to the full list.

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

A web manifest and a service worker make the site installable on iPhone and the
desktop. On Android the site points at the app instead, which is a Capacitor
build that needs no digital asset link. The app registers
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
themselves, with one tap, from the phone they registered. A teacher who would
rather read the roster aloud takes a **roll call** instead, and a student at a
computer marks themselves on the **website** with the code on the board and
their browser position. There are no QR codes and no selfies.

Every role reaches all of this through one **Attendance** item in the
navigation, with tabs inside it: *Take attendance · History · Sheets · Phone
requests* for a teacher, *Mark attendance · My attendance · My phone* for a
student or class representative, and *History · Sheets · Phone changes* for the
office.

### 12.1 The session

A teacher starts attendance for one course, batch and section of the running
semester, prefilled from today's class, and chooses the method: **Bluetooth** or
**Roll call**. Section `ALL` covers both sections. The session stays open for a
duration the teacher picks: the quick chips **5, 10, 15, 20 and 30 minutes**,
**Custom** with any number of minutes from **1 to 240**, or **Open until** a time
later today. **Extend** works the same way, with quick **+5** and **+10** and a
custom number of minutes, and the session can be closed early. The server checks
that the end is after the start and within **four hours** of it, and answers with
a clear message otherwise.

The teacher also chooses the **class hours** of the session, 1 to 4, prefilled
from the routine item's length (up to one hour is 1, more than one and up to two
is 2, and so on) and editable; a session started before this rule counts as 1.
For a digital session the teacher sets when a mark becomes **late**: *at the
original end*, the default, or *late after N minutes* from the start, any number
up to the end. Starting a Bluetooth session sends every roster
student a push, `Attendance open · <course title>`, that opens the mark screen; a
roll call pushes nobody, shows no code and is never offered to a student, because
the teacher marks the roster.

A roll call shows the roster with each student's name, registration number and
photo, three compact buttons — **P**, **L** and **A** for present, late and
absent, each at least 44 px, unmarked until one is tapped and filled afterwards — a **Mark all present** shortcut that leaves
the marks the teacher already set, and a note per student with its own Save
button that says when it is saved. **Finish** asks for a confirmation and then
marks everybody still unmarked absent. The same per-student P, L, A and note
editing works during and after a Bluetooth session and from the history at any
time, and every change is written to the audit log.

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

The same code is how the **website** marks a student. A browser cannot bind a
phone and cannot be asked about a mock provider, so a web mark needs the code
plus the browser's own position under the same rules — within 30 metres of the
teacher, accuracy 50 metres or better, and a teacher position less than 60
seconds old. Such a mark is stored with method `web` and shown with a **Web**
label in the teacher's list, so the teacher can change it like any other mark.
Registering a phone and asking for a new one stay in the app.

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

A mark is **present**, **late** or **absent**. A digital mark (Bluetooth, code or
web) made up to the late boundary is present and one made after it is late. The
boundary is the session's original end unless the teacher set *late after N
minutes*; so in a session that was never extended nobody can be late by time,
and once it is extended every mark after the original end is late. The server
decides this at the moment the mark is saved and tells the student which it was.

### 12.7 Percentages and the warning

Every session is worth its class hours. A student's **credit** for a session is
the hours when present, the hours times the **late weight** when late, and 0 when
absent. The late weight is a site setting, **0.5** by default so that two lates
equal one class, editable in the control room (Site settings) and in the office
portal (Attendance, Sheets). The percentage of a student in a course is
`credit / total hours of the sessions held for that student`. For example, with
the default weight, a 2-hour class present, a 3-hour class late and a 1-hour
class absent give `(2 + 3 × 0.5 + 0) / (2 + 3 + 1) = 3.5 / 6 = 58 %`. The 50 %
alert, the home stat, My attendance, the teacher's course list, the dashboard and
the Excel export all use this number.

When it falls below **50 %**, after at least three sessions, the student
gets one push and bell notification, *Your attendance in <course title> is <n> %
— below 50 %*, and then at most once a week while it stays below. It links to
that course's attendance.

### 12.8 What each role sees

- **Students** — the Attendance tab, a Home quick action and the push open the
  mark screen. **My attendance** lists each course with its percentage, red below
  50 %, its credit over the classes held and its late count, and every session
  date with Present, Late or Absent, the class hours, the time, the teacher's
  name and any note, filtered by course and month, in the app and on the website.
- **Teachers** — start, extend and close a session by Bluetooth or roll call;
  the live count, the latest marks with their method and **Show code**; the
  session detail with every student's status, time, distance and method, each
  settable to Present, Late or Absent with a note; the history; **Sheets** with the
  Excel export and the delete; and **Phone requests**. The sheet list shows each
  course's average attendance and the students below 50 %.
- **The office** — the history of every session, the sheets with the same Excel
  export across every batch and the same delete, and a read-only phone change
  history.

### 12.9 Deleting a sheet or a session

A teacher, for their own classes, and the office, for any class, can delete a
whole attendance sheet or a single session from **Sheets**. The delete first
asks *Are you sure you want to delete the attendance sheet for `<course>` ·
`<batch> <section>`? This cannot be undone.* with **Cancel** and **Delete**.
Every delete is written to the audit log with who did it and which sheet or
session it was. The marks go with the session, so students stop seeing it and
every percentage is recounted from what is left.

### 12.10 The Excel export

Generated on the server with `excelize`. There is **one sheet per course, batch
and section**, and the Bluetooth, code, web and roll-call sessions of that class
all land in it. The sheet carries the student name, the registration number, one
column per session date with `P`, `L` or `A` (late in its own amber colour) and
the note as a cell comment, then **Classes held**, **Attended** (the credit),
**Lates** and **Percentage**, with below 50 % highlighted red and the header
frozen. A **Class hours** row under the students gives the hours of each session
and their total.
**Download all my classes** puts every course-batch-section the teacher teaches
into one workbook and adds a Summary sheet with the course, batch, section,
number of sessions, average percentage and how many students are below 50 %. A
date range filter defaults to the running semester.

### 12.11 Students who join a course later

A student who is approved later, moved into the section, or adds the course as a
drop course in the middle of the semester is counted from the start of the course
in the running semester. Every session of that course, batch and section that was
held before they joined, since the batch's running semester began, counts for
them as **absent**, with the method *system* and the note **Joined later**. They
appear in the roster, the history and the Excel export of those sessions with
`A`, and their percentage and the 50 % alert use every session held since the
semester started. A teacher of the course can still change any of those entries
to present or late with a note, audited like any other edit.

The missing records are created when a registration is approved, when a drop
course is added, and by the scheduler on every run, which also covers a student
moved between sections; the migration that introduced the rule filled them in for
the students who already existed. Only sessions that hold at least one real mark
are counted.

## 13. The control room

### 13.1 Signing in

The super admin signs in at `/control-room`, which is a separate entrance from
`/office-portal`: the office form refuses the super admin and the control-room
form refuses everybody else. Both refuse the Android client outright.

### 13.2 Office admins

**Control room → Office admins** lists every office account and every teacher
who holds office access, with its status, its last sign-in and the number of
actions it has taken. The super admin can:

- invite an office admin by email through the existing mailer. The link lives
  for 72 hours and lands on `/office-portal/accept-invite`, where the invitee
  picks a username under the teacher User ID rules and a password;
- resend or cancel an invitation that is still waiting;
- suspend and restore an office admin, which also invalidates their tokens;
- remove an office admin after typing its username to confirm;
- grant or revoke office access for any active teacher at any time.

Every one of those actions is written to the activity log.

### 13.3 Read-only office pages

The control room links to every office page. The super admin can read them all,
and the backend refuses any write from that role with *The control room opens
the office pages read-only.* Office admins are unaffected.

## 14. History and analytics

### 14.1 The activity log

`audit_events` holds one row per recorded action: the actor and their role, the
action, the target type and id, a summary, the IP address hashed with
`ANALYTICS_SALT`, the user agent, whether it came from the website or the app,
and the time. Every create, update, delete and publish of every role is written
by one middleware that runs after the token check and records only answers in
the 2xx range, and every sign-in, sign-out and failed sign-in is written by the
sign-in handlers themselves. Account removals, restores and permanent deletions
keep their own readable summary and are written once, not twice.

The activity log page filters by person, role, action, client and date range,
searches the summaries, loads more as the page scrolls and exports the current
filter as CSV. From the control room each row offers **Monitor this admin**,
which opens that account's own timeline with its counts per action type. Office
admins keep the same page for the office-wide log.

### 14.2 Page views

Every page opens a privacy-friendly beacon: a visitor id kept in
`localStorage`, a session id kept in `sessionStorage`, the path, the client, the
device kind and the app version, batched and sent to
`POST /api/analytics/view`, which is rate limited per address. No account id and
no readable IP address is stored. An hourly job rolls the rows up into
`analytics_daily` and deletes raw rows older than 90 days.

### 14.3 The dashboard

`/control-room` is the dashboard. A date range picker offers Today, 7 days,
30 days, Semester and a custom range, the numbers refresh themselves every
minute and the header carries the time they were last read.

- **Overview** — KPI cards for unique visitors, page views, active users today,
  sign-ins, new registrations and pending approvals, each with a sparkline and
  the change against the previous period; a line chart of visitors and sign-ins
  per day; a bar chart of sign-ins by role; a donut of website against app; the
  app versions in use; the top pages; the devices.
- **Academic** — attendance average per batch and section, how many students sit
  below 50%, classes held against cancelled against rescheduled per course,
  tomorrow's undecided classes per teacher, pending phone change requests, and
  the notices, polls and messages of the period.
- **System health** — server uptime, version and commit, database size,
  uploaded file storage, the requests of the period, 4xx and 5xx counts, the
  addresses that answered with a server error with their count, their last
  message and when it was, the latest five 5xx answers, the notification failure
  share with the failures broken down by channel and by reason, the token
  counts, emails sent and failed, and the time of the last successful database
  backup.

A 5xx counts only when one of our own handlers answered it while the server was
running normally: the first minute after a start, the platform health probes, the
503 the maintenance gate writes on purpose and a request the caller abandoned
before the answer, whose database call was cancelled under it, are left out. A notification
counts by what the push service actually answered. A subscription the push
service has thrown away — 404, 410 or 403 for web push, UNREGISTERED, NOT_FOUND
or INVALID_ARGUMENT for the app — is dropped at once, and the nightly sweep drops
the web push subscriptions nothing has used for sixty days.

An alerts banner sits at the top for:

| Alert | Fires when |
|---|---|
| Server errors | at least 10 server errors among at least 200 requests in the period |
| Email failures | more than a fifth of at least 5 emails failed |
| Notification failures | more than a fifth of at least 20 sends failed, leaving out the expired subscriptions the server cleaned up |
| Backup | the last successful backup is older than 36 hours |
| Sign-in spike | one address failed to sign in at least 10 times in the last hour |
| Deleted sheet | an attendance sheet was deleted in the last day |

Each alert carries one icon, a **Details** link to its System health section and
**Dismiss for a day**, which the server stores so the alert stays quiet on every
device the super admin signs in from. Every number carries a label and a
tooltip, and each panel has its own empty state.

### 14.4 The backup status endpoint

`POST /api/system/backup-status` records the time of a successful database dump.
It is protected by the `X-Backup-Secret` header, compared against
`BACKUP_STATUS_SECRET`; anything else is refused with `403`. The nightly backup
workflow calls it after it uploads the dump.

## 15. The control centre

### 15.1 Sessions and the confirmation window

Every sign-in opens a row in `user_sessions` holding the device, the client, the
hashed address, the country the address resolves to, when it started and when it
was last seen, and the token carries its id. The token check refuses a revoked or
expired session, so signing a device out ends it at once. **Security** lists
every session of the signed-in account with *Sign out* per row and *Sign out
everywhere else*; changing a password signs every other session out.

Every sensitive action in the control room asks for the password again. The
confirmation opens a five-minute window on that one session; a write without it
answers `428` with *Please confirm your password first.* Each confirmed action
also shows a dialog naming its target and is written to `audit_events`.

### 15.2 Office account security

- An office password is at least 10 characters with a letter and a digit.
- Two-step sign-in is optional per account: after the password the portal mails a
  6-digit code that lives 10 minutes and allows five tries.
- The contact email and phone change only after the password is given again.
- `/control-room/forgot-password` resets the super admin password against the
  phone number on the account, rate limited and recorded.
- `go run ./cmd/admin -reset-super` replaces the super admin password from
  `NEW_SUPER_ADMIN_PASSWORD` in the environment, closes every session and prints
  only *Password updated*.
- Five wrong passwords lock an account for 15 minutes. The control room lists the
  locked accounts and unlocks them, and an office sign-in from an unknown device
  emails the account owner.

### 15.3 Permissions

Each office admin and each teacher with office access holds a set of
permissions: `registrations`, `students`, `teachers`, `catalog`, `periods`,
`routines`, `notices`, `attendance` and `exports`. A new admin receives
everything except `exports`. One middleware looks the matched route up in a
table and answers `You do not have permission for this` when the permission is
missing; the office navigation hides what the admin cannot open, and a test
proves every office route appears in that table. The super admin bypasses the
table and stays read-only on the office pages.

### 15.4 Users

The control room searches every account by name, registration number, User ID or
email and filters by role, status and batch. A profile shows the role, status,
creation, last sign-in, open sessions, phone binding and recent activity, and
offers suspend and restore, sign out everywhere, reset the attendance phone, a
password reset email and switching a student between `student` and `cr`. A
selection can be suspended or signed out together, and the filtered list exports
as CSV.

### 15.5 Site settings

Settings live in a `settings` table, are cached for half a minute and need no
redeploy:

- **Maintenance mode** with a message. Everybody except the office and the
  control room sees a calm page, and the API answers `503` with that message.
- **Registration** open or closed with a message.
- **Announcement banner** for chosen roles with an end date and a severity.
- **Feature switches** for attendance, polls, messages, group chats and
  attendance from the website.
- **Reminder times**, the **morning reminder** and the **days ahead** the routine
  is generated, with the environment value as the default.
- **Weekly off days** and the **holiday list**, with working days that follow
  another weekday (see 4.7). The office portal edits the same calendar on its
  **Routines** page.
- **Minimum supported app version**, below which the app shows a blocking
  *Update required* screen.
- **Retention windows** for the page views, the activity events and the
  notifications.

### 15.6 Security monitor

Failed sign-ins by hashed address and by account over the last week, the accounts
locked by too many failures, and blocking or unblocking a single address or a
range for a chosen number of hours. A request from a blocked address is refused
with `403` and recorded.

### 15.7 The two-person rule

While at least one other active office admin exists, deleting an account for
good, deleting a whole attendance sheet and removing an office admin create a
request that another admin must approve within 24 hours; the request carries what
it will do and expires on its own. Nobody may approve their own request. The
super admin goes ahead immediately after confirming its password, and that
override is recorded.

### 15.8 Data

The data page shows the last backups reported by the nightly workflow, CSV
exports of the users, the attendance summary and the activity log, how many rows
the portal holds, and the retention windows. An hourly job deletes page views,
activity events and notifications past their window and drops closed sessions
after 30 days.

## 16. Interface

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
- In the Android app the sign-in, register, verify, forgot-password and teacher
  activation screens share one shape: a solid primary panel of about 330 px with
  the SUST logo on a white rounded square, the screen title and a one-line
  subtitle, and a white sheet overlapping it by 40 px with 28 px rounded top
  corners that carries the Student | Teacher segmented control, the labelled
  inputs, a full-width 52 px primary button, the *Forgot password?* link,
  *New student? Create an account* and the department and version line. Inputs
  are 50 px high and take a 2 px primary border with a soft tint ring on focus,
  the form scrolls inside the sheet, the keyboard never hides the focused input,
  touch targets stay at least 44 px and dark mode uses the dark tokens. The
  website keeps its own sign-in screens, and no sign-in logic or API call
  differs between the two.
- No admin link, admin wording, super admin link or control room wording on any
  non-admin page, and none of it inside the Android app.
- The only logo is `frontend/public/brand/sust-logo.png`, with a monogram
  fallback. `npm run icons` generates the app icons, the maskable icons, the
  monochrome badge, the apple-touch icon and the favicon from it.

## 17. Demo data

`go run ./cmd/seed -demo` loads a demonstration department: batches CSE-21 to
CSE-24 with sections A and B, each on its own running semester with an older one
behind it, courses with real-looking codes and credits, teachers with chosen
User IDs, students and class representatives, a fixed routine on the class
periods, schedule items in every state, a published final routine, teacher
exams, notices, polls, group chats and messages.

It refuses to run while `APP_ENV` is `production`, it is safe to run twice, every
account uses `DEMO_PASSWORD` from `backend/.env`, and it prints the identifiers
only — never the password.


## 18. The public website and its editor

### 18.1 What can be edited

**Control room → Website** lets the super admin change the public website without
code:

- **Header** — site name, tagline, logo (the SUST logo when empty) and extra menu
  links.
- **Home** — the hero line, title and text, the two call-to-action buttons, the
  campus slideshow (upload, reorder, caption, remove) and the About title and text.
- **Sections** — which home sections show and in which order: the numbers of the
  department, About, the notices preview, the Android app download and any custom
  section.
- **Footer** — address, phone, email, office hours, footer text, quick links and
  social links, or the whole footer hidden.
- **Sign-in and SEO** — the sign-in title and text, the page title, description
  and share image.
- **Pages** — public pages at `/p/<slug>`, each with its own title and SEO fields.

### 18.2 Blocks

Custom home sections and pages are built from the same safe blocks: a heading;
text with basic formatting (`**bold**`, `*italic*`, lines starting with `-` for a
list and `[text](https://…)` for a link, and nothing else); an image with alt
text; an image and text side by side; a button; a list of cards with a title,
text, date and image for news or events; and a list of questions and answers.
Text is parsed into elements by the page; nothing renders raw HTML, and `<` and
`>` are refused when saving.

### 18.3 Rules

Every text has a length limit. A link is `http://`, `https://` or a path of this
site. An image is one uploaded here or built in; an upload must be a JPG, PNG or
WEBP of at most 4 MB and 6000 px, and it is decoded and encoded again before it is
stored in the existing file storage. A slug is small letters, digits and dashes,
unique, and never a name the portal already uses (`student`, `teacher`, `cr`,
`office-portal`, `control-room`, `api`, `uploads`, `about-coordinet` and the
like). Deleting a page or a section asks for confirmation.

### 18.4 Draft, preview, publish and restore

Edits are saved as one **draft**. **Preview** opens the public pages with the
draft in a new tab, for the super admin only; everybody else keeps seeing the
published website. **Publish** makes the draft live and stores it as a new
version with who published it and when. **Restore** makes an earlier version live
again, as a new version, and puts it in the draft. Publishing and restoring need
the password confirmation and are written to the audit log.

The content is versioned JSON in `site_content_versions`. Public pages read it
from `GET /api/site/content`, cached for 30 seconds, and fall back to the built-in
content when nothing is published or the stored content is not valid.

### 18.5 The developer credit

Every public page footer, and the About area of the profile in the app, shows
**Developed by CoordiNet**, linking to an **About CoordiNet** page that lists
Ratul Islam and Rakibul Hasan. The credit is rendered by the code and is not part
of the editable content: the editor neither shows nor accepts it, the content API
refuses any field that tries to hide or replace it, and a hidden footer or custom
footer text never removes it.

## 19. Staying up

- **Error boundary.** Every route and every sheet sits in an error boundary. A
  screen that breaks shows *Something went wrong* with **Try again** and **Go
  home** instead of a blank page; the Android back button keeps working and
  closes an open sheet first. The name, message and route of the problem, and
  nothing personal, are sent to `POST /api/client-log`, written to the server
  log and listed under System health as *Screens that broke*.
- **Feature switches.** A switched-off feature answers **403** with the code
  `feature_off`, never a 5xx, and is not counted as a server error. The app reads
  the switches from the config, does not call or poll a switched-off feature,
  hides its menu item, tab, badge and home tile, and a direct visit says
  *<Feature> is turned off by the department*.
- **Idle connections.** A pooled database connection that sat idle is checked
  before it is used again and replaced when the database had dropped it.
- **Alerts.** The server-error alert fires only with at least 10 real server
  errors and more than 1 % of the requests of the period, and lists the top
  addresses with their counts. A deleted attendance sheet is information.
- **Header art.** Every blue header sizes the IICT art so the building and its
  nearest trees fit the header width, anchored centre bottom, with the sky
  blending into the header; wide desktop headers keep the full-width trees.

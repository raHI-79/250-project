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

### 4.3 The decision window (teachers only)

`DECISION_WINDOW_START` defaults to 16:00 and `DECISION_WINDOW_END` to 23:00.
The window limits **only a teacher's first decision** — Confirm, Cancel or
Request reschedule — on tomorrow's fixed classes.

The teacher's next-day page says:

- before the window: *Opens at 4:00 PM for tomorrow's classes*
- inside it: *Open until 11:00 PM · `<h m>` left*
- after it: *Closed for tomorrow — undecided classes are marked No response*

Outside the window those buttons are disabled for a first decision. A later
change to a class the teacher already confirmed stays allowed until it starts.

The class representative is never limited. Creating a class, asking for a
reschedule, setting a new slot, cancelling, adding to the final routine, editing
a link and publishing all work at any hour, and the CR pages carry no window
banner.

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
| My profile | Their ID card, details, photo and security settings |

A student with a drop course sees that course's classes from the batch that
actually runs it.

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

## 11. Notifications

### 11.1 In-app

A bell in the masthead carries the unread count and opens the recent list. Every
notification knows the page it belongs to, so opening one lands on the item,
highlighted.

Notifications are created when a notice is posted, a class or exam is created,
changed or cancelled, a reschedule is asked for or decided, a routine is
published, a message arrives, a poll opens, and when an undecided class is
marked **No response**.

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
- The site never prompts on load. After signing in, a dismissible banner offers
  **Get notifications on this device**, with help text when the browser has
  blocked permission, and **My profile → Security** carries the switch.

### 11.3 Installable site

A web manifest and a service worker make the site installable. The service worker
shows the notification and, on click, focuses an open tab and navigates it to the
target, or opens a new window.

## 12. Interface

- Serif display headings, a 16px base size, and the department colours: brand
  `#7A1F2B` on a `#FAF9F6` page.
- Every route and link goes through one paths module.
- Shared components: cards, buttons, fields, tables, badges, modals, toasts,
  empty states and skeletons.
- Readable at 360px wide, with touch targets of at least 44px.
- No admin link or admin wording on any non-admin page.
- The crest and the home page use `frontend/public/brand/sust-logo.png` and
  `cse-logo.png` when those files are present and fall back to a monogram when
  they are not; the favicon follows `cse-logo.png` the same way.

## 13. Demo data

`go run ./cmd/seed -demo` loads a demonstration department: batches CSE-21 to
CSE-24 with sections A and B, each on its own running semester with an older one
behind it, courses with real-looking codes and credits, teachers with chosen
User IDs, students and class representatives, a fixed routine on the class
periods, schedule items in every state, a published final routine, teacher
exams, notices, polls, group chats and messages.

It refuses to run while `APP_ENV` is `production`, it is safe to run twice, every
account uses `DEMO_PASSWORD` from `backend/.env`, and it prints the identifiers
only — never the password.

# Changelog

Every released version of the SUST CSE Class & Exam Coordination System. The
version in `frontend/package.json` and `render.yaml` is the one the app reports
and the one the Android release is tagged with.

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

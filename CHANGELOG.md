# Changelog

Every released version of the SUST CSE Class & Exam Coordination System. The
version in `frontend/package.json` and `render.yaml` is the one the app reports
and the one the Android release is tagged with.

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

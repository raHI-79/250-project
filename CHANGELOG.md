# Changelog

Every released version of the SUST CSE Class & Exam Coordination System. The
version in `frontend/package.json` and `render.yaml` is the one the app reports
and the one the Android release is tagged with.

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

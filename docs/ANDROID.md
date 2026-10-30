# The Android app

The app is the same React frontend wrapped in [Capacitor](https://capacitorjs.com)
and installed as a real Android app, package `edu.sust.cse.portal`. It bundles
the pages and talks to the live API at
`https://sustclassexamcoordinationsystem.onrender.com`. Notifications arrive
through Firebase Cloud Messaging instead of web push, which never worked
reliably in the old wrapper.

The website is unchanged for iPhone and desktop users, and web push stays as the
fallback there. On Android the website no longer installs itself: Chrome's
install option is withheld and a small banner points at `/download/android`
instead, so there is one app, not two.

## How the app looks

Inside the app the first screens are the app's own, not the website's. The
campus photo slideshow is gone: the sign-in, register, password reset and
teacher activation screens sit on plain paper with the SUST logo centred near
the top, a Student | Teacher control, 52 px inputs and one full-width button.
`Capacitor.isNativePlatform()` decides this, so the website is untouched.

The launcher icon is the SUST logo on white, as an adaptive icon with a
`#FFFFFF` background and the logo as the foreground inside the safe zone. The
splash is white (`#121212` in dark mode) with the logo centred and nothing else,
and it fades out as soon as the first screen is ready. Run
`npm run icons && npm run android:icons` after changing
`frontend/public/brand/sust-logo.png`: the first writes the web icons and the
transparent logo the app draws, the second writes the launcher, splash and
notification drawables.

There is no human check inside the app. The API accepts the app's requests
without `X-Human-Pass` because of the `X-Client: android` header; the rate
limits still apply. On the website the check is a Turnstile widget inside each
sign-in form.

The status bar and the navigation bar take the surface colour of the resolved
theme and flip their icons with it: near-white with dark icons in light mode,
near-black with light icons in dark mode. There is no coloured band, so the
system bars read as one surface with the app's own top and bottom bars.
`src/lib/shell.js` applies it on start, on every resume and whenever the theme
changes; `values/styles.xml` and `values-night/` set the same colours before the
web layer loads.

## How notifications look

`PortalMessagingService` builds every notification in the app, so they look the
same whether the app is open, closed or the phone is locked. There is **one**
logo: the small icon is the white SUST silhouette and the notification colour is
the primary `#1D4ED8`, so Android 12 and newer draws it inside a blue circle on
the left and the same silhouette appears in the status bar. There is no large
icon on the right any more. Long text expands, and the **SUST CSE** channel is
high-importance with sound and vibration. Tapping one opens the page the
notification points at.

The server sends data-only messages with high priority (`title`, `body`, `url`,
`tag`) so Android always hands them to that service instead of drawing its own
notification. Web push on the website is unchanged.

## How the app updates itself

`GET /api/app/android-latest` reports the newest GitHub Release as
`{version, version_code, apk_url, notes, required}`, read through the GitHub API
and cached for ten minutes. `GITHUB_RELEASES_REPO` picks the repository and
defaults to `raHI-79/250-project`.

The app asks on start and on every resume. When the release is newer than the
installed `versionName` it opens a bottom sheet, **Update available ·
`<version>`**, with the release notes and **Update** or **Later**; a release
whose notes contain `[required]` drops the **Later** button. **Update** pulls
the APK through the Capacitor filesystem plugin with live progress, then the
`ApkInstaller` plugin hands the file to the Android package installer over the
app's own `FileProvider`. The first time, Android asks for **Allow from this
source**: the sheet explains that once and opens the screen. Because the APK
carries the same signing key and the same `edu.sust.cse.portal` package, it
installs over the old app — including the older PWABuilder build — and the
sign-in survives.

The server announces a release it has not announced before at startup and then
every fifteen minutes: one notification per device token reading **SUST CSE
Portal `<version>` is available** with **Tap to update**. `app_meta` remembers
the announced version, so the same release never goes out twice.

`My profile → About` names the installed version and offers **Check for
updates** for anyone who does not want to wait.

## How attendance works in the app

Attendance is the one feature that only works inside this app, and only on the
phone a student registered. Everything else keeps working on any device and on
the website.

### What the teacher's phone does

While a session is open the app runs `AttendanceService`, a foreground service
with the ongoing notification `Attendance open · <course> · <present>/<roster>`
in its own low-importance **Attendance** channel. The service does two things at
once:

- it advertises a Bluetooth Low Energy packet on the fixed 16-bit service UUID
  `0xA7A7` carrying an 8-byte token, `HMAC(token_key, "<session id>:<10 second
  slice>")` truncated to eight bytes, re-advertised on every slice boundary at
  `ADVERTISE_MODE_LOW_POWER` and `ADVERTISE_TX_POWER_LOW`, which reaches about
  10 to 20 metres;
- it asks `LocationManager` for a GPS fix every 15 seconds and posts it to
  `POST /api/teacher/attendance/sessions/{id}/location` itself, with the
  teacher's bearer token, so a locked screen or a backgrounded app does not
  stop the position from refreshing.

The session key never reaches a student. The teacher's app needs it to rotate
the token without the network, and the same key produces the 4-digit code the
screen shows for a projector.

### What the student's phone does

The mark screen scans for that service UUID for ten seconds, reads the 8-byte
service data, and only then enables the big **I am present** button. Tapping it
asks for the phone's own fingerprint, face or screen lock through
`BiometricPrompt` with `BIOMETRIC_WEAK | DEVICE_CREDENTIAL`, takes one GPS fix,
and sends `{token, lat, lng, accuracy_m, is_mock, device_id}`. If nothing is
heard in ten seconds the screen offers **Enter the code on the board** instead;
the location rules are the same either way.

### The permissions it asks for

| Permission | Who needs it | Why |
|---|---|---|
| `ACCESS_FINE_LOCATION` | teachers and students | The 30 metre rule. Coarse location is not accurate enough, and on Android 11 and older a BLE scan needs it too |
| `BLUETOOTH_ADVERTISE` | teachers | Broadcasting the token. Android 12 and newer |
| `BLUETOOTH_SCAN` | students | Hearing the token, declared `neverForLocation`. Android 12 and newer |
| `BLUETOOTH_CONNECT` | both | Reading the adapter state. Android 12 and newer |
| `BLUETOOTH`, `BLUETOOTH_ADMIN` | both | The same thing on Android 11 and older, so both are capped at `maxSdkVersion="30"` |
| `FOREGROUND_SERVICE`, `FOREGROUND_SERVICE_LOCATION`, `FOREGROUND_SERVICE_CONNECTED_DEVICE` | teachers | The service that keeps advertising and sending the position |
| `USE_BIOMETRIC` | students | The fingerprint, face or screen-lock check before a mark |

Each side asks once, from the screen that needs it, and the screen explains what
is missing instead of failing silently. If Bluetooth is off the app asks Android
to turn it on with `ACTION_REQUEST_ENABLE`. `android.hardware.bluetooth_le` is
declared `required="false"`, so a phone without BLE still installs the app and
simply uses the code on the board.

### A note on GPS accuracy

A phone reports accuracy as a radius in metres, and indoors that radius grows:
near a window it is often 10 to 20 metres, deep inside a concrete building it
can pass 50 metres or the fix can be minutes old. A mark is refused above 50
metres with **Weak GPS signal — move near a window and try again**, so a class
in an inner room should expect students near the windows to mark first. The
teacher's own position has to be under a minute old for any mark to be accepted,
which is why the service refreshes it every 15 seconds. A position from a mock
provider is refused outright: `Location.isMock()` on Android 12 and newer,
`isFromMockProvider()` below that.

### The registered phone

The first Android phone a student signs in with becomes their attendance phone,
keyed on `Settings.Secure.ANDROID_ID`, which Android scopes to this app and this
phone. From any other phone the student sees **This is not your registered
phone. Request to use this phone.** with a reason field. The request reaches
every teacher who teaches that student's batch in the running semester as a push
and a bell notification, and any one of them can approve or refuse it from
**Phone requests** in the app or on the website. An approval rebinds the phone,
tells the student who approved it, and is written to the activity log. Two
approved changes per student per semester need no note; beyond that the teacher
has to add one. The office never approves these, it only reads the history.

Students on an iPhone cannot mark attendance at all; their teacher marks them by
hand from the session detail, which keeps the note and who set it.

### The Excel export

The teacher downloads a workbook from the session history or from **My
classes**. One sheet per course-batch-section carries the student name, the
registration number, one column per session date with `P` or `A` and the note as
a cell comment, the present count, the total and the percentage, with below
50 % in red and the header frozen. **Download all my classes** puts every class
the teacher teaches into one workbook and adds a Summary sheet. In the app the
file is saved to **Documents**; on the website the browser downloads it.

## Who the app is for

Students, class representatives and teachers. Inside the app the only sign-in
screens are **Student** and **Teacher**; the office portal is not reachable, and
the API answers `403` to an office token that carries the `X-Client: android`
header the app sends. Office admins use the website.

## 1. Firebase

1. Open the [Firebase console](https://console.firebase.google.com) and create a
   project, or pick the existing one.
2. **Add app → Android**, package name `edu.sust.cse.portal`. Nothing else on
   that screen matters for notifications.
3. Download `google-services.json` and save it as
   `frontend/android-config/google-services.json`. It is the public client
   config, not a secret, and it is committed. The build copies it into the
   Android project.
4. **Project settings → Service accounts → Generate new private key**. That JSON
   **is** a secret: it never goes into git.

## 2. The four GitHub secrets

**Settings → Secrets and variables → Actions → New repository secret**:

| Secret | What it holds |
|---|---|
| `ANDROID_KEYSTORE_BASE64` | the signing keystore, base64 |
| `ANDROID_KEYSTORE_PASSWORD` | the keystore password |
| `ANDROID_KEY_ALIAS` | the key alias inside the keystore |
| `ANDROID_KEY_PASSWORD` | the password of that key |

Use the **same** `signing.keystore` as the old build, or Android refuses to
install the new app over the old one. To turn it into base64 in PowerShell:

```powershell
[Convert]::ToBase64String([IO.File]::ReadAllBytes("signing.keystore")) | Set-Clipboard
```

If you ever need a new keystore:

```powershell
keytool -genkey -v -keystore signing.keystore -alias sustcse -keyalg RSA -keysize 2048 -validity 10000
```

## 3. The Render variable

The backend needs the Firebase **service account** from step 1.4, base64:

```powershell
[Convert]::ToBase64String([IO.File]::ReadAllBytes("service-account.json")) | Set-Clipboard
```

In Render, **Settings → Environment**, add:

| Variable | Value |
|---|---|
| `FIREBASE_SERVICE_ACCOUNT` | the base64 string from the command above |

Leave it empty and the server simply skips Firebase; web push and the website
keep working. The server logs which of the two it is at startup.

## 4. Releasing a version

1. Bump `version` in `frontend/package.json`, and `APP_VERSION` in
   `render.yaml`. That one number becomes `versionName`, the version the app
   reports and the name of the APK.
2. Merge it into `main`.
3. Tag it and push the tag:

   ```powershell
   git tag v2.2.0
   git push origin v2.2.0
   ```

`.github/workflows/android.yml` builds and signs the APK and attaches it to the
GitHub Release as `sust-cse-portal-<version>.apk`. Nothing else is manual:

- `https://<your host>/download/android` redirects to the new file within ten
  minutes. That is the link to share, and every **Download the Android app**
  button already points at it.
- Every installed app offers the update on its next start or resume.
- The server sends one notification per device token announcing the version.

Add `[required]` anywhere in the release notes to make the update unskippable.

To grab an APK without a release, open **Actions → Android APK**, click the
newest green run, and download the `sust-cse-portal-<version>` artifact.

`versionName` comes from `frontend/package.json` and `versionCode` from the run
number, so every run is installable over the previous one.

## 5. Installing and testing

1. Open `https://<your host>/download/android` on the phone, or copy the APK
   across and open it. Allow installing from this source if Android asks.
2. It installs **over** the old app as long as the signing key is the same. If
   Android refuses, the key differs: uninstall the old app first, which also
   clears its data.
3. Open the app and sign in as a student or a teacher. Android 13 and newer ask
   once for notification permission — allow it.
4. From another account, send that student a message, or post a notice for their
   batch. The phone should show a notification within a few seconds, even with
   the app closed.
5. Tap it. The app opens on the page the notification belongs to.
6. To check the server side, look for `push: account N now has 1 device tokens`
   in the Render log, or query `SELECT count(*) FROM fcm_tokens;` in Neon.

If nothing arrives, the Render log tells you why: `fcm:` lines carry the status
only, never a token or a key.

To test the updater, publish a release with a higher version than the installed
app and reopen the app: the update sheet should appear within a second or two.
`My profile → About → Check for updates` forces the same check and says so when
there is nothing new.

## 6. Building locally

```powershell
cd frontend
npm ci
npm run android:sync
npx cap open android
```

`npm run android:sync` builds the pages, copies `google-services.json` and runs
`npx cap sync android`. Gradle and a release build need Android Studio or the
command line SDK; the workflow does that part for you.

The generated `android/` folder is committed so the build is reproducible, but
`android/app/google-services.json`, the build output and `local.properties` are
ignored.

## Never commit

`*.keystore`, `*.jks`, `service-account*.json` and `firebase-adminsdk*.json` are
in `.gitignore`. Keep them that way.

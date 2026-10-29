# The Android app

The app is the same React frontend wrapped in [Capacitor](https://capacitorjs.com)
and installed as a real Android app, package `edu.sust.cse.portal`. It bundles
the pages and talks to the live API at
`https://sustclassexamcoordinationsystem.onrender.com`. Notifications arrive
through Firebase Cloud Messaging instead of web push, which never worked
reliably in the old wrapper.

The website is unchanged. iPhone and desktop users keep using it, and web push
stays as the fallback there.

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

## 4. Getting the APK

The workflow `.github/workflows/android.yml` builds a signed APK on every push
to `main` that touches `frontend/`, and whenever you start it by hand.

1. Open **Actions → Android APK** and click the newest green run.
2. Scroll to **Artifacts** and download `sust-cse-portal-<version>`.
3. Unzip it. Inside is `sust-cse-portal-<version>.apk`.

Push a tag like `v2.0.0` and the same APK is attached to the GitHub Release, so
the **Download the Android app** button on the website points at it.

`versionName` comes from `frontend/package.json` and `versionCode` from the run
number, so every run is installable over the previous one.

## 5. Installing and testing

1. Copy the APK to the phone and open it. Allow installing from this source if
   Android asks.
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

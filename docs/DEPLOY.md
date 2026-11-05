# Putting the portal online for free

Written for: whoever on the team puts the project on the internet. No payment
card is needed anywhere in this guide.

The whole portal runs as **one** service: the Go server answers `/api` and also
serves the built React pages, so there is nothing else to host. Four free
accounts carry it:

| Piece | Service | What it does |
|---|---|---|
| The app | Render, free web service | Builds the Docker image and runs it |
| The database | Neon, free Postgres | Keeps every row, and the uploaded files |
| Email | Brevo | Sends the codes and invitations over HTTPS |
| Staying awake | cron-job.org | Pings the app so it sleeps less |

Work through the steps in order. Each one takes a few minutes.

## 1. The database on Neon

1. Sign up at [neon.tech](https://neon.tech) with GitHub.
2. Create a project. Name it `sust-cse-portal` and pick the region
   **Asia Pacific (Singapore)** — it is the closest one to Bangladesh.
3. Open **Connection string** and copy the one that ends with
   `?sslmode=require`. It looks like
   `postgres://user:password@ep-something.ap-southeast-1.aws.neon.tech/neondb?sslmode=require`.
4. Keep it somewhere safe for step 4. This is your `DATABASE_URL`.

The server creates every table itself the first time it starts, so there is
nothing to import.

## 2. Email

Render's free plan blocks the outgoing SMTP ports, so the portal sends mail over
HTTPS. `MAIL_PROVIDER` picks how.

### Google Apps Script, recommended

Brevo delivers reliably only from a verified domain; a Gmail sender lands in
spam or nowhere. Sending through your own Google account avoids that, because
the mail really does come from that account.

1. Open [script.google.com](https://script.google.com) and create a project.
2. Paste a `doPost(e)` that reads `{secret, to, subject, body, name}` from
   `e.postData.contents`, compares `secret` against a constant you keep in the
   script, calls `MailApp.sendEmail` with `name` as the display name and
   answers `ContentService.createTextOutput(JSON.stringify({ok: true}))` with
   the JSON mime type. Answer `{"ok": false, "error": "..."}` on anything else.
3. **Deploy → New deployment → Web app**, run it as yourself, give access to
   **Anyone**, and copy the `/exec` address. That is `APPS_SCRIPT_URL`.
4. Put the same constant from step 2 in `APPS_SCRIPT_SECRET`, and set
   `MAIL_PROVIDER=appsscript`.

The portal follows the redirect Google answers with and treats only
`{"ok": true}` as success. The secret is never written to a log.

Google's free quota is 100 recipients a day for a consumer account, which is
enough for invitations and reset codes.

### Brevo, the alternative

1. Sign up at [brevo.com](https://www.brevo.com) — the free plan sends 300
   mails a day.
2. Go to **Senders, Domains & Dedicated IPs → Senders** and add the address the
   portal will send from. Brevo emails you a confirmation link; open it, or
   mail will be refused later. Verify a real domain if you can: a Gmail sender
   is what made delivery unreliable before.
3. Go to **SMTP & API → API Keys** and create a key. Copy it once — Brevo never
   shows it again. This is your `BREVO_API_KEY`, with `MAIL_PROVIDER=brevo`.

### The human check

Cloudflare Turnstile guards the sign-in and registration endpoints. Create a
free widget at **Cloudflare → Turnstile**, choose the **Managed** mode, add your
Render hostname, and copy the two keys into `TURNSTILE_SITE_KEY` and
`TURNSTILE_SECRET_KEY`. Leave either one empty and the gate is skipped.

## 3. Google sign-in, push keys and the admin password

- **Google client id** — reuse the one in your local `backend/.env`, or make a
  new one in the Google Cloud console under **APIs & Services → Credentials**.
  You will add the live address to it in step 7.
- **VAPID keys** — reuse the pair from your local `backend/.env`. Never commit
  the private one.
- **Admin values** — choose `ADMIN_IDENTIFIER`, a long `ADMIN_PASSWORD`,
  `ADMIN_EMAIL` and `ADMIN_PHONE`. The office account is created from them on
  the first start.
- **The super admin** — leave `SUPER_ADMIN_IDENTIFIER` empty to turn the
  `ADMIN_IDENTIFIER` account into the super admin, or set it to another login
  id and that account is seeded with the same password and promoted instead.
  There is exactly one super admin and it is the only account that opens
  `/control-room`.
- **JWT secret** — any random string of at least 32 characters.
- **Analytics salt** — `ANALYTICS_SALT`, another long random string. Every IP
  address in the activity log and the page-view rows is hashed with it, so the
  portal never keeps an address it can read back. Changing it later makes the
  old hashes unmatchable, which is harmless.
- **Backup status secret** — `BACKUP_STATUS_SECRET`, one more long random
  string. Step 8 puts the same value in GitHub.

Nothing else moves to Render: from 2.6.0 the maintenance mode, the registration
switch, the announcement banner, the feature switches, the reminder times, the
number of days the routine is generated ahead, the minimum app version and the
retention windows all live in **Control room → Site settings** and take effect
within half a minute. The environment values are only their defaults.

## 4. The web service on Render

1. Sign up at [render.com](https://render.com) with GitHub and allow it to read
   this repository.
2. **New → Web Service**, pick the repository, and choose the branch `main`.
3. Render reads `render.yaml` and fills most of this in. Check that
   **Runtime** is `Docker`, **Region** is `Singapore`, **Instance type** is
   `Free`, and **Health check path** is `/api/health`.
4. Fill in every value marked *secret* below, then create the service.

### The environment

| Name | Value |
|---|---|
| `APP_ENV` | `production` |
| `APP_TIMEZONE` | `Asia/Dhaka` |
| `DATABASE_URL` | the Neon string from step 1, keeping `?sslmode=require` |
| `DB_MAX_CONNS` | `5` |
| `FILE_STORAGE` | `database` |
| `STATIC_DIR` | already set inside the image, leave it alone |
| `FRONTEND_ORIGIN` | your live address, for example `https://sust-cse-portal.onrender.com` |
| `JWT_SECRET` | your long random string |
| `JWT_TTL` | `24h` |
| `MAIL_PROVIDER` | `appsscript` |
| `BREVO_API_KEY` | the key from step 2 |
| `MAIL_FROM` | the sender address you verified in step 2 |
| `MAIL_FROM_NAME` | `SUST CSE Portal` |
| `ADMIN_IDENTIFIER` | the office login id |
| `SUPER_ADMIN_IDENTIFIER` | empty, or another login id for the super admin |
| `ANALYTICS_SALT` | *secret*, a long random string that hashes the IP addresses |
| `BACKUP_STATUS_SECRET` | *secret*, the same value you add to GitHub in step 8 |
| `ADMIN_PASSWORD` | a long password |
| `ADMIN_EMAIL` | the office email |
| `ADMIN_PHONE` | the office phone, 11 digits |
| `GOOGLE_CLIENT_ID` | the Google client id |
| `VAPID_PUBLIC_KEY` | your push public key |
| `VAPID_PRIVATE_KEY` | your push private key |
| `VAPID_SUBJECT` | `mailto:` and your address |
| `STUDENT_EMAIL_DOMAIN` | `student.sust.edu` |
| `TEACHER_REMINDER_TIMES` | `16:00,21:00` |
| `TEACHER_MORNING_REMINDER` | `07:30` |
| `SCHEDULE_DAYS_AHEAD` | `7` |
| `MAX_UPLOAD_MB` | `10` |
| `APPS_SCRIPT_URL` | the `/exec` address of your Apps Script web app |
| `APPS_SCRIPT_SECRET` | the shared secret the script checks |
| `TURNSTILE_SITE_KEY` | the Cloudflare Turnstile site key |
| `TURNSTILE_SECRET_KEY` | the Cloudflare Turnstile secret key |
| `APP_VERSION` | `2.5.0` |
| `FIREBASE_SERVICE_ACCOUNT` | the Firebase service account JSON, base64 |
| `GITHUB_RELEASES_REPO` | `raHI-79/250-project` |
| `GITHUB_TOKEN` | *secret*, optional: a fine-grained token with read-only **Contents** on this one repository |

`GITHUB_TOKEN` is optional but worth setting. The app asks
`/api/app/android-latest` for the newest release, and the server reads that from
the GitHub API. Without a token GitHub allows only 60 requests an hour **per IP**,
and a free Render instance shares its outbound address with other services, so
the lookup can be rate limited for reasons that have nothing to do with this
portal. With a token the limit is 5000 an hour for the account. Create it under
**GitHub → Settings → Developer settings → Fine-grained tokens**, give it access
to this repository only, set **Contents** to *Read-only*, and nothing else — the
server never writes to GitHub. Releases are public, so the lookup still works
without it: it keeps the last good answer for six hours and answers `503` with
*Update service is busy, try again later.* only when GitHub is unreachable and
nothing is cached. The token is never logged.

`FRONTEND_ORIGIN` is a chicken and egg: Render shows the address only after the
service is created. Create it with your best guess, then open
**Settings → Environment**, correct the value and save — Render redeploys.

`GOOGLE_CLIENT_ID` is the **web** OAuth client id. The pages read it at runtime
from `GET /api/auth/config`, so changing it needs no rebuild and no new APK, and
the Android app sends the same id as its server client id. There is no
`VITE_GOOGLE_CLIENT_ID` any more. `docs/ANDROID.md` has the two OAuth clients
Google needs.

## 5. The first deploy

Watch the log. A healthy first start prints the migrations, then
`db: connected`, `auth: admin account ready` and the port it listens on. Open
the address: the home page appears, and `/<your address>/api/health` answers
`{"status":"ok"}`.

If the log stops at the database, the Neon compute was asleep. The server
retries for a minute, so it usually recovers on its own.

## 6. Keeping it awake

A free Render service sleeps after 15 minutes with no traffic and takes about a
minute to wake, which makes the first visit of the day slow.

1. Sign up at [cron-job.org](https://cron-job.org).
2. **Create cronjob**, title it `portal health`, URL
   `https://<your address>/api/health`, and set it to run **every 10 minutes**.
3. Save and enable it.

This does not make the free plan unlimited: Render gives 750 instance hours a
month, which one always-on service fits inside.

## 7. Google sign-in on the live address

In the Google Cloud console, open your OAuth client and add your Render address
to **Authorised JavaScript origins**, for example
`https://sust-cse-portal.onrender.com`. Without it, the Google button on the
registration page refuses to load.

## 8. Nightly backups

The disk of a free Render service is wiped on every deploy, which is exactly why
`FILE_STORAGE` is `database`: the uploaded photos and attachments live in
Postgres with everything else, so one dump holds the whole portal.

1. In GitHub, open **Settings → Secrets and variables → Actions**.
2. **New repository secret**, name it `BACKUP_DATABASE_URL`, and paste the same
   Neon connection string.
3. **New repository secret** again, name it `BACKUP_PASSPHRASE`, and put a long
   random phrase in it. The dump is encrypted with it before it is uploaded, and
   the job fails with a clear message if the secret is missing. Keep the phrase
   somewhere safe and offline: without it the backups cannot be read.
4. **New repository secret** again, name it `BACKUP_STATUS_SECRET`, and paste
   the same value you put in the Render environment in step 4.
5. **New repository secret** once more, name it `BACKUP_STATUS_URL`, and set it
   to `https://<your address>/api/system/backup-status`.
6. The workflow `.github/workflows/backup.yml` then runs every night at 21:30
   UTC, which is 03:30 in Dhaka, and you can also start it by hand from the
   **Actions** tab with **Run workflow**. It installs the PostgreSQL 18 client
   from the PostgreSQL apt repository, because Neon runs PostgreSQL 18 and
   `pg_dump` refuses a server newer than itself.

### Telling the portal the backup worked

After the dump is uploaded, the workflow posts the name of the file to
`POST /api/system/backup-status` with the header `X-Backup-Secret`. The server
compares that header with `BACKUP_STATUS_SECRET` and refuses anything else with
`403`, so the endpoint is not a way in. The control-room dashboard shows the
time of the last successful backup and raises an alert when it is older than 36
hours; if both secrets are missing the workflow skips the report and the
dashboard keeps saying *Never*, which is the honest answer. The secret is never
logged and never reaches the pages.

## 9. The control room

The super admin signs in at `https://<your address>/control-room`. Nothing links
to it, the page carries `noindex,nofollow`, the Android app redirects away from
it, and only the one account whose role is `super_admin` can get in — an office
admin there is refused like anybody else. The office admins keep using
`/office-portal`.

From the control room the super admin opens the dashboard, the office admins
page and the activity log, and can open every office page read-only: the server
refuses any write from that role with
*The control room opens the office pages read-only.*

Office admins are invited by email from **Control room → Office admins**. The
link is valid for 72 hours and lands on `/office-portal/accept-invite`, where
the invitee picks a username (the teacher User ID rules) and a password. The
same page grants or revokes office access for any active teacher; that teacher
then signs in at `/office-portal` with their own credentials and keeps their
teacher account and their app access untouched.

### Downloading a backup

Open **Actions → Database backup**, click the run you want, and download the
artifact at the bottom of the page. On a public repository any signed-in GitHub
user can download an artifact, which is why the dump is encrypted before it is
uploaded: the artifact is a single `.gpg` file named by date, for example
`sust-cse-2026-10-23.sql.gz.gpg`, and GitHub keeps it for 30 days. Only
`BACKUP_PASSPHRASE` opens it, and the passphrase is never printed in the log.

### Decrypting a backup

```bash
gpg --decrypt --output sust-cse-2026-10-23.sql.gz sust-cse-2026-10-23.sql.gz.gpg
```

`gpg` asks for the passphrase. To decrypt without a prompt, for example in a
script, pass it on standard input instead of on the command line:

```bash
gpg --batch --decrypt --passphrase-fd 0 \
  --output sust-cse-2026-10-23.sql.gz sust-cse-2026-10-23.sql.gz.gpg < passphrase.txt
```

### Restoring a backup

```bash
gunzip -c sust-cse-2026-10-23.sql.gz | psql "postgres://...neon.tech/neondb?sslmode=require"
```

Or in one pass, without writing the plain dump to disk:

```bash
gpg --decrypt sust-cse-2026-10-23.sql.gz.gpg | gunzip -c |
  psql "postgres://...neon.tech/neondb?sslmode=require"
```

Restore into an empty database. In Neon, the simplest way to get one is
**Branches → New branch**, which gives a fresh connection string you can point
the command at, test, and then promote.

## 10. The emergency password reset

If the super admin password is lost, reset it from the server — never from an
argument on the command line, which would land in the shell history.

1. In Render, open the service and choose **Shell**.
2. Set the new password in the environment of that shell and run the command.
   The image carries the tool next to the server, so there is no Go toolchain and
   no source to build:

```bash
export NEW_SUPER_ADMIN_PASSWORD='a long new password with 1 digit'
/app/sust-admin -reset-super
unset NEW_SUPER_ADMIN_PASSWORD
```

On a development machine with the source checked out, `cd backend && go run
./cmd/admin -reset-super` does the same thing.

It refuses anything under 10 characters or without a letter and a digit, replaces
the hash, closes every session of that account, prints only `Password updated`
and records the reset in the activity log. Sign in again at `/control-room` and
change the password from **Security** if you want a different one.

## 11. Rebuilding the Android app

The installed Android app is a Trusted Web Activity wrapper around the same
site, package `edu.sust.cse.portal`. The web part updates itself on the next
launch, so a rebuild is only needed when the wrapper itself changes, for
example after the status bar colour changed.

1. Open [PWABuilder](https://www.pwabuilder.com/) and enter the live address.
2. Choose **Package for stores → Android**, then **All settings**.
3. Set **Status bar colour** and **Navigation bar colour** to `#7A1F2B`, and
   leave **Display mode** on `standalone`.
4. Keep the package id `edu.sust.cse.portal` exactly as it is. A different id
   installs as a second app instead of updating the first one.
5. Raise **App version code** by one above the installed build and give
   **App version** the same number as `frontend/package.json`.
6. Under **Signing key**, choose **Use mine** and upload the original
   `signing.keystore` with its key alias and both passwords. A new key makes
   Android refuse the update and breaks the verified link.
7. Download the zip, install `app-release-signed.apk` on the phone over the
   old one, and confirm the address bar stays hidden.

The digital asset link is unchanged as long as the signing key is the same, so
`frontend/public/.well-known/assetlinks.json` needs no edit on a rebuild.

## 12. The app download

`GET /download/android` is served by our own server, not by a redirect to
GitHub, so downloads keep working even if the repository becomes private. The
server asks the GitHub asset API for the newest
`sust-cse-portal-<version>.apk` with `GITHUB_TOKEN`, follows the redirect to the
storage host without passing the token on, writes the file into
`UPLOAD_DIR/apk/` and answers it with
`Content-Type: application/vnd.android.package-archive`, a
`Content-Disposition: attachment` name and a `Content-Length`. Every later
request for that version is answered from the cached file, so GitHub is asked
once per release. `GET /download/android/<version>` serves one specific version,
and that address is what `apk_url` in `GET /api/app/android-latest` points at, so
the in-app updater downloads through the portal too.

Without `GITHUB_TOKEN`, or when GitHub refuses, the answer is `503` with *The
app download is not available right now. Please try again later.* and the log
holds the status only — never the token.

## 13. The Android app

The installed Android app is now a Capacitor build, not a web wrapper, and its
notifications come through Firebase. `docs/ANDROID.md` covers the Firebase
setup, the four GitHub secrets, `FIREBASE_SERVICE_ACCOUNT` and how to get the
APK out of GitHub Actions.

## Updating the portal

Push to `main`. Render sees the commit, builds the image again and swaps the
service over when the health check passes. Nothing else to do.

## If you ever move files into the database later

A portal already running on `FILE_STORAGE=disk` can be moved over without losing
anything:

```powershell
cd backend
go run ./cmd/migratefiles
```

It copies everything under `UPLOAD_DIR` into the database once and is safe to
run twice. Switch `FILE_STORAGE` to `database` afterwards.

## What the free plans give you

| Limit | What it means here |
|---|---|
| Render sleeps after 15 idle minutes | The first visit after a quiet spell waits about a minute |
| Render gives 750 instance hours a month | Enough for one service running all month |
| Render wipes the disk on each deploy | Why the files live in the database |
| Render blocks outgoing SMTP ports | Why mail goes through the Brevo API |
| Neon gives 0.5 GB of storage | Thousands of rows and a few hundred uploads |
| Neon sleeps its compute when idle | The server retries for a minute while it wakes |
| Brevo sends 300 mails a day | Far more than registration needs |
| GitHub keeps artifacts 30 days | Download a dump you want to keep for longer |

## Local development is unchanged

None of this touches how you work. With `backend/.env` as it is —
no `STATIC_DIR`, `FILE_STORAGE=disk`, `MAIL_PROVIDER=smtp` — the backend and
`npm run dev` behave exactly as before.

import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import test from 'node:test'
import {
  CODE_CONTACT_TEXT,
  EMAIL_CODE_MINUTES,
  EXPIRED_TEXT,
  GOOGLE_CONTACT_TEXT,
  codeDeadline,
  codeExpired,
  codeHint,
  contactText,
  googleOffered,
  resendLabel,
  resendNote,
  resentText,
} from './registerEmail.js'

const source = (path) => readFileSync(new URL(path, import.meta.url), 'utf8')

const sent = new Date('2026-11-12T09:00:00Z')

const later = (minutes) => new Date(sent.getTime() + minutes * 60000)

test('the app never offers Google while the website still does', () => {
  assert.equal(googleOffered({ app: true, enabled: true }), false)
  assert.equal(googleOffered({ app: false, enabled: true }), true)
  assert.equal(googleOffered({ app: false, enabled: false }), false)
  assert.equal(contactText(false), CODE_CONTACT_TEXT)
  assert.equal(contactText(true), GOOGLE_CONTACT_TEXT)
  assert.doesNotMatch(CODE_CONTACT_TEXT, /Google/)
})

test('the register screen draws the Google button and its divider only when it is offered', () => {
  const page = source('../pages/auth/RegisterPage.jsx')
  assert.match(page, /const app = isAndroidApp\(\)/)
  assert.match(page, /googleOffered\(\{ app, enabled: googleEnabled \}\)/)
  const gate = page.indexOf('{offerGoogle && (')
  assert.ok(gate > 0, 'the Google block is gated by offerGoogle')
  const block = page.slice(gate, page.indexOf('</>', page.indexOf('<GoogleVerifyButton', gate)))
  assert.match(block, />or</)
  assert.match(block, /<GoogleVerifyButton/)
  assert.equal(page.split('<GoogleVerifyButton').length, 2, 'one Google button on the page')
  assert.doesNotMatch(page.replace(block, ''), />or</)
})

test('the app build carries no native Google sign-in', () => {
  const button = source('../components/forms/GoogleVerifyButton.jsx')
  const google = source('./google.js')
  const manifest = JSON.parse(source('../../package.json'))
  const activity = source('../../android/app/src/main/java/edu/sust/cse/portal/MainActivity.java')
  const gradle = source('../../android/app/capacitor.build.gradle')
  const settings = source('../../android/capacitor.settings.gradle')
  assert.doesNotMatch(button, /isAndroidApp|Native|nativeGoogle/)
  assert.doesNotMatch(google, /SocialLogin|capacitor-social-login|nativeGoogle/)
  assert.equal(manifest.dependencies['@capgo/capacitor-social-login'], undefined)
  assert.doesNotMatch(activity, /social|SocialLogin|GoogleProvider/i)
  assert.doesNotMatch(gradle, /social-login/)
  assert.doesNotMatch(settings, /social-login/)
  assert.match(button, /Continue with Google|renderButton/)
})

test('the code deadline comes from the server, or fifteen minutes from sending', () => {
  assert.equal(codeDeadline('2026-11-12T09:15:00Z', sent).toISOString(), '2026-11-12T09:15:00.000Z')
  assert.equal(codeDeadline('', sent).getTime(), later(EMAIL_CODE_MINUTES).getTime())
  assert.equal(codeDeadline('', null), null)
})

test('the code hint counts down and says when the code has expired', () => {
  const deadline = later(15)
  assert.equal(
    codeHint({ emailHint: 'ra***@student.sust.edu', deadline, now: later(1) }),
    'Sent to ra***@student.sust.edu. It expires in 14 minutes.',
  )
  assert.equal(
    codeHint({ emailHint: '', deadline, now: later(14.5) }),
    'Sent to your university email address. It expires in 1 minute.',
  )
  assert.equal(
    codeHint({ emailHint: '', deadline, now: later(16) }),
    'Sent to your university email address. This code has expired.',
  )
  assert.equal(
    codeHint({ emailHint: '', deadline: null, now: sent }),
    'Sent to your university email address. It expires in 15 minutes.',
  )
  assert.equal(codeExpired(deadline, later(15)), true)
  assert.equal(codeExpired(deadline, later(14)), false)
  assert.equal(codeExpired(null, later(99)), false)
  assert.match(EXPIRED_TEXT, /Resend the code/)
})

test('resending is explained while it waits and after it is sent', () => {
  assert.equal(resendLabel(42), 'Resend in 42s')
  assert.equal(resendLabel(0), 'Resend the code')
  assert.match(resendNote(42), /when the timer ends/)
  assert.match(resendNote(0), /ask for a new code/)
  assert.equal(
    resentText('ra***@student.sust.edu'),
    'A new code was sent to ra***@student.sust.edu. The earlier code no longer works.',
  )
  assert.equal(resentText(''), 'A new code was sent. The earlier code no longer works.')
})

test('the email code path sends, verifies and resends through the account endpoints', () => {
  const page = source('../pages/auth/RegisterPage.jsx')
  assert.match(page, /api\.post\('\/api\/auth\/register', body/)
  assert.match(page, /'\/api\/auth\/verify-email'/)
  assert.match(page, /'\/api\/auth\/resend-verification'/)
  assert.match(page, /Send code/)
})

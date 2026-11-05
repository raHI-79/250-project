import assert from 'node:assert/strict'
import test from 'node:test'
import {
  APP_AUTH_FOOTER,
  APP_AUTH_SUBTITLE,
  APP_AUTH_TITLE,
  appFooter,
  appPanel,
} from './authpanel.js'

test('the app sign-in panel carries the portal name and the tagline', () => {
  assert.deepEqual(appPanel(), { title: APP_AUTH_TITLE, subtitle: APP_AUTH_SUBTITLE })
  assert.equal(APP_AUTH_TITLE, 'SUST CSE Portal')
  assert.equal(APP_AUTH_SUBTITLE, 'Classes, exams and attendance in one place')
})

test('the other app screens keep their own title', () => {
  assert.deepEqual(appPanel('Create your account', 'Fill the form once'), {
    title: 'Create your account',
    subtitle: 'Fill the form once',
  })
  assert.deepEqual(appPanel('Activate your account'), {
    title: 'Activate your account',
    subtitle: '',
  })
})

test('a blank title falls back to the sign-in panel', () => {
  assert.deepEqual(appPanel('   ', 'ignored'), {
    title: APP_AUTH_TITLE,
    subtitle: APP_AUTH_SUBTITLE,
  })
})

test('the sheet footer names the department and the version', () => {
  assert.equal(appFooter('2.5.0'), `${APP_AUTH_FOOTER} · Version 2.5.0`)
  assert.match(appFooter(''), /Version 0\.0\.0$/)
})

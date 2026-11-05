import assert from 'node:assert/strict'
import test from 'node:test'
import { HUMAN_FORMS, humanBlocked, needsHumanCheck, showsHumanCheck } from './humancheck.js'
import { paths } from './paths.js'

const SITE_KEY = 'site-key'

test('every form behind the human gate is listed', () => {
  const expected = [
    paths.studentLogin(),
    paths.studentRegister(),
    paths.studentForgotPassword(),
    paths.teacherLogin(),
    paths.teacherForgotPassword(),
    paths.teacherActivate(),
    paths.adminLogin(),
    paths.adminForgotPassword(),
    paths.adminInvite(),
    paths.controlLogin(),
    paths.controlForgotPassword(),
  ]
  expected.forEach((path) => {
    assert.equal(needsHumanCheck(path), true, `${path} is missing from HUMAN_FORMS`)
  })
  assert.equal(HUMAN_FORMS.length, expected.length)
})

test('each of those forms shows the widget when a site key is configured', () => {
  HUMAN_FORMS.forEach((path) => {
    assert.equal(needsHumanCheck(path), true)
    assert.equal(showsHumanCheck({ siteKey: SITE_KEY }), true, `${path} hides the widget`)
  })
})

test('no widget without a site key and never inside the app', () => {
  assert.equal(showsHumanCheck({ siteKey: '' }), false)
  assert.equal(showsHumanCheck({ siteKey: SITE_KEY, native: true }), false)
  assert.equal(showsHumanCheck({ siteKey: SITE_KEY, passed: true }), false)
})

test('the submit button waits for a token', () => {
  assert.equal(humanBlocked({ ready: false, siteKey: SITE_KEY }), true)
  assert.equal(humanBlocked({ ready: true, siteKey: SITE_KEY, token: '' }), true)
  assert.equal(humanBlocked({ ready: true, siteKey: SITE_KEY, token: 'abc' }), false)
  assert.equal(humanBlocked({ ready: true, siteKey: SITE_KEY, passed: true, token: '' }), false)
  assert.equal(humanBlocked({ ready: true, siteKey: '', token: '' }), false)
  assert.equal(humanBlocked({ ready: true, siteKey: SITE_KEY, native: true, token: '' }), false)
})

test('a page outside the list needs no widget', () => {
  assert.equal(needsHumanCheck(paths.profile()), false)
  assert.equal(needsHumanCheck(''), false)
  assert.equal(needsHumanCheck(undefined), false)
})

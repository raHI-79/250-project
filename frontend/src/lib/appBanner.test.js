import assert from 'node:assert/strict'
import test from 'node:test'
import { bannerShows, isAccountPath } from './appBanner.js'

test('sign-in, sign-up, forgot and reset pages are account pages', () => {
  for (const path of [
    '/student/login',
    '/student/register',
    '/student/register?verify=2023331079',
    '/student/forgot-password',
    '/teacher/login',
    '/teacher/forgot-password/',
    '/teacher/activate',
    '/office-portal',
    '/office-portal/forgot-password',
    '/office-portal/accept-invite',
    '/control-room',
    '/control-room/forgot-password',
  ]) {
    assert.equal(isAccountPath(path), true, path)
  }
})

test('every other page is not an account page', () => {
  for (const path of ['/', '/student/today', '/teacher/attendance', '/office-portal/overview', '/profile']) {
    assert.equal(isAccountPath(path), false, path)
  }
})

test('the banner shows on an Android browser away from the account pages until it is hidden', () => {
  assert.equal(bannerShows({ androidBrowser: true, dismissed: false, pathname: '/' }), true)
  assert.equal(bannerShows({ androidBrowser: true, dismissed: false, pathname: '/student/register' }), false)
  assert.equal(bannerShows({ androidBrowser: true, dismissed: true, pathname: '/' }), false)
  assert.equal(bannerShows({ androidBrowser: false, dismissed: false, pathname: '/' }), false)
})

import assert from 'node:assert/strict'
import test from 'node:test'
import {
  ATTENDANCE_TAB_MINE,
  ATTENDANCE_TAB_PHONES,
  CR_TAB_CREATE,
  CR_TAB_DEFAULT,
  CR_TAB_FINAL,
  CR_TAB_ORDER,
  CR_TAB_ROOMS,
  CR_TAB_UPCOMING,
  PORTAL_CONTROL,
  ROLE_SUPER_ADMIN,
  homeFor,
  loginFor,
  notificationTarget,
  paths,
  portalFor,
} from './paths.js'

test('the class and exam hub opens on the upcoming routine', () => {
  assert.equal(CR_TAB_DEFAULT, CR_TAB_UPCOMING)
  assert.deepEqual(CR_TAB_ORDER, [CR_TAB_UPCOMING, CR_TAB_FINAL, CR_TAB_CREATE, CR_TAB_ROOMS])
})

test('the hub route keeps working with and without a tab', () => {
  assert.equal(paths.cr.schedule(), '/cr/create-class')
  assert.equal(paths.cr.schedule({ tab: CR_TAB_FINAL }), '/cr/create-class?tab=final')
})

test('a phone request leads into the attendance tabs', () => {
  const target = notificationTarget({ kind: 'phone_requests', ref: 7 }, 'teacher')
  assert.match(target, /^\/teacher\/attendance\?/)
  assert.match(target, new RegExp(`tab=${ATTENDANCE_TAB_PHONES}`))
  assert.match(target, /highlight=7/)
})

test('my attendance leads into the attendance tabs', () => {
  const target = notificationTarget({ kind: 'attendance_mine', ref: 4 }, 'student')
  assert.match(target, /^\/student\/attendance\?/)
  assert.match(target, new RegExp(`tab=${ATTENDANCE_TAB_MINE}`))
  assert.match(target, /course=4/)
})

test('a routine notification still opens its own day', () => {
  assert.equal(
    notificationTarget({ kind: 'routine', date: '2026-11-20' }, 'student'),
    '/student/upcoming-routine?date=2026-11-20',
  )
})

test('the website sign-in routes stay where they were', () => {
  assert.equal(paths.studentLogin(), '/student/login')
  assert.equal(paths.teacherLogin(), '/teacher/login')
  assert.equal(paths.studentRegister(), '/student/register')
  assert.equal(paths.teacherActivate(), '/teacher/activate')
  assert.equal(paths.adminLogin(), '/office-portal')
})

test('the control room hides behind its own route', () => {
  assert.equal(paths.controlLogin(), '/control-room')
  assert.equal(paths.control.home(), '/control-room')
  assert.equal(paths.control.admins(), '/control-room/admins')
  assert.equal(paths.control.activity(), '/control-room/activity')
  assert.equal(paths.control.monitor(7), '/control-room/activity/7')
  assert.equal(paths.adminInvite('abc'), '/office-portal/accept-invite?token=abc')
})

test('the super admin lands in the control room and signs in there', () => {
  assert.equal(portalFor(ROLE_SUPER_ADMIN), PORTAL_CONTROL)
  assert.equal(homeFor(ROLE_SUPER_ADMIN), '/control-room')
  assert.equal(loginFor(ROLE_SUPER_ADMIN), '/control-room')
  assert.equal(homeFor('admin'), '/office-portal/overview')
  assert.equal(loginFor('admin'), '/office-portal')
})

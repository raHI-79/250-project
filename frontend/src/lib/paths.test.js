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
  notificationTarget,
  paths,
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

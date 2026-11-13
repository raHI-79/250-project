import assert from 'node:assert/strict'
import test from 'node:test'
import {
  CLOSED_TEXT,
  CODE_ONLY_LABEL,
  EMPTY_CLOSED_TEXT,
  NO_LOCATION_FLAG,
  OFFICE_TAB_HISTORY,
  STUDENT_TAB_MARK,
  TAKE_BLUETOOTH,
  TAKE_ROLL_CALL,
  TEACHER_TAB_TAKE,
  belowPass,
  classKey,
  closedMessage,
  codeOnlyMarkBody,
  deleteSheetQuestion,
  markFlags,
  markMeta,
  methodLabel,
  percentOf,
  presentOfRoster,
  rollNumber,
  sheetTitle,
  sortedRoster,
  startBody,
  takeMethods,
} from './attendanceView.js'

const sheet = {
  course_id: 4,
  course_code: 'CSE245',
  course_title: 'Databases',
  batch: 'CSE-23',
  section: 'A',
}

test('every marking method has a label of its own', () => {
  assert.equal(methodLabel('app'), 'Bluetooth')
  assert.equal(methodLabel('code'), 'Code')
  assert.equal(methodLabel('web'), 'Code')
  assert.equal(methodLabel('manual'), 'Teacher')
  assert.equal(methodLabel('manual', 'roll_call'), 'Roll call')
  assert.equal(methodLabel('system'), 'System')
  assert.equal(methodLabel('nonsense'), '')
})

test('the two ways of taking attendance are named', () => {
  assert.equal(TAKE_BLUETOOTH, 'bluetooth')
  assert.equal(TAKE_ROLL_CALL, 'roll_call')
})

test('each role opens its attendance tabs on the first one', () => {
  assert.equal(TEACHER_TAB_TAKE, 'take')
  assert.equal(STUDENT_TAB_MARK, 'mark')
  assert.equal(OFFICE_TAB_HISTORY, 'history')
})

test('the delete asks about the course, the batch and the section', () => {
  assert.equal(sheetTitle(sheet), 'CSE245 Databases · CSE-23 A')
  assert.equal(
    deleteSheetQuestion(sheet),
    'Are you sure you want to delete the attendance sheet for CSE245 Databases · CSE-23 A? This cannot be undone.',
  )
})

test('one sheet is kept per course, batch and section', () => {
  assert.equal(classKey(sheet), '4|CSE-23|A')
  assert.notEqual(classKey(sheet), classKey({ ...sheet, section: 'B' }))
})

test('a share below half is flagged', () => {
  assert.equal(percentOf(1, 2), 50)
  assert.equal(percentOf(0, 0), 0)
  assert.equal(belowPass(49), true)
  assert.equal(belowPass(50), false)
})

test('reads the roll number off the registration number', () => {
  assert.equal(rollNumber('2023331079'), '079')
  assert.equal(rollNumber('202333107912'), '912')
  assert.equal(rollNumber('202301'), '301')
  assert.equal(rollNumber('2023331001'), '001')
  assert.equal(rollNumber('2023331120'), '120')
  assert.equal(rollNumber('2019-331-7'), '317')
  assert.equal(rollNumber(''), '')
  assert.equal(rollNumber(undefined), '')
})

test('sorts a roster by registration number', () => {
  const roster = [
    { reg_no: '2023331100' },
    { reg_no: '2023331009' },
    { reg_no: '2023331021' },
  ]
  assert.deepEqual(
    sortedRoster(roster).map((entry) => entry.reg_no),
    ['2023331009', '2023331021', '2023331100'],
  )
  assert.deepEqual(sortedRoster(undefined), [])
})

test('a session starts with code and location unless the teacher picks code only', () => {
  const entry = { course_id: 4, batch: 'CSE-23', section: 'A', schedule_item_id: 0 }
  assert.equal(startBody(entry, 10, TAKE_BLUETOOTH, false).code_only, false)
  assert.equal(startBody(entry, 10, TAKE_BLUETOOTH, true).code_only, true)
  assert.equal(startBody(entry, 10, TAKE_ROLL_CALL, true).code_only, false)
  assert.equal(CODE_ONLY_LABEL, 'Code only (no location check)')
  assert.equal(startBody(entry, 10, TAKE_BLUETOOTH, false, 'phone-1').device_id, 'phone-1')
  assert.equal('device_id' in startBody(entry, 10, TAKE_BLUETOOTH, false, ''), false)
})

test('the website offers code with location and roll call, the app offers Bluetooth', () => {
  assert.deepEqual(
    takeMethods(false).map((entry) => entry.label),
    ['Code + location', 'Roll call'],
  )
  assert.deepEqual(
    takeMethods(true).map((entry) => entry.label),
    ['Bluetooth', 'Roll call'],
  )
})

test('a code only mark is flagged No location in the teacher list', () => {
  assert.deepEqual(markFlags({ status: 'present', method: 'code', no_location: true }), [
    NO_LOCATION_FLAG,
  ])
  assert.deepEqual(markFlags({ status: 'present', method: 'code', distance_m: 4 }), [])
  assert.deepEqual(markFlags({ status: 'absent', method: 'manual', no_location: true }), [])
  assert.equal(NO_LOCATION_FLAG, 'No location')
})

test('a code only mark sends the code and no position', () => {
  assert.deepEqual(codeOnlyMarkBody({ id: 9 }, ' 0427 '), { session_id: 9, code: '0427' })
})

test('the teacher list shows the time, the distance and the accuracy of a mark', () => {
  assert.equal(
    markMeta({ marked_at: '09:02', distance_m: 14, accuracy_m: 38 }),
    '09:02 · 14 m · ±38 m',
  )
  assert.equal(markMeta({ marked_at: '09:02' }), '09:02')
  assert.equal(markMeta({ accuracy_m: 61 }), '±61 m')
  assert.equal(markMeta({}), '')
  assert.equal(markMeta({ marked_at: 'x' }, (value) => `at ${value}`), 'at x')
})

test('history shows present over the roster total', () => {
  assert.equal(presentOfRoster({ present: 2, roster: 4 }), '2/4')
  assert.equal(presentOfRoster({}), '0/0')
})

test('closing an attendance nobody marked says it was not kept', () => {
  assert.equal(closedMessage({ status: 'closed', present: 3 }), CLOSED_TEXT)
  assert.equal(closedMessage({ removed: true }), EMPTY_CLOSED_TEXT)
  assert.equal(closedMessage({ removed: true, message: 'Gone.' }), 'Gone.')
})

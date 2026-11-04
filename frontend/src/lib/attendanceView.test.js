import assert from 'node:assert/strict'
import test from 'node:test'
import {
  OFFICE_TAB_HISTORY,
  STUDENT_TAB_MARK,
  TAKE_BLUETOOTH,
  TAKE_ROLL_CALL,
  TEACHER_TAB_TAKE,
  belowPass,
  classKey,
  deleteSheetQuestion,
  methodLabel,
  percentOf,
  rollNumber,
  sheetTitle,
  sortedRoster,
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
  assert.equal(methodLabel('web'), 'Web')
  assert.equal(methodLabel('manual'), 'By teacher')
  assert.equal(methodLabel('system'), 'Not marked')
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

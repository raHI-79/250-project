import assert from 'node:assert/strict'
import test from 'node:test'
import { dayKindNote, holidayOn } from './dayKind.js'

// Friday 13 and Saturday 14 November 2026 are weekly off days; Wednesday 11 is a named
// holiday; Saturday 21 is a make-up working day that follows Sunday.
const calendar = {
  off_days: [5, 6],
  holidays: [
    { date: '2026-11-11', name: 'Victory Day', working_day: false },
    { date: '2026-11-21', name: 'Make-up Saturday', working_day: true, follows_weekday_name: 'Sunday' },
  ],
}

test('a weekly off day is a holiday named after its weekday', () => {
  assert.deepEqual(holidayOn('2026-11-13', calendar), { name: 'Friday' })
  assert.deepEqual(holidayOn('2026-11-14', calendar), { name: 'Saturday' })
})

test('a named holiday carries its own name', () => {
  assert.deepEqual(holidayOn('2026-11-11', calendar), { name: 'Victory Day' })
})

test('a working day set on an off day is not a holiday', () => {
  assert.equal(holidayOn('2026-11-21', calendar), null)
})

test('an ordinary teaching day is not a holiday', () => {
  assert.equal(holidayOn('2026-11-08', calendar), null)
  assert.equal(holidayOn('2026-11-12', calendar), null)
})

test('nothing to go on means no holiday', () => {
  assert.equal(holidayOn('', calendar), null)
  assert.equal(holidayOn('2026-11-13', null), null)
  assert.equal(holidayOn('not a date', calendar), null)
  assert.equal(holidayOn('2026-11-13', { holidays: [] }), null)
})

test('the note for an empty day says what the chip means', () => {
  assert.match(dayKindNote({ holiday: true, label: 'Holiday · Friday' }), /still kept and announced/)
  assert.match(
    dayKindNote({ working_day: true, follows_weekday_name: 'Sunday' }),
    /follows the Sunday routine/,
  )
  assert.equal(dayKindNote(null), '')
})

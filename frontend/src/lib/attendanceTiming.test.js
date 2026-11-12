import assert from 'node:assert/strict'
import test from 'node:test'
import {
  LATE_AFTER,
  MODE_CUSTOM,
  MODE_UNTIL,
  attended,
  creditOf,
  creditPercent,
  creditText,
  extensionProblem,
  lateWeightProblem,
  hoursForClass,
  hoursForMinutes,
  minutesUntil,
  newTiming,
  statusLetter,
  timingBody,
  timingProblem,
} from './attendanceTiming.js'

const at = (hours, minutes, seconds = 0) => new Date(2026, 10, 12, hours, minutes, seconds)

test('the three marks have their own letters and late still counts as attending', () => {
  assert.equal(statusLetter('present'), 'P')
  assert.equal(statusLetter('late'), 'L')
  assert.equal(statusLetter('absent'), 'A')
  assert.equal(statusLetter(''), '')
  assert.equal(attended('late'), true)
  assert.equal(attended('absent'), false)
})

test('class hours follow the length of the class', () => {
  assert.equal(hoursForMinutes(50), 1)
  assert.equal(hoursForMinutes(60), 1)
  assert.equal(hoursForMinutes(61), 2)
  assert.equal(hoursForMinutes(120), 2)
  assert.equal(hoursForMinutes(121), 3)
  assert.equal(hoursForMinutes(600), 4)
  assert.equal(hoursForClass('09:00', '09:50'), 1)
  assert.equal(hoursForClass('09:00:00', '11:00:00'), 2)
  assert.equal(hoursForClass('09:00', '11:30'), 3)
  assert.equal(hoursForClass('', ''), 1)
})

test('a quick chip sends its minutes and one class hour by default', () => {
  const timing = newTiming(15)
  assert.equal(timingProblem(timing, at(9, 0)), '')
  assert.deepEqual(timingBody(timing), { hours: 1, late_after_minutes: 0, minutes: 15 })
})

test('a custom duration is 1 to 240 whole minutes', () => {
  const timing = { ...newTiming(), mode: MODE_CUSTOM, custom: '47', hours: 2 }
  assert.equal(timingProblem(timing, at(9, 0)), '')
  assert.deepEqual(timingBody(timing), { hours: 2, late_after_minutes: 0, minutes: 47 })
  assert.notEqual(timingProblem({ ...timing, custom: '0' }, at(9, 0)), '')
  assert.notEqual(timingProblem({ ...timing, custom: '241' }, at(9, 0)), '')
  assert.notEqual(timingProblem({ ...timing, custom: '12.5' }, at(9, 0)), '')
  assert.notEqual(timingProblem({ ...timing, custom: '' }, at(9, 0)), '')
  assert.equal(timingProblem({ ...timing, custom: '240' }, at(9, 0)), '')
})

test('open until takes a time later today', () => {
  assert.equal(minutesUntil('09:45', at(9, 0, 30)), 45)
  const timing = { ...newTiming(), mode: MODE_UNTIL, until: '09:45' }
  assert.equal(timingProblem(timing, at(9, 0)), '')
  assert.deepEqual(timingBody(timing), { hours: 1, late_after_minutes: 0, until: '09:45' })
  assert.notEqual(timingProblem(timing, at(9, 45)), '')
  assert.notEqual(timingProblem(timing, at(10, 0)), '')
  assert.notEqual(timingProblem({ ...timing, until: '' }, at(9, 0)), '')
  assert.notEqual(timingProblem({ ...timing, until: '14:00' }, at(9, 0)), '')
})

test('late after is never later than the end', () => {
  const timing = { ...newTiming(20), lateMode: LATE_AFTER, lateAfter: '5' }
  assert.equal(timingProblem(timing, at(9, 0)), '')
  assert.equal(timingBody(timing).late_after_minutes, 5)
  assert.equal(timingProblem({ ...timing, lateAfter: '20' }, at(9, 0)), '')
  assert.notEqual(timingProblem({ ...timing, lateAfter: '21' }, at(9, 0)), '')
  assert.notEqual(timingProblem({ ...timing, lateAfter: '' }, at(9, 0)), '')
  assert.notEqual(timingProblem({ ...timing, lateAfter: '0' }, at(9, 0)), '')
  assert.notEqual(timingProblem({ ...newTiming(), hours: 5 }, at(9, 0)), '')
})

test('an extension is any whole number of minutes up to four hours', () => {
  assert.equal(extensionProblem('7'), '')
  assert.equal(extensionProblem(5), '')
  assert.notEqual(extensionProblem('0'), '')
  assert.notEqual(extensionProblem('abc'), '')
  assert.notEqual(extensionProblem('500'), '')
})

test('two lates are worth one class and hours weigh the percentage', () => {
  assert.equal(creditOf(0, 2), 1)
  assert.equal(creditOf(2, 3), 3.5)
  assert.equal(creditOf(2, 3, 1), 5)
  assert.equal(creditPercent(3.5, 6), 58)
  assert.equal(creditPercent(1, 0), 0)
  assert.equal(creditText(3.5), '3.5')
  assert.equal(creditText(4), '4')
})

test('the late weight stays between 0 and 1', () => {
  assert.equal(lateWeightProblem('0.5'), '')
  assert.equal(lateWeightProblem('0'), '')
  assert.equal(lateWeightProblem('1'), '')
  assert.notEqual(lateWeightProblem('1.2'), '')
  assert.notEqual(lateWeightProblem('-1'), '')
  assert.notEqual(lateWeightProblem(''), '')
  assert.notEqual(lateWeightProblem('half'), '')
})

import assert from 'node:assert/strict'
import test from 'node:test'
import {
  ACTION_ADD_TO_FINAL,
  ACTION_REMOVE_FROM_FINAL,
  allows,
  canEditItem,
  canEditLink,
  itemEnded,
  itemStarted,
  secondsUntilClock,
} from './schedule.js'

const classAt = (date, start, end) => ({
  item_date: date,
  start_time: start,
  end_time: end,
  mode: 'room',
  status: 'confirmed',
})

test('decides what has started by the Dhaka clock, not the device clock', () => {
  const item = classAt('2026-11-06', '08:00', '09:30')
  // 2026-11-06 01:00 UTC is 07:00 in Dhaka, so the eight o'clock class has not begun.
  assert.equal(itemStarted(item, new Date('2026-11-06T01:00:00Z')), false)
  // 2026-11-06 02:30 UTC is 08:30 in Dhaka, so it has.
  assert.equal(itemStarted(item, new Date('2026-11-06T02:30:00Z')), true)
  assert.equal(itemEnded(item, new Date('2026-11-06T02:30:00Z')), false)
  assert.equal(itemEnded(item, new Date('2026-11-06T04:00:00Z')), true)
})

test('a class on the next Dhaka day has not started just before midnight', () => {
  const item = classAt('2026-11-06', '08:00', '09:30')
  // 2026-11-05 17:57 UTC is 23:57 on 5 November in Dhaka.
  assert.equal(itemStarted(item, new Date('2026-11-05T17:57:00Z')), false)
  // 2026-11-05 18:03 UTC is already 00:03 on 6 November, still before eight.
  assert.equal(itemStarted(item, new Date('2026-11-05T18:03:00Z')), false)
})

test('keeps editing open until the item starts and links open until it ends', () => {
  const item = classAt('2026-11-06', '08:00', '09:30')
  const before = new Date('2026-11-06T01:00:00Z')
  const during = new Date('2026-11-06T02:30:00Z')
  const after = new Date('2026-11-06T04:00:00Z')
  assert.equal(canEditItem(item, before), true)
  assert.equal(canEditItem(item, during), false)
  assert.equal(canEditItem({ ...item, status: 'cancelled' }, before), false)
  const online = { ...item, mode: 'online', meeting_link: 'https://meet.example/abc' }
  assert.equal(canEditLink(online, during), true)
  assert.equal(canEditLink(online, after), false)
  assert.equal(canEditLink(item, before), false)
})

test('counts down to a clock time in the Dhaka zone', () => {
  // 2026-11-06 02:00 UTC is 08:00 in Dhaka, so 09:30 is ninety minutes away.
  assert.equal(secondsUntilClock('09:30', new Date('2026-11-06T02:00:00Z')), 90 * 60)
  assert.equal(secondsUntilClock('07:00', new Date('2026-11-06T02:00:00Z')), 0)
  assert.equal(secondsUntilClock('nope', new Date('2026-11-06T02:00:00Z')), 0)
})

test('renders only the actions the server allowed', () => {
  const item = { allowed_actions: [ACTION_ADD_TO_FINAL] }
  assert.equal(allows(item, ACTION_ADD_TO_FINAL), true)
  assert.equal(allows(item, ACTION_REMOVE_FROM_FINAL), false)
  assert.equal(allows({}, ACTION_ADD_TO_FINAL), false)
  assert.equal(allows(null, ACTION_ADD_TO_FINAL), false)
})

import assert from 'node:assert/strict'
import test from 'node:test'
import { dayHeading, defaultUpcomingDay, shiftDate, today, tomorrow } from './time.js'

test('names today and tomorrow in a day heading', () => {
  assert.match(dayHeading(today()), /^Today · /)
  assert.match(dayHeading(tomorrow()), /^Tomorrow · /)
})

test('the upcoming views open on tomorrow', () => {
  assert.equal(defaultUpcomingDay(), tomorrow())
  assert.notEqual(defaultUpcomingDay(), today())
})

test('leaves a later day with its own name', () => {
  const heading = dayHeading(shiftDate(today(), 3))
  assert.doesNotMatch(heading, /Today|Tomorrow/)
  assert.match(heading, /^[A-Z][a-z]{2} \d{1,2} [A-Z][a-z]{2}$/)
})

test('says nothing about a broken date', () => {
  assert.equal(dayHeading('not-a-date'), '')
  assert.equal(dayHeading(''), '')
})

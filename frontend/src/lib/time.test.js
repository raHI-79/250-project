import assert from 'node:assert/strict'
import test from 'node:test'
import {
  APP_TIMEZONE,
  dateParts,
  dayHeading,
  dayLabel,
  defaultUpcomingDay,
  formatClock,
  formatDate,
  formatDateTime,
  formatTimeRange,
  fromParts,
  isoDate,
  isValidDate,
  shiftDate,
  stampLabel,
  today,
  tomorrow,
  weekdayName,
  WEEKDAYS,
} from './time.js'

test('reads every date in the Dhaka zone, never the browser zone', () => {
  assert.equal(APP_TIMEZONE, 'Asia/Dhaka')
  assert.equal(isoDate(new Date('2026-11-05T18:03:00Z')), '2026-11-06')
  assert.equal(isoDate(new Date('2026-11-05T17:57:00Z')), '2026-11-05')
})

test('names today and tomorrow in the Dhaka zone', () => {
  const at = new Date('2026-11-05T17:57:00Z')
  assert.equal(today(at), '2026-11-05')
  assert.equal(tomorrow(at), '2026-11-06')
  const justAfter = new Date('2026-11-05T18:03:00Z')
  assert.equal(today(justAfter), '2026-11-06')
  assert.equal(tomorrow(justAfter), '2026-11-07')
})

test('turns over at Dhaka midnight and holds through the early morning', () => {
  const cases = [
    { utc: '2026-11-05T17:57:00Z', today: '2026-11-05', weekday: 'Thursday' },
    { utc: '2026-11-05T18:03:00Z', today: '2026-11-06', weekday: 'Friday' },
    { utc: '2026-11-05T23:59:00Z', today: '2026-11-06', weekday: 'Friday' },
    { utc: '2026-11-06T00:01:00Z', today: '2026-11-06', weekday: 'Friday' },
    { utc: '2026-11-06T17:59:00Z', today: '2026-11-06', weekday: 'Friday' },
    { utc: '2026-11-06T18:01:00Z', today: '2026-11-07', weekday: 'Saturday' },
  ]
  for (const entry of cases) {
    const at = new Date(entry.utc)
    assert.equal(today(at), entry.today, entry.utc)
    assert.equal(weekdayName(entry.today), entry.weekday, entry.utc)
  }
})

test('writes every date with the day before the month', () => {
  assert.equal(formatDate('2026-11-06'), 'Fri 6 Nov 2026')
  assert.equal(formatDate('2026-01-01'), 'Thu 1 Jan 2026')
  assert.equal(formatDate('2026-12-31'), 'Thu 31 Dec 2026')
  assert.equal(formatDate(''), '')
  assert.equal(formatDate('not a date'), '')
})

test('splits and rebuilds a date without any locale', () => {
  assert.deepEqual(dateParts('2026-11-06'), { day: '6', month: '11', year: '2026' })
  assert.equal(fromParts({ day: '6', month: '11', year: '2026' }), '2026-11-06')
  assert.equal(fromParts({ day: '06', month: '11', year: '2026' }), '2026-11-06')
  assert.equal(fromParts({ day: '31', month: '2', year: '2026' }), '')
  assert.equal(fromParts({ day: '6', month: '13', year: '2026' }), '')
  assert.equal(fromParts({ day: '', month: '11', year: '2026' }), '')
  assert.deepEqual(dateParts(''), { day: '', month: '', year: '' })
})

test('accepts only whole days written as year, month and day', () => {
  assert.equal(isValidDate('2026-11-06'), true)
  assert.equal(isValidDate('2026-02-29'), false)
  assert.equal(isValidDate('2026-13-01'), false)
  assert.equal(isValidDate('06/11/2026'), false)
  assert.equal(isValidDate('11/06/2026'), false)
  assert.equal(isValidDate(''), false)
})

test('walks whole days without a zone of its own', () => {
  assert.equal(shiftDate('2026-11-05', 1), '2026-11-06')
  assert.equal(shiftDate('2026-11-06', -1), '2026-11-05')
  assert.equal(shiftDate('2026-11-30', 1), '2026-12-01')
  assert.equal(shiftDate('2026-12-31', 1), '2027-01-01')
  assert.equal(shiftDate('not a date', 1), 'not a date')
})

test('names today and tomorrow in a day heading', () => {
  const at = new Date('2026-11-05T17:57:00Z')
  assert.equal(dayHeading('2026-11-05', at), 'Today · Thu 5 Nov')
  assert.equal(dayHeading('2026-11-06', at), 'Tomorrow · Fri 6 Nov')
  assert.equal(dayHeading('2026-11-09', at), 'Mon 9 Nov')
})

test('the upcoming views open on tomorrow', () => {
  const at = new Date('2026-11-05T17:57:00Z')
  assert.equal(defaultUpcomingDay(at), tomorrow(at))
  assert.notEqual(defaultUpcomingDay(at), today(at))
})

test('says nothing about a broken date', () => {
  assert.equal(dayHeading('not-a-date'), '')
  assert.equal(dayHeading(''), '')
  assert.equal(dayLabel(''), '')
  assert.equal(stampLabel(''), '')
})

test('labels a past day by how far back it is', () => {
  const at = new Date('2026-11-06T06:00:00Z')
  assert.equal(dayLabel('2026-11-06T07:00:00Z', at), 'Today')
  assert.equal(dayLabel('2026-11-05T07:00:00Z', at), 'Yesterday')
  assert.equal(dayLabel('2026-11-02T07:00:00Z', at), 'Monday')
  assert.equal(dayLabel('2026-10-20T07:00:00Z', at), 'Tue 20 Oct 2026')
})

test('stamps a time in the Dhaka zone', () => {
  const at = new Date('2026-11-06T06:00:00Z')
  assert.equal(stampLabel('2026-11-06T05:30:00Z', at), 'Today 11:30 AM')
  assert.equal(stampLabel('2026-11-05T05:30:00Z', at), 'Yesterday 11:30 AM')
  assert.equal(stampLabel('2026-10-20T05:30:00Z', at), '20 Oct, 11:30 AM')
  assert.equal(stampLabel('2025-10-20T05:30:00Z', at), '20 Oct 2025, 11:30 AM')
})

test('writes a full stamp day first', () => {
  assert.equal(formatDateTime('2026-11-06T05:30:00Z'), 'Fri 6 Nov 2026, 11:30 AM')
  assert.equal(formatDateTime('nope'), '')
})

test('writes clock times with a twelve hour suffix', () => {
  assert.equal(formatClock('08:00'), '8:00 AM')
  assert.equal(formatClock('13:05'), '1:05 PM')
  assert.equal(formatClock('00:30'), '12:30 AM')
  assert.equal(formatClock('12:00'), '12:00 PM')
  assert.equal(formatClock('nope'), '')
  assert.equal(formatTimeRange('08:00', '09:30'), '8:00 AM – 9:30 AM')
  assert.equal(formatTimeRange('08:00', ''), '8:00 AM')
  assert.equal(formatTimeRange('', '09:30'), '9:30 AM')
})

test('keeps the weekday names in Sunday first order', () => {
  assert.equal(WEEKDAYS.length, 7)
  assert.equal(WEEKDAYS[0], 'Sunday')
  assert.equal(WEEKDAYS[5], 'Friday')
  assert.equal(WEEKDAYS[6], 'Saturday')
})

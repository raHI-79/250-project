import { weekdayName, weekdayNumber } from './time.js'

// dayKindNote explains an off day, a holiday or a working day in one sentence, for a day
// that carries no item of its own.
export const dayKindNote = (kind) => {
  if (!kind) return ''
  if (kind.working_day) {
    return `The office made this a working day, so it follows the ${kind.follows_weekday_name} routine.`
  }
  return 'No fixed routine is generated for this day. A class or exam created for it is still kept and announced.'
}

// holidayOn reads the calendar /api/calendar answers with and tells whether a date is a
// day the routine leaves empty. It mirrors the rule on the server: a working day the
// office set wins over everything, a named holiday comes next, and a weekly off day is
// named after its weekday. It answers null for an ordinary teaching day.
export const holidayOn = (date, calendar) => {
  if (!date || !calendar) return null
  const entry = (calendar.holidays || []).find((holiday) => holiday.date === date)
  if (entry && entry.working_day) return null
  if (entry) return { name: entry.name || weekdayName(date) }
  const weekday = weekdayNumber(date)
  if (weekday >= 0 && (calendar.off_days || []).includes(weekday)) {
    return { name: weekdayName(date) }
  }
  return null
}

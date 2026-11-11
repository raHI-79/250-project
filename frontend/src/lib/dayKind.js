import { weekdayName, weekdayNumber } from './time.js'

export const dayKindNote = (kind) => {
  if (!kind) return ''
  if (kind.working_day) {
    return `The office made this a working day, so it follows the ${kind.follows_weekday_name} routine.`
  }
  return 'No fixed routine is generated for this day. A class or exam created for it is still kept and announced.'
}

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

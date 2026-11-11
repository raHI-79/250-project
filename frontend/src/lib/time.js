export const APP_TIMEZONE = 'Asia/Dhaka'

export const WEEKDAYS = [
  'Sunday',
  'Monday',
  'Tuesday',
  'Wednesday',
  'Thursday',
  'Friday',
  'Saturday',
]

const MONTHS = [
  'Jan',
  'Feb',
  'Mar',
  'Apr',
  'May',
  'Jun',
  'Jul',
  'Aug',
  'Sep',
  'Oct',
  'Nov',
  'Dec',
]

const DATE_PATTERN = /^(\d{4})-(\d{2})-(\d{2})$/

const UNITS = [
  { limit: 60, divisor: 1, name: 'second' },
  { limit: 3600, divisor: 60, name: 'minute' },
  { limit: 86400, divisor: 3600, name: 'hour' },
  { limit: 604800, divisor: 86400, name: 'day' },
  { limit: 2629800, divisor: 604800, name: 'week' },
  { limit: 31557600, divisor: 2629800, name: 'month' },
]

const pad = (value) => String(value).padStart(2, '0')

const asDate = (value) => {
  if (value instanceof Date) return Number.isNaN(value.getTime()) ? null : value
  if (typeof value === 'string' && DATE_PATTERN.test(value)) {
    const stamp = new Date(`${value}T00:00:00Z`)
    return Number.isNaN(stamp.getTime()) ? null : stamp
  }
  if (value === '' || value === null || value === undefined) return null
  const stamp = new Date(value)
  return Number.isNaN(stamp.getTime()) ? null : stamp
}

// zoneParts reads a stamp in APP_TIMEZONE. Everything that has to agree with the server
// goes through it, so that a phone set to another zone still sees the Dhaka day.
const partFormatter = new Intl.DateTimeFormat('en-US', {
  timeZone: APP_TIMEZONE,
  year: 'numeric',
  month: '2-digit',
  day: '2-digit',
  hour: '2-digit',
  minute: '2-digit',
  hour12: false,
})

const zoneParts = (value) => {
  const date = asDate(value)
  if (!date) return null
  const parts = {}
  for (const part of partFormatter.formatToParts(date)) {
    if (part.type !== 'literal') parts[part.type] = part.value
  }
  const hour = parts.hour === '24' ? '00' : parts.hour
  return {
    year: Number(parts.year),
    month: Number(parts.month),
    day: Number(parts.day),
    hour: Number(hour),
    minute: Number(parts.minute),
  }
}

export const isoDate = (value) => {
  const parts = zoneParts(value)
  if (!parts) return ''
  return `${parts.year}-${pad(parts.month)}-${pad(parts.day)}`
}

export const today = (now = new Date()) => isoDate(now)

export const tomorrow = (now = new Date()) => shiftDate(today(now), 1)

export const defaultUpcomingDay = (now = new Date()) => tomorrow(now)

export const isValidDate = (value) => {
  const matched = DATE_PATTERN.exec(String(value ?? ''))
  if (!matched) return false
  const [, year, month, day] = matched
  const stamp = new Date(`${year}-${month}-${day}T00:00:00Z`)
  if (Number.isNaN(stamp.getTime())) return false
  return (
    stamp.getUTCFullYear() === Number(year) &&
    stamp.getUTCMonth() + 1 === Number(month) &&
    stamp.getUTCDate() === Number(day)
  )
}

// shiftDate walks whole days on the calendar. It never touches a zone, so a day never
// gains or loses an hour on the way.
export const shiftDate = (value, days) => {
  if (!isValidDate(value)) return value
  const stamp = new Date(`${value}T00:00:00Z`)
  stamp.setUTCDate(stamp.getUTCDate() + days)
  return `${stamp.getUTCFullYear()}-${pad(stamp.getUTCMonth() + 1)}-${pad(stamp.getUTCDate())}`
}

export const daysBetween = (from, to) => {
  if (!isValidDate(from) || !isValidDate(to)) return 0
  const start = new Date(`${from}T00:00:00Z`).getTime()
  const end = new Date(`${to}T00:00:00Z`).getTime()
  return Math.round((end - start) / 86400000)
}

export const weekdayNumber = (value) => {
  if (!isValidDate(value)) return -1
  return new Date(`${value}T00:00:00Z`).getUTCDay()
}

export const weekdayName = (value) => {
  const day = weekdayNumber(value)
  return day < 0 ? '' : WEEKDAYS[day]
}

export const dateParts = (value) => {
  const matched = DATE_PATTERN.exec(String(value ?? ''))
  if (!matched) return { day: '', month: '', year: '' }
  const [, year, month, day] = matched
  return { day: String(Number(day)), month: String(Number(month)), year }
}

// fromParts builds a date out of the three numbers a date field collects. It is the only
// place that turns what somebody typed into a date, so no locale can reorder it.
export const fromParts = ({ day, month, year } = {}) => {
  const dayNumber = Number(String(day ?? '').trim())
  const monthNumber = Number(String(month ?? '').trim())
  const yearNumber = Number(String(year ?? '').trim())
  if (!Number.isInteger(dayNumber) || !Number.isInteger(monthNumber) || !Number.isInteger(yearNumber)) {
    return ''
  }
  if (String(year ?? '').trim().length !== 4) return ''
  const stamp = `${yearNumber}-${pad(monthNumber)}-${pad(dayNumber)}`
  return isValidDate(stamp) ? stamp : ''
}

const shortDay = (value) => {
  const name = weekdayName(value)
  return name ? name.slice(0, 3) : ''
}

// formatDate writes "Fri 6 Nov 2026", the one shape every date wears in the portal and
// in the app.
export const formatDate = (value) => {
  const stamp = isValidDate(value) ? value : isoDate(value)
  if (!isValidDate(stamp)) return ''
  const [year, month, day] = stamp.split('-')
  return `${shortDay(stamp)} ${Number(day)} ${MONTHS[Number(month) - 1]} ${year}`
}

export const shortDate = (value) => {
  const stamp = isValidDate(value) ? value : isoDate(value)
  if (!isValidDate(stamp)) return ''
  const [, month, day] = stamp.split('-')
  return `${shortDay(stamp)} ${Number(day)} ${MONTHS[Number(month) - 1]}`
}

export const dayHeading = (value, now = new Date()) => {
  const stamp = isValidDate(value) ? value : isoDate(value)
  if (!isValidDate(stamp)) return ''
  const label = shortDate(stamp)
  if (stamp === today(now)) return `Today · ${label}`
  if (stamp === tomorrow(now)) return `Tomorrow · ${label}`
  return label
}

export const dayLabel = (value, now = new Date()) => {
  const stamp = isoDate(value)
  if (!stamp) return ''
  const days = daysBetween(stamp, today(now))
  if (days <= 0) return 'Today'
  if (days === 1) return 'Yesterday'
  if (days < 7) return weekdayName(stamp)
  return formatDate(stamp)
}

export const formatClock = (value) => {
  if (typeof value !== 'string') return ''
  const [rawHour, rawMinute] = value.split(':')
  const hour = Number(rawHour)
  const minute = Number(rawMinute)
  if (!Number.isInteger(hour) || !Number.isInteger(minute)) return ''
  if (hour < 0 || hour > 23 || minute < 0 || minute > 59) return ''
  const suffix = hour < 12 ? 'AM' : 'PM'
  const display = hour % 12 === 0 ? 12 : hour % 12
  return `${display}:${pad(minute)} ${suffix}`
}

export const formatTimeRange = (start, end) => {
  const from = formatClock(start)
  const to = formatClock(end)
  if (!from) return to
  if (!to) return from
  return `${from} – ${to}`
}

const zoneClock = (value) => {
  const parts = zoneParts(value)
  if (!parts) return ''
  return formatClock(`${pad(parts.hour)}:${pad(parts.minute)}`)
}

export const stampLabel = (value, now = new Date()) => {
  const stamp = isoDate(value)
  if (!stamp) return ''
  const clock = zoneClock(value)
  const days = daysBetween(stamp, today(now))
  if (days <= 0) return `Today ${clock}`
  if (days === 1) return `Yesterday ${clock}`
  const [year, month, day] = stamp.split('-')
  const short = `${Number(day)} ${MONTHS[Number(month) - 1]}`
  if (year !== today(now).slice(0, 4)) return `${short} ${year}, ${clock}`
  return `${short}, ${clock}`
}

export const editedLabel = (value) => {
  if (!value) return ''
  const stamp = stampLabel(value)
  return stamp ? `Edited ${stamp}` : ''
}

export const formatDateTime = (value) => {
  const stamp = isoDate(value)
  if (!stamp) return ''
  return `${formatDate(stamp)}, ${zoneClock(value)}`
}

// monthName and longDate keep the month names out of the browser locale, so that a phone
// set to another language still reads the same as the portal.
export const monthName = (value) => {
  const stamp = isValidDate(value) ? value : isoDate(value)
  if (!isValidDate(stamp)) return ''
  return MONTHS[Number(stamp.slice(5, 7)) - 1]
}

const LONG_MONTHS = [
  'January',
  'February',
  'March',
  'April',
  'May',
  'June',
  'July',
  'August',
  'September',
  'October',
  'November',
  'December',
]

export const longMonth = (value) => {
  const stamp = isValidDate(value) ? value : isoDate(value)
  if (!isValidDate(stamp)) return ''
  return LONG_MONTHS[Number(stamp.slice(5, 7)) - 1]
}

// monthHeading writes "November 2026" for a YYYY-MM key.
export const monthHeading = (key) => {
  const [year, month] = String(key || '').split('-')
  const index = Number(month) - 1
  if (!year || !LONG_MONTHS[index]) return ''
  return `${LONG_MONTHS[index]} ${year}`
}

// longDate writes "Friday 6 November", the greeting line of the Today pages.
export const longDate = (value) => {
  const stamp = isValidDate(value) ? value : isoDate(value)
  if (!isValidDate(stamp)) return ''
  return `${weekdayName(stamp)} ${Number(stamp.slice(8, 10))} ${longMonth(stamp)}`
}

export const dayOfMonth = (value) => {
  const stamp = isValidDate(value) ? value : isoDate(value)
  if (!isValidDate(stamp)) return ''
  return stamp.slice(8, 10)
}

// hourIn reads the hour of a stamp in APP_TIMEZONE, for the greeting of the Today pages.
export const hourIn = (value = new Date()) => {
  const parts = zoneParts(value)
  return parts ? parts.hour : 0
}

// clockIn reads the wall clock of a stamp in APP_TIMEZONE as HH:MM.
export const clockIn = (value = new Date()) => {
  const parts = zoneParts(value)
  if (!parts) return ''
  return `${pad(parts.hour)}:${pad(parts.minute)}`
}

// minutesOfDay turns an HH:MM clock into minutes since midnight.
export const minutesOfDay = (clock) => {
  const [hour, minute] = String(clock || '')
    .split(':')
    .map((part) => Number(part))
  if (!Number.isInteger(hour) || !Number.isInteger(minute)) return -1
  return hour * 60 + minute
}

export const relativeTime = (value) => {
  const date = asDate(value)
  if (!date) return ''
  const seconds = (date.getTime() - Date.now()) / 1000
  const magnitude = Math.abs(seconds)
  if (magnitude < 45) return 'just now'
  const formatter = new Intl.RelativeTimeFormat('en', { numeric: 'auto' })
  const unit = UNITS.find((entry) => magnitude < entry.limit)
  if (!unit) return formatter.format(Math.round(seconds / 31557600), 'year')
  return formatter.format(Math.round(seconds / unit.divisor), unit.name)
}

export const overlaps = (startA, endA, startB, endB) => startA < endB && startB < endA

const UNITS = [
  { limit: 60, divisor: 1, name: 'second' },
  { limit: 3600, divisor: 60, name: 'minute' },
  { limit: 86400, divisor: 3600, name: 'hour' },
  { limit: 604800, divisor: 86400, name: 'day' },
  { limit: 2629800, divisor: 604800, name: 'week' },
  { limit: 31557600, divisor: 2629800, name: 'month' },
]

export const relativeTime = (value) => {
  const date = value instanceof Date ? value : new Date(value)
  if (Number.isNaN(date.getTime())) return ''
  const seconds = (date.getTime() - Date.now()) / 1000
  const magnitude = Math.abs(seconds)
  if (magnitude < 45) return 'just now'
  const formatter = new Intl.RelativeTimeFormat('en', { numeric: 'auto' })
  const unit = UNITS.find((entry) => magnitude < entry.limit)
  if (!unit) return formatter.format(Math.round(seconds / 31557600), 'year')
  return formatter.format(Math.round(seconds / unit.divisor), unit.name)
}

export const formatDate = (value) => {
  const date = value instanceof Date ? value : new Date(value)
  if (Number.isNaN(date.getTime())) return ''
  return date.toLocaleDateString('en-GB', { day: '2-digit', month: 'short', year: 'numeric' })
}

export const WEEKDAYS = [
  'Sunday',
  'Monday',
  'Tuesday',
  'Wednesday',
  'Thursday',
  'Friday',
  'Saturday',
]

export const formatClock = (value) => {
  if (typeof value !== 'string') return ''
  const [rawHour, rawMinute] = value.split(':')
  const hour = Number(rawHour)
  const minute = Number(rawMinute)
  if (!Number.isInteger(hour) || !Number.isInteger(minute)) return ''
  const suffix = hour < 12 ? 'AM' : 'PM'
  const display = hour % 12 === 0 ? 12 : hour % 12
  return `${display}:${String(minute).padStart(2, '0')} ${suffix}`
}

export const formatTimeRange = (start, end) => {
  const from = formatClock(start)
  const to = formatClock(end)
  if (!from) return to
  if (!to) return from
  return `${from} – ${to}`
}

export const formatDateTime = (value) => {
  const date = value instanceof Date ? value : new Date(value)
  if (Number.isNaN(date.getTime())) return ''
  return date.toLocaleString('en-GB', {
    day: '2-digit',
    month: 'short',
    year: 'numeric',
    hour: 'numeric',
    minute: '2-digit',
    hour12: true,
  })
}

export const overlaps = (startA, endA, startB, endB) => startA < endB && startB < endA

export const isoDate = (value) => {
  const date = value instanceof Date ? value : new Date(value)
  if (Number.isNaN(date.getTime())) return ''
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  return `${date.getFullYear()}-${month}-${day}`
}

export const tomorrow = () => {
  const date = new Date()
  date.setDate(date.getDate() + 1)
  return isoDate(date)
}

export const today = () => isoDate(new Date())

export const shiftDate = (value, days) => {
  const date = new Date(`${value}T00:00:00`)
  if (Number.isNaN(date.getTime())) return value
  date.setDate(date.getDate() + days)
  return isoDate(date)
}

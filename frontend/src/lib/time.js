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

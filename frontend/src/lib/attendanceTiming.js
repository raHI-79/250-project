export const QUICK_MINUTES = [5, 10, 15, 20, 30]

export const QUICK_EXTENSIONS = [5, 10]

export const MIN_MINUTES = 1

export const MAX_MINUTES = 240

export const HOUR_CHOICES = [1, 2, 3, 4]

export const MODE_QUICK = 'quick'

export const MODE_CUSTOM = 'custom'

export const MODE_UNTIL = 'until'

export const LATE_AT_END = 'end'

export const LATE_AFTER = 'after'

export const STATUS_PRESENT = 'present'

export const STATUS_LATE = 'late'

export const STATUS_ABSENT = 'absent'

export const STATUS_LETTERS = { present: 'P', late: 'L', absent: 'A' }

export const STATUS_LABELS = { present: 'Present', late: 'Late', absent: 'Absent' }

export const statusLetter = (status) => STATUS_LETTERS[status] || ''

export const attended = (status) => status === STATUS_PRESENT || status === STATUS_LATE

export const newTiming = (minutes = 10) => ({
  mode: MODE_QUICK,
  minutes,
  custom: '',
  until: '',
  lateMode: LATE_AT_END,
  lateAfter: '',
  hours: 1,
})

const whole = (value) => {
  const text = String(value ?? '').trim()
  if (!/^\d+$/.test(text)) return NaN
  return Number(text)
}

export const hoursForMinutes = (minutes) => {
  const value = Number(minutes)
  if (!Number.isFinite(value) || value <= 0) return 1
  return Math.min(4, Math.ceil(value / 60))
}

const clock = (value) => {
  const match = /^(\d{1,2}):(\d{2})/.exec(String(value || ''))
  if (!match) return NaN
  return Number(match[1]) * 60 + Number(match[2])
}

export const hoursForClass = (startTime, endTime) => {
  const from = clock(startTime)
  const to = clock(endTime)
  if (Number.isNaN(from) || Number.isNaN(to) || to <= from) return 1
  return hoursForMinutes(to - from)
}

export const minutesUntil = (until, now = new Date()) => {
  const target = clock(until)
  if (Number.isNaN(target)) return NaN
  const current = now.getHours() * 60 + now.getMinutes() + now.getSeconds() / 60
  return Math.ceil(target - current)
}

export const timingMinutes = (timing, now = new Date()) => {
  if (timing.mode === MODE_CUSTOM) return whole(timing.custom)
  if (timing.mode === MODE_UNTIL) return minutesUntil(timing.until, now)
  return Number(timing.minutes)
}

export const timingProblem = (timing, now = new Date()) => {
  const minutes = timingMinutes(timing, now)
  if (timing.mode === MODE_UNTIL) {
    if (Number.isNaN(minutes) || minutes < MIN_MINUTES) return 'Open until a time later today, after now.'
    if (minutes > MAX_MINUTES) return 'Attendance cannot stay open more than 4 hours.'
  } else if (!Number.isInteger(minutes) || minutes < MIN_MINUTES || minutes > MAX_MINUTES) {
    return 'Enter a number of minutes from 1 to 240.'
  }
  if (timing.lateMode === LATE_AFTER) {
    const late = whole(timing.lateAfter)
    if (!Number.isInteger(late) || late < 1) return 'Late after needs a number of minutes.'
    if (late > minutes) return 'Late after cannot be later than the end of the attendance.'
  }
  if (!HOUR_CHOICES.includes(Number(timing.hours))) return 'Class hours are 1, 2, 3 or 4.'
  return ''
}

export const timingBody = (timing) => {
  const body = {
    hours: Number(timing.hours) || 1,
    late_after_minutes: timing.lateMode === LATE_AFTER ? whole(timing.lateAfter) || 0 : 0,
  }
  if (timing.mode === MODE_UNTIL) return { ...body, until: String(timing.until).slice(0, 5) }
  if (timing.mode === MODE_CUSTOM) return { ...body, minutes: whole(timing.custom) }
  return { ...body, minutes: Number(timing.minutes) }
}

export const extensionProblem = (value) => {
  const minutes = whole(value)
  if (!Number.isInteger(minutes) || minutes < MIN_MINUTES || minutes > MAX_MINUTES) {
    return 'Enter a number of minutes from 1 to 240.'
  }
  return ''
}

export const creditOf = (presentHours, lateHours, weight = 0.5) =>
  Number(presentHours || 0) + Number(lateHours || 0) * Number(weight)

export const creditPercent = (credit, heldHours) => {
  const held = Number(heldHours)
  if (!held || held <= 0) return 0
  return Math.round((Number(credit || 0) * 100) / held)
}

export const creditText = (credit) => String(Math.round(Number(credit || 0) * 100) / 100)

export const lateBoundaryNote = (session) => {
  if (!session?.late_after || !session?.original_ends_at) return ''
  if (session.late_after === session.original_ends_at) {
    return 'Marks after the original end are saved as Late.'
  }
  return 'Marks after the late time are saved as Late.'
}

export const lateWeightProblem = (value) => {
  const text = String(value ?? '').trim()
  const weight = Number(text)
  if (text === '' || !Number.isFinite(weight) || weight < 0 || weight > 1) {
    return 'The late weight is a number between 0 and 1.'
  }
  return ''
}

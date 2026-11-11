import { clockIn, formatClock, minutesOfDay, today } from './time.js'

export const MODE_ROOM = 'room'

export const MODE_ONLINE = 'online'

export const ORIGIN_FIXED = 'fixed'

export const ORIGIN_CR = 'cr'

export const ORIGIN_TEACHER_EXAM = 'teacher_exam'

export const STATUS_AWAITING = 'awaiting_teacher'

export const STATUS_NO_RESPONSE = 'no_response'

export const STATUS_CONFIRMED = 'confirmed'

export const STATUS_CANCELLED = 'cancelled'

export const STATUS_TEACHER_ASK = 'teacher_reschedule_request'

export const STATUS_CR_ASK = 'cr_reschedule_request'

export const STATUS_DECLINED = 'reschedule_declined'

export const STATUS_RESCHEDULED = 'rescheduled'

export const STATUS_SCHEDULED = 'scheduled'

const STATUS_TONES = {
  [STATUS_AWAITING]: 'pending',
  [STATUS_NO_RESPONSE]: 'pending',
  [STATUS_CONFIRMED]: 'approved',
  [STATUS_CANCELLED]: 'cancelled',
  [STATUS_TEACHER_ASK]: 'pending',
  [STATUS_CR_ASK]: 'pending',
  [STATUS_DECLINED]: 'rejected',
  [STATUS_RESCHEDULED]: 'approved',
  [STATUS_SCHEDULED]: 'active',
}

const STATUS_LABELS = {
  [STATUS_AWAITING]: 'Awaiting teacher',
  [STATUS_NO_RESPONSE]: 'No response',
  [STATUS_CONFIRMED]: 'Confirmed',
  [STATUS_CANCELLED]: 'Cancelled',
  [STATUS_TEACHER_ASK]: 'Teacher requested reschedule',
  [STATUS_CR_ASK]: 'Reschedule requested',
  [STATUS_DECLINED]: 'Not possible to reschedule',
  [STATUS_RESCHEDULED]: 'Rescheduled',
  [STATUS_SCHEDULED]: 'Scheduled',
}

export const statusLabel = (status) => STATUS_LABELS[status] || status

export const statusTone = (status) => STATUS_TONES[status] || 'unavailable'

export const teacherStatusLabel = (item) => {
  if (!item) return ''
  if (item.status === STATUS_AWAITING) return 'Undecided'
  if (item.kind === 'exam' && item.status === STATUS_SCHEDULED) return 'Scheduled exam'
  return statusLabel(item.status)
}

export const KIND_LABELS = { class: 'Class', lab: 'Lab', exam: 'Exam' }

export const kindLabel = (kind) => KIND_LABELS[kind] || kind

export const ORIGIN_LABELS = {
  [ORIGIN_FIXED]: 'Fixed routine',
  [ORIGIN_CR]: 'By CR',
  [ORIGIN_TEACHER_EXAM]: 'By teacher',
}

export const originLabel = (origin) => ORIGIN_LABELS[origin] || ''

export const ACTION_ADD_TO_FINAL = 'add_to_final'

export const ACTION_REMOVE_FROM_FINAL = 'remove_from_final'

export const ACTION_REQUEST_RESCHEDULE = 'request_reschedule'

export const ACTION_SET_SLOT = 'set_slot'

export const ACTION_SET_TIME = 'set_time'

export const ACTION_CONFIRM = 'confirm'

export const ACTION_ACCEPT = 'accept_reschedule'

export const ACTION_DECLINE = 'decline_reschedule'

export const ACTION_EDIT = 'edit'

export const ACTION_EDIT_LINK = 'edit_link'

export const ACTION_CANCEL = 'cancel'

export const ACTION_DELETE = 'delete'

export const allows = (item, action) => (item?.allowed_actions || []).includes(action)

export const isOnline = (item) => item?.mode === MODE_ONLINE

export const needsLink = (item) => isOnline(item) && !String(item?.meeting_link || '').trim()

export const placeOf = (item) => {
  if (!item) return ''
  if (isOnline(item)) return needsLink(item) ? 'Online · Link coming soon' : 'Online'
  return item.room || ''
}

export const SECTION_ALL = 'ALL'

export const sectionLabel = (section) =>
  section === SECTION_ALL ? 'Both sections' : `Section ${section}`

export const sectionShort = (section) => (section === SECTION_ALL ? 'Both' : section)

export const courseLine = (item) => {
  if (!item) return ''
  return item.course_title || item.course_code || ''
}

export const courseCode = (item) => {
  if (!item) return ''
  return item.course_title ? item.course_code || '' : ''
}

export const slotError = (mode, room, link) => {
  const address = String(link || '').trim()
  if (mode === MODE_ONLINE) {
    if (address && !/^https?:\/\/\S+$/i.test(address))
      return 'The meeting link must start with http:// or https://.'
    return ''
  }
  if (!String(room || '').trim()) return 'A room is required for a class in a room.'
  return ''
}

export const startsInText = (seconds) => {
  if (seconds <= 0) return 'Now'
  const hours = Math.floor(seconds / 3600)
  const minutes = Math.floor((seconds % 3600) / 60)
  if (hours > 0) return `in ${hours}h ${String(minutes).padStart(2, '0')}m`
  if (minutes > 0) return `in ${minutes} min`
  return 'in under a minute'
}

export const secondsUntilClock = (clock, now = new Date()) => {
  const target = minutesOfDay(clock)
  if (target < 0) return 0
  const here = minutesOfDay(clockIn(now))
  if (here < 0) return 0
  return Math.max(0, (target - here) * 60 - now.getSeconds())
}

export const teachingPeriods = (periods) => (periods || []).filter((period) => !period.is_break)

export const periodEndOptions = (periods, start) => {
  const ordered = [...(periods || [])].sort((left, right) =>
    left.start_time.localeCompare(right.start_time),
  )
  const first = ordered.findIndex((period) => period.start_time === start)
  if (first < 0) return []
  const options = []
  for (let index = first; index < ordered.length; index += 1) {
    if (ordered[index].is_break) break
    options.push(ordered[index])
  }
  return options
}

export const periodRangeError = (periods, start, end) => {
  if (!start || !end) return ''
  if (start >= end) return 'The start time must be before the end time.'
  const teaching = teachingPeriods(periods)
  if (teaching.length === 0) return ''
  const opening = teaching.reduce((lowest, period) =>
    period.start_time < lowest ? period.start_time : lowest, teaching[0].start_time)
  const closing = teaching.reduce((highest, period) =>
    period.end_time > highest ? period.end_time : highest, teaching[0].end_time)
  if (start < opening || end > closing) {
    return `Classes run from ${formatClock(opening)} to ${formatClock(closing)}.`
  }
  const broken = (periods || []).find(
    (period) => period.is_break && start < period.end_time && period.start_time < end,
  )
  if (broken) return `Classes cannot run during ${broken.label}.`
  return ''
}

export const itemStarted = (item, now = new Date()) => {
  if (!item?.item_date || !item?.start_time) return false
  return `${item.item_date} ${item.start_time}` < `${today(now)} ${clockIn(now)}`
}

export const itemEnded = (item, now = new Date()) => {
  if (!item?.item_date || !item?.end_time) return false
  return itemStarted({ item_date: item.item_date, start_time: item.end_time }, now)
}

export const canEditItem = (item, now = new Date()) =>
  Boolean(item) && item.status !== STATUS_CANCELLED && !itemStarted(item, now)

export const canEditLink = (item, now = new Date()) =>
  isOnline(item) && item?.status !== STATUS_CANCELLED && !itemEnded(item, now)

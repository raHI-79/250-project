export const TEACHER_TAB_TAKE = 'take'

export const TEACHER_TAB_HISTORY = 'history'

export const TEACHER_TAB_CLASSES = 'classes'

export const TEACHER_TAB_PHONES = 'phones'

export const STUDENT_TAB_MARK = 'mark'

export const STUDENT_TAB_MINE = 'mine'

export const STUDENT_TAB_PHONE = 'phone'

export const OFFICE_TAB_HISTORY = 'history'

export const OFFICE_TAB_SHEETS = 'sheets'

export const OFFICE_TAB_PHONES = 'phones'

export const TAKE_BLUETOOTH = 'bluetooth'

export const TAKE_ROLL_CALL = 'roll_call'

export const methodLabel = (method) => {
  if (method === 'app') return 'Bluetooth'
  if (method === 'code') return 'Code'
  if (method === 'web') return 'Web'
  if (method === 'manual') return 'By teacher'
  if (method === 'system') return 'Not marked'
  return ''
}

export const sheetTitle = (entry) =>
  `${entry.course_code}${entry.course_title ? ` ${entry.course_title}` : ''} · ${entry.batch} ${entry.section}`

export const deleteSheetQuestion = (entry) =>
  `Are you sure you want to delete the attendance sheet for ${sheetTitle(entry)}? This cannot be undone.`

export const rollNumber = (regNo) => {
  const digits = String(regNo || '').replace(/\D/g, '')
  if (!digits) return ''
  return digits.slice(-3).padStart(3, '0')
}

export const byRegNo = (one, two) =>
  String(one.reg_no || '').localeCompare(String(two.reg_no || ''), 'en', { numeric: true })

export const sortedRoster = (marks) => [...(marks || [])].sort(byRegNo)

export const sectionLabel = (section) => (section === 'ALL' ? 'A and B' : `Section ${section}`)

export const PASS_MARK = 50

export const belowPass = (percent) => Number(percent) < PASS_MARK

export const percentOf = (present, held) => {
  const total = Number(held) || 0
  if (total <= 0) return 0
  return Math.round((Number(present) || 0) * 100 / total)
}

export const classKey = (entry) => `${entry.course_id}|${entry.batch}|${entry.section}`

const joined = (...parts) =>
  parts
    .map((part) => (part === null || part === undefined ? '' : String(part).trim()))
    .filter(Boolean)
    .join(' · ')

export const markMeta = (mark, clock = (value) => value) =>
  joined(
    mark?.marked_at ? clock(mark.marked_at) : '',
    mark?.distance_m ? `${mark.distance_m} m` : '',
    mark?.accuracy_m ? `±${mark.accuracy_m} m` : '',
  )

export const presentOfRoster = (session) =>
  `${Number(session?.present) || 0}/${Number(session?.roster) || 0}`

export const CLOSED_TEXT = 'Attendance closed. Everyone missing is marked absent.'

export const EMPTY_CLOSED_TEXT = 'Nobody was marked, so this empty attendance was not kept.'

export const closedMessage = (result) =>
  result?.removed ? result.message || EMPTY_CLOSED_TEXT : CLOSED_TEXT

export const NO_LOCATION_FLAG = 'No location'

export const markFlags = (mark) =>
  mark?.no_location && mark?.status === 'present' ? [NO_LOCATION_FLAG] : []

export const startBody = (entry, minutes, method, codeOnly) => ({
  course_id: entry.course_id,
  batch: entry.batch,
  section: entry.section,
  schedule_item_id: entry.schedule_item_id || 0,
  minutes,
  method,
  code_only: method !== TAKE_ROLL_CALL && Boolean(codeOnly),
})

export const takeMethods = (app) => [
  { value: TAKE_BLUETOOTH, label: app ? 'Bluetooth' : 'Code + location' },
  { value: TAKE_ROLL_CALL, label: 'Roll call' },
]

export const CODE_ONLY_LABEL = 'Code only (no location check)'

export const codeOnlyMarkBody = (session, code) => ({
  session_id: session.id,
  code: String(code || '').trim(),
})

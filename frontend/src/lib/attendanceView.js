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

export const sectionLabel = (section) => (section === 'ALL' ? 'A and B' : `Section ${section}`)

export const classLabel = (entry) =>
  `${entry.course_code} · ${entry.batch} · ${sectionLabel(entry.section)}`

export const PASS_MARK = 50

export const belowPass = (percent) => Number(percent) < PASS_MARK

export const percentOf = (present, held) => {
  const total = Number(held) || 0
  if (total <= 0) return 0
  return Math.round((Number(present) || 0) * 100 / total)
}

export const classKey = (entry) => `${entry.course_id}|${entry.batch}|${entry.section}`

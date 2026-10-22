export const SECTIONS = ['A', 'B']

export const SEMESTERS = ['1-1', '1-2', '2-1', '2-2', '3-1', '3-2', '4-1', '4-2']

const YEAR_NAMES = ['1st year', '2nd year', '3rd year', '4th year']

const HALF_NAMES = ['1st semester', '2nd semester']

export const semesterLabel = (value) => {
  const [year, half] = String(value || '').split('-')
  const yearName = YEAR_NAMES[Number(year) - 1]
  const halfName = HALF_NAMES[Number(half) - 1]
  return yearName && halfName ? `${yearName} ${halfName}` : value || ''
}

export const semesterOptions = (running) => {
  if (!running || !SEMESTERS.includes(running)) return SEMESTERS
  return [running, ...SEMESTERS.filter((semester) => semester !== running)]
}

export const BLOOD_GROUPS = ['A+', 'A-', 'B+', 'B-', 'AB+', 'AB-', 'O+', 'O-']

export const ACCOUNT_STATUSES = ['active', 'pending', 'rejected', 'removed']

export const PEOPLE_STATUS_TABS = [
  { value: '', label: 'All' },
  { value: 'active', label: 'Active' },
  { value: 'pending', label: 'Pending' },
  { value: 'invited', label: 'Invited' },
  { value: 'rejected', label: 'Rejected' },
  { value: 'removed', label: 'Removed' },
]

export const RETENTION_STATUSES = ['removed', 'rejected']

export const canRestore = (status) => RETENTION_STATUSES.includes(status)

export const AUDIT_ACTION_LABELS = {
  remove: 'Removed',
  restore: 'Restored',
  permanent_delete: 'Deleted permanently',
}

export const MAX_NOTE = 500

export const NOTE_AUDIENCES = [
  { value: 'teacher', label: 'Teacher' },
  { value: 'students', label: 'Students' },
  { value: 'both', label: 'Both' },
]

export const NOTE_AUDIENCE_LABELS = {
  teacher: 'For the teacher',
  students: 'For the students',
  both: 'For the teacher and the students',
}

export const noteError = (note, audience) => {
  const text = String(note || '').trim()
  if (!text) return ''
  if (text.length > MAX_NOTE) return `Note must be at most ${MAX_NOTE} characters long.`
  if (!audience) return 'Note audience is required.'
  return ''
}

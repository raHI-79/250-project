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

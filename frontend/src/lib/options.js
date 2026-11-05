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

export const ACTIVITY_LABELS = {
  ...AUDIT_ACTION_LABELS,
  create: 'Created',
  update: 'Updated',
  delete: 'Deleted',
  publish: 'Published',
  sign_in: 'Signed in',
  sign_out: 'Signed out',
  sign_in_failed: 'Failed sign-in',
  invite_admin: 'Invited an admin',
  resend_invite: 'Resent an invite',
  cancel_invite: 'Cancelled an invite',
  suspend_admin: 'Suspended an admin',
  restore_admin: 'Restored an admin',
  remove_admin: 'Removed an admin',
  grant_office_access: 'Granted office access',
  revoke_office_access: 'Revoked office access',
  activate_admin: 'Accepted an invitation',
  student: 'Student',
  cr: 'Class representative',
  teacher: 'Teacher',
  admin: 'Office admin',
  super_admin: 'Control room',
}

export const activityLabel = (key) => ACTIVITY_LABELS[key] || key || '—'

export const ACTIVITY_ROLES = [
  { value: 'student', label: 'Students' },
  { value: 'cr', label: 'CRs' },
  { value: 'teacher', label: 'Teachers' },
  { value: 'admin', label: 'Office admins' },
  { value: 'super_admin', label: 'Control room' },
]

export const ACTIVITY_ACTIONS = [
  { value: 'sign_in', label: 'Signed in' },
  { value: 'sign_out', label: 'Signed out' },
  { value: 'sign_in_failed', label: 'Failed sign-in' },
  { value: 'create', label: 'Created' },
  { value: 'update', label: 'Updated' },
  { value: 'delete', label: 'Deleted' },
  { value: 'publish', label: 'Published' },
  { value: 'remove', label: 'Removed an account' },
  { value: 'restore', label: 'Restored an account' },
  { value: 'permanent_delete', label: 'Deleted an account' },
  { value: 'invite_admin', label: 'Invited an admin' },
  { value: 'grant_office_access', label: 'Granted office access' },
  { value: 'revoke_office_access', label: 'Revoked office access' },
]

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

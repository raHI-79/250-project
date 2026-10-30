export const MAX_MESSAGE_BODY = 4000

export const EDIT_WINDOW = 2 * 60 * 1000

export const THREAD_INTERVAL = 10000

export const LIST_INTERVAL = 20000

export const CONTACT_CHIPS = [
  { value: '', label: 'All' },
  { value: 'teacher', label: 'Teachers' },
  { value: 'cr', label: 'CRs' },
  { value: 'student', label: 'Students' },
  { value: 'admin', label: 'Office' },
]

export const contactChips = (viewerRole) =>
  viewerRole === 'student' ? CONTACT_CHIPS.filter((chip) => chip.value !== 'student') : CONTACT_CHIPS

export const matchesRole = (conversation, role) => {
  if (!role) return true
  if (conversation.kind === 'group') return false
  return (conversation.other_role || '') === role
}

export const shortRoleLabel = (role) => {
  if (role === 'admin') return 'Admin'
  if (role === 'teacher') return 'Teacher'
  if (role === 'cr') return 'CR'
  return 'Student'
}

export const contactDetail = (contact) => {
  if (contact.designation) return contact.designation
  if (contact.batch && contact.section) return `${contact.batch} · Section ${contact.section}`
  return contact.batch || ''
}

export const conversationDetail = (conversation) => {
  if (!conversation) return ''
  if (conversation.kind !== 'group') return 'Direct conversation'
  const count = conversation.member_count
  const role = conversation.member_role === 'admin' ? ' · you are a group admin' : ''
  if (count) return `${count} member${count === 1 ? '' : 's'}${role}`
  return `Group chat${role}`
}

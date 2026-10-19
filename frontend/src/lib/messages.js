export const MAX_MESSAGE_BODY = 4000

export const THREAD_INTERVAL = 10000

export const LIST_INTERVAL = 20000

const BATCH_SCOPE = { value: 'batch', label: 'A whole batch', batch: 'choose' }

const SECTION_SCOPE = { value: 'section', label: 'One section', batch: 'choose', section: true }

const GROUP_SCOPES = {
  teacher: [BATCH_SCOPE, SECTION_SCOPE],
  cr: [BATCH_SCOPE, SECTION_SCOPE],
  admin: [
    BATCH_SCOPE,
    SECTION_SCOPE,
    { value: 'all_students', label: 'Every student' },
    { value: 'all_teachers', label: 'Every teacher' },
  ],
  student: [{ value: 'batch', label: 'My batch', batch: 'own' }],
}

export const scopesForRole = (role) => GROUP_SCOPES[role] || GROUP_SCOPES.student

export const CONTACT_ROLES = [
  { value: '', label: 'Everybody' },
  { value: 'student', label: 'Students' },
  { value: 'cr', label: 'Class representatives' },
  { value: 'teacher', label: 'Teachers' },
  { value: 'admin', label: 'Administration' },
]

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

export const groupTitleOf = (scope, batch, section) => {
  if (scope === 'all_students') return 'All students'
  if (scope === 'all_teachers') return 'All teachers'
  if (scope === 'section') return `${batch} · Section ${section}`
  return batch || ''
}

export const conversationDetail = (conversation) => {
  if (!conversation) return ''
  if (conversation.kind === 'group') return 'Group conversation'
  return 'Direct conversation'
}

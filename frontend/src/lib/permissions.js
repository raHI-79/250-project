export const PERMISSION_LABELS = {
  registrations: 'Approve registrations',
  students: 'Manage students and CRs',
  teachers: 'Invite and manage teachers',
  catalog: 'Courses and semesters',
  batches: 'Running batches',
  periods: 'Class periods and rooms',
  routines: 'Routines',
  notices: 'Notices',
  attendance: 'Attendance and phone changes',
  exports: 'Exports',
}

export const PERMISSION_ORDER = [
  'registrations',
  'students',
  'teachers',
  'catalog',
  'batches',
  'periods',
  'routines',
  'notices',
  'attendance',
  'exports',
]

export const permissionLabel = (name) => PERMISSION_LABELS[name] || name

export const allowed = (permissions, name) => {
  if (!name) return true
  if (!permissions) return true
  return permissions.includes(name)
}

export const filterNav = (groups, permissions) =>
  groups
    .map((group) => ({
      ...group,
      items: group.items.filter((item) => allowed(permissions, item.permission)),
    }))
    .filter((group) => group.items.length > 0)

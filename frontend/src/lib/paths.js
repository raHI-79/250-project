const prefix = ''

const join = (...parts) => {
  const path = parts.filter(Boolean).join('/').replace(/\/+/g, '/')
  return path.startsWith('/') ? path : `/${path}`
}

const build = (...parts) => join(prefix, ...parts)

export const routePattern = (...parts) => join(prefix, ...parts)

export const paths = {
  root: () => build(),
  login: () => build('login'),
  register: () => build('register'),
  forgotPassword: () => build('forgot-password'),
  adminLogin: () => build('portal-admin'),
  student: {
    home: () => build('student'),
    upcomingRoutine: () => build('student', 'upcoming-routine'),
    dropCourseRoutine: () => build('student', 'drop-course-routine'),
    exams: () => build('student', 'exams'),
    fixedRoutine: () => build('student', 'fixed-routine'),
    crNotices: () => build('student', 'cr-notices'),
    adminNotices: () => build('student', 'admin-notices'),
    materials: () => build('student', 'materials'),
    students: () => build('student', 'students'),
    polls: () => build('student', 'polls'),
    teachers: () => build('student', 'teachers'),
    messages: () => build('student', 'messages'),
  },
  cr: {
    home: () => build('cr'),
    fixedRoutine: () => build('cr', 'fixed-routine'),
    createClass: () => build('cr', 'create-class'),
    extraClass: () => build('cr', 'extra-class'),
    requests: () => build('cr', 'requests'),
    sendMessage: () => build('cr', 'send-message'),
    teacherMessages: () => build('cr', 'teacher-messages'),
    notices: () => build('cr', 'notices'),
    materials: () => build('cr', 'materials'),
    polls: () => build('cr', 'polls'),
    teachers: () => build('cr', 'teachers'),
  },
  teacher: {
    home: () => build('teacher'),
    schedule: () => build('teacher', 'schedule'),
    requests: () => build('teacher', 'requests'),
    cancelClass: () => build('teacher', 'cancel-class'),
    crMessages: () => build('teacher', 'cr-messages'),
    sendMessage: () => build('teacher', 'send-message'),
  },
  admin: {
    home: () => build('portal-admin', 'overview'),
    overview: () => build('portal-admin', 'overview'),
    registrations: () => build('portal-admin', 'registrations'),
    students: () => build('portal-admin', 'students'),
    crs: () => build('portal-admin', 'crs'),
    teachers: () => build('portal-admin', 'teachers'),
    courses: () => build('portal-admin', 'courses'),
    routines: () => build('portal-admin', 'routines'),
    notices: () => build('portal-admin', 'notices'),
    materials: () => build('portal-admin', 'materials'),
  },
}

export const homeFor = (role) => {
  if (role === 'admin') return paths.admin.home()
  if (role === 'teacher') return paths.teacher.home()
  if (role === 'cr') return paths.cr.home()
  if (role === 'student') return paths.student.home()
  return paths.login()
}

export default paths

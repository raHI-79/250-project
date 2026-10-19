const prefix = ''

const join = (...parts) => {
  const path = parts.filter(Boolean).join('/').replace(/\/+/g, '/')
  return path.startsWith('/') ? path : `/${path}`
}

const build = (...parts) => join(prefix, ...parts)

export const routePattern = (...parts) => join(prefix, ...parts)

export const paths = {
  root: () => build(),
  profile: () => build('profile'),
  person: (userId) => build('people', String(userId)),
  personPattern: () => routePattern('people', ':userId'),
  studentLogin: () => build('student', 'login'),
  studentRegister: () => build('student', 'register'),
  studentVerifyEmail: (regNo) =>
    `${build('student', 'register')}?verify=${encodeURIComponent(regNo)}`,
  studentForgotPassword: () => build('student', 'forgot-password'),
  teacherLogin: () => build('teacher', 'login'),
  teacherForgotPassword: () => build('teacher', 'forgot-password'),
  teacherActivate: () => build('teacher', 'activate'),
  adminLogin: () => build('portal-admin'),
  adminForgotPassword: () => build('portal-admin', 'forgot-password'),
  student: {
    home: () => build('student'),
    today: () => build('student', 'today'),
    upcomingRoutine: () => build('student', 'upcoming-routine'),
    exams: () => build('student', 'exams'),
    fixedRoutine: () => build('student', 'fixed-routine'),
    notices: () => build('student', 'notices'),
    materials: () => build('student', 'materials'),
    students: () => build('student', 'students'),
    polls: () => build('student', 'polls'),
    teachers: () => build('student', 'teachers'),
    messages: () => build('student', 'messages'),
  },
  cr: {
    home: () => build('cr'),
    createClass: () => build('cr', 'create-class'),
    extraClass: () => build('cr', 'extra-class'),
    postNotices: () => build('cr', 'post-notices'),
    driveLink: () => build('cr', 'drive-link'),
    polls: () => build('cr', 'polls'),
  },
  teacher: {
    home: () => build('teacher'),
    schedule: () => build('teacher', 'schedule'),
    requests: () => build('teacher', 'requests'),
    cancelClass: () => build('teacher', 'cancel-class'),
    notices: () => build('teacher', 'notices'),
    messages: () => build('teacher', 'messages'),
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
    messages: () => build('portal-admin', 'messages'),
  },
}

export const PORTAL_STUDENT = 'student'

export const PORTAL_TEACHER = 'teacher'

export const PORTAL_ADMIN = 'admin'

export const portalFor = (role) => {
  if (role === 'admin') return PORTAL_ADMIN
  if (role === 'teacher') return PORTAL_TEACHER
  return PORTAL_STUDENT
}

export const loginFor = (role) => {
  const portal = portalFor(role)
  if (portal === PORTAL_ADMIN) return paths.adminLogin()
  if (portal === PORTAL_TEACHER) return paths.teacherLogin()
  return paths.studentLogin()
}

export const loginForPortal = (portal) => {
  if (portal === PORTAL_ADMIN) return paths.adminLogin()
  if (portal === PORTAL_TEACHER) return paths.teacherLogin()
  return paths.studentLogin()
}

export const forgotPasswordForPortal = (portal) => {
  if (portal === PORTAL_ADMIN) return paths.adminForgotPassword()
  if (portal === PORTAL_TEACHER) return paths.teacherForgotPassword()
  return paths.studentForgotPassword()
}

export const homeFor = (role) => {
  if (role === 'admin') return paths.admin.home()
  if (role === 'teacher') return paths.teacher.home()
  if (role === 'cr' || role === 'student') return paths.student.today()
  return paths.studentLogin()
}

export default paths

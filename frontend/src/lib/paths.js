const prefix = ''

const join = (...parts) => {
  const path = parts.filter(Boolean).join('/').replace(/\/+/g, '/')
  return path.startsWith('/') ? path : `/${path}`
}

const build = (...parts) => join(prefix, ...parts)

export const routePattern = (...parts) => join(prefix, ...parts)

const query = (path, params) => {
  const search = new URLSearchParams()
  Object.entries(params || {}).forEach(([key, value]) => {
    if (value === undefined || value === null || value === '') return
    search.set(key, String(value))
  })
  const text = search.toString()
  return text ? `${path}?${text}` : path
}

export const paths = {
  root: () => build(),
  downloadAndroid: () => '/download/android',
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
  adminLogin: () => build('office-portal'),
  adminForgotPassword: () => build('office-portal', 'forgot-password'),
  student: {
    home: () => build('student'),
    today: (params) => query(build('student', 'today'), params),
    upcomingRoutine: (params) => query(build('student', 'upcoming-routine'), params),
    exams: (params) => query(build('student', 'exams'), params),
    fixedRoutine: () => build('student', 'fixed-routine'),
    courses: () => build('student', 'courses'),
    notices: (params) => query(build('student', 'notices'), params),
    materials: () => build('student', 'materials'),
    students: () => build('student', 'students'),
    polls: (params) => query(build('student', 'polls'), params),
    teachers: () => build('student', 'teachers'),
    messages: (params) => query(build('student', 'messages'), params),
  },
  cr: {
    home: () => build('cr'),
    schedule: (params) => query(build('cr', 'create-class'), params),
    postNotices: () => build('cr', 'post-notices'),
    driveLink: () => build('cr', 'drive-link'),
    polls: (params) => query(build('cr', 'polls'), params),
  },
  teacher: {
    home: () => build('teacher'),
    fixedRoutine: () => build('teacher', 'fixed-routine'),
    upcoming: (params) => query(build('teacher', 'upcoming-routine'), params),
    createExam: (params) => query(build('teacher', 'create-exam'), params),
    notices: (params) => query(build('teacher', 'notices'), params),
    messages: (params) => query(build('teacher', 'messages'), params),
  },
  admin: {
    home: () => build('office-portal', 'overview'),
    overview: () => build('office-portal', 'overview'),
    registrations: () => build('office-portal', 'registrations'),
    students: () => build('office-portal', 'students'),
    crs: () => build('office-portal', 'crs'),
    teachers: () => build('office-portal', 'teachers'),
    courses: () => build('office-portal', 'courses'),
    periods: () => build('office-portal', 'class-periods'),
    routines: () => build('office-portal', 'routines'),
    notices: (params) => query(build('office-portal', 'notices'), params),
    messages: (params) => query(build('office-portal', 'messages'), params),
    activity: () => build('office-portal', 'activity'),
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

export const messagesFor = (role, params) => {
  if (role === 'admin') return paths.admin.messages(params)
  if (role === 'teacher') return paths.teacher.messages(params)
  return paths.student.messages(params)
}

export const noticesFor = (role, params) => {
  if (role === 'admin') return paths.admin.notices(params)
  if (role === 'teacher') return paths.teacher.notices(params)
  return paths.student.notices(params)
}

export const pollsFor = (role, params) => {
  if (role === 'cr') return paths.cr.polls(params)
  return paths.student.polls(params)
}

export const CR_TAB_CREATE = 'create'

export const CR_TAB_ROOMS = 'rooms'

export const CR_TAB_UPCOMING = 'upcoming'

export const CR_TAB_FINAL = 'final'

export const TEACHER_TAB_NEXT_DAY = 'next-day'

export const TEACHER_TAB_FINAL = 'final'

export const notificationTarget = (link, role) => {
  const kind = link?.kind || ''
  const ref = link?.ref || ''
  const date = link?.date || ''
  if (kind === 'teacher_next_day') {
    return paths.teacher.upcoming({ tab: TEACHER_TAB_NEXT_DAY, item: ref })
  }
  if (kind === 'teacher_final') {
    return paths.teacher.upcoming({ tab: TEACHER_TAB_FINAL, item: ref })
  }
  if (kind === 'cr_upcoming') return paths.cr.schedule({ tab: CR_TAB_UPCOMING, item: ref })
  if (kind === 'cr_final') return paths.cr.schedule({ tab: CR_TAB_FINAL, item: ref })
  if (kind === 'routine') return paths.student.upcomingRoutine({ date })
  if (kind === 'exam') {
    if (role === 'teacher') return paths.teacher.createExam({ highlight: ref })
    return paths.student.exams({ highlight: ref })
  }
  if (kind === 'notice') return noticesFor(role, { highlight: ref })
  if (kind === 'message') return messagesFor(role, { c: ref })
  if (kind === 'poll') return pollsFor(role, { highlight: ref })
  if (kind === 'cr_role') return paths.profile()
  return homeFor(role)
}

export const homeFor = (role) => {
  if (role === 'admin') return paths.admin.home()
  if (role === 'teacher') return paths.teacher.home()
  if (role === 'cr' || role === 'student') return paths.student.today()
  return paths.studentLogin()
}

export default paths

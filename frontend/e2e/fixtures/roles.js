
const DEMO_PASSWORD = process.env.DEMO_PASSWORD || 'sust-250-project'
const OFFICE_ID = process.env.E2E_OFFICE_ID || 'office'
const OFFICE_PASSWORD = process.env.E2E_OFFICE_PASSWORD || DEMO_PASSWORD
const CONTROL_ID = process.env.E2E_CONTROL_ID || 'chief'
const CONTROL_PASSWORD = process.env.E2E_CONTROL_PASSWORD || OFFICE_PASSWORD

export const STUDENT_PAGES = [
  { path: '/student/today' },
  { path: '/student/upcoming-routine' },
  { path: '/student/exams' },
  { path: '/student/fixed-routine' },
  { path: '/student/courses' },
  { path: '/student/notices' },
  { path: '/student/materials' },
  { path: '/student/students' },
  { path: '/student/polls' },
  { path: '/student/attendance' },
  { path: '/student/my-attendance' },
  { path: '/student/teachers' },
  { path: '/student/messages' },
  { path: '/profile' },
]

export const CR_PAGES = [
  ...STUDENT_PAGES,
  { path: '/cr/create-class' },
  { path: '/cr/create-class?tab=final' },
  { path: '/cr/create-class?tab=create' },
  { path: '/cr/create-class?tab=rooms' },
  { path: '/cr/post-notices' },
  { path: '/cr/drive-link' },
  { path: '/cr/polls' },
]

export const TEACHER_PAGES = [
  { path: '/teacher/upcoming-routine' },
  { path: '/teacher/upcoming-routine?tab=final' },
  { path: '/teacher/fixed-routine' },
  { path: '/teacher/create-exam' },
  { path: '/teacher/notices' },
  { path: '/teacher/attendance' },
  { path: '/teacher/phone-requests' },
  { path: '/teacher/messages' },
  { path: '/profile' },
]

export const OFFICE_PAGES = [
  { path: '/office-portal/overview' },
  { path: '/office-portal/registrations' },
  { path: '/office-portal/students' },
  { path: '/office-portal/crs' },
  { path: '/office-portal/teachers' },
  { path: '/office-portal/courses' },
  { path: '/office-portal/class-periods' },
  { path: '/office-portal/routines' },
  { path: '/office-portal/notices' },
  { path: '/office-portal/messages' },
  { path: '/office-portal/activity' },
  { path: '/office-portal/attendance' },
  { path: '/office-portal/phone-changes' },
  { path: '/office-portal/account' },
  { path: '/office-portal/security' },
  { path: '/office-portal/approvals' },
]

export const CONTROL_PAGES = [
  { path: '/control-room' },
  { path: '/control-room/admins' },
  { path: '/control-room/users' },
  { path: '/control-room/activity' },
  { path: '/control-room/settings' },
  { path: '/control-room/settings/account' },
  { path: '/control-room/settings/security' },
  { path: '/control-room/settings/monitor' },
  { path: '/control-room/settings/app' },
  { path: '/control-room/approvals' },
  { path: '/control-room/data' },
]

export const ROLES = {
  student: {
    label: 'student',
    loginPath: '/student/login',
    identifier: process.env.E2E_STUDENT_ID || 's23a2',
    password: DEMO_PASSWORD,
    landing: '/student/today',
    pages: STUDENT_PAGES,
  },
  cr: {
    label: 'class representative',
    loginPath: '/student/login',
    identifier: process.env.E2E_CR_ID || 's23a1',
    password: DEMO_PASSWORD,
    landing: '/student/today',
    pages: CR_PAGES,
  },
  teacher: {
    label: 'teacher',
    loginPath: '/teacher/login',
    identifier: process.env.E2E_TEACHER_ID || 'mrahman',
    password: DEMO_PASSWORD,
    landing: '/teacher/upcoming-routine',
    pages: TEACHER_PAGES,
  },
  office: {
    label: 'office admin',
    loginPath: '/office-portal',
    identifier: OFFICE_ID,
    password: OFFICE_PASSWORD,
    landing: '/office-portal/overview',
    pages: OFFICE_PAGES,
  },
  control: {
    label: 'super admin',
    loginPath: '/control-room',
    identifier: CONTROL_ID,
    password: CONTROL_PASSWORD,
    landing: '/control-room',
    pages: CONTROL_PAGES,
  },
}

export const DIALOG_PAGES = {
  student: ['/student/polls', '/profile'],
  cr: ['/cr/create-class', '/cr/create-class?tab=final', '/cr/post-notices', '/cr/polls'],
  teacher: ['/teacher/upcoming-routine', '/teacher/create-exam', '/teacher/notices'],
  office: [
    '/office-portal/teachers',
    '/office-portal/courses',
    '/office-portal/notices',
    '/office-portal/class-periods',
    '/office-portal/routines',
  ],
  control: ['/control-room/admins', '/control-room/settings'],
}

export const DIALOG_OPENERS =
  /^(Invite|Add|New|Create|Post|Edit|Remove|Suspend|Delete|Cancel|Reschedule|Set |Confirm|Publish|Preview|Request|Change|Mark|Open|Close|Export|Block)/i

const pathsOf = (pages) => pages.map((entry) => entry.path)

const STUDENT_TABS = ['/student/attendance?tab=mine', '/student/attendance?tab=phone']

export const PHONE_PAGES = {
  student: [...pathsOf(STUDENT_PAGES), ...STUDENT_TABS],
  cr: [...pathsOf(CR_PAGES), ...STUDENT_TABS],
  teacher: [
    ...pathsOf(TEACHER_PAGES),
    '/teacher/attendance?tab=history',
    '/teacher/attendance?tab=classes',
    '/teacher/attendance?tab=phones',
  ],
  office: [
    ...pathsOf(OFFICE_PAGES),
    '/office-portal/attendance?tab=sheets',
    '/office-portal/attendance?tab=phones',
  ],
  control: pathsOf(CONTROL_PAGES),
}

export const DATE_PAGES = {
  student: ['/student/upcoming-routine', '/student/exams'],
  cr: [
    '/student/upcoming-routine',
    '/student/exams',
    '/cr/create-class',
    '/cr/create-class?tab=final',
    '/cr/create-class?tab=create',
  ],
  teacher: ['/teacher/upcoming-routine', '/teacher/upcoming-routine?tab=final', '/teacher/create-exam'],
  office: ['/office-portal/activity', '/office-portal/routines', '/office-portal/teachers'],
  control: ['/control-room', '/control-room/activity', '/control-room/settings'],
}

export const DATE_PANELS = ['/student/register', '/student/forgot-password', '/teacher/activate']

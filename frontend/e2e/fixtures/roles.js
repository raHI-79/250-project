// The five roles the suite signs in as, with the pages each one can reach and the main
// dialogs worth opening. The identifiers come from the demo seed, which the suite expects
// to be loaded with `go run ./cmd/seed -demo` against TEST_DATABASE_URL.
//
// A page is named by its address. The visit helper decides when a page has loaded, from
// the content it shows rather than from a heading, because the shell moves the title into
// a bar on a phone.

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

// Each role signs in through the portal it belongs to. A class representative is a
// student the office promoted, so it uses the student sign-in.
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

// DIALOG_PAGES are the pages of each role that carry an action worth a dialog. The page
// walk already visits every page; this shorter list keeps the dialog pass to the places
// something opens.
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

// DIALOG_OPENERS matches the controls that open a dialog. The suite presses them on every
// page of a role and measures whatever appears, so no table of dialogs can go stale.
export const DIALOG_OPENERS =
  /^(Invite|Add|New|Create|Post|Edit|Remove|Suspend|Delete|Cancel|Reschedule|Set |Confirm|Publish|Preview|Request|Change|Mark|Open|Close|Export|Block)/i

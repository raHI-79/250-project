import {
  BookOpen,
  CalendarClock,
  CalendarDays,
  CalendarPlus,
  CalendarRange,
  CalendarX,
  ClipboardCheck,
  ClipboardList,
  FolderOpen,
  GaugeCircle,
  GraduationCap,
  Link2,
  Mail,
  Megaphone,
  Send,
  Sunrise,
  UserCircle,
  Users,
  UserCheck,
  UserCog,
  Vote,
} from 'lucide-react'
import { paths } from './paths'

export const MESSAGES_BADGE = 'messages'

const overview = [
  { label: 'Today', to: paths.student.today, icon: Sunrise },
  { label: 'Upcoming routine', to: paths.student.upcomingRoutine, icon: CalendarClock },
  { label: 'Exams', to: paths.student.exams, icon: ClipboardList },
  { label: 'Weekly routine', to: paths.student.fixedRoutine, icon: BookOpen },
]

const communication = [
  { label: 'Notices', to: paths.student.notices, icon: Megaphone },
  { label: 'Messages', to: paths.student.messages, icon: Mail, badge: MESSAGES_BADGE },
]

const community = [
  { label: 'Polls', to: paths.student.polls, icon: Vote },
  { label: 'Materials', to: paths.student.materials, icon: FolderOpen },
  { label: 'Directory', to: paths.student.students, icon: Users },
  { label: 'Teachers', to: paths.student.teachers, icon: GraduationCap },
]

const account = [{ label: 'My profile', to: paths.profile, icon: UserCircle }]

export const studentNav = [
  { label: 'Overview', items: overview },
  { label: 'Communication', items: communication },
  { label: 'Community', items: community },
  { label: 'Account', items: account },
]

export const crNav = [
  { label: 'Student', items: [...overview, ...communication, ...community] },
  {
    label: 'Class representative',
    items: [
      { label: 'Create class / exam', to: paths.cr.createClass, icon: CalendarPlus },
      { label: 'Extra class', to: paths.cr.extraClass, icon: CalendarRange },
      { label: 'Post notice', to: paths.cr.postNotices, icon: Send },
      { label: 'Batch Drive link', to: paths.cr.driveLink, icon: Link2 },
      { label: 'Polls', to: paths.cr.polls, icon: Vote },
    ],
  },
  { label: 'Account', items: account },
]

export const teacherNav = [
  {
    items: [
      { label: 'Daily Schedule', to: paths.teacher.schedule, icon: CalendarDays },
      { label: 'Approval Requests', to: paths.teacher.requests, icon: ClipboardCheck },
      { label: 'Cancel Class', to: paths.teacher.cancelClass, icon: CalendarX },
      { label: 'Notices', to: paths.teacher.notices, icon: Megaphone },
      { label: 'Messages', to: paths.teacher.messages, icon: Mail, badge: MESSAGES_BADGE },
    ],
  },
]

export const adminNav = [
  {
    items: [
      { label: 'Overview', to: paths.admin.overview, icon: GaugeCircle },
      { label: 'Registrations', to: paths.admin.registrations, icon: UserCheck },
      { label: 'Students', to: paths.admin.students, icon: Users },
      { label: 'CRs', to: paths.admin.crs, icon: UserCog },
      { label: 'Teachers', to: paths.admin.teachers, icon: GraduationCap },
      { label: 'Courses', to: paths.admin.courses, icon: BookOpen },
      { label: 'Routines', to: paths.admin.routines, icon: CalendarDays },
      { label: 'Notices', to: paths.admin.notices, icon: Megaphone },
      { label: 'Messages', to: paths.admin.messages, icon: Mail, badge: MESSAGES_BADGE },
    ],
  },
]

export const navForRole = (role) => {
  if (role === 'admin') return adminNav
  if (role === 'teacher') return teacherNav
  if (role === 'cr') return crNav
  return studentNav
}

export const roleLabel = (role) => {
  if (role === 'admin') return 'Administration'
  if (role === 'teacher') return 'Teacher'
  if (role === 'cr') return 'Class Representative'
  return 'Student'
}

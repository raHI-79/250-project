import {
  BookOpen,
  CalendarClock,
  CalendarDays,
  CalendarPlus,
  CheckCircle2,
  ClipboardList,
  Clock,
  FolderOpen,
  GaugeCircle,
  GraduationCap,
  Library,
  Link2,
  Mail,
  Megaphone,
  PieChart,
  Radio,
  ScrollText,
  Smartphone,
  Send,
  Sunrise,
  UserCircle,
  Users,
  UserCheck,
  UserCog,
  Vote,
} from 'lucide-react'
import { paths } from './paths'
import {
  BADGE_COURSES,
  BADGE_EXAMS,
  BADGE_MATERIALS,
  BADGE_NOTICES,
  BADGE_POLLS,
  BADGE_ROUTINE,
} from './badges'

export const MESSAGES_BADGE = 'messages'

const overview = [
  { label: 'Today', to: paths.student.today, icon: Sunrise },
  {
    label: 'Upcoming routine',
    to: paths.student.upcomingRoutine,
    icon: CalendarClock,
    badge: BADGE_ROUTINE,
  },
  { label: 'Exams', to: paths.student.exams, icon: ClipboardList, badge: BADGE_EXAMS },
  { label: 'Attendance', to: paths.student.attendance, icon: CheckCircle2 },
  { label: 'My attendance', to: paths.student.myAttendance, icon: PieChart },
  { label: 'Weekly routine', to: paths.student.fixedRoutine, icon: BookOpen },
  { label: 'Courses', to: paths.student.courses, icon: Library, badge: BADGE_COURSES },
]

const communication = [
  { label: 'Notices', to: paths.student.notices, icon: Megaphone, badge: BADGE_NOTICES },
  { label: 'Messages', to: paths.student.messages, icon: Mail, badge: MESSAGES_BADGE },
]

const community = [
  { label: 'Polls', to: paths.student.polls, icon: Vote, badge: BADGE_POLLS },
  { label: 'Materials', to: paths.student.materials, icon: FolderOpen, badge: BADGE_MATERIALS },
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
      { label: 'Create Class & Exam', to: paths.cr.schedule, icon: CalendarPlus },
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
      { label: 'Fixed weekly routine', to: paths.teacher.fixedRoutine, icon: CalendarDays },
      { label: 'Upcoming routine', to: paths.teacher.upcoming, icon: CalendarClock },
      { label: 'Create exam', to: paths.teacher.createExam, icon: ClipboardList },
      { label: 'Attendance', to: paths.teacher.attendance, icon: Radio },
      { label: 'Phone requests', to: paths.teacher.phoneRequests, icon: Smartphone },
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
      { label: 'Class periods', to: paths.admin.periods, icon: Clock },
      { label: 'Routines', to: paths.admin.routines, icon: CalendarDays },
      { label: 'Attendance', to: paths.admin.attendance, icon: PieChart },
      { label: 'Phone changes', to: paths.admin.phoneChanges, icon: Smartphone },
      { label: 'Notices', to: paths.admin.notices, icon: Megaphone },
      { label: 'Messages', to: paths.admin.messages, icon: Mail, badge: MESSAGES_BADGE },
      { label: 'Activity log', to: paths.admin.activity, icon: ScrollText },
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

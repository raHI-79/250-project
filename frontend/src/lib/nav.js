import {
  BookOpen,
  CalendarClock,
  CalendarDays,
  CalendarPlus,
  Database,
  CheckCircle2,
  ClipboardList,
  Clock,
  Layers,
  FolderOpen,
  GaugeCircle,
  Globe,
  GraduationCap,
  Library,
  Link2,
  Mail,
  Megaphone,
  PieChart,
  Radio,
  ScrollText,
  Send,
  ShieldAlert,
  ShieldCheck,
  Settings,
  Smartphone,
  Sunrise,
  UserCircle,
  Users,
  UserCheck,
  UserCog,
  Vote,
} from 'lucide-react'
import { paths } from './paths.js'
import {
  BADGE_COURSES,
  BADGE_EXAMS,
  BADGE_MATERIALS,
  BADGE_NOTICES,
  BADGE_POLLS,
  BADGE_ROUTINE,
} from './badgekeys.js'

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
      { label: 'Notices', to: paths.teacher.notices, icon: Megaphone },
      { label: 'Messages', to: paths.teacher.messages, icon: Mail, badge: MESSAGES_BADGE },
    ],
  },
]

export const adminNav = [
  {
    items: [
      { label: 'Overview', to: paths.admin.overview, icon: GaugeCircle },
      {
        label: 'Registrations',
        to: paths.admin.registrations,
        icon: UserCheck,
        permission: 'registrations',
      },
      { label: 'Students', to: paths.admin.students, icon: Users, permission: 'students' },
      { label: 'CRs', to: paths.admin.crs, icon: UserCog, permission: 'students' },
      {
        label: 'Teachers',
        to: paths.admin.teachers,
        icon: GraduationCap,
        permission: 'teachers',
      },
      { label: 'Courses', to: paths.admin.courses, icon: BookOpen, permission: 'catalog' },
      { label: 'Running batches', to: paths.admin.batches, icon: Layers, permission: 'batches' },
      { label: 'Class periods', to: paths.admin.periods, icon: Clock, permission: 'periods' },
      {
        label: 'Routines',
        to: paths.admin.routines,
        icon: CalendarDays,
        permission: 'routines',
      },
      {
        label: 'Attendance',
        to: paths.admin.attendance,
        icon: PieChart,
        permission: 'attendance',
      },
      { label: 'Notices', to: paths.admin.notices, icon: Megaphone, permission: 'notices' },
      { label: 'Messages', to: paths.admin.messages, icon: Mail, badge: MESSAGES_BADGE },
      { label: 'Activity log', to: paths.admin.activity, icon: ScrollText },
      { label: 'Approvals', to: paths.admin.approvals, icon: UserCheck },
      { label: 'My account', to: paths.admin.account, icon: ShieldCheck },
    ],
  },
]

export const controlNav = [
  {
    items: [
      { label: 'Dashboard', to: paths.control.home, icon: GaugeCircle },
      { label: 'Office admins', to: paths.control.admins, icon: ShieldCheck },
      { label: 'Users', to: paths.control.users, icon: Users },
      { label: 'Activity log', to: paths.control.activity, icon: ScrollText },
      { label: 'Approvals', to: paths.control.approvals, icon: UserCheck },
      { label: 'Site settings', to: paths.control.settings, icon: Settings },
      { label: 'Website', to: paths.control.website, icon: Globe },
      { label: 'Security monitor', to: paths.control.watch, icon: ShieldAlert },
      { label: 'Android app', to: paths.control.release, icon: Smartphone },
      { label: 'Data', to: paths.control.data, icon: Database },
      { label: 'My account', to: paths.control.account, icon: ShieldCheck },
    ],
  },
  {
    label: 'Office pages (read-only)',
    items: [
      { label: 'Overview', to: paths.admin.overview, icon: PieChart },
      { label: 'Registrations', to: paths.admin.registrations, icon: UserCheck },
      { label: 'Students', to: paths.admin.students, icon: Users },
      { label: 'CRs', to: paths.admin.crs, icon: UserCog },
      { label: 'Teachers', to: paths.admin.teachers, icon: GraduationCap },
      { label: 'Courses', to: paths.admin.courses, icon: BookOpen },
      { label: 'Class periods', to: paths.admin.periods, icon: Clock },
      { label: 'Routines', to: paths.admin.routines, icon: CalendarDays },
      { label: 'Attendance', to: paths.admin.attendance, icon: CheckCircle2 },
      { label: 'Notices', to: paths.admin.notices, icon: Megaphone },
    ],
  },
]

export const activeNavPath = (groups, pathname) => {
  const here = String(pathname || '').replace(/\/+$/, '') || '/'
  let best = ''
  groups.forEach((group) => {
    group.items.forEach((item) => {
      const target = String(item.to() || '').split('?')[0].replace(/\/+$/, '') || '/'
      const exact = target === here
      const inside = target !== '/' && here.startsWith(target + '/')
      if (!exact && !inside) return
      if (target.length > best.length) best = target
    })
  })
  return best
}

export const isNavActive = (item, activePath) =>
  String(item.to() || '').split('?')[0].replace(/\/+$/, '') === activePath

export const navForRole = (role) => {
  if (role === 'super_admin') return controlNav
  if (role === 'admin') return adminNav
  if (role === 'teacher') return teacherNav
  if (role === 'cr') return crNav
  return studentNav
}

export const roleLabel = (role) => {
  if (role === 'super_admin') return 'Control room'
  if (role === 'admin') return 'Administration'
  if (role === 'teacher') return 'Teacher'
  if (role === 'cr') return 'Class Representative'
  return 'Student'
}

const NOTIFIED_ROLES = ['student', 'cr', 'teacher']

export const receivesNotifications = (role) => NOTIFIED_ROLES.includes(role)

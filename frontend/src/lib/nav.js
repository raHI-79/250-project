import {
  BellRing,
  BookOpen,
  CalendarClock,
  CalendarDays,
  CalendarPlus,
  CalendarX,
  ClipboardCheck,
  ClipboardList,
  FolderOpen,
  GaugeCircle,
  GraduationCap,
  Inbox,
  Link2,
  Mail,
  MessageSquare,
  Megaphone,
  Send,
  Users,
  UserCheck,
  UserCog,
  Vote,
} from 'lucide-react'
import { paths } from './paths'

export const studentNav = [
  { label: 'Upcoming Routine', to: paths.student.upcomingRoutine, icon: CalendarClock },
  { label: 'Drop Course Routine', to: paths.student.dropCourseRoutine, icon: CalendarDays },
  { label: 'Exams', to: paths.student.exams, icon: ClipboardList },
  { label: 'Fixed Routine', to: paths.student.fixedRoutine, icon: BookOpen },
  { label: 'CR Notices', to: paths.student.crNotices, icon: Megaphone },
  { label: 'Admin Notices', to: paths.student.adminNotices, icon: BellRing },
  { label: 'Materials', to: paths.student.materials, icon: FolderOpen },
  { label: 'Students', to: paths.student.students, icon: Users },
  { label: 'Polls', to: paths.student.polls, icon: Vote },
  { label: 'Teachers', to: paths.student.teachers, icon: GraduationCap },
  { label: 'Messages', to: paths.student.messages, icon: Mail },
]

export const crNav = [
  { label: 'Fixed Routine', to: paths.cr.fixedRoutine, icon: BookOpen },
  { label: 'Create Class/Exam', to: paths.cr.createClass, icon: CalendarPlus },
  { label: 'Extra Class', to: paths.cr.extraClass, icon: CalendarClock },
  { label: 'My Requests', to: paths.cr.requests, icon: ClipboardList },
  { label: 'Send Message', to: paths.cr.sendMessage, icon: Send },
  { label: 'Teacher Messages', to: paths.cr.teacherMessages, icon: Inbox },
  { label: 'Notices', to: paths.cr.notices, icon: Megaphone },
  { label: 'Materials', to: paths.cr.materials, icon: FolderOpen },
  { label: 'Polls', to: paths.cr.polls, icon: Vote },
  { label: 'Teachers', to: paths.cr.teachers, icon: GraduationCap },
]

export const teacherNav = [
  { label: 'Daily Schedule', to: paths.teacher.schedule, icon: CalendarDays },
  { label: 'Approval Requests', to: paths.teacher.requests, icon: ClipboardCheck },
  { label: 'Cancel Class', to: paths.teacher.cancelClass, icon: CalendarX },
  { label: 'CR Messages', to: paths.teacher.crMessages, icon: MessageSquare },
  { label: 'Send Message', to: paths.teacher.sendMessage, icon: Send },
]

export const adminNav = [
  { label: 'Overview', to: paths.admin.overview, icon: GaugeCircle },
  { label: 'Registrations', to: paths.admin.registrations, icon: UserCheck },
  { label: 'Students', to: paths.admin.students, icon: Users },
  { label: 'CRs', to: paths.admin.crs, icon: UserCog },
  { label: 'Teachers', to: paths.admin.teachers, icon: GraduationCap },
  { label: 'Courses', to: paths.admin.courses, icon: BookOpen },
  { label: 'Routines', to: paths.admin.routines, icon: CalendarDays },
  { label: 'Notices', to: paths.admin.notices, icon: Megaphone },
  { label: 'Materials', to: paths.admin.materials, icon: Link2 },
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

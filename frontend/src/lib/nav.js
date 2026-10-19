import {
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
  Link2,
  Mail,
  Megaphone,
  Send,
  Users,
  UserCheck,
  UserCog,
  Vote,
} from 'lucide-react'
import { paths } from './paths'

export const MESSAGES_BADGE = 'messages'

export const studentNav = [
  { label: 'Upcoming Routine', to: paths.student.upcomingRoutine, icon: CalendarClock },
  { label: 'Drop Course Routine', to: paths.student.dropCourseRoutine, icon: CalendarDays },
  { label: 'Exams', to: paths.student.exams, icon: ClipboardList },
  { label: 'Fixed Routine', to: paths.student.fixedRoutine, icon: BookOpen },
  { label: 'Notices', to: paths.student.notices, icon: Megaphone },
  { label: 'Materials', to: paths.student.materials, icon: FolderOpen },
  { label: 'Students', to: paths.student.students, icon: Users },
  { label: 'Polls', to: paths.student.polls, icon: Vote },
  { label: 'Teachers', to: paths.student.teachers, icon: GraduationCap },
  { label: 'Messages', to: paths.student.messages, icon: Mail, badge: MESSAGES_BADGE },
]

export const crNav = [
  { label: 'Fixed Routine', to: paths.cr.fixedRoutine, icon: BookOpen },
  { label: 'Create Class/Exam', to: paths.cr.createClass, icon: CalendarPlus },
  { label: 'Extra Class', to: paths.cr.extraClass, icon: CalendarClock },
  { label: 'My Requests', to: paths.cr.requests, icon: ClipboardList },
  { label: 'Notices', to: paths.cr.notices, icon: Megaphone },
  { label: 'Post a Notice', to: paths.cr.postNotices, icon: Send },
  { label: 'Materials', to: paths.cr.materials, icon: FolderOpen },
  { label: 'Polls', to: paths.cr.polls, icon: Vote },
  { label: 'Teachers', to: paths.cr.teachers, icon: GraduationCap },
  { label: 'Messages', to: paths.cr.messages, icon: Mail, badge: MESSAGES_BADGE },
]

export const teacherNav = [
  { label: 'Daily Schedule', to: paths.teacher.schedule, icon: CalendarDays },
  { label: 'Approval Requests', to: paths.teacher.requests, icon: ClipboardCheck },
  { label: 'Cancel Class', to: paths.teacher.cancelClass, icon: CalendarX },
  { label: 'Notices', to: paths.teacher.notices, icon: Megaphone },
  { label: 'Messages', to: paths.teacher.messages, icon: Mail, badge: MESSAGES_BADGE },
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
  { label: 'Messages', to: paths.admin.messages, icon: Mail, badge: MESSAGES_BADGE },
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

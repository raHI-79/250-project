import {
  CalendarDays,
  CalendarPlus,
  ClipboardList,
  House,
  MessageCircle,
  UserRound,
} from 'lucide-react'
import { MESSAGES_BADGE } from './nav'
import { BADGE_EXAMS, BADGE_ROUTINE } from './badges'
import { paths } from './paths'

const profileTab = { label: 'Profile', to: paths.profile, icon: UserRound }

const messagesTab = {
  label: 'Messages',
  to: paths.student.messages,
  icon: MessageCircle,
  badge: MESSAGES_BADGE,
}

export const studentTabs = [
  { label: 'Home', to: paths.student.today, icon: House },
  {
    label: 'Routine',
    to: paths.student.upcomingRoutine,
    icon: CalendarDays,
    badge: BADGE_ROUTINE,
  },
  { label: 'Exams', to: paths.student.exams, icon: ClipboardList, badge: BADGE_EXAMS },
  messagesTab,
  profileTab,
]

export const crTabs = [
  { label: 'Home', to: paths.student.today, icon: House },
  {
    label: 'Routine',
    to: paths.student.upcomingRoutine,
    icon: CalendarDays,
    badge: BADGE_ROUTINE,
  },
  { label: 'CR hub', to: paths.cr.schedule, icon: CalendarPlus },
  messagesTab,
  profileTab,
]

export const teacherTabs = [
  { label: 'Upcoming', to: paths.teacher.upcoming, icon: House },
  { label: 'Routine', to: paths.teacher.fixedRoutine, icon: CalendarDays },
  { label: 'Exams', to: paths.teacher.createExam, icon: ClipboardList },
  {
    label: 'Messages',
    to: paths.teacher.messages,
    icon: MessageCircle,
    badge: MESSAGES_BADGE,
  },
  profileTab,
]

export const tabsForRole = (role) => {
  if (role === 'teacher') return teacherTabs
  if (role === 'cr') return crTabs
  if (role === 'student') return studentTabs
  return []
}

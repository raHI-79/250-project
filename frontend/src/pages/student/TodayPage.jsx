import {
  FEATURE_ATTENDANCE,
  FEATURE_MESSAGES,
  featureOn,
  visibleItems,
} from '../../lib/features'
import { useFeatures } from '../../lib/featureState'
import {
  CalendarClock,
  ClipboardList,
  CheckCircle2,
  FolderOpen,
  Library,
  MessageCircle,
  Megaphone,
  Users,
  Vote,
} from 'lucide-react'
import CRPanel from '../../components/today/CRPanel'
import ExamStrip from '../../components/today/ExamStrip'
import HomeStats from '../../components/today/HomeStats'
import MessagePreview from '../../components/today/MessagePreview'
import NextUpCard from '../../components/today/NextUpCard'
import NoticeCarousel from '../../components/today/NoticeCarousel'
import QuickActions from '../../components/today/QuickActions'
import TodayClasses from '../../components/today/TodayClasses'
import TodayHero from '../../components/today/TodayHero'
import { Alert, EmptyState, PullToRefresh, SkeletonCard, SkeletonRows } from '../../components/ui'
import { useAuth } from '../../context/auth'
import { useNow, useResource } from '../../lib/hooks'
import {
  BADGE_COURSES,
  BADGE_EXAMS,
  BADGE_MATERIALS,
  BADGE_NOTICES,
  BADGE_POLLS,
  BADGE_ROUTINE,
  useSectionBadges,
} from '../../lib/badges'
import { paths } from '../../lib/paths'
import { creditPercent } from '../../lib/attendanceTiming'
import { STATUS_CANCELLED } from '../../lib/schedule'
import { today as todayDate } from '../../lib/time'

const MESSAGE_PREVIEW = 3

const minutesOf = (clock) => {
  const [hour, minute] = String(clock || '')
    .split(':')
    .map((part) => Number(part))
  if (!Number.isInteger(hour) || !Number.isInteger(minute)) return -1
  return hour * 60 + minute
}

function Section({ title, action, children }) {
  return (
    <section className="mt-7">
      <div className="mb-3 flex items-end justify-between gap-3">
        <h2 className="font-display text-section font-bold text-ink">{title}</h2>
        {action}
      </div>
      {children}
    </section>
  )
}

export default function TodayPage() {
  const { user, role } = useAuth()
  const badges = useSectionBadges()
  const now = useNow(60000)
  const summary = useResource('/api/students/today')
  const routine = useResource('/api/students/upcoming-routine')
  const features = useFeatures()
  const messagesOn = featureOn(features, FEATURE_MESSAGES)
  const chats = useResource(messagesOn ? '/api/messages/conversations' : '')
  const attendance = useResource(
    featureOn(features, FEATURE_ATTENDANCE) ? '/api/students/attendance/mine' : '',
  )
  const crItems = useResource(role === 'cr' ? '/api/cr/upcoming' : null, {
    date: todayDate(),
  })
  const crFinal = useResource(role === 'cr' ? '/api/cr/final' : null)
  const data = summary.data
  const groups = routine.data?.groups || []
  const items = groups.flatMap((group) => group.items)
  const sorted = [...items].sort((a, b) => minutesOf(a.start_time) - minutesOf(b.start_time))
  const minutesNow = now.getHours() * 60 + now.getMinutes()
  const nextUp = sorted.find(
    (item) => item.status !== STATUS_CANCELLED && minutesOf(item.end_time) >= minutesNow,
  )
  const notices = data?.latest_notices || []
  const exams = data?.next_exams || []
  const unreadChats = (chats.data?.items || [])
    .filter((conversation) => conversation.unread_count > 0)
    .slice(0, MESSAGE_PREVIEW)
  const meta = data?.week ? `Week ${data.week}` : ''
  const myCourses = attendance.data?.courses || []
  const held = myCourses.reduce((total, entry) => total + (Number(entry.held) || 0), 0)
  const attended = myCourses.reduce((total, entry) => total + (Number(entry.credit) || 0), 0)
  const overall = held > 0 ? creditPercent(attended, held) : null
  const actions = [
    {
      label: 'Routine',
      icon: CalendarClock,
      to: paths.student.upcomingRoutine(),
      tone: 'bg-tint-blue text-tint-blue-ink',
      count: badges[BADGE_ROUTINE],
    },
    {
      label: 'Exams',
      icon: ClipboardList,
      to: paths.student.exams(),
      tone: 'bg-tint-orange text-tint-orange-ink',
      count: badges[BADGE_EXAMS] || data?.upcoming_exams,
    },
    {
      label: 'Courses',
      icon: Library,
      to: paths.student.courses(),
      tone: 'bg-tint-brown text-tint-brown-ink',
      count: badges[BADGE_COURSES],
    },
    {
      label: 'Notices',
      icon: Megaphone,
      to: paths.student.notices(),
      tone: 'bg-tint-red text-tint-red-ink',
      count: badges[BADGE_NOTICES],
    },
    {
      label: 'Attendance',
      icon: CheckCircle2,
      to: paths.student.attendance(),
      tone: 'bg-tint-green text-tint-green-ink',
    },
    {
      label: 'Polls',
      icon: Vote,
      to: paths.student.polls(),
      tone: 'bg-tint-purple text-tint-purple-ink',
      count: badges[BADGE_POLLS] || data?.open_polls,
    },
    {
      label: 'Directory',
      icon: Users,
      to: paths.student.students(),
      tone: 'bg-tint-teal text-tint-teal-ink',
    },
    {
      label: 'Materials',
      icon: FolderOpen,
      to: paths.student.materials(),
      tone: 'bg-tint-yellow text-tint-yellow-ink',
      count: badges[BADGE_MATERIALS],
    },
  ]
  const refresh = () =>
    Promise.all([
      summary.reload(),
      routine.reload(),
      chats.reload(),
      attendance.reload(),
      crItems.reload(),
      crFinal.reload(),
    ])
  return (
    <PullToRefresh onRefresh={refresh}>
      <TodayHero
        name={user?.name || user?.identifier}
        regNo={user?.identifier}
        photo={user?.photo_url}
        meta={meta || (user?.batch ? `${user.batch} · Section ${user.section}` : '')}
        now={now}
      />
      <HomeStats
        classesToday={data?.classes_today}
        attendance={overall}
        semester={data?.semester_label}
      />
      {summary.error && (
        <Alert tone="error" className="mt-5">
          {summary.error}
        </Alert>
      )}
      <div className="mt-5">
        {routine.loading ? <SkeletonCard /> : <NextUpCard item={nextUp} />}
      </div>
      <QuickActions actions={visibleItems(features, actions)} className="mt-6" />
      <Section title="Today">
        {routine.error ? (
          <Alert tone="error">{routine.error}</Alert>
        ) : routine.loading ? (
          <SkeletonCard />
        ) : sorted.length === 0 ? (
          <EmptyState
            icon={CalendarClock}
            title="Nothing scheduled"
            text="No class or exam is planned for today."
          />
        ) : (
          <TodayClasses items={sorted} />
        )}
      </Section>
      {role === 'cr' && (
        <Section title="Your class">
          {crItems.loading || crFinal.loading ? (
            <SkeletonCard />
          ) : (
            <CRPanel
              items={crItems.data?.items || []}
              final={crFinal.data}
              today={todayDate()}
            />
          )}
        </Section>
      )}
      <Section title="Notices">
        {summary.loading ? (
          <SkeletonCard />
        ) : notices.length === 0 ? (
          <EmptyState
            icon={Megaphone}
            title="No notices"
            text="Nothing has been posted for your batch yet."
          />
        ) : (
          <NoticeCarousel notices={notices} role={role} />
        )}
      </Section>
      {exams.length > 0 && (
        <Section title="Next exams">
          <ExamStrip exams={exams} to={paths.student.exams()} />
        </Section>
      )}
      {messagesOn && (
        <Section title="Messages">
          {chats.loading ? (
            <SkeletonRows rows={3} />
          ) : unreadChats.length === 0 ? (
            <EmptyState
              icon={MessageCircle}
              title="You are all caught up"
              text="New messages from teachers and classmates land here."
            />
          ) : (
            <MessagePreview conversations={unreadChats} role={role} />
          )}
        </Section>
      )}
    </PullToRefresh>
  )
}

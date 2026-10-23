import {
  CalendarClock,
  ClipboardList,
  FolderOpen,
  Library,
  MessageCircle,
  Megaphone,
  Users,
  Vote,
} from 'lucide-react'
import CRPanel from '../../components/today/CRPanel'
import DayTimeline from '../../components/today/DayTimeline'
import ExamStrip from '../../components/today/ExamStrip'
import MessagePreview from '../../components/today/MessagePreview'
import NextUpCard from '../../components/today/NextUpCard'
import NoticeCarousel from '../../components/today/NoticeCarousel'
import QuickActions from '../../components/today/QuickActions'
import TodayHero from '../../components/today/TodayHero'
import {
  Alert,
  Card,
  EmptyState,
  PullToRefresh,
  Skeleton,
  SkeletonCard,
  SkeletonRows,
} from '../../components/ui'
import { useAuth } from '../../context/auth'
import { useNow, useResource } from '../../lib/hooks'
import { paths } from '../../lib/paths'
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
  const now = useNow(60000)
  const summary = useResource('/api/students/today')
  const routine = useResource('/api/students/upcoming-routine')
  const chats = useResource('/api/messages/conversations')
  const crItems = useResource(role === 'cr' ? '/api/cr/upcoming' : null)
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
  const meta = [data?.semester_label, data?.week ? `Week ${data.week}` : '']
    .filter(Boolean)
    .join(' · ')
  const actions = [
    { label: 'Routine', icon: CalendarClock, to: paths.student.upcomingRoutine() },
    {
      label: 'Exams',
      icon: ClipboardList,
      to: paths.student.exams(),
      count: data?.upcoming_exams,
    },
    { label: 'Courses', icon: Library, to: paths.student.courses() },
    { label: 'Notices', icon: Megaphone, to: paths.student.notices() },
    {
      label: 'Messages',
      icon: MessageCircle,
      to: paths.student.messages(),
      count: data?.unread_messages,
    },
    { label: 'Polls', icon: Vote, to: paths.student.polls(), count: data?.open_polls },
    { label: 'Directory', icon: Users, to: paths.student.students() },
    { label: 'Materials', icon: FolderOpen, to: paths.student.materials() },
  ]
  const refresh = () =>
    Promise.all([summary.reload(), routine.reload(), chats.reload(), crItems.reload(), crFinal.reload()])
  return (
    <PullToRefresh onRefresh={refresh}>
      <TodayHero
        name={user?.name || user?.identifier}
        meta={meta || (user?.batch ? `${user.batch} · Section ${user.section}` : '')}
        now={now}
      />
      {summary.error && (
        <Alert tone="error" className="mt-5">
          {summary.error}
        </Alert>
      )}
      <div className="mt-5">
        {routine.loading ? <SkeletonCard /> : <NextUpCard item={nextUp} />}
      </div>
      <QuickActions actions={actions} className="mt-7" />
      <Section title="Today">
        <Card hover={false}>
          {routine.error ? (
            <Alert tone="error">{routine.error}</Alert>
          ) : routine.loading ? (
            <div className="flex flex-col gap-3">
              <Skeleton className="h-4 w-2/3" />
              <Skeleton className="h-4 w-1/2" />
              <Skeleton className="h-4 w-3/5" />
            </div>
          ) : sorted.length === 0 ? (
            <EmptyState
              icon={CalendarClock}
              title="Nothing scheduled"
              text="No class or exam is planned for today."
              className="border-0 py-6"
            />
          ) : (
            <DayTimeline items={sorted} />
          )}
        </Card>
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
    </PullToRefresh>
  )
}

import { CalendarCheck, CalendarClock, ClipboardList, Mail, Megaphone, Vote } from 'lucide-react'
import AttachmentView from '../../components/AttachmentView'
import NoteLine from '../../components/NoteLine'
import StatCard from '../../components/StatCard'
import {
  Alert,
  Avatar,
  Card,
  EmptyState,
  SkeletonCard,
  SkeletonLines,
  StatusBadge,
} from '../../components/ui'
import { useAuth } from '../../context/auth'
import { useResource } from '../../lib/hooks'
import { paths } from '../../lib/paths'
import { formatDateTime, formatTimeRange } from '../../lib/time'
import cx from '../../lib/cx'

const KIND_LABELS = { lab: 'Lab', exam: 'Exam', extra: 'Extra class' }

const SOURCE_LABELS = { admin: 'Department', teacher: 'Teacher', cr: 'CR' }

const firstName = (name) => (name || '').trim().split(/\s+/)[0] || 'there'

const longDate = (value) => {
  const date = new Date(`${value}T00:00:00`)
  if (Number.isNaN(date.getTime())) return ''
  const weekday = date.toLocaleDateString('en-GB', { weekday: 'long' })
  const rest = date.toLocaleDateString('en-GB', { day: 'numeric', month: 'long', year: 'numeric' })
  return `${weekday} · ${rest}`
}

function TimelineRow({ item }) {
  const cancelled = item.status === 'cancelled'
  const kind = KIND_LABELS[item.kind]
  return (
    <li className="flex flex-wrap items-center gap-x-3 gap-y-1 border-b border-brand-50 py-2.5 text-sm last:border-b-0">
      <span className={cx('w-32 shrink-0 font-medium tabular-nums text-brand-600', cancelled && 'line-through')}>
        {formatTimeRange(item.start_time, item.end_time)}
      </span>
      <span className={cx('font-medium text-brand-900', cancelled && 'line-through')}>
        {item.course_code}
        {item.course_title ? ` ${item.course_title}` : ''}
        {kind ? ` ${kind}` : ''}
        {item.room ? ` @ ${item.room}` : ''}
      </span>
      {item.teacher_name && <span className="text-brand-500">{item.teacher_name}</span>}
      {cancelled && <StatusBadge status="cancelled" />}
      <NoteLine text={item.note} className="w-full" />
    </li>
  )
}

export default function TodayPage() {
  const { user } = useAuth()
  const summary = useResource('/api/students/today')
  const routine = useResource('/api/students/upcoming-routine')
  const data = summary.data
  const groups = routine.data?.groups || []
  const notices = data?.latest_notices || []
  const nextExam = data?.next_exam
  return (
    <>
      <p className="text-sm font-medium text-brand-500">{longDate(data?.date || routine.data?.date)}</p>
      <h1 className="font-display mt-1 text-2xl font-semibold text-brand-900 sm:text-3xl">
        Hello, {firstName(user?.name)}
      </h1>
      <p className="mt-1 text-sm text-brand-500">
        {user?.batch ? `${user.batch} · Section ${user.section}` : 'Your day at a glance'}
      </p>
      {summary.error && (
        <Alert tone="error" className="mt-5">
          {summary.error}
        </Alert>
      )}
      <div className="mt-5 grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <StatCard
          icon={CalendarCheck}
          label="Classes today"
          value={data?.classes_today || 0}
          to={paths.student.upcomingRoutine()}
        />
        <StatCard
          icon={ClipboardList}
          label="Upcoming exams"
          value={data?.upcoming_exams || 0}
          hint={nextExam ? `Next: ${nextExam.course} on ${nextExam.date}` : 'Nothing scheduled'}
          to={paths.student.exams()}
        />
        <StatCard
          icon={Mail}
          label="Unread messages"
          value={data?.unread_messages || 0}
          to={paths.student.messages()}
        />
        <StatCard
          icon={Vote}
          label="Open polls"
          value={data?.open_polls || 0}
          to={paths.student.polls()}
        />
      </div>
      <div className="mt-6 grid gap-5 lg:grid-cols-2">
        <Card hover={false}>
          <p className="font-display text-base font-semibold text-brand-800">Today</p>
          {routine.error && (
            <Alert tone="error" className="mt-3">
              {routine.error}
            </Alert>
          )}
          {routine.loading ? (
            <SkeletonLines lines={4} className="mt-4" />
          ) : groups.length === 0 ? (
            <EmptyState
              icon={CalendarClock}
              title="Nothing scheduled"
              text="No class or exam is planned for today."
              className="mt-4 border-0 py-8"
            />
          ) : (
            groups.map((group) => (
              <div key={group.key} className="mt-3">
                <p className="text-sm font-semibold text-brand-700">{group.section_label}:</p>
                <ul className="mt-1 flex flex-col">
                  {group.items.map((item, index) => (
                    <TimelineRow key={`${item.start_time}-${item.course_code}-${index}`} item={item} />
                  ))}
                </ul>
              </div>
            ))
          )}
        </Card>
        <Card hover={false}>
          <p className="font-display text-base font-semibold text-brand-800">Latest notices</p>
          {summary.loading ? (
            <SkeletonCard className="mt-4" />
          ) : notices.length === 0 ? (
            <EmptyState
              icon={Megaphone}
              title="No notices"
              text="Nothing has been posted for your batch yet."
              className="mt-4 border-0 py-8"
            />
          ) : (
            <ul className="mt-3 flex flex-col gap-4">
              {notices.map((notice) => (
                <li key={notice.id} className="border-b border-brand-50 pb-4 last:border-b-0 last:pb-0">
                  <div className="flex items-start gap-3">
                    <Avatar src={notice.posted_by_photo_url} name={notice.created_by_name} size="sm" />
                    <div className="min-w-0 flex-1">
                      <div className="flex flex-wrap items-center gap-2">
                        <p className="font-medium text-brand-900">{notice.title}</p>
                        <StatusBadge
                          status="unavailable"
                          label={SOURCE_LABELS[notice.source] || notice.source}
                        />
                      </div>
                      {notice.description && (
                        <p className="mt-1 text-sm whitespace-pre-line text-brand-600">
                          {notice.description}
                        </p>
                      )}
                      <AttachmentView
                        fileUrl={notice.file_url}
                        fileName={notice.file_name}
                        fileMime={notice.file_mime}
                        fileSize={notice.file_size}
                        linkUrl={notice.link_url}
                        className="mt-2"
                      />
                      <p className="mt-2 text-xs text-brand-400">
                        {notice.created_by_name} · {formatDateTime(notice.created_at)}
                      </p>
                    </div>
                  </div>
                </li>
              ))}
            </ul>
          )}
        </Card>
      </div>
    </>
  )
}

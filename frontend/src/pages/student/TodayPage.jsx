import { CalendarCheck, CalendarClock, ClipboardList, Mail, Megaphone, Vote } from 'lucide-react'
import AttachmentView from '../../components/AttachmentView'
import PublishedLine from '../../components/schedule/PublishedLine'
import StatCard from '../../components/StatCard'
import Masthead from '../../components/layout/Masthead'
import {
  Alert,
  Avatar,
  Card,
  CardHeader,
  EmptyState,
  SkeletonCard,
  SkeletonLines,
  StatusBadge,
} from '../../components/ui'
import { useAuth } from '../../context/auth'
import { useResource } from '../../lib/hooks'
import { paths } from '../../lib/paths'
import { formatDateTime } from '../../lib/time'

const SOURCE_LABELS = { admin: 'Department', teacher: 'Teacher', cr: 'CR' }

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
      <Masthead
        subtitle={user?.batch ? `${user.batch} · Section ${user.section}` : 'Your day at a glance'}
      />
      {summary.error && (
        <Alert tone="error" className="mt-5">
          {summary.error}
        </Alert>
      )}
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
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
        <Card hover={false} className="p-0">
          <CardHeader title="Today" />
          <div className="p-4 sm:p-5">
            {routine.error && <Alert tone="error">{routine.error}</Alert>}
            {routine.loading ? (
              <SkeletonLines lines={4} />
            ) : groups.length === 0 ? (
              <EmptyState
                icon={CalendarClock}
                title="Nothing scheduled"
                text="No class or exam is planned for today."
                className="border-0 py-8"
              />
            ) : (
              groups.map((group) => (
                <div key={group.key} className="mt-4 first:mt-0">
                  <p className="label-caps">{group.section_label}</p>
                  <ul className="mt-1 flex flex-col">
                    {group.items.map((item, index) => (
                      <PublishedLine
                        key={`${item.start_time}-${item.course_code}-${index}`}
                        item={item}
                        showBatch={group.drop_course}
                      />
                    ))}
                  </ul>
                </div>
              ))
            )}
          </div>
        </Card>
        <Card hover={false} className="p-0">
          <CardHeader title="Latest notices" />
          <div className="p-4 sm:p-5">
            {summary.loading ? (
              <SkeletonCard />
            ) : notices.length === 0 ? (
              <EmptyState
                icon={Megaphone}
                title="No notices"
                text="Nothing has been posted for your batch yet."
                className="border-0 py-8"
              />
            ) : (
              <ul className="flex flex-col gap-4">
                {notices.map((notice) => (
                  <li
                    key={notice.id}
                    className="border-b border-dashed border-line pb-4 last:border-b-0 last:pb-0"
                  >
                    <div className="flex items-start gap-3">
                      <Avatar
                        src={notice.posted_by_photo_url}
                        name={notice.created_by_name}
                        size="xs"
                      />
                      <div className="min-w-0 flex-1">
                        <div className="flex flex-wrap items-center gap-2">
                          <p className="text-sm font-semibold text-ink">{notice.title}</p>
                          <StatusBadge
                            status="unavailable"
                            label={SOURCE_LABELS[notice.source] || notice.source}
                          />
                        </div>
                        {notice.description && (
                          <p className="mt-1 text-sm leading-relaxed whitespace-pre-line text-ink-soft">
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
                        <p className="mt-2 text-xs text-faint">
                          {notice.created_by_name} · {formatDateTime(notice.created_at)}
                        </p>
                      </div>
                    </div>
                  </li>
                ))}
              </ul>
            )}
          </div>
        </Card>
      </div>
    </>
  )
}

import { useState } from 'react'
import { CalendarClock } from 'lucide-react'
import {
  Alert,
  Card,
  EmptyState,
  Input,
  PageHeader,
  SkeletonCard,
  StatusBadge,
} from '../../components/ui'
import { useResource } from '../../lib/hooks'
import { formatTimeRange } from '../../lib/time'
import cx from '../../lib/cx'

const KIND_LABELS = { lab: 'Lab', exam: 'Exam', extra: 'Extra class' }

function PlanLine({ item }) {
  const cancelled = item.status === 'cancelled'
  const kind = KIND_LABELS[item.kind]
  return (
    <li className="flex flex-wrap items-center gap-x-3 gap-y-1 border-b border-brand-50 py-2 text-sm last:border-b-0">
      <span className={cx('font-medium tabular-nums text-brand-600', cancelled && 'line-through')}>
        {formatTimeRange(item.start_time, item.end_time)}
      </span>
      <span className={cx('font-medium text-brand-900', cancelled && 'line-through')}>
        {item.course_code}
        {item.course_title ? ` ${item.course_title}` : ''}
        {kind ? ` ${kind}` : ''}
        {item.room ? ` @ ${item.room}` : ''}
      </span>
      {item.teacher_name && (
        <span className={cx('text-brand-500', cancelled && 'line-through')}>
          {item.teacher_name}
        </span>
      )}
      {cancelled && <StatusBadge status="cancelled" />}
    </li>
  )
}

export default function UpcomingRoutinePage() {
  const [picked, setPicked] = useState('')
  const { data, loading, error } = useResource(
    '/api/students/upcoming-routine',
    picked ? { date: picked } : null,
  )
  const groups = data?.groups || []
  return (
    <>
      <PageHeader
        title="Upcoming Routine"
        subtitle="The fixed routine and the approved events of your batch, merged for one day"
      />
      <Card hover={false} className="mb-5">
        <Input
          label="Day"
          type="date"
          value={picked || data?.date || ''}
          onChange={(event) => setPicked(event.target.value)}
          hint="Empty shows tomorrow"
          className="max-w-xs"
        />
      </Card>
      {error && <Alert tone="error" className="mb-5">{error}</Alert>}
      {loading ? (
        <SkeletonCard />
      ) : groups.length === 0 ? (
        <EmptyState
          icon={CalendarClock}
          title="Nothing scheduled"
          text="No class or exam is planned for this day."
        />
      ) : (
        <div className="flex flex-col gap-4">
          <p className="font-display text-lg font-semibold text-brand-800">
            {data.batch} · {data.date_label} ({data.weekday})
          </p>
          {groups.map((group) => (
            <Card key={group.section} hover={false} className="animate-fade-in-up">
              <p className="text-sm font-semibold text-brand-700">{group.section_label}:</p>
              <ul className="mt-2 flex flex-col">
                {group.items.map((item, index) => (
                  <PlanLine key={`${item.start_time}-${item.course_code}-${index}`} item={item} />
                ))}
              </ul>
            </Card>
          ))}
        </div>
      )}
    </>
  )
}

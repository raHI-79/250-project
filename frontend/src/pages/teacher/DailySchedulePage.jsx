import { useState } from 'react'
import { CalendarClock, ChevronLeft, ChevronRight } from 'lucide-react'
import {
  Alert,
  Button,
  Card,
  EmptyState,
  Input,
  PageHeader,
  SkeletonCard,
  StatusBadge,
} from '../../components/ui'
import { useResource } from '../../lib/hooks'
import { formatTimeRange, shiftDate, today } from '../../lib/time'
import cx from '../../lib/cx'

const KIND_LABELS = { fixed: 'Fixed', class: 'Class', exam: 'Exam', extra: 'Extra' }

const sectionLabel = (section) => (section === 'ALL' ? 'Both sections' : `Section ${section}`)

const byTime = (left, right) => left.start_time.localeCompare(right.start_time)

export default function DailySchedulePage() {
  const [date, setDate] = useState(today())
  const { data, loading, error } = useResource('/api/teacher/schedule', { date })
  const items = [...(data?.items || [])].sort(byTime)
  return (
    <>
      <PageHeader
        title="Daily Schedule"
        subtitle="Your fixed routine for the day plus the approved classes and exams"
      />
      <Card hover={false} className="mb-5">
        <div className="flex flex-wrap items-end gap-3">
          <Input
            label="Date"
            type="date"
            value={date}
            onChange={(event) => setDate(event.target.value || today())}
            className="max-w-xs"
          />
          <div className="flex gap-2">
            <Button variant="secondary" onClick={() => setDate((current) => shiftDate(current, -1))}>
              <ChevronLeft size={15} />
              Previous day
            </Button>
            <Button variant="secondary" onClick={() => setDate((current) => shiftDate(current, 1))}>
              Next day
              <ChevronRight size={15} />
            </Button>
          </div>
        </div>
      </Card>
      {error && <Alert tone="error" className="mb-5">{error}</Alert>}
      {loading ? (
        <SkeletonCard />
      ) : items.length === 0 ? (
        <EmptyState
          icon={CalendarClock}
          title="Nothing scheduled"
          text="You have no class or exam on this day."
        />
      ) : (
        <Card hover={false} className="animate-fade-in-up">
          <p className="font-display text-base font-semibold text-brand-800">
            {data.date} ({data.weekday})
          </p>
          <ul className="mt-2 flex flex-col">
            {items.map((item) => (
              <li
                key={`${item.kind}-${item.id}-${item.start_time}`}
                className="flex flex-wrap items-center gap-x-3 gap-y-1 border-b border-brand-50 py-2.5 text-sm last:border-b-0"
              >
                <span
                  className={cx(
                    'w-40 shrink-0 font-medium tabular-nums text-brand-600',
                    item.cancelled && 'line-through',
                  )}
                >
                  {formatTimeRange(item.start_time, item.end_time)}
                </span>
                <span className={cx('text-brand-700', item.cancelled && 'line-through')}>
                  {item.batch} · {sectionLabel(item.section)}
                </span>
                <span className={cx('font-medium text-brand-900', item.cancelled && 'line-through')}>
                  {item.course_code}
                  {item.course_title ? ` ${item.course_title}` : ''}
                  {item.room ? ` @ ${item.room}` : ''}
                </span>
                <StatusBadge
                  status={item.kind === 'fixed' ? 'unavailable' : 'approved'}
                  label={KIND_LABELS[item.kind] || item.kind}
                />
                {item.cancelled && <StatusBadge status="cancelled" />}
              </li>
            ))}
          </ul>
        </Card>
      )}
    </>
  )
}

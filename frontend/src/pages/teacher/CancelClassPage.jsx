import { useState } from 'react'
import { CalendarX } from 'lucide-react'
import {
  Alert,
  Button,
  Card,
  EmptyState,
  Input,
  PageHeader,
  SkeletonCard,
  StatusBadge,
  useConfirm,
  useToast,
} from '../../components/ui'
import api from '../../lib/api'
import { useResource } from '../../lib/hooks'
import { formatDate, formatTimeRange, today } from '../../lib/time'
import cx from '../../lib/cx'

const CANCELLED_MESSAGE = 'Class cancelled. Students and the CR have been notified.'

const TABS = [
  { value: 'upcoming', label: 'Upcoming classes' },
  { value: 'date', label: 'Browse by date' },
]

const KIND_LABELS = { fixed: 'Fixed', class: 'Class', exam: 'Exam', extra: 'Extra' }

const TYPE_LABELS = { class: 'Class', exam: 'Exam', extra: 'Extra class' }

const sectionLabel = (section) => (section === 'ALL' ? 'Both sections' : `Section ${section}`)

const byTime = (left, right) => left.start_time.localeCompare(right.start_time)

export default function CancelClassPage() {
  const toast = useToast()
  const confirm = useConfirm()
  const [tab, setTab] = useState('upcoming')
  const [date, setDate] = useState(today())
  const [busy, setBusy] = useState('')
  const upcoming = useResource(tab === 'upcoming' ? '/api/teacher/classes/upcoming' : null)
  const schedule = useResource(tab === 'date' ? '/api/teacher/schedule' : null, { date })
  const cancelClass = async (item, label) => {
    const ok = await confirm({
      title: 'Cancel this class?',
      message: `${item.course_code} for ${item.batch} ${sectionLabel(item.section)} on ${label} will be marked cancelled.`,
      confirmLabel: 'Cancel the class',
    })
    if (!ok) return
    setBusy(`class-${item.id}`)
    try {
      await api.post(`/api/teacher/classes/${item.id}/cancel`)
      toast.success(CANCELLED_MESSAGE)
      upcoming.reload()
      schedule.reload()
    } catch (failure) {
      toast.error(failure.message)
    } finally {
      setBusy('')
    }
  }
  const cancelRoutineDay = async (item) => {
    const ok = await confirm({
      title: 'Cancel this day?',
      message: `${item.course_code} for ${item.batch} ${sectionLabel(item.section)} on ${formatDate(date)} will be cancelled for this date only.`,
      confirmLabel: 'Cancel this day',
    })
    if (!ok) return
    setBusy(`routine-${item.id}`)
    try {
      await api.post(`/api/teacher/routines/${item.id}/cancel`, { date })
      toast.success(CANCELLED_MESSAGE)
      schedule.reload()
    } catch (failure) {
      toast.error(failure.message)
    } finally {
      setBusy('')
    }
  }
  const classes = upcoming.data?.items || []
  const items = [...(schedule.data?.items || [])].sort(byTime)
  return (
    <>
      <PageHeader
        title="Cancel Class"
        subtitle="Cancel an approved class or one day of your fixed routine"
        actions={
          <div className="inline-flex rounded-lg border border-brand-200 bg-surface p-1">
            {TABS.map((entry) => (
              <Button
                key={entry.value}
                size="sm"
                variant={tab === entry.value ? 'primary' : 'ghost'}
                onClick={() => setTab(entry.value)}
              >
                {entry.label}
              </Button>
            ))}
          </div>
        }
      />
      {tab === 'upcoming' ? (
        <>
          {upcoming.error && <Alert tone="error" className="mb-5">{upcoming.error}</Alert>}
          {upcoming.loading ? (
            <SkeletonCard />
          ) : classes.length === 0 ? (
            <EmptyState
              icon={CalendarX}
              title="No upcoming classes"
              text="You have no approved class ahead."
            />
          ) : (
            <div className="flex flex-col gap-4">
              {classes.map((item) => (
                <Card key={item.id} hover={false} className="animate-fade-in-up">
                  <div className="flex flex-wrap items-center gap-x-3 gap-y-2 text-sm">
                    <span className="font-medium text-brand-700">
                      {formatDate(item.event_date)}
                    </span>
                    <span className="font-medium tabular-nums text-brand-600">
                      {formatTimeRange(item.start_time, item.end_time)}
                    </span>
                    <span className="font-medium text-brand-900">
                      {item.course_code}
                      {item.room ? ` @ ${item.room}` : ''}
                    </span>
                    <span className="text-brand-700">
                      {item.batch} · {sectionLabel(item.section)}
                    </span>
                    <StatusBadge
                      status="approved"
                      label={TYPE_LABELS[item.event_type] || item.event_type}
                    />
                    <Button
                      size="sm"
                      variant="danger"
                      loading={busy === `class-${item.id}`}
                      onClick={() => cancelClass(item, formatDate(item.event_date))}
                      className="ml-auto"
                    >
                      Cancel
                    </Button>
                  </div>
                </Card>
              ))}
            </div>
          )}
        </>
      ) : (
        <>
          <Card hover={false} className="mb-5">
            <Input
              label="Date"
              type="date"
              value={date}
              onChange={(event) => setDate(event.target.value || today())}
              className="max-w-xs"
            />
          </Card>
          {schedule.error && <Alert tone="error" className="mb-5">{schedule.error}</Alert>}
          {schedule.loading ? (
            <SkeletonCard />
          ) : items.length === 0 ? (
            <EmptyState
              icon={CalendarX}
              title="Nothing scheduled"
              text="You have no class or exam on this day."
            />
          ) : (
            <Card hover={false} className="animate-fade-in-up">
              <ul className="flex flex-col">
                {items.map((item) => (
                  <li
                    key={`${item.kind}-${item.id}`}
                    className="flex flex-wrap items-center gap-x-3 gap-y-2 border-b border-brand-50 py-2.5 text-sm last:border-b-0"
                  >
                    <span
                      className={cx(
                        'font-medium tabular-nums text-brand-600',
                        item.cancelled && 'line-through',
                      )}
                    >
                      {formatTimeRange(item.start_time, item.end_time)}
                    </span>
                    <span
                      className={cx('font-medium text-brand-900', item.cancelled && 'line-through')}
                    >
                      {item.course_code}
                      {item.room ? ` @ ${item.room}` : ''}
                    </span>
                    <span className={cx('text-brand-700', item.cancelled && 'line-through')}>
                      {item.batch} · {sectionLabel(item.section)}
                    </span>
                    <StatusBadge
                      status={item.kind === 'fixed' ? 'unavailable' : 'approved'}
                      label={KIND_LABELS[item.kind] || item.kind}
                    />
                    {item.cancelled ? (
                      <StatusBadge status="cancelled" className="ml-auto" />
                    ) : item.kind === 'fixed' ? (
                      <Button
                        size="sm"
                        variant="danger"
                        loading={busy === `routine-${item.id}`}
                        onClick={() => cancelRoutineDay(item)}
                        className="ml-auto"
                      >
                        Cancel this day
                      </Button>
                    ) : (
                      <Button
                        size="sm"
                        variant="danger"
                        loading={busy === `class-${item.id}`}
                        onClick={() => cancelClass(item, formatDate(date))}
                        className="ml-auto"
                      >
                        Cancel
                      </Button>
                    )}
                  </li>
                ))}
              </ul>
            </Card>
          )}
        </>
      )}
    </>
  )
}

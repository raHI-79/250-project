import { useEffect } from 'react'
import { CalendarSearch } from 'lucide-react'
import { Alert, Card, EmptyState, SkeletonLines, StatusBadge } from '../ui'
import { useResource } from '../../lib/hooks'
import { formatTimeRange, overlaps } from '../../lib/time'
import cx from '../../lib/cx'

const SOURCE_LABELS = {
  fixed: 'Fixed routine',
  class: 'Class',
  exam: 'Exam',
  extra: 'Extra class',
}

const byRoomThenTime = (left, right) => {
  if (left.room !== right.room) return left.room.localeCompare(right.room)
  return left.start_time.localeCompare(right.start_time)
}

const sameRoom = (left, right) =>
  left.trim().toLowerCase() === right.trim().toLowerCase() && left.trim() !== ''

export default function RoomAvailabilityPanel({ date, room = '', start = '', end = '', onStatus }) {
  const { data, loading, error } = useResource(
    date ? '/api/cr/room-availability' : null,
    date ? { date } : null,
  )
  const items = [...(data?.items || [])].sort(byRoomThenTime)
  const clashing = (item) =>
    Boolean(room && start && end) &&
    sameRoom(item.room, room) &&
    overlaps(start, end, item.start_time, item.end_time)
  const conflicts = items.filter(clashing).length
  const loaded = Boolean(date) && !loading && !error
  useEffect(() => {
    if (onStatus) onStatus({ loaded, conflicts })
  }, [onStatus, loaded, conflicts])
  if (!date) {
    return (
      <Card hover={false}>
        <p className="font-display text-lg font-bold text-ink">Room availability</p>
        <p className="mt-2 text-sm text-muted">
          Choose a date to see everything booked that day.
        </p>
      </Card>
    )
  }
  return (
    <Card hover={false}>
      <p className="font-display text-lg font-bold text-ink">Room availability</p>
      <p className="mt-1 text-sm text-muted">
        Everything booked on {date}, from the fixed routine and the dated bookings.
      </p>
      {error && (
        <Alert tone="error" className="mt-3">
          {error}
        </Alert>
      )}
      {conflicts > 0 && (
        <Alert tone="warning" title="This room is taken" className="mt-3">
          {conflicts} booking{conflicts === 1 ? '' : 's'} overlap the room and time you chose.
          Pick another room or another time before you send the request.
        </Alert>
      )}
      {loading ? (
        <SkeletonLines lines={4} className="mt-4" />
      ) : items.length === 0 ? (
        <EmptyState
          icon={CalendarSearch}
          title="Every room is free"
          text="Nothing is booked on this date yet."
          className="mt-4 border-0 py-8"
        />
      ) : (
        <ul className="mt-3 flex flex-col">
          {items.map((item, index) => {
            const clash = clashing(item)
            return (
              <li
                key={`${item.room}-${item.start_time}-${item.course_code}-${index}`}
                className={cx(
                  'flex flex-wrap items-center gap-x-3 gap-y-1 border-b border-line px-2 py-2 text-sm last:border-b-0',
                  clash && 'rounded-md bg-amber-50 ring-1 ring-amber-200',
                )}
              >
                <span className="w-16 shrink-0 font-semibold text-ink">{item.room}</span>
                <span className="font-medium tabular-nums text-brand-600">
                  {formatTimeRange(item.start_time, item.end_time)}
                </span>
                <span className="text-ink-soft">
                  {item.batch}
                  {item.section ? ` · ${item.section === 'ALL' ? 'Both' : item.section}` : ''}
                </span>
                <span className="font-medium text-ink">{item.course_code}</span>
                <span className="text-xs text-faint">
                  {SOURCE_LABELS[item.source] || item.source}
                </span>
                <StatusBadge
                  status={item.label === 'yours' ? 'approved' : 'rejected'}
                  label={item.label === 'yours' ? 'yours' : 'unavailable'}
                  className="ml-auto"
                />
              </li>
            )
          })}
        </ul>
      )}
    </Card>
  )
}

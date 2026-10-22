import { Video } from 'lucide-react'
import { Button, StatusBadge } from '../ui'
import NoteLine from '../NoteLine'
import { STATUS_CANCELLED, isOnline, kindLabel, statusLabel } from '../../lib/schedule'
import { formatTimeRange } from '../../lib/time'
import cx from '../../lib/cx'

export default function PublishedLine({ item, showBatch = false }) {
  const cancelled = item.status === STATUS_CANCELLED
  const struck = cancelled && 'line-through'
  return (
    <li className="flex flex-wrap items-center gap-x-3 gap-y-1 border-b border-brand-50 py-2.5 text-sm last:border-b-0">
      <span className={cx('font-medium tabular-nums text-brand-600', struck)}>
        {formatTimeRange(item.start_time, item.end_time)}
      </span>
      {showBatch && (
        <span className="rounded-full border border-brand-200 bg-brand-50 px-2 py-0.5 text-xs font-medium text-brand-700">
          {item.batch}
        </span>
      )}
      <span className={cx('font-medium text-brand-900', struck)}>
        {item.course_code}
        {item.course_title ? ` ${item.course_title}` : ''}
        {item.kind !== 'class' ? ` ${kindLabel(item.kind)}` : ''}
        {!isOnline(item) && item.room ? ` @ ${item.room}` : ''}
      </span>
      {item.teacher_name && (
        <span className={cx('text-brand-500', struck)}>{item.teacher_name}</span>
      )}
      {isOnline(item) && (
        <span className="inline-flex items-center gap-1 rounded-full border border-brand-200 bg-brand-50 px-2.5 py-0.5 text-xs font-medium text-brand-700">
          <Video size={12} />
          Online
        </span>
      )}
      {isOnline(item) && item.meeting_link && !cancelled && (
        <a href={item.meeting_link} target="_blank" rel="noreferrer">
          <Button size="sm">Join</Button>
        </a>
      )}
      {cancelled && <StatusBadge status="cancelled" label={statusLabel(item.status)} />}
      <NoteLine text={item.note} className="w-full" />
    </li>
  )
}

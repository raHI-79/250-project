import { Video } from 'lucide-react'
import { Button, StatusBadge } from '../ui'
import NoteLine from '../NoteLine'
import TimelineItem from './TimelineItem'
import CourseName from './CourseName'
import { STATUS_CANCELLED, isOnline, kindLabel, statusLabel } from '../../lib/schedule'
import cx from '../../lib/cx'

export default function PublishedLine({ item, showBatch = false }) {
  const cancelled = item.status === STATUS_CANCELLED
  const struck = cancelled && 'line-through'
  return (
    <TimelineItem
      start={item.start_time}
      end={item.end_time}
      tone={cancelled ? 'muted' : 'brand'}
    >
      <div className="flex flex-wrap items-center gap-x-2.5 gap-y-1.5">
        <CourseName
          item={item}
          className={cx('text-sm font-semibold text-ink', struck)}
        />
        {item.kind !== 'class' && (
          <StatusBadge status="unavailable" label={kindLabel(item.kind)} />
        )}
        {showBatch && <StatusBadge status="unavailable" label={item.batch} />}
        {cancelled && <StatusBadge status="cancelled" label={statusLabel(item.status)} />}
        {isOnline(item) && <StatusBadge status="gold" label="Online" />}
        {isOnline(item) && item.meeting_link && !cancelled && (
          <a href={item.meeting_link} target="_blank" rel="noreferrer">
            <Button size="sm" variant="secondary">
              <Video size={13} />
              Join
            </Button>
          </a>
        )}
      </div>
      <p className={cx('mt-1 text-xs text-muted', struck)}>
        {!isOnline(item) && item.room ? `Room ${item.room}` : isOnline(item) ? 'Online class' : ''}
        {item.teacher_name && (!isOnline(item) && item.room ? ' · ' : '')}
        {item.teacher_name}
      </p>
      <NoteLine text={item.note} className="mt-2" />
    </TimelineItem>
  )
}

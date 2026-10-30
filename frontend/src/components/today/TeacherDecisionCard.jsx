import { Link2, Video } from 'lucide-react'
import ItemBadges from '../schedule/ItemBadges'
import { Alert, Button, Card } from '../ui'
import cx from '../../lib/cx'
import {
  ACTION_ACCEPT,
  ACTION_CANCEL,
  ACTION_CONFIRM,
  ACTION_DECLINE,
  ACTION_REQUEST_RESCHEDULE,
  STATUS_CANCELLED,
  STATUS_CR_ASK,
  allows,
  isOnline,
  needsLink,
  placeOf,
  sectionShort,
  teacherStatusLabel,
} from '../../lib/schedule'
import { formatDate, formatTimeRange } from '../../lib/time'
import CourseName from '../schedule/CourseName'

export default function TeacherDecisionCard({
  item,
  busy,
  highlighted,
  onConfirm,
  onCancel,
  onAsk,
  onLink,
  onDecide,
}) {
  const cancelled = item.status === STATUS_CANCELLED
  return (
    <Card
      hover={false}
      raised
      className={cx('animate-fade-in-up', highlighted && 'border-brand-300 bg-brand-50/60')}
    >
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <CourseName
            item={item}
            as="p"
            className="font-display text-base font-bold text-ink"
          />
          <p className="mt-1 text-xs text-muted">
            {item.batch} {sectionShort(item.section)}
            {placeOf(item) ? ` · ${placeOf(item)}` : ''}
          </p>
        </div>
        <span className="shrink-0 text-sm font-semibold text-brand-600 tabular-nums">
          {formatTimeRange(item.start_time, item.end_time) || 'No time yet'}
        </span>
      </div>
      <div className="mt-2 flex flex-wrap gap-1.5">
        <ItemBadges item={item} statusText={teacherStatusLabel(item)} />
      </div>
      {item.status === STATUS_CR_ASK && item.proposed && (
        <Alert tone="info" title="The class representative proposed" className="mt-3">
          {formatDate(item.proposed.item_date)}{' '}
          {formatTimeRange(item.proposed.start_time, item.proposed.end_time)}
          {item.proposed.room ? ` · ${item.proposed.room}` : ''}
          {item.proposed.mode === 'online' ? ' · Online' : ''}
        </Alert>
      )}
      {item.cancel_reason && <p className="mt-2 text-xs text-muted">Reason: {item.cancel_reason}</p>}
      <div className="mt-4 flex flex-wrap gap-2">
        {allows(item, ACTION_ACCEPT) && (
          <Button size="sm" loading={busy} onClick={() => onDecide(item, true)}>
            Accept
          </Button>
        )}
        {allows(item, ACTION_DECLINE) && (
          <Button size="sm" variant="secondary" loading={busy} onClick={() => onDecide(item, false)}>
            Decline
          </Button>
        )}
        {allows(item, ACTION_CONFIRM) && (
          <Button size="sm" onClick={() => onConfirm(item)}>
            Confirm
          </Button>
        )}
        {allows(item, ACTION_REQUEST_RESCHEDULE) && (
          <Button size="sm" variant="secondary" onClick={() => onAsk(item)}>
            Reschedule
          </Button>
        )}
        {allows(item, ACTION_CANCEL) && (
          <Button size="sm" variant="danger" onClick={() => onCancel(item)}>
            Cancel
          </Button>
        )}
        {isOnline(item) && !cancelled && (
          <Button size="sm" variant="secondary" onClick={() => onLink(item)}>
            {needsLink(item) ? <Link2 size={14} /> : <Video size={14} />}
            {needsLink(item) ? 'Add link' : 'Edit link'}
          </Button>
        )}
      </div>
    </Card>
  )
}

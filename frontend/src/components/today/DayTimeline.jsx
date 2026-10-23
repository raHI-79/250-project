import { Video } from 'lucide-react'
import { StatusBadge } from '../ui'
import cx from '../../lib/cx'
import { STATUS_CANCELLED, isOnline, kindLabel, placeOf } from '../../lib/schedule'
import { formatClock } from '../../lib/time'

export default function DayTimeline({ items, className }) {
  return (
    <ol className={cx('flex flex-col', className)}>
      {items.map((item, index) => {
        const cancelled = item.status === STATUS_CANCELLED
        return (
          <li
            key={`${item.start_time}-${item.course_code}-${index}`}
            className="flex gap-3 border-b border-line py-3 last:border-b-0 last:pb-0 first:pt-0"
          >
            <div className="w-16 shrink-0 pt-0.5">
              <p className="text-xs font-bold text-ink tabular-nums">
                {formatClock(item.start_time)}
              </p>
              <p className="text-2xs text-faint tabular-nums">{formatClock(item.end_time)}</p>
            </div>
            <span
              aria-hidden="true"
              className={cx(
                'mt-1 w-1 shrink-0 rounded-full',
                cancelled ? 'bg-line-strong' : 'bg-brand-600',
              )}
            />
            <div className="min-w-0 flex-1">
              <p
                className={cx(
                  'truncate text-sm font-semibold text-ink',
                  cancelled && 'text-muted line-through',
                )}
              >
                {item.course_code}
                {item.course_title ? ` · ${item.course_title}` : ''}
              </p>
              <p className="mt-0.5 truncate text-xs text-muted">
                {[placeOf(item), item.teacher_name].filter(Boolean).join(' · ')}
              </p>
              <div className="mt-1.5 flex flex-wrap items-center gap-1.5">
                {item.kind !== 'class' && (
                  <StatusBadge status="unavailable" label={kindLabel(item.kind)} />
                )}
                {cancelled && <StatusBadge status="cancelled" />}
                {isOnline(item) && !cancelled && (
                  <span className="inline-flex items-center gap-1 text-2xs font-semibold text-gold-700">
                    <Video size={12} />
                    Online
                  </span>
                )}
              </div>
            </div>
          </li>
        )
      })}
    </ol>
  )
}

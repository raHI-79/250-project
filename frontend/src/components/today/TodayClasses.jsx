import { MapPin, Video } from 'lucide-react'
import { Carousel, StatusBadge } from '../ui'
import cx from '../../lib/cx'
import {
  STATUS_CANCELLED,
  courseCode,
  courseLine,
  isOnline,
  kindLabel,
  placeOf,
} from '../../lib/schedule'
import { formatClock } from '../../lib/time'

const PASTELS = [
  'bg-tint-blue',
  'bg-tint-green',
  'bg-tint-orange',
  'bg-tint-purple',
  'bg-tint-teal',
  'bg-tint-yellow',
]

export default function TodayClasses({ items, className }) {
  return (
    <Carousel label="Today's classes" className={className}>
      {items.map((item, index) => {
        const cancelled = item.status === STATUS_CANCELLED
        return (
          <article
            key={`${item.start_time}-${item.course_code}-${index}`}
            className={cx(
              'flex w-[15.5rem] shrink-0 snap-start flex-col gap-2 rounded-card border border-line p-4',
              cancelled ? 'bg-wash' : PASTELS[index % PASTELS.length],
            )}
          >
            <p className="text-xs font-extrabold text-ink-soft tabular-nums">
              {formatClock(item.start_time)}
              <span className="font-semibold text-muted"> – {formatClock(item.end_time)}</span>
            </p>
            <p
              className={cx(
                'wrap-long text-sm leading-snug font-bold text-ink',
                cancelled && 'text-muted line-through',
              )}
            >
              {courseLine(item)}
            </p>
            {courseCode(item) && (
              <p className="wrap-long text-2xs font-bold text-ink-soft">{courseCode(item)}</p>
            )}
            <p className="mt-auto flex items-start gap-1.5 wrap-long text-xs text-ink-soft">
              {isOnline(item) ? (
                <Video size={13} className="mt-0.5 shrink-0" />
              ) : (
                <MapPin size={13} className="mt-0.5 shrink-0" />
              )}
              <span className="min-w-0">
                {[placeOf(item), item.teacher_name].filter(Boolean).join(' · ')}
              </span>
            </p>
            <div className="flex flex-wrap items-center gap-1.5">
              {item.kind !== 'class' && <StatusBadge tone="brand" label={kindLabel(item.kind)} />}
              {cancelled && <StatusBadge status="cancelled" />}
              {isOnline(item) && !cancelled && <StatusBadge tone="warning" label="Online" />}
            </div>
          </article>
        )
      })}
    </Carousel>
  )
}

import { useEffect, useState } from 'react'
import { CalendarCheck, MapPin, Video } from 'lucide-react'
import { Button, Card, StatusBadge } from '../ui'
import {
  STATUS_CANCELLED,
  isOnline,
  kindLabel,
  secondsUntilClock,
  startsInText,
} from '../../lib/schedule'
import { formatTimeRange } from '../../lib/time'
import CourseName from '../schedule/CourseName'
import { externalLinkProps } from '../../lib/native'

export default function NextUpCard({ item }) {
  const [seconds, setSeconds] = useState(() => secondsUntilClock(item?.start_time))
  useEffect(() => {
    if (!item) return undefined
    setSeconds(secondsUntilClock(item.start_time))
    const timer = window.setInterval(() => setSeconds(secondsUntilClock(item.start_time)), 1000)
    return () => window.clearInterval(timer)
  }, [item])
  if (!item) {
    return (
      <Card hover={false} raised className="flex items-center gap-3">
        <span className="grid size-11 shrink-0 place-items-center rounded-full bg-success-50 text-success-600">
          <CalendarCheck size={20} />
        </span>
        <div className="min-w-0">
          <p className="text-sm font-semibold text-ink">Nothing left today</p>
          <p className="text-xs text-muted">No more classes or exams on your routine.</p>
        </div>
      </Card>
    )
  }
  const online = isOnline(item)
  const cancelled = item.status === STATUS_CANCELLED
  return (
    <Card hover={false} raised className="border-l-[3px] border-l-brand-600">
      <div className="flex items-start justify-between gap-3">
        <p className="label-caps">Next up</p>
        <span className="font-numeral text-sm font-bold text-brand-600 tabular-nums">
          {cancelled ? 'Cancelled' : startsInText(seconds)}
        </span>
      </div>
      <p className="font-display mt-2 text-section leading-tight font-bold text-ink wrap-long">
        <CourseName item={item} codeClassName="text-white/70" />
      </p>
      <p className="mt-1.5 text-sm font-medium text-ink-soft tabular-nums">
        {formatTimeRange(item.start_time, item.end_time)}
      </p>
      <div className="mt-2 flex flex-wrap items-center gap-x-2 gap-y-1.5 text-xs text-muted">
        {item.kind !== 'class' && <StatusBadge status="unavailable" label={kindLabel(item.kind)} />}
        {cancelled && <StatusBadge status="cancelled" />}
        {online ? (
          <span className="inline-flex items-center gap-1">
            <Video size={13} />
            Online
          </span>
        ) : item.room ? (
          <span className="inline-flex items-center gap-1">
            <MapPin size={13} />
            Room {item.room}
          </span>
        ) : null}
        {item.teacher_name && <span>{item.teacher_name}</span>}
      </div>
      {online && item.meeting_link && !cancelled && (
        <a {...externalLinkProps(item.meeting_link)} className="mt-4 block">
          <Button as="span" className="w-full">
            <Video size={16} />
            Join the class
          </Button>
        </a>
      )}
    </Card>
  )
}

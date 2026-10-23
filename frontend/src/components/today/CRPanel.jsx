import { Link } from 'react-router-dom'
import { CalendarClock, CircleCheck, CircleX, Send } from 'lucide-react'
import { Button, Card, ListRow } from '../ui'
import paths, { CR_TAB_FINAL, CR_TAB_UPCOMING } from '../../lib/paths'
import {
  STATUS_CANCELLED,
  STATUS_CONFIRMED,
  STATUS_TEACHER_ASK,
  courseLine,
} from '../../lib/schedule'
import { formatClock } from '../../lib/time'

export default function CRPanel({ items, final, today }) {
  const decisions = items.filter(
    (item) =>
      item.item_date === today &&
      (item.status === STATUS_CONFIRMED || item.status === STATUS_CANCELLED),
  )
  const needSlot = items.filter((item) => item.status === STATUS_TEACHER_ASK)
  const finalItems = (final?.days || []).flatMap((day) => day.items)
  const unpublished = finalItems.filter((item) => !item.published || item.changed_since_publish)
  return (
    <Card hover={false} raised flush className="overflow-hidden">
      <div className="flex items-center justify-between gap-3 px-4 pt-4 pb-2 sm:px-5">
        <h2 className="font-display text-section font-bold text-ink">Class representative</h2>
        <Link
          to={paths.cr.schedule()}
          className="text-xs font-semibold text-brand-600 hover:underline"
        >
          Open hub
        </Link>
      </div>
      {decisions.length > 0 && (
        <div className="px-4 pb-2 sm:px-5">
          <p className="label-caps">Today from teachers</p>
          <ul className="mt-2 flex flex-col gap-1.5">
            {decisions.map((item) => (
              <li key={item.id} className="flex items-center gap-2 text-sm">
                {item.status === STATUS_CONFIRMED ? (
                  <CircleCheck size={15} className="shrink-0 text-success-600" />
                ) : (
                  <CircleX size={15} className="shrink-0 text-danger-600" />
                )}
                <span className="min-w-0 flex-1 truncate text-ink-soft">{courseLine(item)}</span>
                <span className="shrink-0 text-xs text-muted tabular-nums">
                  {formatClock(item.start_time)}
                </span>
              </li>
            ))}
          </ul>
        </div>
      )}
      <div className="mt-2 divide-y divide-line border-t border-line">
        <ListRow
          icon={CalendarClock}
          iconTone={needSlot.length > 0 ? 'gold' : 'neutral'}
          title="Needs a new slot"
          subtitle={
            needSlot.length > 0
              ? `${needSlot.length} class ${needSlot.length === 1 ? 'waits' : 'wait'} for another time`
              : 'No teacher is waiting for a new slot'
          }
          value={needSlot.length > 0 ? String(needSlot.length) : ''}
          to={paths.cr.schedule({ tab: CR_TAB_UPCOMING })}
        />
        <ListRow
          icon={Send}
          iconTone={unpublished.length > 0 ? 'brand' : 'neutral'}
          title={unpublished.length > 0 ? 'Final routine ready' : 'Final routine'}
          subtitle={
            unpublished.length > 0
              ? `${unpublished.length} item${unpublished.length === 1 ? '' : 's'} waiting for the batch`
              : 'Everything in the final routine is published'
          }
          to={paths.cr.schedule({ tab: CR_TAB_FINAL })}
          trailing={
            unpublished.length > 0 ? (
              <Button
                as="span"
                size="sm"
                className="pointer-events-none px-3"
              >
                Publish
              </Button>
            ) : undefined
          }
          chevron={unpublished.length === 0}
        />
      </div>
    </Card>
  )
}

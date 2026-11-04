import cx from '../../lib/cx'
import { formatClock } from '../../lib/time'

export default function TimelineItem({ start, end, tone = 'brand', className, children }) {
  return (
    <li className={cx('grid grid-cols-[1.125rem_1fr] gap-x-3 pb-4 last:pb-0', className)}>
      <span aria-hidden="true" className="row-span-2 flex flex-col items-center">
        <span
          className={cx(
            'mt-1 size-3 shrink-0 rounded-full border-[3px] bg-surface',
            tone === 'muted' ? 'border-line-strong' : 'border-brand-600',
          )}
        />
        <span className="mt-1 w-px flex-1 bg-line" />
      </span>
      <span className="text-xs leading-tight font-extrabold text-ink tabular-nums">
        {formatClock(start)}
        {end && <span className="font-semibold text-muted"> – {formatClock(end)}</span>}
      </span>
      <div className="surface-card mt-2 min-w-0 p-3.5">{children}</div>
    </li>
  )
}

import cx from '../../lib/cx'
import { formatClock } from '../../lib/time'

export default function TimelineItem({ start, end, tone = 'brand', className, children }) {
  return (
    <li
      className={cx(
        'grid grid-cols-[3.75rem_0.75rem_1fr] gap-x-2.5 border-b border-dashed border-line py-3 last:border-b-0 sm:grid-cols-[4.75rem_0.875rem_1fr] sm:gap-x-3',
        className,
      )}
    >
      <span className="font-numeral pt-0.5 text-right text-xs leading-tight">
        <span className="block font-semibold text-ink">{formatClock(start)}</span>
        {end && <span className="block text-faint">{formatClock(end)}</span>}
      </span>
      <span aria-hidden="true" className="relative flex justify-center pt-1.5">
        <span
          className={cx(
            'size-2.5 shrink-0 rounded-full border-[3px] bg-surface',
            tone === 'muted' ? 'border-line-strong' : 'border-brand-600',
          )}
        />
        <span className="absolute top-5 bottom-[-0.75rem] w-px bg-line" />
      </span>
      <div className="min-w-0">{children}</div>
    </li>
  )
}

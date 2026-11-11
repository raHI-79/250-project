import cx from '../lib/cx'
import { dayOfMonth, isoDate, monthName, weekdayName } from '../lib/time'

export default function DateBlock({ value, className }) {
  const stamp = isoDate(value)
  if (!stamp) return null
  return (
    <div
      className={cx(
        'w-14 shrink-0 overflow-hidden rounded-xl border border-line bg-surface text-center',
        className,
      )}
    >
      <p className="bg-brand-600 py-0.5 text-2xs font-bold tracking-[0.08em] text-page uppercase">
        {monthName(stamp)}
      </p>
      <p className="font-numeral pt-1 text-2xl leading-none font-bold text-ink">
        {dayOfMonth(stamp)}
      </p>
      <p className="pb-1.5 text-2xs font-semibold tracking-[0.06em] text-muted uppercase">
        {weekdayName(stamp).slice(0, 3)}
      </p>
    </div>
  )
}

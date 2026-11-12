import cx from '../../lib/cx'
import { BADGE_MAX, badgeLabel } from '../../lib/notificationState'

export default function CountBadge({ count, max = BADGE_MAX, className }) {
  const value = Number(count) || 0
  if (value <= 0) return null
  return (
    <span
      aria-label={`${value} unread`}
      className={cx(
        'inline-flex min-w-5 items-center justify-center rounded-full bg-danger-600 px-1.5 text-2xs font-bold text-page',
        className,
      )}
    >
      {badgeLabel(value, max)}
    </span>
  )
}

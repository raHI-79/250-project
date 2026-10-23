import cx from '../../lib/cx'

export default function CountBadge({ count, max = 9, className }) {
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
      {value > max ? `${max}+` : value}
    </span>
  )
}

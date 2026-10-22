import cx from '../../lib/cx'
import { useAuth } from '../../context/auth'

const greeting = (hour) => {
  if (hour < 12) return 'Good morning'
  if (hour < 17) return 'Good afternoon'
  return 'Good evening'
}

const dateLine = (date) =>
  date
    .toLocaleDateString('en-GB', {
      weekday: 'long',
      day: 'numeric',
      month: 'long',
      year: 'numeric',
    })
    .toUpperCase()

export default function Masthead({ subtitle, actions, className }) {
  const { user } = useAuth()
  const now = new Date()
  const first = String(user?.name || user?.identifier || '')
    .trim()
    .split(/\s+/)[0]
  return (
    <div
      className={cx(
        'mb-6 flex flex-col gap-3 border-b border-ink pb-3 sm:flex-row sm:items-end sm:justify-between',
        className,
      )}
    >
      <div className="min-w-0">
        <p className="text-xs font-semibold tracking-[0.1em] text-muted">{dateLine(now)}</p>
        <h1 className="font-display mt-1.5 text-[30px] leading-tight font-bold text-ink sm:text-[32px]">
          {greeting(now.getHours())}
          {first ? (
            <>
              , <em className="text-brand-600 not-italic">{first}</em>
            </>
          ) : null}
        </h1>
        {subtitle && <p className="mt-1.5 text-sm text-muted">{subtitle}</p>}
      </div>
      {actions && <div className="flex flex-wrap gap-2">{actions}</div>}
    </div>
  )
}

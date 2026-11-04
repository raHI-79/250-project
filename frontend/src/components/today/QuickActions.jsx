import { Link } from 'react-router-dom'
import CountBadge from '../ui/CountBadge'
import cx from '../../lib/cx'

export default function QuickActions({ actions, className }) {
  return (
    <nav aria-label="Quick actions" className={cx('grid grid-cols-4 gap-2.5', className)}>
      {actions.map((action) => {
        const Icon = action.icon
        return (
          <Link
            key={action.label}
            to={action.to}
            className="surface-card press relative flex min-h-24 flex-col items-center justify-center gap-2 p-2 text-center"
          >
            <span
              aria-hidden="true"
              className={cx(
                'relative grid size-11 place-items-center rounded-full',
                action.tone || 'bg-tint-blue text-tint-blue-ink',
              )}
            >
              <Icon size={21} strokeWidth={1.8} />
            </span>
            <span className="wrap-long text-2xs leading-tight font-bold text-ink-soft">
              {action.label}
            </span>
            {action.count > 0 && (
              <CountBadge count={action.count} className="absolute top-1.5 right-1.5" />
            )}
          </Link>
        )
      })}
    </nav>
  )
}

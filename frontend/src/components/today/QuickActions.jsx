import { Link } from 'react-router-dom'
import CountBadge from '../ui/CountBadge'
import cx from '../../lib/cx'

export default function QuickActions({ actions, className }) {
  return (
    <nav aria-label="Quick actions" className={cx('grid grid-cols-4 gap-y-4', className)}>
      {actions.map((action) => {
        const Icon = action.icon
        return (
          <Link
            key={action.label}
            to={action.to}
            className="flex min-h-12 flex-col items-center gap-1.5 text-center"
          >
            <span className="relative grid size-14 place-items-center rounded-full bg-brand-50 text-brand-600 transition-colors active:bg-brand-100">
              <Icon size={22} strokeWidth={1.9} />
              {action.count > 0 && (
                <CountBadge count={action.count} className="absolute -top-0.5 -right-0.5" />
              )}
            </span>
            <span className="text-2xs leading-tight font-medium text-ink-soft">{action.label}</span>
          </Link>
        )
      })}
    </nav>
  )
}

import { NavLink } from 'react-router-dom'
import CountBadge from '../ui/CountBadge'
import cx from '../../lib/cx'

export default function TabBar({ tabs }) {
  if (tabs.length === 0) return null
  return (
    <nav
      aria-label="Main"
      className="glass fixed inset-x-0 bottom-0 z-50 border-t border-line pb-[env(safe-area-inset-bottom,0px)] lg:hidden"
    >
      <ul className="flex items-stretch">
        {tabs.map((tab) => {
          const Icon = tab.icon
          return (
            <li key={tab.label} className="flex-1">
              <NavLink
                to={tab.to()}
                end={tab.end}
                className={({ isActive }) =>
                  cx(
                    'flex min-h-14 flex-col items-center justify-center gap-0.5 pt-1.5 pb-1 transition-colors',
                    isActive ? 'text-brand-600' : 'text-muted',
                  )
                }
              >
                {({ isActive }) => (
                  <>
                    <span className="relative">
                      <Icon
                        size={23}
                        strokeWidth={isActive ? 2.2 : 1.8}
                        fill={isActive ? 'currentColor' : 'none'}
                        fillOpacity={isActive ? 0.2 : 0}
                      />
                      {tab.count > 0 && (
                        <CountBadge count={tab.count} className="absolute -top-1 -right-2.5" />
                      )}
                    </span>
                    <span
                      className={cx('text-2xs leading-none', isActive ? 'font-bold' : 'font-medium')}
                    >
                      {tab.label}
                    </span>
                  </>
                )}
              </NavLink>
            </li>
          )
        })}
      </ul>
    </nav>
  )
}

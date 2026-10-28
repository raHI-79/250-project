import { NavLink } from 'react-router-dom'
import CountBadge from '../ui/CountBadge'
import cx from '../../lib/cx'

export default function TabBar({ tabs }) {
  if (tabs.length === 0) return null
  return (
    <nav
      aria-label="Main"
      className="glass fixed inset-x-0 bottom-0 z-50 border-t pb-[env(safe-area-inset-bottom,0px)] lg:hidden"
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
                    'press relative flex min-h-14 flex-col items-center justify-center gap-1 pt-2 pb-1',
                    isActive ? 'text-brand-600' : 'text-muted',
                  )
                }
              >
                {({ isActive }) => (
                  <>
                    <span
                      aria-hidden="true"
                      className={cx(
                        'absolute inset-x-5 top-0 h-0.5 rounded-full bg-brand-600 transition-all duration-300 motion-reduce:transition-none',
                        isActive ? 'opacity-100' : 'scale-x-0 opacity-0',
                      )}
                    />
                    <span className="relative">
                      <span
                        aria-hidden="true"
                        className={cx(
                          'absolute -inset-x-3 -inset-y-1.5 rounded-full bg-brand-50 transition-all duration-300 motion-reduce:transition-none',
                          isActive ? 'scale-100 opacity-100' : 'scale-75 opacity-0',
                        )}
                      />
                      <Icon
                        size={22}
                        className="relative"
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

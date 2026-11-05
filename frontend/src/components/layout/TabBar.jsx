import { NavLink, useLocation } from 'react-router-dom'
import CountBadge from '../ui/CountBadge'
import cx from '../../lib/cx'
import { activeNavPath, isNavActive } from '../../lib/nav'

export default function TabBar({ tabs }) {
  const { pathname } = useLocation()
  const activePath = activeNavPath([{ items: tabs }], pathname)
  if (tabs.length === 0) return null
  return (
    <nav
      aria-label="Main"
      className="fixed inset-x-0 bottom-0 z-50 border-t border-line bg-surface pb-[env(safe-area-inset-bottom,0px)] lg:hidden"
    >
      <ul className="flex items-stretch">
        {tabs.map((tab) => {
          const Icon = tab.icon
          const active = isNavActive(tab, activePath)
          return (
            <li key={tab.label} className="flex-1">
              <NavLink
                to={tab.to()}
                aria-current={active ? 'page' : undefined}
                className={cx(
                  'press relative flex min-h-14 flex-col items-center justify-center gap-1 pt-2.5 pb-1',
                  active ? 'text-brand-600' : 'text-muted',
                )}
              >
                <span
                  aria-hidden="true"
                  className={cx(
                    'absolute inset-x-3 top-0 h-[3px] rounded-b-full bg-brand-600 transition-opacity duration-200 motion-reduce:transition-none',
                    active ? 'opacity-100' : 'opacity-0',
                  )}
                />
                <span className="relative">
                  <Icon size={22} strokeWidth={active ? 2.4 : 1.8} />
                  {tab.count > 0 && (
                    <CountBadge count={tab.count} className="absolute -top-1 -right-2.5" />
                  )}
                </span>
                <span
                  className={cx('text-2xs leading-none', active ? 'font-extrabold' : 'font-semibold')}
                >
                  {tab.label}
                </span>
              </NavLink>
            </li>
          )
        })}
      </ul>
    </nav>
  )
}

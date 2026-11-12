import { Link } from 'react-router-dom'
import { Menu, Search } from 'lucide-react'
import NotificationBell from './NotificationBell'
import ConfirmedNote from './ConfirmedNote'
import LogoBadge from '../ui/LogoBadge'
import cx from '../../lib/cx'
import { useAuth } from '../../context/auth'
import { DEPARTMENT } from '../ui/IdentityBar'
import { roleLabel } from '../../lib/nav'
import { homeFor } from '../../lib/paths'

export default function TopBar({ title, collapsed, onOpenMenu, onOpenSearch, showBell = true }) {
  const { role } = useAuth()
  return (
    <header className="sticky top-0 z-50 bg-header pt-[env(safe-area-inset-top,0px)]">
      <div className="mx-auto flex min-h-14 max-w-6xl items-center gap-2 px-4 sm:px-6 lg:px-8">
        {onOpenMenu && (
          <button
            type="button"
            onClick={onOpenMenu}
            aria-label="Open menu"
            className="press tap-target -ml-3 grid shrink-0 place-items-center rounded-full text-white hover:bg-white/15 lg:hidden"
          >
            <Menu size={21} />
          </button>
        )}
        <Link
          to={homeFor(role)}
          aria-label="Home"
          className={cx('shrink-0 lg:hidden', collapsed && title ? 'hidden' : 'block')}
        >
          <LogoBadge size="sm" />
        </Link>
        <p
          aria-hidden={!collapsed}
          className={cx(
            'font-display min-w-0 flex-1 truncate text-base text-white transition-opacity duration-200 lg:hidden',
            collapsed && title ? 'opacity-100' : 'opacity-0',
          )}
        >
          {title}
        </p>
        <p className="hidden min-w-0 flex-1 truncate text-xs text-white/75 lg:block">
          {DEPARTMENT}
          <span className="px-2 text-white/40">·</span>
          <span className="font-bold text-white">{roleLabel(role)}</span>
        </p>
        <div className="ml-auto flex shrink-0 items-center gap-1.5">
          <ConfirmedNote />
          {onOpenSearch && (
            <button
              type="button"
              onClick={onOpenSearch}
              aria-label="Search"
              className="press tap-target grid place-items-center rounded-full text-white hover:bg-white/15"
            >
              <Search size={20} />
            </button>
          )}
          {showBell && <NotificationBell />}
        </div>
      </div>
    </header>
  )
}

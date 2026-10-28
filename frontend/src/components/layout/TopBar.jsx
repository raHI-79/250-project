import { Link } from 'react-router-dom'
import { Menu, Search } from 'lucide-react'
import NotificationBell from './NotificationBell'
import LogoBadge from '../ui/LogoBadge'
import cx from '../../lib/cx'
import { useAuth } from '../../context/auth'
import { DEPARTMENT } from '../ui/IdentityBar'
import { roleLabel } from '../../lib/nav'
import { homeFor } from '../../lib/paths'

export default function TopBar({ title, collapsed, onOpenMenu, onOpenSearch, showBell = true }) {
  const { role } = useAuth()
  return (
    <header className="glass sticky top-0 z-50 border-b pt-[env(safe-area-inset-top,0px)]">
      <div className="mx-auto flex min-h-14 max-w-6xl items-center gap-2 px-4 sm:px-6 lg:px-8">
        {onOpenMenu && (
          <button
            type="button"
            onClick={onOpenMenu}
            aria-label="Open menu"
            className="press tap-target -ml-3 grid shrink-0 place-items-center rounded-full text-ink-soft hover:bg-wash lg:hidden"
          >
            <Menu size={21} />
          </button>
        )}
        <Link
          to={homeFor(role)}
          aria-label="Home"
          className={cx(
            'shrink-0 transition-opacity lg:hidden',
            collapsed && title ? 'opacity-0' : 'opacity-100',
          )}
        >
          <LogoBadge size="sm" />
        </Link>
        <p
          aria-hidden={!collapsed}
          className={cx(
            'font-display min-w-0 flex-1 truncate text-base font-semibold text-ink transition-opacity duration-200 lg:hidden',
            collapsed && title ? 'opacity-100' : 'opacity-0',
          )}
        >
          {title}
        </p>
        <p className="hidden min-w-0 flex-1 truncate text-xs text-muted lg:block">
          {DEPARTMENT}
          <span className="px-2 text-line-strong">·</span>
          <span className="font-semibold text-ink-soft">{roleLabel(role)}</span>
        </p>
        <div className="ml-auto flex shrink-0 items-center gap-0.5">
          {onOpenSearch && (
            <button
              type="button"
              onClick={onOpenSearch}
              aria-label="Search"
              className="press tap-target grid place-items-center rounded-full text-ink-soft hover:bg-wash"
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

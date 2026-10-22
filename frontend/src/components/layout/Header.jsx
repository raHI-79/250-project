import { Link } from 'react-router-dom'
import { Menu } from 'lucide-react'
import NotificationBell from './NotificationBell'
import Crest from '../ui/Crest'
import { useAuth } from '../../context/auth'
import { DEPARTMENT } from '../ui/IdentityBar'
import { roleLabel } from '../../lib/nav'
import { homeFor } from '../../lib/paths'

export default function Header({ onOpenMenu, showBell = true }) {
  const { role } = useAuth()
  return (
    <header className="sticky top-0 z-40 flex items-center gap-3 border-b border-line bg-surface px-4 py-2.5 sm:px-6">
      <button
        type="button"
        onClick={onOpenMenu}
        aria-label="Open menu"
        className="tap-target -ml-2 grid place-items-center text-ink-soft transition-colors hover:bg-wash lg:hidden"
      >
        <Menu size={20} />
      </button>
      <Link to={homeFor(role)} className="flex min-w-0 items-center gap-2.5 lg:hidden">
        <Crest size="sm" />
        <span className="font-display truncate text-base font-semibold text-ink">SUST CSE</span>
      </Link>
      <p className="hidden min-w-0 truncate text-xs text-muted lg:block">
        {DEPARTMENT}
        <span className="px-2 text-line-strong">·</span>
        <span className="font-semibold text-ink-soft">{roleLabel(role)}</span>
      </p>
      <div className="ml-auto flex items-center gap-1">{showBell && <NotificationBell />}</div>
    </header>
  )
}

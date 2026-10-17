import { Link } from 'react-router-dom'
import { LogOut, Menu } from 'lucide-react'
import Button from '../ui/Button'
import NotificationBell from './NotificationBell'
import { useAuth } from '../../context/auth'
import { roleLabel } from '../../lib/nav'
import { homeFor } from '../../lib/paths'

export default function Header({ onOpenMenu, showBell = true }) {
  const { user, role, logout } = useAuth()
  return (
    <header className="sticky top-0 z-40 flex items-center gap-3 border-b border-brand-100 bg-surface px-4 py-3 sm:px-6">
      <button
        type="button"
        onClick={onOpenMenu}
        aria-label="Open menu"
        className="rounded-lg p-2 text-brand-700 transition-colors hover:bg-brand-50 active:scale-95 lg:hidden"
      >
        <Menu size={20} />
      </button>
      <Link to={homeFor(role)} className="font-display text-xl font-semibold text-brand-700">
        SUST CSE
      </Link>
      <div className="ml-auto flex items-center gap-2 sm:gap-3">
        <div className="hidden text-right leading-tight sm:block">
          <p className="text-sm font-semibold text-brand-900">{user?.name || user?.identifier}</p>
          <p className="text-xs text-brand-500">{roleLabel(role)}</p>
        </div>
        {showBell && <NotificationBell />}
        <Button variant="secondary" size="sm" onClick={logout}>
          <LogOut size={15} />
          <span className="hidden sm:inline">Log out</span>
        </Button>
      </div>
    </header>
  )
}

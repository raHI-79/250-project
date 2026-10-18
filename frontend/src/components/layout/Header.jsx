import { useEffect, useRef, useState } from 'react'
import { Link } from 'react-router-dom'
import { ChevronDown, LogOut, Menu, UserCircle } from 'lucide-react'
import Avatar from '../ui/Avatar'
import NotificationBell from './NotificationBell'
import { useAuth } from '../../context/auth'
import cx from '../../lib/cx'
import { roleLabel } from '../../lib/nav'
import { homeFor, paths } from '../../lib/paths'

export default function Header({ onOpenMenu, showBell = true }) {
  const { user, role, logout } = useAuth()
  const [open, setOpen] = useState(false)
  const holder = useRef(null)
  useEffect(() => {
    if (!open) return undefined
    const onPointerDown = (event) => {
      if (!holder.current?.contains(event.target)) setOpen(false)
    }
    const onKeyDown = (event) => {
      if (event.key === 'Escape') setOpen(false)
    }
    window.addEventListener('pointerdown', onPointerDown)
    window.addEventListener('keydown', onKeyDown)
    return () => {
      window.removeEventListener('pointerdown', onPointerDown)
      window.removeEventListener('keydown', onKeyDown)
    }
  }, [open])
  const name = user?.name || user?.identifier || ''
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
        {showBell && <NotificationBell />}
        <div ref={holder} className="relative">
          <button
            type="button"
            onClick={() => setOpen((current) => !current)}
            aria-haspopup="menu"
            aria-expanded={open}
            className="flex items-center gap-2 rounded-full border border-brand-100 py-1 pr-2 pl-1 transition-colors hover:bg-brand-50"
          >
            <Avatar src={user?.photo_url} name={name} size="sm" />
            <span className="hidden text-right leading-tight sm:block">
              <span className="block max-w-32 truncate text-sm font-semibold text-brand-900">
                {name}
              </span>
              <span className="block text-xs text-brand-500">{roleLabel(role)}</span>
            </span>
            <ChevronDown
              size={15}
              className={cx('text-brand-400 transition-transform', open && 'rotate-180')}
            />
          </button>
          {open && (
            <div
              role="menu"
              className="animate-fade-in absolute right-0 z-50 mt-2 w-56 overflow-hidden rounded-xl border border-brand-100 bg-surface shadow-xl"
            >
              <div className="border-b border-brand-100 px-4 py-3">
                <p className="truncate text-sm font-semibold text-brand-900">{name}</p>
                <p className="truncate text-xs text-brand-500">{user?.identifier}</p>
              </div>
              <Link
                to={paths.profile()}
                onClick={() => setOpen(false)}
                className="flex items-center gap-2.5 px-4 py-2.5 text-sm text-brand-700 transition-colors hover:bg-brand-50"
              >
                <UserCircle size={16} className="text-brand-500" />
                My Profile
              </Link>
              <button
                type="button"
                onClick={() => {
                  setOpen(false)
                  logout()
                }}
                className="flex w-full items-center gap-2.5 border-t border-brand-100 px-4 py-2.5 text-left text-sm text-accent-700 transition-colors hover:bg-accent-50"
              >
                <LogOut size={16} />
                Log out
              </button>
            </div>
          )}
        </div>
      </div>
    </header>
  )
}

import { Link, NavLink } from 'react-router-dom'
import { X } from 'lucide-react'
import ProfileNavLink from './ProfileNavLink'
import Crest from '../ui/Crest'
import CountBadge from '../ui/CountBadge'
import cx from '../../lib/cx'
import { homeFor } from '../../lib/paths'
import { useAuth } from '../../context/auth'

export default function Sidebar({ groups, open, onClose }) {
  const { role } = useAuth()
  const masthead = (
    <Link
      to={homeFor(role)}
      onClick={onClose}
      className="flex items-center gap-3 border-b border-line px-5 py-4"
    >
      <Crest size="sm" />
      <span className="min-w-0">
        <span className="font-display block text-base font-semibold text-ink">SUST CSE</span>
        <span className="block truncate text-2xs tracking-[0.08em] text-muted uppercase">
          Class and exam coordination
        </span>
      </span>
    </Link>
  )
  const links = (
    <nav className="flex flex-1 flex-col overflow-y-auto px-2 py-3">
      {groups.map((group, index) => (
        <div key={group.label || index} className={cx(index > 0 && 'mt-5')}>
          {group.label && (
            <p className="px-3 pb-2 text-2xs font-bold tracking-[0.1em] text-faint uppercase">
              {group.label}
            </p>
          )}
          {group.items.map((item) => (
            <NavLink
              key={item.to()}
              to={item.to()}
              onClick={onClose}
              className={({ isActive }) =>
                cx(
                  'flex min-h-12 items-center gap-3 rounded-xl px-3 text-sm transition-colors',
                  isActive
                    ? 'bg-brand-50 font-bold text-brand-700'
                    : 'font-medium text-ink-soft hover:bg-wash hover:text-ink',
                )
              }
            >
              {({ isActive }) => (
                <>
                  <item.icon
                    size={18}
                    className="shrink-0"
                    strokeWidth={isActive ? 2.2 : 1.8}
                    fill={isActive ? 'currentColor' : 'none'}
                    fillOpacity={isActive ? 0.18 : 0}
                  />
                  <span className="truncate">{item.label}</span>
                  <CountBadge count={item.count} className="ml-auto" />
                </>
              )}
            </NavLink>
          ))}
        </div>
      ))}
    </nav>
  )
  const content = (
    <div className="flex h-full min-h-0 flex-col">
      {masthead}
      {links}
      <div className="border-t border-line">
        <ProfileNavLink onNavigate={onClose} />
      </div>
    </div>
  )
  return (
    <>
      <aside className="hidden w-68 shrink-0 border-r border-line bg-surface lg:block">
        {content}
      </aside>
      {open && (
        <div className="fixed inset-0 z-60 lg:hidden">
          <button
            type="button"
            aria-label="Close menu"
            onClick={onClose}
            className="animate-fade-in absolute inset-0 cursor-default bg-black/55"
          />
          <div className="animate-slide-in-right absolute inset-y-0 left-0 flex w-72 max-w-[85vw] flex-col bg-surface pt-[env(safe-area-inset-top,0px)]">
            <button
              type="button"
              onClick={onClose}
              aria-label="Close menu"
              className="tap-target absolute top-[calc(0.5rem+env(safe-area-inset-top,0px))] right-1 z-10 grid place-items-center rounded-full text-muted transition-colors hover:bg-wash hover:text-ink"
            >
              <X size={18} />
            </button>
            <div className="min-h-0 flex-1">{content}</div>
          </div>
        </div>
      )}
    </>
  )
}

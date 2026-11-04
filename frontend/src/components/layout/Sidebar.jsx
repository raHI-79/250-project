import { useEffect } from 'react'
import { Link, NavLink } from 'react-router-dom'
import { X } from 'lucide-react'
import ProfileNavLink from './ProfileNavLink'
import CountBadge from '../ui/CountBadge'
import IconButton from '../ui/IconButton'
import LogoBadge from '../ui/LogoBadge'
import cx from '../../lib/cx'
import { homeFor } from '../../lib/paths'
import { lockScroll } from '../../lib/scrollLock'
import { useAuth } from '../../context/auth'

export default function Sidebar({ groups, open, onClose }) {
  const { role } = useAuth()
  useEffect(() => {
    if (!open) return undefined
    const release = lockScroll()
    return release
  }, [open])
  const masthead = (
    <Link to={homeFor(role)} onClick={onClose} className="flex items-center gap-3 px-4 py-4">
      <LogoBadge size="md" />
      <span className="min-w-0">
        <span className="font-display block text-base font-semibold text-ink">SUST CSE</span>
        <span className="block text-2xs leading-tight tracking-[0.08em] text-muted uppercase">
          Class and exam coordination
        </span>
      </span>
    </Link>
  )
  const links = (
    <nav className="flex min-h-0 flex-1 flex-col gap-5 overflow-x-hidden overflow-y-auto px-3 py-2">
      {groups.map((group, index) => (
        <div key={group.label || index}>
          {group.label && (
            <p className="px-3 pb-2 text-2xs font-bold tracking-[0.12em] text-faint uppercase">
              {group.label}
            </p>
          )}
          <div className="flex flex-col gap-0.5">
            {group.items.map((item) => (
              <NavLink
                key={item.to()}
                to={item.to()}
                onClick={onClose}
                className={({ isActive }) =>
                  cx(
                    'press relative flex min-h-12 items-center gap-3 rounded-xl px-2.5 py-1.5 text-sm',
                    isActive
                      ? 'bg-brand-50 font-semibold text-brand-700 ring-1 ring-brand-200'
                      : 'font-medium text-ink-soft hover:bg-wash',
                  )
                }
              >
                {({ isActive }) => (
                  <>
                    <span
                      className={cx(
                        'grid size-9 shrink-0 place-items-center rounded-[0.7rem] transition-colors',
                        isActive ? 'bg-brand-600 text-white' : 'bg-wash text-muted',
                      )}
                    >
                      <item.icon size={17} strokeWidth={isActive ? 2.2 : 1.8} />
                    </span>
                    <span className="min-w-0 flex-1 wrap-long">{item.label}</span>
                    <CountBadge count={item.count} className="shrink-0" />
                  </>
                )}
              </NavLink>
            ))}
          </div>
        </div>
      ))}
    </nav>
  )
  const content = (
    <div className="flex h-full min-h-0 w-full flex-col">
      {masthead}
      {links}
      <div className="p-3">
        <div className="rounded-2xl border border-line bg-wash">
          <ProfileNavLink onNavigate={onClose} />
        </div>
      </div>
    </div>
  )
  return (
    <>
      <aside className="fixed inset-y-0 left-0 z-40 hidden w-sidebar border-r border-line bg-surface lg:block">
        {content}
      </aside>
      {open && (
        <div className="fixed inset-0 z-60 overflow-hidden lg:hidden">
          <button
            type="button"
            aria-label="Close menu"
            onClick={onClose}
            className="animate-fade-in absolute inset-0 cursor-default bg-scrim"
          />
          <div className="animate-slide-in-left absolute inset-y-0 left-0 flex w-sidebar max-w-[88vw] flex-col rounded-r-sheet border-r border-line bg-surface shadow-[var(--shadow-lift)]">
            <IconButton
              label="Close menu"
              onClick={onClose}
              size="sm"
              className="absolute top-3 right-2 z-10"
            >
              <X size={18} />
            </IconButton>
            <div className="min-h-0 flex-1">{content}</div>
          </div>
        </div>
      )}
    </>
  )
}

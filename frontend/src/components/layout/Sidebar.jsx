import { Link, NavLink } from 'react-router-dom'
import { X } from 'lucide-react'
import ProfileNavLink from './ProfileNavLink'
import CountBadge from '../ui/CountBadge'
import IconButton from '../ui/IconButton'
import LogoBadge from '../ui/LogoBadge'
import cx from '../../lib/cx'
import { homeFor } from '../../lib/paths'
import { useAuth } from '../../context/auth'

export default function Sidebar({ groups, open, onClose }) {
  const { role } = useAuth()
  const masthead = (
    <Link
      to={homeFor(role)}
      onClick={onClose}
      className="flex items-center gap-3 px-4 py-4"
    >
      <LogoBadge size="md" />
      <span className="min-w-0">
        <span className="font-display block text-base font-semibold text-ink">SUST CSE</span>
        <span className="block truncate text-2xs tracking-[0.08em] text-muted uppercase">
          Class and exam coordination
        </span>
      </span>
    </Link>
  )
  const links = (
    <nav className="flex flex-1 flex-col gap-5 overflow-y-auto px-3 py-2">
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
                    'press relative flex min-h-12 items-center gap-3 rounded-xl px-2.5 text-sm',
                    isActive ? 'font-semibold text-brand-700' : 'font-medium text-ink-soft',
                  )
                }
              >
                {({ isActive }) => (
                  <>
                    {isActive && (
                      <span
                        aria-hidden="true"
                        className="animate-fade-in absolute inset-0 -z-10 rounded-xl bg-brand-50 ring-1 ring-brand-200"
                      />
                    )}
                    <span
                      className={cx(
                        'grid size-9 shrink-0 place-items-center rounded-[0.7rem] transition-colors',
                        isActive ? 'bg-brand-600 text-page' : 'bg-wash text-muted',
                      )}
                    >
                      <item.icon size={17} strokeWidth={isActive ? 2.2 : 1.8} />
                    </span>
                    <span className="truncate">{item.label}</span>
                    <CountBadge count={item.count} className="ml-auto" />
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
    <div className="flex h-full min-h-0 flex-col">
      {masthead}
      {links}
      <div className="p-3">
        <div className="glass-chip rounded-2xl">
          <ProfileNavLink onNavigate={onClose} />
        </div>
      </div>
    </div>
  )
  return (
    <>
      <aside className="glass hidden w-68 shrink-0 border-r lg:block">{content}</aside>
      {open && (
        <div className="fixed inset-0 z-60 lg:hidden">
          <button
            type="button"
            aria-label="Close menu"
            onClick={onClose}
            className="animate-fade-in absolute inset-0 cursor-default bg-black/55"
          />
          <div className="glass-sheet animate-slide-in-right absolute inset-y-0 left-0 flex w-72 max-w-[85vw] flex-col rounded-r-sheet pt-[env(safe-area-inset-top,0px)]">
            <IconButton
              label="Close menu"
              onClick={onClose}
              size="sm"
              className="absolute top-[calc(0.75rem+env(safe-area-inset-top,0px))] right-2 z-10"
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

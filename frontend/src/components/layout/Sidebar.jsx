import { Link, NavLink } from 'react-router-dom'
import { X } from 'lucide-react'
import ProfileNavLink from './ProfileNavLink'
import Crest from '../ui/Crest'
import cx from '../../lib/cx'
import { homeFor } from '../../lib/paths'
import { useAuth } from '../../context/auth'

const TONES = {
  brand: { panel: 'bg-brand-900', edge: 'border-brand-950', hover: 'hover:bg-white/8' },
  ink: { panel: 'bg-ink', edge: 'border-black/40', hover: 'hover:bg-white/8' },
}

export default function Sidebar({ groups, tone = 'brand', open, onClose }) {
  const { role } = useAuth()
  const skin = TONES[tone] || TONES.brand
  const masthead = (
    <Link
      to={homeFor(role)}
      onClick={onClose}
      className={cx('flex items-center gap-3 border-b px-4 py-4', skin.edge)}
    >
      <Crest tone="light" size="sm" />
      <span className="min-w-0">
        <span className="font-display block text-base font-semibold text-white">SUST CSE</span>
        <span className="block truncate text-xs tracking-[0.04em] text-white/70 uppercase">
          Department record
        </span>
      </span>
    </Link>
  )
  const links = (
    <nav className="flex flex-1 flex-col overflow-y-auto py-3">
      {groups.map((group, index) => (
        <div key={group.label || index} className={cx(index > 0 && 'mt-5')}>
          {group.label && (
            <p className="px-4 pb-2 text-xs font-semibold tracking-[0.1em] text-white/60 uppercase">
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
                  'tap-target flex items-center gap-3 border-l-[3px] px-4 py-3 text-sm transition-colors',
                  isActive
                    ? 'border-gold-400 bg-white/15 font-bold text-white'
                    : cx('border-transparent font-medium text-white/85 hover:text-white', skin.hover),
                )
              }
            >
              <item.icon size={16} className="shrink-0" />
              <span className="truncate">{item.label}</span>
              {item.count > 0 && (
                <span className="ml-auto inline-flex min-w-5 items-center justify-center rounded-[3px] bg-accent-600 px-1.5 text-xs font-semibold text-white">
                  {item.count > 9 ? '9+' : item.count}
                </span>
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
      <div className={cx('border-t', skin.edge)}>
        <ProfileNavLink onNavigate={onClose} />
      </div>
    </div>
  )
  return (
    <>
      <aside className={cx('hidden w-64 shrink-0 border-r lg:block', skin.panel, skin.edge)}>
        {content}
      </aside>
      {open && (
        <div className="fixed inset-0 z-60 lg:hidden">
          <button
            type="button"
            aria-label="Close menu"
            onClick={onClose}
            className="animate-fade-in absolute inset-0 cursor-default bg-ink/55"
          />
          <div
            className={cx(
              'animate-slide-in-right absolute inset-y-0 left-0 flex w-72 max-w-[85vw] flex-col',
              skin.panel,
            )}
          >
            <button
              type="button"
              onClick={onClose}
              aria-label="Close menu"
              className="tap-target absolute top-3 right-2 z-10 grid place-items-center text-white/80 transition-colors hover:text-white"
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

import { NavLink } from 'react-router-dom'
import { X } from 'lucide-react'
import ProfileNavLink from './ProfileNavLink'
import cx from '../../lib/cx'

export default function Sidebar({ groups, open, onClose }) {
  const links = (
    <nav className="flex flex-1 flex-col gap-1 overflow-y-auto px-3 py-4">
      {groups.map((group, index) => (
        <div key={group.label || index} className={cx('flex flex-col gap-1', index > 0 && 'mt-4')}>
          {group.label && (
            <p className="px-3 pb-1 text-[11px] font-semibold tracking-wider text-brand-200/70 uppercase">
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
                  'group flex items-center gap-3 rounded-lg px-3 py-2.5 text-sm font-medium transition-all duration-200',
                  isActive
                    ? 'bg-sidebar-active text-white shadow-sm'
                    : 'text-brand-100/80 hover:translate-x-1 hover:bg-sidebar-hover hover:text-white',
                )
              }
            >
              <item.icon
                size={17}
                className="shrink-0 transition-transform group-hover:scale-110"
              />
              <span className="truncate">{item.label}</span>
              {item.count > 0 && (
                <span className="ml-auto inline-flex min-w-5 items-center justify-center rounded-full bg-accent-600 px-1.5 text-[11px] font-semibold text-white">
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
      {links}
      <div className="border-t border-white/10 pt-3">
        <ProfileNavLink onNavigate={onClose} />
      </div>
    </div>
  )
  return (
    <>
      <aside className="hidden w-64 shrink-0 bg-sidebar lg:block">{content}</aside>
      {open && (
        <div className="fixed inset-0 z-[60] lg:hidden">
          <button
            type="button"
            aria-label="Close menu"
            onClick={onClose}
            className="animate-fade-in absolute inset-0 cursor-default bg-brand-950/50"
          />
          <div className="animate-slide-in-right absolute inset-y-0 left-0 flex w-72 max-w-[85vw] flex-col bg-sidebar shadow-2xl">
            <div className="flex items-center justify-between px-4 py-4">
              <span className="font-display text-lg font-semibold text-white">SUST CSE</span>
              <button
                type="button"
                onClick={onClose}
                aria-label="Close menu"
                className="rounded p-1 text-brand-100 transition-colors hover:bg-sidebar-hover hover:text-white"
              >
                <X size={18} />
              </button>
            </div>
            <div className="min-h-0 flex-1">{content}</div>
          </div>
        </div>
      )}
    </>
  )
}

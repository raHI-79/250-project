import { NavLink } from 'react-router-dom'
import { X } from 'lucide-react'
import ProfileNavLink from './ProfileNavLink'
import cx from '../../lib/cx'

export default function Sidebar({ items, open, onClose }) {
  const links = (
    <nav className="flex flex-1 flex-col gap-1 overflow-y-auto px-3 py-4">
      {items.map((item) => (
        <NavLink
          key={item.label}
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
          <item.icon size={17} className="shrink-0 transition-transform group-hover:scale-110" />
          <span className="truncate">{item.label}</span>
        </NavLink>
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

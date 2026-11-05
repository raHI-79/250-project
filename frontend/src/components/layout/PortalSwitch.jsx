import { NavLink } from 'react-router-dom'
import cx from '../../lib/cx'
import paths from '../../lib/paths'

const PORTALS = [
  { label: 'Student', to: paths.studentLogin() },
  { label: 'Teacher', to: paths.teacherLogin() },
]

export function PortalSwitch({ className }) {
  return (
    <div
      role="tablist"
      aria-label="Portal"
      className={cx('glass-chip flex gap-1 rounded-full p-1', className)}
    >
      {PORTALS.map((portal) => (
        <NavLink
          key={portal.label}
          to={portal.to}
          role="tab"
          className={({ isActive }) =>
            cx(
              'min-h-10 flex-1 rounded-full px-3 text-center text-sm leading-10 whitespace-nowrap transition-colors',
              isActive
                ? 'bg-surface font-semibold text-brand-700 shadow-[var(--shadow-glass)]'
                : 'font-medium text-muted hover:text-ink',
            )
          }
        >
          {portal.label}
        </NavLink>
      ))}
    </div>
  )
}

export function AppPortalSwitch({ className }) {
  return (
    <div
      role="tablist"
      aria-label="Portal"
      className={cx('flex h-[2.625rem] gap-1 rounded-input bg-wash p-1 ring-1 ring-line', className)}
    >
      {PORTALS.map((portal) => (
        <NavLink
          key={portal.label}
          to={portal.to}
          role="tab"
          replace
          className={({ isActive }) =>
            cx(
              'grid flex-1 place-items-center rounded-[0.625rem] px-3 text-sm whitespace-nowrap transition-colors',
              isActive ? 'bg-brand-600 font-bold text-white' : 'font-semibold text-muted',
            )
          }
        >
          {portal.label}
        </NavLink>
      ))}
    </div>
  )
}

export default PortalSwitch

import { NavLink } from 'react-router-dom'
import cx from '../../lib/cx'
import paths from '../../lib/paths'
import LogoBadge from '../ui/LogoBadge'

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

export default function AuthHeading({ eyebrow, title, subtitle, switcher = false }) {
  return (
    <div className="flex flex-col items-center text-center">
      <LogoBadge size="lg" tone="glass" />
      {eyebrow && <p className="eyebrow mt-4">{eyebrow}</p>}
      <h1 className="font-display mt-1.5 text-title font-bold text-ink">{title}</h1>
      {subtitle && <p className="mt-2 text-sm text-muted">{subtitle}</p>}
      {switcher && <PortalSwitch className="mt-5 w-full" />}
    </div>
  )
}

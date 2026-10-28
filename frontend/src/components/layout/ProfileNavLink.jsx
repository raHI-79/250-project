import { NavLink } from 'react-router-dom'
import { LogOut } from 'lucide-react'
import Avatar from '../ui/Avatar'
import { useAuth } from '../../context/auth'
import cx from '../../lib/cx'
import { roleLabel } from '../../lib/nav'
import { paths } from '../../lib/paths'

export default function ProfileNavLink({ onNavigate }) {
  const { user, role, logout } = useAuth()
  const name = user?.name || user?.identifier || ''
  return (
    <div className="flex items-center gap-2 px-3 py-3">
      <NavLink
        to={paths.profile()}
        onClick={onNavigate}
        className={({ isActive }) =>
          cx(
            'flex min-w-0 flex-1 items-center gap-3 rounded-xl px-2 py-1.5 transition-colors',
            isActive ? 'bg-brand-50 text-brand-700' : 'text-ink-soft hover:bg-wash hover:text-ink',
          )
        }
      >
        <Avatar src={user?.photo_url} name={name} size="sm" />
        <span className="min-w-0 leading-tight">
          <span className="block truncate text-sm font-semibold">{name}</span>
          <span className="block truncate text-2xs tracking-[0.06em] text-muted uppercase">
            {roleLabel(role)}
          </span>
        </span>
      </NavLink>
      <button
        type="button"
        onClick={logout}
        aria-label="Sign out"
        title="Sign out"
        className="press tap-target grid shrink-0 place-items-center rounded-full text-muted hover:bg-wash hover:text-ink"
      >
        <LogOut size={17} />
      </button>
    </div>
  )
}

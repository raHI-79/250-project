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
            'flex min-w-0 flex-1 items-center gap-3 px-1 py-1 transition-colors',
            isActive ? 'text-white' : 'text-white/80 hover:text-white',
          )
        }
      >
        <Avatar src={user?.photo_url} name={name} size="sm" />
        <span className="min-w-0 leading-tight">
          <span className="block truncate text-sm font-semibold">{name}</span>
          <span className="block truncate text-xs tracking-[0.04em] text-white/70 uppercase">
            {roleLabel(role)}
          </span>
        </span>
      </NavLink>
      <button
        type="button"
        onClick={logout}
        aria-label="Sign out"
        title="Sign out"
        className="tap-target grid shrink-0 place-items-center text-white/80 transition-colors hover:bg-white/10 hover:text-white"
      >
        <LogOut size={16} />
      </button>
    </div>
  )
}

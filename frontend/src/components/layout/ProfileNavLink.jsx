import { NavLink } from 'react-router-dom'
import Avatar from '../ui/Avatar'
import { useAuth } from '../../context/auth'
import cx from '../../lib/cx'
import { roleLabel } from '../../lib/nav'
import { paths } from '../../lib/paths'

export default function ProfileNavLink({ onNavigate }) {
  const { user, role } = useAuth()
  const name = user?.name || user?.identifier || ''
  return (
    <NavLink
      to={paths.profile()}
      onClick={onNavigate}
      className={({ isActive }) =>
        cx(
          'mx-3 mb-3 flex items-center gap-3 rounded-lg px-3 py-2.5 transition-colors',
          isActive ? 'bg-sidebar-active text-white' : 'text-brand-100/80 hover:bg-sidebar-hover',
        )
      }
    >
      <Avatar src={user?.photo_url} name={name} size="sm" className="border-white/20" />
      <span className="min-w-0 leading-tight">
        <span className="block truncate text-sm font-medium text-white">{name}</span>
        <span className="block truncate text-xs text-brand-200/80">{roleLabel(role)}</span>
      </span>
    </NavLink>
  )
}

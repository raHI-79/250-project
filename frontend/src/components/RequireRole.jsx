import { Navigate, Outlet, useLocation } from 'react-router-dom'
import { useAuth } from '../context/auth'
import { homeFor, paths } from '../lib/paths'

export default function RequireRole({ roles, children }) {
  const { isAuthenticated, role } = useAuth()
  const location = useLocation()
  const allowed = Array.isArray(roles) ? roles : [roles]
  if (!isAuthenticated) {
    const target = allowed.includes('admin') ? paths.adminLogin() : paths.login()
    return <Navigate to={target} replace state={{ from: location.pathname }} />
  }
  if (!allowed.includes(role)) {
    return <Navigate to={homeFor(role)} replace />
  }
  return children || <Outlet />
}

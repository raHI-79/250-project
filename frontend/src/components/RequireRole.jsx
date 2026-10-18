import { Navigate, Outlet, useLocation } from 'react-router-dom'
import { useAuth } from '../context/auth'
import { homeFor, loginForPortal, paths, portalFor } from '../lib/paths'

export default function RequireRole({ roles, children }) {
  const { isAuthenticated, role } = useAuth()
  const location = useLocation()
  const allowed = Array.isArray(roles) ? roles : [roles]
  if (!isAuthenticated) {
    const target = loginForPortal(portalFor(allowed[0]))
    return <Navigate to={target} replace state={{ from: location.pathname }} />
  }
  if (!allowed.includes(role)) {
    return <Navigate to={homeFor(role)} replace />
  }
  return children || <Outlet />
}

export function RequireAuth() {
  const { isAuthenticated, role } = useAuth()
  const location = useLocation()
  if (!isAuthenticated) {
    return <Navigate to={paths.studentLogin()} replace state={{ from: location.pathname }} />
  }
  return <Outlet key={role} />
}

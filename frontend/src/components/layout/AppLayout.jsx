import AppShell from './AppShell'
import { useAuth } from '../../context/auth'
import { navForRole } from '../../lib/nav'

export default function AppLayout() {
  const { role } = useAuth()
  return <AppShell groups={navForRole(role)} showBell={role !== 'admin'} />
}

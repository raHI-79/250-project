import AppShell from './AppShell'
import { useAuth } from '../../context/auth'
import { navForRole } from '../../lib/nav'
import { tabsForRole } from '../../lib/tabs'

export default function AppLayout() {
  const { role } = useAuth()
  return <AppShell groups={navForRole(role)} tabs={tabsForRole(role)} showBell={role !== 'admin'} />
}

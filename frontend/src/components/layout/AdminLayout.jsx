import AppShell from './AppShell'
import { adminNav } from '../../lib/nav'

export default function AdminLayout() {
  return <AppShell groups={adminNav} showBell={false} />
}

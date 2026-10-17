import AppShell from './AppShell'
import { adminNav } from '../../lib/nav'

export default function AdminLayout() {
  return <AppShell items={adminNav} showBell={false} />
}

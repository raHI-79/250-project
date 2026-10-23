import AppShell from './AppShell'
import NoIndex from './NoIndex'
import { adminNav } from '../../lib/nav'

export default function AdminLayout() {
  return (
    <>
      <NoIndex />
      <AppShell groups={adminNav} showBell={false} showSearch={false} />
    </>
  )
}

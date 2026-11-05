import AppShell from './AppShell'
import NoIndex from './NoIndex'
import Alert from '../ui/Alert'
import { useAuth } from '../../context/auth'
import { ROLE_SUPER_ADMIN } from '../../lib/paths'
import { adminNav, controlNav } from '../../lib/nav'

export default function AdminLayout() {
  const { role } = useAuth()
  const watching = role === ROLE_SUPER_ADMIN
  return (
    <>
      <NoIndex />
      {watching && (
        <Alert tone="info" className="fixed inset-x-0 bottom-0 z-30 rounded-none text-center">
          Control room view. The office pages are read-only here.
        </Alert>
      )}
      <AppShell groups={watching ? controlNav : adminNav} showBell={false} showSearch={false} />
    </>
  )
}

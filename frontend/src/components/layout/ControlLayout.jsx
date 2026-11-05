import AppShell from './AppShell'
import NoIndex from './NoIndex'
import { controlNav } from '../../lib/nav'

export default function ControlLayout() {
  return (
    <>
      <NoIndex />
      <AppShell groups={controlNav} showBell={false} showSearch={false} />
    </>
  )
}

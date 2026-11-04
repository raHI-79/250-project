import { AppPortalSwitch } from './PortalSwitch'

export default function AppAuthHeading({ switcher = false }) {
  if (!switcher) return null
  return <AppPortalSwitch className="mb-1" />
}

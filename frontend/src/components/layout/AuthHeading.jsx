import LogoBadge from '../ui/LogoBadge'
import AndroidDownload from '../home/AndroidDownload'
import AppAuthHeading from './AppAuthHeading'
import { PortalSwitch } from './PortalSwitch'
import { isNative } from '../../lib/native'

export { PortalSwitch }

export default function AuthHeading({ eyebrow, title, subtitle, switcher = false }) {
  if (isNative()) return <AppAuthHeading title={title} subtitle={subtitle} switcher={switcher} />
  return (
    <div className="flex flex-col items-center text-center">
      <LogoBadge size="lg" tone="glass" />
      {eyebrow && <p className="eyebrow mt-4">{eyebrow}</p>}
      <h1 className="font-display mt-1.5 text-title font-bold text-ink">{title}</h1>
      {subtitle && <p className="mt-2 text-sm text-muted">{subtitle}</p>}
      {switcher && <PortalSwitch className="mt-5 w-full" />}
      {switcher && <AndroidDownload variant="ghost" size="sm" className="mt-4 w-full" />}
    </div>
  )
}

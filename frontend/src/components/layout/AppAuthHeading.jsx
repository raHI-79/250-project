import BrandMark from '../ui/BrandMark'
import { SUST_MARK } from '../../lib/brand'
import { UNIVERSITY } from '../ui/IdentityBar'
import { AppPortalSwitch } from './PortalSwitch'

export default function AppAuthHeading({ title, subtitle, switcher = false }) {
  return (
    <div className="flex flex-col items-center text-center">
      <BrandMark src={SUST_MARK} alt={UNIVERSITY} className="size-20" />
      <h1 className="font-display mt-4 text-title font-bold text-ink">SUST CSE Portal</h1>
      <p className="mt-1.5 text-sm text-muted">{subtitle || title}</p>
      {switcher && <AppPortalSwitch className="mt-6 w-full" />}
    </div>
  )
}

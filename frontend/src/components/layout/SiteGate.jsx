import { CloudOff, Download, ShieldAlert } from 'lucide-react'
import Alert from '../ui/Alert'
import Button from '../ui/Button'
import LogoBadge from '../ui/LogoBadge'
import { useAuth } from '../../context/auth'
import { announcementFor, updateRequired, useSite } from '../../lib/site'
import { downloadUpdate, fetchLatest, openInstaller } from '../../lib/update'
import { ROLE_SUPER_ADMIN } from '../../lib/paths'

const OFFICE_ROLES = ['admin', ROLE_SUPER_ADMIN]

function Curtain({ icon: Icon, title, children, action }) {
  return (
    <div className="grid min-h-dvh place-items-center bg-page px-5 py-10">
      <div className="surface-card flex max-w-md flex-col items-center gap-4 p-7 text-center">
        <LogoBadge size="lg" tone="glass" />
        <span aria-hidden="true" className="text-brand-600">
          <Icon size={28} />
        </span>
        <h1 className="font-display text-section font-bold text-ink">{title}</h1>
        <p className="text-sm leading-relaxed text-ink-soft">{children}</p>
        {action}
      </div>
    </div>
  )
}

export function MaintenanceCurtain({ message }) {
  return (
    <Curtain icon={CloudOff} title="The portal is having a rest">
      {message || 'The portal is closed for maintenance. Please try again later.'}
    </Curtain>
  )
}

export function UpdateCurtain({ floor }) {
  const update = async () => {
    const latest = await fetchLatest().catch(() => null)
    if (!latest?.apk_url) return
    const file = await downloadUpdate(latest.apk_url, latest.version).catch(() => null)
    if (file) await openInstaller(file).catch(() => null)
  }
  return (
    <Curtain
      icon={ShieldAlert}
      title="Update required"
      action={
        <Button onClick={update} size="lg" className="w-full">
          <Download size={16} aria-hidden="true" />
          Update
        </Button>
      }
    >
      This version of the app is too old to keep using. Version {floor} or newer is needed.
    </Curtain>
  )
}

export function Announcement() {
  const { role } = useAuth()
  const site = useSite()
  const notice = announcementFor(site, role)
  if (!notice) return null
  return (
    <Alert tone={notice.tone === 'error' ? 'error' : notice.tone === 'warning' ? 'warning' : 'info'}>
      {notice.text}
    </Alert>
  )
}

export default function SiteGate({ children }) {
  const { role } = useAuth()
  const site = useSite()
  const office = OFFICE_ROLES.includes(role)
  if (updateRequired(site)) return <UpdateCurtain floor={site.min_app_version} />
  if (site.maintenance && !office) return <MaintenanceCurtain message={site.maintenance_message} />
  return children
}

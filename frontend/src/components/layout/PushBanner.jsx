import { BellRing, X } from 'lucide-react'
import { Button } from '../ui'
import { useAuth } from '../../context/auth'
import { usePush } from '../../lib/push'

export default function PushBanner() {
  const { role } = useAuth()
  const push = usePush(role)
  if (!push.allowed || !push.ready || push.enabled || push.dismissed) return null
  return (
    <div className="animate-fade-in mb-6 flex flex-wrap items-center gap-3 border border-line border-l-[3px] border-l-brand-600 bg-surface px-4 py-3">
      <span className="shrink-0 text-brand-600">
        <BellRing size={18} />
      </span>
      <div className="min-w-0 flex-1">
        <p className="text-sm font-semibold text-ink">Get notifications on this device</p>
        <p className="text-sm text-muted">
          Classes, exams, notices and messages reach you even when this tab is closed.
          {push.error ? ` ${push.error}` : ''}
        </p>
      </div>
      <div className="flex flex-wrap gap-2">
        <Button size="sm" loading={push.busy} onClick={push.subscribe}>
          Turn on
        </Button>
        <Button variant="ghost" size="sm" onClick={push.dismiss} aria-label="Dismiss">
          <X size={15} />
          Not now
        </Button>
      </div>
    </div>
  )
}

import { Video } from 'lucide-react'
import { StatusBadge } from '../ui'
import { isOnline, kindLabel, originLabel, statusLabel, statusTone } from '../../lib/schedule'

export default function ItemBadges({ item, showOrigin = false, showFinal = false, statusText }) {
  return (
    <>
      <StatusBadge status={statusTone(item.status)} label={statusText || statusLabel(item.status)} />
      {item.kind !== 'class' && <StatusBadge status="unavailable" label={kindLabel(item.kind)} />}
      {isOnline(item) && (
        <span className="inline-flex items-center gap-1 rounded-full border border-line-strong bg-wash px-2.5 py-0.5 text-xs font-medium text-ink-soft">
          <Video size={12} />
          Online
        </span>
      )}
      {showOrigin && originLabel(item.origin) && (
        <StatusBadge status="unavailable" label={originLabel(item.origin)} />
      )}
      {item.needs_time && <StatusBadge status="pending" label="Needs time" />}
      {item.changed_since_publish && <StatusBadge status="rejected" label="Changed since publish" />}
      {showFinal && item.published && <StatusBadge status="approved" label="Published" />}
    </>
  )
}

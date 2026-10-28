import { Inbox } from 'lucide-react'
import cx from '../../lib/cx'

export default function EmptyState({ icon: Icon = Inbox, title, text, action, className }) {
  return (
    <div
      className={cx(
        'glass-card flex flex-col items-center justify-center gap-3 rounded-card border-dashed px-5 py-10 text-center',
        className,
      )}
    >
      <span className="grid size-14 place-items-center rounded-full bg-brand-50 text-brand-600">
        <Icon size={24} />
      </span>
      <div className="max-w-xs">
        <p className="font-display text-base font-bold text-ink">{title}</p>
        {text && <p className="mt-1 text-sm text-muted">{text}</p>}
      </div>
      {action && <div className="mt-1 flex flex-wrap justify-center gap-2">{action}</div>}
    </div>
  )
}

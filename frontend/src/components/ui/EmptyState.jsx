import { Inbox } from 'lucide-react'
import cx from '../../lib/cx'

export default function EmptyState({ icon: Icon = Inbox, title, text, action, className }) {
  return (
    <div
      className={cx(
        'flex flex-col items-center justify-center gap-3 border border-dashed border-line-strong bg-surface px-5 py-12 text-center sm:px-6',
        className,
      )}
    >
      <span className="grid size-12 place-items-center rounded-full border border-line bg-wash text-brand-600">
        <Icon size={22} />
      </span>
      <div className="max-w-md">
        <p className="font-display text-lg font-bold text-ink">{title}</p>
        {text && <p className="mt-1.5 text-sm text-muted">{text}</p>}
      </div>
      {action && <div className="mt-1 flex flex-wrap justify-center gap-2">{action}</div>}
    </div>
  )
}

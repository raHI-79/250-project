import { Inbox } from 'lucide-react'
import cx from '../../lib/cx'

export default function EmptyState({ icon: Icon = Inbox, title, text, action, className }) {
  return (
    <div
      className={cx(
        'flex flex-col items-center justify-center gap-3 rounded-xl border border-dashed border-brand-200 bg-surface px-6 py-12 text-center',
        className,
      )}
    >
      <span className="flex size-12 items-center justify-center rounded-full bg-brand-50 text-brand-500">
        <Icon size={22} />
      </span>
      <div>
        <p className="font-display text-base font-semibold text-brand-900">{title}</p>
        {text && <p className="mt-1 text-sm text-brand-500">{text}</p>}
      </div>
      {action}
    </div>
  )
}

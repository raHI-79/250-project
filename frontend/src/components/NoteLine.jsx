import { StickyNote } from 'lucide-react'
import cx from '../lib/cx'
import { NOTE_AUDIENCE_LABELS } from '../lib/options'

export default function NoteLine({ text, audience, className }) {
  if (!text) return null
  return (
    <p className={cx('flex items-start gap-1.5 text-xs text-brand-500', className)}>
      <StickyNote size={13} className="mt-0.5 shrink-0 text-brand-400" />
      <span className="whitespace-pre-line">
        {text}
        {audience && NOTE_AUDIENCE_LABELS[audience] && (
          <span className="text-brand-400"> · {NOTE_AUDIENCE_LABELS[audience]}</span>
        )}
      </span>
    </p>
  )
}

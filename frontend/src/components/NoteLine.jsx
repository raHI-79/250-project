import { StickyNote } from 'lucide-react'
import cx from '../lib/cx'
import { NOTE_AUDIENCE_LABELS } from '../lib/options'

export default function NoteLine({ text, audience, className }) {
  if (!text) return null
  return (
    <p
      className={cx(
        'flex items-start gap-1.5 border-l-[3px] border-gold-500 bg-gold-50 px-2.5 py-1.5 text-xs text-gold-800',
        className,
      )}
    >
      <StickyNote size={13} className="mt-0.5 shrink-0 text-gold-600" />
      <span className="whitespace-pre-line">
        {text}
        {audience && NOTE_AUDIENCE_LABELS[audience] && (
          <span className="text-gold-700"> · {NOTE_AUDIENCE_LABELS[audience]}</span>
        )}
      </span>
    </p>
  )
}

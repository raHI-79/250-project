import { useEffect } from 'react'
import { X } from 'lucide-react'
import cx from '../../lib/cx'

export default function Modal({ open, onClose, title, children, footer, className }) {
  useEffect(() => {
    if (!open) return undefined
    const onKeyDown = (event) => {
      if (event.key === 'Escape') onClose?.()
    }
    window.addEventListener('keydown', onKeyDown)
    return () => window.removeEventListener('keydown', onKeyDown)
  }, [open, onClose])
  if (!open) return null
  return (
    <div className="fixed inset-0 z-80 flex items-end justify-center p-0 sm:items-center sm:p-4">
      <button
        type="button"
        aria-label="Close dialog"
        onClick={() => onClose?.()}
        className="animate-fade-in absolute inset-0 cursor-default bg-ink/55"
      />
      <div
        role="dialog"
        aria-modal="true"
        aria-label={typeof title === 'string' ? title : undefined}
        className={cx(
          'animate-scale-in relative flex max-h-[92dvh] w-full max-w-lg flex-col border border-line bg-surface shadow-xl',
          className,
        )}
      >
        <div className="flex shrink-0 items-start justify-between gap-4 border-b border-line px-5 py-4">
          <h2 className="font-display text-lg font-bold text-ink">{title}</h2>
          <button
            type="button"
            onClick={() => onClose?.()}
            aria-label="Close"
            className="tap-target -mr-2 grid place-items-center rounded text-faint transition-colors hover:bg-wash hover:text-ink-soft"
          >
            <X size={18} />
          </button>
        </div>
        <div className="min-h-0 flex-1 overflow-y-auto px-5 py-4 text-sm text-ink-soft">
          {children}
        </div>
        {footer && (
          <div className="sticky bottom-0 flex shrink-0 flex-wrap justify-end gap-2 border-t border-line bg-surface px-5 py-4">
            {footer}
          </div>
        )}
      </div>
    </div>
  )
}

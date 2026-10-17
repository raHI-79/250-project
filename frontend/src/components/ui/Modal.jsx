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
    <div className="fixed inset-0 z-[80] flex items-center justify-center p-4">
      <button
        type="button"
        aria-label="Close dialog"
        onClick={() => onClose?.()}
        className="animate-fade-in absolute inset-0 cursor-default bg-brand-950/45"
      />
      <div
        role="dialog"
        aria-modal="true"
        aria-label={typeof title === 'string' ? title : undefined}
        className={cx(
          'animate-scale-in relative w-full max-w-lg rounded-xl border border-brand-100 bg-surface shadow-2xl',
          className,
        )}
      >
        <div className="flex items-start justify-between gap-4 border-b border-brand-100 px-5 py-4">
          <h2 className="font-display text-lg font-semibold text-brand-900">{title}</h2>
          <button
            type="button"
            onClick={() => onClose?.()}
            aria-label="Close"
            className="rounded p-1 text-brand-400 transition-colors hover:bg-brand-50 hover:text-brand-700"
          >
            <X size={18} />
          </button>
        </div>
        <div className="px-5 py-4 text-sm text-brand-700">{children}</div>
        {footer && (
          <div className="flex justify-end gap-2 border-t border-brand-100 px-5 py-4">{footer}</div>
        )}
      </div>
    </div>
  )
}

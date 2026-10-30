import { useCallback, useEffect, useRef, useState } from 'react'
import { X } from 'lucide-react'
import cx from '../../lib/cx'

const CLOSE_AFTER = 96

export default function Sheet({
  open,
  onClose,
  title,
  description,
  full = false,
  footer,
  className,
  children,
}) {
  const [drag, setDrag] = useState(0)
  const startY = useRef(0)
  const dragging = useRef(false)
  const body = useRef(null)
  const keepVisible = useCallback((event) => {
    const field = event.target
    if (!field || typeof field.scrollIntoView !== 'function') return
    window.setTimeout(() => field.scrollIntoView({ block: 'nearest', behavior: 'smooth' }), 120)
  }, [])
  useEffect(() => {
    if (!open) return undefined
    const onKeyDown = (event) => {
      if (event.key === 'Escape') onClose?.()
    }
    const previous = document.body.style.overflow
    document.body.style.overflow = 'hidden'
    window.addEventListener('keydown', onKeyDown)
    return () => {
      document.body.style.overflow = previous
      window.removeEventListener('keydown', onKeyDown)
    }
  }, [open, onClose])
  useEffect(() => {
    if (!open) setDrag(0)
  }, [open])
  if (!open) return null
  const onTouchStart = (event) => {
    startY.current = event.touches[0].clientY
    dragging.current = true
  }
  const onTouchMove = (event) => {
    if (!dragging.current) return
    setDrag(Math.max(0, event.touches[0].clientY - startY.current))
  }
  const onTouchEnd = () => {
    dragging.current = false
    if (drag > CLOSE_AFTER) onClose?.()
    else setDrag(0)
  }
  const dismissible = Boolean(onClose)
  return (
    <div className="fixed inset-0 z-80 flex items-end justify-center sm:items-center sm:p-4">
      {dismissible ? (
        <button
          type="button"
          aria-label="Close"
          onClick={() => onClose()}
          className="animate-fade-in absolute inset-0 cursor-default bg-black/55"
        />
      ) : (
        <div aria-hidden="true" className="animate-fade-in absolute inset-0 bg-black/55" />
      )}
      <section
        role="dialog"
        aria-modal="true"
        aria-label={typeof title === 'string' ? title : undefined}
        style={drag ? { transform: `translateY(${drag}px)` } : undefined}
        className={cx(
          'glass-sheet animate-sheet-up relative flex w-full flex-col overflow-hidden rounded-t-sheet sm:animate-scale-in sm:rounded-card',
          full
            ? 'h-[92dvh] max-h-[92dvh] sm:h-[90dvh] sm:max-w-2xl'
            : 'h-auto max-h-[92dvh] sm:max-w-lg',
          className,
        )}
      >
        <div
          onTouchStart={dismissible ? onTouchStart : undefined}
          onTouchMove={dismissible ? onTouchMove : undefined}
          onTouchEnd={dismissible ? onTouchEnd : undefined}
          className="flex shrink-0 cursor-grab touch-none justify-center pt-[max(0.5rem,env(safe-area-inset-top,0px))] pb-1 sm:hidden"
        >
          <span aria-hidden="true" className="h-1.5 w-11 rounded-full bg-line-strong" />
        </div>
        <header
          className={cx(
            'flex shrink-0 items-start gap-3 px-5 pb-3 sm:pt-4',
            full ? 'pt-[max(0.75rem,env(safe-area-inset-top,0px))]' : 'pt-2',
          )}
        >
          <div className="min-w-0 flex-1">
            <h2 className="font-display text-section font-bold text-ink">{title}</h2>
            {description && <p className="mt-1 text-xs text-muted">{description}</p>}
          </div>
          {dismissible && (
            <button
              type="button"
              onClick={() => onClose()}
              aria-label="Close"
              className="press tap-target -mt-1.5 -mr-3 grid shrink-0 place-items-center rounded-full text-muted hover:bg-wash hover:text-ink"
            >
              <X size={20} />
            </button>
          )}
        </header>
        <div
          ref={body}
          onFocus={keepVisible}
          className={cx(
            'min-h-0 flex-1 overscroll-contain overflow-y-auto px-5 text-sm text-ink-soft',
            footer ? 'pb-4' : 'pb-[max(1.25rem,env(safe-area-inset-bottom,0px))]',
          )}
        >
          {children}
        </div>
        {footer && (
          <footer className="flex shrink-0 flex-wrap justify-end gap-2 border-t border-line bg-surface px-5 pt-4 pb-[max(1rem,env(safe-area-inset-bottom,0px))]">
            {footer}
          </footer>
        )}
      </section>
    </div>
  )
}

import { useCallback, useEffect, useRef, useState } from 'react'
import { createPortal } from 'react-dom'
import { X } from 'lucide-react'
import cx from '../../lib/cx'
import { lockScroll } from '../../lib/scrollLock'
import { pushSheet } from '../../lib/sheetStack'
import { SheetBoundary } from './ErrorBoundary'

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
  const close = useRef(onClose)
  close.current = onClose
  const keepVisible = useCallback((event) => {
    const field = event.target
    if (!field || typeof field.scrollIntoView !== 'function') return
    const reveal = () => field.scrollIntoView({ block: 'nearest', behavior: 'smooth' })
    window.setTimeout(reveal, 120)
    window.setTimeout(reveal, 420)
  }, [])
  useEffect(() => {
    if (!open) return undefined
    const onKeyDown = (event) => {
      if (event.key === 'Escape') close.current?.()
    }
    const release = lockScroll()
    const leave = pushSheet({ close: () => close.current?.() })
    window.addEventListener('keydown', onKeyDown)
    return () => {
      release()
      leave()
      window.removeEventListener('keydown', onKeyDown)
    }
  }, [open])
  useEffect(() => {
    if (!open) setDrag(0)
  }, [open])
  if (!open || typeof document === 'undefined') return null
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
    if (drag > CLOSE_AFTER) close.current?.()
    else setDrag(0)
  }
  const dismissible = Boolean(onClose)
  return createPortal(
    <div
      data-sheet-root
      className="fixed inset-0 z-80 flex items-end justify-center overflow-hidden sm:items-center sm:p-4"
    >
      {dismissible ? (
        <button
          type="button"
          aria-label="Close"
          onClick={() => onClose()}
          className="animate-fade-in absolute inset-0 cursor-default bg-scrim"
        />
      ) : (
        <div aria-hidden="true" className="animate-fade-in absolute inset-0 bg-scrim" />
      )}
      <section
        role="dialog"
        aria-modal="true"
        aria-label={typeof title === 'string' ? title : undefined}
        style={drag ? { transform: `translateY(${drag}px)` } : undefined}
        className={cx(
          'sheet-panel animate-sheet-up relative flex w-full max-w-full flex-col overflow-hidden rounded-t-sheet border border-line bg-surface shadow-[var(--shadow-lift)] sm:animate-scale-in sm:rounded-card',
          full ? 'sheet-full sm:max-w-2xl' : 'h-auto sm:max-w-lg',
          className,
        )}
      >
        <div
          onTouchStart={dismissible ? onTouchStart : undefined}
          onTouchMove={dismissible ? onTouchMove : undefined}
          onTouchEnd={dismissible ? onTouchEnd : undefined}
          className="flex shrink-0 cursor-grab touch-none justify-center pt-2 pb-1 sm:hidden"
        >
          <span aria-hidden="true" className="h-1.5 w-11 rounded-full bg-line-strong" />
        </div>
        <header className="flex shrink-0 items-start gap-3 border-b border-line bg-surface px-5 pt-2 pb-3 sm:pt-4">
          <div className="min-w-0 flex-1">
            <h2 className="font-display wrap-long text-section font-bold text-ink">{title}</h2>
            {description && <p className="mt-1 wrap-long text-xs text-muted">{description}</p>}
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
          data-sheet-body
          onFocus={keepVisible}
          className={cx(
            'min-h-0 flex-1 overscroll-contain overflow-x-hidden overflow-y-auto px-5 pt-4 text-sm text-ink-soft',
            footer ? 'pb-4' : 'pb-[max(1.25rem,env(safe-area-inset-bottom,0px))]',
          )}
        >
          <SheetBoundary onClose={dismissible ? () => onClose() : null}>{children}</SheetBoundary>
        </div>
        {footer && (
          <footer
            data-sheet-footer
            className="flex shrink-0 flex-wrap justify-end gap-2 border-t border-line bg-surface px-5 pt-4 pb-[max(1rem,env(safe-area-inset-bottom,0px))]"
          >
            {footer}
          </footer>
        )}
      </section>
    </div>,
    document.body,
  )
}

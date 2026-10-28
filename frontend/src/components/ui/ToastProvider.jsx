import { useCallback, useMemo, useRef, useState } from 'react'
import { CheckCircle2, Info, X, XCircle } from 'lucide-react'
import cx from '../../lib/cx'
import { ToastContext } from './toast'

const MAX_TOASTS = 3

const DISMISS_AFTER = 4000

const tones = {
  success: { icon: CheckCircle2, className: 'text-success-700' },
  error: { icon: XCircle, className: 'text-danger-700' },
  info: { icon: Info, className: 'text-brand-700' },
}

export default function ToastProvider({ children }) {
  const [toasts, setToasts] = useState([])
  const timers = useRef(new Map())
  const counter = useRef(0)
  const dismiss = useCallback((id) => {
    const timer = timers.current.get(id)
    if (timer) {
      window.clearTimeout(timer)
      timers.current.delete(id)
    }
    setToasts((current) => current.filter((toast) => toast.id !== id))
  }, [])
  const push = useCallback(
    (message, tone = 'info') => {
      counter.current += 1
      const id = counter.current
      setToasts((current) => [...current, { id, message, tone }].slice(-MAX_TOASTS))
      const timer = window.setTimeout(() => dismiss(id), DISMISS_AFTER)
      timers.current.set(id, timer)
      return id
    },
    [dismiss],
  )
  const value = useMemo(
    () => ({
      toast: push,
      success: (message) => push(message, 'success'),
      error: (message) => push(message, 'error'),
      info: (message) => push(message, 'info'),
      dismiss,
    }),
    [push, dismiss],
  )
  return (
    <ToastContext.Provider value={value}>
      {children}
      <div className="pointer-events-none fixed inset-x-0 bottom-[calc(4.5rem+env(safe-area-inset-bottom,0px))] z-90 mx-auto flex w-[min(26rem,calc(100vw-2rem))] flex-col gap-2 lg:bottom-auto lg:top-4 lg:right-4 lg:mx-0 lg:ml-auto">
        {toasts.map((toast) => {
          const config = tones[toast.tone] || tones.info
          const Icon = config.icon
          return (
            <div
              key={toast.id}
              role="status"
              className={cx(
                'glass-sheet animate-fade-in-up pointer-events-auto flex items-start gap-3 rounded-card px-4 py-3 text-sm',
                config.className,
              )}
            >
              <Icon size={18} className="mt-0.5 shrink-0" />
              <p className="min-w-0 flex-1 wrap-long text-ink">{toast.message}</p>
              <button
                type="button"
                onClick={() => dismiss(toast.id)}
                aria-label="Dismiss"
                className="rounded p-0.5 text-faint transition-colors hover:bg-wash hover:text-ink-soft"
              >
                <X size={15} />
              </button>
            </div>
          )
        })}
      </div>
    </ToastContext.Provider>
  )
}

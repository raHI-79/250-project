import { useCallback, useMemo, useRef, useState } from 'react'
import { CheckCircle2, Info, X, XCircle } from 'lucide-react'
import cx from '../../lib/cx'
import { ToastContext } from './toast'

const MAX_TOASTS = 4
const DISMISS_AFTER = 4000

const tones = {
  success: { icon: CheckCircle2, className: 'border-l-emerald-600 text-ink' },
  error: { icon: XCircle, className: 'border-l-accent-600 text-accent-700' },
  info: { icon: Info, className: 'border-l-brand-600 text-ink' },
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
      <div className="pointer-events-none fixed top-4 right-4 z-[70] flex w-[min(22rem,calc(100vw-2rem))] flex-col gap-2">
        {toasts.map((toast) => {
          const config = tones[toast.tone] || tones.info
          const Icon = config.icon
          return (
            <div
              key={toast.id}
              role="status"
              className={cx(
                'animate-slide-in-right pointer-events-auto flex items-start gap-3 border-l-[3px] border border-line bg-surface px-4 py-3 text-sm shadow-lg',
                config.className,
              )}
            >
              <Icon size={18} className="mt-0.5 shrink-0" />
              <p className="min-w-0 flex-1 break-words">{toast.message}</p>
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

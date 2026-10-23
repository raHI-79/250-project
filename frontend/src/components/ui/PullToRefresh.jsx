import { useCallback, useEffect, useRef, useState } from 'react'
import { RefreshCw } from 'lucide-react'
import cx from '../../lib/cx'

const TRIGGER = 72

const MAX_PULL = 110

export default function PullToRefresh({ onRefresh, className, children }) {
  const [pull, setPull] = useState(0)
  const [busy, setBusy] = useState(false)
  const startY = useRef(0)
  const tracking = useRef(false)
  const run = useCallback(async () => {
    setBusy(true)
    try {
      await onRefresh?.()
    } finally {
      setBusy(false)
      setPull(0)
    }
  }, [onRefresh])
  useEffect(() => {
    if (!onRefresh) return undefined
    const blocked = () => document.body.style.overflow === 'hidden'
    const atTop = () => (document.scrollingElement?.scrollTop ?? 0) <= 0
    const onStart = (event) => {
      if (busy || blocked() || event.touches.length !== 1 || !atTop()) return
      startY.current = event.touches[0].clientY
      tracking.current = true
    }
    const onMove = (event) => {
      if (!tracking.current) return
      const distance = event.touches[0].clientY - startY.current
      if (distance <= 0) {
        tracking.current = false
        setPull(0)
        return
      }
      setPull(Math.min(MAX_PULL, distance * 0.5))
    }
    const onEnd = () => {
      if (!tracking.current) return
      tracking.current = false
      setPull((current) => {
        if (current >= TRIGGER * 0.5) {
          run()
          return TRIGGER * 0.5
        }
        return 0
      })
    }
    window.addEventListener('touchstart', onStart, { passive: true })
    window.addEventListener('touchmove', onMove, { passive: true })
    window.addEventListener('touchend', onEnd)
    window.addEventListener('touchcancel', onEnd)
    return () => {
      window.removeEventListener('touchstart', onStart)
      window.removeEventListener('touchmove', onMove)
      window.removeEventListener('touchend', onEnd)
      window.removeEventListener('touchcancel', onEnd)
    }
  }, [onRefresh, busy, run])
  const active = pull > 0 || busy
  return (
    <div className={cx('relative', className)}>
      <div
        aria-hidden={!active}
        style={{ height: active ? `${Math.max(pull, busy ? 36 : 0)}px` : '0px' }}
        className="flex items-center justify-center overflow-hidden text-brand-600 transition-[height] duration-200 lg:hidden"
      >
        <RefreshCw size={18} className={cx(busy && 'animate-spin')} />
      </div>
      {children}
    </div>
  )
}

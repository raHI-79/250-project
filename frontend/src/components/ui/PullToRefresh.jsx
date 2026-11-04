import { useCallback, useEffect, useRef, useState } from 'react'
import { RefreshCw } from 'lucide-react'
import cx from '../../lib/cx'
import { isScrollLocked } from '../../lib/scrollLock'

const TRIGGER = 72

const MAX_PULL = 110

const SLACK = 12

export default function PullToRefresh({ onRefresh, className, children }) {
  const [pull, setPull] = useState(0)
  const [busy, setBusy] = useState(false)
  const startY = useRef(0)
  const startX = useRef(0)
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
    const atTop = () => (document.scrollingElement?.scrollTop ?? 0) <= 0
    const onStart = (event) => {
      tracking.current = false
      if (busy || isScrollLocked() || event.touches.length !== 1 || !atTop()) return
      startY.current = event.touches[0].clientY
      startX.current = event.touches[0].clientX
      tracking.current = true
    }
    const onMove = (event) => {
      if (!tracking.current) return
      const touch = event.touches[0]
      const down = touch.clientY - startY.current
      const sideways = Math.abs(touch.clientX - startX.current)
      if (down <= SLACK || sideways > Math.abs(down)) {
        if (down < 0 || sideways > Math.abs(down)) tracking.current = false
        setPull(0)
        return
      }
      setPull(Math.min(MAX_PULL, (down - SLACK) * 0.5))
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
    const options = { passive: true }
    window.addEventListener('touchstart', onStart, options)
    window.addEventListener('touchmove', onMove, options)
    window.addEventListener('touchend', onEnd, options)
    window.addEventListener('touchcancel', onEnd, options)
    return () => {
      window.removeEventListener('touchstart', onStart, options)
      window.removeEventListener('touchmove', onMove, options)
      window.removeEventListener('touchend', onEnd, options)
      window.removeEventListener('touchcancel', onEnd, options)
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

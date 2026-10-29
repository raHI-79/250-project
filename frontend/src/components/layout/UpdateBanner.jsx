import { useEffect, useState } from 'react'
import { RefreshCw } from 'lucide-react'
import Button from '../ui/Button'

export default function UpdateBanner() {
  const [waiting, setWaiting] = useState(null)
  useEffect(() => {
    if (!('serviceWorker' in navigator)) return undefined
    let live = true
    const watch = (registration) => {
      if (!registration) return
      if (registration.waiting) setWaiting(registration.waiting)
      registration.addEventListener('updatefound', () => {
        const installing = registration.installing
        if (!installing) return
        installing.addEventListener('statechange', () => {
          if (installing.state === 'installed' && navigator.serviceWorker.controller && live) {
            setWaiting(installing)
          }
        })
      })
    }
    const check = () => navigator.serviceWorker.getRegistration('/').then((registration) => {
      if (!live || !registration) return
      watch(registration)
      registration.update().catch(() => null)
    })
    check()
    const onVisible = () => {
      if (document.visibilityState === 'visible') check()
    }
    document.addEventListener('visibilitychange', onVisible)
    return () => {
      live = false
      document.removeEventListener('visibilitychange', onVisible)
    }
  }, [])
  if (!waiting) return null
  const reload = () => {
    navigator.serviceWorker.addEventListener('controllerchange', () => window.location.reload(), {
      once: true,
    })
    waiting.postMessage({ type: 'SKIP_WAITING' })
  }
  return (
    <div className="glass-sheet animate-fade-in-up fixed inset-x-3 bottom-[calc(4.75rem+env(safe-area-inset-bottom,0px))] z-70 flex items-center gap-3 rounded-card px-4 py-3 text-sm lg:inset-x-auto lg:right-6 lg:bottom-6 lg:w-80">
      <RefreshCw size={16} className="shrink-0 text-brand-600" />
      <p className="min-w-0 flex-1 text-ink-soft">A new version is ready</p>
      <Button size="sm" onClick={reload}>
        Reload
      </Button>
    </div>
  )
}

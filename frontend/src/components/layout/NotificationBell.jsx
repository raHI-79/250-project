import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { Bell, BellOff } from 'lucide-react'
import api from '../../lib/api'
import cx from '../../lib/cx'
import { readRaw, writeRaw } from '../../lib/storage'
import { relativeTime } from '../../lib/time'
import { useAuth } from '../../context/auth'
import EmptyState from '../ui/EmptyState'

const POLL_INTERVAL = 30000

const lastSeenKey = (userId) => `sustcse.notifications.lastSeen.${userId}`

export default function NotificationBell() {
  const { user } = useAuth()
  const userId = user?.id
  const [items, setItems] = useState([])
  const [open, setOpen] = useState(false)
  const [lastSeen, setLastSeen] = useState(() =>
    userId ? readRaw(lastSeenKey(userId)) : null,
  )
  const containerRef = useRef(null)
  const load = useCallback(async () => {
    const data = await api.get('/api/notifications', { query: { limit: 30 } }).catch(() => null)
    if (!data) return
    setItems(Array.isArray(data.items) ? data.items : [])
  }, [])
  useEffect(() => {
    if (!userId) return undefined
    setLastSeen(readRaw(lastSeenKey(userId)))
    load()
    let timer = window.setInterval(load, POLL_INTERVAL)
    const onVisibility = () => {
      window.clearInterval(timer)
      if (document.visibilityState === 'visible') {
        load()
        timer = window.setInterval(load, POLL_INTERVAL)
      }
    }
    document.addEventListener('visibilitychange', onVisibility)
    return () => {
      window.clearInterval(timer)
      document.removeEventListener('visibilitychange', onVisibility)
    }
  }, [userId, load])
  useEffect(() => {
    if (!open) return undefined
    const onClickOutside = (event) => {
      if (!containerRef.current?.contains(event.target)) setOpen(false)
    }
    const onKeyDown = (event) => {
      if (event.key === 'Escape') setOpen(false)
    }
    document.addEventListener('mousedown', onClickOutside)
    document.addEventListener('keydown', onKeyDown)
    return () => {
      document.removeEventListener('mousedown', onClickOutside)
      document.removeEventListener('keydown', onKeyDown)
    }
  }, [open])
  const unread = useMemo(() => {
    if (!items.length) return 0
    const since = lastSeen ? new Date(lastSeen).getTime() : 0
    if (!since) return items.length
    return items.filter((item) => new Date(item.created_at).getTime() > since).length
  }, [items, lastSeen])
  const toggle = () => {
    const next = !open
    setOpen(next)
    if (next && userId) {
      const now = new Date().toISOString()
      writeRaw(lastSeenKey(userId), now)
      setLastSeen(now)
    }
  }
  return (
    <div ref={containerRef} className="relative">
      <button
        type="button"
        onClick={toggle}
        aria-label="Notifications"
        aria-expanded={open}
        className="relative rounded-lg p-2 text-brand-700 transition-colors hover:bg-brand-50 active:scale-95"
      >
        <Bell size={19} />
        {unread > 0 && (
          <span className="absolute -top-0.5 -right-0.5 flex min-w-4.5 items-center justify-center rounded-full bg-accent-600 px-1 text-[10px] font-semibold text-white">
            {unread > 9 ? '9+' : unread}
          </span>
        )}
      </button>
      {open && (
        <div
          className={cx(
            'animate-scale-in absolute right-0 z-50 mt-2 w-[min(22rem,calc(100vw-2rem))] origin-top-right rounded-xl border border-brand-100 bg-surface shadow-xl',
          )}
        >
          <div className="border-b border-brand-100 px-4 py-3">
            <p className="font-display text-sm font-semibold text-brand-900">Notifications</p>
          </div>
          <div className="max-h-80 overflow-y-auto">
            {items.length === 0 ? (
              <div className="p-3">
                <EmptyState
                  icon={BellOff}
                  title="Nothing yet"
                  text="New updates will show up here."
                  className="border-0 py-6"
                />
              </div>
            ) : (
              <ul>
                {items.map((item) => (
                  <li
                    key={item.id}
                    className="border-b border-brand-50 px-4 py-3 last:border-b-0 hover:bg-brand-50/60"
                  >
                    <p className="text-sm font-medium text-brand-900">{item.title}</p>
                    {item.body && <p className="mt-0.5 text-sm text-brand-600">{item.body}</p>}
                    <p className="mt-1 text-xs text-brand-400">{relativeTime(item.created_at)}</p>
                  </li>
                ))}
              </ul>
            )}
          </div>
        </div>
      )}
    </div>
  )
}

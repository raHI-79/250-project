import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Bell, BellOff, CheckCheck } from 'lucide-react'
import api from '../../lib/api'
import cx from '../../lib/cx'
import { notificationTarget } from '../../lib/paths'
import { readRaw, writeRaw } from '../../lib/storage'
import { relativeTime } from '../../lib/time'
import { useAuth } from '../../context/auth'
import Button from '../ui/Button'
import EmptyState from '../ui/EmptyState'

const POLL_INTERVAL = 30000

const lastSeenKey = (userId) => `sustcse.notifications.lastSeen.${userId}`

export default function NotificationBell() {
  const { user } = useAuth()
  const navigate = useNavigate()
  const userId = user?.id
  const [items, setItems] = useState([])
  const [open, setOpen] = useState(false)
  const [readIds, setReadIds] = useState([])
  const [lastSeen, setLastSeen] = useState(() => (userId ? readRaw(lastSeenKey(userId)) : null))
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
  const seenAt = lastSeen ? new Date(lastSeen).getTime() : 0
  const isUnread = useCallback(
    (item) =>
      !readIds.includes(item.id) &&
      (!seenAt || new Date(item.created_at).getTime() > seenAt),
    [readIds, seenAt],
  )
  const unread = useMemo(() => items.filter(isUnread).length, [items, isUnread])
  const markAll = () => {
    if (!userId) return
    const now = new Date().toISOString()
    writeRaw(lastSeenKey(userId), now)
    setLastSeen(now)
    setReadIds(items.map((item) => item.id))
  }
  const openItem = (item) => {
    setReadIds((current) => (current.includes(item.id) ? current : [...current, item.id]))
    setOpen(false)
    navigate(notificationTarget(item.link, user?.role))
  }
  return (
    <div ref={containerRef} className="relative">
      <button
        type="button"
        onClick={() => setOpen((current) => !current)}
        aria-label="Notifications"
        aria-expanded={open}
        className="tap-target relative grid place-items-center rounded-md text-ink-soft transition-colors hover:bg-wash active:scale-95"
      >
        <Bell size={19} />
        {unread > 0 && (
          <span className="absolute -top-0.5 -right-0.5 flex min-w-4.5 items-center justify-center rounded-full bg-accent-600 px-1 text-xs font-semibold text-white">
            {unread > 9 ? '9+' : unread}
          </span>
        )}
      </button>
      {open && (
        <div className="animate-scale-in absolute right-0 z-50 mt-2 flex max-h-[80dvh] w-[min(22rem,calc(100vw-2rem))] origin-top-right flex-col border border-line bg-surface shadow-lg">
          <div className="flex items-center gap-2 border-b border-line px-4 py-3">
            <p className="font-display text-sm font-semibold text-ink">Notifications</p>
            {items.length > 0 && (
              <Button variant="ghost" size="sm" onClick={markAll} className="ml-auto">
                <CheckCheck size={14} />
                Mark all as read
              </Button>
            )}
          </div>
          <div className="min-h-0 flex-1 overflow-y-auto">
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
                  <li key={item.id}>
                    <button
                      type="button"
                      onClick={() => openItem(item)}
                      className={cx(
                        'flex w-full items-start gap-2 border-b border-line px-4 py-3 text-left transition-colors last:border-b-0 hover:bg-wash',
                        isUnread(item) && 'bg-wash/40',
                      )}
                    >
                      <span
                        className={cx(
                          'mt-1.5 size-2 shrink-0 rounded-full',
                          isUnread(item) ? 'bg-accent-600' : 'bg-transparent',
                        )}
                      />
                      <span className="min-w-0">
                        <span className="block text-sm font-medium text-ink">
                          {item.title}
                        </span>
                        {item.body && (
                          <span className="mt-0.5 block text-sm text-brand-600">{item.body}</span>
                        )}
                        <span className="mt-1 block text-xs text-faint">
                          {relativeTime(item.created_at)}
                        </span>
                      </span>
                    </button>
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

import { useEffect, useRef, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Bell, BellOff, CheckCheck } from 'lucide-react'
import cx from '../../lib/cx'
import { notificationTarget } from '../../lib/paths'
import {
  markAllNotificationsRead,
  markNotificationRead,
  useNotifications,
} from '../../lib/notifications'
import { stampLabel } from '../../lib/time'
import { useAuth } from '../../context/auth'
import Button from '../ui/Button'
import CountBadge from '../ui/CountBadge'
import EmptyState from '../ui/EmptyState'
import Sheet from '../ui/Sheet'
import { useMediaQuery } from '../../lib/hooks'

export default function NotificationBell() {
  const { user } = useAuth()
  const navigate = useNavigate()
  const { items, unread } = useNotifications()
  const [open, setOpen] = useState(false)
  const containerRef = useRef(null)
  const phone = useMediaQuery('(max-width: 639.98px)')
  useEffect(() => {
    if (!open) return undefined
    const onClickOutside = (event) => {
      if (event.target instanceof Element && event.target.closest('[data-sheet-root]')) return
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
  const isUnread = (item) => !item.read
  const markAll = () => {
    markAllNotificationsRead()
  }
  const openItem = (item) => {
    markNotificationRead(item.id)
    setOpen(false)
    navigate(notificationTarget(item.link, user?.role))
  }
  const markAllButton =
    unread > 0 ? (
      <Button variant="ghost" size="sm" onClick={markAll} className="ml-auto">
        <CheckCheck size={14} />
        Mark all as read
      </Button>
    ) : null
  const list = (
    <>
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
                data-notification={item.id}
                data-unread={isUnread(item) ? 'true' : 'false'}
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
                    {stampLabel(item.created_at)}
                  </span>
                </span>
              </button>
            </li>
          ))}
        </ul>
      )}
    </>
  )
  return (
    <div ref={containerRef} className="relative">
      <button
        type="button"
        onClick={() => setOpen((current) => !current)}
        aria-label="Notifications"
        aria-expanded={open}
        className="press tap-target relative grid place-items-center rounded-full text-white hover:bg-white/15"
      >
        <Bell size={19} />
        {unread > 0 && (
          <CountBadge count={unread} className="absolute -top-0.5 -right-0.5" />
        )}
      </button>
      {open && phone && (
        <Sheet open title="Notifications" onClose={() => setOpen(false)} footer={markAllButton}>
          <div className="-mx-5 -mt-4">{list}</div>
        </Sheet>
      )}
      {open && !phone && (
        <div className="animate-scale-in absolute right-0 z-50 mt-2 flex max-h-[min(80dvh,calc(100dvh-6rem))] w-[min(22rem,calc(100vw-2rem))] origin-top-right flex-col overflow-hidden rounded-card border border-line bg-surface shadow-[var(--shadow-lift)]">
          <div className="flex items-center gap-2 border-b border-line px-4 py-3">
            <p className="font-display text-sm font-semibold text-ink">Notifications</p>
            {markAllButton}
          </div>
          <div className="min-h-0 flex-1 overscroll-contain overflow-y-auto">{list}</div>
        </div>
      )}
    </div>
  )
}

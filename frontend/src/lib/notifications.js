import { useEffect, useSyncExternalStore } from 'react'
import api, { hasSession } from './api'
import { refreshBadges } from './badges'
import {
  EMPTY_NOTIFICATIONS,
  pointsAtAnItem,
  received,
  unreadOnPage,
  withAllRead,
  withRead,
  withUnread,
} from './notificationState'
import { notificationTarget } from './paths'
import { readRaw } from './storage'

const POLL_INTERVAL = 30000

const LIST_LIMIT = 50

const legacySeenKey = (userId) => `sustcse.notifications.lastSeen.${userId}`

let state = EMPTY_NOTIFICATIONS

const listeners = new Set()

const set = (next) => {
  if (next === state) return
  state = next
  listeners.forEach((listener) => listener())
}

const subscribe = (listener) => {
  listeners.add(listener)
  return () => listeners.delete(listener)
}

const snapshot = () => state

export const loadNotifications = async () => {
  if (!hasSession()) return
  const data = await api.get('/api/notifications', { query: { limit: LIST_LIMIT } }).catch(() => null)
  if (data) set(received(data))
}

const send = async (path, body, before, optimistic) => {
  try {
    const answer = await api.post(path, body)
    set(withUnread(state, answer?.unread))
    refreshBadges()
    return true
  } catch {
    if (state === optimistic) set(before)
    else loadNotifications()
    return false
  }
}

export const markNotificationRead = (id) => {
  const before = state
  const optimistic = withRead(before, [id])
  if (optimistic === before) return Promise.resolve(true)
  set(optimistic)
  return send('/api/notifications/read', { id }, before, optimistic)
}

export const markPageRead = (page, role) => {
  if (!hasSession() || !pointsAtAnItem(page)) return Promise.resolve(false)
  const before = state
  if (before.loaded && before.unread === 0) return Promise.resolve(false)
  const ids = unreadOnPage(before, page, (item) => notificationTarget(item.link, role))
  if (before.loaded && ids.length === 0) return Promise.resolve(false)
  const optimistic = withRead(before, ids)
  set(optimistic)
  return send('/api/notifications/read', { page }, before, optimistic)
}

export const markAllNotificationsRead = (before) => {
  const earlier = state
  const optimistic = before ? earlier : withAllRead(earlier)
  set(optimistic)
  const body = before ? { before } : {}
  return send('/api/notifications/read-all', body, earlier, optimistic).then((done) => {
    if (done) loadNotifications()
    return done
  })
}

const forget = (key) => {
  try {
    window.localStorage.removeItem(key)
  } catch {
    return
  }
}

const carryOver = async (userId) => {
  const key = legacySeenKey(userId)
  const seen = readRaw(key)
  if (!seen) return
  const stamp = new Date(seen)
  if (Number.isNaN(stamp.getTime())) {
    forget(key)
    return
  }
  const done = await markAllNotificationsRead(stamp.toISOString())
  if (done) forget(key)
}

export const resetNotifications = () => set(EMPTY_NOTIFICATIONS)

export const useNotifications = () => useSyncExternalStore(subscribe, snapshot, snapshot)

export const useNotificationFeed = (userId) => {
  useEffect(() => {
    if (!userId) {
      resetNotifications()
      return undefined
    }
    let timer = 0
    const start = () => {
      window.clearInterval(timer)
      timer = window.setInterval(loadNotifications, POLL_INTERVAL)
    }
    carryOver(userId).then(loadNotifications)
    start()
    const onVisibility = () => {
      window.clearInterval(timer)
      if (document.visibilityState !== 'visible') return
      loadNotifications()
      start()
    }
    document.addEventListener('visibilitychange', onVisibility)
    return () => {
      window.clearInterval(timer)
      document.removeEventListener('visibilitychange', onVisibility)
    }
  }, [userId])
}

export const useFollowedLinks = (location, role, userId) => {
  const page = `${location.pathname}${location.search}`
  useEffect(() => {
    if (!userId) return
    markPageRead(page, role)
  }, [page, role, userId])
}

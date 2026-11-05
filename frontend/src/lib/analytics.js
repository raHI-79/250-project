import api from './api'
import { readRaw, writeRaw } from './storage'
import { APP_VERSION } from './version'
import { CLIENT_ANDROID, isNative } from './native'

const VISITOR_KEY = 'sustcse.visitor'

const SESSION_KEY = 'sustcse.visit'

const FLUSH_AFTER = 4000

const MAX_BATCH = 20

const random = () => {
  if (typeof crypto !== 'undefined' && crypto.randomUUID) return crypto.randomUUID()
  return `${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 10)}`
}

export const visitorId = () => {
  let id = readRaw(VISITOR_KEY)
  if (!id) {
    id = random()
    writeRaw(VISITOR_KEY, id)
  }
  return id
}

export const sessionId = () => {
  try {
    let id = sessionStorage.getItem(SESSION_KEY)
    if (!id) {
      id = random()
      sessionStorage.setItem(SESSION_KEY, id)
    }
    return id
  } catch {
    return visitorId()
  }
}

export const deviceKind = () => {
  if (typeof navigator === 'undefined') return 'unknown'
  const agent = navigator.userAgent || ''
  if (/iPad|Tablet/i.test(agent)) return 'tablet'
  if (/Mobi|Android|iPhone/i.test(agent)) return 'phone'
  return 'desktop'
}

let queue = []

let timer = 0

const send = async () => {
  timer = 0
  const items = queue.slice(0, MAX_BATCH)
  queue = queue.slice(items.length)
  if (items.length === 0) return
  try {
    await api.post(
      '/api/analytics/view',
      {
        visitor_id: visitorId(),
        session_id: sessionId(),
        client: isNative() ? CLIENT_ANDROID : 'web',
        device: deviceKind(),
        app_version: APP_VERSION,
        items,
      },
      { auth: false },
    )
  } catch {
    queue = []
  }
}

export const trackView = (path) => {
  const target = String(path || '')
  if (!target) return
  if (queue.length >= MAX_BATCH) return
  queue.push({ path: target })
  if (timer) return
  timer = window.setTimeout(send, FLUSH_AFTER)
}

export const flushViews = () => {
  if (timer) {
    window.clearTimeout(timer)
    timer = 0
  }
  return send()
}

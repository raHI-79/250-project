import { useEffect } from 'react'
import api, { hasSession } from './api'
import { readRaw, remove, writeRaw } from './storage'
import { isAndroidApp } from './native'

const DEVICE_KEY = 'sustcse.device'

const TOKEN_KEY = 'sustcse.fcm'

export const FCM_ROLES = ['student', 'cr', 'teacher']

export const deviceID = () => {
  const stored = readRaw(DEVICE_KEY)
  if (stored) return stored
  const fresh =
    typeof crypto !== 'undefined' && crypto.randomUUID
      ? crypto.randomUUID()
      : `d${Date.now()}${Math.random().toString(16).slice(2)}`
  writeRaw(DEVICE_KEY, fresh)
  return fresh
}

const report = (stage, error) => {
  if (!hasSession()) return
  api
    .post('/api/push/client-log', {
      stage,
      name: error?.name || 'Error',
      message: error?.message || String(error || ''),
    })
    .catch(() => null)
}

const postToken = (token) =>
  api.post('/api/push/fcm', { token, platform: 'android', device_id: deviceID() })

let started = false

const OPEN_EVENT = 'sustcse:open'

const REPEAT_WITHIN = 5000

let lastOpened = { url: '', at: 0 }

let pendingOpen = ''

let openHandler = null

const onNativeOpen = (event) => {
  const url = event?.detail?.url || event?.url || ''
  if (!url) return
  if (openHandler) openHandler(url)
  else pendingOpen = url
}

if (typeof window !== 'undefined') window.addEventListener(OPEN_EVENT, onNativeOpen)

const openTarget = (data, navigate) => {
  const url = String(data?.url || '')
  if (!url) return
  const now = Date.now()
  if (url === lastOpened.url && now - lastOpened.at < REPEAT_WITHIN) return
  lastOpened = { url, at: now }
  try {
    const target = url.startsWith('http') ? new URL(url) : null
    navigate(target ? `${target.pathname}${target.search}` : url)
  } catch {
    navigate('/')
  }
}

export const registerFCM = async (role, navigate) => {
  if (!isAndroidApp() || !FCM_ROLES.includes(role) || !hasSession()) return false
  const { PushNotifications } = await import('@capacitor/push-notifications')
  let status = await PushNotifications.checkPermissions()
  if (status.receive === 'prompt' || status.receive === 'prompt-with-rationale') {
    status = await PushNotifications.requestPermissions()
  }
  if (status.receive !== 'granted') return false
  if (!started) {
    started = true
    await PushNotifications.addListener('registration', (token) => {
      if (!token?.value || token.value === readRaw(TOKEN_KEY)) return
      postToken(token.value)
        .then(() => writeRaw(TOKEN_KEY, token.value))
        .catch((failure) => report('fcm-save', failure))
    })
    await PushNotifications.addListener('registrationError', (failure) => {
      report('fcm-register', failure)
    })
    await PushNotifications.addListener('pushNotificationActionPerformed', (action) => {
      openTarget(action?.notification?.data, navigate)
    })
  }
  await PushNotifications.register()
  return true
}

export const dropFCM = async () => {
  if (!isAndroidApp()) return
  const token = readRaw(TOKEN_KEY)
  remove(TOKEN_KEY)
  if (!token) return
  await api.delete('/api/push/fcm', { body: { token } }).catch(() => null)
  const { PushNotifications } = await import('@capacitor/push-notifications')
  await PushNotifications.unregister().catch(() => null)
}

export const useFCM = (role, navigate) => {
  useEffect(() => {
    if (!isAndroidApp() || !FCM_ROLES.includes(role)) return undefined
    let live = true
    const sync = () => {
      if (!live) return
      registerFCM(role, navigate).catch((failure) => report('fcm-start', failure))
    }
    sync()
    const onVisible = () => {
      if (document.visibilityState === 'visible') sync()
    }
    openHandler = (url) => {
      if (live) openTarget({ url }, navigate)
    }
    if (pendingOpen) {
      const waiting = pendingOpen
      pendingOpen = ''
      openHandler(waiting)
    }
    document.addEventListener('visibilitychange', onVisible)
    return () => {
      live = false
      openHandler = null
      document.removeEventListener('visibilitychange', onVisible)
    }
  }, [role, navigate])
}

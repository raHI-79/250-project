import { useEffect } from 'react'
import api from './api'
import { readRaw, remove, writeRaw } from './storage'

const STAGE_KEY = 'sustcse.push.ask'

const STAGE_SIGN_IN = 'sign-in'

const STAGE_DONE = 'done'

export const PUSH_ROLES = ['student', 'cr', 'teacher']

export const PERMISSION_GRANTED = 'granted'

export const PERMISSION_DENIED = 'denied'

export const PERMISSION_DEFAULT = 'default'

export const pushSupported = () =>
  typeof window !== 'undefined' &&
  'serviceWorker' in navigator &&
  'PushManager' in window &&
  'Notification' in window

export const pushAllowed = (role) => PUSH_ROLES.includes(role) && pushSupported()

const base64ToBytes = (value) => {
  const padded = value.padEnd(value.length + ((4 - (value.length % 4)) % 4), '=')
  const raw = window.atob(padded.replace(/-/g, '+').replace(/_/g, '/'))
  const bytes = new Uint8Array(raw.length)
  for (let index = 0; index < raw.length; index += 1) bytes[index] = raw.charCodeAt(index)
  return bytes
}

const sameKey = (subscription, key) => {
  const current = subscription.options?.applicationServerKey
  if (!current) return false
  const bytes = new Uint8Array(current)
  if (bytes.length !== key.length) return false
  return bytes.every((byte, index) => byte === key[index])
}

const registerWorker = () => navigator.serviceWorker.register('/sw.js', { scope: '/' })

const serverKey = async () => {
  const config = await api.get('/api/push/public-key').catch(() => null)
  if (!config?.enabled || !config?.public_key) return null
  return config.public_key
}

const postSubscription = (subscription) => {
  const payload = subscription.toJSON()
  return api.post('/api/push/subscribe', {
    endpoint: payload.endpoint,
    keys: { p256dh: payload.keys?.p256dh, auth: payload.keys?.auth },
    user_agent: navigator.userAgent,
  })
}

export const registerPushWorker = async () => {
  if (!pushSupported()) return null
  return registerWorker().catch(() => null)
}

export const syncPushSubscription = async () => {
  if (!pushSupported() || Notification.permission !== PERMISSION_GRANTED) return false
  const publicKey = await serverKey()
  if (!publicKey) return false
  const registration = await registerWorker()
  const key = base64ToBytes(publicKey)
  let subscription = await registration.pushManager.getSubscription()
  if (subscription && !sameKey(subscription, key)) {
    await subscription.unsubscribe().catch(() => null)
    subscription = null
  }
  if (!subscription) {
    subscription = await registration.pushManager.subscribe({
      userVisibleOnly: true,
      applicationServerKey: key,
    })
  }
  await postSubscription(subscription)
  return true
}

export const askPushPermission = async (role) => {
  if (!pushAllowed(role)) return PERMISSION_DENIED
  if (Notification.permission !== PERMISSION_DEFAULT) {
    writeRaw(STAGE_KEY, STAGE_DONE)
    await syncPushSubscription().catch(() => null)
    return Notification.permission
  }
  let permission = PERMISSION_DEFAULT
  try {
    permission = await Notification.requestPermission()
  } catch {
    permission = Notification.permission
  }
  writeRaw(STAGE_KEY, permission === PERMISSION_DEFAULT ? STAGE_SIGN_IN : STAGE_DONE)
  if (permission === PERMISSION_GRANTED) await syncPushSubscription().catch(() => null)
  return permission
}

export const retryAskPending = (role) =>
  pushAllowed(role) &&
  Notification.permission === PERMISSION_DEFAULT &&
  readRaw(STAGE_KEY) === STAGE_SIGN_IN

export const closePushAsk = () => writeRaw(STAGE_KEY, STAGE_DONE)

export const resetPushAsk = () => remove(STAGE_KEY)

export const usePushSync = (role) => {
  useEffect(() => {
    if (!pushSupported()) return undefined
    let live = true
    const sync = () => {
      if (!live || document.visibilityState !== 'visible') return
      registerPushWorker().then(() => {
        if (PUSH_ROLES.includes(role)) syncPushSubscription().catch(() => null)
      })
    }
    sync()
    document.addEventListener('visibilitychange', sync)
    return () => {
      live = false
      document.removeEventListener('visibilitychange', sync)
    }
  }, [role])
}

export const dropPushOnSignOut = async () => {
  if (!pushSupported()) return
  resetPushAsk()
  try {
    const registration = await navigator.serviceWorker.getRegistration('/sw.js')
    const subscription = await registration?.pushManager.getSubscription()
    if (!subscription) return
    await api
      .delete('/api/push/subscribe', { body: { endpoint: subscription.endpoint } })
      .catch(() => null)
    await subscription.unsubscribe()
  } catch {
    return
  }
}

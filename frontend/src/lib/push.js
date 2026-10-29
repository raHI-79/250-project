import { useEffect } from 'react'
import api, { hasSession } from './api'
import { PushKeyError, sameApplicationKey, urlBase64ToUint8Array } from './pushkey'
import { readRaw, remove, writeRaw } from './storage'

const STAGE_KEY = 'sustcse.push.ask'

const STAGE_ASKED = 'asked'

const WORKER_URL = '/sw.js'

const WORKER_SCOPE = '/'

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

const registerWorker = () => navigator.serviceWorker.register(WORKER_URL, { scope: WORKER_SCOPE })

export const registerPushWorker = async () => {
  if (!pushSupported()) return null
  try {
    return await registerWorker()
  } catch (error) {
    report('register', error)
    return null
  }
}

const serverKey = async () => {
  const config = await api.get('/api/push/public-key', { auth: false })
  if (!config?.enabled) {
    throw new PushKeyError('Notifications are switched off on the server.')
  }
  return urlBase64ToUint8Array(config.public_key)
}

const postSubscription = (subscription) => {
  const payload = subscription.toJSON()
  return api.post('/api/push/subscribe', {
    endpoint: payload.endpoint,
    keys: { p256dh: payload.keys?.p256dh, auth: payload.keys?.auth },
    user_agent: navigator.userAgent,
  })
}

const subscribeNow = async () => {
  const key = await serverKey()
  const registration = (await registerWorker()) || (await navigator.serviceWorker.ready)
  await navigator.serviceWorker.ready
  let subscription = await registration.pushManager.getSubscription()
  if (subscription && !sameApplicationKey(subscription, key)) {
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

let running = null

export const syncPushSubscription = async () => {
  if (!pushSupported() || !hasSession()) return false
  if (Notification.permission !== PERMISSION_GRANTED) return false
  if (!running) {
    running = subscribeNow()
      .catch((error) => {
        report('subscribe', error)
        return false
      })
      .finally(() => {
        running = null
      })
  }
  return running
}

export const askPushPermission = async (role) => {
  if (!pushAllowed(role)) return PERMISSION_DENIED
  if (Notification.permission === PERMISSION_GRANTED) {
    writeRaw(STAGE_KEY, STAGE_ASKED)
    await syncPushSubscription()
    return PERMISSION_GRANTED
  }
  if (Notification.permission === PERMISSION_DENIED) {
    writeRaw(STAGE_KEY, STAGE_ASKED)
    return PERMISSION_DENIED
  }
  let permission = PERMISSION_DEFAULT
  try {
    permission = await Notification.requestPermission()
  } catch (error) {
    report('permission', error)
    permission = Notification.permission
  }
  if (permission !== PERMISSION_DEFAULT) writeRaw(STAGE_KEY, STAGE_ASKED)
  if (permission === PERMISSION_GRANTED) await syncPushSubscription()
  return permission
}

export const retryAskPending = (role) =>
  pushAllowed(role) &&
  Notification.permission === PERMISSION_DEFAULT &&
  readRaw(STAGE_KEY) !== STAGE_ASKED

export const closePushAsk = () => writeRaw(STAGE_KEY, STAGE_ASKED)

export const resetPushAsk = () => remove(STAGE_KEY)

export const usePushSync = (role) => {
  useEffect(() => {
    if (!pushSupported()) return undefined
    let live = true
    const sync = () => {
      if (!live) return
      registerPushWorker().then(() => {
        if (!live || !PUSH_ROLES.includes(role)) return
        syncPushSubscription()
      })
    }
    sync()
    const onVisible = () => {
      if (document.visibilityState === 'visible') sync()
    }
    document.addEventListener('visibilitychange', onVisible)
    window.addEventListener('focus', onVisible)
    return () => {
      live = false
      document.removeEventListener('visibilitychange', onVisible)
      window.removeEventListener('focus', onVisible)
    }
  }, [role])
}

export const dropPushOnSignOut = async () => {
  if (!pushSupported()) return
  await (running || Promise.resolve())
  resetPushAsk()
  try {
    const registration = await navigator.serviceWorker.getRegistration(WORKER_SCOPE)
    const subscription = await registration?.pushManager.getSubscription()
    if (!subscription) return
    await api
      .delete('/api/push/subscribe', { body: { endpoint: subscription.endpoint } })
      .catch(() => null)
  } catch {
    return
  }
}

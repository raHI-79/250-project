import { useCallback, useEffect, useState } from 'react'
import api from './api'
import { readRaw, writeRaw } from './storage'

const DISMISS_KEY = 'sustcse.push.dismissed'

export const PUSH_ROLES = ['student', 'cr', 'teacher']

export const pushSupported = () =>
  typeof window !== 'undefined' &&
  'serviceWorker' in navigator &&
  'PushManager' in window &&
  'Notification' in window

const base64ToBytes = (value) => {
  const padded = value.padEnd(value.length + ((4 - (value.length % 4)) % 4), '=')
  const raw = window.atob(padded.replace(/-/g, '+').replace(/_/g, '/'))
  const bytes = new Uint8Array(raw.length)
  for (let index = 0; index < raw.length; index += 1) bytes[index] = raw.charCodeAt(index)
  return bytes
}

const register = () => navigator.serviceWorker.register('/sw.js')

export const usePush = (role) => {
  const allowed = PUSH_ROLES.includes(role) && pushSupported()
  const [enabled, setEnabled] = useState(false)
  const [ready, setReady] = useState(false)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const [dismissed, setDismissed] = useState(() => readRaw(DISMISS_KEY) === 'yes')
  useEffect(() => {
    if (!allowed) {
      setReady(true)
      return undefined
    }
    let live = true
    api
      .get('/api/push/public-key')
      .then(async (data) => {
        if (!data?.enabled) return
        const registration = await register()
        const existing = await registration.pushManager.getSubscription()
        if (live) setEnabled(Boolean(existing))
      })
      .catch(() => null)
      .finally(() => {
        if (live) setReady(true)
      })
    return () => {
      live = false
    }
  }, [allowed])
  const subscribe = useCallback(async () => {
    setBusy(true)
    setError('')
    try {
      const config = await api.get('/api/push/public-key')
      if (!config?.enabled || !config?.public_key) {
        throw new Error('Notifications are not set up on this server yet.')
      }
      const permission = await Notification.requestPermission()
      if (permission !== 'granted') {
        throw new Error('Your browser blocked notifications for this site.')
      }
      const registration = await register()
      const subscription =
        (await registration.pushManager.getSubscription()) ||
        (await registration.pushManager.subscribe({
          userVisibleOnly: true,
          applicationServerKey: base64ToBytes(config.public_key),
        }))
      const payload = subscription.toJSON()
      await api.post('/api/push/subscribe', {
        endpoint: payload.endpoint,
        keys: { p256dh: payload.keys?.p256dh, auth: payload.keys?.auth },
        user_agent: navigator.userAgent,
      })
      setEnabled(true)
      return true
    } catch (failure) {
      setError(failure.message)
      return false
    } finally {
      setBusy(false)
    }
  }, [])
  const unsubscribe = useCallback(async () => {
    setBusy(true)
    setError('')
    try {
      const registration = await register()
      const subscription = await registration.pushManager.getSubscription()
      if (subscription) {
        await api
          .delete('/api/push/subscribe', { body: { endpoint: subscription.endpoint } })
          .catch(() => null)
        await subscription.unsubscribe()
      }
      setEnabled(false)
      return true
    } catch (failure) {
      setError(failure.message)
      return false
    } finally {
      setBusy(false)
    }
  }, [])
  const dismiss = useCallback(() => {
    writeRaw(DISMISS_KEY, 'yes')
    setDismissed(true)
  }, [])
  return { allowed, enabled, ready, busy, error, dismissed, subscribe, unsubscribe, dismiss }
}

export const dropPushOnSignOut = async () => {
  if (!pushSupported()) return
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

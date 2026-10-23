const FALLBACK_TITLE = 'SUST CSE'

const ICON = '/icon-192.png'

const BADGE = '/badge-96.png'

const VIBRATE = [60, 40, 60]

self.addEventListener('install', () => self.skipWaiting())

self.addEventListener('activate', (event) => event.waitUntil(self.clients.claim()))

self.addEventListener('push', (event) => {
  let payload = {}
  try {
    payload = event.data ? event.data.json() : {}
  } catch {
    payload = {}
  }
  const title = payload.title || FALLBACK_TITLE
  const tag = payload.tag || 'sustcse'
  event.waitUntil(
    self.registration.showNotification(title, {
      body: payload.body || '',
      icon: ICON,
      badge: BADGE,
      tag,
      renotify: true,
      vibrate: VIBRATE,
      timestamp: Date.now(),
      data: { url: payload.url || '/' },
    }),
  )
})

self.addEventListener('notificationclick', (event) => {
  event.notification.close()
  const target = (event.notification.data && event.notification.data.url) || '/'
  event.waitUntil(
    self.clients.matchAll({ type: 'window', includeUncontrolled: true }).then((clientList) => {
      for (const client of clientList) {
        if ('focus' in client && 'navigate' in client) {
          return client.focus().then((focused) => (focused || client).navigate(target))
        }
      }
      return self.clients.openWindow(target)
    }),
  )
})

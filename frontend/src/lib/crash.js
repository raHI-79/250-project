export const CRASH_PATH = '/api/client-log'

const MAX_NAME = 80

const MAX_MESSAGE = 300

const MAX_ROUTE = 120

const clip = (value, limit) => String(value ?? '').replace(/\s+/g, ' ').trim().slice(0, limit)

export const crashRoute = (location) => {
  const path = String(location?.pathname || '/').split('?')[0].split('#')[0]
  return clip(path.startsWith('/') ? path : `/${path}`, MAX_ROUTE)
}

export const crashReport = (error, location, client, version) => ({
  name: clip(error?.name || 'Error', MAX_NAME),
  message: clip(error?.message || error || 'Unknown problem', MAX_MESSAGE),
  route: crashRoute(location),
  client: client === 'app' ? 'app' : 'web',
  version: clip(version, 20),
})

export const crashKey = (report) => `${report.route}|${report.name}|${report.message}`

export const createCrashReporter = (send) => {
  const seen = new Set()
  return (report) => {
    const key = crashKey(report)
    if (seen.has(key)) return false
    seen.add(key)
    try {
      Promise.resolve(send(report)).catch(() => null)
    } catch {
      return false
    }
    return true
  }
}

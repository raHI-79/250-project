const READ_METHODS = ['GET', 'HEAD', 'OPTIONS']

let version = 0

const listeners = new Set()

const QUIET_PATHS = [
  /^\/api\/analytics\//,
  /^\/api\/push\//,
  /^\/api\/client-log$/,
  /^\/api\/notifications\/read(-all)?$/,
  /^\/api\/me\/badges\/[^/]+\/seen$/,
  /^\/api\/teacher\/attendance\/sessions\/\d+\/location$/,
]

const pathOnly = (path) => String(path || '').split('?')[0]

export const isQuietPath = (path) => QUIET_PATHS.some((pattern) => pattern.test(pathOnly(path)))

export const shouldInvalidate = (method, path = '') =>
  !READ_METHODS.includes(String(method || '').toUpperCase()) && !isQuietPath(path)

export const resourceVersion = () => version

export const bumpResources = () => {
  version += 1
  listeners.forEach((listener) => listener(version))
  return version
}

export const watchResources = (listener) => {
  listeners.add(listener)
  return () => listeners.delete(listener)
}

export const afterRequest = (method, path = '') => {
  if (!shouldInvalidate(method, path)) return version
  return bumpResources()
}

export const resourceSource = (path, key) => (path ? `${path}?${key}` : '')

export const resourceView = (state, source, busy) => {
  const own = Boolean(source) && state.source === source
  const data = own ? state.data : null
  return {
    data,
    error: own ? state.error : '',
    status: own ? state.status || 0 : 0,
    loading: Boolean(source) && data === null && (busy || !own),
    refreshing: Boolean(source) && busy && data !== null,
  }
}

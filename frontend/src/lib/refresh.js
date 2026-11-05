const READ_METHODS = ['GET', 'HEAD', 'OPTIONS']

let version = 0

const listeners = new Set()

export const shouldInvalidate = (method) =>
  !READ_METHODS.includes(String(method || '').toUpperCase())

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

export const afterRequest = (method) => {
  if (!shouldInvalidate(method)) return version
  return bumpResources()
}

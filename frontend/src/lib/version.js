export const APP_VERSION = import.meta.env.VITE_APP_VERSION || '2.0.1'

export const APP_COMMIT = import.meta.env.VITE_APP_COMMIT || ''

export const shortCommit = (value) => String(value || '').slice(0, 7)

export const buildLabel = (commit) => {
  const short = shortCommit(commit || APP_COMMIT)
  return short ? `${APP_VERSION} · ${short}` : APP_VERSION
}

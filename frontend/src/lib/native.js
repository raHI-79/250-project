import { Capacitor } from '@capacitor/core'

export const API_BASE = 'https://sustclassexamcoordinationsystem.onrender.com'

export const CLIENT_HEADER = 'X-Client'

export const CLIENT_ANDROID = 'android'

export const LATEST_APK = 'https://github.com/raHI-79/250-project/releases/latest'

export const isNative = () => Capacitor.isNativePlatform()

export const nativePlatform = () => Capacitor.getPlatform()

export const isAndroidApp = () => isNative() && nativePlatform() === 'android'

export const apiURL = (path) => (isNative() ? `${API_BASE}${path}` : path)

export const assetURL = (path) => {
  const text = String(path || '')
  if (!text || !text.startsWith('/') || !isNative()) return text
  return `${API_BASE}${text}`
}

export const isApplePhone = () => {
  if (typeof navigator === 'undefined') return false
  const agent = navigator.userAgent || ''
  if (/iPad|iPhone|iPod/.test(agent)) return true
  return agent.includes('Mac') && typeof document !== 'undefined' && 'ontouchend' in document
}

export const showsAndroidDownload = () => !isNative() && !isApplePhone()

export const openExternal = async (url) => {
  const target = assetURL(url)
  if (!target) return
  if (!isNative()) {
    window.open(target, '_blank', 'noopener,noreferrer')
    return
  }
  const { Browser } = await import('@capacitor/browser')
  await Browser.open({ url: target, presentationStyle: 'popover' })
}

export const externalLinkProps = (url) => ({
  href: assetURL(url),
  target: '_blank',
  rel: 'noreferrer',
  onClick: (event) => {
    if (!isNative()) return
    event.preventDefault()
    openExternal(url)
  },
})

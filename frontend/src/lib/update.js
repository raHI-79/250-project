import { registerPlugin } from '@capacitor/core'
import api from './api'
import { isAndroidApp } from './native'
import { newerThan } from './semver'
import { readRaw, writeRaw } from './storage'
import { APP_VERSION } from './version'

const ApkInstaller = registerPlugin('ApkInstaller')

export const LATEST_PATH = '/api/app/android-latest'

export const CHECK_EVENT = 'sustcse.update-check'

const GUIDED_KEY = 'sustcse.installguide'

export const isNewer = (latest, installed = APP_VERSION) => newerThan(latest, installed)

export const fetchLatest = () => api.get(LATEST_PATH, { auth: false })

export const requestUpdateCheck = () => window.dispatchEvent(new Event(CHECK_EVENT))

export const guidedOnce = () => readRaw(GUIDED_KEY) === '1'

export const markGuided = () => writeRaw(GUIDED_KEY, '1')

export const apkName = (version) => `sust-cse-portal-${version || 'latest'}.apk`

export const downloadUpdate = async (url, version, onProgress) => {
  const { Directory, Filesystem } = await import('@capacitor/filesystem')
  const path = apkName(version)
  const listener = await Filesystem.addListener('progress', (status) => {
    const total = Number(status?.contentLength) || 0
    const bytes = Number(status?.bytes) || 0
    onProgress?.(total > 0 ? Math.min(100, Math.round((bytes / total) * 100)) : 0)
  })
  try {
    await Filesystem.deleteFile({ path, directory: Directory.Cache }).catch(() => null)
    const result = await Filesystem.downloadFile({
      url,
      path,
      directory: Directory.Cache,
      method: 'GET',
      progress: true,
    })
    return result?.path || ''
  } finally {
    await listener.remove()
  }
}

export const canInstall = async () => {
  const result = await ApkInstaller.canInstall()
  return Boolean(result?.allowed)
}

export const openInstallSettings = () => ApkInstaller.openSettings()

export const openInstaller = (path) => ApkInstaller.install({ path })

export const updatesSupported = () => isAndroidApp()

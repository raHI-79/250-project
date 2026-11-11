import { useEffect, useState } from 'react'
import api from './api'
import { newerThan } from './semver'
import { APP_VERSION } from './version'
import { isNative } from './native'

const REFRESH_AFTER = 120000

let cached = null

const listeners = new Set()

const announce = () => {
  listeners.forEach((listener) => listener(cached))
}

export const readSite = async () => {
  try {
    const data = await api.get('/api/auth/config', { auth: false })
    cached = data || {}
    announce()
  } catch {
    if (!cached) cached = {}
  }
  return cached
}

export const useSite = () => {
  const [state, setState] = useState(cached)
  useEffect(() => {
    listeners.add(setState)
    if (!cached) readSite()
    const timer = window.setInterval(readSite, REFRESH_AFTER)
    return () => {
      listeners.delete(setState)
      window.clearInterval(timer)
    }
  }, [])
  return state || {}
}

export const updateRequired = (site) => {
  if (!isNative()) return false
  const floor = String(site?.min_app_version || '').trim()
  if (!floor) return false
  return newerThan(floor, APP_VERSION)
}

export const announcementFor = (site, role) => {
  const text = String(site?.announcement || '').trim()
  if (!text) return null
  const roles = site?.announcement_roles || []
  if (roles.length > 0 && role && !roles.includes(role) && !roles.includes('all')) return null
  return { text, tone: site?.announcement_tone || 'info' }
}

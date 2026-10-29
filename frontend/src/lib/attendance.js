import { registerPlugin } from '@capacitor/core'
import { SESSION_KEY } from './api'
import { API_BASE, isAndroidApp } from './native'
import { readJson } from './storage'

const AttendanceBeacon = registerPlugin('AttendanceBeacon')

export const PASS_MARK = 50

export const DURATIONS = [5, 10, 15, 20, 30]

export const DEFAULT_MINUTES = 10

export const TEACHER_TAB_TAKE = 'take'

export const TEACHER_TAB_HISTORY = 'history'

export const TEACHER_TAB_CLASSES = 'classes'

export const sectionLabel = (section) => (section === 'ALL' ? 'A and B' : `Section ${section}`)

export const classLabel = (entry) =>
  `${entry.course_code} · ${entry.batch} · ${sectionLabel(entry.section)}`

export const belowPass = (percent) => Number(percent) < PASS_MARK

export const percentOf = (present, held) => {
  const total = Number(held) || 0
  if (total <= 0) return 0
  return Math.round((Number(present) || 0) * 100 / total)
}

export const classKey = (entry) =>
  `${entry.course_id}|${entry.batch}|${entry.section}`

export const beaconSupported = () => isAndroidApp()

export const beaconStatus = () => AttendanceBeacon.status()

export const askBeacon = () => AttendanceBeacon.ensure()

export const turnBluetoothOn = () => AttendanceBeacon.enableBluetooth()

export const startBeacon = (session) =>
  AttendanceBeacon.start({
    session_id: session.id,
    token_key: session.token_key || '',
    course: `${session.course_code} · ${session.batch} ${session.section}`,
    present: session.present || 0,
    roster: session.roster || 0,
    endpoint: `${API_BASE}/api/teacher/attendance/sessions/${session.id}/location`,
    bearer: sessionToken(),
  })

export const updateBeacon = (session) =>
  AttendanceBeacon.update({ present: session.present || 0, roster: session.roster || 0 })

export const stopBeacon = () => AttendanceBeacon.stop()

const sessionToken = () => readJson(SESSION_KEY)?.token || ''

const exportQuery = (filters) => {
  const search = new URLSearchParams()
  Object.entries(filters || {}).forEach(([key, value]) => {
    if (value === undefined || value === null || value === '') return
    search.set(key, String(value))
  })
  const text = search.toString()
  return text ? `?${text}` : ''
}

const nameFromHeader = (header, fallback) => {
  const match = /filename="([^"]+)"/.exec(String(header || ''))
  return match ? match[1] : fallback
}

export const downloadWorkbook = async (path, filters, fallback = 'attendance.xlsx') => {
  const url = `${isAndroidApp() ? API_BASE : ''}${path}${exportQuery(filters)}`
  const headers = { Authorization: `Bearer ${sessionToken()}` }
  if (isAndroidApp()) {
    headers['X-Client'] = 'android'
    const { Directory, Filesystem } = await import('@capacitor/filesystem')
    const saved = await Filesystem.downloadFile({
      url,
      path: fallback,
      directory: Directory.Documents,
      method: 'GET',
      headers,
    })
    return { path: saved?.path || '', name: fallback }
  }
  const response = await fetch(url, { headers })
  if (!response.ok) {
    let message = 'The workbook could not be built.'
    try {
      const body = await response.json()
      if (body?.error) message = body.error
    } catch {
      message = 'The workbook could not be built.'
    }
    throw new Error(message)
  }
  const name = nameFromHeader(response.headers.get('Content-Disposition'), fallback)
  const blob = await response.blob()
  const link = document.createElement('a')
  link.href = URL.createObjectURL(blob)
  link.download = name
  document.body.appendChild(link)
  link.click()
  link.remove()
  window.setTimeout(() => URL.revokeObjectURL(link.href), 2000)
  return { path: '', name }
}

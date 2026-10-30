import { registerPlugin } from '@capacitor/core'
import { SESSION_KEY } from './api'
import { API_BASE, isAndroidApp } from './native'
import { readJson } from './storage'
import { SERVICE_UUID } from './attendanceToken'

const AttendanceBeacon = registerPlugin('AttendanceBeacon')

const AttendanceMark = registerPlugin('AttendanceMark')

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

export const ATTENDANCE_SERVICE_UUID = SERVICE_UUID

export const startBeacon = (session) =>
  AttendanceBeacon.start({
    session_id: String(session.id),
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

export const askBeaconLocation = () => AttendanceBeacon.ensureLocation()

export const openAppSettings = () => AttendanceBeacon.openSettings()

export const openLocationSettings = () => AttendanceBeacon.openLocationSettings()

export const POSITION_FRESH_MS = 60000

export const positionState = (session, status, now) => {
  if (!status) return { label: 'Waiting', tone: 'idle' }
  if (!status.place) {
    return status.asked
      ? { label: 'Denied', tone: 'bad', fix: 'settings' }
      : { label: 'Not allowed yet', tone: 'bad', fix: 'ask' }
  }
  if (!status.location_on) return { label: 'Off', tone: 'bad', fix: 'location' }
  if (!session?.location_updated_at) return { label: 'Allowed · waiting for GPS', tone: 'idle' }
  const age = now.getTime() - new Date(session.location_updated_at).getTime()
  if (age > POSITION_FRESH_MS) return { label: 'Out of date', tone: 'warn' }
  const accuracy = Number(session.accuracy_m) || 0
  return {
    label: accuracy > 0 ? `Sending · accuracy ${accuracy} m` : 'Sending',
    tone: 'good',
  }
}

export const bluetoothState = (status) => {
  if (!status) return { label: 'Waiting', tone: 'idle' }
  if (!status.supported) {
    return { label: 'Not supported', tone: 'bad', reason: status.error || '' }
  }
  if (!status.bluetooth_on) {
    return { label: 'Off', tone: 'bad', fix: 'bluetooth', reason: 'Bluetooth is off.' }
  }
  if (!status.nearby) {
    return status.asked
      ? { label: 'Denied', tone: 'bad', fix: 'settings', reason: 'Nearby devices is not allowed.' }
      : { label: 'Not allowed yet', tone: 'bad', fix: 'ask' }
  }
  if (status.advertising) return { label: 'Broadcasting', tone: 'good' }
  return { label: 'Not broadcasting', tone: 'warn', reason: status.error || '' }
}

export const markSupported = () => isAndroidApp()

export const phoneId = async () => {
  if (!isAndroidApp()) return ''
  const result = await AttendanceMark.deviceId()
  return result?.device_id || ''
}

export const markStatus = () => AttendanceMark.status()

export const askMark = () => AttendanceMark.ensure()

export const turnMarkBluetoothOn = () => AttendanceMark.enableBluetooth()

export const listenForToken = (seconds = 10) => AttendanceMark.scan({ seconds })

export const readPosition = () => AttendanceMark.position()

export const confirmIdentity = () => AttendanceMark.confirm()

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

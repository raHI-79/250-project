import { inflateRawSync } from 'node:zlib'

const API = process.env.E2E_API_URL || 'http://localhost:8080'

const tokens = new Map()

const RATE_LIMITED = 429

const RATE_WINDOW = 61_000

const pause = (milliseconds) => new Promise((resolve) => setTimeout(resolve, milliseconds))

export async function signInByApi(role, portal) {
  const key = `${portal}:${role.identifier}`
  if (tokens.has(key)) return tokens.get(key)
  for (let attempt = 0; attempt < 3; attempt += 1) {
    const response = await fetch(`${API}/api/auth/login`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ identifier: role.identifier, password: role.password, portal }),
    })
    if (response.status === RATE_LIMITED) {
      await pause(RATE_WINDOW)
      continue
    }
    const body = await response.json()
    if (!response.ok) throw new Error(`could not sign in as ${role.label}: ${body.error || response.status}`)
    tokens.set(key, body.token)
    return body.token
  }
  throw new Error(`could not sign in as ${role.label}: the sign-in stayed rate limited`)
}

export async function call(token, method, path, body) {
  const form = typeof FormData !== 'undefined' && body instanceof FormData
  const response = await fetch(`${API}${path}`, {
    method,
    headers: {
      Authorization: `Bearer ${token}`,
      ...(body && !form ? { 'Content-Type': 'application/json' } : {}),
    },
    body: body ? (form ? body : JSON.stringify(body)) : undefined,
  })
  const text = await response.text()
  let data = {}
  try {
    data = text ? JSON.parse(text) : {}
  } catch {
    data = { raw: text }
  }
  return { status: response.status, ok: response.ok, data }
}

export async function need(token, method, path, body) {
  const answer = await call(token, method, path, body)
  if (!answer.ok) {
    throw new Error(`${method} ${path} answered ${answer.status}: ${answer.data.error || ''}`)
  }
  return answer.data
}

export async function classOf(teacherToken, batch, section) {
  const options = await need(teacherToken, 'GET', '/api/teacher/attendance/options')
  const all = (options.today || []).concat(options.classes || [])
  const found = all.find(
    (entry) => entry.batch === batch && (entry.section === section || entry.section === 'ALL'),
  )
  if (!found) throw new Error(`the teacher has no class for ${batch} ${section}`)
  return found
}

export async function clearToday(teacherToken, entry) {
  const history = await need(teacherToken, 'GET', '/api/teacher/attendance/sessions')
  const today = new Date().toDateString()
  for (const session of history.items || []) {
    const same =
      session.course_id === entry.course_id &&
      session.batch === entry.batch &&
      session.section === entry.section
    if (!same || new Date(session.started_at).toDateString() !== today) continue
    if (session.status === 'open') {
      await call(teacherToken, 'POST', `/api/teacher/attendance/sessions/${session.id}/close`)
    }
    await call(teacherToken, 'DELETE', `/api/teacher/attendance/sessions/${session.id}`)
  }
  const options = await need(teacherToken, 'GET', '/api/teacher/attendance/options')
  if (options.open) {
    await call(teacherToken, 'POST', `/api/teacher/attendance/sessions/${options.open.id}/close`)
    await call(teacherToken, 'DELETE', `/api/teacher/attendance/sessions/${options.open.id}`)
  }
}

export async function workbookEntry(teacherToken, query, name) {
  const response = await fetch(`${API}/api/teacher/attendance/export?${query}`, {
    headers: { Authorization: `Bearer ${teacherToken}` },
  })
  if (!response.ok) throw new Error(`the export answered ${response.status}`)
  const buffer = Buffer.from(await response.arrayBuffer())
  const end = buffer.lastIndexOf(Buffer.from([0x50, 0x4b, 0x05, 0x06]))
  if (end < 0) throw new Error('the export is not a workbook')
  const total = buffer.readUInt16LE(end + 10)
  let offset = buffer.readUInt32LE(end + 16)
  for (let index = 0; index < total; index += 1) {
    const method = buffer.readUInt16LE(offset + 10)
    const size = buffer.readUInt32LE(offset + 20)
    const nameLength = buffer.readUInt16LE(offset + 28)
    const extraLength = buffer.readUInt16LE(offset + 30)
    const commentLength = buffer.readUInt16LE(offset + 32)
    const local = buffer.readUInt32LE(offset + 42)
    const entry = buffer.toString('utf8', offset + 46, offset + 46 + nameLength)
    if (entry === name) {
      const start = local + 30 + buffer.readUInt16LE(local + 26) + buffer.readUInt16LE(local + 28)
      const raw = buffer.subarray(start, start + size)
      return (method === 8 ? inflateRawSync(raw) : raw).toString('utf8')
    }
    offset += 46 + nameLength + extraLength + commentLength
  }
  throw new Error(`the workbook has no ${name}`)
}

let chief = ''

export async function controlToken(role) {
  if (chief) return chief
  const response = await fetch(`${API}/api/auth/control-room/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ identifier: role.identifier, password: role.password }),
  })
  const body = await response.json()
  if (!response.ok) throw new Error(`could not sign in to the control room: ${body.error || response.status}`)
  await need(body.token, 'POST', '/api/office/security/confirm', { password: role.password })
  chief = body.token
  return chief
}

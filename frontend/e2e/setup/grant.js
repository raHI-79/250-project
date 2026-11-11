import { ROLES } from '../fixtures/roles.js'

const API = process.env.E2E_API_URL || 'http://localhost:8080'

async function call(path, { method = 'GET', token, body } = {}) {
  const response = await fetch(`${API}${path}`, {
    method,
    headers: {
      'Content-Type': 'application/json',
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
    },
    body: body ? JSON.stringify(body) : undefined,
  })
  const text = await response.text()
  let payload = {}
  try {
    payload = text ? JSON.parse(text) : {}
  } catch {
    payload = { raw: text }
  }
  if (!response.ok) {
    throw new Error(`${method} ${path} answered ${response.status}: ${payload.error || text}`)
  }
  return payload
}

export default async function grantOfficePermissions() {
  const chief = ROLES.control
  const office = ROLES.office
  const { token } = await call('/api/auth/control-room/login', {
    method: 'POST',
    body: { identifier: chief.identifier, password: chief.password },
  })
  await call('/api/office/security/confirm', {
    method: 'POST',
    token,
    body: { password: chief.password },
  })
  const choices = await call('/api/super-admin/permissions', { token })
  const every = (choices.items || []).map((entry) => entry.value)
  const admins = await call('/api/super-admin/admins', { token })
  const account = (admins.items || []).find((entry) => entry.username === office.identifier)
  if (!account) {
    throw new Error(`no office account called ${office.identifier} exists to grant permissions to`)
  }
  await call(`/api/super-admin/admins/${account.id}/permissions`, {
    method: 'PUT',
    token,
    body: { permissions: every },
  })
}

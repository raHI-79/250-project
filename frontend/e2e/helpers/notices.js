import { ROLES } from '../fixtures/roles.js'

const API = process.env.E2E_API_URL || 'http://localhost:8080'

async function token(role) {
  const response = await fetch(`${API}/api/auth/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ identifier: role.identifier, password: role.password, portal: 'student' }),
  })
  const body = await response.json()
  if (!response.ok) throw new Error(`could not sign in as ${role.label}: ${body.error || response.status}`)
  return body.token
}

export async function postNotice(title) {
  const bearer = await token(ROLES.cr)
  const form = new FormData()
  form.set('title', title)
  form.set('description', 'Posted by the phone suite to follow a notice link.')
  form.set('target', 'own')
  const response = await fetch(`${API}/api/cr/notices`, {
    method: 'POST',
    headers: { Authorization: `Bearer ${bearer}` },
    body: form,
  })
  const body = await response.json()
  if (!response.ok) throw new Error(`could not post a notice: ${body.error || response.status}`)
  const list = await fetch(`${API}/api/cr/notices`, { headers: { Authorization: `Bearer ${bearer}` } })
  const items = (await list.json()).items || []
  const found = items.find((entry) => entry.title === title)
  if (!found) throw new Error('the posted notice is not in the list')
  return { id: found.id, bearer }
}

export async function removeNotice(notice) {
  await fetch(`${API}/api/cr/notices/${notice.id}`, {
    method: 'DELETE',
    headers: { Authorization: `Bearer ${notice.bearer}` },
  }).catch(() => null)
}

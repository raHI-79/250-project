import { expect, test } from '@playwright/test'
import { ROLES } from '../fixtures/roles.js'

const API = process.env.E2E_API_URL || 'http://localhost:8080'

const PASSWORD = 'a-good-password'

const RATE_WINDOW = 61_000

const CLOSED_TEXT = 'Your batch is not active. Please contact the department office.'

const tail = (length) =>
  String(Date.now() + Math.floor(Math.random() * 1000)).slice(-length).padStart(length, '0')

async function call(path, { method = 'GET', token, body, form } = {}) {
  const headers = token ? { Authorization: `Bearer ${token}` } : {}
  if (body) headers['Content-Type'] = 'application/json'
  const build = () => {
    if (!form) return body ? JSON.stringify(body) : undefined
    const fields = new FormData()
    Object.entries(form).forEach(([key, value]) => fields.append(key, value))
    return fields
  }
  const payload = build()
  let response = await fetch(`${API}${path}`, { method, headers, body: payload })
  for (let attempt = 0; attempt < 2 && response.status === 429; attempt += 1) {
    await new Promise((resolve) => setTimeout(resolve, RATE_WINDOW))
    response = await fetch(`${API}${path}`, { method, headers, body: build() })
  }
  const text = await response.text()
  let data = {}
  try {
    data = text ? JSON.parse(text) : {}
  } catch {
    data = { raw: text }
  }
  return { status: response.status, data }
}

async function officeToken() {
  const office = ROLES.office
  const answer = await call('/api/auth/admin/login', {
    method: 'POST',
    body: { identifier: office.identifier, password: office.password },
  })
  expect(answer.status, 'the office admin could not sign in').toBe(200)
  return answer.data.token
}

async function openBatch(token, name) {
  const created = await call('/api/admin/batches', {
    method: 'POST',
    token,
    body: { name, running_semester: '1-1' },
  })
  expect([201, 409], `${name} could not be added`).toContain(created.status)
  if (created.status === 409) {
    const activated = await call(`/api/admin/batches/${name}/activate`, { method: 'POST', token })
    expect([200, 409], `${name} could not be activated`).toContain(activated.status)
  }
}

function registration(regNo, extra = {}) {
  return {
    full_name: 'Browser Check Student',
    reg_no: regNo,
    section: 'A',
    email: `e2e${regNo}@student.sust.edu`,
    phone: `017${regNo.slice(-8)}`,
    blood_group: 'O+',
    date_of_birth: '2004-03-12',
    password: PASSWORD,
    confirm_password: PASSWORD,
    ...extra,
  }
}

async function registerAndApprove(token, regNo, extra) {
  const registered = await call('/api/auth/register', { method: 'POST', form: registration(regNo, extra) })
  expect(registered.status, `registering ${regNo}: ${registered.data.error || ''}`).toBe(201)
  if (registered.data.dev_code) {
    const verified = await call('/api/auth/verify-email', {
      method: 'POST',
      body: { reg_no: regNo, code: registered.data.dev_code },
    })
    expect(verified.status, `verifying ${regNo}: ${verified.data.error || ''}`).toBe(200)
  }
  const waiting = await call('/api/admin/registrations', { token })
  const entry = (waiting.data.items || []).find((item) => item.reg_no === regNo)
  expect(entry, `${regNo} is not waiting for approval`).toBeTruthy()
  const approved = await call(`/api/admin/registrations/${entry.user_id}/approve`, {
    method: 'POST',
    token,
  })
  expect(approved.status, `approving ${regNo}: ${approved.data.error || ''}`).toBe(200)
  return entry
}

const signIn = (regNo) =>
  call('/api/auth/login', { method: 'POST', body: { identifier: regNo, password: PASSWORD } })

test.describe('registration', () => {
  test('fills the batch in from the registration number', async ({ page }) => {
    await page.goto('/student/register', { waitUntil: 'domcontentloaded' })
    const regNo = page.getByLabel('Registration number').first()
    await regNo.waitFor({ state: 'visible', timeout: 30_000 })
    const batch = page.getByLabel('Batch', { exact: true }).first()
    for (const [typed, derived] of [
      ['2021331001', 'CSE-21'],
      ['2023331079', 'CSE-23'],
      ['202533107912', 'CSE-25'],
    ]) {
      await regNo.fill(typed)
      await expect(batch, `${typed} should give ${derived}`).toHaveValue(derived)
    }
    await expect(batch).not.toBeEditable()
    await regNo.fill('2099331001')
    await expect(
      page.getByText('CSE-99 is not open for registration. Please contact the department office.').first(),
    ).toBeVisible()
  })

  test('takes 10 and 12 digit numbers and refuses a deactivated batch at sign-in', async () => {
    test.skip(test.info().project.name !== 'desktop-light', 'the accounts are created once')
    const token = await officeToken()
    await openBatch(token, 'CSE-41')
    await openBatch(token, 'CSE-42')
    const ten = `2041${tail(6)}`
    const twelve = `2042${tail(8)}`
    const first = await registerAndApprove(token, ten, { batch: 'CSE-23' })
    expect(first.batch, 'the batch a client sends is ignored').toBe('CSE-41')
    const second = await registerAndApprove(token, twelve)
    expect(second.batch).toBe('CSE-42')
    for (const regNo of [ten, twelve]) {
      const answer = await signIn(regNo)
      expect(answer.status, `${regNo} could not sign in: ${answer.data.error || ''}`).toBe(200)
      expect(answer.data.user.batch).toBe(regNo === ten ? 'CSE-41' : 'CSE-42')
    }
    const unknown = await call('/api/auth/register', {
      method: 'POST',
      form: registration(`2098${tail(6)}`),
    })
    expect(unknown.status).toBe(400)
    expect(unknown.data.error).toBe(
      'CSE-98 is not open for registration. Please contact the department office.',
    )
    const closed = await call('/api/admin/batches/CSE-42/deactivate', { method: 'POST', token })
    expect(closed.status, `deactivating CSE-42: ${closed.data.error || ''}`).toBe(200)
    try {
      const refused = await signIn(twelve)
      expect(refused.status).toBe(403)
      expect(refused.data.error).toBe(CLOSED_TEXT)
      const lookups = await call('/api/lookups/batches', { token })
      expect(lookups.data.items).not.toContain('CSE-42')
      expect(lookups.data.items).toContain('CSE-41')
      const still = await signIn(ten)
      expect(still.status).toBe(200)
    } finally {
      await call('/api/admin/batches/CSE-42/activate', { method: 'POST', token })
    }
    const again = await signIn(twelve)
    expect(again.status, 'an activated batch signs in again').toBe(200)
  })
})

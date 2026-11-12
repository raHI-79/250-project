import { expect } from '@playwright/test'

export async function signIn(page, role) {
  await page.goto(role.loginPath, { waitUntil: 'domcontentloaded' })
  const identifier = page
    .locator('input[autocomplete="username"], input[name="identifier"]')
    .first()
  await identifier.waitFor({ state: 'visible', timeout: 30_000 })
  await identifier.fill(role.identifier)
  const password = page.locator('input[type="password"]').first()
  await password.fill(role.password)
  for (let attempt = 0; attempt < 3; attempt += 1) {
    const answer = page.waitForResponse(
      (response) => response.url().includes('/api/auth/') && response.request().method() === 'POST',
      { timeout: 30_000 },
    )
    await page.locator('button[type="submit"]').first().click()
    const response = await answer.catch(() => null)
    if (!response || response.status() !== 429) break
    await page.waitForTimeout(61_000)
  }
  await expect(password, `${role.label} was not signed in`).toBeHidden({ timeout: 30_000 })
  await expect(page, `${role.label} did not reach ${role.landing}`).toHaveURL(
    new RegExp(role.landing.replace(/\//g, '\\/')),
    { timeout: 30_000 },
  )
}

const DISMISS = [/^Hide this$/i, /^Not now$/i, /^Dismiss$/i]

async function dismissPrompts(page) {
  for (const name of DISMISS) {
    const control = page.getByRole('button', { name }).locator('visible=true').first()
    if ((await control.count()) === 0) continue
    await control.click({ timeout: 5_000 }).catch(() => {})
    await page.waitForTimeout(150)
  }
}

export async function visit(page, path) {
  await page.goto(path, { waitUntil: 'domcontentloaded' })
  await page.waitForLoadState('networkidle', { timeout: 30_000 }).catch(() => {})
  await dismissPrompts(page)
  await expect
    .poll(
      async () =>
        page.evaluate(() => {
          const main = document.querySelector('main') || document.body
          const text = (main.innerText || '').trim()
          const loading = main.querySelectorAll('.shimmer-bg').length
          return loading === 0 && text.length > 20 ? text.length : 0
        }),
      {
        message: `${path} never finished loading anything to read`,
        timeout: 25_000,
        intervals: [200, 300, 500, 1000],
      },
    )
    .toBeGreaterThan(20)
  await page.waitForTimeout(200)
}

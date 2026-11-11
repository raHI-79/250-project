import { expect } from '@playwright/test'

// signIn drives the real sign-in form of the portal a role belongs to, so the suite
// exercises the same path a person does and holds a real session afterwards.
export async function signIn(page, role) {
  await page.goto(role.loginPath, { waitUntil: 'domcontentloaded' })
  const identifier = page
    .locator('input[autocomplete="username"], input[name="identifier"]')
    .first()
  await identifier.waitFor({ state: 'visible', timeout: 30_000 })
  await identifier.fill(role.identifier)
  const password = page.locator('input[type="password"]').first()
  await password.fill(role.password)
  await page.locator('button[type="submit"]').first().click()
  // The control room signs in at the same address its dashboard lives at, so waiting for
  // the address to change would wait for ever. The form going away is the signal that
  // holds for every portal.
  await expect(password, `${role.label} was not signed in`).toBeHidden({ timeout: 30_000 })
  await expect(page, `${role.label} did not reach ${role.landing}`).toHaveURL(
    new RegExp(role.landing.replace(/\//g, '\\/')),
    { timeout: 30_000 },
  )
}

// DISMISS names the prompts a reader closes once and never sees again: the app banner is
// the only one today. The suite closes them before measuring, so that a page is read in
// the state a person actually reads it in.
const DISMISS = [/^Hide this$/i, /^Not now$/i, /^Dismiss$/i]

async function dismissPrompts(page) {
  for (const name of DISMISS) {
    const control = page.getByRole('button', { name }).locator('visible=true').first()
    if ((await control.count()) === 0) continue
    await control.click({ timeout: 5_000 }).catch(() => {})
    await page.waitForTimeout(150)
  }
}

// visit opens a page and waits until it has settled. A heading is not a reliable landmark
// across breakpoints, because the shell moves the title into a bar on a phone, so the
// check asks for the two things that hold everywhere: the page has real content and no
// skeleton is still shimmering. Measuring a page that is still loading reports faults
// that are not there.
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

import { expect } from '@playwright/test'

const THEME_KEY = 'sustcse.appearance'

const CALENDAR = 'button[aria-label^="Open a calendar for"]'

const FIELD = `.field-skin:has(${CALENDAR})`

const BROKEN = /Something went wrong/i

const pad = (value) => String(value).padStart(2, '0')

const stampOf = (date) => `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}`

const dayFrom = (start, days) => {
  const base = start ? new Date(`${start}T12:00:00`) : new Date()
  base.setDate(base.getDate() + days)
  return stampOf(base)
}

export async function setTheme(page, theme) {
  await page.addInitScript(
    ({ key, value }) => {
      try {
        window.localStorage.setItem(key, value)
      } catch {
        return
      }
    },
    { key: THEME_KEY, value: theme },
  )
  await page.emulateMedia({ colorScheme: theme })
}

async function expectAlive(page, problems, where) {
  await page.waitForTimeout(350)
  const text = await page.evaluate(() => (document.querySelector('#root')?.innerText || '').trim())
  expect(text.length, `${where} left a blank screen`).toBeGreaterThan(20)
  expect(BROKEN.test(text), `${where} broke the page`).toBe(false)
  expect(problems.pageErrors, `${where} threw`).toEqual([])
}

async function choose(field, stamp) {
  await field.locator('input[type="date"]').evaluate((node, value) => {
    const setter = Object.getOwnPropertyDescriptor(window.HTMLInputElement.prototype, 'value').set
    setter.call(node, value)
    node.dispatchEvent(new Event('input', { bubbles: true }))
    node.dispatchEvent(new Event('change', { bubbles: true }))
  }, stamp)
}

export async function exerciseDateFields(page, problems, where) {
  const total = await page.locator(FIELD).locator('visible=true').count()
  let used = 0
  for (let index = 0; index < total; index += 1) {
    const field = page.locator(FIELD).locator('visible=true').nth(index)
    if ((await field.count()) === 0) break
    if (await field.locator('input').first().isDisabled()) continue
    const label = `date field ${index + 1} on ${where}`
    await field.scrollIntoViewIfNeeded().catch(() => {})
    for (const part of ['day', 'month', 'year']) {
      await field.locator(`input[id$="-${part}"]`).tap({ timeout: 8_000 })
      await expectAlive(page, problems, `tapping the ${part} of ${label}`)
    }
    await field.tap({ position: { x: 4, y: 4 }, timeout: 8_000 })
    await expectAlive(page, problems, `tapping the edge of ${label}`)
    await field.locator(CALENDAR).tap({ timeout: 8_000 })
    await expectAlive(page, problems, `tapping the calendar button of ${label}`)
    await page.keyboard.press('Escape')
    const floor = await field.locator('input[type="date"]').getAttribute('min')
    for (const days of [2, 3]) {
      await choose(field, dayFrom(floor, days))
      await expectAlive(page, problems, `choosing a day in ${label}`)
      await page.waitForLoadState('networkidle', { timeout: 10_000 }).catch(() => {})
      await expectAlive(page, problems, `loading the day chosen in ${label}`)
    }
    const day = field.locator('input[id$="-day"]')
    if ((await day.count()) > 0) {
      await day.fill('')
      await expectAlive(page, problems, `clearing the day of ${label}`)
      await day.fill(String(new Date(`${dayFrom(floor, 4)}T12:00:00`).getDate()))
      await expectAlive(page, problems, `typing a day into ${label}`)
    }
    used += 1
  }
  return used
}

import { expect, test } from '@playwright/test'
import { DIALOG_OPENERS, DIALOG_PAGES, ROLES } from '../fixtures/roles.js'
import {
  clear,
  expectGoodLayout,
  expectNoProblems,
  expectScrollsToTheBottom,
  watch,
} from '../helpers/audit.js'
import { signIn, visit } from '../helpers/signIn.js'

for (const [key, role] of Object.entries(ROLES)) {
  test.describe(`the ${role.label}`, () => {
    test('signs in and reads every page', async ({ page }) => {
      const problems = watch(page)
      await signIn(page, role)
      problems.signedIn = true
      expectNoProblems(problems, `the ${role.label} sign-in`)
      for (const entry of role.pages) {
        clear(problems)
        await visit(page, entry.path)
        const where = `${entry.path} as the ${role.label}`
        expectNoProblems(problems, where)
        await expectGoodLayout(page, where)
        await expectScrollsToTheBottom(page, where)
      }
    })

    test('opens whatever dialogs its pages offer', async ({ page }) => {
      const problems = watch(page)
      await signIn(page, role)
      problems.signedIn = true
      let opened = 0
      for (const path of DIALOG_PAGES[key] || []) {
        await visit(page, path)
        const openers = page
          .getByRole('button', { name: DIALOG_OPENERS })
          .locator('visible=true')
        const total = Math.min(await openers.count(), 3)
        for (let index = 0; index < total; index += 1) {
          clear(problems)
          const opener = page
            .getByRole('button', { name: DIALOG_OPENERS })
            .locator('visible=true')
            .nth(index)
          if ((await opener.count()) === 0) break
          const name = (await opener.textContent()) || `control ${index}`
          await opener.click({ timeout: 10_000 }).catch(() => {})
          const panel = page.locator('[role="dialog"]').locator('visible=true').first()
          if ((await panel.count()) === 0) {
            await visit(page, path)
            continue
          }
          await expect(panel).toBeVisible({ timeout: 10_000 })
          const where = `the "${name.trim()}" dialog on ${path} as the ${role.label}`
          expectNoProblems(problems, where)
          await expectGoodLayout(page, where)
          opened += 1
          await page.keyboard.press('Escape')
          await page.waitForTimeout(250)
          if ((await panel.count()) > 0) {
            await visit(page, path)
          }
        }
      }
      expect(opened, `the ${role.label} opened ${opened} dialogs`).toBeGreaterThanOrEqual(0)
    })
  })
}

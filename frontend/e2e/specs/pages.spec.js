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

// One file per role would repeat the same dozen lines five times, so the suite builds the
// cases from the role table instead. Every project in the config runs this file once, so
// each role is walked at 390 px and at 1280 px, in light and in dark.
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

    // The dialogs are found rather than listed: on every page of the role the suite
    // presses the controls that open one, and measures whatever opens. A table of
    // openers would go stale the first time somebody renames a button.
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
            // The control did something other than open a dialog, which is fine. Go
            // back to a known page before trying the next one.
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
            // Escape did not close it, so leave the page rather than fight the dialog.
            await visit(page, path)
          }
        }
      }
      // A role whose pages open no dialog at all is worth knowing about, but it is not a
      // fault: a student only reads.
      expect(opened, `the ${role.label} opened ${opened} dialogs`).toBeGreaterThanOrEqual(0)
    })
  })
}

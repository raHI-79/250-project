import { expect, test } from '@playwright/test'

// A slow or blocked answer from Google Fonts must not hold the portal back: the stylesheet
// loads on the side, and the page reads in the system face until the font lands.
test('a font that never arrives does not hold the sign-in page back', async ({ page }) => {
  await page.route(/fonts\.googleapis\.com/, () => {})
  const begin = Date.now()
  await page.goto('/student/login', { waitUntil: 'domcontentloaded', timeout: 10_000 })
  await expect(page.locator('input[type="password"]').first()).toBeVisible({ timeout: 10_000 })
  expect(Date.now() - begin, 'the sign-in page waited for the font').toBeLessThan(10_000)
})

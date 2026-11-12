import { expect } from '@playwright/test'
import { expectGoodLayout, expectScrollsToTheBottom, settleAnimations } from './audit.js'

export const ROWS_PER_PAGE = 2
export const BUTTONS_PER_PAGE = 4
export const LINKS_PER_PAGE = 2

const SKIP =
  /(delete|remove|suspend|block|unblock|sign ?out|log ?out|close attendance|approve|reject|mark all|start|submit|save|send|publish|upload|download|export|excel|install|update|refresh|retry|try again|allow|turn |use this phone|request to use|present|absent|extend|vote|accept|decline|leave|copy|share|print|clear|reset|resend|restore|archive|unarchive|grant|revoke|dismiss|hide this|not now|log in|sign in|switch|next|previous|back|today|tomorrow|listen|mark me|i am present|enter the code|show code|hide code|read more|show more|load more|see all|pin|unpin|mute|unmute|logout|regenerate|rotate|test|run|retire|enable|disable|activate|deactivate)/i

const OPENER = /^(open|view|edit|add|new|create|invite|change|set |reschedule|cancel|details|members|attach|crop|permissions|profile|search|preview|request|report|write|compose|plan|propose|choose|pick|filter|sort|manage|holidays?)/i

export const DIALOG = '[role="dialog"][aria-modal="true"]'

export async function collectOpeners(page) {
  return page.evaluate(
    ({ rowsCap, buttonsCap, linksCap, skip, opener }) => {
      const skipRe = new RegExp(skip, 'i')
      const openerRe = new RegExp(opener, 'i')
      document.querySelectorAll('[data-e2e-opener]').forEach((node) => node.removeAttribute('data-e2e-opener'))
      const main = document.querySelector('main') || document.body
      const visible = (node) => {
        const box = node.getBoundingClientRect()
        const style = window.getComputedStyle(node)
        return box.width > 0 && box.height > 0 && style.visibility !== 'hidden' && style.display !== 'none'
      }
      const nameOf = (node) =>
        (node.getAttribute('aria-label') || node.innerText || node.textContent || '')
          .replace(/\s+/g, ' ')
          .trim()
      const usable = (node) =>
        visible(node) &&
        !node.disabled &&
        node.getAttribute('aria-disabled') !== 'true' &&
        node.getAttribute('role') !== 'tab' &&
        node.getAttribute('role') !== 'radio' &&
        !node.closest('[role="tablist"], [role="radiogroup"], form') &&
        !skipRe.test(nameOf(node))
      const picked = []
      const seen = new Set()
      const take = (node, kind) => {
        if (seen.has(node)) return
        seen.add(node)
        picked.push({ node, kind, name: nameOf(node).slice(0, 120) })
      }
      const rows = [...main.querySelectorAll('li > button, li > a[href], [role="listitem"] > button')]
        .filter(usable)
        .slice(0, rowsCap)
      rows.forEach((node) => take(node, 'row'))
      const buttons = [...main.querySelectorAll('button')]
        .filter((node) => usable(node) && !seen.has(node))
        .filter((node) => openerRe.test(nameOf(node)) || node.getAttribute('aria-haspopup') === 'dialog')
        .slice(0, buttonsCap)
      buttons.forEach((node) => take(node, 'button'))
      const links = [...main.querySelectorAll('a[href^="/"]')]
        .filter((node) => usable(node) && !seen.has(node) && !node.hasAttribute('download'))
        .filter((node) => !/^\/(api|uploads)\//.test(node.getAttribute('href') || ''))
        .filter((node) => (node.getAttribute('href') || '').split('?')[0] !== window.location.pathname)
        .slice(0, linksCap)
      links.forEach((node) => take(node, 'link'))
      return picked.map((entry, index) => {
        entry.node.setAttribute('data-e2e-opener', String(index))
        return { index, kind: entry.kind, name: entry.name }
      })
    },
    {
      rowsCap: ROWS_PER_PAGE,
      buttonsCap: BUTTONS_PER_PAGE,
      linksCap: LINKS_PER_PAGE,
      skip: SKIP.source,
      opener: OPENER.source,
    },
  )
}

const TOKEN = /[A-Za-z][A-Za-z0-9-]{3,}|\d{3,}[A-Za-z0-9-]*|[A-Z]{2,}\d{2,}/g

const GENERIC = new Set(
  'open view edit details the and with from your this that class classes section present absent today session sessions attendance minutes notice notices message messages course courses student students teacher teachers'.split(
    ' ',
  ),
)

export const tokensOf = (text) =>
  [...new Set((String(text || '').match(TOKEN) || []).map((word) => word.toLowerCase()))].filter(
    (word) => !GENERIC.has(word),
  )

const DAY_MONTH = /\b\d{1,2} (Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)\b/

export async function readDialog(page) {
  return page.evaluate((selector) => {
    const visible = (node) => {
      const box = node.getBoundingClientRect()
      return box.width > 0 && box.height > 0
    }
    const all = [...document.querySelectorAll(selector)].filter(visible)
    const dialog = all[all.length - 1]
    if (!dialog) return null
    const bar = document.querySelector('header.sticky')
    const barBottom = bar && !dialog.closest('[data-sheet-root]')?.contains(bar) ? bar.getBoundingClientRect().bottom : 0
    const heading = dialog.querySelector('h1, h2')
    return {
      title: ((heading && heading.innerText) || dialog.getAttribute('aria-label') || '').trim(),
      text: (dialog.innerText || '').replace(/\s+/g, ' ').trim(),
      top: Math.round(dialog.getBoundingClientRect().top),
      headingTop: heading ? Math.round(heading.getBoundingClientRect().top) : null,
      bottom: Math.round(dialog.getBoundingClientRect().bottom),
      barBottom: Math.round(barBottom),
      viewport: window.innerHeight,
      drawer: dialog.getAttribute('aria-label') === 'Menu',
    }
  }, DIALOG)
}

export async function scrollDialogToTheBottom(page) {
  return page.evaluate(async (selector) => {
    const settle = () => new Promise((resolve) => window.setTimeout(resolve, 120))
    const all = [...document.querySelectorAll(selector)]
    const dialog = all[all.length - 1]
    const body = dialog.querySelector('[data-sheet-body]') || dialog.querySelector('nav') || dialog
    body.scrollTop = body.scrollHeight
    await settle()
    const reached = body.scrollTop + body.clientHeight >= body.scrollHeight - 2
    const box = body.getBoundingClientRect()
    const footer = dialog.querySelector('[data-sheet-footer]')
    const actions = [...(footer || body).querySelectorAll('button, a[href]')].filter((node) => {
      const rect = node.getBoundingClientRect()
      return rect.width > 0 && rect.height > 0
    })
    const action = actions[actions.length - 1]
    const rect = action ? action.getBoundingClientRect() : null
    return {
      reached,
      scrollable: body.scrollHeight > body.clientHeight + 2,
      bodyBottom: Math.round(box.bottom),
      viewport: window.innerHeight,
      action: action ? (action.innerText || action.getAttribute('aria-label') || '').trim() : '',
      actionVisible: rect ? rect.top >= 0 && rect.bottom <= window.innerHeight + 1 : true,
    }
  }, DIALOG)
}

export async function closeDialog(page) {
  const dialog = page.locator(DIALOG).last()
  const close = dialog.getByRole('button', { name: /^(Close|Close menu)$/ }).first()
  if ((await close.count()) > 0 && (await close.isVisible())) {
    await close.click()
  } else {
    await page.keyboard.press('Escape')
  }
  await expect(page.locator(DIALOG), 'the view did not close').toHaveCount(0, { timeout: 5_000 })
}

export async function expectPageScrollsAgain(page, where) {
  const state = await page.evaluate(async () => {
    const settle = () => new Promise((resolve) => window.setTimeout(resolve, 120))
    const locked = document.body.style.overflow === 'hidden'
    const room = document.documentElement.scrollHeight - window.innerHeight
    window.scrollTo({ top: room, behavior: 'instant' })
    await settle()
    const moved = room <= 2 || window.scrollY > 0
    window.scrollTo({ top: 0, behavior: 'instant' })
    await settle()
    return { locked, moved, room }
  })
  expect(state.locked, `${where} left the page locked`).toBe(false)
  expect(state.moved, `${where} left the page unable to scroll (${JSON.stringify(state)})`).toBe(true)
}

export async function expectDialogWhole(page, opener, where) {
  await settleAnimations(page)
  const view = await readDialog(page)
  expect(view, `${where} opened nothing readable`).not.toBeNull()
  expect(view.title, `${where} opened a view without a title`).not.toBe('')
  const floor = view.drawer ? 0 : view.barBottom
  expect(view.top, `${where} starts above the screen (${JSON.stringify(view)})`).toBeGreaterThanOrEqual(0)
  if (view.headingTop !== null) {
    expect(
      view.headingTop,
      `${where} has its title under the top bar (${JSON.stringify(view)})`,
    ).toBeGreaterThanOrEqual(floor - 1)
  }
  expect(view.bottom, `${where} runs past the bottom of the screen`).toBeLessThanOrEqual(view.viewport + 1)
  if (opener.kind === 'row') {
    const words = tokensOf(opener.name)
    const text = view.text.toLowerCase()
    if (words.length > 0) {
      expect(
        words.some((word) => text.includes(word)),
        `${where} opened another item: "${opener.name}" became "${view.title}"`,
      ).toBe(true)
    }
    const day = DAY_MONTH.exec(opener.name)
    if (day) {
      expect(view.text, `${where} opened the session of another day`).toContain(day[0])
    }
  }
  await expectGoodLayout(page, where)
  const scrolled = await scrollDialogToTheBottom(page)
  expect(scrolled.reached, `${where} cannot be scrolled to its end (${JSON.stringify(scrolled)})`).toBe(true)
  expect(
    scrolled.bodyBottom,
    `${where} has its body below the screen (${JSON.stringify(scrolled)})`,
  ).toBeLessThanOrEqual(scrolled.viewport + 1)
  expect(
    scrolled.actionVisible,
    `${where} hides its main action "${scrolled.action}" (${JSON.stringify(scrolled)})`,
  ).toBe(true)
}

export async function expectPageWhole(page, where) {
  const head = await page.evaluate(() => {
    window.scrollTo({ top: 0, behavior: 'instant' })
    const bar = document.querySelector('header.sticky')
    const heading = document.querySelector('main h1')
    return {
      barBottom: bar ? Math.round(bar.getBoundingClientRect().bottom) : 0,
      top: heading ? Math.round(heading.getBoundingClientRect().top) : null,
      title: heading ? heading.innerText.trim() : '',
    }
  })
  if (head.top !== null) {
    expect(head.title, `${where} has an empty title`).not.toBe('')
    expect(head.top, `${where} has its title under the top bar`).toBeGreaterThanOrEqual(head.barBottom - 1)
  }
  await expectScrollsToTheBottom(page, where)
}

import { expect } from '@playwright/test'

// Console messages that say nothing about the portal. A browser writes some of these on
// every page, and a suite that fails on them fails on noise instead of on bugs.
const IGNORED_CONSOLE = [
  /favicon/i,
  /Download the React DevTools/i,
  /\[vite\]/i,
  /was preloaded using link preload but not used/i,
  /Content-Security-Policy/i,
  /ResizeObserver loop/i,
]

// Addresses whose failure is expected: the push key is absent on a server with no VAPID
// pair, and the beacon is allowed to be refused by the rate limiter.
const TOLERATED_API = [
  { path: '/api/push/public-key', statuses: [404, 503] },
  { path: '/api/analytics/view', statuses: [429] },
  { path: '/api/app/android-latest', statuses: [404, 502, 503] },
]

export function watch(page) {
  const problems = {
    console: [],
    pageErrors: [],
    api: [],
  }
  page.on('console', (message) => {
    if (message.type() !== 'error') return
    const text = message.text()
    if (IGNORED_CONSOLE.some((pattern) => pattern.test(text))) return
    problems.console.push(text)
  })
  page.on('pageerror', (error) => {
    problems.pageErrors.push(String(error && error.message ? error.message : error))
  })
  page.on('response', (response) => {
    const url = response.url()
    if (!url.includes('/api/')) return
    const status = response.status()
    if (status < 400) return
    const tolerated = TOLERATED_API.find((entry) => url.includes(entry.path))
    if (tolerated && tolerated.statuses.includes(status)) return
    // A 401 on a page the suite has not signed in for yet is the portal doing its job.
    if (status === 401 && !problems.signedIn) return
    problems.api.push(`${status} ${response.request().method()} ${new URL(url).pathname}`)
  })
  return problems
}

export function clear(problems) {
  problems.console.length = 0
  problems.pageErrors.length = 0
  problems.api.length = 0
}

export function expectNoProblems(problems, where) {
  expect(problems.pageErrors, `${where} threw`).toEqual([])
  expect(problems.console, `${where} wrote console errors`).toEqual([])
  expect(problems.api, `${where} made failing API calls`).toEqual([])
}

// measure reads the layout faults the brief asks about, in one pass in the page, so that
// one round trip covers a whole page instead of one per element.
export async function measure(page) {
  return page.evaluate(async () => {
    const viewport = window.innerWidth
    const doc = document.documentElement
    const report = {
      viewport,
      horizontalScroll: doc.scrollWidth > viewport + 1,
      tooWide: [],
      clipped: [],
      covered: [],
      tallDialogs: [],
      bottomReachable: true,
    }
    const describe = (node) => {
      const text = (node.innerText || node.textContent || '').trim().slice(0, 40)
      const id = node.id ? `#${node.id}` : ''
      const classes = typeof node.className === 'string' ? node.className.slice(0, 60) : ''
      const where = classes ? `.${classes.split(/\s+/).slice(0, 3).join('.')}` : ''
      return `${node.tagName.toLowerCase()}${id}${where}${text ? ` "${text}"` : ''}`
    }
    const visible = (node) => {
      const style = window.getComputedStyle(node)
      if (style.display === 'none' || style.visibility === 'hidden') return false
      if (Number(style.opacity) === 0) return false
      const box = node.getBoundingClientRect()
      return box.width > 0 && box.height > 0
    }
    // A label only a screen reader reads is one pixel across with its painting clipped
    // away. It is meant to be unreadable on screen, so it is never a cut-off name.
    const screenReaderOnly = (node) => {
      const style = window.getComputedStyle(node)
      if (style.clipPath && style.clipPath !== 'none') return true
      if (style.clip && style.clip !== 'auto') return true
      const box = node.getBoundingClientRect()
      return box.width <= 2 || box.height <= 2
    }
    // Decoration cannot be a layout fault: a shape nobody can touch, that says nothing
    // and that its own layer clips, is background art rather than content pushed off
    // the screen.
    const decoration = (node) => {
      if ((node.innerText || '').trim() !== '') return false
      for (let walk = node; walk && walk !== document.body; walk = walk.parentElement) {
        const style = window.getComputedStyle(walk)
        if (style.pointerEvents === 'none') return true
        if (walk.getAttribute('aria-hidden') === 'true') return true
      }
      return false
    }
    // A wide table or a week grid is meant to be read by scrolling the panel it sits in,
    // not the page. Only content the reader cannot bring into view is a fault, which is
    // what the page-level horizontal scroll above already catches.
    const insideHorizontalScroller = (node) => {
      for (let walk = node.parentElement; walk && walk !== document.body; walk = walk.parentElement) {
        const style = window.getComputedStyle(walk)
        if (/auto|scroll/.test(style.overflowX)) return true
      }
      return false
    }
    // Elements wider than the viewport.
    for (const node of document.querySelectorAll('body *')) {
      if (!visible(node) || screenReaderOnly(node) || decoration(node)) continue
      if (insideHorizontalScroller(node)) continue
      const box = node.getBoundingClientRect()
      if (box.width > viewport + 2 && box.left >= -2) {
        report.tooWide.push(`${describe(node)} is ${Math.round(box.width)}px wide`)
        if (report.tooWide.length > 5) break
      }
    }
    // Names, ids and titles cut off with an ellipsis.
    const clipCandidates = document.querySelectorAll(
      'h1, h2, h3, h4, td, th, a, button, label, p, span, li, dt, dd',
    )
    for (const node of clipCandidates) {
      if (!visible(node) || screenReaderOnly(node)) continue
      const style = window.getComputedStyle(node)
      const ellipsis = style.textOverflow === 'ellipsis' || style.overflow === 'hidden'
      if (!ellipsis) continue
      if (node.scrollWidth > node.clientWidth + 1 && node.clientWidth > 0) {
        report.clipped.push(describe(node))
        if (report.clipped.length > 5) break
      }
    }
    // Controls a fixed bar sits on top of. A control that happens to lie behind the bar
    // at this scroll position is not a fault: the reader scrolls it out. It is a fault
    // only when the control stays behind the bar once it has been scrolled into the
    // middle of the screen, which is what too little room at the bottom looks like.
    const bars = [...document.querySelectorAll('body *')].filter((node) => {
      if (!visible(node)) return false
      const style = window.getComputedStyle(node)
      return style.position === 'fixed' || style.position === 'sticky'
    })
    const settle = () =>
      new Promise((resolve) => {
        window.requestAnimationFrame(() => window.setTimeout(resolve, 40))
      })
    // A control that lives inside the bar is not covered by it: the sidebar and the tab
    // bar are sticky, so every link in them answers with one of their own nodes.
    const onTopOf = (node) => {
      const box = node.getBoundingClientRect()
      if (box.bottom < 0 || box.top > window.innerHeight) return null
      const found = document.elementFromPoint(box.left + box.width / 2, box.top + box.height / 2)
      if (!found) return null
      if (found === node || node.contains(found) || found.contains(node)) return null
      const bar = bars.find((entry) => entry === found || entry.contains(found))
      if (!bar || bar.contains(node)) return null
      return bar
    }
    // While a dialog is open everything behind it is meant to be out of reach, so only
    // the controls of the dialog itself are asked whether something covers them.
    // A dialog that is still fading in counts as open: it is what the reader is about to
    // look at, whatever its opacity this instant.
    const openDialog = [...document.querySelectorAll('[role="dialog"]')].find((node) => {
      const box = node.getBoundingClientRect()
      return box.width > 0 && box.height > 0 && window.getComputedStyle(node).display !== 'none'
    })
    const scope = openDialog || document
    const suspects = []
    for (const control of scope.querySelectorAll('button, a[href], input, select')) {
      if (!visible(control)) continue
      if (onTopOf(control)) suspects.push(control)
      if (suspects.length > 40) break
    }
    const atStart = window.scrollY
    for (const control of suspects) {
      control.scrollIntoView({ block: 'center', behavior: 'instant' })
      await settle()
      const bar = onTopOf(control)
      if (bar) {
        report.covered.push(`${describe(control)} stays under ${describe(bar)}`)
        if (report.covered.length > 5) break
      }
    }
    window.scrollTo({ top: atStart, behavior: 'instant' })
    await settle()
    // A dialog taller than the screen has to scroll inside itself.
    for (const dialog of document.querySelectorAll('[role="dialog"], dialog')) {
      if (!visible(dialog)) continue
      const box = dialog.getBoundingClientRect()
      if (box.height <= window.innerHeight + 1) continue
      const scrollers = [dialog, ...dialog.querySelectorAll('*')].filter((node) => {
        const style = window.getComputedStyle(node)
        const value = `${style.overflowY} ${style.overflow}`
        return /auto|scroll/.test(value) && node.scrollHeight > node.clientHeight + 1
      })
      if (scrollers.length === 0) {
        report.tallDialogs.push(`${describe(dialog)} is ${Math.round(box.height)}px tall`)
      }
    }
    return report
  })
}

// settleAnimations waits for the font and then for every running animation to finish. The
// font stylesheet loads without holding the page back, so a reading taken before it lands
// measures the fallback face; a dialog slides and fades in, and a reading taken
// mid-animation sees buttons narrower than they end up.
export async function settleAnimations(page) {
  await page
    .waitForFunction(
      () =>
        [...document.querySelectorAll('link[rel="stylesheet"]')].every((link) => link.media !== 'print') &&
        document.fonts.status === 'loaded',
      null,
      { timeout: 15_000 },
    )
    .catch(() => {})
  await page
    .evaluate(() =>
      Promise.race([
        Promise.all(
          document
            .getAnimations()
            .filter((animation) => {
              const timing = animation.effect && animation.effect.getTiming()
              return timing && timing.iterations !== Infinity
            })
            // A cancelled animation, such as the ripple of the button that opened a
            // dialog, rejects its promise; on its own it must not end the wait early.
            .map((animation) => animation.finished.catch(() => undefined)),
        ),
        new Promise((resolve) => window.setTimeout(resolve, 2000)),
      ]),
    )
    .catch(() => {})
}

export async function expectGoodLayout(page, where) {
  await settleAnimations(page)
  const report = await measure(page)
  expect(report.horizontalScroll, `${where} scrolls sideways`).toBe(false)
  expect(report.tooWide, `${where} has elements wider than the viewport`).toEqual([])
  expect(report.clipped, `${where} cuts text off with an ellipsis`).toEqual([])
  expect(report.covered, `${where} has controls under a bar`).toEqual([])
  expect(report.tallDialogs, `${where} has a dialog taller than the screen`).toEqual([])
}

// expectScrollsToTheBottom proves the page can be read to its end. A page whose body is
// locked while its content overflows leaves the last rows unreachable. The question is
// asked of the content rather than of scrollHeight, because a bar pinned below the fold
// inflates scrollHeight without adding anything a reader can scroll to.
export async function expectScrollsToTheBottom(page, where) {
  const report = await page.evaluate(async () => {
    const settle = () =>
      new Promise((resolve) => {
        window.requestAnimationFrame(() => window.setTimeout(resolve, 60))
      })
    const root = document.querySelector('main') || document.body
    const inFlow = (node) => {
      const style = window.getComputedStyle(node)
      if (style.position === 'fixed' || style.position === 'sticky') return false
      if (style.display === 'none' || style.visibility === 'hidden') return false
      const box = node.getBoundingClientRect()
      return box.height > 0 && box.width > 0
    }
    // The last thing in normal flow is the bottom of the page as a reader sees it.
    const candidates = [...root.querySelectorAll('*')].filter(inFlow)
    if (candidates.length === 0) return { reached: true, reason: 'nothing to read' }
    let last = candidates[0]
    for (const node of candidates) {
      if (node.getBoundingClientRect().bottom > last.getBoundingClientRect().bottom) last = node
    }
    const before = last.getBoundingClientRect().bottom
    if (before <= window.innerHeight + 2) {
      return { reached: true, reason: 'the whole page already fits', bottom: Math.round(before) }
    }
    const grownFrom = document.documentElement.scrollHeight
    last.scrollIntoView({ block: 'end', behavior: 'instant' })
    await settle()
    await new Promise((resolve) => window.setTimeout(resolve, 400))
    const after = last.getBoundingClientRect().bottom
    // An endless list, such as the activity log, loads the next page when the reader
    // reaches its end, so the end moves away. A page that grew is a page that can still
    // be read further, not a dead end.
    const grew = document.documentElement.scrollHeight > grownFrom + 4
    return {
      reached: grew || after <= window.innerHeight + 4,
      grew,
      bottomBefore: Math.round(before),
      bottomAfter: Math.round(after),
      viewport: window.innerHeight,
      scrollY: Math.round(window.scrollY),
    }
  })
  expect(
    report.reached,
    `${where} cannot be scrolled to the bottom (${JSON.stringify(report)})`,
  ).toBe(true)
}

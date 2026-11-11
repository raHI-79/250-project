import { expect } from '@playwright/test'

const IGNORED_CONSOLE = [
  /favicon/i,
  /Download the React DevTools/i,
  /\[vite\]/i,
  /was preloaded using link preload but not used/i,
  /Content-Security-Policy/i,
  /ResizeObserver loop/i,
]

const NETWORK_FAILURE = /^Failed to load resource: net::ERR_/

const isOtherSite = (page, address) => {
  try {
    return new URL(address).origin !== new URL(page.url()).origin
  } catch {
    return false
  }
}

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
    const source = message.location()?.url || ''
    if (NETWORK_FAILURE.test(text) && isOtherSite(page, source)) return
    problems.console.push(source ? `${text} (${source})` : text)
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
    const screenReaderOnly = (node) => {
      const style = window.getComputedStyle(node)
      if (style.clipPath && style.clipPath !== 'none') return true
      if (style.clip && style.clip !== 'auto') return true
      const box = node.getBoundingClientRect()
      return box.width <= 2 || box.height <= 2
    }
    const decoration = (node) => {
      if ((node.innerText || '').trim() !== '') return false
      for (let walk = node; walk && walk !== document.body; walk = walk.parentElement) {
        const style = window.getComputedStyle(walk)
        if (style.pointerEvents === 'none') return true
        if (walk.getAttribute('aria-hidden') === 'true') return true
      }
      return false
    }
    const insideHorizontalScroller = (node) => {
      for (let walk = node.parentElement; walk && walk !== document.body; walk = walk.parentElement) {
        const style = window.getComputedStyle(walk)
        if (/auto|scroll/.test(style.overflowX)) return true
      }
      return false
    }
    for (const node of document.querySelectorAll('body *')) {
      if (!visible(node) || screenReaderOnly(node) || decoration(node)) continue
      if (insideHorizontalScroller(node)) continue
      const box = node.getBoundingClientRect()
      if (box.width > viewport + 2 && box.left >= -2) {
        report.tooWide.push(`${describe(node)} is ${Math.round(box.width)}px wide`)
        if (report.tooWide.length > 5) break
      }
    }
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
    const bars = [...document.querySelectorAll('body *')].filter((node) => {
      if (!visible(node)) return false
      const style = window.getComputedStyle(node)
      return style.position === 'fixed' || style.position === 'sticky'
    })
    const settle = () =>
      new Promise((resolve) => {
        window.requestAnimationFrame(() => window.setTimeout(resolve, 40))
      })
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

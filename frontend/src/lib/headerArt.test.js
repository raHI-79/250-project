import assert from 'node:assert/strict'
import { existsSync, readFileSync, readdirSync } from 'node:fs'
import test from 'node:test'

const root = new URL('../', import.meta.url)

const read = (path) => readFileSync(new URL(path, root), 'utf8')

const files = (dir) =>
  readdirSync(new URL(dir, root), { recursive: true })
    .map((name) => `${dir}/${String(name).replace(/\\/g, '/')}`)
    .filter((name) => name.endsWith('.jsx'))

const HEADERS = [
  { file: 'components/ui/PageHeader.jsx', variant: 'page' },
  { file: 'components/today/TodayHero.jsx', variant: 'hero' },
  { file: 'components/layout/AppAuthScreen.jsx', variant: 'panel' },
]

const NAV_BAR = 'components/layout/TopBar.jsx'

const ruleOf = (css, selector) => {
  const start = css.indexOf(`${selector} {`)
  assert.ok(start >= 0, `${selector} is defined`)
  return css.slice(start, css.indexOf('}', start))
}

test('the shared layer is decorative and never takes a tap', () => {
  const art = read('components/ui/HeaderArt.jsx')
  assert.match(art, /aria-hidden="true"/)
  assert.match(art, /'header-art'/)
  const css = read('index.css')
  const base = ruleOf(css, '.header-art')
  assert.match(base, /position: absolute;/)
  assert.match(base, /inset: 0;/)
  assert.match(base, /z-index: -1;/)
  assert.match(base, /pointer-events: none;/)
  assert.match(base, /background-position: right bottom;/)
  assert.match(base, /background-size: auto 100%;/)
})

test('the illustration is one static file loaded by the stylesheet', () => {
  assert.ok(existsSync(new URL('../public/brand/header-iict.svg', root)))
  const svg = read('../public/brand/header-iict.svg')
  assert.doesNotMatch(svg, /<script|href=/i)
  const css = read('index.css')
  assert.match(ruleOf(css, '.header-art'), /url\("\/brand\/header-iict\.svg"\)/)
  const sources = files('components').concat(files('pages'))
  sources.forEach((name) => {
    assert.doesNotMatch(read(name), /header-iict\.svg/, `${name} does not load the art itself`)
  })
})

test('phones get the smaller art, short page headers smaller and fainter still', () => {
  const css = read('index.css')
  const phone = css.slice(css.indexOf('@media (max-width: 639.98px)'))
  assert.match(ruleOf(phone, '.header-art'), /background-size: auto 58%;/)
  const opacity = (block) => Number(/--header-art-opacity: ([\d.]+);/.exec(block)[1])
  const size = (block) => Number(/background-size: auto (\d+)%;/.exec(block)[1])
  const hero = ruleOf(css, '.header-art-hero')
  const page = ruleOf(css, '.header-art-page')
  const phonePage = ruleOf(phone, '.header-art-page')
  assert.ok(opacity(page) < opacity(hero))
  assert.ok(size(page) < 100)
  assert.ok(size(phonePage) < 58)
  assert.ok(opacity(phonePage) <= opacity(page))
})

test('dark mode dims the art in both theme switches', () => {
  const css = read('index.css')
  const forced = ruleOf(css, ':root[data-theme="dark"] .header-art')
  const system = ruleOf(css, ':root:not([data-theme="light"]) .header-art')
  ;[forced, system].forEach((rule) => {
    assert.match(rule, /opacity: calc\(var\(--header-art-opacity, [\d.]+\) \* 0\.\d+\)/)
  })
})

test('every blue header draws the shared header art behind its content', () => {
  HEADERS.forEach(({ file, variant }) => {
    const source = read(file)
    assert.match(source, /bg-header/, `${file} keeps the solid header colour`)
    assert.match(source, /relative isolate/, `${file} keeps the art inside the header`)
    assert.match(source, new RegExp(`<HeaderArt variant="${variant}" />`), `${file} uses the art`)
  })
  const blue = files('components')
    .concat(files('pages'))
    .filter((name) => /\bbg-header\b/.test(read(name)) && name !== NAV_BAR)
  assert.deepEqual(blue.sort(), HEADERS.map((entry) => entry.file).sort())
})

test('every main page opens with one of the shared headers', () => {
  const shared = /<(PageHeader|TodayHero|AuthScreen)\b/
  const pages = files('pages').filter(
    (name) => name.endsWith('Page.jsx') && !/\/(HomePage|NotFoundPage)\.jsx$/.test(name),
  )
  assert.ok(pages.length > 40, `found ${pages.length} pages`)
  const missing = pages.filter((name) => !shared.test(read(name)))
  assert.deepEqual(missing, [])
  assert.match(read('components/layout/AuthScreen.jsx'), /<AppAuthScreen/)
})

test('content the hero or the app panel overlap stays above the art', () => {
  assert.match(read('components/today/HomeStats.jsx'), /surface-card relative -mt-9/)
  assert.match(read('components/layout/AppAuthScreen.jsx'), /className="relative -mt-10/)
})

test('header tabs and the search bar sit on a dark chip so white text keeps its contrast', () => {
  assert.match(read('components/ui/StatusTabs.jsx'), /bg-\[rgb\(4_40_66\/0\.3\)\] font-semibold text-white/)
  assert.match(read('components/today/TodayHero.jsx'), /bg-\[rgb\(4_40_66\/0\.3\)\]/)
})

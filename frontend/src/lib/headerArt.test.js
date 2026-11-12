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
  const base = ruleOf(read('index.css'), '.header-art')
  assert.match(base, /position: absolute;/)
  assert.match(base, /inset: 0;/)
  assert.match(base, /z-index: -1;/)
  assert.match(base, /pointer-events: none;/)
})

const ART_WIDTH = 3200

const ART_HEIGHT = 420

const BUILDING_WIDTH = 1000

const artWidth = (width, height) =>
  Math.max(width, Math.min((ART_WIDTH / BUILDING_WIDTH) * width, (ART_WIDTH / ART_HEIGHT) * height))

test('the whole building fits the header at every width, anchored centre bottom', () => {
  const css = read('index.css')
  const base = ruleOf(css, '.header-art')
  assert.match(base, /container-type: size;/)
  assert.match(base, /background: #0a6faf;/)
  const art = ruleOf(css, '.header-art::before')
  assert.match(
    art,
    /background: url\("\/brand\/header-iict\.svg"\) center bottom \/ auto 100% no-repeat;/,
  )
  assert.match(art, /background-size: max\(100cqw, min\(320cqw, 761\.905cqh\)\) auto;/)
  assert.equal((ART_WIDTH / BUILDING_WIDTH) * 100, 320)
  assert.equal(((ART_WIDTH / ART_HEIGHT) * 100).toFixed(3), '761.905')
  ;[
    [360, 190],
    [390, 210],
    [412, 230],
    [768, 200],
    [1280, 190],
    [1920, 220],
    [390, 520],
  ].forEach(([width, height]) => {
    const drawn = artWidth(width, height)
    const building = (BUILDING_WIDTH / ART_WIDTH) * drawn
    assert.ok(building <= width + 0.5, `the building is cut at ${width}px`)
    assert.ok(drawn >= width, `the art leaves a gap at ${width}px`)
  })
  assert.equal(artWidth(1920, 220), 1920, 'a wide header lost its full-width trees')
  const svg = read('../public/brand/header-iict.svg')
  assert.match(svg, /viewBox="0 0 3200 420"/)
  assert.match(svg, /preserveAspectRatio="xMidYMax slice"/)
})

test('the illustration is one static file loaded by the stylesheet', () => {
  assert.ok(existsSync(new URL('../public/brand/header-iict.svg', root)))
  assert.doesNotMatch(read('../public/brand/header-iict.svg'), /<script|href=/i)
  files('components')
    .concat(files('pages'))
    .forEach((name) => {
      assert.doesNotMatch(read(name), /header-iict\.svg/, `${name} does not load the art itself`)
    })
})

const alphas = (rule) => [...rule.matchAll(/rgb\(4 32 54 \/ ([\d.]+)\)/g)].map((m) => Number(m[1]))

test('a darker overlay runs top to bottom on phones and left to right on desktop', () => {
  const css = read('index.css')
  const phone = ruleOf(css, '.header-art::after')
  assert.match(phone, /to bottom/)
  assert.equal(Math.max(...alphas(phone)), 0.55)
  assert.equal(alphas(phone)[0], 0)
  const wide = css.slice(css.indexOf('.header-art::after {') + 1)
  const desktop = ruleOf(wide, '.header-art::after')
  assert.match(desktop, /to right/)
  assert.equal(Math.max(...alphas(desktop)), 0.55)
  assert.match(desktop, /mask-image: linear-gradient\(to bottom, transparent 0%, #000 35%\)/)
})

test('dark mode dims the art in both theme switches', () => {
  const css = read('index.css')
  ;[':root[data-theme="dark"] .header-art', ':root:not([data-theme="light"]) .header-art'].forEach(
    (selector) => assert.match(ruleOf(css, selector), /filter: brightness\(0\.7\);/),
  )
})

test('the home hero is compact on phones and moderate on desktop', () => {
  const hero = read('components/today/TodayHero.jsx')
  assert.match(hero, /pt-2\.5 pb-6 sm:-mx-6 sm:px-6 sm:pt-5 sm:pb-14/)
  assert.match(hero, /size="md"/)
  assert.match(hero, /text-lg leading-tight text-white sm:text-title/)
  assert.match(hero, /mt-2\.5 flex h-9 w-full/)
  assert.match(read('components/today/HomeStats.jsx'), /-mt-4 flex sm:-mt-9/)
})

test('page headers show only the title and the tabs on phones', () => {
  const header = read('components/ui/PageHeader.jsx')
  assert.match(header, /pt-3 pb-3 sm:-mx-6 sm:px-6 sm:pt-5 sm:pb-5/)
  assert.match(header, /hidden text-2xs font-extrabold[^"]*sm:block/)
  assert.match(header, /mt-2 hidden max-w-3xl[^"]*sm:block/)
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
    (name) =>
      name.endsWith('Page.jsx') &&
      !/\/(HomePage|NotFoundPage)\.jsx$/.test(name) &&
      !name.startsWith('pages/site/'),
  )
  assert.ok(pages.length > 40, `found ${pages.length} pages`)
  const missing = pages.filter((name) => !shared.test(read(name)))
  assert.deepEqual(missing, [])
  assert.match(read('components/layout/AuthScreen.jsx'), /<AppAuthScreen/)
})

test('every public website page opens with the website header and ends with the credit', () => {
  const website = files('pages').filter((name) => name.startsWith('pages/site/'))
  assert.ok(website.length >= 2, `found ${website.length} website pages`)
  website.concat(['pages/HomePage.jsx']).forEach((name) => {
    assert.match(read(name), /<PublicHeader\b/, `${name} opens with the website header`)
    assert.match(read(name), /<SiteFooter\b/, `${name} ends with the website footer`)
  })
  const footer = read('components/home/SiteFooter.jsx')
  assert.match(footer, /<DeveloperCredit \/>/)
  assert.doesNotMatch(
    footer.slice(footer.indexOf('<DeveloperCredit />') - 400, footer.indexOf('<DeveloperCredit />')),
    /\{shown &&/,
    'the credit sits outside the part of the footer that can be hidden',
  )
  assert.match(read('components/site/DeveloperCredit.jsx'), /DEVELOPER\.label/)
  assert.doesNotMatch(read('pages/control/ControlWebsitePage.jsx'), /credit:/i)
})

test('content the hero or the app panel overlap stays above the art', () => {
  assert.match(read('components/today/HomeStats.jsx'), /surface-card relative -mt-4/)
  assert.match(read('components/layout/AppAuthScreen.jsx'), /className="relative -mt-10/)
})

test('header tabs and the search bar sit on a dark chip so white text keeps its contrast', () => {
  assert.match(read('components/ui/StatusTabs.jsx'), /bg-\[rgb\(4_40_66\/0\.3\)\] font-semibold text-white/)
  assert.match(read('components/today/TodayHero.jsx'), /bg-\[rgb\(4_40_66\/0\.3\)\]/)
})

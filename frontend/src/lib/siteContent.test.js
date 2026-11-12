import assert from 'node:assert/strict'
import test from 'node:test'
import {
  BLOCK_TYPES,
  DEFAULT_CONTENT,
  DEVELOPER,
  cloneContent,
  footerParts,
  homeSections,
  isSafeImage,
  isSafeLink,
  move,
  newBlock,
  pageBySlug,
  parseInline,
  parseRich,
  seoFor,
  slugFrom,
  slugProblem,
  stripCredit,
  visibleSections,
  withDefaults,
} from './siteContent.js'

test('the developer credit is fixed in the code', () => {
  assert.equal(DEVELOPER.label, 'Developed by CoordiNet')
  assert.equal(DEVELOPER.path, '/about-coordinet')
  assert.deepEqual([...DEVELOPER.people], ['Ratul Islam', 'Rakibul Hasan'])
  assert.throws(() => {
    'use strict'
    DEVELOPER.label = 'Somebody else'
  })
  assert.equal(JSON.stringify(DEFAULT_CONTENT).includes('CoordiNet'), false)
})

test('the credit shows with an empty, hidden or rewritten footer', () => {
  const hidden = cloneContent(DEFAULT_CONTENT)
  hidden.footer = { visible: false }
  assert.equal(footerParts(hidden).shown, false)
  assert.equal(footerParts(hidden).credit.label, 'Developed by CoordiNet')
  const rewritten = cloneContent(DEFAULT_CONTENT)
  rewritten.footer.note = 'Developed by somebody else'
  rewritten.footer.credit = 'Somebody else'
  assert.equal(footerParts(rewritten).credit.label, 'Developed by CoordiNet')
  assert.equal(footerParts(null).credit.path, '/about-coordinet')
  assert.equal(footerParts({}).shown, true)
})

test('the editor never sends a field that touches the credit', () => {
  const dirty = {
    footer: { note: 'x', hide_credit: true, credit: 'y' },
    developer: { label: 'z' },
    pages: [{ slug: 'a', developed_by: 'q', blocks: [{ type: 'text', text: 'ok', coordinet: 1 }] }],
  }
  assert.deepEqual(stripCredit(dirty), {
    footer: { note: 'x' },
    pages: [{ slug: 'a', blocks: [{ type: 'text', text: 'ok' }] }],
  })
})

test('nothing saved falls back to the built-in content', () => {
  assert.deepEqual(withDefaults(null), cloneContent(DEFAULT_CONTENT))
  const partial = withDefaults({ header: { site_name: 'CSE at SUST' }, home: { slides: [] } })
  assert.equal(partial.header.site_name, 'CSE at SUST')
  assert.equal(partial.header.logo_url, '/brand/sust-logo.png')
  assert.equal(partial.home.slides.length, 4)
  assert.equal(partial.footer.address.includes('Sylhet'), true)
  assert.deepEqual(partial.pages, [])
})

test('links are http(s) or site paths and images are uploaded or built in', () => {
  for (const good of ['https://www.sust.edu', 'http://a.b/c', '/p/admission', '/student/login']) {
    assert.equal(isSafeLink(good), true, good)
  }
  for (const bad of ['', 'javascript:alert(1)', '//evil.example', 'data:text/html,x', 'mailto:a@b.c', 'p/x', '/a b', 'https://']) {
    assert.equal(isSafeLink(bad), false, bad)
  }
  assert.equal(isSafeImage('/uploads/photos/0123456789abcdef0123456789abcdef.jpg'), true)
  assert.equal(isSafeImage('/brand/sust-logo.png'), true)
  assert.equal(isSafeImage('https://evil.example/a.png'), false)
  assert.equal(isSafeImage('/uploads/../secret'), false)
  assert.equal(isSafeImage(''), false)
})

test('formatted text becomes bold, italic, lists and links and never markup', () => {
  assert.deepEqual(parseInline('Apply **before** the *deadline* at [the office](https://www.sust.edu).'), [
    { type: 'text', text: 'Apply ' },
    { type: 'bold', text: 'before' },
    { type: 'text', text: ' the ' },
    { type: 'italic', text: 'deadline' },
    { type: 'text', text: ' at ' },
    { type: 'link', text: 'the office', url: 'https://www.sust.edu' },
    { type: 'text', text: '.' },
  ])
  assert.deepEqual(parseInline('[click](javascript:alert(1))')[0], { type: 'text', text: 'click' })
  assert.deepEqual(parseInline('<img src=x onerror=alert(1)>'), [
    { type: 'text', text: '<img src=x onerror=alert(1)>' },
  ])
  const groups = parseRich('First line\nsecond line\n\n- one\n- **two**\n\nLast')
  assert.deepEqual(
    groups.map((group) => group.type),
    ['paragraph', 'list', 'paragraph'],
  )
  assert.equal(groups[0].lines.length, 2)
  assert.equal(groups[1].items.length, 2)
  assert.deepEqual(groups[1].items[1], [{ type: 'bold', text: 'two' }])
  assert.deepEqual(parseRich(''), [])
})

test('home sections keep their order, hide when switched off and take custom ones anywhere', () => {
  const content = cloneContent(DEFAULT_CONTENT)
  assert.deepEqual(
    visibleSections(content).map((entry) => entry.key),
    ['stats', 'about', 'app'],
  )
  content.home.custom = [{ id: 'news1', title: 'News', blocks: [] }]
  content.home.sections = [
    { key: 'app', visible: true },
    { key: 'custom:news1', visible: true },
    { key: 'stats', visible: false },
    { key: 'custom:ghost', visible: true },
  ]
  assert.deepEqual(
    homeSections(content).map((entry) => `${entry.key}:${entry.visible}`),
    ['app:true', 'custom:news1:true', 'stats:false'],
  )
  assert.equal(homeSections(content)[1].label, 'News')
  assert.equal(visibleSections(content)[1].custom.id, 'news1')
})

test('a page address is unique and never the name of an app route', () => {
  assert.equal(slugProblem('admission-2027'), '')
  assert.notEqual(slugProblem('Admission'), '')
  assert.notEqual(slugProblem('a b'), '')
  assert.notEqual(slugProblem('student'), '')
  assert.notEqual(slugProblem('control-room'), '')
  assert.notEqual(slugProblem('about-coordinet'), '')
  assert.notEqual(slugProblem('labs', ['labs']), '')
  assert.equal(slugFrom('  Admission 2027: How to apply? '), 'admission-2027-how-to-apply')
  const content = { pages: [{ slug: 'labs', title: 'Labs' }] }
  assert.equal(pageBySlug(content, 'labs').title, 'Labs')
  assert.equal(pageBySlug(content, 'nope'), null)
})

test('every block type starts empty and blocks can be reordered', () => {
  for (const { value } of BLOCK_TYPES) assert.equal(newBlock(value).type, value)
  assert.equal(BLOCK_TYPES.length, 7)
  assert.deepEqual(move(['a', 'b', 'c'], 0, 1), ['b', 'a', 'c'])
  assert.deepEqual(move(['a', 'b', 'c'], 2, -1), ['a', 'c', 'b'])
  assert.deepEqual(move(['a', 'b'], 0, -1), ['a', 'b'])
})

test('each page has its own title and description', () => {
  const content = cloneContent(DEFAULT_CONTENT)
  assert.equal(seoFor(content).title, 'SUST CSE · Class and exam coordination')
  const page = { title: 'Admission', seo_title: '', seo_description: 'How to apply' }
  assert.equal(seoFor(content, page).title, 'Admission · SUST CSE · Class and exam coordination')
  assert.equal(seoFor(content, page).description, 'How to apply')
  assert.equal(seoFor(content, { ...page, seo_title: 'Apply to CSE' }).title, 'Apply to CSE')
})

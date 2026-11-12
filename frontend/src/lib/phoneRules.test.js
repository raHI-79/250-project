import assert from 'node:assert/strict'
import { readFileSync, readdirSync } from 'node:fs'
import test from 'node:test'

const root = new URL('../', import.meta.url)

const read = (path) => readFileSync(new URL(path, root), 'utf8')

const sources = (dir) =>
  readdirSync(new URL(dir, root), { recursive: true })
    .map((name) => `${dir}/${String(name).split('\\').join('/')}`)
    .filter((name) => name.endsWith('.jsx'))

test('every popup is the one shared sheet, rendered on the page body', () => {
  const sheet = read('components/ui/Sheet.jsx')
  assert.match(sheet, /createPortal\(/)
  assert.match(sheet, /document\.body,\n {2}\)/)
  assert.match(sheet, /sheet-panel/)
  assert.match(sheet, /data-sheet-body/)
  assert.match(sheet, /overscroll-contain overflow-x-hidden overflow-y-auto/)
  assert.match(sheet, /data-sheet-footer/)
  assert.match(sheet, /env\(safe-area-inset-bottom/)
  assert.match(read('components/ui/Modal.jsx'), /<Sheet/)
  const own = sources('components')
    .concat(sources('pages'))
    .filter((name) => /role="dialog"/.test(read(name)))
  assert.deepEqual(own.sort(), ['components/layout/Sidebar.jsx', 'components/ui/Sheet.jsx'])
})

test('a sheet stays below the top bar and inside the screen, with a fallback for old webviews', () => {
  const css = read('index.css')
  const panel = css.slice(css.indexOf('.sheet-panel {'), css.indexOf('}', css.indexOf('.sheet-panel {')))
  assert.match(panel, /max-height: calc\(100vh - 4\.5rem\);/)
  assert.match(panel, /max-height: calc\(100dvh - max\(env\(safe-area-inset-top, 0px\), 0px\) - 4\.5rem\);/)
})

test('no entry animation leaves a transform behind that would trap a fixed sheet', () => {
  const css = read('index.css')
  ;['fade-in', 'fade-in-up', 'scale-in', 'slide-in-right', 'slide-in-left', 'sheet-up'].forEach((name) => {
    const line = css.split('\n').find((text) => text.includes(`--animate-${name}:`))
    assert.ok(line, `${name} is defined`)
    assert.match(line, /backwards;$/, `${name} keeps its last frame`)
  })
})

test('the shell scrolls the page itself, with opaque bars and padding for them', () => {
  const shell = read('components/layout/AppShell.jsx')
  assert.match(shell, /min-h-dvh/)
  assert.doesNotMatch(shell, /[ '"](h-screen|h-dvh)[ '"]|overflow-hidden|100vh/)
  assert.match(shell, /pb-\[calc\(5rem\+env\(safe-area-inset-bottom,0px\)\)\]/)
  assert.match(shell, /releaseStaleLock\(openSheetCount\(\)\)/)
  assert.match(read('components/layout/TopBar.jsx'), /sticky top-0 z-50 bg-header/)
  assert.match(read('components/layout/TabBar.jsx'), /bg-surface/)
  const dragOnly = sources('components')
    .concat(sources('pages'))
    .filter((name) => /touch-none|touch-action/.test(read(name)))
  assert.deepEqual(dragOnly, ['components/ui/Sheet.jsx'])
})

test('pull to refresh only starts at the top and never inside another scroller', () => {
  const pull = read('components/ui/PullToRefresh.jsx')
  assert.match(pull, /if \(insideScroller\(event\.target\)\) return/)
  assert.match(pull, /const options = \{ passive: true \}/)
  assert.doesNotMatch(pull, /preventDefault/)
})

test('the keyboard resizes the page instead of covering the focused field', () => {
  assert.match(read('../index.html'), /interactive-widget=resizes-content/)
  assert.match(read('../android/app/src/main/AndroidManifest.xml'), /android:windowSoftInputMode="adjustResize"/)
})

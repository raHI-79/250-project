import assert from 'node:assert/strict'
import test from 'node:test'
import { activeNavPath, adminNav, controlNav, isNavActive } from './nav.js'
import { paths } from './paths.js'

const activeItems = (groups, pathname) => {
  const activePath = activeNavPath(groups, pathname)
  return groups.flatMap((group) => group.items.filter((item) => isNavActive(item, activePath)))
}

const everyRouteOf = (groups) =>
  groups.flatMap((group) => group.items.map((item) => String(item.to()).split('?')[0]))

test('exactly one control room item is active on each of its routes', () => {
  everyRouteOf(controlNav).forEach((route) => {
    const active = activeItems(controlNav, route)
    assert.equal(active.length, 1, `${route} highlighted ${active.length} items`)
    assert.equal(String(active[0].to()).split('?')[0], route)
  })
})

test('exactly one office item is active on each of its routes', () => {
  everyRouteOf(adminNav).forEach((route) => {
    const active = activeItems(adminNav, route)
    assert.equal(active.length, 1, `${route} highlighted ${active.length} items`)
    assert.equal(String(active[0].to()).split('?')[0], route)
  })
})

test('the dashboard is active only on the control room root', () => {
  assert.equal(activeNavPath(controlNav, paths.control.home()), paths.control.home())
  assert.notEqual(activeNavPath(controlNav, paths.control.settings()), paths.control.home())
  assert.notEqual(activeNavPath(controlNav, paths.control.account()), paths.control.home())
  assert.notEqual(activeNavPath(controlNav, paths.control.watch()), paths.control.home())
})

test('the settings pages no longer light up together', () => {
  const active = activeItems(controlNav, paths.control.account())
  assert.equal(active.length, 1)
  assert.equal(active[0].label, 'My account')
  const monitor = activeItems(controlNav, paths.control.watch())
  assert.equal(monitor.length, 1)
  assert.equal(monitor[0].label, 'Security monitor')
  const settings = activeItems(controlNav, paths.control.settings())
  assert.equal(settings.length, 1)
  assert.equal(settings[0].label, 'Site settings')
})

test('a detail page keeps its own list highlighted', () => {
  const active = activeItems(controlNav, paths.control.monitor(7))
  assert.equal(active.length, 1)
  assert.equal(String(active[0].to()).split('?')[0], paths.control.activity())
})

test('a route outside the menu highlights nothing', () => {
  assert.equal(activeNavPath(controlNav, paths.profile()), '')
  assert.equal(activeItems(controlNav, paths.profile()).length, 0)
})

test('a trailing slash does not change the match', () => {
  assert.equal(activeNavPath(controlNav, `${paths.control.admins()}/`), paths.control.admins())
})

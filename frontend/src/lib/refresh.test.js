import assert from 'node:assert/strict'
import test from 'node:test'
import {
  afterRequest,
  bumpResources,
  isQuietPath,
  resourceSource,
  resourceVersion,
  resourceView,
  shouldInvalidate,
  watchResources,
} from './refresh.js'

test('only a change invalidates what the pages show', () => {
  assert.equal(shouldInvalidate('GET'), false)
  assert.equal(shouldInvalidate('head'), false)
  assert.equal(shouldInvalidate('POST'), true)
  assert.equal(shouldInvalidate('PUT'), true)
  assert.equal(shouldInvalidate('PATCH'), true)
  assert.equal(shouldInvalidate('DELETE'), true)
})

test('a list refreshes after a mutation', () => {
  const seen = []
  const stop = watchResources((version) => seen.push(version))
  const before = resourceVersion()
  afterRequest('GET')
  assert.equal(resourceVersion(), before, 'a read invalidated the lists')
  assert.equal(seen.length, 0)
  afterRequest('POST')
  assert.equal(resourceVersion(), before + 1, 'a mutation did not invalidate the lists')
  assert.equal(seen.length, 1)
  assert.equal(seen[0], before + 1)
  afterRequest('DELETE')
  assert.equal(seen.length, 2)
  stop()
  bumpResources()
  assert.equal(seen.length, 2, 'the listener was still called after it stopped')
})

test('background posts do not reload what the pages show', () => {
  assert.equal(shouldInvalidate('POST', '/api/analytics/view'), false)
  assert.equal(shouldInvalidate('POST', '/api/push/subscribe'), false)
  assert.equal(shouldInvalidate('POST', '/api/me/badges/notices/seen'), false)
  assert.equal(shouldInvalidate('POST', '/api/teacher/attendance/sessions/12/location'), false)
  assert.equal(shouldInvalidate('POST', '/api/teacher/attendance/sessions/12/close'), true)
  assert.equal(shouldInvalidate('PATCH', '/api/teacher/attendance/sessions/12/students/4'), true)
  assert.equal(isQuietPath('/api/analytics/view?x=1'), true)
  const before = resourceVersion()
  afterRequest('POST', '/api/analytics/view')
  assert.equal(resourceVersion(), before, 'a page view reloaded every list')
})

test('a list keeps its rows while it reloads and never shows another source', () => {
  const source = resourceSource('/api/teacher/attendance/sessions', '{}')
  const rows = { items: [{ id: 1 }, { id: 2 }] }
  const kept = resourceView({ source, data: rows, error: '' }, source, true)
  assert.equal(kept.data, rows)
  assert.equal(kept.loading, false, 'a reload swapped the rows for a skeleton')
  assert.equal(kept.refreshing, true)
  const first = resourceView({ source: '', data: null, error: '' }, source, true)
  assert.equal(first.loading, true)
  const other = resourceView({ source, data: rows, error: '' }, resourceSource('/api/x/2', '{}'), true)
  assert.equal(other.data, null, 'a detail showed the data of another item')
  assert.equal(other.loading, true)
  const changed = resourceView({ source, data: rows, error: '' }, resourceSource('/api/x/2', '{}'), false)
  assert.equal(changed.data, null)
  assert.equal(changed.loading, true, 'a new day was drawn before its data arrived')
  const failed = resourceView({ source, data: null, error: 'Offline' }, source, false)
  assert.equal(failed.loading, false)
  assert.equal(failed.error, 'Offline')
  const none = resourceView({ source, data: rows, error: '' }, '', false)
  assert.equal(none.data, null)
  assert.equal(none.loading, false)
  assert.equal(resourceSource('', '{}'), '')
})

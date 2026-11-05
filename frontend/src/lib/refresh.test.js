import assert from 'node:assert/strict'
import test from 'node:test'
import {
  afterRequest,
  bumpResources,
  resourceVersion,
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

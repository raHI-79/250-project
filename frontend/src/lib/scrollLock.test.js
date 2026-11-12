import assert from 'node:assert/strict'
import test from 'node:test'
import { createScrollLock } from './scrollLock.js'

const fakeBody = (overflow = '') => {
  const body = { style: { overflow } }
  return { body, bodyOf: () => body }
}

test('locks the page on the first dialog and unlocks on the last', () => {
  const { body, bodyOf } = fakeBody()
  const lock = createScrollLock(bodyOf)
  const release = lock.lock()
  assert.equal(body.style.overflow, 'hidden')
  assert.equal(lock.locked(), true)
  release()
  assert.equal(body.style.overflow, '')
  assert.equal(lock.locked(), false)
})

test('counts nested dialogs so the inner one never unlocks the page', () => {
  const { body, bodyOf } = fakeBody()
  const lock = createScrollLock(bodyOf)
  const outer = lock.lock()
  const inner = lock.lock()
  assert.equal(lock.depth(), 2)
  inner()
  assert.equal(body.style.overflow, 'hidden')
  outer()
  assert.equal(body.style.overflow, '')
  assert.equal(lock.depth(), 0)
})

test('unlocks whichever order the dialogs close in', () => {
  const { body, bodyOf } = fakeBody()
  const lock = createScrollLock(bodyOf)
  const outer = lock.lock()
  const inner = lock.lock()
  outer()
  assert.equal(body.style.overflow, 'hidden')
  inner()
  assert.equal(body.style.overflow, '')
})

test('ignores a release that already ran', () => {
  const { body, bodyOf } = fakeBody()
  const lock = createScrollLock(bodyOf)
  const first = lock.lock()
  const second = lock.lock()
  first()
  first()
  first()
  assert.equal(lock.depth(), 1)
  assert.equal(body.style.overflow, 'hidden')
  second()
  assert.equal(lock.depth(), 0)
  assert.equal(body.style.overflow, '')
})

test('keeps the value the page had before the first lock', () => {
  const { body, bodyOf } = fakeBody('clip')
  const lock = createScrollLock(bodyOf)
  const release = lock.lock()
  assert.equal(body.style.overflow, 'hidden')
  release()
  assert.equal(body.style.overflow, 'clip')
})

test('never leaves the lock on when a dialog tree is thrown away', () => {
  const { body, bodyOf } = fakeBody()
  const lock = createScrollLock(bodyOf)
  lock.lock()
  lock.lock()
  lock.reset()
  assert.equal(body.style.overflow, '')
  assert.equal(lock.locked(), false)
})

test('a lock nobody holds any more is released after a route change', async () => {
  const { isScrollLocked, lockScroll, releaseStaleLock } = await import('./scrollLock.js')
  const release = lockScroll()
  assert.equal(isScrollLocked(), true)
  assert.equal(releaseStaleLock(1), false, 'an open sheet lost its lock')
  assert.equal(isScrollLocked(), true)
  assert.equal(releaseStaleLock(0), true)
  assert.equal(isScrollLocked(), false)
  release()
  assert.equal(isScrollLocked(), false, 'a late release locked the page again')
  assert.equal(releaseStaleLock(0), false)
})

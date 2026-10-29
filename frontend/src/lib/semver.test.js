import assert from 'node:assert/strict'
import test from 'node:test'
import { newerThan, versionCode } from './semver.js'

test('reads a version into one comparable number', () => {
  assert.equal(versionCode('2.1.0'), 20100)
  assert.equal(versionCode('v2.1.0'), 20100)
  assert.equal(versionCode('2.1.3'), 20103)
  assert.equal(versionCode('3'), 30000)
})

test('ignores a pre-release suffix', () => {
  assert.equal(versionCode('2.1.0-beta.2'), versionCode('2.1.0'))
})

test('treats junk as the oldest possible version', () => {
  assert.equal(versionCode(''), 0)
  assert.equal(versionCode(undefined), 0)
  assert.equal(versionCode('not-a-version'), 0)
})

test('offers an update only for a newer release', () => {
  assert.equal(newerThan('2.1.0', '2.0.1'), true)
  assert.equal(newerThan('2.1.1', '2.1.0'), true)
  assert.equal(newerThan('3.0.0', '2.9.9'), true)
  assert.equal(newerThan('2.1.0', '2.1.0'), false)
  assert.equal(newerThan('2.0.1', '2.1.0'), false)
  assert.equal(newerThan('', '2.1.0'), false)
})

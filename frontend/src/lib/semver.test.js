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

test('compares the releases this portal actually shipped', () => {
  assert.equal(newerThan('2.2.2', '2.2.1'), true)
  assert.equal(newerThan('2.2.3', '2.2.2'), true)
  assert.equal(newerThan('2.2.1', '2.2.2'), false)
  assert.equal(newerThan('2.2.2', '2.2.2'), false)
})

test('reads ten as ten, not as one followed by zero', () => {
  assert.equal(newerThan('2.10.0', '2.9.9'), true)
  assert.equal(newerThan('2.9.9', '2.10.0'), false)
  assert.equal(newerThan('2.2.10', '2.2.9'), true)
  assert.equal(newerThan('10.0.0', '9.99.99'), true)
})

test('the version code is major times ten thousand plus minor times a hundred plus patch', () => {
  const cases = [
    ['2.2.1', 20201],
    ['2.2.2', 20202],
    ['2.2.3', 20203],
    ['2.10.0', 21000],
    ['2.9.9', 20909],
    ['1.0.0', 10000],
  ]
  for (const [version, want] of cases) {
    const parts = version.split('.').map(Number)
    assert.equal(versionCode(version), want)
    assert.equal(versionCode(version), parts[0] * 10000 + parts[1] * 100 + parts[2])
  }
})

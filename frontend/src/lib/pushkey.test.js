import assert from 'node:assert/strict'
import test from 'node:test'
import { PushKeyError, sameApplicationKey, urlBase64ToUint8Array } from './pushkey.js'

const vapidBytes = () => {
  const bytes = new Uint8Array(65)
  bytes[0] = 4
  for (let index = 1; index < bytes.length; index += 1) bytes[index] = index
  return bytes
}

const toUrlBase64 = (bytes) =>
  btoa(String.fromCharCode(...bytes)).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '')

test('decodes an unpadded base64url VAPID key', () => {
  const expected = vapidBytes()
  assert.deepEqual(urlBase64ToUint8Array(toUrlBase64(expected)), expected)
})

test('accepts a key with stray whitespace and padding', () => {
  const expected = vapidBytes()
  const padded = `  ${toUrlBase64(expected)}==\n`
  assert.deepEqual(urlBase64ToUint8Array(padded), expected)
})

test('rejects an empty key', () => {
  assert.throws(() => urlBase64ToUint8Array('   '), PushKeyError)
})

test('rejects a key with characters outside base64url', () => {
  assert.throws(() => urlBase64ToUint8Array('abc+def/ghi'), PushKeyError)
})

test('rejects a key of the wrong length', () => {
  assert.throws(() => urlBase64ToUint8Array(toUrlBase64(new Uint8Array([4, 1, 2, 3]))), PushKeyError)
})

test('rejects a key that is not an uncompressed point', () => {
  const bytes = vapidBytes()
  bytes[0] = 3
  assert.throws(() => urlBase64ToUint8Array(toUrlBase64(bytes)), PushKeyError)
})

test('compares the key of an existing subscription', () => {
  const key = vapidBytes()
  assert.equal(sameApplicationKey({ options: { applicationServerKey: key.buffer } }, key), true)
  assert.equal(sameApplicationKey({ options: {} }, key), false)
  assert.equal(sameApplicationKey({ options: { applicationServerKey: new Uint8Array(3) } }, key), false)
})

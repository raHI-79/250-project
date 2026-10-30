import assert from 'node:assert/strict'
import test from 'node:test'
import {
  CODE_SLICE_SECONDS,
  TOKEN_BYTES,
  TOKEN_SLICE_SECONDS,
  TOKEN_VECTOR,
  attendanceCode,
  attendanceToken,
  sliceAt,
  sliceOf,
} from './attendanceToken.js'

test('the shared vector gives the same eight bytes as the server', async () => {
  const got = await attendanceToken(
    TOKEN_VECTOR.key,
    TOKEN_VECTOR.sessionId,
    TOKEN_VECTOR.slice,
  )
  assert.equal(got, TOKEN_VECTOR.token)
  assert.equal(got.length, TOKEN_BYTES * 2)
})

test('the shared vector gives the same four digit code', async () => {
  const got = await attendanceCode(
    TOKEN_VECTOR.key,
    TOKEN_VECTOR.sessionId,
    TOKEN_VECTOR.slice,
  )
  assert.equal(got, TOKEN_VECTOR.code)
})

test('a different key, session or slice changes the token', async () => {
  const { key, sessionId, slice, token } = TOKEN_VECTOR
  assert.notEqual(await attendanceToken(`${key}x`, sessionId, slice), token)
  assert.notEqual(await attendanceToken(key, sessionId + 1, slice), token)
  assert.notEqual(await attendanceToken(key, sessionId, slice + 1), token)
})

test('the slice of the vector matches its own timestamp', () => {
  const at = new Date(TOKEN_VECTOR.slice * TOKEN_SLICE_SECONDS * 1000)
  assert.equal(sliceAt(at, TOKEN_SLICE_SECONDS), TOKEN_VECTOR.slice)
})

test('slices step with time and never divide by zero', () => {
  assert.equal(sliceOf(0, TOKEN_SLICE_SECONDS), 0)
  assert.equal(sliceOf(9, TOKEN_SLICE_SECONDS), 0)
  assert.equal(sliceOf(10, TOKEN_SLICE_SECONDS), 1)
  assert.equal(sliceOf(19, CODE_SLICE_SECONDS), 0)
  assert.equal(sliceOf(20, CODE_SLICE_SECONDS), 1)
  assert.equal(sliceOf(100, 0), 0)
})

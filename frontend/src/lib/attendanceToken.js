export const TOKEN_SLICE_SECONDS = 10

export const CODE_SLICE_SECONDS = 20

export const TOKEN_BYTES = 8

export const CODE_DIGITS = 4

export const SERVICE_UUID = '0000a7a7-0000-1000-8000-00805f9b34fb'

export const TOKEN_VECTOR = {
  key: 'sust-cse-attendance-test-key',
  sessionId: 4221,
  slice: 176200000,
  token: 'd937b886187a23ae',
  code: '7590',
}

export const sliceOf = (epochSeconds, seconds) => {
  if (!seconds || seconds <= 0) return 0
  return Math.floor(epochSeconds / seconds)
}

export const sliceAt = (date, seconds) =>
  sliceOf(Math.floor((date instanceof Date ? date.getTime() : Number(date)) / 1000), seconds)

const hex = (bytes) =>
  Array.from(bytes)
    .map((value) => value.toString(16).padStart(2, '0'))
    .join('')

const digest = async (key, sessionId, slice) => {
  const encoder = new TextEncoder()
  const secret = await globalThis.crypto.subtle.importKey(
    'raw',
    encoder.encode(String(key)),
    { name: 'HMAC', hash: 'SHA-256' },
    false,
    ['sign'],
  )
  const message = encoder.encode(`${sessionId}:${slice}`)
  return new Uint8Array(await globalThis.crypto.subtle.sign('HMAC', secret, message))
}

export const attendanceToken = async (key, sessionId, slice) =>
  hex((await digest(key, sessionId, slice)).slice(0, TOKEN_BYTES))

export const attendanceCode = async (key, sessionId, slice) => {
  const sum = await digest(key, sessionId, slice)
  const value =
    ((sum[0] << 24) | (sum[1] << 16) | (sum[2] << 8) | sum[3]) >>> 0
  return String(value % 10 ** CODE_DIGITS).padStart(CODE_DIGITS, '0')
}

export class PushKeyError extends Error {
  constructor(message) {
    super(message)
    this.name = 'PushKeyError'
  }
}

const BASE64URL = /^[A-Za-z0-9\-_]+$/

const VAPID_KEY_BYTES = 65

const UNCOMPRESSED_POINT = 4

export const urlBase64ToUint8Array = (value) => {
  const trimmed = String(value ?? '')
    .trim()
    .replace(/\s+/g, '')
    .replace(/=+$/, '')
  if (!trimmed) {
    throw new PushKeyError('The server did not send a notification key.')
  }
  if (!BASE64URL.test(trimmed)) {
    throw new PushKeyError('The notification key on the server is not valid base64url.')
  }
  const padded = trimmed.padEnd(trimmed.length + ((4 - (trimmed.length % 4)) % 4), '=')
  let raw
  try {
    raw = atob(padded.replace(/-/g, '+').replace(/_/g, '/'))
  } catch {
    throw new PushKeyError('The notification key on the server could not be decoded.')
  }
  const bytes = new Uint8Array(raw.length)
  for (let index = 0; index < raw.length; index += 1) bytes[index] = raw.charCodeAt(index)
  if (bytes.length !== VAPID_KEY_BYTES || bytes[0] !== UNCOMPRESSED_POINT) {
    throw new PushKeyError('The notification key on the server is not a valid VAPID public key.')
  }
  return bytes
}

export const sameApplicationKey = (subscription, key) => {
  const current = subscription?.options?.applicationServerKey
  if (!current) return false
  const bytes = new Uint8Array(current)
  if (bytes.length !== key.length) return false
  return bytes.every((byte, index) => byte === key[index])
}

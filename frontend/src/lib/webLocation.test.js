import assert from 'node:assert/strict'
import test from 'node:test'
import {
  PLACE_ASKING,
  PLACE_DENIED,
  PLACE_GRANTED,
  PLACE_IDLE,
  PLACE_OFF,
  PLACE_TIMEOUT,
  PLACE_UNSUPPORTED,
  WEB_BLUETOOTH_LINE,
  blockedSteps,
  browserKind,
  classifyLocationError,
  devicePlatform,
  offStep,
  studentPlaceState,
  webPositionState,
} from './webLocation.js'

const ANDROID_CHROME =
  'Mozilla/5.0 (Linux; Android 14; Pixel 8) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/130.0.0.0 Mobile Safari/537.36'
const DESKTOP_CHROME =
  'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/130.0.0.0 Safari/537.36'
const IPHONE_SAFARI =
  'Mozilla/5.0 (iPhone; CPU iPhone OS 17_5 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.5 Mobile/15E148 Safari/604.1'
const IPHONE_CHROME =
  'Mozilla/5.0 (iPhone; CPU iPhone OS 17_5 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) CriOS/130.0 Mobile/15E148 Safari/604.1'
const MAC_SAFARI =
  'Mozilla/5.0 (Macintosh; Intel Mac OS X 14_5) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.5 Safari/605.1.15'
const FIREFOX = 'Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:131.0) Gecko/20100101 Firefox/131.0'

const now = new Date('2026-11-12T09:00:30Z')

const openSession = (extra = {}) => ({
  id: 9,
  method: 'bluetooth',
  code_only: false,
  location_updated_at: '2026-11-12T09:00:20Z',
  accuracy_m: 12,
  ...extra,
})

test('every browser gets its own way to allow location', () => {
  assert.equal(browserKind(ANDROID_CHROME), 'chrome-android')
  assert.equal(browserKind(DESKTOP_CHROME), 'chrome-desktop')
  assert.equal(browserKind(IPHONE_SAFARI), 'safari-ios')
  assert.equal(browserKind(IPHONE_CHROME), 'ios-other')
  assert.equal(browserKind(MAC_SAFARI), 'safari-mac')
  assert.equal(browserKind(MAC_SAFARI, true), 'safari-ios')
  assert.equal(browserKind(FIREFOX), 'firefox')
  assert.equal(browserKind(''), 'other')
})

test('chrome on android is told to use the lock icon, then reload', () => {
  const steps = blockedSteps('chrome-android')
  assert.match(steps[0], /lock icon in the address bar/)
  assert.match(steps[1], /Permissions, then Location/)
  assert.match(steps[2], /Allow, then reload/)
})

test('chrome desktop and safari have steps of their own', () => {
  assert.match(blockedSteps('chrome-desktop').join(' '), /Site settings/)
  assert.match(blockedSteps('safari-ios').join(' '), /aA in the address bar/)
  assert.match(blockedSteps('safari-mac').join(' '), /Safari → Settings → Websites → Location/)
  assert.deepEqual(blockedSteps('nonsense'), blockedSteps('other'))
})

test('a blocked site and a switched off location service are told apart', () => {
  assert.equal(classifyLocationError(1, 'User denied Geolocation', 'denied'), PLACE_DENIED)
  assert.equal(classifyLocationError(1, 'User denied Geolocation', 'prompt'), PLACE_DENIED)
  assert.equal(classifyLocationError(1, '', 'granted'), PLACE_OFF)
  assert.equal(classifyLocationError(2, 'Position unavailable'), PLACE_OFF)
  assert.equal(classifyLocationError(3, 'Timeout expired'), PLACE_TIMEOUT)
})

test('a switched off location service names the device setting', () => {
  assert.equal(devicePlatform(ANDROID_CHROME), 'android')
  assert.equal(devicePlatform(IPHONE_SAFARI), 'ios')
  assert.equal(devicePlatform(MAC_SAFARI), 'mac')
  assert.equal(devicePlatform(DESKTOP_CHROME), 'windows')
  assert.match(offStep('android'), /Swipe down .* turn Location on/)
  assert.match(offStep('ios'), /Location Services/)
  const state = webPositionState({ state: PLACE_OFF }, openSession(), now, { platform: 'android' })
  assert.equal(state.label, 'Location is off on this device')
  assert.equal(state.fix, 'retry')
  assert.match(state.reason, /Swipe down/)
})

test('the teacher on the website is asked with a clear Allow location button', () => {
  const state = webPositionState({ state: PLACE_IDLE }, openSession(), now)
  assert.equal(state.fix, 'allow')
  assert.equal(state.tone, 'bad')
  assert.equal(webPositionState({ state: PLACE_ASKING }, openSession(), now).fix, undefined)
})

test('a denied browser shows the steps and a Try again, never the app settings', () => {
  const state = webPositionState({ state: PLACE_DENIED }, openSession(), now, {
    kind: 'chrome-android',
  })
  assert.equal(state.fix, 'retry')
  assert.notEqual(state.fix, 'settings')
  assert.deepEqual(state.steps, blockedSteps('chrome-android'))
})

test('a granted browser reports how fresh the position on the server is', () => {
  const granted = { state: PLACE_GRANTED }
  assert.equal(webPositionState(granted, openSession(), now).label, 'Sending · accuracy 12 m')
  assert.equal(
    webPositionState(granted, openSession({ location_updated_at: '' }), now).label,
    'Allowed · waiting for GPS',
  )
  assert.equal(
    webPositionState(granted, openSession({ location_updated_at: '2026-11-12T08:58:00Z' }), now)
      .label,
    'Out of date',
  )
})

test('a code only session needs no position and says the marks are flagged', () => {
  const state = webPositionState({ state: PLACE_DENIED }, openSession({ code_only: true }), now)
  assert.equal(state.label, 'Not checked · code only')
  assert.equal(state.fix, undefined)
  assert.match(state.reason, /No location/)
})

test('a browser without geolocation says so plainly', () => {
  assert.match(
    webPositionState({ state: PLACE_UNSUPPORTED }, openSession(), now).reason,
    /cannot share a position/,
  )
  assert.match(
    webPositionState({ state: PLACE_UNSUPPORTED }, openSession(), now, { insecure: true }).reason,
    /https/,
  )
})

test('the student sees the same steps and a ready position', () => {
  assert.equal(studentPlaceState({ state: PLACE_IDLE }).fix, 'allow')
  assert.deepEqual(
    studentPlaceState({ state: PLACE_DENIED }, { kind: 'safari-ios' }).steps,
    blockedSteps('safari-ios'),
  )
  assert.equal(
    studentPlaceState({ state: PLACE_GRANTED, accuracy: 8.4 }).label,
    'Location ready · accuracy 8 m',
  )
})

test('the website explains that Bluetooth belongs to the app', () => {
  assert.equal(
    WEB_BLUETOOTH_LINE,
    'Bluetooth works only in the Android app; on the website students use the 4-digit code',
  )
})

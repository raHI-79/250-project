import assert from 'node:assert/strict'
import test from 'node:test'
import {
  BEST_WINDOW_MS,
  GOOD_ENOUGH_M,
  PLACE_ASKING,
  PLACE_DENIED,
  PLACE_TIMEOUT,
  PLACE_WEAK,
  WEB_ACCURACY_LIMIT_M,
  betterReading,
  placeHelp,
  readBestPosition,
  studentPlaceState,
} from './webLocation.js'

const fix = (accuracy) => ({ coords: { latitude: 24.9169, longitude: 91.8316, accuracy } })

const fakeGeo = (script) => {
  const state = { cleared: [], options: null }
  globalThis.navigator = {
    geolocation: {
      watchPosition(onFix, onError, options) {
        state.options = options
        script(onFix, onError)
        return 7
      },
      clearWatch(id) {
        state.cleared.push(id)
      },
    },
    permissions: { query: async () => ({ state: 'denied' }) },
  }
  return state
}

test('the reading window is fifteen seconds, stops at 30 m and the website takes 100 m', () => {
  assert.equal(BEST_WINDOW_MS, 15000)
  assert.equal(GOOD_ENOUGH_M, 30)
  assert.equal(WEB_ACCURACY_LIMIT_M, 100)
})

test('the most accurate reading is kept', () => {
  const wide = { accuracy_m: 120 }
  const near = { accuracy_m: 42 }
  assert.equal(betterReading(null, wide), wide)
  assert.equal(betterReading(wide, near), near)
  assert.equal(betterReading(near, wide), near)
  assert.equal(betterReading(near, null), near)
})

test('readings are watched with high accuracy and the best one wins when the window ends', async () => {
  const seen = []
  const state = fakeGeo((onFix) => {
    onFix(fix(140))
    onFix(fix(65))
    onFix(fix(90))
  })
  const result = await readBestPosition({ windowMs: 40, onReading: (best) => seen.push(best.accuracy_m) })
  assert.equal(result.ok, true)
  assert.equal(result.position.accuracy_m, 65)
  assert.deepEqual(seen, [140, 65, 65])
  assert.equal(state.options.enableHighAccuracy, true)
  assert.equal(state.options.maximumAge, 0)
  assert.deepEqual(state.cleared, [7])
})

test('a reading of 30 m or better stops the search at once', async () => {
  const state = fakeGeo((onFix) => {
    onFix(fix(80))
    onFix(fix(28))
    onFix(fix(5))
  })
  const started = Date.now()
  const result = await readBestPosition({ windowMs: 5000 })
  assert.ok(Date.now() - started < 1000, 'the search waited for the whole window')
  assert.equal(result.position.accuracy_m, 28)
  assert.deepEqual(state.cleared, [7])
})

test('no reading at all ends as a timeout, a refusal as blocked', async () => {
  fakeGeo(() => {})
  assert.deepEqual(await readBestPosition({ windowMs: 20 }), { ok: false, state: PLACE_TIMEOUT })
  fakeGeo((onFix, onError) => onError({ code: 1, message: 'User denied Geolocation' }))
  assert.deepEqual(await readBestPosition({ windowMs: 1000 }), { ok: false, state: PLACE_DENIED })
  fakeGeo((onFix, onError) => {
    onFix(fix(70))
    onError({ code: 2, message: 'Position unavailable' })
  })
  const kept = await readBestPosition({ windowMs: 1000 })
  assert.equal(kept.position.accuracy_m, 70, 'a later error threw away a good reading')
})

test('the search shows its live accuracy and a weak reading offers Try again', () => {
  assert.equal(placeHelp({ state: PLACE_ASKING }).label, 'Finding your position…')
  assert.equal(placeHelp({ state: PLACE_ASKING, accuracy: 41.6 }).label, 'Finding your position… ±42 m')
  const weak = studentPlaceState({ state: PLACE_WEAK, accuracy: 150 })
  assert.equal(weak.label, 'Weak GPS signal · ±150 m')
  assert.equal(weak.fix, 'retry')
  assert.match(weak.reason, /±100 m or better/)
})

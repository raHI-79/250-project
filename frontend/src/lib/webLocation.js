export const PLACE_IDLE = 'idle'

export const PLACE_ASKING = 'asking'

export const PLACE_GRANTED = 'granted'

export const PLACE_DENIED = 'denied'

export const PLACE_OFF = 'off'

export const PLACE_TIMEOUT = 'timeout'

export const PLACE_UNSUPPORTED = 'unsupported'

export const PLACE_WEAK = 'weak'

export const BEST_WINDOW_MS = 15000

export const GOOD_ENOUGH_M = 30

export const WEB_ACCURACY_LIMIT_M = 100

export const POSITION_FRESH_MS = 60000

export const WEB_BLUETOOTH_LINE =
  'Bluetooth works only in the Android app; on the website students use the 4-digit code'

const ERROR_DENIED = 1

const ERROR_UNAVAILABLE = 2

const ERROR_TIMEOUT = 3

export const browserKind = (agent = '', touchMac = false) => {
  const text = String(agent || '')
  const apple = /iPad|iPhone|iPod/.test(text) || (touchMac && /Macintosh/.test(text))
  if (apple) return /CriOS|FxiOS|EdgiOS/.test(text) ? 'ios-other' : 'safari-ios'
  if (/Firefox\//.test(text)) return 'firefox'
  if (/Android/i.test(text)) return 'chrome-android'
  if (/Chrome\/|Chromium\/|Edg\//.test(text)) return 'chrome-desktop'
  if (/Safari\//.test(text) && /Macintosh/.test(text)) return 'safari-mac'
  return 'other'
}

export const devicePlatform = (agent = '', touchMac = false) => {
  const text = String(agent || '')
  if (/iPad|iPhone|iPod/.test(text) || (touchMac && /Macintosh/.test(text))) return 'ios'
  if (/Android/i.test(text)) return 'android'
  if (/Macintosh|Mac OS X/.test(text)) return 'mac'
  if (/Windows/.test(text)) return 'windows'
  return 'other'
}

const BLOCKED_STEPS = {
  'chrome-android': [
    'Tap the lock icon in the address bar.',
    'Tap Permissions, then Location.',
    'Choose Allow, then reload this page.',
  ],
  'chrome-desktop': [
    'Click the icon to the left of the address in the address bar.',
    'Turn Location on, or open Site settings and set Location to Allow.',
    'Reload this page.',
  ],
  'safari-ios': [
    'Tap aA in the address bar, then Website Settings.',
    'Set Location to Allow, then reload this page.',
    'Still blocked? Open Settings → Privacy & Security → Location Services → Safari Websites and choose While Using the App.',
  ],
  'ios-other': [
    'Open Settings → Privacy & Security → Location Services.',
    'Pick this browser and choose While Using the App.',
    'Come back and reload this page.',
  ],
  'safari-mac': [
    'Open Safari → Settings → Websites → Location.',
    'Set this site to Allow.',
    'Reload this page.',
  ],
  firefox: [
    'Click or tap the lock icon in the address bar.',
    'Clear the blocked Location permission.',
    'Reload this page and choose Allow.',
  ],
  other: [
    'Open the site settings from the icon next to the address.',
    'Allow Location for this site.',
    'Reload this page.',
  ],
}

const OFF_STEPS = {
  android: 'Swipe down from the top of the screen and turn Location on, then tap Try again.',
  ios: 'Open Settings → Privacy & Security → Location Services and turn it on, then tap Try again.',
  mac: 'Open System Settings → Privacy & Security → Location Services, turn it on for your browser, then tap Try again.',
  windows: 'Open Settings → Privacy & security → Location, turn Location services on, then tap Try again.',
  other: 'Turn on the location service of this device, then tap Try again.',
}

export const blockedSteps = (kind) => BLOCKED_STEPS[kind] || BLOCKED_STEPS.other

export const offStep = (platform) => OFF_STEPS[platform] || OFF_STEPS.other

export const classifyLocationError = (code, message = '', permission = '') => {
  if (code === ERROR_DENIED) {
    return permission === 'granted' ? PLACE_OFF : PLACE_DENIED
  }
  if (code === ERROR_UNAVAILABLE) return PLACE_OFF
  if (code === ERROR_TIMEOUT) return PLACE_TIMEOUT
  if (/turned off|disabled|location services|location settings/i.test(String(message))) return PLACE_OFF
  return PLACE_OFF
}

export const freshness = (session, now) => {
  if (!session?.location_updated_at) return { label: 'Allowed · waiting for GPS', tone: 'idle' }
  const age = now.getTime() - new Date(session.location_updated_at).getTime()
  if (age > POSITION_FRESH_MS) return { label: 'Out of date', tone: 'warn' }
  const accuracy = Number(session.accuracy_m) || 0
  return {
    label: accuracy > 0 ? `Sending · accuracy ${accuracy} m` : 'Sending',
    tone: 'good',
  }
}

export const placeHelp = (place, env = {}) => {
  const state = place?.state || PLACE_IDLE
  if (state === PLACE_IDLE) {
    return {
      label: 'Not allowed yet',
      tone: 'bad',
      fix: 'allow',
      reason: 'Tap Allow location and choose Allow when the browser asks.',
    }
  }
  if (state === PLACE_ASKING) {
    const accuracy = Math.round(Number(place?.accuracy) || 0)
    return {
      label: accuracy > 0 ? `Finding your position… ±${accuracy} m` : 'Finding your position…',
      tone: 'idle',
      reason: 'Keep the phone still for a few seconds; the most accurate reading is kept.',
    }
  }
  if (state === PLACE_WEAK) {
    const accuracy = Math.round(Number(place?.accuracy) || 0)
    return {
      label: accuracy > 0 ? `Weak GPS signal · ±${accuracy} m` : 'Weak GPS signal',
      tone: 'warn',
      fix: 'retry',
      reason: `Move near a window or outside the corridor, then tap Try again. The website needs ±${WEB_ACCURACY_LIMIT_M} m or better.`,
    }
  }
  if (state === PLACE_DENIED) {
    return {
      label: 'Blocked in the browser',
      tone: 'bad',
      fix: 'retry',
      reason: 'Location is blocked for this site. To allow it:',
      steps: blockedSteps(env.kind),
    }
  }
  if (state === PLACE_OFF) {
    return {
      label: 'Location is off on this device',
      tone: 'bad',
      fix: 'retry',
      reason: offStep(env.platform),
    }
  }
  if (state === PLACE_TIMEOUT) {
    return {
      label: 'No position yet',
      tone: 'warn',
      fix: 'retry',
      reason: 'The position took too long. Move near a window, then tap Try again.',
    }
  }
  if (state === PLACE_UNSUPPORTED) {
    return {
      label: 'Not available',
      tone: 'bad',
      reason: env.insecure
        ? 'The browser shares a position only on a secure (https) page.'
        : 'This browser cannot share a position.',
    }
  }
  return null
}

export const webPositionState = (place, session, now, env = {}) => {
  if (session?.code_only) {
    return {
      label: 'Not checked · code only',
      tone: 'warn',
      reason: 'Marks are flagged No location for you to review.',
    }
  }
  if (place?.state === PLACE_GRANTED) return freshness(session, now)
  return placeHelp(place, env)
}

export const studentPlaceState = (place, env = {}) => {
  if (place?.state === PLACE_GRANTED) {
    const accuracy = Math.round(Number(place.accuracy) || 0)
    return {
      label: accuracy > 0 ? `Location ready · accuracy ${accuracy} m` : 'Location ready',
      tone: 'good',
    }
  }
  return placeHelp(place, env)
}

export const browserEnv = () => {
  if (typeof navigator === 'undefined') return { kind: 'other', platform: 'other', insecure: false }
  const touchMac = typeof document !== 'undefined' && 'ontouchend' in document
  return {
    kind: browserKind(navigator.userAgent, touchMac),
    platform: devicePlatform(navigator.userAgent, touchMac),
    insecure: typeof window !== 'undefined' && window.isSecureContext === false,
  }
}

export const locationAvailable = () =>
  typeof navigator !== 'undefined' &&
  Boolean(navigator.geolocation) &&
  !(typeof window !== 'undefined' && window.isSecureContext === false)

export const geolocationPermission = async () => {
  try {
    if (typeof navigator === 'undefined' || !navigator.permissions?.query) return 'unknown'
    const status = await navigator.permissions.query({ name: 'geolocation' })
    return status?.state || 'unknown'
  } catch {
    return 'unknown'
  }
}

export const betterReading = (best, next) => {
  if (!next) return best
  if (!best) return next
  return Number(next.accuracy_m) < Number(best.accuracy_m) ? next : best
}

const readingOf = (position) => ({
  lat: position.coords.latitude,
  lng: position.coords.longitude,
  accuracy_m: position.coords.accuracy,
})

export const readBestPosition = ({
  windowMs = BEST_WINDOW_MS,
  goodEnough = GOOD_ENOUGH_M,
  onReading,
} = {}) =>
  new Promise((resolve) => {
    if (!locationAvailable()) {
      resolve({ ok: false, state: PLACE_UNSUPPORTED })
      return
    }
    const geo = navigator.geolocation
    let best = null
    let done = false
    let watch = null
    let timer = null
    const finish = (result) => {
      if (done) return
      done = true
      if (watch !== null) geo.clearWatch(watch)
      if (timer !== null) clearTimeout(timer)
      resolve(result)
    }
    timer = setTimeout(
      () => finish(best ? { ok: true, position: best } : { ok: false, state: PLACE_TIMEOUT }),
      windowMs,
    )
    const id = geo.watchPosition(
      (position) => {
        best = betterReading(best, readingOf(position))
        onReading?.(best)
        if (Number(best.accuracy_m) <= goodEnough) finish({ ok: true, position: best })
      },
      async (failure) => {
        if (best) {
          finish({ ok: true, position: best })
          return
        }
        if (failure?.code === ERROR_TIMEOUT) return
        const permission = failure?.code === ERROR_DENIED ? await geolocationPermission() : ''
        finish({ ok: false, state: classifyLocationError(failure?.code, failure?.message, permission) })
      },
      { enableHighAccuracy: true, maximumAge: 0, timeout: windowMs },
    )
    if (done) geo.clearWatch(id)
    else watch = id
  })

export const readBrowserPosition = readBestPosition

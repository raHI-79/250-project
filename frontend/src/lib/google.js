const SCRIPT_SRC = 'https://accounts.google.com/gsi/client'

const SOCIAL_PROVIDER = 'google'

export const GOOGLE_UNREACHABLE = 'Google sign-in could not be reached. Check your connection.'

export const GOOGLE_NOT_CONFIGURED = 'Google sign-in is not set up for this portal yet.'

export const GOOGLE_WRONG_ORIGIN =
  'Google refused this address. The portal address has to be listed in the Google console.'

export const GOOGLE_POPUP_BLOCKED =
  'Your browser blocked the Google window. Allow popups for this site and try again.'

export const GOOGLE_CANCELLED = 'Google sign-in was cancelled.'

let pending = null

let social = null

export const GOOGLE_APP_CONSOLE =
  'Google refused this app. The Android OAuth client for edu.sust.cse.portal with this signing SHA-1 is missing from the Google project.'

export const GOOGLE_NO_ACCOUNT = (domain) =>
  domain
    ? `No @${domain} Google account is signed in on this phone. Add it in Android settings and try again.`
    : 'No Google account is signed in on this phone. Add one in Android settings and try again.'

export const googleMessage = (reason) => {
  const raw = String(reason?.message || reason || '').trim()
  const text = raw.toLowerCase()
  if (!text) return GOOGLE_UNREACHABLE
  if (text.includes('origin') || text.includes('client id') || text.includes('invalid_client')) {
    return GOOGLE_WRONG_ORIGIN
  }
  if (text.includes('popup') || text.includes('window')) return GOOGLE_POPUP_BLOCKED
  if (text.includes('cancel') || text.includes('dismiss') || text.includes('closed')) {
    return GOOGLE_CANCELLED
  }
  return raw || GOOGLE_UNREACHABLE
}

export const nativeGoogleMessage = (reason, hostedDomain) => {
  const raw = String(reason?.message || reason || '').trim()
  const text = raw.toLowerCase()
  if (!raw) return GOOGLE_UNREACHABLE
  if (text.includes('28433') || text.includes('developer_error') || text.includes('10:')) {
    return `${GOOGLE_APP_CONSOLE} Google said: ${raw}`
  }
  if (
    text.includes('nocredential') ||
    text.includes('no credentials available') ||
    text.includes('16:')
  ) {
    return `${GOOGLE_NO_ACCOUNT(hostedDomain)} Google said: ${raw}`
  }
  if (text.includes('cancel') || text.includes('dismiss') || text.includes('closed')) {
    return GOOGLE_CANCELLED
  }
  return raw
}

export const loadGoogleIdentity = () => {
  if (window.google?.accounts?.id) return Promise.resolve(window.google.accounts.id)
  if (pending) return pending
  pending = new Promise((resolve, reject) => {
    const ready = () => {
      if (window.google?.accounts?.id) resolve(window.google.accounts.id)
      else reject(new Error(GOOGLE_UNREACHABLE))
    }
    const failed = () => {
      pending = null
      reject(new Error(GOOGLE_UNREACHABLE))
    }
    const existing = document.querySelector(`script[src="${SCRIPT_SRC}"]`)
    if (existing) {
      existing.addEventListener('load', ready, { once: true })
      existing.addEventListener('error', failed, { once: true })
      if (window.google?.accounts?.id) ready()
      return
    }
    const script = document.createElement('script')
    script.src = SCRIPT_SRC
    script.async = true
    script.defer = true
    script.addEventListener('load', ready, { once: true })
    script.addEventListener('error', failed, { once: true })
    document.head.appendChild(script)
  })
  return pending
}

const socialLogin = async (clientId, hostedDomain) => {
  const { SocialLogin } = await import('@capgo/capacitor-social-login')
  const key = `${clientId}|${hostedDomain || ''}`
  if (social !== key) {
    social = null
    await SocialLogin.initialize({
      google: {
        webClientId: clientId,
        mode: 'online',
        hostedDomain: hostedDomain || undefined,
      },
    })
    social = key
  }
  return SocialLogin
}

export const nativeGoogleCredential = async (clientId, hostedDomain) => {
  if (!clientId) throw new Error(GOOGLE_NOT_CONFIGURED)
  const login = await socialLogin(clientId, hostedDomain)
  const answer = await login.login({ provider: SOCIAL_PROVIDER, options: {} })
  const token = answer?.result?.idToken || answer?.result?.credential?.idToken || answer?.idToken || ''
  if (!token) throw new Error('Google returned no identity token for this account.')
  return token
}

import { isAndroidApp } from './native'

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

export const googleMessage = (reason) => {
  const text = String(reason?.message || reason || '').toLowerCase()
  if (!text) return GOOGLE_UNREACHABLE
  if (text.includes('origin') || text.includes('client id') || text.includes('invalid_client')) {
    return GOOGLE_WRONG_ORIGIN
  }
  if (text.includes('popup') || text.includes('window')) return GOOGLE_POPUP_BLOCKED
  if (text.includes('cancel') || text.includes('dismiss') || text.includes('closed')) {
    return GOOGLE_CANCELLED
  }
  return reason?.message || GOOGLE_UNREACHABLE
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
    await SocialLogin.initialize({
      google: { webClientId: clientId, hostedDomain: hostedDomain || undefined },
    })
    social = key
  }
  return SocialLogin
}

export const nativeGoogleCredential = async (clientId, hostedDomain) => {
  const login = await socialLogin(clientId, hostedDomain)
  const answer = await login.login({ provider: SOCIAL_PROVIDER, options: {} })
  const token =
    answer?.result?.idToken ||
    answer?.result?.credential?.idToken ||
    answer?.idToken ||
    ''
  if (!token) throw new Error('Google did not return an identity token.')
  return token
}

export const nativeGoogleSignOut = async (clientId) => {
  if (!isAndroidApp() || !clientId) return
  try {
    const login = await socialLogin(clientId)
    await login.logout({ provider: SOCIAL_PROVIDER })
  } catch {
    social = null
  }
}

export const googleSupported = () => typeof window !== 'undefined'

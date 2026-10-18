const SCRIPT_SRC = 'https://accounts.google.com/gsi/client'

export const GOOGLE_CLIENT_ID = import.meta.env.VITE_GOOGLE_CLIENT_ID || ''

let pending = null

export const loadGoogleIdentity = () => {
  if (window.google?.accounts?.id) return Promise.resolve(window.google.accounts.id)
  if (pending) return pending
  pending = new Promise((resolve, reject) => {
    const ready = () => {
      if (window.google?.accounts?.id) resolve(window.google.accounts.id)
      else reject(new Error('Google sign-in could not start.'))
    }
    const failed = () => {
      pending = null
      reject(new Error('Google sign-in could not be reached.'))
    }
    const existing = document.querySelector(`script[src="${SCRIPT_SRC}"]`)
    if (existing) {
      existing.addEventListener('load', ready, { once: true })
      existing.addEventListener('error', failed, { once: true })
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

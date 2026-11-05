export const SCRIPT_URL = 'https://challenges.cloudflare.com/turnstile/v0/api.js?render=explicit'

export const SCRIPT_ID = 'turnstile-script'

const READY_TIMEOUT = 12000

const POLL_EVERY = 100

const LOAD_FAILED = 'The check could not load. Please reload the page.'

let pending = null

export const loadTurnstile = () => {
  if (typeof window === 'undefined') return Promise.reject(new Error(LOAD_FAILED))
  if (window.turnstile) return Promise.resolve(window.turnstile)
  if (pending) return pending
  pending = new Promise((resolve, reject) => {
    let settled = false
    let timer = 0
    const finish = (value, failure) => {
      if (settled) return
      settled = true
      window.clearInterval(timer)
      if (failure) {
        pending = null
        reject(failure)
        return
      }
      resolve(value)
    }
    let script = document.getElementById(SCRIPT_ID)
    if (!script) {
      script = document.createElement('script')
      script.id = SCRIPT_ID
      script.src = SCRIPT_URL
      script.async = true
      script.defer = true
      script.addEventListener('error', () => finish(null, new Error(LOAD_FAILED)))
      document.head.appendChild(script)
    } else {
      script.addEventListener('error', () => finish(null, new Error(LOAD_FAILED)))
    }
    const started = Date.now()
    timer = window.setInterval(() => {
      if (window.turnstile) {
        finish(window.turnstile)
        return
      }
      if (Date.now() - started > READY_TIMEOUT) finish(null, new Error(LOAD_FAILED))
    }, POLL_EVERY)
  })
  return pending
}

export default loadTurnstile

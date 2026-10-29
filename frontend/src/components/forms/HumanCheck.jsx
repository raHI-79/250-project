import { useEffect, useRef, useState } from 'react'
import Alert from '../ui/Alert'
import { useTheme } from '../../context/theme'

const SCRIPT_URL = 'https://challenges.cloudflare.com/turnstile/v0/api.js?render=explicit'

const SCRIPT_ID = 'turnstile-script'

const loadTurnstile = () =>
  new Promise((resolve, reject) => {
    if (window.turnstile) {
      resolve(window.turnstile)
      return
    }
    let script = document.getElementById(SCRIPT_ID)
    if (!script) {
      script = document.createElement('script')
      script.id = SCRIPT_ID
      script.src = SCRIPT_URL
      script.async = true
      script.defer = true
      document.head.appendChild(script)
    }
    script.addEventListener('load', () => resolve(window.turnstile))
    script.addEventListener('error', () => reject(new Error('The check could not load.')))
  })

export default function HumanCheck({ siteKey, round = 0, onToken, className }) {
  const holder = useRef(null)
  const [problem, setProblem] = useState('')
  const { resolved } = useTheme()
  useEffect(() => {
    let live = true
    let widget = null
    setProblem('')
    loadTurnstile()
      .then((turnstile) => {
        if (!live || !turnstile || !holder.current) return
        widget = turnstile.render(holder.current, {
          sitekey: siteKey,
          theme: resolved,
          size: 'normal',
          callback: (token) => {
            if (live) onToken(token)
          },
          'error-callback': () => {
            if (live) setProblem('The check could not run. Please try again.')
          },
          'expired-callback': () => {
            if (!live) return
            onToken('')
            if (widget) window.turnstile?.reset(widget)
          },
        })
      })
      .catch((failure) => {
        if (live) setProblem(failure.message)
      })
    return () => {
      live = false
      if (widget && window.turnstile) window.turnstile.remove(widget)
    }
  }, [siteKey, round, resolved, onToken])
  return (
    <div className={className}>
      <div ref={holder} className="flex justify-center" />
      {problem && (
        <Alert tone="error" className="mt-3">
          {problem}
        </Alert>
      )}
    </div>
  )
}

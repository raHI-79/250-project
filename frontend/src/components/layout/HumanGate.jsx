import { useEffect, useRef, useState } from 'react'
import CampusBackdrop from './CampusBackdrop'
import LogoBadge from '../ui/LogoBadge'
import Alert from '../ui/Alert'
import { DEPARTMENT, UNIVERSITY } from '../ui/IdentityBar'
import { HERO_IMAGES } from '../../lib/campus'
import { verifyHuman } from '../../lib/human'

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
    script.addEventListener('error', () => reject(new Error('The human check could not load.')))
  })

export default function HumanGate({ siteKey, onPass }) {
  const holder = useRef(null)
  const [problem, setProblem] = useState('')
  useEffect(() => {
    let live = true
    let widget = null
    loadTurnstile()
      .then((turnstile) => {
        if (!live || !turnstile || !holder.current) return
        widget = turnstile.render(holder.current, {
          sitekey: siteKey,
          theme: 'auto',
          callback: (token) => {
            verifyHuman(token)
              .then(() => {
                if (live) onPass()
              })
              .catch((failure) => {
                if (!live) return
                setProblem(failure.message)
                turnstile.reset(widget)
              })
          },
          'error-callback': () => {
            if (live) setProblem('The human check could not run. Please reload the page.')
          },
          'expired-callback': () => {
            if (widget) turnstile.reset(widget)
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
  }, [siteKey, onPass])
  return (
    <div className="relative flex min-h-dvh items-center justify-center px-4">
      <CampusBackdrop
        images={HERO_IMAGES}
        overlay="bg-linear-to-b from-maroon-950/55 via-maroon-900/40 to-maroon-950/65"
        className="fixed"
      />
      <div className="glass-auth animate-fade-in-up relative w-full max-w-sm rounded-[1.5rem] p-7 text-center">
        <div className="flex justify-center">
          <LogoBadge size="lg" tone="glass" />
        </div>
        <p className="eyebrow mt-4">{UNIVERSITY}</p>
        <h1 className="font-display mt-1.5 text-title font-bold text-ink">Quick human check</h1>
        <p className="mt-2 text-sm text-muted">
          One short check keeps {DEPARTMENT} safe from automated traffic.
        </p>
        <div ref={holder} className="mt-6 flex justify-center" />
        {problem && (
          <Alert tone="error" className="mt-4 text-left">
            {problem}
          </Alert>
        )}
      </div>
    </div>
  )
}

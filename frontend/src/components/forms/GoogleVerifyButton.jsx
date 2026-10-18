import { useEffect, useRef, useState } from 'react'
import { GOOGLE_CLIENT_ID, loadGoogleIdentity } from '../../lib/google'

export default function GoogleVerifyButton({ hostedDomain, onCredential, onError }) {
  const holder = useRef(null)
  const handler = useRef(onCredential)
  const [failed, setFailed] = useState('')
  useEffect(() => {
    handler.current = onCredential
  }, [onCredential])
  useEffect(() => {
    if (!GOOGLE_CLIENT_ID) {
      setFailed('Google sign-in is not configured for this build.')
      return undefined
    }
    let active = true
    loadGoogleIdentity()
      .then((identity) => {
        if (!active || !holder.current) return
        identity.initialize({
          client_id: GOOGLE_CLIENT_ID,
          hd: hostedDomain || undefined,
          auto_select: false,
          cancel_on_tap_outside: true,
          callback: (response) => {
            if (response?.credential) handler.current?.(response.credential)
            else onError?.('Google did not return a credential.')
          },
        })
        holder.current.replaceChildren()
        identity.renderButton(holder.current, {
          theme: 'outline',
          size: 'large',
          shape: 'pill',
          text: 'continue_with',
          logo_alignment: 'center',
        })
      })
      .catch((failure) => {
        if (active) setFailed(failure.message)
      })
    return () => {
      active = false
    }
  }, [hostedDomain, onError])
  if (failed) return <p className="text-xs text-accent-700">{failed}</p>
  return <div ref={holder} className="flex min-h-11 justify-center" />
}

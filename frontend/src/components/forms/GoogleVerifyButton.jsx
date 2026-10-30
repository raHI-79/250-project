import { useCallback, useEffect, useRef, useState } from 'react'
import Button from '../ui/Button'
import { useAuthConfig } from '../../lib/hooks'
import { isAndroidApp } from '../../lib/native'
import {
  GOOGLE_NOT_CONFIGURED,
  googleMessage,
  loadGoogleIdentity,
  nativeGoogleCredential,
} from '../../lib/google'

function NativeGoogleButton({ clientId, hostedDomain, onCredential, onError }) {
  const [busy, setBusy] = useState(false)
  const start = async () => {
    setBusy(true)
    try {
      const token = await nativeGoogleCredential(clientId, hostedDomain)
      onCredential?.(token)
    } catch (reason) {
      onError?.(googleMessage(reason))
    } finally {
      setBusy(false)
    }
  }
  return (
    <Button variant="secondary" size="lg" loading={busy} onClick={start} className="w-full">
      Continue with Google
    </Button>
  )
}

function WebGoogleButton({ clientId, hostedDomain, onCredential, onError }) {
  const holder = useRef(null)
  const handler = useRef(onCredential)
  const [failed, setFailed] = useState('')
  useEffect(() => {
    handler.current = onCredential
  }, [onCredential])
  const refused = useCallback(
    (reason) => {
      const text = googleMessage(reason)
      setFailed(text)
      onError?.(text)
    },
    [onError],
  )
  useEffect(() => {
    let active = true
    setFailed('')
    loadGoogleIdentity()
      .then((identity) => {
        if (!active || !holder.current) return
        identity.initialize({
          client_id: clientId,
          hd: hostedDomain || undefined,
          auto_select: false,
          cancel_on_tap_outside: true,
          use_fedcm_for_prompt: false,
          callback: (response) => {
            if (response?.credential) handler.current?.(response.credential)
            else refused(new Error('Google did not return a credential.'))
          },
          error_callback: (reason) => refused(reason),
        })
        holder.current.replaceChildren()
        identity.renderButton(holder.current, {
          theme: 'outline',
          size: 'large',
          shape: 'pill',
          text: 'continue_with',
          logo_alignment: 'center',
          width: Math.min(holder.current.offsetWidth || 320, 400),
        })
      })
      .catch((reason) => {
        if (active) refused(reason)
      })
    return () => {
      active = false
    }
  }, [clientId, hostedDomain, refused])
  return (
    <div>
      <div ref={holder} className="flex min-h-11 justify-center" />
      {failed && <p className="mt-2 text-center text-xs text-danger-600">{failed}</p>}
    </div>
  )
}

export default function GoogleVerifyButton({ hostedDomain, onCredential, onError }) {
  const { googleClientId, googleEnabled, loaded } = useAuthConfig()
  if (!loaded) return <div className="min-h-11" />
  if (!googleEnabled || !googleClientId) {
    return <p className="text-center text-xs text-muted">{GOOGLE_NOT_CONFIGURED}</p>
  }
  const Chosen = isAndroidApp() ? NativeGoogleButton : WebGoogleButton
  return (
    <Chosen
      clientId={googleClientId}
      hostedDomain={hostedDomain}
      onCredential={onCredential}
      onError={onError}
    />
  )
}

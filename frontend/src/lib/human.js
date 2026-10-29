import { useCallback, useEffect, useState } from 'react'
import api, { HUMAN_EVENT } from './api'
import { readJson, remove, writeJson } from './storage'
import { isStandalone } from './standalone'

const PASS_KEY = 'sustcse.human'

export const HUMAN_HEADER = 'X-Human-Pass'

export const humanPass = () => {
  const stored = readJson(PASS_KEY)
  if (!stored?.pass || !stored?.expires_at) return null
  if (new Date(stored.expires_at).getTime() <= Date.now()) {
    remove(PASS_KEY)
    return null
  }
  return stored.pass
}

export const clearHumanPass = () => remove(PASS_KEY)

export const saveHumanPass = (pass, expiresAt) => writeJson(PASS_KEY, { pass, expires_at: expiresAt })

export const verifyHuman = async (token) => {
  const data = await api.post(
    '/api/human/verify',
    { token, standalone: isStandalone() },
    { auth: false },
  )
  if (!data?.pass) throw new Error('The human check did not pass. Please try again.')
  saveHumanPass(data.pass, data.expires_at)
  return data.pass
}

export const useHumanGate = () => {
  const [config, setConfig] = useState(null)
  const [passed, setPassed] = useState(() => Boolean(humanPass()))
  useEffect(() => {
    let live = true
    api
      .get('/api/auth/config', { auth: false })
      .then((data) => {
        if (live) setConfig(data || {})
      })
      .catch(() => {
        if (live) setConfig({})
      })
    return () => {
      live = false
    }
  }, [])
  const pass = useCallback(() => setPassed(true), [])
  useEffect(() => {
    const onLost = () => setPassed(false)
    window.addEventListener(HUMAN_EVENT, onLost)
    return () => window.removeEventListener(HUMAN_EVENT, onLost)
  }, [])
  const needed = Boolean(
    config?.human_check_enabled && config?.turnstile_site_key && !passed,
  )
  return { ready: config !== null, needed, siteKey: config?.turnstile_site_key || '', pass }
}

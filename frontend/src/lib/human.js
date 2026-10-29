import { useCallback, useEffect, useState } from 'react'
import api from './api'
import { readJson, remove, writeJson } from './storage'
import { isNative } from './native'
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

export const useHumanCheck = () => {
  const [siteKey, setSiteKey] = useState('')
  const [ready, setReady] = useState(() => isNative())
  const [passed, setPassed] = useState(() => Boolean(humanPass()))
  const [token, setToken] = useState('')
  const [round, setRound] = useState(0)
  useEffect(() => {
    if (isNative()) return undefined
    let live = true
    const settle = (key) => {
      if (!live) return
      setSiteKey(key)
      setReady(true)
    }
    api
      .get('/api/auth/config', { auth: false })
      .then((data) => settle(data?.human_check_enabled ? data?.turnstile_site_key || '' : ''))
      .catch(() => settle(''))
    return () => {
      live = false
    }
  }, [])
  const reset = useCallback(() => {
    setToken('')
    setPassed(Boolean(humanPass()))
    setRound((current) => current + 1)
  }, [])
  const needed = !isNative() && Boolean(siteKey)
  const confirm = useCallback(async () => {
    if (!needed || humanPass()) return
    if (!token) throw new Error('Please complete the check below.')
    await verifyHuman(token)
    setPassed(true)
  }, [needed, token])
  return {
    ready,
    siteKey,
    token,
    setToken,
    reset,
    round,
    confirm,
    needed: needed && !passed,
    blocked: !ready || (needed && !passed && !token),
  }
}

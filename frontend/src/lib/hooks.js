import { useCallback, useEffect, useState } from 'react'
import api from './api'

const NO_QUERY = {}

export const useResource = (path, query) => {
  const key = JSON.stringify(query || NO_QUERY)
  const [data, setData] = useState(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [attempt, setAttempt] = useState(0)
  const reload = useCallback(() => setAttempt((current) => current + 1), [])
  useEffect(() => {
    if (!path) {
      setLoading(false)
      return undefined
    }
    const controller = new AbortController()
    let active = true
    setLoading(true)
    setError('')
    api
      .get(path, { query: JSON.parse(key), signal: controller.signal })
      .then((result) => {
        if (active) setData(result)
      })
      .catch((failure) => {
        if (!active || failure?.name === 'AbortError') return
        setError(failure.message)
      })
      .finally(() => {
        if (active) setLoading(false)
      })
    return () => {
      active = false
      controller.abort()
    }
  }, [path, key, attempt])
  return { data, loading, error, reload }
}

export const useDebounced = (value, delay = 300) => {
  const [settled, setSettled] = useState(value)
  useEffect(() => {
    const timer = window.setTimeout(() => setSettled(value), delay)
    return () => window.clearTimeout(timer)
  }, [value, delay])
  return settled
}

export const useCountUp = (target, duration = 900) => {
  const [value, setValue] = useState(0)
  useEffect(() => {
    const goal = Number(target) || 0
    if (goal <= 0) {
      setValue(goal)
      return undefined
    }
    const reduced = window.matchMedia?.('(prefers-reduced-motion: reduce)').matches
    if (reduced) {
      setValue(goal)
      return undefined
    }
    let frame = 0
    const started = performance.now()
    const step = (now) => {
      const progress = Math.min((now - started) / duration, 1)
      setValue(Math.round(goal * (1 - (1 - progress) ** 3)))
      if (progress < 1) frame = window.requestAnimationFrame(step)
    }
    frame = window.requestAnimationFrame(step)
    return () => window.cancelAnimationFrame(frame)
  }, [target, duration])
  return value
}

let authConfig = null

export const useAuthConfig = () => {
  const [config, setConfig] = useState(authConfig)
  useEffect(() => {
    if (authConfig) return undefined
    let active = true
    api
      .get('/api/auth/config', { auth: false })
      .then((data) => {
        authConfig = data || {}
        if (active) setConfig(authConfig)
      })
      .catch(() => null)
    return () => {
      active = false
    }
  }, [])
  return {
    studentEmailDomain: config?.student_email_domain || '',
    googleEnabled: Boolean(config?.google_enabled),
    loaded: Boolean(config),
  }
}

export const useCountdown = (initial = 0) => {
  const [seconds, setSeconds] = useState(initial)
  useEffect(() => {
    if (seconds <= 0) return undefined
    const timer = window.setTimeout(() => setSeconds((current) => current - 1), 1000)
    return () => window.clearTimeout(timer)
  }, [seconds])
  return [seconds, setSeconds]
}

export const usePolling = (action, interval) => {
  useEffect(() => {
    if (!action || !interval) return undefined
    let timer = window.setInterval(action, interval)
    const onVisibility = () => {
      window.clearInterval(timer)
      if (document.visibilityState === 'visible') {
        action()
        timer = window.setInterval(action, interval)
      }
    }
    document.addEventListener('visibilitychange', onVisibility)
    return () => {
      window.clearInterval(timer)
      document.removeEventListener('visibilitychange', onVisibility)
    }
  }, [action, interval])
}

const UNREAD_EVENT = 'sustcse.unread'

export const refreshUnread = () => window.dispatchEvent(new Event(UNREAD_EVENT))

export const useUnreadMessages = (enabled = true) => {
  const [count, setCount] = useState(0)
  const load = useCallback(() => {
    if (!enabled) return undefined
    return api
      .get('/api/messages/unread-count')
      .then((data) => setCount(Number(data?.count) || 0))
      .catch(() => null)
  }, [enabled])
  useEffect(() => {
    load()
    window.addEventListener(UNREAD_EVENT, load)
    return () => window.removeEventListener(UNREAD_EVENT, load)
  }, [load])
  usePolling(enabled ? load : null, 20000)
  return count
}

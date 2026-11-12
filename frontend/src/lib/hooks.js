import { useCallback, useEffect, useRef, useState } from 'react'
import { useSearchParams } from 'react-router-dom'
import api from './api'
import { resourceSource, resourceVersion, resourceView, watchResources } from './refresh'
import { shiftDate } from './time'

const NO_QUERY = {}

export const usePeriods = () => {
  const { data } = useResource('/api/lookups/periods')
  return data?.items || []
}

export const useResourceVersion = () => {
  const [version, setVersion] = useState(resourceVersion)
  useEffect(() => watchResources(setVersion), [])
  return version
}

export const useResource = (path, query) => {
  const key = JSON.stringify(query || NO_QUERY)
  const fresh = useResourceVersion()
  const source = resourceSource(path, key)
  const [state, setState] = useState({ source: '', data: null, error: '' })
  const [busy, setBusy] = useState(Boolean(path))
  const [attempt, setAttempt] = useState(0)
  const waiting = useRef([])
  const settle = useCallback(() => {
    const waiters = waiting.current
    waiting.current = []
    waiters.forEach((resolve) => resolve())
  }, [])
  const reload = useCallback(() => {
    setAttempt((current) => current + 1)
    return new Promise((resolve) => waiting.current.push(resolve))
  }, [])
  useEffect(() => {
    if (!path) {
      setBusy(false)
      settle()
      return undefined
    }
    const own = resourceSource(path, key)
    const controller = new AbortController()
    let active = true
    setBusy(true)
    setState((current) => (current.source === own ? current : { source: own, data: null, error: '' }))
    api
      .get(path, { query: JSON.parse(key), signal: controller.signal })
      .then((result) => {
        if (active) setState({ source: own, data: result, error: '' })
      })
      .catch((failure) => {
        if (!active || failure?.name === 'AbortError') return
        setState((current) => ({
          source: own,
          data: current.source === own ? current.data : null,
          error: failure.message,
          status: failure.status || 0,
        }))
      })
      .finally(() => {
        if (active) setBusy(false)
        settle()
      })
    return () => {
      active = false
      controller.abort()
    }
  }, [path, key, attempt, fresh, settle])
  const view = resourceView(state, source, busy)
  return { ...view, reload }
}

export const useHighlight = (ready, key = 'highlight') => {
  const [params] = useSearchParams()
  const target = params.get(key) || ''
  const [active, setActive] = useState('')
  const [missing, setMissing] = useState(false)
  const nodes = useRef(new Map())
  const register = useCallback(
    (id) => (node) => {
      const name = String(id)
      if (node) nodes.current.set(name, node)
      else nodes.current.delete(name)
    },
    [],
  )
  useEffect(() => {
    if (!target || !ready) {
      setMissing(false)
      return undefined
    }
    const node = nodes.current.get(target)
    setMissing(!node)
    if (!node) return undefined
    node.scrollIntoView({ behavior: 'smooth', block: 'center' })
    setActive(target)
    const timer = window.setTimeout(() => setActive(''), 2500)
    return () => window.clearTimeout(timer)
  }, [target, ready])
  const highlighted = useCallback((id) => String(id) === active, [active])
  return { register, highlighted, target, missing }
}

export const useQueryTab = (tabs, fallback, key = 'tab') => {
  const [params, setParams] = useSearchParams()
  const raw = params.get(key) || ''
  const value = tabs.includes(raw) ? raw : fallback
  const setValue = useCallback(
    (next) => {
      setParams(
        (current) => {
          const copy = new URLSearchParams(current)
          copy.set(key, next)
          copy.delete('item')
          copy.delete('highlight')
          return copy
        },
        { replace: true },
      )
    },
    [key, setParams],
  )
  return [value, setValue]
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
    googleClientId: config?.google_client_id || '',
    loaded: Boolean(config),
  }
}

export const useNow = (interval = 30000) => {
  const [now, setNow] = useState(() => new Date())
  useEffect(() => {
    const timer = window.setInterval(() => setNow(new Date()), interval)
    return () => window.clearInterval(timer)
  }, [interval])
  return now
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

export const useMoreDays = (path, after, horizon) => {
  const [days, setDays] = useState([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [done, setDone] = useState(false)
  const reset = useCallback(() => {
    setDays([])
    setError('')
    setDone(false)
  }, [])
  useEffect(() => {
    reset()
  }, [path, after, reset])
  const last = days.length > 0 ? days[days.length - 1].date : after
  const exhausted = done || !path || !last || (horizon && last >= horizon)
  const more = useCallback(async () => {
    if (!path || !last) return
    setLoading(true)
    setError('')
    try {
      const from = shiftDate(last, 1)
      const result = await api.get(path, {
        query: { from, to: horizon || from, limit_days: 1 },
      })
      const next = result?.days || []
      if (next.length === 0) setDone(true)
      else setDays((current) => [...current, ...next])
      if (!result?.has_more && next.length <= 1) setDone(true)
    } catch (failure) {
      setError(failure.message)
    } finally {
      setLoading(false)
    }
  }, [path, last, horizon])
  return { days, loading, error, exhausted, more, reset }
}

const matchesQuery = (query) =>
  typeof window !== 'undefined' &&
  typeof window.matchMedia === 'function' &&
  window.matchMedia(query).matches

export const useMediaQuery = (query) => {
  const [matches, setMatches] = useState(() => matchesQuery(query))
  useEffect(() => {
    if (typeof window === 'undefined' || typeof window.matchMedia !== 'function') return undefined
    const list = window.matchMedia(query)
    const update = () => setMatches(list.matches)
    update()
    list.addEventListener?.('change', update)
    return () => list.removeEventListener?.('change', update)
  }, [query])
  return matches
}

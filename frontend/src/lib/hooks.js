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

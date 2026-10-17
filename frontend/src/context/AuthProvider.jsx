import { useCallback, useMemo, useState } from 'react'
import api, { SESSION_KEY } from '../lib/api'
import { readJson, remove, writeJson } from '../lib/storage'
import { AuthContext } from './auth'

const loadSession = () => {
  const stored = readJson(SESSION_KEY)
  if (stored?.token && stored?.user) return stored
  return { token: null, user: null }
}

export default function AuthProvider({ children }) {
  const [session, setSession] = useState(loadSession)
  const persist = useCallback((next) => {
    writeJson(SESSION_KEY, next)
    setSession(next)
  }, [])
  const clear = useCallback(() => {
    remove(SESSION_KEY)
    setSession({ token: null, user: null })
  }, [])
  const login = useCallback(
    async (identifier, password) => {
      const data = await api.post('/api/auth/login', { identifier, password }, { auth: false })
      const next = { token: data.token, user: data.user }
      persist(next)
      return next.user
    },
    [persist],
  )
  const adminLogin = useCallback(
    async (identifier, password) => {
      const data = await api.post(
        '/api/auth/admin/login',
        { identifier, password },
        { auth: false },
      )
      const next = { token: data.token, user: { ...data.user, role: 'admin' } }
      persist(next)
      return next.user
    },
    [persist],
  )
  const logout = useCallback(async () => {
    await api.post('/api/auth/logout').catch(() => null)
    clear()
  }, [clear])
  const value = useMemo(
    () => ({
      token: session.token,
      user: session.user,
      role: session.user?.role || null,
      isAuthenticated: Boolean(session.token && session.user),
      login,
      adminLogin,
      logout,
    }),
    [session, login, adminLogin, logout],
  )
  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}

import { useState } from 'react'
import { Link, Navigate, useLocation, useNavigate } from 'react-router-dom'
import Alert from '../../components/ui/Alert'
import Button from '../../components/ui/Button'
import { Input } from '../../components/ui/Field'
import AuthScreen from '../../components/layout/AuthScreen'
import NoIndex from '../../components/layout/NoIndex'
import { useAuth } from '../../context/auth'
import { HERO_BUILDING } from '../../lib/campus'
import { homeFor, paths } from '../../lib/paths'

export default function AdminLoginPage() {
  const { adminLogin, isAuthenticated, role } = useAuth()
  const navigate = useNavigate()
  const location = useLocation()
  const notice = location.state?.notice || ''
  const [identifier, setIdentifier] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)
  if (isAuthenticated) return <Navigate to={homeFor(role)} replace />
  const onSubmit = async (event) => {
    event.preventDefault()
    setError('')
    setLoading(true)
    try {
      const user = await adminLogin(identifier.trim(), password)
      navigate(homeFor(user.role), { replace: true })
    } catch (failure) {
      setError(failure.message)
    } finally {
      setLoading(false)
    }
  }
  return (
    <>
      <NoIndex />
      <AuthScreen
        images={HERO_BUILDING}
        tone="dark"
        note="Department office access for registrations, teachers, courses and the weekly routine."
      >
        <div>
          <p className="text-xs font-semibold tracking-[0.14em] text-gold-300 uppercase">
            Department office
          </p>
          <h1 className="font-display mt-2 border-b border-white/25 pb-3 text-[30px] font-bold text-white">
            Staff sign in
          </h1>
          <p className="mt-3 text-sm text-white/55">
            This entrance is for department staff accounts only.
          </p>
          <form onSubmit={onSubmit} className="mt-6 flex flex-col gap-4">
            {notice && <Alert tone="success">{notice}</Alert>}
            {error && <Alert tone="error">{error}</Alert>}
            <Input
              label="Admin ID"
              value={identifier}
              onChange={(event) => setIdentifier(event.target.value)}
              autoComplete="username"
              tone="dark"
              required
            />
            <Input
              label="Password"
              type="password"
              value={password}
              onChange={(event) => setPassword(event.target.value)}
              autoComplete="current-password"
              tone="dark"
              required
            />
            <Button type="submit" loading={loading} variant="light" className="w-full">
              Sign in
            </Button>
          </form>
          <div className="mt-5 text-sm">
            <Link to={paths.adminForgotPassword()} className="text-gold-300 hover:underline">
              Forgot your password?
            </Link>
          </div>
        </div>
      </AuthScreen>
    </>
  )
}

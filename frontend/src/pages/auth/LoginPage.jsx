import { useState } from 'react'
import { Link, Navigate, useNavigate } from 'react-router-dom'
import Alert from '../../components/ui/Alert'
import Button from '../../components/ui/Button'
import Card from '../../components/ui/Card'
import { Input } from '../../components/ui/Field'
import AuthScreen from '../../components/layout/AuthScreen'
import { useAuth } from '../../context/auth'
import { HERO_IMAGES } from '../../lib/campus'
import { homeFor, paths } from '../../lib/paths'

const SLIDE_AFTER = 6000

const OVERLAY = 'bg-linear-to-br from-brand-950/88 via-brand-800/72 to-brand-600/45'

export default function LoginPage() {
  const { login, isAuthenticated, role } = useAuth()
  const navigate = useNavigate()
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
      const user = await login(identifier.trim(), password)
      navigate(homeFor(user.role), { replace: true })
    } catch (failure) {
      setError(failure.message)
    } finally {
      setLoading(false)
    }
  }
  return (
    <AuthScreen images={HERO_IMAGES} rotateAfter={SLIDE_AFTER} overlay={OVERLAY}>
      <Card hover={false} className="animate-fade-in-up w-full backdrop-blur-sm">
        <div className="text-center">
          <p className="font-display text-2xl font-semibold text-brand-700">SUST CSE</p>
          <p className="mt-1 text-sm text-brand-500">
            Class and exam coordination for the Department of CSE
          </p>
        </div>
        <form onSubmit={onSubmit} className="mt-6 flex flex-col gap-4">
          {error && <Alert tone="error">{error}</Alert>}
          <Input
            label="Registration No. or Teacher ID"
            value={identifier}
            onChange={(event) => setIdentifier(event.target.value)}
            autoComplete="username"
            required
          />
          <Input
            label="Password"
            type="password"
            value={password}
            onChange={(event) => setPassword(event.target.value)}
            autoComplete="current-password"
            required
          />
          <Button type="submit" loading={loading} className="w-full">
            Log in
          </Button>
        </form>
        <div className="mt-5 flex flex-col gap-1 text-center text-sm">
          <Link to={paths.forgotPassword()} className="text-brand-600 hover:underline">
            Forgot your password?
          </Link>
          <p className="text-brand-500">
            New student?{' '}
            <Link to={paths.register()} className="text-brand-600 hover:underline">
              Create an account
            </Link>
          </p>
        </div>
      </Card>
    </AuthScreen>
  )
}

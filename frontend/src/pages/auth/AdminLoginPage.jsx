import { useState } from 'react'
import { Link, Navigate, useNavigate } from 'react-router-dom'
import { ShieldCheck } from 'lucide-react'
import Alert from '../../components/ui/Alert'
import Button from '../../components/ui/Button'
import Card from '../../components/ui/Card'
import { Input } from '../../components/ui/Field'
import AuthScreen from '../../components/layout/AuthScreen'
import { useAuth } from '../../context/auth'
import { HERO_BUILDING } from '../../lib/campus'
import { homeFor, paths } from '../../lib/paths'

const OVERLAY = 'bg-brand-950/92'

export default function AdminLoginPage() {
  const { adminLogin, isAuthenticated, role } = useAuth()
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
      const user = await adminLogin(identifier.trim(), password)
      navigate(homeFor(user.role), { replace: true })
    } catch (failure) {
      setError(failure.message)
    } finally {
      setLoading(false)
    }
  }
  return (
    <AuthScreen images={HERO_BUILDING} overlay={OVERLAY}>
      <Card hover={false} className="animate-fade-in-up w-full">
        <div className="text-center">
          <span className="inline-flex size-11 items-center justify-center rounded-full bg-brand-50 text-brand-600">
            <ShieldCheck size={22} />
          </span>
          <p className="font-display mt-3 text-xl font-semibold text-brand-900">Administration</p>
          <p className="mt-1 text-sm text-brand-500">Department staff access only</p>
        </div>
        <form onSubmit={onSubmit} className="mt-6 flex flex-col gap-4">
          {error && <Alert tone="error">{error}</Alert>}
          <Input
            label="Admin ID"
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
        <div className="mt-5 text-center text-sm">
          <Link to={paths.adminForgotPassword()} className="text-brand-600 hover:underline">
            Forgot password?
          </Link>
        </div>
      </Card>
    </AuthScreen>
  )
}

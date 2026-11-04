import { useState } from 'react'
import { Navigate, useNavigate } from 'react-router-dom'
import { KeyRound, ShieldCheck } from 'lucide-react'
import Alert from '../../components/ui/Alert'
import Button from '../../components/ui/Button'
import { Input, PasswordInput } from '../../components/ui/Field'
import AuthScreen from '../../components/layout/AuthScreen'
import AuthHeading from '../../components/layout/AuthHeading'
import NoIndex from '../../components/layout/NoIndex'
import HumanCheck from '../../components/forms/HumanCheck'
import { useAuth } from '../../context/auth'
import { useHumanCheck } from '../../lib/human'
import { HERO_BUILDING } from '../../lib/campus'
import { homeFor } from '../../lib/paths'

export default function ControlLoginPage() {
  const { controlLogin, isAuthenticated, role } = useAuth()
  const navigate = useNavigate()
  const human = useHumanCheck()
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
      await human.confirm()
      const user = await controlLogin(identifier.trim(), password)
      navigate(homeFor(user.role), { replace: true })
    } catch (failure) {
      setError(failure.message)
      human.reset()
    } finally {
      setLoading(false)
    }
  }
  return (
    <>
      <NoIndex />
      <AuthScreen images={HERO_BUILDING} tone="dark" note="Control room access only.">
        <div>
          <AuthHeading
            eyebrow="Control room"
            title="Sign in"
            subtitle="This entrance belongs to the portal owner."
          />
          <form onSubmit={onSubmit} className="mt-6 flex flex-col gap-4">
            {error && <Alert tone="error">{error}</Alert>}
            <Input
              label="Username"
              icon={ShieldCheck}
              value={identifier}
              onChange={(event) => setIdentifier(event.target.value)}
              autoComplete="username"
              required
            />
            <PasswordInput
              label="Password"
              icon={KeyRound}
              value={password}
              onChange={(event) => setPassword(event.target.value)}
              autoComplete="current-password"
              required
            />
            {human.needed && (
              <HumanCheck siteKey={human.siteKey} round={human.round} onToken={human.setToken} />
            )}
            <Button type="submit" loading={loading} disabled={human.blocked} className="w-full">
              Sign in
            </Button>
          </form>
        </div>
      </AuthScreen>
    </>
  )
}

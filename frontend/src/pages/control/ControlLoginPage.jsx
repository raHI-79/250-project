import { useState } from 'react'
import { Link, Navigate, useNavigate } from 'react-router-dom'
import { KeyRound, ShieldCheck } from 'lucide-react'
import Alert from '../../components/ui/Alert'
import Button from '../../components/ui/Button'
import { Input, PasswordInput } from '../../components/ui/Field'
import AuthScreen from '../../components/layout/AuthScreen'
import AuthHeading from '../../components/layout/AuthHeading'
import NoIndex from '../../components/layout/NoIndex'
import HumanCheck from '../../components/forms/HumanCheck'
import TwoStepForm from '../../components/forms/TwoStepForm'
import { useAuth } from '../../context/auth'
import { useHumanCheck } from '../../lib/human'
import { HERO_BUILDING } from '../../lib/campus'
import { homeFor, paths } from '../../lib/paths'

export default function ControlLoginPage() {
  const { controlLogin, twoStepLogin, isAuthenticated, role } = useAuth()
  const navigate = useNavigate()
  const human = useHumanCheck()
  const [identifier, setIdentifier] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [challenge, setChallenge] = useState(null)
  const [loading, setLoading] = useState(false)
  if (isAuthenticated) return <Navigate to={homeFor(role)} replace />
  const onSubmit = async (event) => {
    event.preventDefault()
    setError('')
    setLoading(true)
    try {
      await human.confirm()
      const user = await controlLogin(identifier.trim(), password)
      if (user.two_step) {
        setChallenge(user)
        return
      }
      navigate(homeFor(user.role), { replace: true })
    } catch (failure) {
      setError(failure.message)
      human.reset()
    } finally {
      setLoading(false)
    }
  }
  const finishTwoStep = async (code) => {
    const user = await twoStepLogin(identifier.trim(), code, challenge.pending)
    navigate(homeFor(user.role), { replace: true })
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
          {challenge ? (
            <TwoStepForm
              challenge={challenge}
              onSubmit={finishTwoStep}
              onCancel={() => setChallenge(null)}
            />
          ) : (
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
          )}
          <div className="mt-5 text-sm">
            <Link to={paths.controlForgotPassword()} className="text-brand-600 hover:underline">
              Forgot your password?
            </Link>
          </div>
        </div>
      </AuthScreen>
    </>
  )
}

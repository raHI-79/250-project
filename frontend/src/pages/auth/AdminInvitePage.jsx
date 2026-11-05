import { useEffect, useState } from 'react'
import { Link, useSearchParams } from 'react-router-dom'
import { KeyRound, UserRound } from 'lucide-react'
import Alert from '../../components/ui/Alert'
import Button from '../../components/ui/Button'
import { Input, PasswordInput } from '../../components/ui/Field'
import AuthScreen from '../../components/layout/AuthScreen'
import AuthHeading from '../../components/layout/AuthHeading'
import NoIndex from '../../components/layout/NoIndex'
import api from '../../lib/api'
import { HERO_BUILDING } from '../../lib/campus'
import { paths } from '../../lib/paths'

export default function AdminInvitePage() {
  const [params] = useSearchParams()
  const token = params.get('token') || ''
  const [invite, setInvite] = useState(null)
  const [checking, setChecking] = useState(true)
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)
  const [done, setDone] = useState(false)
  useEffect(() => {
    if (!token) {
      setError('This invitation link is invalid or has expired.')
      setChecking(false)
      return
    }
    api
      .get('/api/auth/admin-invite', { query: { token }, auth: false })
      .then((data) => setInvite(data))
      .catch((failure) => setError(failure.message))
      .finally(() => setChecking(false))
  }, [token])
  const onSubmit = async (event) => {
    event.preventDefault()
    setError('')
    setLoading(true)
    try {
      await api.post(
        '/api/auth/admin-invite/accept',
        {
          token,
          username: username.trim(),
          password,
          confirm_password: confirmPassword,
        },
        { auth: false },
      )
      setDone(true)
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
        className="max-w-md"
        note="Office admins choose their own username while accepting the invitation."
      >
        <div>
          <AuthHeading
            eyebrow="Office invitation"
            title={done ? 'Your account is ready' : 'Accept your invitation'}
            subtitle={invite?.masked_email ? `Invited as ${invite.masked_email}` : ''}
          />
          {error && (
            <Alert tone="error" className="mt-5">
              {error}
            </Alert>
          )}
          {done ? (
            <div className="mt-5 flex flex-col gap-4">
              <Alert tone="success" title="Welcome to the office pages">
                Sign in with the username and password you just chose.
              </Alert>
              <Link to={paths.adminLogin()} className="text-center text-sm text-brand-600">
                Go to the office sign in
              </Link>
            </div>
          ) : (
            !checking &&
            invite && (
              <form onSubmit={onSubmit} className="mt-6 flex flex-col gap-4">
                <Input
                  label="Username"
                  icon={UserRound}
                  value={username}
                  onChange={(event) => setUsername(event.target.value)}
                  hint="Letters, digits, dots, dashes and underscores. Not digits only."
                  autoComplete="username"
                  required
                />
                <PasswordInput
                  label="Password"
                  icon={KeyRound}
                  value={password}
                  onChange={(event) => setPassword(event.target.value)}
                  autoComplete="new-password"
                  required
                />
                <PasswordInput
                  label="Confirm password"
                  icon={KeyRound}
                  value={confirmPassword}
                  onChange={(event) => setConfirmPassword(event.target.value)}
                  autoComplete="new-password"
                  required
                />
                <Button type="submit" loading={loading} className="w-full">
                  Create my account
                </Button>
              </form>
            )
          )}
        </div>
      </AuthScreen>
    </>
  )
}

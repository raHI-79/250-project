import { useCallback, useEffect, useState } from 'react'
import { Laptop, Lock, Mail, ShieldCheck, Smartphone } from 'lucide-react'
import {
  Alert,
  Button,
  Card,
  CardHeader,
  EmptyState,
  Input,
  PasswordInput,
  PageHeader,
  SkeletonLines,
  StatusBadge,
  useConfirm,
  useToast,
} from '../../components/ui'
import api from '../../lib/api'
import { useAuth } from '../../context/auth'
import { formatDateTime } from '../../lib/time'

const EMPTY_PASSWORD = { current_password: '', new_password: '', confirm_password: '' }

export default function SecurityPage({ eyebrow = 'Department office' }) {
  const toast = useToast()
  const confirm = useConfirm()
  const { logout } = useAuth()
  const [state, setState] = useState(null)
  const [sessions, setSessions] = useState([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [password, setPassword] = useState(EMPTY_PASSWORD)
  const [contact, setContact] = useState({ email: '', phone: '', password: '' })
  const [twoStepPassword, setTwoStepPassword] = useState('')
  const [busy, setBusy] = useState('')
  const load = useCallback(async () => {
    setLoading(true)
    try {
      const [profile, list] = await Promise.all([
        api.get('/api/office/security'),
        api.get('/api/office/security/sessions'),
      ])
      setState(profile)
      setSessions(list.items || [])
      setError('')
    } catch (failure) {
      setError(failure.message)
    } finally {
      setLoading(false)
    }
  }, [])
  useEffect(() => {
    load()
  }, [load])
  const run = async (key, call, after) => {
    setBusy(key)
    try {
      const data = await call()
      toast.success(data.message)
      if (after) await after()
      else await load()
    } catch (failure) {
      toast.error(failure.message)
    } finally {
      setBusy('')
    }
  }
  const changePassword = (event) => {
    event.preventDefault()
    run('password', () => api.post('/api/office/security/password', password), async () => {
      setPassword(EMPTY_PASSWORD)
      await load()
    })
  }
  const changeContact = (event) => {
    event.preventDefault()
    run('contact', () => api.patch('/api/office/security/contact', contact), async () => {
      setContact({ email: '', phone: '', password: '' })
      await load()
    })
  }
  const toggleTwoStep = (event) => {
    event.preventDefault()
    run('two-step', () =>
      api.put('/api/office/security/two-step', {
        password: twoStepPassword,
        enabled: !state?.two_step,
      }),
    async () => {
      setTwoStepPassword('')
      await load()
    })
  }
  const closeSession = async (session) => {
    const ok = await confirm({
      title: 'Sign this session out?',
      message: `${session.device} · last seen ${formatDateTime(session.last_seen_at)}`,
      confirmLabel: 'Sign out',
    })
    if (!ok) return
    if (session.current) {
      await api.delete(`/api/office/security/sessions/${session.id}`).catch(() => null)
      await logout()
      return
    }
    run(`session-${session.id}`, () => api.delete(`/api/office/security/sessions/${session.id}`))
  }
  const closeOthers = async () => {
    const ok = await confirm({
      title: 'Sign every other session out?',
      message: 'Only this device stays signed in.',
      confirmLabel: 'Sign out everywhere else',
    })
    if (!ok) return
    run('others', () => api.delete('/api/office/security/sessions'))
  }
  return (
    <>
      <PageHeader
        eyebrow={eyebrow}
        title="Security"
        subtitle="Your password, your contact details, two-step sign-in and the devices you are signed in on"
      />
      {error && (
        <Alert tone="error" className="mb-5">
          {error}
        </Alert>
      )}
      <div className="grid gap-4 lg:grid-cols-2">
        <Card hover={false}>
          <CardHeader title="Change password" subtitle="At least 10 characters with a letter and a digit." />
          <form onSubmit={changePassword} className="mt-4 flex flex-col gap-3">
            <PasswordInput
              label="Current password"
              icon={Lock}
              value={password.current_password}
              onChange={(event) =>
                setPassword((current) => ({ ...current, current_password: event.target.value }))
              }
              autoComplete="current-password"
              required
            />
            <PasswordInput
              label="New password"
              icon={Lock}
              value={password.new_password}
              onChange={(event) =>
                setPassword((current) => ({ ...current, new_password: event.target.value }))
              }
              autoComplete="new-password"
              required
            />
            <PasswordInput
              label="New password again"
              icon={Lock}
              value={password.confirm_password}
              onChange={(event) =>
                setPassword((current) => ({ ...current, confirm_password: event.target.value }))
              }
              autoComplete="new-password"
              required
            />
            <Button type="submit" loading={busy === 'password'}>
              Change password
            </Button>
            <p className="text-xs text-muted">
              Every other session is signed out when the password changes.
            </p>
          </form>
        </Card>
        <Card hover={false}>
          <CardHeader
            title="Contact details"
            subtitle="The address and number the portal uses to reach you."
          />
          <form onSubmit={changeContact} className="mt-4 flex flex-col gap-3">
            <Input
              label="New email"
              type="email"
              icon={Mail}
              value={contact.email}
              onChange={(event) =>
                setContact((current) => ({ ...current, email: event.target.value }))
              }
            />
            <Input
              label="New phone"
              icon={Smartphone}
              value={contact.phone}
              onChange={(event) =>
                setContact((current) => ({ ...current, phone: event.target.value }))
              }
            />
            <PasswordInput
              label="Your password"
              icon={Lock}
              value={contact.password}
              onChange={(event) =>
                setContact((current) => ({ ...current, password: event.target.value }))
              }
              autoComplete="current-password"
              required
            />
            <Button type="submit" variant="secondary" loading={busy === 'contact'}>
              Save contact details
            </Button>
          </form>
        </Card>
        <Card hover={false}>
          <CardHeader
            title="Two-step sign-in"
            subtitle="A 6-digit code by email after your password, valid for 10 minutes."
          />
          <div className="mt-3 flex items-center gap-2">
            <StatusBadge
              status={state?.two_step ? 'active' : 'removed'}
              label={state?.two_step ? 'On' : 'Off'}
            />
            <span className="text-sm text-muted">
              {state?.two_step ? 'Codes are asked for at every sign-in.' : 'Only your password is asked for.'}
            </span>
          </div>
          <form onSubmit={toggleTwoStep} className="mt-4 flex flex-col gap-3">
            <PasswordInput
              label="Your password"
              icon={Lock}
              value={twoStepPassword}
              onChange={(event) => setTwoStepPassword(event.target.value)}
              autoComplete="current-password"
              required
            />
            <Button
              type="submit"
              variant={state?.two_step ? 'secondary' : 'primary'}
              loading={busy === 'two-step'}
            >
              {state?.two_step ? 'Turn two-step off' : 'Turn two-step on'}
            </Button>
          </form>
        </Card>
        <Card hover={false}>
          <CardHeader
            title="Where you are signed in"
            subtitle="Sign out anything you do not recognise."
            meta={
              sessions.length > 1 && (
                <Button size="sm" variant="secondary" onClick={closeOthers} loading={busy === 'others'}>
                  Sign out everywhere else
                </Button>
              )
            }
          />
          {loading ? (
            <SkeletonLines lines={3} />
          ) : sessions.length === 0 ? (
            <EmptyState icon={ShieldCheck} title="No other sessions" text="Only this device is signed in." />
          ) : (
            <ul className="mt-2 flex flex-col divide-y divide-line">
              {sessions.map((session) => (
                <li key={session.id} className="flex items-center justify-between gap-3 py-3">
                  <div className="flex min-w-0 items-center gap-3">
                    <span className="grid size-9 shrink-0 place-items-center rounded-full bg-wash text-brand-600">
                      {session.client === 'android' ? <Smartphone size={16} /> : <Laptop size={16} />}
                    </span>
                    <div className="min-w-0">
                      <p className="wrap-long text-sm font-semibold text-ink">
                        {session.device}
                        {session.current && <span className="ml-2 text-xs text-brand-600">This device</span>}
                      </p>
                      <p className="text-xs text-muted">
                        {session.client === 'android' ? 'App' : 'Website'} · {session.country} · last seen{' '}
                        {formatDateTime(session.last_seen_at)}
                      </p>
                    </div>
                  </div>
                  <Button
                    size="sm"
                    variant="secondary"
                    onClick={() => closeSession(session)}
                    loading={busy === `session-${session.id}`}
                  >
                    Sign out
                  </Button>
                </li>
              ))}
            </ul>
          )}
        </Card>
      </div>
    </>
  )
}

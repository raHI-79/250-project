import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import Alert from '../../components/ui/Alert'
import Button from '../../components/ui/Button'
import Card from '../../components/ui/Card'
import { Input } from '../../components/ui/Field'
import AuthScreen from '../../components/layout/AuthScreen'
import api from '../../lib/api'
import { HERO_IMAGES } from '../../lib/campus'
import cx from '../../lib/cx'
import { paths } from '../../lib/paths'

const OVERLAY = 'bg-linear-to-b from-white/92 via-white/88 to-brand-50/90'

const RESEND_SECONDS = 60

const STEPS = ['Your account', 'Email code', 'New password']

function Stepper({ step }) {
  return (
    <ol className="mt-5 flex items-center gap-2">
      {STEPS.map((label, position) => {
        const number = position + 1
        const reached = step >= number
        return (
          <li key={label} className="flex flex-1 flex-col items-center gap-1.5 text-center">
            <span
              className={cx(
                'inline-flex size-7 items-center justify-center rounded-full text-xs font-semibold transition-colors',
                reached ? 'bg-brand-600 text-white' : 'bg-brand-100 text-brand-500',
              )}
            >
              {number}
            </span>
            <span
              className={cx(
                'text-xs',
                reached ? 'font-medium text-brand-700' : 'text-brand-400',
              )}
            >
              {label}
            </span>
          </li>
        )
      })}
    </ol>
  )
}

export default function ForgotPasswordPage({ admin = false }) {
  const loginPath = admin ? paths.adminLogin() : paths.login()
  const [step, setStep] = useState(1)
  const [identifier, setIdentifier] = useState('')
  const [phone, setPhone] = useState('')
  const [code, setCode] = useState('')
  const [password, setPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [emailHint, setEmailHint] = useState('')
  const [devCode, setDevCode] = useState('')
  const [resetToken, setResetToken] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)
  const [secondsLeft, setSecondsLeft] = useState(0)
  useEffect(() => {
    if (secondsLeft <= 0) return undefined
    const timer = window.setTimeout(() => setSecondsLeft((current) => current - 1), 1000)
    return () => window.clearTimeout(timer)
  }, [secondsLeft])
  const requestCode = async () => {
    setError('')
    setLoading(true)
    try {
      const data = await api.post(
        '/api/auth/forgot-password',
        { identifier: identifier.trim(), phone: phone.trim() },
        { auth: false },
      )
      setEmailHint(data?.email_hint || '')
      setDevCode(data?.dev_code || '')
      setCode('')
      setSecondsLeft(RESEND_SECONDS)
      setStep(2)
    } catch (failure) {
      setError(failure.message)
    } finally {
      setLoading(false)
    }
  }
  const verifyCode = async () => {
    setError('')
    if (!/^[0-9]{6}$/.test(code.trim())) {
      setError('Please enter the 6-digit code from the email.')
      return
    }
    setLoading(true)
    try {
      const data = await api.post(
        '/api/auth/verify-reset-code',
        { identifier: identifier.trim(), code: code.trim() },
        { auth: false },
      )
      setResetToken(data?.reset_token || '')
      setStep(3)
    } catch (failure) {
      setError(failure.message)
    } finally {
      setLoading(false)
    }
  }
  const savePassword = async () => {
    setError('')
    if (password.length < 8) {
      setError('New password must be at least 8 characters long.')
      return
    }
    if (password !== confirmPassword) {
      setError('New password and the confirmation do not match.')
      return
    }
    setLoading(true)
    try {
      await api.post(
        '/api/auth/reset-password',
        {
          identifier: identifier.trim(),
          reset_token: resetToken,
          new_password: password,
          confirm_password: confirmPassword,
        },
        { auth: false },
      )
      setStep(4)
    } catch (failure) {
      setError(failure.message)
    } finally {
      setLoading(false)
    }
  }
  const onSubmit = (event) => {
    event.preventDefault()
    if (step === 1) requestCode()
    else if (step === 2) verifyCode()
    else if (step === 3) savePassword()
  }
  return (
    <AuthScreen images={HERO_IMAGES} rotateAfter={0} overlay={OVERLAY} className="max-w-md">
      <Card hover={false} className="animate-fade-in-up w-full">
        <div className="text-center">
          <p className="font-display text-xl font-semibold text-brand-800">
            {admin ? 'Administration password reset' : 'Reset your password'}
          </p>
          <p className="mt-1 text-sm text-brand-500">
            We check your ID against the phone number registered with your account.
          </p>
        </div>
        {step < 4 && <Stepper step={step} />}
        {step === 4 ? (
          <div className="mt-5 flex flex-col gap-4">
            <Alert tone="success" title="Password updated">
              You can now log in with your new password.
            </Alert>
            <Link to={loginPath} className="text-center text-sm text-brand-600 hover:underline">
              Back to log in
            </Link>
          </div>
        ) : (
          <form onSubmit={onSubmit} className="mt-5 flex flex-col gap-4">
            {error && <Alert tone="error">{error}</Alert>}
            {step === 1 && (
              <>
                <Input
                  label={admin ? 'Admin ID' : 'Registration No. or Teacher ID'}
                  value={identifier}
                  onChange={(event) => setIdentifier(event.target.value)}
                  autoComplete="username"
                  required
                />
                <Input
                  label="Registered phone number"
                  value={phone}
                  onChange={(event) => setPhone(event.target.value)}
                  autoComplete="tel"
                  hint="The number saved on your account, for example 01712345678"
                  required
                />
                <Button type="submit" loading={loading} className="w-full">
                  Send the code
                </Button>
              </>
            )}
            {step === 2 && (
              <>
                <Alert tone="info" title="Check your email">
                  {emailHint
                    ? `We sent a 6-digit code to ${emailHint}. It expires in 10 minutes.`
                    : 'We sent a 6-digit code to the email address registered with your account. It expires in 10 minutes.'}
                </Alert>
                {devCode && (
                  <Alert tone="info" title="Development code">
                    No mail server is configured, so the code is shown here: {devCode}
                  </Alert>
                )}
                <Input
                  label="6-digit code"
                  value={code}
                  onChange={(event) => setCode(event.target.value)}
                  inputMode="numeric"
                  maxLength={6}
                  autoComplete="one-time-code"
                  required
                />
                <Button type="submit" loading={loading} className="w-full">
                  Verify the code
                </Button>
                <Button
                  variant="secondary"
                  onClick={requestCode}
                  disabled={loading || secondsLeft > 0}
                  className="w-full"
                >
                  {secondsLeft > 0 ? `Resend in ${secondsLeft}s` : 'Resend the code'}
                </Button>
              </>
            )}
            {step === 3 && (
              <>
                <Input
                  label="New password"
                  type="password"
                  value={password}
                  onChange={(event) => setPassword(event.target.value)}
                  autoComplete="new-password"
                  hint="At least 8 characters"
                  required
                />
                <Input
                  label="Confirm new password"
                  type="password"
                  value={confirmPassword}
                  onChange={(event) => setConfirmPassword(event.target.value)}
                  autoComplete="new-password"
                  required
                />
                <Button type="submit" loading={loading} className="w-full">
                  Save the new password
                </Button>
              </>
            )}
            <Link to={loginPath} className="text-center text-sm text-brand-600 hover:underline">
              Back to log in
            </Link>
          </form>
        )}
      </Card>
    </AuthScreen>
  )
}

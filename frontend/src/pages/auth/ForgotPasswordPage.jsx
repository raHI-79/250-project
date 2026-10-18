import { useState } from 'react'
import { Link } from 'react-router-dom'
import { Alert, Button, Card, Input, Stepper } from '../../components/ui'
import PhoneInput from '../../components/forms/PhoneInput'
import AuthScreen from '../../components/layout/AuthScreen'
import api from '../../lib/api'
import { HERO_IMAGES } from '../../lib/campus'
import { useCountdown } from '../../lib/hooks'
import { PORTAL_ADMIN, loginForPortal } from '../../lib/paths'
import {
  CODE_PATTERN,
  PHONE_MESSAGE,
  birthDateBounds,
  isPhone,
  passwordError,
} from '../../lib/account'

const OVERLAY = 'bg-linear-to-b from-white/92 via-white/88 to-brand-50/90'

const RESEND_SECONDS = 60

const STEPS = ['Your account', 'Email code', 'New password']

const copy = {
  student: {
    heading: 'Reset your password',
    subtitle: 'We check your registration number against the date of birth on your account.',
    identifier: 'Registration number',
  },
  teacher: {
    heading: 'Reset your password',
    subtitle: 'We check your teacher ID against the date of birth on your account.',
    identifier: 'Teacher ID',
  },
  admin: {
    heading: 'Administration password reset',
    subtitle: 'We check your ID against the phone number registered with your account.',
    identifier: 'Admin ID',
  },
}

export default function ForgotPasswordPage({ portal }) {
  const page = copy[portal] || copy.student
  const byPhone = portal === PORTAL_ADMIN
  const loginPath = loginForPortal(portal)
  const bounds = birthDateBounds()
  const [step, setStep] = useState(1)
  const [identifier, setIdentifier] = useState('')
  const [phone, setPhone] = useState('')
  const [dateOfBirth, setDateOfBirth] = useState('')
  const [code, setCode] = useState('')
  const [password, setPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [emailHint, setEmailHint] = useState('')
  const [devCode, setDevCode] = useState('')
  const [resetToken, setResetToken] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)
  const [seconds, setSeconds] = useCountdown(0)
  const requestCode = async () => {
    setError('')
    if (byPhone && !isPhone(phone)) {
      setError(PHONE_MESSAGE)
      return
    }
    if (!byPhone && !dateOfBirth) {
      setError('Date of birth is required.')
      return
    }
    setLoading(true)
    try {
      const body = byPhone
        ? { identifier: identifier.trim(), phone: phone.trim() }
        : { identifier: identifier.trim(), date_of_birth: dateOfBirth }
      const data = await api.post('/api/auth/forgot-password', body, { auth: false })
      setEmailHint(data?.email_hint || '')
      setDevCode(data?.dev_code || '')
      setCode('')
      setSeconds(RESEND_SECONDS)
      setStep(2)
    } catch (failure) {
      setError(failure.message)
    } finally {
      setLoading(false)
    }
  }
  const verifyCode = async () => {
    setError('')
    if (!CODE_PATTERN.test(code.trim())) {
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
    const problem = passwordError(password, confirmPassword)
    if (problem) {
      setError(problem.replace('Password', 'New password'))
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
          <p className="font-display text-xl font-semibold text-brand-800">{page.heading}</p>
          <p className="mt-1 text-sm text-brand-500">{page.subtitle}</p>
        </div>
        {step < 4 && <Stepper steps={STEPS} current={step} className="mt-5" />}
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
                  label={page.identifier}
                  value={identifier}
                  onChange={(event) => setIdentifier(event.target.value)}
                  autoComplete="username"
                  required
                />
                {byPhone ? (
                  <PhoneInput
                    label="Registered phone number"
                    value={phone}
                    onChange={setPhone}
                    hint="The number saved on your account"
                    required
                  />
                ) : (
                  <Input
                    label="Date of birth"
                    type="date"
                    value={dateOfBirth}
                    onChange={(event) => setDateOfBirth(event.target.value)}
                    min={bounds.min}
                    max={bounds.max}
                    hint="The date saved on your account"
                    required
                  />
                )}
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
                  disabled={loading || seconds > 0}
                  className="w-full"
                >
                  {seconds > 0 ? `Resend in ${seconds}s` : 'Resend the code'}
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

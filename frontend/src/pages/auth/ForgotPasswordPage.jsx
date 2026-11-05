import { useState } from 'react'
import { Link } from 'react-router-dom'
import { IdCard, KeyRound, Lock } from 'lucide-react'
import { Alert, Button, DateField, Input, PasswordInput } from '../../components/ui'
import FormSection from '../../components/forms/FormSection'
import PhoneInput from '../../components/forms/PhoneInput'
import AuthScreen from '../../components/layout/AuthScreen'
import AuthHeading from '../../components/layout/AuthHeading'
import HumanCheck from '../../components/forms/HumanCheck'
import api from '../../lib/api'
import { useHumanCheck } from '../../lib/human'
import { HERO_BUILDING } from '../../lib/campus'
import { useCountdown } from '../../lib/hooks'
import { PORTAL_ADMIN, PORTAL_CONTROL, loginForPortal } from '../../lib/paths'
import { resetErrors } from '../../lib/account'

const RESEND_SECONDS = 60

const copy = {
  student: {
    heading: 'Reset your password',
    subtitle: 'We check your registration number against the date of birth on your account.',
    identifier: 'Registration number',
  },
  teacher: {
    heading: 'Reset your password',
    subtitle: 'We check your User ID or email against the date of birth on your account.',
    identifier: 'User ID or email',
  },
  admin: {
    heading: 'Administration password reset',
    subtitle: 'We check your ID against the phone number registered with your account.',
    identifier: 'Admin ID',
  },
  control: {
    heading: 'Control room password reset',
    subtitle: 'We check your username against the phone number registered with your account.',
    identifier: 'Username',
  },
}

const EMPTY = {
  identifier: '',
  phone: '',
  dateOfBirth: '',
  code: '',
  password: '',
  confirmPassword: '',
}

export default function ForgotPasswordPage({ portal }) {
  const page = copy[portal] || copy.student
  const byPhone = portal === PORTAL_ADMIN || portal === PORTAL_CONTROL
  const loginPath = loginForPortal(portal)
  const human = useHumanCheck()
  const [form, setForm] = useState(EMPTY)
  const [errors, setErrors] = useState({})
  const [sent, setSent] = useState(false)
  const [emailHint, setEmailHint] = useState('')
  const [devCode, setDevCode] = useState('')
  const [resetToken, setResetToken] = useState('')
  const [done, setDone] = useState(false)
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)
  const [seconds, setSeconds] = useCountdown(0)
  const set = (key) => (event) => {
    const { value } = event.target
    setForm((current) => ({ ...current, [key]: value }))
    setErrors((current) => ({ ...current, [key]: '' }))
  }
  const sendCode = async () => {
    const found = resetErrors(form, { byPhone, stage: 'identify' })
    setErrors(found)
    if (Object.keys(found).length > 0) {
      setError('Please correct the fields marked below.')
      return
    }
    setError('')
    setLoading(true)
    try {
      await human.confirm()
      const body = byPhone
        ? { identifier: form.identifier.trim(), phone: form.phone.trim() }
        : { identifier: form.identifier.trim(), date_of_birth: form.dateOfBirth }
      const data = await api.post('/api/auth/forgot-password', body, { auth: false })
      setEmailHint(data?.email_hint || '')
      setDevCode(data?.dev_code || '')
      setForm((current) => ({ ...current, code: '' }))
      setSeconds(RESEND_SECONDS)
      setSent(true)
    } catch (failure) {
      setError(failure.message)
      human.reset()
    } finally {
      setLoading(false)
    }
  }
  const save = async () => {
    const found = resetErrors(form, { byPhone })
    setErrors(found)
    if (Object.keys(found).length > 0) {
      setError('Please correct the fields marked below.')
      return
    }
    setError('')
    setLoading(true)
    try {
      await human.confirm()
      let token = resetToken
      if (!token) {
        const data = await api.post(
          '/api/auth/verify-reset-code',
          { identifier: form.identifier.trim(), code: form.code.trim() },
          { auth: false },
        )
        token = data?.reset_token || ''
        setResetToken(token)
      }
      await api.post(
        '/api/auth/reset-password',
        {
          identifier: form.identifier.trim(),
          reset_token: token,
          new_password: form.password,
          confirm_password: form.confirmPassword,
        },
        { auth: false },
      )
      setDone(true)
    } catch (failure) {
      setResetToken('')
      setError(failure.message)
      human.reset()
    } finally {
      setLoading(false)
    }
  }
  const onSubmit = (event) => {
    event.preventDefault()
    if (sent) save()
    else sendCode()
  }
  return (
    <AuthScreen
      images={HERO_BUILDING}
      className="max-w-md"
      appTitle={page.heading}
      appSubtitle={page.subtitle}
      note="Lost your password? The account is checked before a code is sent."
    >
      <div>
        <AuthHeading eyebrow="Password reset" title={page.heading} subtitle={page.subtitle} />
        {done ? (
          <div className="mt-5 flex flex-col gap-4">
            <Alert tone="success" title="Password updated">
              You can now log in with your new password.
            </Alert>
            <Link to={loginPath} className="text-sm text-brand-600 hover:underline">
              Back to log in
            </Link>
          </div>
        ) : (
          <form onSubmit={onSubmit} className="mt-5 flex flex-col gap-6">
            {error && <Alert tone="error">{error}</Alert>}
            <FormSection title="Your account">
              <Input
                label={page.identifier}
                icon={IdCard}
                value={form.identifier}
                onChange={set('identifier')}
                error={errors.identifier}
                autoComplete="username"
                required
              />
              {byPhone ? (
                <PhoneInput
                  label="Registered phone number"
                  value={form.phone}
                  onChange={(next) => {
                    setForm((current) => ({ ...current, phone: next }))
                    setErrors((current) => ({ ...current, phone: '' }))
                  }}
                  error={errors.phone}
                  hint="The number saved on your account"
                  required
                />
              ) : (
                <DateField
                  label="Date of birth"
                  value={form.dateOfBirth}
                  onChange={(next) => set('dateOfBirth')({ target: { value: next } })}
                  error={errors.dateOfBirth}
                  hint="The date saved on your account"
                  required
                />
              )}
              {!sent && (
                <Button
                  variant="secondary"
                  onClick={sendCode}
                  loading={loading}
                  disabled={human.blocked}
                  className="w-full"
                >
                  Send code
                </Button>
              )}
            </FormSection>
            {sent && (
              <>
                <FormSection title="Email code">
                  <Input
                    label="6-digit code"
                    icon={KeyRound}
                    value={form.code}
                    onChange={set('code')}
                    error={errors.code}
                    hint={
                      emailHint
                        ? `Sent to ${emailHint}. It expires in 10 minutes.`
                        : 'Sent to the email address registered with your account. It expires in 10 minutes.'
                    }
                    inputMode="numeric"
                    maxLength={6}
                    autoComplete="one-time-code"
                    required
                  />
                  {devCode && (
                    <Alert tone="info" title="Development code">
                      No mail server is configured, so the code is shown here: {devCode}
                    </Alert>
                  )}
                  <Button
                    variant="secondary"
                    onClick={sendCode}
                    disabled={loading || seconds > 0 || human.blocked}
                    className="w-full"
                  >
                    {seconds > 0 ? `Resend in ${seconds}s` : 'Resend the code'}
                  </Button>
                </FormSection>
                <FormSection title="New password">
                  <PasswordInput
                    label="New password"
                    icon={Lock}
                    value={form.password}
                    onChange={set('password')}
                    error={errors.password}
                    autoComplete="new-password"
                    hint="At least 8 characters"
                    required
                  />
                  <PasswordInput
                    label="Confirm new password"
                    icon={Lock}
                    value={form.confirmPassword}
                    onChange={set('confirmPassword')}
                    autoComplete="new-password"
                    required
                  />
                </FormSection>
              </>
            )}
            {human.needed && (
              <HumanCheck siteKey={human.siteKey} round={human.round} onToken={human.setToken} />
            )}
            {sent && (
              <Button
                type="submit"
                loading={loading}
                disabled={human.blocked}
                size="lg"
                className="w-full"
              >
                Save the new password
              </Button>
            )}
            <Link to={loginPath} className="text-sm text-brand-600 hover:underline">
              Back to log in
            </Link>
          </form>
        )}
      </div>
    </AuthScreen>
  )
}

import { useState } from 'react'
import { Link, useSearchParams } from 'react-router-dom'
import { BadgeCheck, IdCard, KeyRound, Lock, Mail } from 'lucide-react'
import { Alert, Button, DateField, Field, Input, PasswordInput, Select } from '../../components/ui'
import FormSection from '../../components/forms/FormSection'
import GoogleVerifyButton from '../../components/forms/GoogleVerifyButton'
import PhoneInput from '../../components/forms/PhoneInput'
import PhotoPicker from '../../components/forms/PhotoPicker'
import SectionPicker from '../../components/forms/SectionPicker'
import AuthScreen from '../../components/layout/AuthScreen'
import AuthHeading from '../../components/layout/AuthHeading'
import HumanCheck from '../../components/forms/HumanCheck'
import api from '../../lib/api'
import { useHumanCheck } from '../../lib/human'
import { HERO_BUILDING } from '../../lib/campus'
import { useAuthConfig, useCountdown } from '../../lib/hooks'
import { paths } from '../../lib/paths'
import { BLOOD_GROUPS } from '../../lib/options'
import { appendIf, isCode, registrationErrors } from '../../lib/account'

const RESEND_SECONDS = 60

const EMPTY = {
  fullName: '',
  regNo: '',
  dateOfBirth: '',
  batch: '',
  section: '',
  email: '',
  phone: '',
  bloodGroup: '',
  password: '',
  confirmPassword: '',
}

export default function RegisterPage() {
  const { studentEmailDomain, googleEnabled } = useAuthConfig()
  const human = useHumanCheck()
  const [params] = useSearchParams()
  const resuming = params.get('verify') || ''
  const [form, setForm] = useState(resuming ? { ...EMPTY, regNo: resuming } : EMPTY)
  const [photo, setPhoto] = useState(null)
  const [errors, setErrors] = useState({})
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)
  const [emailProof, setEmailProof] = useState('')
  const [code, setCode] = useState('')
  const [sent, setSent] = useState(Boolean(resuming))
  const [emailHint, setEmailHint] = useState('')
  const [devCode, setDevCode] = useState('')
  const [done, setDone] = useState(false)
  const [seconds, setSeconds] = useCountdown(0)
  const set = (key) => (event) => {
    const { value } = event.target
    setForm((current) => ({ ...current, [key]: value }))
    setErrors((current) => ({ ...current, [key]: '' }))
  }
  const clearGoogleEmail = () => {
    setEmailProof('')
    setForm((current) => ({ ...current, email: '' }))
    setErrors((current) => ({ ...current, email: '' }))
  }
  const onGoogleCredential = async (credential) => {
    setError('')
    setLoading(true)
    try {
      await human.confirm()
      const data = await api.post('/api/auth/google/verify-email', { credential }, { auth: false })
      setEmailProof(data?.email_proof || '')
      setForm((current) => ({ ...current, email: data?.email || current.email }))
      setErrors((current) => ({ ...current, email: '' }))
    } catch (failure) {
      setError(failure.message)
      human.reset()
    } finally {
      setLoading(false)
    }
  }
  const register = async () => {
    const found = registrationErrors(form, studentEmailDomain)
    setErrors(found)
    if (Object.keys(found).length > 0) {
      setError('Please correct the fields marked below.')
      return false
    }
    const body = new FormData()
    body.append('full_name', form.fullName.trim())
    body.append('reg_no', form.regNo.trim())
    body.append('batch', form.batch.trim().toUpperCase())
    body.append('section', form.section.trim().toUpperCase())
    body.append('email', form.email.trim().toLowerCase())
    body.append('phone', form.phone.trim())
    body.append('blood_group', form.bloodGroup)
    body.append('date_of_birth', form.dateOfBirth)
    body.append('password', form.password)
    body.append('confirm_password', form.confirmPassword)
    appendIf(body, 'photo', photo)
    appendIf(body, 'email_proof', emailProof)
    setError('')
    setLoading(true)
    try {
      await human.confirm()
      const data = await api.post('/api/auth/register', body, { auth: false })
      setEmailHint(data?.email_hint || '')
      setDevCode(data?.dev_code || '')
      if (emailProof) {
        setDone(true)
        return true
      }
      setSent(true)
      setSeconds(RESEND_SECONDS)
      return true
    } catch (failure) {
      setError(failure.message)
      human.reset()
      return false
    } finally {
      setLoading(false)
    }
  }
  const confirm = async () => {
    if (!isCode(code)) {
      setErrors((current) => ({ ...current, code: 'Enter the 6-digit code from the email.' }))
      return
    }
    setError('')
    setLoading(true)
    try {
      await human.confirm()
      await api.post(
        '/api/auth/verify-email',
        { reg_no: form.regNo.trim(), code: code.trim() },
        { auth: false },
      )
      setDone(true)
    } catch (failure) {
      setError(failure.message)
      human.reset()
    } finally {
      setLoading(false)
    }
  }
  const resend = async () => {
    setError('')
    setLoading(true)
    try {
      await human.confirm()
      const data = await api.post(
        '/api/auth/resend-verification',
        { reg_no: form.regNo.trim() },
        { auth: false },
      )
      setEmailHint(data?.email_hint || emailHint)
      setDevCode(data?.dev_code || '')
      setCode('')
      setSeconds(RESEND_SECONDS)
    } catch (failure) {
      setError(failure.message)
      human.reset()
    } finally {
      setLoading(false)
    }
  }
  const onSubmit = (event) => {
    event.preventDefault()
    if (sent) confirm()
    else register()
  }
  const codeField = (
    <>
      <Input
        label="6-digit code"
        icon={KeyRound}
        value={code}
        onChange={(event) => {
          setCode(event.target.value)
          setErrors((current) => ({ ...current, code: '' }))
        }}
        error={errors.code}
        hint={
          emailHint
            ? `Sent to ${emailHint}. It expires in 15 minutes.`
            : 'Sent to your university email address. It expires in 15 minutes.'
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
        onClick={resend}
        disabled={loading || seconds > 0 || human.blocked}
        className="w-full"
      >
        {seconds > 0 ? `Resend in ${seconds}s` : 'Resend the code'}
      </Button>
    </>
  )
  return (
    <AuthScreen
      images={HERO_BUILDING}
      wide
      appTitle={resuming ? 'Confirm your email' : 'Create your account'}
      appSubtitle={
        resuming
          ? 'Enter the code we sent to your university email'
          : 'Fill the form once and verify your university email'
      }
      note="Students register with their registration number and verify their university email."
    >
      <div>
        <AuthHeading
          eyebrow="Student registration"
          title="Create your account"
          subtitle="Fill the form once and verify your university email."
        />
        <div className="mt-5">
          {done ? (
            <div className="flex flex-col gap-4">
              <Alert tone="success" title="Registration submitted">
                Your email address is confirmed. The administration reviews new accounts, so you
                can log in once your registration is approved.
              </Alert>
              <Link
                to={paths.studentLogin()}
                className="text-center text-sm text-brand-600 hover:underline"
              >
                Go to the student login
              </Link>
            </div>
          ) : (
            <form onSubmit={onSubmit} className="flex flex-col gap-6">
              {error && <Alert tone="error">{error}</Alert>}
              {resuming ? (
                <FormSection
                  title="Confirm your email"
                  description="Enter the code we sent to your university email address."
                >
                  <Input
                    label="Registration number"
                    icon={IdCard}
                    value={form.regNo}
                    onChange={set('regNo')}
                    autoComplete="username"
                    required
                  />
                  {codeField}
                </FormSection>
              ) : (
                <>
                  <FormSection
                    title="Account"
                    description="The registration number and password you will log in with."
                  >
                    <Input
                      label="Full name"
                      value={form.fullName}
                      onChange={set('fullName')}
                      error={errors.fullName}
                      autoComplete="name"
                      required
                    />
                    <div className="grid gap-4 sm:grid-cols-2">
                      <Input
                        label="Registration number"
                        icon={IdCard}
                        value={form.regNo}
                        onChange={set('regNo')}
                        error={errors.regNo}
                        autoComplete="username"
                        required
                      />
                      <Input
                        label="Batch"
                        value={form.batch}
                        onChange={(event) => {
                          const batch = event.target.value.toUpperCase()
                          setForm((current) => ({ ...current, batch }))
                          setErrors((current) => ({ ...current, batch: '' }))
                        }}
                        error={errors.batch}
                        hint="Like CSE-23"
                        placeholder="CSE-23"
                        required
                      />
                      <SectionPicker
                        value={form.section}
                        error={errors.section}
                        onChange={(section) => {
                          setForm((current) => ({ ...current, section }))
                          setErrors((current) => ({ ...current, section: '' }))
                        }}
                      />
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
                        label="Confirm password"
                        icon={Lock}
                        value={form.confirmPassword}
                        onChange={set('confirmPassword')}
                        autoComplete="new-password"
                        required
                      />
                    </div>
                  </FormSection>
                  <FormSection
                    title="Contact"
                    description="Verify your university address with Google, or let us email you a code."
                  >
                    {emailProof ? (
                      <Field label="University email">
                        <div className="flex flex-wrap items-center justify-between gap-2 rounded-xl border border-line-strong bg-wash px-3 py-2">
                          <span className="wrap-long min-w-0 text-sm text-ink">{form.email}</span>
                          <span className="flex flex-wrap items-center gap-2">
                            <span className="inline-flex items-center gap-1 rounded-full bg-brand-600 px-2.5 py-0.5 text-xs font-medium text-page">
                              <BadgeCheck size={13} />
                              Verified with Google
                            </span>
                            <button
                              type="button"
                              onClick={clearGoogleEmail}
                              className="text-xs text-brand-600 hover:underline"
                            >
                              Change
                            </button>
                          </span>
                        </div>
                      </Field>
                    ) : (
                      <>
                        <Input
                          label="University email"
                          type="email"
                          icon={Mail}
                          value={form.email}
                          onChange={set('email')}
                          error={errors.email}
                          hint={
                            studentEmailDomain
                              ? `Must end with @${studentEmailDomain}`
                              : 'Use your university address'
                          }
                          autoComplete="email"
                          required
                        />
                        {sent ? (
                          codeField
                        ) : (
                          <>
                            <Button
                              variant="secondary"
                              onClick={register}
                              loading={loading}
                              disabled={human.blocked}
                              className="w-full"
                            >
                              Send code
                            </Button>
                            {googleEnabled && (
                              <>
                                <div className="flex items-center gap-3">
                                  <span className="h-px flex-1 bg-brand-100" />
                                  <span className="text-xs text-faint">or</span>
                                  <span className="h-px flex-1 bg-brand-100" />
                                </div>
                                <GoogleVerifyButton
                                  hostedDomain={studentEmailDomain}
                                  onCredential={onGoogleCredential}
                                  onError={setError}
                                />
                                <p className="text-center text-xs text-faint">
                                  Google confirms your address at once, so you can skip the email
                                  code.
                                </p>
                              </>
                            )}
                          </>
                        )}
                      </>
                    )}
                    <PhoneInput
                      value={form.phone}
                      onChange={(next) => {
                        setForm((current) => ({ ...current, phone: next }))
                        setErrors((current) => ({ ...current, phone: '' }))
                      }}
                      error={errors.phone}
                      required
                    />
                  </FormSection>
                  <FormSection title="Personal">
                    <div className="grid gap-4 sm:grid-cols-2">
                      <DateField
                        label="Date of birth"
                        value={form.dateOfBirth}
                        onChange={(next) => set('dateOfBirth')({ target: { value: next } })}
                        error={errors.dateOfBirth}
                        hint="Between 15 and 60 years ago"
                        required
                      />
                      <Select
                        label="Blood group"
                        value={form.bloodGroup}
                        onChange={set('bloodGroup')}
                        error={errors.bloodGroup}
                        required
                      >
                        <option value="">Choose a blood group</option>
                        {BLOOD_GROUPS.map((group) => (
                          <option key={group} value={group}>
                            {group}
                          </option>
                        ))}
                      </Select>
                    </div>
                  </FormSection>
                  <FormSection title="Photo" description="Optional.">
                    <PhotoPicker
                      file={photo}
                      onChange={(next) => {
                        setPhoto(next)
                        setErrors((current) => ({ ...current, photo: '' }))
                      }}
                      error={errors.photo}
                      hint="Optional. Your initials are shown until you add one."
                    />
                  </FormSection>
                </>
              )}
              {human.needed && (
                <HumanCheck
                  siteKey={human.siteKey}
                  round={human.round}
                  onToken={human.setToken}
                />
              )}
              <div className="flex flex-col gap-3">
                <Button
                  type="submit"
                  loading={loading}
                  disabled={human.blocked}
                  size="lg"
                  className="w-full"
                >
                  {sent ? 'Confirm email and submit' : 'Submit registration'}
                </Button>
                <p className="text-center text-sm text-muted">
                  Already registered?{' '}
                  <Link to={paths.studentLogin()} className="text-brand-600 hover:underline">
                    Log in
                  </Link>
                </p>
              </div>
            </form>
          )}
        </div>
      </div>
    </AuthScreen>
  )
}

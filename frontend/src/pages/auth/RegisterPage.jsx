import { useState } from 'react'
import { Link, useSearchParams } from 'react-router-dom'
import { BadgeCheck } from 'lucide-react'
import { Alert, Button, Field, Input, Select, Stepper } from '../../components/ui'
import GoogleVerifyButton from '../../components/forms/GoogleVerifyButton'
import PhoneInput from '../../components/forms/PhoneInput'
import PhotoPicker from '../../components/forms/PhotoPicker'
import SectionPicker from '../../components/forms/SectionPicker'
import AuthScreen from '../../components/layout/AuthScreen'
import api from '../../lib/api'
import { HERO_IMAGES } from '../../lib/campus'
import { useAuthConfig, useCountdown } from '../../lib/hooks'
import { paths } from '../../lib/paths'
import { BLOOD_GROUPS } from '../../lib/options'
import {
  BATCH_PATTERN,
  CODE_PATTERN,
  MAX_FIELD,
  PHONE_MESSAGE,
  SECTION_PATTERN,
  appendIf,
  birthDateBounds,
  birthDateError,
  emailDomainError,
  isPhone,
  passwordError,
} from '../../lib/account'

const STEPS = ['Personal', 'Academic', 'Account', 'Verify']

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
  const [params] = useSearchParams()
  const resuming = params.get('verify') || ''
  const [step, setStep] = useState(resuming ? 4 : 1)
  const [form, setForm] = useState(resuming ? { ...EMPTY, regNo: resuming } : EMPTY)
  const [photo, setPhoto] = useState(null)
  const [errors, setErrors] = useState({})
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)
  const [emailProof, setEmailProof] = useState('')
  const [code, setCode] = useState('')
  const [emailHint, setEmailHint] = useState('')
  const [devCode, setDevCode] = useState('')
  const [seconds, setSeconds] = useCountdown(0)
  const bounds = birthDateBounds()
  const set = (key) => (event) => {
    const { value } = event.target
    setForm((current) => ({ ...current, [key]: value }))
  }
  const checkPersonal = () => {
    const found = {}
    const name = form.fullName.trim()
    if (!name) found.fullName = 'Full name is required.'
    else if (name.length > MAX_FIELD)
      found.fullName = `Full name must be at most ${MAX_FIELD} characters long.`
    if (!form.regNo.trim()) found.regNo = 'Registration number is required.'
    const birth = birthDateError(form.dateOfBirth)
    if (birth) found.dateOfBirth = birth
    if (!photo) found.photo = 'A profile photo is required.'
    return found
  }
  const checkAcademic = () => {
    const found = {}
    if (!BATCH_PATTERN.test(form.batch.trim().toUpperCase()))
      found.batch = 'Batch must look like CSE-23.'
    if (!SECTION_PATTERN.test(form.section.trim().toUpperCase()))
      found.section = 'Section must be a single letter from A to Z.'
    return found
  }
  const checkAccount = () => {
    const found = {}
    const email = emailDomainError(form.email, studentEmailDomain)
    if (email) found.email = email
    if (!isPhone(form.phone)) found.phone = PHONE_MESSAGE
    if (!BLOOD_GROUPS.includes(form.bloodGroup)) found.bloodGroup = 'Please choose your blood group.'
    const password = passwordError(form.password, form.confirmPassword)
    if (password) found.password = password
    return found
  }
  const advance = (check) => {
    setError('')
    const found = check()
    setErrors(found)
    if (Object.keys(found).length > 0) return
    setStep((current) => current + 1)
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
      const data = await api.post(
        '/api/auth/google/verify-email',
        { credential },
        { auth: false },
      )
      setEmailProof(data?.email_proof || '')
      setForm((current) => ({ ...current, email: data?.email || current.email }))
      setErrors((current) => ({ ...current, email: '' }))
    } catch (failure) {
      setError(failure.message)
    } finally {
      setLoading(false)
    }
  }
  const submit = async () => {
    setError('')
    const found = checkAccount()
    setErrors(found)
    if (Object.keys(found).length > 0) return
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
    body.append('photo', photo)
    appendIf(body, 'email_proof', emailProof)
    setLoading(true)
    try {
      const data = await api.post('/api/auth/register', body, { auth: false })
      setEmailHint(data?.email_hint || '')
      setDevCode(data?.dev_code || '')
      if (emailProof) {
        setStep(5)
        return
      }
      setSeconds(RESEND_SECONDS)
      setStep(4)
    } catch (failure) {
      setError(failure.message)
    } finally {
      setLoading(false)
    }
  }
  const verify = async () => {
    setError('')
    if (!CODE_PATTERN.test(code.trim())) {
      setError('Please enter the 6-digit code from the email.')
      return
    }
    setLoading(true)
    try {
      await api.post(
        '/api/auth/verify-email',
        { reg_no: form.regNo.trim(), code: code.trim() },
        { auth: false },
      )
      setStep(5)
    } catch (failure) {
      setError(failure.message)
    } finally {
      setLoading(false)
    }
  }
  const resend = async () => {
    setError('')
    setLoading(true)
    try {
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
    } finally {
      setLoading(false)
    }
  }
  const onSubmit = (event) => {
    event.preventDefault()
    if (step === 1) advance(checkPersonal)
    else if (step === 2) advance(checkAcademic)
    else if (step === 3) submit()
    else if (step === 4) verify()
  }
  return (
    <AuthScreen
      images={HERO_IMAGES}
      wide
      note="Students register with their registration number and verify their university email."
    >
      <div>
        <p className="eyebrow">Student registration</p>
        <h1 className="font-display mt-2 text-title font-bold text-ink">
          Create your account
        </h1>
        <div className="mt-5">
          {step === 5 ? (
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
            <>
              <Stepper steps={STEPS} current={step} className="mb-6" />
              <form onSubmit={onSubmit} className="flex flex-col gap-4">
                {error && <Alert tone="error">{error}</Alert>}
                {step === 1 && (
                  <>
                    <PhotoPicker
                      file={photo}
                      onChange={(next) => {
                        setPhoto(next)
                        setErrors((current) => ({ ...current, photo: '' }))
                      }}
                      error={errors.photo}
                      required
                    />
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
                        value={form.regNo}
                        onChange={set('regNo')}
                        error={errors.regNo}
                        autoComplete="username"
                        required
                      />
                      <Input
                        label="Date of birth"
                        type="date"
                        value={form.dateOfBirth}
                        onChange={set('dateOfBirth')}
                        error={errors.dateOfBirth}
                        min={bounds.min}
                        max={bounds.max}
                        required
                      />
                    </div>
                  </>
                )}
                {step === 2 && (
                  <>
                    <div className="grid gap-4 sm:grid-cols-2">
                      <Input
                        label="Batch"
                        value={form.batch}
                        onChange={(event) =>
                          setForm((current) => ({
                            ...current,
                            batch: event.target.value.toUpperCase(),
                          }))
                        }
                        error={errors.batch}
                        hint="Like CSE-23"
                        placeholder="CSE-23"
                        required
                      />
                      <SectionPicker
                        value={form.section}
                        error={errors.section}
                        onChange={(section) => setForm((current) => ({ ...current, section }))}
                      />
                    </div>
                  </>
                )}
                {step === 3 && (
                  <>
                    {emailProof ? (
                      <Field label="University email">
                        <div className="flex items-center justify-between gap-3 rounded-xl border border-line-strong bg-wash px-3 py-2">
                          <span className="min-w-0 truncate text-sm text-ink">
                            {form.email}
                          </span>
                          <span className="flex shrink-0 items-center gap-2">
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
                              Google confirms your address at once, so you can skip the email code.
                            </p>
                          </>
                        )}
                      </>
                    )}
                    <div className="grid gap-4 sm:grid-cols-2">
                      <PhoneInput
                        value={form.phone}
                        onChange={(next) => setForm((current) => ({ ...current, phone: next }))}
                        error={errors.phone}
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
                      <Input
                        label="New password"
                        type="password"
                        value={form.password}
                        onChange={set('password')}
                        error={errors.password}
                        autoComplete="new-password"
                        hint="At least 8 characters"
                        required
                      />
                      <Input
                        label="Confirm password"
                        type="password"
                        value={form.confirmPassword}
                        onChange={set('confirmPassword')}
                        autoComplete="new-password"
                        required
                      />
                    </div>
                  </>
                )}
                {step === 4 && (
                  <>
                    {resuming && (
                      <Input
                        label="Registration number"
                        value={form.regNo}
                        onChange={set('regNo')}
                        autoComplete="username"
                        required
                      />
                    )}
                    <Alert tone="info" title="Confirm your email address">
                      {emailHint
                        ? `We sent a 6-digit code to ${emailHint}. It expires in 15 minutes.`
                        : 'We sent a 6-digit code to your university email address. It expires in 15 minutes.'}
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
                    <Button
                      variant="secondary"
                      onClick={resend}
                      disabled={loading || seconds > 0}
                      className="w-full"
                    >
                      {seconds > 0 ? `Resend in ${seconds}s` : 'Resend the code'}
                    </Button>
                  </>
                )}
                <div className="flex gap-2">
                  {step > 1 && step < 4 && (
                    <Button
                      variant="secondary"
                      onClick={() => {
                        setError('')
                        setStep((current) => current - 1)
                      }}
                      className="flex-1"
                    >
                      Back
                    </Button>
                  )}
                  <Button type="submit" loading={loading} className="flex-1">
                    {step === 3 ? 'Create account' : step === 4 ? 'Confirm email' : 'Continue'}
                  </Button>
                </div>
                <p className="text-center text-sm text-muted">
                  Already registered?{' '}
                  <Link to={paths.studentLogin()} className="text-brand-600 hover:underline">
                    Log in
                  </Link>
                </p>
              </form>
            </>
          )}
        </div>
      </div>
    </AuthScreen>
  )
}

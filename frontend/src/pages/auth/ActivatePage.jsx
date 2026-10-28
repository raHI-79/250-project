import { useEffect, useState } from 'react'
import { Link, useSearchParams } from 'react-router-dom'
import { Check, Copy, Eye, EyeOff, Loader2, X } from 'lucide-react'
import { Alert, Button, Input, PasswordInput, SkeletonLines, Stepper } from '../../components/ui'
import PhotoPicker from '../../components/forms/PhotoPicker'
import AuthScreen from '../../components/layout/AuthScreen'
import AuthHeading from '../../components/layout/AuthHeading'
import api from '../../lib/api'
import { HERO_BUILDING } from '../../lib/campus'
import { useDebounced } from '../../lib/hooks'
import { paths } from '../../lib/paths'
import { USER_ID_RULES, birthDateBounds, passwordError, userIdError } from '../../lib/account'

const STEPS = ['Confirm who you are', 'Choose your User ID']

function Availability({ checking, result }) {
  if (checking) {
    return (
      <span className="flex items-center gap-1.5 text-xs text-faint">
        <Loader2 size={13} className="animate-spin" />
        Checking…
      </span>
    )
  }
  if (!result) return null
  if (result.available) {
    return (
      <span className="flex items-center gap-1.5 text-xs font-medium text-ink-soft">
        <Check size={13} />
        Available
      </span>
    )
  }
  return (
    <span className="flex items-center gap-1.5 text-xs font-medium text-accent-700">
      <X size={13} />
      {result.reason}
    </span>
  )
}

export default function ActivatePage() {
  const [params] = useSearchParams()
  const token = params.get('token') || ''
  const bounds = birthDateBounds()
  const [invite, setInvite] = useState(null)
  const [loadError, setLoadError] = useState('')
  const [loadingInvite, setLoadingInvite] = useState(true)
  const [step, setStep] = useState(1)
  const [dateOfBirth, setDateOfBirth] = useState('')
  const [userId, setUserId] = useState('')
  const [availability, setAvailability] = useState(null)
  const [checking, setChecking] = useState(false)
  const [password, setPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [photo, setPhoto] = useState(null)
  const [error, setError] = useState('')
  const [saving, setSaving] = useState(false)
  const [done, setDone] = useState(null)
  const [showPassword, setShowPassword] = useState(false)
  const [copied, setCopied] = useState(false)
  const settledUserId = useDebounced(userId.trim())
  useEffect(() => {
    if (!token) {
      setLoadError('This invitation link is invalid or has expired.')
      setLoadingInvite(false)
      return undefined
    }
    let active = true
    setLoadingInvite(true)
    api
      .get('/api/auth/invite', { query: { token }, auth: false })
      .then((data) => {
        if (active) setInvite(data)
      })
      .catch((failure) => {
        if (active) setLoadError(failure.message)
      })
      .finally(() => {
        if (active) setLoadingInvite(false)
      })
    return () => {
      active = false
    }
  }, [token])
  useEffect(() => {
    if (!settledUserId) {
      setAvailability(null)
      setChecking(false)
      return undefined
    }
    const problem = userIdError(settledUserId)
    if (problem) {
      setAvailability({ available: false, reason: problem })
      setChecking(false)
      return undefined
    }
    let active = true
    setChecking(true)
    api
      .get('/api/auth/user-id-available', { query: { user_id: settledUserId }, auth: false })
      .then((data) => {
        if (active) setAvailability(data || null)
      })
      .catch(() => {
        if (active) setAvailability(null)
      })
      .finally(() => {
        if (active) setChecking(false)
      })
    return () => {
      active = false
    }
  }, [settledUserId])
  const activate = async () => {
    setError('')
    const idProblem = userIdError(userId)
    if (idProblem) {
      setError(idProblem)
      return
    }
    const problem = passwordError(password, confirmPassword)
    if (problem) {
      setError(problem)
      return
    }
    const body = new FormData()
    body.append('token', token)
    body.append('user_id', userId.trim())
    body.append('date_of_birth', dateOfBirth)
    body.append('password', password)
    body.append('confirm_password', confirmPassword)
    if (photo) body.append('photo', photo)
    setSaving(true)
    try {
      const result = await api.post('/api/auth/activate', body, { auth: false })
      setDone(result)
    } catch (failure) {
      setError(failure.message)
    } finally {
      setSaving(false)
    }
  }
  const onSubmit = (event) => {
    event.preventDefault()
    if (step === 1) {
      setError('')
      if (!dateOfBirth) {
        setError('Date of birth is required.')
        return
      }
      setStep(2)
      return
    }
    activate()
  }
  const copyUserId = async () => {
    try {
      await navigator.clipboard.writeText(done?.user_id || '')
      setCopied(true)
      window.setTimeout(() => setCopied(false), 2000)
    } catch {
      setCopied(false)
    }
  }
  return (
    <AuthScreen
      images={HERO_BUILDING}
      className="max-w-md"
      note="Teachers choose their own user ID while activating the invitation from the office."
    >
      <div>
        <AuthHeading
          eyebrow="Faculty invitation"
          title={done ? 'Your account is ready' : 'Activate your account'}
        />
        {invite && !done && (
          <p className="mt-3 text-sm text-muted">
            {invite.name}
            {invite.designation ? ` · ${invite.designation}` : ''}
          </p>
        )}
        {done && <p className="mt-3 text-sm text-muted">{done.name}</p>}
        {done ? (
          <div className="mt-5 flex flex-col gap-4">
            <div className="rounded-xl border border-line bg-wash p-4">
              <p className="text-xs font-medium tracking-wide text-muted uppercase">Your User ID</p>
              <div className="mt-1 flex items-center justify-between gap-3">
                <p className="font-display text-lg font-semibold text-ink">{done.user_id}</p>
                <Button size="sm" variant="secondary" onClick={copyUserId}>
                  <Copy size={14} />
                  {copied ? 'Copied' : 'Copy'}
                </Button>
              </div>
            </div>
            <div className="rounded-xl border border-line p-4">
              <p className="text-xs font-medium tracking-wide text-muted uppercase">Your password</p>
              <div className="mt-1 flex items-center justify-between gap-3">
                <p className="font-mono text-sm text-ink">
                  {showPassword ? password : '•'.repeat(password.length)}
                </p>
                <Button
                  size="sm"
                  variant="secondary"
                  onClick={() => setShowPassword((current) => !current)}
                >
                  {showPassword ? <EyeOff size={14} /> : <Eye size={14} />}
                  {showPassword ? 'Hide' : 'Show'}
                </Button>
              </div>
            </div>
            <Alert tone="info">
              We also emailed your User ID to {invite?.masked_email || 'your official address'}. Keep
              your password to yourself.
            </Alert>
            <Link to={paths.teacherLogin()}>
              <Button className="w-full">Go to the teacher sign-in</Button>
            </Link>
          </div>
        ) : loadingInvite ? (
          <SkeletonLines className="mt-6" />
        ) : loadError ? (
          <div className="mt-5 flex flex-col gap-4">
            <Alert tone="error" title="This link does not work">
              {loadError}
            </Alert>
            <Link
              to={paths.teacherLogin()}
              className="text-center text-sm text-brand-600 hover:underline"
            >
              Go to the teacher sign-in
            </Link>
          </div>
        ) : (
          <>
            <Stepper steps={STEPS} current={step} className="mt-5" />
            <form onSubmit={onSubmit} className="mt-5 flex flex-col gap-4">
              {error && <Alert tone="error">{error}</Alert>}
              {step === 1 ? (
                <>
                  <Alert tone="info">
                    The invitation was sent to {invite?.masked_email || 'your official address'}.
                    Confirm your date of birth to continue.
                  </Alert>
                  <Input
                    label="Date of birth"
                    type="date"
                    value={dateOfBirth}
                    onChange={(event) => setDateOfBirth(event.target.value)}
                    min={bounds.min}
                    max={bounds.max}
                    required
                  />
                </>
              ) : (
                <>
                  <div className="flex flex-col gap-1.5">
                    <Input
                      label="Choose your User ID"
                      value={userId}
                      onChange={(event) => setUserId(event.target.value)}
                      autoComplete="username"
                      placeholder="karim.cse"
                      hint={USER_ID_RULES}
                      required
                    />
                    <Availability checking={checking} result={availability} />
                  </div>
                  <PasswordInput
                    label="New password"
                    value={password}
                    onChange={(event) => setPassword(event.target.value)}
                    autoComplete="new-password"
                    hint="At least 8 characters"
                    required
                  />
                  <PasswordInput
                    label="Confirm password"
                    value={confirmPassword}
                    onChange={(event) => setConfirmPassword(event.target.value)}
                    autoComplete="new-password"
                    required
                  />
                  <PhotoPicker
                    file={photo}
                    onChange={setPhoto}
                    label="Profile photo"
                    hint="Optional, JPG, PNG or WEBP, at most 2 MB, at least 200 by 200 pixels"
                  />
                </>
              )}
              <div className="flex gap-2">
                {step === 2 && (
                  <Button
                    variant="secondary"
                    onClick={() => {
                      setError('')
                      setStep(1)
                    }}
                    className="flex-1"
                  >
                    Back
                  </Button>
                )}
                <Button type="submit" loading={saving} className="flex-1">
                  {step === 1 ? 'Continue' : 'Activate my account'}
                </Button>
              </div>
            </form>
          </>
        )}
      </div>
    </AuthScreen>
  )
}

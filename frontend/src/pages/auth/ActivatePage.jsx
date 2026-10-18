import { useEffect, useState } from 'react'
import { Link, useNavigate, useSearchParams } from 'react-router-dom'
import { KeyRound } from 'lucide-react'
import { Alert, Button, Card, Input, SkeletonLines, Stepper } from '../../components/ui'
import PhotoPicker from '../../components/forms/PhotoPicker'
import AuthScreen from '../../components/layout/AuthScreen'
import api from '../../lib/api'
import { HERO_BUILDING } from '../../lib/campus'
import { paths } from '../../lib/paths'
import { birthDateBounds, passwordError } from '../../lib/account'

const OVERLAY = 'bg-brand-950/92'

const STEPS = ['Confirm who you are', 'Set your password']

export default function ActivatePage() {
  const [params] = useSearchParams()
  const token = params.get('token') || ''
  const navigate = useNavigate()
  const bounds = birthDateBounds()
  const [invite, setInvite] = useState(null)
  const [loadError, setLoadError] = useState('')
  const [loadingInvite, setLoadingInvite] = useState(true)
  const [step, setStep] = useState(1)
  const [teacherId, setTeacherId] = useState('')
  const [dateOfBirth, setDateOfBirth] = useState('')
  const [password, setPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [photo, setPhoto] = useState(null)
  const [error, setError] = useState('')
  const [saving, setSaving] = useState(false)
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
  const activate = async () => {
    setError('')
    const problem = passwordError(password, confirmPassword)
    if (problem) {
      setError(problem)
      return
    }
    const body = new FormData()
    body.append('token', token)
    body.append('teacher_id', teacherId.trim())
    body.append('date_of_birth', dateOfBirth)
    body.append('password', password)
    body.append('confirm_password', confirmPassword)
    if (photo) body.append('photo', photo)
    setSaving(true)
    try {
      await api.post('/api/auth/activate', body, { auth: false })
      navigate(paths.teacherLogin(), {
        replace: true,
        state: { notice: 'Your account is active. Log in with your teacher ID and new password.' },
      })
    } catch (failure) {
      setError(failure.message)
      setStep(1)
    } finally {
      setSaving(false)
    }
  }
  const onSubmit = (event) => {
    event.preventDefault()
    if (step === 1) {
      setError('')
      if (!teacherId.trim()) {
        setError('Teacher ID is required.')
        return
      }
      if (!dateOfBirth) {
        setError('Date of birth is required.')
        return
      }
      setStep(2)
      return
    }
    activate()
  }
  return (
    <AuthScreen images={HERO_BUILDING} overlay={OVERLAY} className="max-w-md">
      <Card hover={false} className="animate-fade-in-up w-full">
        <div className="text-center">
          <span className="inline-flex size-11 items-center justify-center rounded-full bg-brand-50 text-brand-600">
            <KeyRound size={22} />
          </span>
          <p className="font-display mt-3 text-xl font-semibold text-brand-900">
            Activate your account
          </p>
          {invite && (
            <p className="mt-1 text-sm text-brand-500">
              {invite.name}
              {invite.designation ? ` · ${invite.designation}` : ''}
            </p>
          )}
        </div>
        {loadingInvite ? (
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
              Go to the teacher login
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
                    Confirm your teacher ID and date of birth to continue.
                  </Alert>
                  <Input
                    label="Teacher ID"
                    value={teacherId}
                    onChange={(event) => setTeacherId(event.target.value)}
                    autoComplete="username"
                    required
                  />
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
                    label="Confirm password"
                    type="password"
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
      </Card>
    </AuthScreen>
  )
}

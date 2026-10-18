import { useState } from 'react'
import { Link, Navigate, useLocation, useNavigate } from 'react-router-dom'
import { GraduationCap, Users } from 'lucide-react'
import Alert from '../../components/ui/Alert'
import Button from '../../components/ui/Button'
import Card from '../../components/ui/Card'
import { Input } from '../../components/ui/Field'
import AuthScreen from '../../components/layout/AuthScreen'
import { useAuth } from '../../context/auth'
import { HERO_IMAGES } from '../../lib/campus'
import { PORTAL_TEACHER, homeFor, paths } from '../../lib/paths'

const SLIDE_AFTER = 6000

const OVERLAY = 'bg-linear-to-br from-brand-950/88 via-brand-800/72 to-brand-600/45'

const VERIFY_EMAIL = 'Please verify your email first.'

const STUDENT_PORTAL = 'Use the student login page.'

const TEACHER_PORTAL = 'Use the teacher login page.'

const copy = {
  student: {
    heading: 'SUST CSE',
    subtitle: 'Class and exam coordination for the Department of CSE',
    identifier: 'Registration number',
    forgot: () => paths.studentForgotPassword(),
    otherLabel: 'Are you a teacher?',
    otherLink: () => paths.teacherLogin(),
    otherIcon: GraduationCap,
  },
  teacher: {
    heading: 'Faculty login',
    subtitle: 'Department of CSE, Shahjalal University of Science and Technology',
    identifier: 'Teacher ID',
    forgot: () => paths.teacherForgotPassword(),
    otherLabel: 'Are you a student?',
    otherLink: () => paths.studentLogin(),
    otherIcon: Users,
  },
}

export default function LoginPage({ portal }) {
  const { login, isAuthenticated, role } = useAuth()
  const navigate = useNavigate()
  const location = useLocation()
  const [identifier, setIdentifier] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)
  const page = copy[portal] || copy.student
  const notice = location.state?.notice || ''
  if (isAuthenticated) return <Navigate to={homeFor(role)} replace />
  const onSubmit = async (event) => {
    event.preventDefault()
    setError('')
    setLoading(true)
    try {
      const user = await login(identifier.trim(), password, portal)
      navigate(homeFor(user.role), { replace: true })
    } catch (failure) {
      setError(failure.message)
    } finally {
      setLoading(false)
    }
  }
  const OtherIcon = page.otherIcon
  return (
    <AuthScreen images={HERO_IMAGES} rotateAfter={SLIDE_AFTER} overlay={OVERLAY}>
      <Card hover={false} className="animate-fade-in-up w-full backdrop-blur-sm">
        <div className="text-center">
          <p className="font-display text-2xl font-semibold text-brand-700">{page.heading}</p>
          <p className="mt-1 text-sm text-brand-500">{page.subtitle}</p>
        </div>
        <form onSubmit={onSubmit} className="mt-6 flex flex-col gap-4">
          {notice && <Alert tone="success">{notice}</Alert>}
          {error && (
            <Alert tone="error">
              <p>{error}</p>
              {error === VERIFY_EMAIL && (
                <Link
                  to={paths.studentVerifyEmail(identifier.trim())}
                  className="mt-1 inline-block font-medium text-accent-800 underline"
                >
                  Verify email
                </Link>
              )}
              {error === STUDENT_PORTAL && (
                <Link
                  to={paths.studentLogin()}
                  className="mt-1 inline-block font-medium text-accent-800 underline"
                >
                  Go to the student login
                </Link>
              )}
              {error === TEACHER_PORTAL && (
                <Link
                  to={paths.teacherLogin()}
                  className="mt-1 inline-block font-medium text-accent-800 underline"
                >
                  Go to the teacher login
                </Link>
              )}
            </Alert>
          )}
          <Input
            label={page.identifier}
            value={identifier}
            onChange={(event) => setIdentifier(event.target.value)}
            autoComplete="username"
            required
          />
          <Input
            label="Password"
            type="password"
            value={password}
            onChange={(event) => setPassword(event.target.value)}
            autoComplete="current-password"
            required
          />
          <Button type="submit" loading={loading} className="w-full">
            Log in
          </Button>
        </form>
        <div className="mt-5 flex flex-col gap-1 text-center text-sm">
          <Link to={page.forgot()} className="text-brand-600 hover:underline">
            Forgot your password?
          </Link>
          {portal === PORTAL_TEACHER ? (
            <p className="text-brand-500">
              Invited but not activated?{' '}
              <Link to={paths.teacherActivate()} className="text-brand-600 hover:underline">
                Use your invitation link
              </Link>
            </p>
          ) : (
            <p className="text-brand-500">
              New student?{' '}
              <Link to={paths.studentRegister()} className="text-brand-600 hover:underline">
                Create an account
              </Link>
            </p>
          )}
        </div>
        <Link
          to={page.otherLink()}
          className="mt-4 flex items-center justify-center gap-1.5 border-t border-brand-100 pt-4 text-xs text-brand-500 transition-colors hover:text-brand-700"
        >
          <OtherIcon size={14} />
          {page.otherLabel}
        </Link>
      </Card>
    </AuthScreen>
  )
}

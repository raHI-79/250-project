import { useState } from 'react'
import { Link } from 'react-router-dom'
import Alert from '../../components/ui/Alert'
import Button from '../../components/ui/Button'
import Card from '../../components/ui/Card'
import { Field, Input, Select } from '../../components/ui/Field'
import CampusBackdrop from '../../components/layout/CampusBackdrop'
import api from '../../lib/api'
import { BANNER_MONUMENT } from '../../lib/campus'
import { paths } from '../../lib/paths'

const BLOOD_GROUPS = ['A+', 'A-', 'B+', 'B-', 'AB+', 'AB-', 'O+', 'O-']

const MAX_FIELD = 120

const PATTERNS = {
  batch: /^[A-Z]{2,6}-[0-9]{2}$/,
  section: /^[A-Z]$/,
  email: /^[^@\s]+@[^@\s]+\.[^@\s.]{2,}$/,
  phone: /^\+?[0-9]{10,15}$/,
  courseCode: /^[A-Z]{2,6}-?[0-9]{3}$/,
}

const EMPTY = {
  fullName: '',
  regNo: '',
  batch: '',
  section: '',
  email: '',
  phone: '',
  bloodGroup: '',
  password: '',
  confirmPassword: '',
  hasDropCourse: false,
  dropCourseCode: '',
}

const checkForm = (form) => {
  const errors = {}
  const name = form.fullName.trim()
  if (!name) errors.fullName = 'Full name is required.'
  else if (name.length > MAX_FIELD) errors.fullName = `Full name must be at most ${MAX_FIELD} characters long.`
  if (!form.regNo.trim()) errors.regNo = 'Registration number is required.'
  if (!PATTERNS.batch.test(form.batch.trim().toUpperCase())) errors.batch = 'Batch must look like CSE-23.'
  if (!PATTERNS.section.test(form.section.trim().toUpperCase()))
    errors.section = 'Section must be a single letter from A to Z.'
  if (!PATTERNS.email.test(form.email.trim().toLowerCase()))
    errors.email = 'Email must be a valid email address.'
  if (!PATTERNS.phone.test(form.phone.trim()))
    errors.phone = 'Phone must be 10 to 15 digits, with an optional leading +.'
  if (!BLOOD_GROUPS.includes(form.bloodGroup)) errors.bloodGroup = 'Please choose your blood group.'
  if (form.password.length < 8) errors.password = 'Password must be at least 8 characters long.'
  else if (form.password.length > MAX_FIELD)
    errors.password = `Password must be at most ${MAX_FIELD} characters long.`
  if (form.password !== form.confirmPassword)
    errors.confirmPassword = 'Password and the confirmation do not match.'
  if (form.hasDropCourse && !PATTERNS.courseCode.test(form.dropCourseCode.trim().toUpperCase()))
    errors.dropCourseCode = 'Drop course code must look like CSE-233 or CSE233.'
  return errors
}

const toPayload = (form) => ({
  full_name: form.fullName.trim(),
  reg_no: form.regNo.trim(),
  batch: form.batch.trim().toUpperCase(),
  section: form.section.trim().toUpperCase(),
  email: form.email.trim().toLowerCase(),
  phone: form.phone.trim(),
  blood_group: form.bloodGroup,
  password: form.password,
  confirm_password: form.confirmPassword,
  has_drop_course: form.hasDropCourse,
  drop_course_code: form.hasDropCourse ? form.dropCourseCode.trim().toUpperCase() : '',
})

export default function RegisterPage() {
  const [form, setForm] = useState(EMPTY)
  const [errors, setErrors] = useState({})
  const [error, setError] = useState('')
  const [done, setDone] = useState('')
  const [loading, setLoading] = useState(false)
  const set = (key) => (event) => {
    const { value } = event.target
    setForm((current) => ({ ...current, [key]: value }))
  }
  const setDropCourse = (hasDropCourse) => () => {
    setForm((current) => ({
      ...current,
      hasDropCourse,
      dropCourseCode: hasDropCourse ? current.dropCourseCode : '',
    }))
  }
  const onSubmit = async (event) => {
    event.preventDefault()
    setError('')
    const found = checkForm(form)
    setErrors(found)
    if (Object.keys(found).length > 0) return
    setLoading(true)
    try {
      const data = await api.post('/api/auth/register', toPayload(form), { auth: false })
      setDone(data?.message || 'Registration submitted.')
      setForm(EMPTY)
    } catch (failure) {
      setError(failure.message)
    } finally {
      setLoading(false)
    }
  }
  return (
    <div className="flex min-h-screen items-start justify-center bg-page px-4 py-10">
      <Card hover={false} className="animate-fade-in-up w-full max-w-2xl p-0">
        <div className="relative h-36 overflow-hidden rounded-t-xl sm:h-44">
          <CampusBackdrop
            images={BANNER_MONUMENT}
            overlay="bg-linear-to-t from-brand-950/85 via-brand-900/45 to-brand-700/20"
          />
          <div className="relative flex h-full flex-col justify-end p-5 sm:p-6">
            <p className="font-display text-xl font-semibold text-white sm:text-2xl">
              Student registration
            </p>
            <p className="mt-1 text-sm text-brand-100">
              Shahjalal University of Science and Technology, Department of CSE
            </p>
          </div>
        </div>
        <div className="p-5 sm:p-6">
          {done ? (
            <div className="flex flex-col gap-4">
              <Alert tone="success" title="Registration submitted">
                {done} The administration reviews new accounts, so you cannot log in until your
                registration is approved.
              </Alert>
              <Link
                to={paths.login()}
                className="text-center text-sm text-brand-600 hover:underline"
              >
                Back to log in
              </Link>
            </div>
          ) : (
            <form onSubmit={onSubmit} className="flex flex-col gap-4">
              {error && <Alert tone="error">{error}</Alert>}
              <div className="grid gap-4 sm:grid-cols-2">
                <Input
                  label="Full name"
                  value={form.fullName}
                  onChange={set('fullName')}
                  error={errors.fullName}
                  autoComplete="name"
                  className="sm:col-span-2"
                  required
                />
                <Input
                  label="Registration number"
                  value={form.regNo}
                  onChange={set('regNo')}
                  error={errors.regNo}
                  autoComplete="username"
                  required
                />
                <Input
                  label="Batch"
                  value={form.batch}
                  onChange={set('batch')}
                  error={errors.batch}
                  hint="For example CSE-23"
                  required
                />
                <Input
                  label="Section"
                  value={form.section}
                  onChange={set('section')}
                  error={errors.section}
                  maxLength={1}
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
                  label="Email"
                  type="email"
                  value={form.email}
                  onChange={set('email')}
                  error={errors.email}
                  autoComplete="email"
                  required
                />
                <Input
                  label="Phone"
                  value={form.phone}
                  onChange={set('phone')}
                  error={errors.phone}
                  autoComplete="tel"
                  hint="For example 01712345678"
                  required
                />
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
                  error={errors.confirmPassword}
                  autoComplete="new-password"
                  required
                />
              </div>
              <Field label="Do you have any drop course?">
                <div className="flex gap-2">
                  <Button
                    variant={form.hasDropCourse ? 'primary' : 'secondary'}
                    onClick={setDropCourse(true)}
                  >
                    Yes
                  </Button>
                  <Button
                    variant={form.hasDropCourse ? 'secondary' : 'primary'}
                    onClick={setDropCourse(false)}
                  >
                    No
                  </Button>
                </div>
              </Field>
              {form.hasDropCourse && (
                <Input
                  label="Drop course code"
                  value={form.dropCourseCode}
                  onChange={set('dropCourseCode')}
                  error={errors.dropCourseCode}
                  hint="For example CSE-233"
                  className="animate-fade-in"
                  required
                />
              )}
              <Button type="submit" loading={loading} className="w-full">
                Create account
              </Button>
              <p className="text-center text-sm text-brand-500">
                Already registered?{' '}
                <Link to={paths.login()} className="text-brand-600 hover:underline">
                  Log in
                </Link>
              </p>
            </form>
          )}
        </div>
      </Card>
    </div>
  )
}

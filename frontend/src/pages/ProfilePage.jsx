import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Mail, ShieldCheck } from 'lucide-react'
import IdentityCard from '../components/IdentityCard'
import {
  Alert,
  Button,
  Card,
  Input,
  PageHeader,
  SkeletonCard,
  StatusBadge,
  Textarea,
  useToast,
} from '../components/ui'
import CourseMultiSelect from '../components/forms/CourseMultiSelect'
import PhoneInput from '../components/forms/PhoneInput'
import PhotoPicker from '../components/forms/PhotoPicker'
import { useAuth } from '../context/auth'
import api from '../lib/api'
import { useResource } from '../lib/hooks'
import { loginFor } from '../lib/paths'
import { MAX_BIO, PHONE_MESSAGE, accountRoleLabel, isPhone, passwordError } from '../lib/account'
import { formatDate } from '../lib/time'

const ROW_LABEL = 'text-xs font-medium tracking-wide text-faint uppercase'

function Row({ label, children }) {
  return (
    <div className="flex flex-col gap-0.5 py-2">
      <span className={ROW_LABEL}>{label}</span>
      <span className="text-sm text-ink">{children || '—'}</span>
    </div>
  )
}

function Section({ title, icon: Icon, actions, children }) {
  return (
    <Card hover={false}>
      <div className="flex items-center justify-between gap-3 border-b border-line pb-3">
        <h2 className="font-display inline-flex items-center gap-2 text-base font-semibold text-ink">
          {Icon && <Icon size={16} className="text-muted" />}
          {title}
        </h2>
        {actions}
      </div>
      <div className="pt-2">{children}</div>
    </Card>
  )
}

export default function ProfilePage() {
  const { role, updateUser, logout } = useAuth()
  const navigate = useNavigate()
  const toast = useToast()
  const { data, loading, error, reload } = useResource('/api/profile')
  const [saveError, setSaveError] = useState('')
  const [saving, setSaving] = useState(false)
  const [photo, setPhoto] = useState(null)
  const [editContact, setEditContact] = useState(false)
  const [editWork, setEditWork] = useState(false)
  const [editCourses, setEditCourses] = useState(false)
  const [phone, setPhone] = useState('')
  const [officeRoom, setOfficeRoom] = useState('')
  const [bio, setBio] = useState('')
  const [contactEmail, setContactEmail] = useState('')
  const [contactPhone, setContactPhone] = useState('')
  const [courses, setCourses] = useState([])
  const [currentPassword, setCurrentPassword] = useState('')
  const [newPassword, setNewPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [passwordProblem, setPasswordProblem] = useState('')
  const [changing, setChanging] = useState(false)
  useEffect(() => {
    if (!data) return
    setPhone(data.phone || '')
    setOfficeRoom(data.office_room || '')
    setBio(data.bio || '')
    setContactEmail(data.contact_email || '')
    setContactPhone(data.contact_phone || '')
    setCourses(data.drop_courses || [])
  }, [data])
  const isStudent = role === 'student' || role === 'cr'
  const isTeacher = role === 'teacher'
  const isAdmin = role === 'admin'
  const displayName = isAdmin ? data?.identifier : data?.full_name || data?.name
  const save = async (fields, done) => {
    setSaveError('')
    const body = new FormData()
    Object.entries(fields).forEach(([key, value]) => {
      if (Array.isArray(value)) {
        if (value.length === 0) body.append(key, '')
        else value.forEach((entry) => body.append(key, String(entry)))
        return
      }
      body.append(key, value)
    })
    setSaving(true)
    try {
      const next = await api.patch('/api/profile', body)
      if (next?.photo_url !== undefined) updateUser({ photo_url: next.photo_url })
      toast.success('Profile updated')
      done?.()
      reload()
    } catch (failure) {
      setSaveError(failure.message)
    } finally {
      setSaving(false)
    }
  }
  const savePhoto = async () => {
    if (!photo) return
    const body = new FormData()
    body.append('photo', photo)
    setSaveError('')
    setSaving(true)
    try {
      const next = await api.patch('/api/profile', body)
      if (next?.photo_url !== undefined) updateUser({ photo_url: next.photo_url })
      toast.success('Photo updated')
      setPhoto(null)
      reload()
    } catch (failure) {
      setSaveError(failure.message)
    } finally {
      setSaving(false)
    }
  }
  const changePassword = async (event) => {
    event.preventDefault()
    setPasswordProblem('')
    const problem = passwordError(newPassword, confirmPassword)
    if (problem) {
      setPasswordProblem(problem.replace('Password', 'New password'))
      return
    }
    setChanging(true)
    try {
      await api.post('/api/profile/password', {
        current_password: currentPassword,
        new_password: newPassword,
        confirm_password: confirmPassword,
      })
      const target = loginFor(role)
      await logout()
      navigate(target, {
        replace: true,
        state: { notice: 'Password updated. Please log in with your new password.' },
      })
    } catch (failure) {
      setPasswordProblem(failure.message)
    } finally {
      setChanging(false)
    }
  }
  if (loading) {
    return (
      <>
        <PageHeader eyebrow="Your account" title="My Profile" subtitle="Your account details" />
        <div className="grid gap-5 lg:grid-cols-[20rem_1fr]">
          <SkeletonCard />
          <div className="flex flex-col gap-5">
            <SkeletonCard />
            <SkeletonCard />
          </div>
        </div>
      </>
    )
  }
  if (error) {
    return (
      <>
        <PageHeader eyebrow="Your account" title="My Profile" subtitle="Your account details" />
        <Alert tone="error">{error}</Alert>
      </>
    )
  }
  return (
    <>
      <PageHeader eyebrow="Your account" title="My Profile" subtitle="Your account details and sign-in settings" />
      {saveError && (
        <Alert tone="error" className="mb-5">
          {saveError}
        </Alert>
      )}
      <div className="grid gap-5 lg:grid-cols-[20rem_1fr] lg:items-start">
        <IdentityCard
          name={displayName}
          photoUrl={data?.photo_url}
          role={accountRoleLabel(role)}
          chips={[
            role === 'cr' ? (
              <StatusBadge key="cr" status="approved" label="Class Representative" />
            ) : null,
            data?.status ? <StatusBadge key="status" status={data.status} /> : null,
          ].filter(Boolean)}
          rows={[
            isStudent ? { label: 'Batch', value: data?.batch } : null,
            isStudent ? { label: 'Section', value: data?.section } : null,
            isTeacher ? { label: 'Designation', value: data?.designation } : null,
            isTeacher ? { label: 'Office room', value: data?.office_room } : null,
            isAdmin ? { label: 'Role', value: accountRoleLabel(role) } : null,
            data?.created_at
              ? { label: 'Member since', value: formatDate(data.created_at) }
              : null,
          ].filter(Boolean)}
          code={data?.reg_no || data?.teacher_id || data?.identifier}
          footer={
            <div className="border-t border-line p-5">
              <PhotoPicker
                file={photo}
                onChange={setPhoto}
                currentUrl={data?.photo_url || ''}
                label="Change photo"
                hint="JPG, PNG or WEBP, at most 2 MB, at least 200 by 200 pixels"
              />
              {photo && (
                <Button loading={saving} onClick={savePhoto} className="mt-3 w-full">
                  Save the new photo
                </Button>
              )}
            </div>
          }
        />
        <div className="flex flex-col gap-5">
          {isStudent && (
            <Section
              title="Academic"
              actions={
                <Button
                  variant="secondary"
                  size="sm"
                  onClick={() => {
                    setCourses(data?.drop_courses || [])
                    setEditCourses((current) => !current)
                  }}
                >
                  {editCourses ? 'Cancel' : 'Edit'}
                </Button>
              }
            >
              <div className="grid gap-x-6 sm:grid-cols-2">
                <Row label="Batch">{data?.batch}</Row>
                <Row label="Section">{data?.section}</Row>
                <Row label="Date of birth">
                  {data?.date_of_birth ? formatDate(data.date_of_birth) : ''}
                </Row>
                <Row label="Blood group">{data?.blood_group}</Row>
              </div>
              {editCourses ? (
                <div className="mt-2 flex flex-col gap-3">
                  <CourseMultiSelect value={courses} onChange={setCourses} />
                  <div className="flex gap-2">
                    <Button
                      loading={saving}
                      onClick={() =>
                        save({ drop_course_ids: courses.map((course) => course.id) }, () =>
                          setEditCourses(false),
                        )
                      }
                    >
                      Save drop courses
                    </Button>
                  </div>
                </div>
              ) : (
                <div className="py-2">
                  <span className={ROW_LABEL}>Drop courses</span>
                  {(data?.drop_courses || []).length === 0 ? (
                    <p className="text-sm text-muted">No drop course</p>
                  ) : (
                    <div className="mt-1 flex flex-wrap gap-2">
                      {data.drop_courses.map((course) => (
                        <StatusBadge key={course.id} status="approved" label={course.code} />
                      ))}
                    </div>
                  )}
                </div>
              )}
            </Section>
          )}
          {isTeacher && (
            <Section
              title="Work"
              actions={
                <Button
                  variant="secondary"
                  size="sm"
                  onClick={() => {
                    setOfficeRoom(data?.office_room || '')
                    setBio(data?.bio || '')
                    setEditWork((current) => !current)
                  }}
                >
                  {editWork ? 'Cancel' : 'Edit'}
                </Button>
              }
            >
              <div className="grid gap-x-6 sm:grid-cols-2">
                <Row label="Designation">{data?.designation}</Row>
                <Row label="Joining date">
                  {data?.joining_date ? formatDate(data.joining_date) : ''}
                </Row>
                <Row label="Date of birth">
                  {data?.date_of_birth ? formatDate(data.date_of_birth) : ''}
                </Row>
                <Row label="Blood group">{data?.blood_group}</Row>
              </div>
              {editWork ? (
                <div className="mt-2 flex flex-col gap-3">
                  <Input
                    label="Office room"
                    value={officeRoom}
                    onChange={(event) => setOfficeRoom(event.target.value)}
                  />
                  <Textarea
                    label="Bio"
                    value={bio}
                    onChange={(event) => setBio(event.target.value)}
                    maxLength={MAX_BIO}
                    hint={`${bio.length} of ${MAX_BIO} characters`}
                  />
                  <div className="flex gap-2">
                    <Button
                      loading={saving}
                      onClick={() =>
                        save({ office_room: officeRoom.trim(), bio: bio.trim() }, () =>
                          setEditWork(false),
                        )
                      }
                    >
                      Save changes
                    </Button>
                  </div>
                </div>
              ) : (
                <>
                  <Row label="Office room">{data?.office_room}</Row>
                  <Row label="Bio">{data?.bio}</Row>
                </>
              )}
            </Section>
          )}
          <Section
            title="Contact"
            icon={Mail}
            actions={
              <Button
                variant="secondary"
                size="sm"
                onClick={() => {
                  setPhone(data?.phone || '')
                  setContactEmail(data?.contact_email || '')
                  setContactPhone(data?.contact_phone || '')
                  setEditContact((current) => !current)
                }}
              >
                {editContact ? 'Cancel' : 'Edit'}
              </Button>
            }
          >
            {isAdmin ? (
              editContact ? (
                <div className="flex flex-col gap-3 pt-1">
                  <Input
                    label="Contact email"
                    type="email"
                    value={contactEmail}
                    onChange={(event) => setContactEmail(event.target.value)}
                  />
                  <PhoneInput
                    label="Contact phone"
                    value={contactPhone}
                    onChange={setContactPhone}
                  />
                  <div className="flex gap-2">
                    <Button
                      loading={saving}
                      disabled={Boolean(contactPhone) && !isPhone(contactPhone)}
                      onClick={() => {
                        const fields = {}
                        if (contactEmail.trim()) fields.contact_email = contactEmail.trim()
                        if (contactPhone.trim()) fields.contact_phone = contactPhone.trim()
                        save(fields, () => setEditContact(false))
                      }}
                    >
                      Save changes
                    </Button>
                  </div>
                </div>
              ) : (
                <>
                  <Row label="Contact email">{data?.contact_email}</Row>
                  <Row label="Contact phone">{data?.contact_phone}</Row>
                </>
              )
            ) : (
              <>
                <Row label="Email">{data?.email}</Row>
                {editContact ? (
                  <div className="flex flex-col gap-3 pt-1">
                    <PhoneInput value={phone} onChange={setPhone} />
                    <div className="flex gap-2">
                      <Button
                        loading={saving}
                        disabled={!isPhone(phone)}
                        onClick={() => save({ phone: phone.trim() }, () => setEditContact(false))}
                      >
                        Save phone number
                      </Button>
                    </div>
                    {!isPhone(phone) && phone && (
                      <p className="text-xs text-accent-700">{PHONE_MESSAGE}</p>
                    )}
                  </div>
                ) : (
                  <Row label="Phone">{data?.phone}</Row>
                )}
              </>
            )}
            {!isAdmin && (
              <p className="pt-1 text-xs text-faint">
                The administration keeps your name, ID and email address up to date.
              </p>
            )}
          </Section>
          <Section title="Security" icon={ShieldCheck}>
            <form onSubmit={changePassword} className="flex flex-col gap-3 pt-1">
              {passwordProblem && <Alert tone="error">{passwordProblem}</Alert>}
              <Input
                label="Current password"
                type="password"
                value={currentPassword}
                onChange={(event) => setCurrentPassword(event.target.value)}
                autoComplete="current-password"
                required
              />
              <div className="grid gap-3 sm:grid-cols-2">
                <Input
                  label="New password"
                  type="password"
                  value={newPassword}
                  onChange={(event) => setNewPassword(event.target.value)}
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
              </div>
              <p className="text-xs text-faint">
                Changing your password signs you out everywhere, so you log in again afterwards.
              </p>
              <div className="flex gap-2">
                <Button type="submit" loading={changing}>
                  Change password
                </Button>
              </div>
            </form>
          </Section>
        </div>
      </div>
    </>
  )
}

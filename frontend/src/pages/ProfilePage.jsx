import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import {
  BookOpen,
  Briefcase,
  Camera,
  KeyRound,
  LogOut,
  Mail,
  Palette,
} from 'lucide-react'
import IdentityCard from '../components/IdentityCard'
import {
  Alert,
  Button,
  Card,
  Input,
  ListRow,
  PasswordInput,
  PageHeader,
  Segmented,
  Sheet,
  SkeletonCard,
  StatusBadge,
  Textarea,
  useConfirm,
  useToast,
} from '../components/ui'
import CourseMultiSelect from '../components/forms/CourseMultiSelect'
import PhoneInput from '../components/forms/PhoneInput'
import PhotoPicker from '../components/forms/PhotoPicker'
import { useAuth } from '../context/auth'
import { useTheme } from '../context/theme'
import api from '../lib/api'
import { useResource } from '../lib/hooks'
import { loginFor } from '../lib/paths'
import { THEME_CHOICES } from '../lib/theme'
import { MAX_BIO, PHONE_MESSAGE, accountRoleLabel, isPhone, passwordError } from '../lib/account'
import { formatDate } from '../lib/time'

export default function ProfilePage() {
  const { role, updateUser, logout } = useAuth()
  const theme = useTheme()
  const navigate = useNavigate()
  const toast = useToast()
  const confirm = useConfirm()
  const confirmDropCourse = (course) =>
    confirm({
      title: 'Remove this drop course?',
      message: `${course.title || course.code} disappears from your today, routine and exams.`,
      confirmLabel: 'Remove',
    })
  const { data, loading, error, reload } = useResource('/api/profile')
  const [sheet, setSheet] = useState('')
  const [saveError, setSaveError] = useState('')
  const [saving, setSaving] = useState(false)
  const [photo, setPhoto] = useState(null)
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
  const close = () => {
    setSheet('')
    setSaveError('')
  }
  const save = async (fields) => {
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
      close()
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
      close()
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
        <PageHeader eyebrow="Your account" title="My profile" />
        <div className="grid gap-5 lg:grid-cols-[22rem_1fr]">
          <SkeletonCard />
          <SkeletonCard />
        </div>
      </>
    )
  }
  if (error) {
    return (
      <>
        <PageHeader eyebrow="Your account" title="My profile" />
        <Alert tone="error">{error}</Alert>
      </>
    )
  }
  const idRows = [
    isStudent ? { label: 'Batch', value: data?.batch } : null,
    isStudent ? { label: 'Section', value: data?.section } : null,
    isTeacher ? { label: 'Designation', value: data?.designation } : null,
    isTeacher ? { label: 'Office room', value: data?.office_room } : null,
    { label: 'Blood group', value: data?.blood_group },
    data?.created_at ? { label: 'Member since', value: formatDate(data.created_at) } : null,
  ].filter(Boolean)
  const code = data?.reg_no || data?.teacher_id || data?.identifier
  const codeLabel = isStudent ? 'Registration number' : 'User ID'
  const themeLabel = THEME_CHOICES.find((entry) => entry.value === theme.choice)?.label || 'System'
  return (
    <>
      <PageHeader eyebrow="Your account" title="My profile" />
      <div className="grid gap-5 lg:grid-cols-[22rem_1fr] lg:items-start">
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
          rows={idRows}
          code={code}
          codeLabel={codeLabel}
        />
        <div className="flex flex-col gap-5">
          <Card hover={false} flush className="divide-y divide-line overflow-hidden">
            <ListRow icon={Camera} title="Change photo" onClick={() => setSheet('photo')} />
            {isStudent && (
              <ListRow
                icon={BookOpen}
                title="My drop courses"
                subtitle={
                  (data?.drop_courses || []).length === 0
                    ? 'No drop course'
                    : data.drop_courses.map((course) => course.code).join(', ')
                }
                onClick={() => setSheet('courses')}
              />
            )}
            {isTeacher && (
              <ListRow
                icon={Briefcase}
                title="Office and bio"
                subtitle={data?.office_room || 'Not set'}
                onClick={() => setSheet('work')}
              />
            )}
            <ListRow
              icon={Mail}
              title="Contact"
              subtitle={isAdmin ? data?.contact_email || 'Not set' : data?.email}
              onClick={() => setSheet('contact')}
            />
          </Card>
          <Card hover={false} flush className="divide-y divide-line overflow-hidden">
            <ListRow
              icon={Palette}
              title="Appearance"
              subtitle="Light, dark or follow your phone"
              value={themeLabel}
              onClick={() => setSheet('theme')}
            />
            <ListRow
              icon={KeyRound}
              title="Change password"
              onClick={() => setSheet('password')}
            />
            <ListRow icon={LogOut} title="Sign out" danger onClick={logout} chevron={false} />
          </Card>
        </div>
      </div>
      <Sheet open={sheet === 'photo'} onClose={close} title="Change photo">
        {saveError && (
          <Alert tone="error" className="mb-4">
            {saveError}
          </Alert>
        )}
        <PhotoPicker
          file={photo}
          onChange={setPhoto}
          currentUrl={data?.photo_url || ''}
          label="New photo"
          hint="JPG, PNG or WEBP, at most 2 MB, at least 200 by 200 pixels"
        />
        <Button loading={saving} disabled={!photo} onClick={savePhoto} className="mt-4 w-full">
          Save the new photo
        </Button>
      </Sheet>
      <Sheet
        open={sheet === 'courses'}
        onClose={close}
        title="My drop courses"
        description="Courses you repeat with another batch"
      >
        {saveError && (
          <Alert tone="error" className="mb-4">
            {saveError}
          </Alert>
        )}
        <CourseMultiSelect value={courses} onChange={setCourses} onRemove={confirmDropCourse} />
        <Button
          loading={saving}
          onClick={() => save({ drop_course_ids: courses.map((course) => course.id) })}
          className="mt-4 w-full"
        >
          Save drop courses
        </Button>
      </Sheet>
      <Sheet open={sheet === 'work'} onClose={close} title="Office and bio">
        {saveError && (
          <Alert tone="error" className="mb-4">
            {saveError}
          </Alert>
        )}
        <div className="flex flex-col gap-4">
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
          <Button
            loading={saving}
            onClick={() => save({ office_room: officeRoom.trim(), bio: bio.trim() })}
          >
            Save changes
          </Button>
        </div>
      </Sheet>
      <Sheet open={sheet === 'contact'} onClose={close} title="Contact">
        {saveError && (
          <Alert tone="error" className="mb-4">
            {saveError}
          </Alert>
        )}
        {isAdmin ? (
          <div className="flex flex-col gap-4">
            <Input
              label="Contact email"
              type="email"
              value={contactEmail}
              onChange={(event) => setContactEmail(event.target.value)}
            />
            <PhoneInput label="Contact phone" value={contactPhone} onChange={setContactPhone} />
            <Button
              loading={saving}
              disabled={Boolean(contactPhone) && !isPhone(contactPhone)}
              onClick={() => {
                const fields = {}
                if (contactEmail.trim()) fields.contact_email = contactEmail.trim()
                if (contactPhone.trim()) fields.contact_phone = contactPhone.trim()
                save(fields)
              }}
            >
              Save changes
            </Button>
          </div>
        ) : (
          <div className="flex flex-col gap-4">
            <div>
              <p className="label-caps">Email</p>
              <p className="mt-1 text-sm text-ink">{data?.email || '—'}</p>
            </div>
            <PhoneInput value={phone} onChange={setPhone} />
            {!isPhone(phone) && phone && <p className="text-xs text-danger-600">{PHONE_MESSAGE}</p>}
            <Button
              loading={saving}
              disabled={!isPhone(phone)}
              onClick={() => save({ phone: phone.trim() })}
            >
              Save phone number
            </Button>
            <p className="text-xs text-faint">
              The administration keeps your name, ID and email address up to date.
            </p>
          </div>
        )}
      </Sheet>
      <Sheet open={sheet === 'theme'} onClose={close} title="Appearance">
        <Segmented
          label="Appearance"
          options={THEME_CHOICES}
          value={theme.choice}
          onChange={theme.select}
        />
        <p className="mt-3 text-xs text-muted">
          System follows the light or dark setting of your phone.
        </p>
      </Sheet>
      <Sheet open={sheet === 'password'} onClose={close} title="Change password">
        <form onSubmit={changePassword} className="flex flex-col gap-4">
          {passwordProblem && <Alert tone="error">{passwordProblem}</Alert>}
          <PasswordInput
            label="Current password"
            value={currentPassword}
            onChange={(event) => setCurrentPassword(event.target.value)}
            autoComplete="current-password"
            required
          />
          <PasswordInput
            label="New password"
            value={newPassword}
            onChange={(event) => setNewPassword(event.target.value)}
            autoComplete="new-password"
            hint="At least 8 characters"
            required
          />
          <PasswordInput
            label="Confirm new password"
            value={confirmPassword}
            onChange={(event) => setConfirmPassword(event.target.value)}
            autoComplete="new-password"
            required
          />
          <p className="text-xs text-faint">
            Changing your password signs you out everywhere, so you log in again afterwards.
          </p>
          <Button type="submit" loading={changing}>
            Change password
          </Button>
        </form>
      </Sheet>
    </>
  )
}

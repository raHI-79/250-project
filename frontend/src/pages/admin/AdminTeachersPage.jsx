import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { GraduationCap, Mail, Plus } from 'lucide-react'
import {
  Alert,
  Avatar,
  Button,
  Card,
  DataTable,
  EmptyState,
  Input,
  Modal,
  PageHeader,
  Select,
  SkeletonTable,
  StatusBadge,
  StatusTabs,
  useConfirm,
  useToast,
} from '../../components/ui'
import DeleteAccountModal from '../../components/admin/DeleteAccountModal'
import useAccountActions from '../../components/admin/useAccountActions'
import PhoneInput from '../../components/forms/PhoneInput'
import PhotoPicker from '../../components/forms/PhotoPicker'
import api from '../../lib/api'
import { useDebounced, useResource } from '../../lib/hooks'
import { paths } from '../../lib/paths'
import { BLOOD_GROUPS, PEOPLE_STATUS_TABS, canRestore } from '../../lib/options'
import {
  PHONE_MESSAGE,
  appendIf,
  birthDateBounds,
  birthDateError,
  emailDomainError,
  isPhone,
} from '../../lib/account'
import { formatDate } from '../../lib/time'

const HEADINGS = ['Name', 'User ID', 'Designation', 'Email', 'Phone', 'Status', '']

const NOT_ACTIVATED = 'Not activated yet'

const EMPTY = {
  name: '',
  designation: '',
  email: '',
  phone: '',
  date_of_birth: '',
  blood_group: '',
  office_room: '',
  joining_date: '',
}

const statusLabel = (status) => {
  if (status === 'invited') return 'Invited'
  if (status === 'active') return 'Active'
  if (status === 'removed') return 'Removed'
  if (status === 'rejected') return 'Rejected'
  if (status === 'pending') return 'Pending'
  return status
}

const accountOf = (item) => ({
  id: item.id,
  name: item.name,
  identifier: item.identifier,
  email: item.email,
  status: item.status,
})

export default function AdminTeachersPage() {
  const toast = useToast()
  const confirm = useConfirm()
  const bounds = birthDateBounds()
  const [search, setSearch] = useState('')
  const [status, setStatus] = useState('')
  const [inviteOpen, setInviteOpen] = useState(false)
  const [editing, setEditing] = useState(null)
  const [form, setForm] = useState(EMPTY)
  const [photo, setPhoto] = useState(null)
  const [formError, setFormError] = useState('')
  const [saving, setSaving] = useState(false)
  const [busy, setBusy] = useState(0)
  const [inviteLink, setInviteLink] = useState('')
  const settled = useDebounced(search)
  const query = {}
  if (settled.trim()) query.search = settled.trim()
  if (status) query.status = status
  const { data, loading, error, reload } = useResource('/api/admin/teachers', query)
  const { busy: actionBusy, deleting, setDeleting, restore, deletePermanently } = useAccountActions(reload)
  useEffect(() => {
    if (!editing) return
    setForm({
      name: editing.name || '',
      designation: editing.designation || '',
      email: editing.email || '',
      phone: editing.phone || '',
      date_of_birth: editing.date_of_birth || '',
      blood_group: editing.blood_group || '',
      office_room: editing.office_room || '',
      joining_date: editing.joining_date || '',
    })
  }, [editing])
  const set = (key) => (event) => setForm((current) => ({ ...current, [key]: event.target.value }))
  const closeInvite = () => {
    setInviteOpen(false)
    setFormError('')
    setForm(EMPTY)
    setPhoto(null)
  }
  const closeEdit = () => {
    setEditing(null)
    setFormError('')
    setForm(EMPTY)
  }
  const checkShared = () => {
    const email = emailDomainError(form.email)
    if (email) return email
    if (!form.name.trim()) return 'Name is required.'
    if (!form.designation.trim()) return 'Designation is required.'
    const birth = birthDateError(form.date_of_birth)
    if (birth) return birth
    if (!BLOOD_GROUPS.includes(form.blood_group)) return 'Please choose a blood group.'
    if (!form.office_room.trim()) return 'Office room is required.'
    return ''
  }
  const invite = async (event) => {
    event.preventDefault()
    setFormError('')
    const problem = checkShared() || (isPhone(form.phone) ? '' : PHONE_MESSAGE)
    if (problem) {
      setFormError(problem)
      return
    }
    const body = new FormData()
    body.append('name', form.name.trim())
    body.append('designation', form.designation.trim())
    body.append('email', form.email.trim().toLowerCase())
    body.append('phone', form.phone.trim())
    body.append('date_of_birth', form.date_of_birth)
    body.append('blood_group', form.blood_group)
    body.append('office_room', form.office_room.trim())
    appendIf(body, 'joining_date', form.joining_date)
    if (photo) body.append('photo', photo)
    setSaving(true)
    try {
      const result = await api.post('/api/admin/teachers', body)
      toast.success('Invitation sent')
      setInviteLink(result?.invite_link || '')
      closeInvite()
      reload()
    } catch (failure) {
      setFormError(failure.message)
    } finally {
      setSaving(false)
    }
  }
  const saveEdit = async (event) => {
    event.preventDefault()
    setFormError('')
    const problem = checkShared()
    if (problem) {
      setFormError(problem)
      return
    }
    const body = {
      name: form.name.trim(),
      designation: form.designation.trim(),
      email: form.email.trim().toLowerCase(),
      date_of_birth: form.date_of_birth,
      blood_group: form.blood_group,
      office_room: form.office_room.trim(),
    }
    if (form.joining_date) body.joining_date = form.joining_date
    setSaving(true)
    try {
      await api.patch(`/api/admin/teachers/${editing.id}`, body)
      toast.success('Teacher updated')
      closeEdit()
      reload()
    } catch (failure) {
      setFormError(failure.message)
    } finally {
      setSaving(false)
    }
  }
  const resend = async (item) => {
    setBusy(item.id)
    try {
      const result = await api.post(`/api/admin/teachers/${item.id}/resend-invite`)
      toast.success('Invitation sent again')
      setInviteLink(result?.invite_link || '')
      reload()
    } catch (failure) {
      toast.error(failure.message)
    } finally {
      setBusy(0)
    }
  }
  const deactivate = async (item) => {
    const ok = await confirm({
      title: 'Deactivate this teacher?',
      message: `${item.name} (${item.identifier || item.email}) loses access immediately.`,
      confirmLabel: 'Deactivate',
    })
    if (!ok) return
    setBusy(item.id)
    try {
      await api.post(`/api/admin/teachers/${item.id}/deactivate`)
      toast.success('Teacher deactivated')
      reload()
    } catch (failure) {
      toast.error(failure.message)
    } finally {
      setBusy(0)
    }
  }
  const items = data?.items || []
  const fields = (
    <>
      <Input label="Name" value={form.name} onChange={set('name')} required />
      <div className="grid gap-4 sm:grid-cols-2">
        <Input
          label="Designation"
          value={form.designation}
          onChange={set('designation')}
          placeholder="Assistant Professor"
          required
        />
        <Input
          label="Official email"
          type="email"
          value={form.email}
          onChange={set('email')}
          hint="Any working address the teacher reads"
          required
        />
        <Input
          label="Date of birth"
          type="date"
          value={form.date_of_birth}
          onChange={set('date_of_birth')}
          min={bounds.min}
          max={bounds.max}
          hint="The teacher confirms this when activating the account"
          required
        />
        <Select
          label="Blood group"
          value={form.blood_group}
          onChange={set('blood_group')}
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
          label="Office room"
          value={form.office_room}
          onChange={set('office_room')}
          placeholder="AB-401"
          required
        />
        <Input
          label="Joining date"
          type="date"
          value={form.joining_date}
          onChange={set('joining_date')}
          hint="Optional"
        />
      </div>
    </>
  )
  return (
    <>
      <PageHeader
        eyebrow="Faculty records" title="Teachers"
        subtitle="Teachers are invited and set their own password"
        actions={
          <Button onClick={() => setInviteOpen(true)}>
            <Plus size={15} />
            Invite teacher
          </Button>
        }
      />
      {inviteLink && (
        <Alert tone="info" title="Invitation link" className="mb-5">
          <p>No mail server is configured, so share this link with the teacher yourself:</p>
          <p className="mt-1 break-all font-medium">{inviteLink}</p>
        </Alert>
      )}
      <Card hover={false} className="mb-5">
        <Input
          label="Search"
          value={search}
          onChange={(event) => setSearch(event.target.value)}
          placeholder="Name or User ID"
        />
        <StatusTabs tabs={PEOPLE_STATUS_TABS} value={status} onChange={setStatus} className="mt-4" />
      </Card>
      {error && (
        <Alert tone="error" className="mb-5">
          {error}
        </Alert>
      )}
      {loading ? (
        <SkeletonTable columns={7} />
      ) : items.length === 0 ? (
        <EmptyState
          icon={GraduationCap}
          title="No teachers"
          text="No teacher matches these filters."
        />
      ) : (
        <DataTable className="animate-fade-in">
          <thead>
            <tr className="border-b border-line bg-wash">
              {HEADINGS.map((heading, index) => (
                <th key={heading || index} className="px-4 py-3 font-semibold text-ink-soft">
                  {heading}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {items.map((item) => (
              <tr key={item.id} className="border-b border-line last:border-b-0">
                <td className="px-4 py-3">
                  <span className="flex items-center gap-2.5">
                    <Avatar src={item.photo_url} name={item.name} size="sm" />
                    <Link
                      to={paths.person(item.user_id || item.id)}
                      className="font-medium text-ink hover:text-brand-600 hover:underline"
                    >
                      {item.name}
                    </Link>
                  </span>
                </td>
                <td className="px-4 py-3 text-ink-soft">
                  {item.identifier || <span className="text-faint">{NOT_ACTIVATED}</span>}
                </td>
                <td className="px-4 py-3 text-ink-soft">{item.designation || '—'}</td>
                <td className="px-4 py-3 text-ink-soft">{item.email}</td>
                <td className="px-4 py-3 text-ink-soft">{item.phone}</td>
                <td className="px-4 py-3">
                  <StatusBadge
                    status={item.status === 'invited' ? 'pending' : item.status}
                    label={statusLabel(item.status)}
                  />
                </td>
                <td className="px-4 py-3">
                  <div className="flex flex-wrap justify-end gap-2">
                    <Button size="sm" variant="secondary" onClick={() => setEditing(item)}>
                      Edit
                    </Button>
                    {item.status === 'invited' && (
                      <Button
                        size="sm"
                        variant="secondary"
                        loading={busy === item.id}
                        onClick={() => resend(item)}
                      >
                        <Mail size={14} />
                        Resend invite
                      </Button>
                    )}
                    {canRestore(item.status) ? (
                      <>
                        <Button
                          size="sm"
                          variant="secondary"
                          loading={actionBusy === item.id}
                          onClick={() => restore(accountOf(item))}
                        >
                          Restore
                        </Button>
                        <Button
                          size="sm"
                          variant="danger"
                          onClick={() => setDeleting(accountOf(item))}
                        >
                          Delete permanently
                        </Button>
                      </>
                    ) : (
                      <Button
                        size="sm"
                        variant="danger"
                        loading={busy === item.id}
                        onClick={() => deactivate(item)}
                      >
                        Deactivate
                      </Button>
                    )}
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </DataTable>
      )}
      <Modal
        open={inviteOpen}
        onClose={closeInvite}
        title="Invite a teacher"
        className="max-w-2xl"
        footer={
          <>
            <Button variant="secondary" onClick={closeInvite}>
              Cancel
            </Button>
            <Button form="invite-teacher" type="submit" loading={saving}>
              Send the invitation
            </Button>
          </>
        }
      >
        <form id="invite-teacher" onSubmit={invite} className="flex flex-col gap-4">
          {formError && <Alert tone="error">{formError}</Alert>}
          <Alert tone="info">
            The teacher picks their own User ID and password from the invitation email, which stays
            valid for 72 hours.
          </Alert>
          {fields}
          <PhoneInput
            value={form.phone}
            onChange={(next) => setForm((current) => ({ ...current, phone: next }))}
            required
          />
          <PhotoPicker
            file={photo}
            onChange={setPhoto}
            label="Profile photo"
            hint="Optional, JPG, PNG or WEBP, at most 2 MB, at least 200 by 200 pixels"
          />
        </form>
      </Modal>
      <Modal
        open={Boolean(editing)}
        onClose={closeEdit}
        title="Edit teacher"
        className="max-w-2xl"
        footer={
          <>
            <Button variant="secondary" onClick={closeEdit}>
              Cancel
            </Button>
            <Button form="edit-teacher" type="submit" loading={saving}>
              Save changes
            </Button>
          </>
        }
      >
        <form id="edit-teacher" onSubmit={saveEdit} className="flex flex-col gap-4">
          {formError && <Alert tone="error">{formError}</Alert>}
          <Alert tone="info">
            The teacher keeps their own phone number, office room and bio up to date from their
            profile.
          </Alert>
          {fields}
          {editing?.joining_date && (
            <p className="text-xs text-faint">
              Joined on {formatDate(editing.joining_date)}
            </p>
          )}
        </form>
      </Modal>
      <DeleteAccountModal
        account={deleting}
        busy={actionBusy === deleting?.id}
        onClose={() => setDeleting(null)}
        onConfirmed={deletePermanently}
      />
    </>
  )
}

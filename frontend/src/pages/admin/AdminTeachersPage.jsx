import { useState } from 'react'
import { GraduationCap, Plus } from 'lucide-react'
import {
  Alert,
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
  useConfirm,
  useToast,
} from '../../components/ui'
import api from '../../lib/api'
import { useDebounced, useResource } from '../../lib/hooks'
import { ACCOUNT_STATUSES } from '../../lib/options'

const EMPTY = {
  name: '',
  teacher_id: '',
  email: '',
  phone: '',
  password: '',
  confirm_password: '',
}

const HEADINGS = ['Name', 'Teacher ID', 'Email', 'Phone', 'Status', '']

export default function AdminTeachersPage() {
  const toast = useToast()
  const confirm = useConfirm()
  const [search, setSearch] = useState('')
  const [status, setStatus] = useState('')
  const [open, setOpen] = useState(false)
  const [form, setForm] = useState(EMPTY)
  const [formError, setFormError] = useState('')
  const [saving, setSaving] = useState(false)
  const [busy, setBusy] = useState(0)
  const settled = useDebounced(search)
  const query = {}
  if (settled.trim()) query.search = settled.trim()
  if (status) query.status = status
  const { data, loading, error, reload } = useResource('/api/admin/teachers', query)
  const set = (key) => (event) => setForm((current) => ({ ...current, [key]: event.target.value }))
  const close = () => {
    setOpen(false)
    setFormError('')
    setForm(EMPTY)
  }
  const save = async (event) => {
    event.preventDefault()
    setFormError('')
    if (form.password !== form.confirm_password) {
      setFormError('Password and the confirmation do not match.')
      return
    }
    setSaving(true)
    try {
      await api.post('/api/admin/teachers', form)
      toast.success('Teacher account created')
      close()
      reload()
    } catch (failure) {
      setFormError(failure.message)
    } finally {
      setSaving(false)
    }
  }
  const deactivate = async (item) => {
    const ok = await confirm({
      title: 'Deactivate this teacher?',
      message: `${item.name} (${item.teacher_id}) loses access immediately.`,
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
  return (
    <>
      <PageHeader
        title="Teachers"
        subtitle="Teacher accounts"
        actions={
          <Button onClick={() => setOpen(true)}>
            <Plus size={15} />
            Add teacher
          </Button>
        }
      />
      <Card hover={false} className="mb-5">
        <div className="grid gap-4 sm:grid-cols-2">
          <Input
            label="Search"
            value={search}
            onChange={(event) => setSearch(event.target.value)}
            placeholder="Teacher name"
          />
          <Select label="Status" value={status} onChange={(event) => setStatus(event.target.value)}>
            <option value="">Every status</option>
            {ACCOUNT_STATUSES.map((entry) => (
              <option key={entry} value={entry}>
                {entry[0].toUpperCase() + entry.slice(1)}
              </option>
            ))}
          </Select>
        </div>
      </Card>
      {error && <Alert tone="error" className="mb-5">{error}</Alert>}
      {loading ? (
        <SkeletonTable columns={6} />
      ) : items.length === 0 ? (
        <EmptyState
          icon={GraduationCap}
          title="No teachers"
          text="No teacher matches these filters."
        />
      ) : (
        <DataTable className="animate-fade-in">
          <thead>
            <tr className="border-b border-brand-100 bg-brand-50/60">
              {HEADINGS.map((heading, index) => (
                <th key={heading || index} className="px-4 py-3 font-semibold text-brand-700">
                  {heading}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {items.map((item) => (
              <tr key={item.id} className="border-b border-brand-50 last:border-b-0">
                <td className="px-4 py-3 font-medium text-brand-900">{item.name}</td>
                <td className="px-4 py-3 text-brand-700">{item.teacher_id}</td>
                <td className="px-4 py-3 text-brand-700">{item.email}</td>
                <td className="px-4 py-3 text-brand-700">{item.phone}</td>
                <td className="px-4 py-3">
                  <StatusBadge status={item.status} />
                </td>
                <td className="px-4 py-3 text-right">
                  <Button
                    size="sm"
                    variant="danger"
                    loading={busy === item.id}
                    disabled={item.status === 'removed'}
                    onClick={() => deactivate(item)}
                  >
                    Deactivate
                  </Button>
                </td>
              </tr>
            ))}
          </tbody>
        </DataTable>
      )}
      <Modal
        open={open}
        onClose={close}
        title="Add a teacher"
        footer={
          <>
            <Button variant="secondary" onClick={close}>
              Cancel
            </Button>
            <Button form="teacher-form" type="submit" loading={saving}>
              Create the account
            </Button>
          </>
        }
      >
        <form id="teacher-form" onSubmit={save} className="flex flex-col gap-4">
          {formError && <Alert tone="error">{formError}</Alert>}
          <Input label="Name" value={form.name} onChange={set('name')} required />
          <div className="grid gap-4 sm:grid-cols-2">
            <Input
              label="Teacher ID"
              value={form.teacher_id}
              onChange={set('teacher_id')}
              placeholder="T-001"
              hint="3 to 20 letters, digits or hyphens"
              required
            />
            <Input label="Phone" value={form.phone} onChange={set('phone')} required />
            <Input
              label="Email"
              type="email"
              value={form.email}
              onChange={set('email')}
              className="sm:col-span-2"
              required
            />
            <Input
              label="Password"
              type="password"
              value={form.password}
              onChange={set('password')}
              autoComplete="new-password"
              hint="At least 8 characters"
              required
            />
            <Input
              label="Confirm password"
              type="password"
              value={form.confirm_password}
              onChange={set('confirm_password')}
              autoComplete="new-password"
              required
            />
          </div>
        </form>
      </Modal>
    </>
  )
}

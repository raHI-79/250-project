import { useState } from 'react'
import { Plus, UserCog } from 'lucide-react'
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
import { BLOOD_GROUPS } from '../../lib/options'

const EMPTY = {
  full_name: '',
  reg_no: '',
  batch: '',
  section: '',
  email: '',
  phone: '',
  blood_group: '',
  password: '',
  confirm_password: '',
}

const HEADINGS = ['Name', 'Reg No', 'Batch', 'Section', 'Email', 'Phone', 'Status', '']

export default function CRsPage() {
  const toast = useToast()
  const confirm = useConfirm()
  const [search, setSearch] = useState('')
  const [batch, setBatch] = useState('')
  const [open, setOpen] = useState(false)
  const [form, setForm] = useState(EMPTY)
  const [formError, setFormError] = useState('')
  const [saving, setSaving] = useState(false)
  const [busy, setBusy] = useState(0)
  const settled = useDebounced(search)
  const batches = useResource('/api/lookups/batches')
  const query = {}
  if (settled.trim()) query.search = settled.trim()
  if (batch) query.batch = batch
  const { data, loading, error, reload } = useResource('/api/admin/crs', query)
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
      await api.post('/api/admin/crs', form)
      toast.success('CR account created')
      close()
      reload()
    } catch (failure) {
      setFormError(failure.message)
    } finally {
      setSaving(false)
    }
  }
  const act = async (item, action) => {
    const ok = await confirm(
      action === 'demote'
        ? {
            title: 'Demote this CR?',
            message: `${item.full_name} becomes an ordinary student and has to log in again.`,
            confirmLabel: 'Demote',
          }
        : {
            title: 'Deactivate this account?',
            message: `${item.full_name} (${item.reg_no}) loses access immediately.`,
            confirmLabel: 'Deactivate',
          },
    )
    if (!ok) return
    setBusy(item.id)
    try {
      await api.post(`/api/admin/crs/${item.id}/${action}`)
      toast.success(action === 'demote' ? 'CR demoted to student' : 'Account deactivated')
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
        title="CRs"
        subtitle="Class representative accounts"
        actions={
          <Button onClick={() => setOpen(true)}>
            <Plus size={15} />
            Add CR
          </Button>
        }
      />
      <Card hover={false} className="mb-5">
        <div className="grid gap-4 sm:grid-cols-2">
          <Input
            label="Search"
            value={search}
            onChange={(event) => setSearch(event.target.value)}
            placeholder="Name or registration number"
          />
          <Select label="Batch" value={batch} onChange={(event) => setBatch(event.target.value)}>
            <option value="">Every batch</option>
            {(batches.data?.items || []).map((entry) => (
              <option key={entry} value={entry}>
                {entry}
              </option>
            ))}
          </Select>
        </div>
      </Card>
      {error && <Alert tone="error" className="mb-5">{error}</Alert>}
      {loading ? (
        <SkeletonTable columns={8} />
      ) : items.length === 0 ? (
        <EmptyState icon={UserCog} title="No CRs" text="No class representative matches this." />
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
                <td className="px-4 py-3 font-medium text-brand-900">{item.full_name}</td>
                <td className="px-4 py-3 tabular-nums text-brand-700">{item.reg_no}</td>
                <td className="px-4 py-3 text-brand-700">{item.batch}</td>
                <td className="px-4 py-3 text-brand-700">{item.section}</td>
                <td className="px-4 py-3 text-brand-700">{item.email}</td>
                <td className="px-4 py-3 text-brand-700">{item.phone}</td>
                <td className="px-4 py-3">
                  <StatusBadge status={item.status} />
                </td>
                <td className="px-4 py-3">
                  <div className="flex justify-end gap-2">
                    <Button
                      size="sm"
                      variant="secondary"
                      loading={busy === item.id}
                      onClick={() => act(item, 'demote')}
                    >
                      Demote
                    </Button>
                    <Button
                      size="sm"
                      variant="danger"
                      loading={busy === item.id}
                      disabled={item.status === 'removed'}
                      onClick={() => act(item, 'deactivate')}
                    >
                      Deactivate
                    </Button>
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </DataTable>
      )}
      <Modal
        open={open}
        onClose={close}
        title="Add a CR"
        footer={
          <>
            <Button variant="secondary" onClick={close}>
              Cancel
            </Button>
            <Button form="cr-form" type="submit" loading={saving}>
              Create the account
            </Button>
          </>
        }
      >
        <form id="cr-form" onSubmit={save} className="flex flex-col gap-4">
          {formError && <Alert tone="error">{formError}</Alert>}
          <Input label="Full name" value={form.full_name} onChange={set('full_name')} required />
          <div className="grid gap-4 sm:grid-cols-2">
            <Input
              label="Registration number"
              value={form.reg_no}
              onChange={set('reg_no')}
              required
            />
            <Input
              label="Batch"
              value={form.batch}
              onChange={set('batch')}
              placeholder="CSE-23"
              required
            />
            <Input
              label="Section"
              value={form.section}
              onChange={set('section')}
              maxLength={1}
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
              label="Email"
              type="email"
              value={form.email}
              onChange={set('email')}
              required
            />
            <Input label="Phone" value={form.phone} onChange={set('phone')} required />
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

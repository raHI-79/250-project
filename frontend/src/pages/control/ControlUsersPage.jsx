import { useCallback, useEffect, useState } from 'react'
import { Download, Users } from 'lucide-react'
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
import { useResource } from '../../lib/hooks'
import { activityLabel } from '../../lib/options'
import { formatDateTime } from '../../lib/time'

const HEADINGS = ['', 'Account', 'Role', 'Status', 'Last sign-in', '']

const ROLES = [
  { value: 'student', label: 'Students' },
  { value: 'cr', label: 'CRs' },
  { value: 'teacher', label: 'Teachers' },
  { value: 'admin', label: 'Office admins' },
]

const STATUSES = ['active', 'pending', 'invited', 'rejected', 'removed', 'unverified']

const PAGE_SIZE = 50

const statusTone = (status) => {
  if (status === 'active') return 'active'
  if (status === 'pending' || status === 'invited' || status === 'unverified') return 'pending'
  return 'removed'
}

const when = (value) => (value ? formatDateTime(value) : 'Never')

function PersonSheet({ userId, onClose, onChanged }) {
  const toast = useToast()
  const confirm = useConfirm()
  const { data, loading, error, reload } = useResource(
    userId ? `/api/super-admin/users/${userId}` : null,
  )
  const [busy, setBusy] = useState('')
  const person = data?.person
  const act = async (key, call, question) => {
    if (question) {
      const ok = await confirm({ title: question, message: person?.name, confirmLabel: 'Do it' })
      if (!ok) return
    }
    setBusy(key)
    try {
      const result = await call()
      toast(result.message)
      await reload()
      await onChanged()
    } catch (failure) {
      toast(failure.message, 'error')
    } finally {
      setBusy('')
    }
  }
  return (
    <Modal open={Boolean(userId)} onClose={onClose} title={person?.name || 'Account'}>
      {error && <Alert tone="error">{error}</Alert>}
      {loading || !person ? (
        <SkeletonTable columns={2} />
      ) : (
        <div className="flex flex-col gap-4">
          <dl className="grid grid-cols-2 gap-3 text-sm">
            <div>
              <dt className="text-xs text-muted">Identifier</dt>
              <dd className="font-semibold text-ink">{person.identifier || '—'}</dd>
            </div>
            <div>
              <dt className="text-xs text-muted">Role</dt>
              <dd className="font-semibold text-ink">{activityLabel(person.role)}</dd>
            </div>
            <div>
              <dt className="text-xs text-muted">Status</dt>
              <dd>
                <StatusBadge status={statusTone(person.status)} label={person.status} />
              </dd>
            </div>
            <div>
              <dt className="text-xs text-muted">Created</dt>
              <dd className="text-ink-soft">{when(person.created_at)}</dd>
            </div>
            <div>
              <dt className="text-xs text-muted">Last sign-in</dt>
              <dd className="text-ink-soft">{when(person.last_sign_in)}</dd>
            </div>
            <div>
              <dt className="text-xs text-muted">Sessions</dt>
              <dd className="text-ink-soft">{data.sessions?.length || 0}</dd>
            </div>
            <div>
              <dt className="text-xs text-muted">Attendance phone</dt>
              <dd className="text-ink-soft">{data.device ? 'Bound' : 'Not bound'}</dd>
            </div>
            <div>
              <dt className="text-xs text-muted">Two-step</dt>
              <dd className="text-ink-soft">{data.two_step ? 'On' : 'Off'}</dd>
            </div>
          </dl>
          <div className="flex flex-wrap gap-2">
            {person.status === 'removed' ? (
              <Button
                size="sm"
                variant="secondary"
                loading={busy === 'restore'}
                onClick={() =>
                  act('restore', () => api.post(`/api/super-admin/users/${person.id}/restore`))
                }
              >
                Restore
              </Button>
            ) : (
              <Button
                size="sm"
                variant="danger"
                loading={busy === 'suspend'}
                onClick={() =>
                  act(
                    'suspend',
                    () => api.post(`/api/super-admin/users/${person.id}/suspend`),
                    'Suspend this account?',
                  )
                }
              >
                Suspend
              </Button>
            )}
            <Button
              size="sm"
              variant="secondary"
              loading={busy === 'sign-out'}
              onClick={() =>
                act(
                  'sign-out',
                  () => api.post(`/api/super-admin/users/${person.id}/sign-out`),
                  'Sign this account out everywhere?',
                )
              }
            >
              Sign out everywhere
            </Button>
            <Button
              size="sm"
              variant="secondary"
              loading={busy === 'phone'}
              onClick={() =>
                act(
                  'phone',
                  () => api.post(`/api/super-admin/users/${person.id}/reset-phone`),
                  'Reset the attendance phone?',
                )
              }
            >
              Reset attendance phone
            </Button>
            {(person.role === 'student' || person.role === 'cr') && (
              <Button
                size="sm"
                variant="secondary"
                loading={busy === 'role'}
                onClick={() =>
                  act(
                    'role',
                    () =>
                      api.put(`/api/super-admin/users/${person.id}/role`, {
                        role: person.role === 'cr' ? 'student' : 'cr',
                      }),
                    person.role === 'cr' ? 'Make this student an ordinary student?' : 'Make this student a CR?',
                  )
                }
              >
                {person.role === 'cr' ? 'Remove CR' : 'Make CR'}
              </Button>
            )}
          </div>
          <div>
            <h3 className="font-display mb-2 text-sm font-bold text-ink">Recent activity</h3>
            {(data.activity || []).length === 0 ? (
              <p className="text-sm text-muted">Nothing recorded yet.</p>
            ) : (
              <ul className="flex flex-col gap-1 text-sm">
                {data.activity.map((entry) => (
                  <li key={entry.id} className="flex justify-between gap-3">
                    <span className="wrap-long text-ink-soft">{activityLabel(entry.action)}</span>
                    <span className="shrink-0 text-xs text-muted">
                      {formatDateTime(entry.created_at)}
                    </span>
                  </li>
                ))}
              </ul>
            )}
          </div>
        </div>
      )}
    </Modal>
  )
}

export default function ControlUsersPage() {
  const toast = useToast()
  const confirm = useConfirm()
  const [search, setSearch] = useState('')
  const [role, setRole] = useState('')
  const [status, setStatus] = useState('')
  const [batch, setBatch] = useState('')
  const [items, setItems] = useState([])
  const [total, setTotal] = useState(0)
  const [offset, setOffset] = useState(0)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [chosen, setChosen] = useState([])
  const [open, setOpen] = useState(null)
  const [busy, setBusy] = useState(false)
  const query = { search, role, status, batch, limit: PAGE_SIZE, offset }
  const load = useCallback(async () => {
    setLoading(true)
    try {
      const data = await api.get('/api/super-admin/users', {
        query: { search, role, status, batch, limit: PAGE_SIZE, offset },
      })
      setItems(data.items || [])
      setTotal(data.total || 0)
      setError('')
    } catch (failure) {
      setError(failure.message)
    } finally {
      setLoading(false)
    }
  }, [search, role, status, batch, offset])
  useEffect(() => {
    load()
  }, [load])
  const toggle = (id) =>
    setChosen((current) =>
      current.includes(id) ? current.filter((entry) => entry !== id) : [...current, id],
    )
  const bulk = async (action) => {
    const ok = await confirm({
      title: action === 'suspend' ? 'Suspend the selected accounts?' : 'Sign the selected accounts out?',
      message: `${chosen.length} accounts`,
      confirmLabel: 'Do it',
    })
    if (!ok) return
    setBusy(true)
    try {
      const data = await api.post('/api/super-admin/users/bulk', { ids: chosen, action })
      toast(data.message)
      setChosen([])
      await load()
    } catch (failure) {
      toast(failure.message, 'error')
    } finally {
      setBusy(false)
    }
  }
  const exportCSV = async () => {
    try {
      const text = await api.get('/api/super-admin/users/export', { query })
      const blob = new Blob([typeof text === 'string' ? text : JSON.stringify(text)], {
        type: 'text/csv;charset=utf-8',
      })
      const url = URL.createObjectURL(blob)
      const link = document.createElement('a')
      link.href = url
      link.download = 'users.csv'
      link.click()
      URL.revokeObjectURL(url)
    } catch (failure) {
      toast(failure.message, 'error')
    }
  }
  return (
    <>
      <PageHeader
        eyebrow="Control room"
        title="Users"
        subtitle={`Every account in the portal · ${total} match this filter`}
        actions={
          <Button variant="secondary" onClick={exportCSV}>
            <Download size={16} aria-hidden="true" />
            Export CSV
          </Button>
        }
      />
      <div className="mb-5 grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
        <Input
          label="Search"
          value={search}
          onChange={(event) => {
            setOffset(0)
            setSearch(event.target.value)
          }}
        />
        <Select
          label="Role"
          value={role}
          onChange={(event) => {
            setOffset(0)
            setRole(event.target.value)
          }}
        >
          <option value="">Every role</option>
          {ROLES.map((entry) => (
            <option key={entry.value} value={entry.value}>
              {entry.label}
            </option>
          ))}
        </Select>
        <Select
          label="Status"
          value={status}
          onChange={(event) => {
            setOffset(0)
            setStatus(event.target.value)
          }}
        >
          <option value="">Every status</option>
          {STATUSES.map((entry) => (
            <option key={entry} value={entry}>
              {entry}
            </option>
          ))}
        </Select>
        <Input
          label="Batch"
          value={batch}
          onChange={(event) => {
            setOffset(0)
            setBatch(event.target.value)
          }}
        />
      </div>
      {error && (
        <Alert tone="error" className="mb-5">
          {error}
        </Alert>
      )}
      {chosen.length > 0 && (
        <Card className="mb-4 flex flex-wrap items-center justify-between gap-3" hover={false}>
          <p className="text-sm font-semibold text-ink">{chosen.length} selected</p>
          <div className="flex gap-2">
            <Button size="sm" variant="secondary" disabled={busy} onClick={() => bulk('sign_out')}>
              Sign out
            </Button>
            <Button size="sm" variant="danger" disabled={busy} onClick={() => bulk('suspend')}>
              Suspend
            </Button>
            <Button size="sm" variant="secondary" onClick={() => setChosen([])}>
              Clear
            </Button>
          </div>
        </Card>
      )}
      {loading ? (
        <SkeletonTable columns={6} />
      ) : items.length === 0 ? (
        <EmptyState icon={Users} title="No accounts found" text="Try another search or filter." />
      ) : (
        <DataTable>
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
                  <input
                    type="checkbox"
                    aria-label={`Select ${item.name}`}
                    checked={chosen.includes(item.id)}
                    onChange={() => toggle(item.id)}
                    className="size-4 accent-brand-600"
                  />
                </td>
                <td className="px-4 py-3">
                  <p className="font-semibold text-ink">{item.name}</p>
                  <p className="text-xs text-muted">
                    {item.identifier || item.email}
                    {item.batch ? ` · ${item.batch} ${item.section}` : ''}
                  </p>
                </td>
                <td className="px-4 py-3 text-ink-soft">{activityLabel(item.role)}</td>
                <td className="px-4 py-3">
                  <StatusBadge status={statusTone(item.status)} label={item.status} />
                </td>
                <td className="px-4 py-3 whitespace-nowrap text-ink-soft">
                  {when(item.last_sign_in)}
                </td>
                <td className="px-4 py-3 text-right">
                  <Button size="sm" variant="secondary" onClick={() => setOpen(item.id)}>
                    Open
                  </Button>
                </td>
              </tr>
            ))}
          </tbody>
        </DataTable>
      )}
      {total > PAGE_SIZE && (
        <div className="mt-4 flex items-center justify-between gap-3">
          <Button
            variant="secondary"
            disabled={offset === 0}
            onClick={() => setOffset(Math.max(0, offset - PAGE_SIZE))}
          >
            Previous
          </Button>
          <p className="text-sm text-muted">
            {offset + 1}–{Math.min(offset + PAGE_SIZE, total)} of {total}
          </p>
          <Button
            variant="secondary"
            disabled={offset + PAGE_SIZE >= total}
            onClick={() => setOffset(offset + PAGE_SIZE)}
          >
            Next
          </Button>
        </div>
      )}
      <PersonSheet userId={open} onClose={() => setOpen(null)} onChanged={load} />
    </>
  )
}

import { useState } from 'react'
import { Mail, ShieldCheck } from 'lucide-react'
import {
  Alert,
  Button,
  Card,
  CardHeader,
  DataTable,
  EmptyState,
  Input,
  Modal,
  PageHeader,
  SkeletonTable,
  StatusBadge,
  useToast,
} from '../../components/ui'
import PermissionPicker from '../../components/control/PermissionPicker'
import api from '../../lib/api'
import { permissionLabel } from '../../lib/permissions'
import { useResource } from '../../lib/hooks'
import { formatDateTime } from '../../lib/time'

const TEACHER_HEADINGS = ['Teacher', 'Office access', 'Last sign-in', '']

const statusTone = (status) => {
  if (status === 'active') return 'active'
  if (status === 'invited') return 'pending'
  return 'removed'
}

const when = (value) => (value ? formatDateTime(value) : 'Never')

export default function ControlAdminsPage() {
  const toast = useToast()
  const admins = useResource('/api/super-admin/admins')
  const [search, setSearch] = useState('')
  const teachers = useResource('/api/super-admin/teachers', { search })
  const [email, setEmail] = useState('')
  const [inviting, setInviting] = useState(false)
  const [error, setError] = useState('')
  const [removing, setRemoving] = useState(null)
  const [editing, setEditing] = useState(null)
  const [confirmName, setConfirmName] = useState('')
  const [busy, setBusy] = useState(false)
  const items = admins.data?.items || []
  const staff = teachers.data?.items || []
  const invite = async (event) => {
    event.preventDefault()
    setError('')
    setInviting(true)
    try {
      const data = await api.post('/api/super-admin/admins/invite', { email: email.trim() })
      toast.success(data.message)
      setEmail('')
      await admins.reload()
    } catch (failure) {
      setError(failure.message)
    } finally {
      setInviting(false)
    }
  }
  const act = async (call) => {
    setBusy(true)
    try {
      const data = await call()
      toast.success(data.message)
      await admins.reload()
      await teachers.reload()
    } catch (failure) {
      toast.error(failure.message)
    } finally {
      setBusy(false)
    }
  }
  const remove = async () => {
    const target = removing
    setRemoving(null)
    const name = confirmName.trim()
    setConfirmName('')
    await act(() => api.delete(`/api/super-admin/admins/${target.id}`, { body: { confirm: name } }))
  }
  return (
    <>
      <PageHeader
        eyebrow="Control room"
        title="Office admins"
        subtitle="Invite office admins, pause them and lend any teacher the office pages."
      />
      <Card className="mb-5">
        <CardHeader title="Invite an office admin" subtitle="The link works for 72 hours." />
        <form onSubmit={invite} className="mt-4 flex flex-col gap-3 sm:flex-row sm:items-start">
          <Input
            label="Email address"
            type="email"
            value={email}
            onChange={(event) => setEmail(event.target.value)}
            className="flex-1"
            required
          />
          <Button type="submit" loading={inviting} className="sm:w-40">
            Send invite
          </Button>
        </form>
        {error && (
          <Alert tone="error" className="mt-3">
            {error}
          </Alert>
        )}
        <p className="mt-3 text-sm text-muted">
          The invitee picks their own username and password.
        </p>
      </Card>
      {admins.error && (
        <Alert tone="error" className="mb-5">
          {admins.error}
        </Alert>
      )}
      {admins.loading ? (
        <SkeletonTable columns={3} />
      ) : items.length === 0 ? (
        <EmptyState icon={ShieldCheck} title="No office admins yet" text="Invite one above." />
      ) : (
        <ul className="mb-6 flex flex-col gap-3">
          {items.map((item) => (
            <li key={item.id} className="surface-card flex flex-col gap-3 p-4">
              <div className="flex flex-wrap items-start justify-between gap-x-4 gap-y-2">
                <div className="min-w-0 flex-1">
                  <p className="wrap-long font-semibold text-ink">
                    {item.username || item.name || item.email}
                  </p>
                  <p className="wrap-long text-xs text-muted">
                    {item.role === 'super_admin'
                      ? 'Super admin'
                      : item.source === 'teacher'
                        ? 'Teacher with office access'
                        : 'Office admin'}
                    {item.email ? ` · ${item.email}` : ''}
                  </p>
                  <p className="mt-1 text-xs text-muted">
                    Last sign-in {when(item.last_sign_in)} · {item.action_count} actions
                  </p>
                </div>
                <div className="flex shrink-0 items-center gap-2">
                  <StatusBadge status={statusTone(item.status)} label={item.status} />
                </div>
              </div>
              <div className="min-w-0">
                {item.role === 'super_admin' ? (
                  <span className="text-xs text-muted">May do everything</span>
                ) : (
                  <button
                    type="button"
                    onClick={() => setEditing(item)}
                    className="wrap-long text-left text-xs font-semibold text-brand-600 hover:underline"
                  >
                    {item.permissions?.length
                      ? item.permissions.map(permissionLabel).join(', ')
                      : 'No permissions yet'}
                  </button>
                )}
              </div>
              <div className="flex flex-wrap gap-2 border-t border-line pt-3">
                {item.status === 'invited' && (
                  <>
                    <Button
                      size="sm"
                      variant="secondary"
                      disabled={busy}
                      onClick={() =>
                        act(() => api.post(`/api/super-admin/admins/${item.id}/resend-invite`))
                      }
                    >
                      Resend
                    </Button>
                    <Button
                      size="sm"
                      variant="secondary"
                      disabled={busy}
                      onClick={() => act(() => api.delete(`/api/super-admin/admins/${item.id}/invite`))}
                    >
                      Cancel invite
                    </Button>
                  </>
                )}
                {item.role === 'admin' && item.status === 'active' && (
                  <Button
                    size="sm"
                    variant="secondary"
                    disabled={busy}
                    onClick={() => act(() => api.post(`/api/super-admin/admins/${item.id}/suspend`))}
                  >
                    Suspend
                  </Button>
                )}
                {item.role === 'admin' && item.status === 'removed' && (
                  <Button
                    size="sm"
                    variant="secondary"
                    disabled={busy}
                    onClick={() => act(() => api.post(`/api/super-admin/admins/${item.id}/restore`))}
                  >
                    Restore
                  </Button>
                )}
                {item.role === 'admin' && item.status !== 'invited' && (
                  <Button
                    size="sm"
                    variant="danger"
                    disabled={busy}
                    onClick={() => {
                      setConfirmName('')
                      setRemoving(item)
                    }}
                  >
                    Remove
                  </Button>
                )}
                {item.source === 'teacher' && (
                  <Button
                    size="sm"
                    variant="secondary"
                    disabled={busy}
                    onClick={() =>
                      act(() => api.delete(`/api/super-admin/teachers/${item.id}/office-access`))
                    }
                  >
                    Revoke access
                  </Button>
                )}
              </div>
            </li>
          ))}
        </ul>
      )}
      <Card>
        <CardHeader title="Teachers" subtitle="Lend the office pages to a teacher." />
        <Input
          label="Search teachers"
          value={search}
          onChange={(event) => setSearch(event.target.value)}
          className="mt-4"
        />
        {teachers.loading ? (
          <SkeletonTable columns={4} />
        ) : staff.length === 0 ? (
          <EmptyState icon={Mail} title="No teachers found" text="Try another name." />
        ) : (
          <DataTable className="mt-4">
            <thead>
              <tr className="border-b border-line bg-wash">
                {TEACHER_HEADINGS.map((heading, index) => (
                  <th key={heading || index} className="px-4 py-3 font-semibold text-ink-soft">
                    {heading}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody>
              {staff.map((teacher) => (
                <tr key={teacher.id} className="border-b border-line last:border-b-0">
                  <td className="px-4 py-3">
                    <p className="font-semibold text-ink">{teacher.name}</p>
                    <p className="text-xs text-muted">
                      {teacher.designation}
                      {teacher.email ? ` · ${teacher.email}` : ''}
                    </p>
                  </td>
                  <td className="px-4 py-3">
                    <StatusBadge
                      status={teacher.office_access ? 'active' : 'removed'}
                      label={teacher.office_access ? 'Granted' : 'None'}
                    />
                  </td>
                  <td className="px-4 py-3 whitespace-nowrap text-ink-soft">
                    {when(teacher.last_sign_in)}
                  </td>
                  <td className="px-4 py-3 text-right">
                    <Button
                      size="sm"
                      variant={teacher.office_access ? 'secondary' : 'primary'}
                      disabled={busy}
                      onClick={() =>
                        act(() =>
                          teacher.office_access
                            ? api.delete(`/api/super-admin/teachers/${teacher.id}/office-access`)
                            : api.post(`/api/super-admin/teachers/${teacher.id}/office-access`),
                        )
                      }
                    >
                      {teacher.office_access ? 'Revoke' : 'Grant'}
                    </Button>
                  </td>
                </tr>
              ))}
            </tbody>
          </DataTable>
        )}
      </Card>
      {editing && (
        <PermissionPicker
          admin={editing}
          open={Boolean(editing)}
          onClose={() => setEditing(null)}
          onSave={async (permissions) => {
            const target = editing
            setEditing(null)
            await act(() =>
              api.put(`/api/super-admin/admins/${target.id}/permissions`, { permissions }),
            )
          }}
        />
      )}
      <Modal
        open={Boolean(removing)}
        onClose={() => setRemoving(null)}
        title="Remove this office admin?"
        footer={
          <>
            <Button variant="secondary" onClick={() => setRemoving(null)}>
              Cancel
            </Button>
            <Button
              variant="danger"
              disabled={confirmName.trim().toLowerCase() !== (removing?.username || '').toLowerCase()}
              onClick={remove}
            >
              Remove
            </Button>
          </>
        }
      >
        <p className="text-sm text-ink-soft">
          Type <span className="font-semibold text-ink">{removing?.username}</span> to confirm. The
          account and its office access disappear.
        </p>
        <Input
          label="Username"
          value={confirmName}
          onChange={(event) => setConfirmName(event.target.value)}
          className="mt-4"
        />
      </Modal>
    </>
  )
}

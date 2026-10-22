import { useState } from 'react'
import { Link } from 'react-router-dom'
import { UserCog, UserPlus } from 'lucide-react'
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
  SkeletonLines,
  SkeletonTable,
  StatusBadge,
  StatusTabs,
  useConfirm,
  useToast,
} from '../../components/ui'
import DeleteAccountModal from '../../components/admin/DeleteAccountModal'
import useAccountActions from '../../components/admin/useAccountActions'
import api from '../../lib/api'
import { useDebounced, useResource } from '../../lib/hooks'
import { paths } from '../../lib/paths'
import { PEOPLE_STATUS_TABS, canRestore } from '../../lib/options'

const HEADINGS = ['Name', 'Reg No', 'Batch', 'Section', 'Email', 'Phone', 'Status', '']

const accountOf = (item) => ({
  id: item.id,
  name: item.full_name,
  identifier: item.reg_no,
  email: item.email,
  status: item.status,
})

function PromoteModal({ open, onClose, batches, onPromoted }) {
  const confirm = useConfirm()
  const toast = useToast()
  const [search, setSearch] = useState('')
  const [batch, setBatch] = useState('')
  const [busy, setBusy] = useState(0)
  const [problem, setProblem] = useState('')
  const settled = useDebounced(search)
  const query = { status: 'active', limit: 20 }
  if (settled.trim()) query.search = settled.trim()
  if (batch) query.batch = batch
  const { data, loading, error, reload } = useResource(open ? '/api/admin/students' : null, query)
  const candidates = (data?.items || []).filter((item) => !item.is_cr)
  const promote = async (item) => {
    const ok = await confirm({
      title: 'Make this student a CR?',
      message: `${item.full_name} (${item.reg_no}) becomes the class representative of ${item.batch} section ${item.section} and has to log in again.`,
      confirmLabel: 'Promote',
    })
    if (!ok) return
    setProblem('')
    setBusy(item.user_id || item.id)
    try {
      await api.post('/api/admin/crs/promote', { user_id: item.user_id || item.id })
      toast.success('Class representative added')
      reload()
      onPromoted()
      onClose()
    } catch (failure) {
      setProblem(failure.message)
    } finally {
      setBusy(0)
    }
  }
  return (
    <Modal
      open={open}
      onClose={onClose}
      title="Make CR"
      className="max-w-2xl"
      footer={
        <Button variant="secondary" onClick={onClose}>
          Close
        </Button>
      }
    >
      <div className="flex flex-col gap-4">
        {problem && <Alert tone="error">{problem}</Alert>}
        <div className="grid gap-4 sm:grid-cols-2">
          <Input
            label="Search active students"
            value={search}
            onChange={(event) => setSearch(event.target.value)}
            placeholder="Name or registration number"
          />
          <Select label="Batch" value={batch} onChange={(event) => setBatch(event.target.value)}>
            <option value="">Every batch</option>
            {batches.map((entry) => (
              <option key={entry} value={entry}>
                {entry}
              </option>
            ))}
          </Select>
        </div>
        {error && <Alert tone="error">{error}</Alert>}
        {loading ? (
          <SkeletonLines lines={4} />
        ) : candidates.length === 0 ? (
          <p className="py-4 text-center text-sm text-muted">
            No active student matches this search.
          </p>
        ) : (
          <ul className="flex max-h-80 flex-col gap-1 overflow-y-auto">
            {candidates.map((item) => (
              <li
                key={item.user_id || item.id}
                className="flex items-center justify-between gap-3 rounded-md border border-line px-3 py-2"
              >
                <span className="flex min-w-0 items-center gap-2.5">
                  <Avatar src={item.photo_url} name={item.full_name} size="sm" />
                  <span className="min-w-0 leading-tight">
                    <span className="block truncate text-sm font-medium text-ink">
                      {item.full_name}
                    </span>
                    <span className="block text-xs text-muted">
                      {item.reg_no} · {item.batch} section {item.section}
                    </span>
                  </span>
                </span>
                <Button
                  size="sm"
                  loading={busy === (item.user_id || item.id)}
                  onClick={() => promote(item)}
                >
                  Promote
                </Button>
              </li>
            ))}
          </ul>
        )}
      </div>
    </Modal>
  )
}

export default function CRsPage() {
  const toast = useToast()
  const confirm = useConfirm()
  const [search, setSearch] = useState('')
  const [batch, setBatch] = useState('')
  const [status, setStatus] = useState('')
  const [open, setOpen] = useState(false)
  const [busy, setBusy] = useState(0)
  const settled = useDebounced(search)
  const batches = useResource('/api/lookups/batches')
  const query = {}
  if (settled.trim()) query.search = settled.trim()
  if (batch) query.batch = batch
  if (status) query.status = status
  const { data, loading, error, reload } = useResource('/api/admin/crs', query)
  const { busy: actionBusy, deleting, setDeleting, restore, deletePermanently } = useAccountActions(reload)
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
        eyebrow="Class representatives" title="CRs"
        subtitle="Class representatives are promoted from active students"
        actions={
          <Button onClick={() => setOpen(true)}>
            <UserPlus size={15} />
            Make CR
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
        <StatusTabs tabs={PEOPLE_STATUS_TABS} value={status} onChange={setStatus} className="mt-4" />
      </Card>
      {error && (
        <Alert tone="error" className="mb-5">
          {error}
        </Alert>
      )}
      {loading ? (
        <SkeletonTable columns={8} />
      ) : items.length === 0 ? (
        <EmptyState
          icon={UserCog}
          title="No CRs"
          text="No class representative matches this."
          action={<Button onClick={() => setOpen(true)}>Make CR</Button>}
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
                    <Avatar src={item.photo_url} name={item.full_name} size="sm" />
                    <Link
                      to={paths.person(item.user_id || item.id)}
                      className="font-medium text-ink hover:text-brand-600 hover:underline"
                    >
                      {item.full_name}
                    </Link>
                  </span>
                </td>
                <td className="px-4 py-3 tabular-nums text-ink-soft">{item.reg_no}</td>
                <td className="px-4 py-3 text-ink-soft">{item.batch}</td>
                <td className="px-4 py-3 text-ink-soft">{item.section}</td>
                <td className="px-4 py-3 text-ink-soft">{item.email}</td>
                <td className="px-4 py-3 text-ink-soft">{item.phone}</td>
                <td className="px-4 py-3">
                  <StatusBadge status={item.status} />
                </td>
                <td className="px-4 py-3">
                  <div className="flex flex-wrap justify-end gap-2">
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
                      <>
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
                          onClick={() => act(item, 'deactivate')}
                        >
                          Deactivate
                        </Button>
                      </>
                    )}
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </DataTable>
      )}
      <PromoteModal
        open={open}
        onClose={() => setOpen(false)}
        batches={batches.data?.items || []}
        onPromoted={reload}
      />
      <DeleteAccountModal
        account={deleting}
        busy={actionBusy === deleting?.id}
        onClose={() => setDeleting(null)}
        onConfirmed={deletePermanently}
      />
    </>
  )
}

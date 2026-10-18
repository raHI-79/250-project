import { useState } from 'react'
import { Link } from 'react-router-dom'
import { Users } from 'lucide-react'
import {
  Alert,
  Avatar,
  Button,
  Card,
  DataTable,
  EmptyState,
  Input,
  PageHeader,
  Select,
  SkeletonTable,
  StatusBadge,
  useConfirm,
  useToast,
} from '../../components/ui'
import api from '../../lib/api'
import { useDebounced, useResource } from '../../lib/hooks'
import { paths } from '../../lib/paths'
import { ACCOUNT_STATUSES } from '../../lib/options'

const PAGE_SIZE = 50

const SECTIONS = ['A', 'B', 'C', 'D']

const HEADINGS = ['Name', 'Reg No', 'Batch', 'Section', 'Email', 'Phone', 'Status', '']

export default function AdminStudentsPage() {
  const toast = useToast()
  const confirm = useConfirm()
  const [search, setSearch] = useState('')
  const [batch, setBatch] = useState('')
  const [section, setSection] = useState('')
  const [status, setStatus] = useState('')
  const [page, setPage] = useState(0)
  const [busy, setBusy] = useState(0)
  const settled = useDebounced(search)
  const batches = useResource('/api/lookups/batches')
  const query = { limit: PAGE_SIZE, offset: page * PAGE_SIZE }
  if (settled.trim()) query.search = settled.trim()
  if (batch) query.batch = batch
  if (section) query.section = section
  if (status) query.status = status
  const { data, loading, error, reload } = useResource('/api/admin/students', query)
  const items = data?.items || []
  const total = data?.total || 0
  const firstShown = total === 0 ? 0 : page * PAGE_SIZE + 1
  const lastShown = page * PAGE_SIZE + items.length
  const onFilter = (setter) => (event) => {
    setter(event.target.value)
    setPage(0)
  }
  const remove = async (item) => {
    const ok = await confirm({
      title: 'Remove this student?',
      message: `${item.full_name} (${item.reg_no}) loses access immediately.`,
      confirmLabel: 'Remove',
    })
    if (!ok) return
    setBusy(item.id)
    try {
      await api.post(`/api/admin/students/${item.id}/remove`)
      toast.success('Student removed')
      reload()
    } catch (failure) {
      toast.error(failure.message)
    } finally {
      setBusy(0)
    }
  }
  return (
    <>
      <PageHeader
        title="Students"
        subtitle={total > 0 ? `${total} student${total === 1 ? '' : 's'} match` : 'Every student account'}
      />
      <Card hover={false} className="mb-5">
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
          <Input
            label="Search"
            value={search}
            onChange={(event) => {
              setSearch(event.target.value)
              setPage(0)
            }}
            placeholder="Name or registration number"
          />
          <Select label="Batch" value={batch} onChange={onFilter(setBatch)}>
            <option value="">Every batch</option>
            {(batches.data?.items || []).map((entry) => (
              <option key={entry} value={entry}>
                {entry}
              </option>
            ))}
          </Select>
          <Select label="Section" value={section} onChange={onFilter(setSection)}>
            <option value="">Every section</option>
            {SECTIONS.map((entry) => (
              <option key={entry} value={entry}>
                {entry}
              </option>
            ))}
          </Select>
          <Select label="Status" value={status} onChange={onFilter(setStatus)}>
            <option value="">Active and pending</option>
            {ACCOUNT_STATUSES.map((entry) => (
              <option key={entry} value={entry}>
                {entry[0].toUpperCase() + entry.slice(1)}
              </option>
            ))}
            <option value="all">Every status</option>
          </Select>
        </div>
      </Card>
      {error && <Alert tone="error" className="mb-5">{error}</Alert>}
      {loading ? (
        <SkeletonTable columns={8} />
      ) : items.length === 0 ? (
        <EmptyState
          icon={Users}
          title="No students found"
          text="No account matches these filters."
        />
      ) : (
        <>
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
                  <td className="px-4 py-3 text-brand-900">
                    <span className="flex items-center gap-2.5">
                      <Avatar src={item.photo_url} name={item.full_name} size="sm" />
                      <Link
                        to={paths.person(item.user_id || item.id)}
                        className="font-medium text-brand-900 hover:text-brand-600 hover:underline"
                      >
                        {item.full_name}
                      </Link>
                      {item.is_cr && <StatusBadge status="approved" label="CR" />}
                    </span>
                  </td>
                  <td className="px-4 py-3 tabular-nums text-brand-700">{item.reg_no}</td>
                  <td className="px-4 py-3 text-brand-700">{item.batch}</td>
                  <td className="px-4 py-3 text-brand-700">{item.section}</td>
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
                      onClick={() => remove(item)}
                    >
                      Remove
                    </Button>
                  </td>
                </tr>
              ))}
            </tbody>
          </DataTable>
          <div className="mt-4 flex flex-wrap items-center justify-between gap-3">
            <p className="text-sm text-brand-500">
              Showing {firstShown}–{lastShown} of {total}
            </p>
            <div className="flex gap-2">
              <Button
                variant="secondary"
                size="sm"
                disabled={page === 0}
                onClick={() => setPage((current) => Math.max(current - 1, 0))}
              >
                Previous
              </Button>
              <Button
                variant="secondary"
                size="sm"
                disabled={lastShown >= total}
                onClick={() => setPage((current) => current + 1)}
              >
                Next
              </Button>
            </div>
          </div>
        </>
      )}
    </>
  )
}

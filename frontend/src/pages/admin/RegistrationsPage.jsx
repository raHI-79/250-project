import { useState } from 'react'
import { UserCheck } from 'lucide-react'
import {
  Alert,
  Avatar,
  Button,
  DataTable,
  EmptyState,
  PageHeader,
  SkeletonTable,
  useConfirm,
  useToast,
} from '../../components/ui'
import api from '../../lib/api'
import { useResource } from '../../lib/hooks'
import { formatDate } from '../../lib/time'

const HEADINGS = [
  'Name',
  'Reg No',
  'Batch',
  'Section',
  'Email',
  'Phone',
  'Blood Group',
  'Date of Birth',
  'Registered',
  '',
]

export default function RegistrationsPage() {
  const toast = useToast()
  const confirm = useConfirm()
  const { data, loading, error, reload } = useResource('/api/admin/registrations')
  const [busy, setBusy] = useState(0)
  const [decided, setDecided] = useState([])
  const items = (data?.items || []).filter((item) => !decided.includes(item.id))
  const decide = async (item, action) => {
    if (action === 'reject') {
      const ok = await confirm({
        title: 'Reject this registration?',
        message: `${item.full_name} (${item.reg_no}) will not be able to log in.`,
        confirmLabel: 'Reject',
      })
      if (!ok) return
    }
    setBusy(item.id)
    try {
      await api.post(`/api/admin/registrations/${item.id}/${action}`)
      toast.success(action === 'approve' ? 'Registration approved' : 'Registration rejected')
      setDecided((current) => [...current, item.id])
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
        eyebrow="Student records" title="Registrations"
        subtitle="Student accounts waiting for your approval, oldest first"
      />
      {error && <Alert tone="error" className="mb-5">{error}</Alert>}
      {loading ? (
        <SkeletonTable columns={10} />
      ) : items.length === 0 ? (
        <EmptyState
          icon={UserCheck}
          title="Nothing to review"
          text="Every registration has been decided."
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
                    <span className="font-medium text-ink">{item.full_name}</span>
                  </span>
                </td>
                <td className="px-4 py-3 tabular-nums text-ink-soft">{item.reg_no}</td>
                <td className="px-4 py-3 text-ink-soft">{item.batch}</td>
                <td className="px-4 py-3 text-ink-soft">{item.section}</td>
                <td className="px-4 py-3 text-ink-soft">{item.email}</td>
                <td className="px-4 py-3 text-ink-soft">{item.phone}</td>
                <td className="px-4 py-3 text-ink-soft">{item.blood_group}</td>
                <td className="px-4 py-3 whitespace-nowrap text-ink-soft">
                  {item.date_of_birth ? formatDate(item.date_of_birth) : '—'}
                </td>
                <td className="px-4 py-3 whitespace-nowrap text-ink-soft">
                  {formatDate(item.created_at)}
                </td>
                <td className="px-4 py-3">
                  <div className="flex flex-wrap justify-end gap-2">
                    <Button
                      size="sm"
                      loading={busy === item.id}
                      onClick={() => decide(item, 'approve')}
                    >
                      Approve
                    </Button>
                    <Button
                      size="sm"
                      variant="danger"
                      loading={busy === item.id}
                      onClick={() => decide(item, 'reject')}
                    >
                      Reject
                    </Button>
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </DataTable>
      )}
    </>
  )
}

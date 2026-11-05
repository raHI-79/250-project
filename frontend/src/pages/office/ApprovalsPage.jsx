import { useState } from 'react'
import { UserCheck } from 'lucide-react'
import {
  Alert,
  Button,
  DataTable,
  EmptyState,
  PageHeader,
  SkeletonTable,
  StatusBadge,
  useToast,
} from '../../components/ui'
import api from '../../lib/api'
import { useResource } from '../../lib/hooks'
import { formatDateTime } from '../../lib/time'

const HEADINGS = ['Asked', 'What', 'Who asked', 'Status', '']

const ACTION_LABELS = {
  delete_user: 'Delete an account for good',
  delete_sheet: 'Delete a whole attendance sheet',
  remove_admin: 'Remove an office admin',
}

const tone = (status) => {
  if (status === 'pending') return 'pending'
  if (status === 'approved') return 'active'
  return 'removed'
}

export default function ApprovalsPage({ eyebrow = 'Department office' }) {
  const toast = useToast()
  const { data, loading, error, reload } = useResource('/api/office/approvals')
  const [busy, setBusy] = useState('')
  const items = data?.items || []
  const decide = async (item, approve) => {
    setBusy(`${item.id}-${approve}`)
    try {
      const result = await api.post(
        `/api/office/approvals/${item.id}/${approve ? 'approve' : 'decline'}`,
      )
      toast.success(result.message)
      await reload()
    } catch (failure) {
      toast.error(failure.message)
    } finally {
      setBusy('')
    }
  }
  return (
    <>
      <PageHeader
        eyebrow={eyebrow}
        title="Waiting for a second admin"
        subtitle="The most dangerous actions need another office admin to agree, within 24 hours"
      />
      {error && (
        <Alert tone="error" className="mb-5">
          {error}
        </Alert>
      )}
      {loading ? (
        <SkeletonTable columns={5} />
      ) : items.length === 0 ? (
        <EmptyState
          icon={UserCheck}
          title="Nothing is waiting"
          text="Dangerous actions show up here while they wait for a second admin."
        />
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
                <td className="px-4 py-3 whitespace-nowrap text-ink-soft">
                  {formatDateTime(item.created_at)}
                </td>
                <td className="px-4 py-3">
                  <p className="font-semibold text-ink">
                    {ACTION_LABELS[item.action] || item.action}
                  </p>
                  <p className="wrap-long text-xs text-muted">{item.summary}</p>
                </td>
                <td className="px-4 py-3 text-ink-soft">
                  {item.requester}
                  {item.mine && <span className="ml-2 text-xs text-brand-600">You</span>}
                </td>
                <td className="px-4 py-3">
                  <StatusBadge status={tone(item.status)} label={item.status} />
                </td>
                <td className="px-4 py-3">
                  {item.status === 'pending' && !item.mine && (
                    <div className="flex justify-end gap-2">
                      <Button
                        size="sm"
                        loading={busy === `${item.id}-true`}
                        onClick={() => decide(item, true)}
                      >
                        Approve
                      </Button>
                      <Button
                        size="sm"
                        variant="secondary"
                        loading={busy === `${item.id}-false`}
                        onClick={() => decide(item, false)}
                      >
                        Decline
                      </Button>
                    </div>
                  )}
                  {item.status === 'pending' && item.mine && (
                    <p className="text-right text-xs text-muted">Another admin must agree</p>
                  )}
                </td>
              </tr>
            ))}
          </tbody>
        </DataTable>
      )}
    </>
  )
}

import { ScrollText } from 'lucide-react'
import {
  Alert,
  DataTable,
  EmptyState,
  PageHeader,
  SkeletonTable,
  StatusBadge,
} from '../../components/ui'
import { useResource } from '../../lib/hooks'
import { AUDIT_ACTION_LABELS } from '../../lib/options'
import { formatDateTime } from '../../lib/time'

const HEADINGS = ['When', 'Action', 'Account', 'By']

const tone = (action) => {
  if (action === 'restore') return 'active'
  if (action === 'permanent_delete') return 'rejected'
  return 'removed'
}

export default function ActivityLogPage() {
  const { data, loading, error } = useResource('/api/admin/audit')
  const items = data?.items || []
  return (
    <>
      <PageHeader
        title="Activity log"
        subtitle="The last 100 removals, restores and permanent deletions"
      />
      {error && (
        <Alert tone="error" className="mb-5">
          {error}
        </Alert>
      )}
      {loading ? (
        <SkeletonTable columns={4} />
      ) : items.length === 0 ? (
        <EmptyState
          icon={ScrollText}
          title="Nothing logged yet"
          text="Removing, restoring or deleting an account shows up here."
        />
      ) : (
        <DataTable className="animate-fade-in">
          <thead>
            <tr className="border-b border-brand-100 bg-brand-50/60">
              {HEADINGS.map((heading) => (
                <th key={heading} className="px-4 py-3 font-semibold text-brand-700">
                  {heading}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {items.map((item) => (
              <tr key={item.id} className="border-b border-brand-50 last:border-b-0">
                <td className="px-4 py-3 whitespace-nowrap text-brand-700">
                  {formatDateTime(item.created_at)}
                </td>
                <td className="px-4 py-3">
                  <StatusBadge
                    status={tone(item.action)}
                    label={AUDIT_ACTION_LABELS[item.action] || item.action}
                  />
                </td>
                <td className="px-4 py-3 text-brand-900">{item.target_summary}</td>
                <td className="px-4 py-3 text-brand-700">{item.admin_name}</td>
              </tr>
            ))}
          </tbody>
        </DataTable>
      )}
    </>
  )
}

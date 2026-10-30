import { Smartphone } from 'lucide-react'
import {
  Alert,
  DataTable,
  EmptyState,
  PageHeader,
  SkeletonTable,
  StatusBadge,
} from '../../components/ui'
import { useResource } from '../../lib/hooks'
import { formatDateTime } from '../../lib/time'
import { sectionLabel } from '../../lib/attendance'

const HEADINGS = ['Asked', 'Student', 'Batch', 'Reason', 'Status', 'Decided by', 'Note']

export default function AdminPhoneChangesPage() {
  const { data, loading, error } = useResource('/api/admin/phone-changes')
  const items = data?.items || []
  return (
    <>
      <PageHeader
        eyebrow="Department office"
        title="Phone changes"
        subtitle="Every attendance phone a student asked for and what their teachers decided. The office does not approve these."
      />
      {error && (
        <Alert tone="error" className="mb-5">
          {error}
        </Alert>
      )}
      {loading ? (
        <SkeletonTable columns={7} />
      ) : items.length === 0 ? (
        <EmptyState
          icon={Smartphone}
          title="Nothing recorded yet"
          text="When a student asks to move their attendance to another phone, the request and its answer are kept here."
        />
      ) : (
        <DataTable className="animate-fade-in">
          <thead>
            <tr className="border-b border-line bg-wash">
              {HEADINGS.map((heading) => (
                <th key={heading} className="px-4 py-3 font-semibold text-ink-soft">
                  {heading}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {items.map((entry) => (
              <tr key={entry.id} className="border-b border-line last:border-b-0">
                <td className="px-4 py-3 whitespace-nowrap text-ink-soft">
                  {formatDateTime(entry.created_at)}
                </td>
                <td className="px-4 py-3 font-medium text-ink">
                  {entry.student_name}
                  <span className="ml-2 text-xs text-muted">{entry.reg_no}</span>
                </td>
                <td className="px-4 py-3 whitespace-nowrap text-ink-soft">
                  {entry.batch}
                  {entry.section ? ` · ${sectionLabel(entry.section)}` : ''}
                </td>
                <td className="px-4 py-3 text-ink-soft">{entry.reason}</td>
                <td className="px-4 py-3">
                  <StatusBadge status={entry.status} />
                </td>
                <td className="px-4 py-3 whitespace-nowrap text-ink-soft">
                  {entry.decider_name || '—'}
                  {entry.decided_at && (
                    <span className="block text-xs text-muted">
                      {formatDateTime(entry.decided_at)}
                    </span>
                  )}
                </td>
                <td className="px-4 py-3 text-ink-soft">{entry.decision_note || '—'}</td>
              </tr>
            ))}
          </tbody>
        </DataTable>
      )}
    </>
  )
}

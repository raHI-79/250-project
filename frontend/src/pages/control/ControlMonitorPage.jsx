import { Link, useParams } from 'react-router-dom'
import { ScrollText } from 'lucide-react'
import {
  Alert,
  Card,
  DataTable,
  EmptyState,
  PageHeader,
  SkeletonTable,
  StatusBadge,
} from '../../components/ui'
import { useResource } from '../../lib/hooks'
import { activityLabel } from '../../lib/options'
import { paths } from '../../lib/paths'
import { formatDateTime } from '../../lib/time'

const HEADINGS = ['When', 'Action', 'What', 'Where']

export default function ControlMonitorPage() {
  const { adminId } = useParams()
  const { data, loading, error } = useResource(`/api/super-admin/activity/${adminId}`)
  const items = data?.items || []
  const counts = data?.counts || []
  const who = items[0]?.actor || `Account ${adminId}`
  return (
    <>
      <PageHeader
        eyebrow="Control room"
        title={`Monitoring ${who}`}
        subtitle="Everything this account did, newest first"
      />
      <Link
        to={paths.control.activity()}
        className="mb-5 inline-block text-sm font-semibold text-brand-600 hover:underline"
      >
        Back to the activity log
      </Link>
      {error && (
        <Alert tone="error" className="mb-5">
          {error}
        </Alert>
      )}
      {counts.length > 0 && (
        <div className="mb-5 grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
          {counts.map((count) => (
            <Card key={count.action} hover={false}>
              <p className="text-2xs font-semibold tracking-wide text-muted uppercase">
                {activityLabel(count.action)}
              </p>
              <p className="font-display mt-1 text-section font-bold text-ink">{count.total}</p>
            </Card>
          ))}
        </div>
      )}
      {loading ? (
        <SkeletonTable columns={4} />
      ) : items.length === 0 ? (
        <EmptyState icon={ScrollText} title="Nothing yet" text="This account has done nothing so far." />
      ) : (
        <DataTable>
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
            {items.map((item) => (
              <tr key={item.id} className="border-b border-line last:border-b-0">
                <td className="px-4 py-3 whitespace-nowrap text-ink-soft">
                  {formatDateTime(item.created_at)}
                </td>
                <td className="px-4 py-3">
                  <StatusBadge status="pending" label={activityLabel(item.action)} />
                </td>
                <td className="px-4 py-3 wrap-long text-ink">{item.summary}</td>
                <td className="px-4 py-3 text-ink-soft">
                  {item.client === 'android' ? 'App' : 'Website'}
                </td>
              </tr>
            ))}
          </tbody>
        </DataTable>
      )}
    </>
  )
}

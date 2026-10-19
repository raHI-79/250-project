import { Fragment, useState } from 'react'
import { ClipboardList } from 'lucide-react'
import {
  Alert,
  Card,
  DataTable,
  EmptyState,
  PageHeader,
  Select,
  SkeletonTable,
  StatusBadge,
} from '../../components/ui'
import NoteLine from '../../components/NoteLine'
import { useResource } from '../../lib/hooks'
import { formatDate, formatTimeRange } from '../../lib/time'

const STATUSES = ['pending', 'approved', 'rejected', 'cancelled', 'auto_cancelled']

const TYPE_LABELS = { class: 'Class', exam: 'Exam', extra: 'Extra class' }

const HEADINGS = ['Type', 'Course', 'Teacher', 'Date', 'Time', 'Room', 'Status']

export default function MyRequestsPage() {
  const [status, setStatus] = useState('')
  const { data, loading, error } = useResource(
    '/api/cr/requests',
    status ? { status } : null,
  )
  const items = data?.items || []
  return (
    <>
      <PageHeader title="My Requests" subtitle="What you asked for and where it stands" />
      <Card hover={false} className="mb-5">
        <Select
          label="Status"
          value={status}
          onChange={(event) => setStatus(event.target.value)}
          className="max-w-xs"
        >
          <option value="">Every status</option>
          {STATUSES.map((entry) => (
            <option key={entry} value={entry}>
              {entry === 'auto_cancelled' ? 'Auto cancelled' : entry[0].toUpperCase() + entry.slice(1)}
            </option>
          ))}
        </Select>
      </Card>
      {error && <Alert tone="error" className="mb-5">{error}</Alert>}
      {loading ? (
        <SkeletonTable columns={7} />
      ) : items.length === 0 ? (
        <EmptyState
          icon={ClipboardList}
          title="No requests"
          text="Nothing matches this filter yet."
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
              <Fragment key={item.id}>
                <tr className="border-b border-brand-50 last:border-b-0">
                  <td className="px-4 py-3 text-brand-700">
                    {TYPE_LABELS[item.event_type] || item.event_type}
                  </td>
                  <td className="px-4 py-3 text-brand-900">
                    <span className="font-medium">{item.course_code}</span>
                    {item.course_title && <span className="text-brand-500"> {item.course_title}</span>}
                  </td>
                  <td className="px-4 py-3 text-brand-700">{item.teacher_name}</td>
                  <td className="px-4 py-3 whitespace-nowrap text-brand-700">
                    {formatDate(item.event_date)}
                  </td>
                  <td className="px-4 py-3 whitespace-nowrap tabular-nums text-brand-600">
                    {formatTimeRange(item.start_time, item.end_time)}
                  </td>
                  <td className="px-4 py-3 text-brand-700">{item.room}</td>
                  <td className="px-4 py-3">
                    <StatusBadge status={item.status} />
                  </td>
                </tr>
                {item.note && (
                  <tr className="border-b border-brand-50 last:border-b-0">
                    <td colSpan={HEADINGS.length} className="px-4 pb-3">
                      <NoteLine text={item.note} audience={item.note_audience} />
                    </td>
                  </tr>
                )}
              </Fragment>
            ))}
          </tbody>
        </DataTable>
      )}
    </>
  )
}

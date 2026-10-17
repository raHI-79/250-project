import { ClipboardList } from 'lucide-react'
import { Alert, DataTable, EmptyState, PageHeader, SkeletonTable } from '../../components/ui'
import { useResource } from '../../lib/hooks'
import { formatDate, formatTimeRange } from '../../lib/time'

const HEADINGS = ['Date', 'Time', 'Course', 'Batch', 'Section', 'Room']

export default function ExamsPage() {
  const { data, loading, error } = useResource('/api/students/exams')
  const items = data?.items || []
  return (
    <>
      <PageHeader
        title="Exams"
        subtitle="Approved exams for your batch and section, from today onward"
      />
      {error && <Alert tone="error" className="mb-5">{error}</Alert>}
      {loading ? (
        <SkeletonTable columns={6} />
      ) : items.length === 0 ? (
        <EmptyState
          icon={ClipboardList}
          title="No exams ahead"
          text="Nothing has been approved for your section yet."
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
                  {formatDate(item.event_date)}
                </td>
                <td className="px-4 py-3 whitespace-nowrap tabular-nums text-brand-600">
                  {formatTimeRange(item.start_time, item.end_time)}
                </td>
                <td className="px-4 py-3 text-brand-900">
                  <span className="font-medium">{item.course_code}</span>
                  {item.course_title && (
                    <span className="text-brand-500"> {item.course_title}</span>
                  )}
                </td>
                <td className="px-4 py-3 text-brand-700">{item.batch}</td>
                <td className="px-4 py-3 text-brand-700">
                  {item.section === 'ALL' ? 'Both' : item.section}
                </td>
                <td className="px-4 py-3 text-brand-700">{item.room}</td>
              </tr>
            ))}
          </tbody>
        </DataTable>
      )}
    </>
  )
}

import { BellRing, FileText } from 'lucide-react'
import { Alert, Card, EmptyState, PageHeader, SkeletonCard } from '../../components/ui'
import { useResource } from '../../lib/hooks'
import { formatDateTime } from '../../lib/time'

export default function AdminNoticesPage() {
  const { data, loading, error } = useResource('/api/students/notices/admin')
  const items = data?.items || []
  return (
    <>
      <PageHeader
        title="Admin Notices"
        subtitle="Notices from the department administration"
      />
      {error && <Alert tone="error" className="mb-5">{error}</Alert>}
      {loading ? (
        <SkeletonCard />
      ) : items.length === 0 ? (
        <EmptyState
          icon={BellRing}
          title="No notices"
          text="The administration has not posted anything yet."
        />
      ) : (
        <div className="flex flex-col gap-4">
          {items.map((item) => (
            <Card key={item.id} hover={false} className="animate-fade-in-up">
              <p className="font-display text-base font-semibold text-brand-900">{item.title}</p>
              {item.description && (
                <p className="mt-2 text-sm whitespace-pre-line text-brand-700">
                  {item.description}
                </p>
              )}
              <div className="mt-3 flex flex-wrap items-center gap-x-4 gap-y-2 text-xs text-brand-500">
                <span>{formatDateTime(item.created_at)}</span>
                {item.file_url && (
                  <a
                    href={item.file_url}
                    target="_blank"
                    rel="noreferrer"
                    className="inline-flex items-center gap-1.5 text-sm text-brand-600 hover:underline"
                  >
                    <FileText size={15} />
                    Open the PDF
                  </a>
                )}
              </div>
            </Card>
          ))}
        </div>
      )}
    </>
  )
}

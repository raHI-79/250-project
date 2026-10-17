import { ExternalLink, FolderOpen } from 'lucide-react'
import {
  Alert,
  Card,
  EmptyState,
  PageHeader,
  SkeletonCard,
  StatusBadge,
} from '../../components/ui'
import { useResource } from '../../lib/hooks'
import { formatDateTime } from '../../lib/time'

export default function MaterialsPage() {
  const { data, loading, error } = useResource('/api/students/materials')
  const items = data?.items || []
  return (
    <>
      <PageHeader
        title="Materials"
        subtitle="Shared links for your batch and the ones shared with everyone"
      />
      {error && <Alert tone="error" className="mb-5">{error}</Alert>}
      {loading ? (
        <div className="grid gap-4 sm:grid-cols-2">
          <SkeletonCard />
          <SkeletonCard />
        </div>
      ) : items.length === 0 ? (
        <EmptyState
          icon={FolderOpen}
          title="No materials"
          text="Nothing has been shared with your batch yet."
        />
      ) : (
        <div className="grid gap-4 sm:grid-cols-2">
          {items.map((item) => (
            <Card key={item.id} className="animate-fade-in-up flex flex-col gap-3">
              <div className="flex flex-wrap items-start justify-between gap-2">
                <p className="font-display text-base font-semibold text-brand-900">{item.title}</p>
                <StatusBadge
                  status={item.batch === 'ALL' ? 'approved' : 'unavailable'}
                  label={item.batch === 'ALL' ? 'ALL' : item.batch}
                />
              </div>
              <a
                href={item.link}
                target="_blank"
                rel="noreferrer"
                className="inline-flex items-center gap-1.5 text-sm text-brand-600 hover:underline"
              >
                <ExternalLink size={15} />
                Open the link
              </a>
              <p className="text-xs text-brand-500">
                {item.created_by_name} · {formatDateTime(item.created_at)}
              </p>
            </Card>
          ))}
        </div>
      )}
    </>
  )
}

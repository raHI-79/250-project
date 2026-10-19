import { ExternalLink, FolderOpen } from 'lucide-react'
import { Alert, Card, EmptyState, PageHeader, SkeletonCard } from '../../components/ui'
import { useResource } from '../../lib/hooks'
import { relativeTime } from '../../lib/time'
import cx from '../../lib/cx'

export default function MaterialsPage() {
  const { data, loading, error } = useResource('/api/students/drive-links')
  const items = data?.items || []
  return (
    <>
      <PageHeader
        title="Materials"
        subtitle="One Drive folder per batch, kept by that batch's class representative"
      />
      {error && (
        <Alert tone="error" className="mb-5">
          {error}
        </Alert>
      )}
      {loading ? (
        <div className="grid gap-4 sm:grid-cols-2">
          <SkeletonCard />
          <SkeletonCard />
        </div>
      ) : items.length === 0 ? (
        <EmptyState
          icon={FolderOpen}
          title="No Drive links yet"
          text="No batch has shared a Drive folder so far."
        />
      ) : (
        <div className="grid gap-4 sm:grid-cols-2">
          {items.map((item) => (
            <Card
              key={item.batch}
              className={cx(
                'animate-fade-in-up flex flex-col gap-3',
                item.is_mine && 'border-brand-400 ring-1 ring-brand-200',
              )}
            >
              <div className="flex flex-wrap items-start justify-between gap-2">
                <p className="font-display text-2xl font-semibold text-brand-900">{item.title}</p>
                {item.is_mine && (
                  <span className="rounded-full bg-brand-600 px-2.5 py-0.5 text-xs font-medium text-white">
                    My batch
                  </span>
                )}
              </div>
              <p className="text-xs text-brand-500">Updated {relativeTime(item.updated_at)}</p>
              <a
                href={item.link}
                target="_blank"
                rel="noreferrer"
                className="inline-flex w-fit items-center gap-2 rounded-lg bg-brand-600 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-brand-700"
              >
                <ExternalLink size={16} />
                Open Drive
              </a>
            </Card>
          ))}
        </div>
      )}
    </>
  )
}

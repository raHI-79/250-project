import { ExternalLink, FolderOpen } from 'lucide-react'
import { Alert, Button, Card, EmptyState, PageHeader, SkeletonCard } from '../../components/ui'
import { useResource } from '../../lib/hooks'
import { relativeTime } from '../../lib/time'
import cx from '../../lib/cx'
import { BADGE_MATERIALS, useSectionSeen } from '../../lib/badges'

export default function MaterialsPage() {
  useSectionSeen(BADGE_MATERIALS)
  const { data, loading, error } = useResource('/api/students/drive-links')
  const items = data?.items || []
  return (
    <>
      <PageHeader
        eyebrow="Course materials" title="Materials"
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
          text="No batch has shared a Drive folder so far. Ask your class representative to add the folder your batch uses."
        />
      ) : (
        <div className="grid gap-4 sm:grid-cols-2">
          {items.map((item) => (
            <Card
              key={item.batch}
              className={cx(
                'animate-fade-in-up flex flex-col gap-3',
                item.is_mine && 'border-line-strong ring-1 ring-brand-200',
              )}
            >
              <div className="flex flex-wrap items-start justify-between gap-2">
                <p className="font-display wrap-long text-lg font-bold text-ink">{item.title}</p>
                {item.is_mine && (
                  <span className="rounded-full bg-brand-600 px-2.5 py-0.5 text-xs font-medium text-page">
                    My batch
                  </span>
                )}
              </div>
              <p className="text-xs text-muted">Updated {relativeTime(item.updated_at)}</p>
              <Button
                as="a"
                href={item.link}
                target="_blank"
                rel="noreferrer"
                className="w-fit"
              >
                <ExternalLink size={16} />
                Open Drive
              </Button>
            </Card>
          ))}
        </div>
      )}
    </>
  )
}

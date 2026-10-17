import { useState } from 'react'
import { FileText, Megaphone } from 'lucide-react'
import {
  Alert,
  Button,
  Card,
  EmptyState,
  PageHeader,
  SkeletonCard,
  StatusBadge,
} from '../../components/ui'
import { useResource } from '../../lib/hooks'
import { formatDateTime } from '../../lib/time'

const SCOPES = [
  { value: 'my', label: 'My Batch' },
  { value: 'other', label: 'Other Batches' },
]

export default function CRNoticesPage() {
  const [scope, setScope] = useState('my')
  const { data, loading, error } = useResource('/api/students/notices/cr', { scope })
  const items = data?.items || []
  return (
    <>
      <PageHeader
        title="CR Notices"
        subtitle="Notices posted by the class representatives"
        actions={
          <div className="inline-flex rounded-lg border border-brand-200 bg-surface p-1">
            {SCOPES.map((entry) => (
              <Button
                key={entry.value}
                size="sm"
                variant={scope === entry.value ? 'primary' : 'ghost'}
                onClick={() => setScope(entry.value)}
              >
                {entry.label}
              </Button>
            ))}
          </div>
        }
      />
      {error && <Alert tone="error" className="mb-5">{error}</Alert>}
      {loading ? (
        <SkeletonCard />
      ) : items.length === 0 ? (
        <EmptyState
          icon={Megaphone}
          title="No notices"
          text={
            scope === 'my'
              ? 'Your class representatives have not posted anything yet.'
              : 'No other batch has a notice right now.'
          }
        />
      ) : (
        <div className="flex flex-col gap-4">
          {items.map((item) => (
            <Card key={item.id} hover={false} className="animate-fade-in-up">
              <div className="flex flex-wrap items-center gap-2">
                <p className="font-display text-base font-semibold text-brand-900">{item.title}</p>
                {scope === 'other' && item.batch && (
                  <StatusBadge status="unavailable" label={item.batch} />
                )}
              </div>
              {item.description && (
                <p className="mt-2 text-sm whitespace-pre-line text-brand-700">
                  {item.description}
                </p>
              )}
              <div className="mt-3 flex flex-wrap items-center gap-x-4 gap-y-2 text-xs text-brand-500">
                <span>{item.created_by_name}</span>
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

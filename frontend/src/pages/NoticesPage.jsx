import { useState } from 'react'
import { Megaphone } from 'lucide-react'
import AttachmentView from '../components/AttachmentView'
import {
  Alert,
  Avatar,
  Button,
  Card,
  EmptyState,
  PageHeader,
  SkeletonCard,
  StatusBadge,
} from '../components/ui'
import { useResource } from '../lib/hooks'
import { formatDateTime } from '../lib/time'

const TABS = [
  { value: 'all', label: 'All' },
  { value: 'admin', label: 'Department' },
  { value: 'teacher', label: 'Teachers' },
  { value: 'cr', label: 'CR' },
]

const SCOPES = [
  { value: 'my', label: 'My batch' },
  { value: 'other', label: 'Other batches' },
]

const SOURCE_LABELS = { admin: 'Department', teacher: 'Teacher', cr: 'CR' }

export default function NoticesPage() {
  const [source, setSource] = useState('all')
  const [scope, setScope] = useState('my')
  const { data, loading, error } = useResource('/api/students/notices', {
    source,
    scope: source === 'cr' ? scope : '',
  })
  const items = data?.items || []
  return (
    <>
      <PageHeader
        title="Notices"
        subtitle="Everything the department, your teachers and the class representatives posted"
        actions={
          <div className="inline-flex flex-wrap rounded-lg border border-brand-200 bg-surface p-1">
            {TABS.map((entry) => (
              <Button
                key={entry.value}
                size="sm"
                variant={source === entry.value ? 'primary' : 'ghost'}
                onClick={() => setSource(entry.value)}
              >
                {entry.label}
              </Button>
            ))}
          </div>
        }
      />
      {source === 'cr' && (
        <div className="mb-5 inline-flex rounded-lg border border-brand-200 bg-surface p-1">
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
      )}
      {error && (
        <Alert tone="error" className="mb-5">
          {error}
        </Alert>
      )}
      {loading ? (
        <SkeletonCard />
      ) : items.length === 0 ? (
        <EmptyState
          icon={Megaphone}
          title="No notices"
          text="Nothing has been posted for this filter yet."
        />
      ) : (
        <div className="flex flex-col gap-4">
          {items.map((item) => (
            <Card key={`${item.source}-${item.id}`} hover={false} className="animate-fade-in-up">
              <div className="flex flex-wrap items-center gap-2">
                <p className="font-display text-base font-semibold text-brand-900">{item.title}</p>
                <StatusBadge
                  status={item.source === 'admin' ? 'approved' : 'unavailable'}
                  label={SOURCE_LABELS[item.source] || item.source}
                />
                {item.batch && (
                  <StatusBadge
                    status="unavailable"
                    label={item.batch === 'ALL' ? 'All batches' : item.batch}
                  />
                )}
                {item.section && (
                  <StatusBadge status="unavailable" label={`Section ${item.section}`} />
                )}
                <span className="ml-auto text-xs whitespace-nowrap text-brand-500">
                  {formatDateTime(item.created_at)}
                </span>
              </div>
              {item.description && (
                <p className="mt-2 text-sm whitespace-pre-line text-brand-700">
                  {item.description}
                </p>
              )}
              <AttachmentView
                fileUrl={item.file_url}
                fileName={item.file_name}
                fileMime={item.file_mime}
                fileSize={item.file_size}
                linkUrl={item.link_url}
                className="mt-3"
              />
              {item.created_by_name && (
                <div className="mt-3 flex items-center gap-2 text-xs text-brand-500">
                  <Avatar
                    src={item.posted_by_photo_url}
                    name={item.created_by_name}
                    size="sm"
                  />
                  {item.created_by_name}
                </div>
              )}
            </Card>
          ))}
        </div>
      )}
    </>
  )
}

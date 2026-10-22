import { useState } from 'react'
import { Megaphone } from 'lucide-react'
import AttachmentView from '../components/AttachmentView'
import DateBlock from '../components/DateBlock'
import {
  Alert,
  Avatar,
  Card,
  EmptyState,
  PageHeader,
  SkeletonCard,
  StatusBadge,
  StatusTabs,
} from '../components/ui'
import { useHighlight, useResource } from '../lib/hooks'
import cx from '../lib/cx'
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
  const highlight = useHighlight(true)
  const { data, loading, error } = useResource('/api/students/notices', {
    source,
    scope: source === 'cr' ? scope : '',
  })
  const items = data?.items || []
  return (
    <>
      <PageHeader
        eyebrow="Notice board"
        title="Notices"
        subtitle="Everything the department, your teachers and the class representatives posted"
      />
      <StatusTabs tabs={TABS} value={source} onChange={setSource} className="mb-5" />
      {source === 'cr' && (
        <StatusTabs tabs={SCOPES} value={scope} onChange={setScope} className="mb-5" />
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
            <Card
              key={`${item.source}-${item.id}`}
              ref={highlight.register(item.id)}
              hover={false}
              className={cx(
                'animate-fade-in-up',
                highlight.highlighted(item.id) && 'border-accent-300 bg-accent-50/60',
              )}
            >
              <div className="flex gap-4">
                <DateBlock value={item.created_at} />
                <div className="min-w-0 flex-1">
                  <div className="flex flex-wrap items-center gap-2">
                    <p className="font-display wrap-long text-lg font-bold text-ink">{item.title}</p>
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
                    <span className="ml-auto text-xs whitespace-nowrap text-muted">
                      {formatDateTime(item.created_at)}
                    </span>
                  </div>
                  {item.description && (
                    <p className="wrap-long mt-2 text-sm leading-relaxed whitespace-pre-line text-ink-soft">
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
                    <div className="mt-3 flex items-center gap-2 text-xs text-muted">
                      <Avatar src={item.posted_by_photo_url} name={item.created_by_name} size="xs" />
                      {item.created_by_name}
                    </div>
                  )}
                </div>
              </div>
            </Card>
          ))}
        </div>
      )}
    </>
  )
}

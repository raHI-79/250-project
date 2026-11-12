import { useState } from 'react'
import { Navigate, useSearchParams } from 'react-router-dom'
import { Megaphone } from 'lucide-react'
import NoticeCard from '../components/NoticeCard'
import {
  Alert,
  EmptyState,
  PageHeader,
  SkeletonCard,
  StatusTabs,
  PullToRefresh,
} from '../components/ui'
import { useResource } from '../lib/hooks'
import paths from '../lib/paths'
import { BADGE_NOTICES, useSectionSeen } from '../lib/badges'

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

export default function NoticesPage() {
  useSectionSeen(BADGE_NOTICES)
  const [source, setSource] = useState('all')
  const [scope, setScope] = useState('my')
  const [params] = useSearchParams()
  const wanted = params.get('highlight') || ''
  const { data, loading, error, reload } = useResource('/api/students/notices', {
    source,
    scope: source === 'cr' ? scope : '',
  })
  const items = data?.items || []
  if (/^\d+$/.test(wanted)) return <Navigate to={paths.student.notice(wanted)} replace />
  return (
    <PullToRefresh onRefresh={reload}>
      <PageHeader
        eyebrow="Notice board"
        title="Notices"
        subtitle="Everything the department, your teachers and the class representatives posted"
        tabs={<StatusTabs tabs={TABS} value={source} onChange={setSource} tone="header" />}
      />
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
            <NoticeCard key={`${item.source}-${item.id}`} item={item} />
          ))}
        </div>
      )}
    </PullToRefresh>
  )
}

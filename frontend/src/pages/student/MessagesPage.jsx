import { FileText, Mail } from 'lucide-react'
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

const ROLE_LABELS = { teacher: 'Teacher', cr: 'CR' }

const targetLabel = (item) => {
  if (item.target_type === 'batch') return `To ${item.target_batch}`
  if (item.target_type === 'section') return `To ${item.target_batch} section ${item.target_section}`
  return 'To you'
}

export default function MessagesPage() {
  const { data, loading, error } = useResource('/api/students/messages')
  const items = data?.items || []
  return (
    <>
      <PageHeader
        title="Messages"
        subtitle="Messages from your teachers and class representatives"
      />
      {error && <Alert tone="error" className="mb-5">{error}</Alert>}
      {loading ? (
        <SkeletonCard />
      ) : items.length === 0 ? (
        <EmptyState icon={Mail} title="No messages" text="Your inbox is empty." />
      ) : (
        <div className="flex flex-col gap-4">
          {items.map((item) => (
            <Card key={item.id} hover={false} className="animate-fade-in-up">
              <div className="flex flex-wrap items-center gap-2">
                <p className="font-medium text-brand-900">{item.sender_name}</p>
                <StatusBadge
                  status={item.sender_role === 'teacher' ? 'approved' : 'yours'}
                  label={ROLE_LABELS[item.sender_role] || item.sender_role}
                />
                <span className="text-xs text-brand-400">{targetLabel(item)}</span>
                <span className="ml-auto text-xs text-brand-500">
                  {formatDateTime(item.created_at)}
                </span>
              </div>
              <p className="mt-2 text-sm whitespace-pre-line text-brand-700">{item.content}</p>
              {item.file_url && (
                <a
                  href={item.file_url}
                  target="_blank"
                  rel="noreferrer"
                  className="mt-3 inline-flex items-center gap-1.5 text-sm text-brand-600 hover:underline"
                >
                  <FileText size={15} />
                  Open the PDF
                </a>
              )}
            </Card>
          ))}
        </div>
      )}
    </>
  )
}

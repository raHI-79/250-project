import { useNavigate } from 'react-router-dom'
import { FileText, Inbox, Reply } from 'lucide-react'
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
import { paths } from '../../lib/paths'
import { formatDateTime } from '../../lib/time'

export default function TeacherMessagesPage() {
  const navigate = useNavigate()
  const { data, loading, error } = useResource('/api/cr/messages/inbox')
  const items = data?.items || []
  const reply = (item) => {
    navigate(paths.cr.sendMessage(), { state: { teacherName: item.sender_name } })
  }
  return (
    <>
      <PageHeader title="Teacher Messages" subtitle="Messages the teachers sent you" />
      {error && <Alert tone="error" className="mb-5">{error}</Alert>}
      {loading ? (
        <SkeletonCard />
      ) : items.length === 0 ? (
        <EmptyState icon={Inbox} title="No messages" text="No teacher has written to you yet." />
      ) : (
        <div className="flex flex-col gap-4">
          {items.map((item) => (
            <Card key={item.id} hover={false} className="animate-fade-in-up">
              <div className="flex flex-wrap items-center gap-2">
                <p className="font-medium text-brand-900">{item.sender_name}</p>
                <StatusBadge status="approved" label="Teacher" />
                <span className="ml-auto text-xs text-brand-500">
                  {formatDateTime(item.created_at)}
                </span>
              </div>
              <p className="mt-2 text-sm whitespace-pre-line text-brand-700">{item.content}</p>
              <div className="mt-3 flex flex-wrap items-center gap-4">
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
                <Button variant="secondary" size="sm" onClick={() => reply(item)}>
                  <Reply size={15} />
                  Reply
                </Button>
              </div>
            </Card>
          ))}
        </div>
      )}
    </>
  )
}

import { useState } from 'react'
import { FileText, Inbox, Reply } from 'lucide-react'
import {
  Alert,
  Button,
  Card,
  EmptyState,
  Modal,
  PageHeader,
  SkeletonCard,
  StatusBadge,
  Textarea,
  useToast,
} from '../../components/ui'
import PdfInput from '../../components/forms/PdfInput'
import api from '../../lib/api'
import { useResource } from '../../lib/hooks'
import { formatDateTime } from '../../lib/time'

export default function CRMessagesPage() {
  const toast = useToast()
  const { data, loading, error, reload } = useResource('/api/teacher/messages/inbox')
  const [answering, setAnswering] = useState(null)
  const [content, setContent] = useState('')
  const [file, setFile] = useState(null)
  const [formError, setFormError] = useState('')
  const [sending, setSending] = useState(false)
  const close = () => {
    setAnswering(null)
    setContent('')
    setFile(null)
    setFormError('')
  }
  const send = async (event) => {
    event.preventDefault()
    setFormError('')
    if (!content.trim() && !file) {
      setFormError('Write a reply or attach a PDF.')
      return
    }
    const payload = new FormData()
    payload.set('content', content.trim())
    payload.set('reply_to_id', String(answering.id))
    if (file) payload.set('file', file)
    setSending(true)
    try {
      await api.post('/api/teacher/messages', payload)
      toast.success('Reply sent')
      close()
      reload()
    } catch (failure) {
      setFormError(failure.message)
    } finally {
      setSending(false)
    }
  }
  const items = data?.items || []
  return (
    <>
      <PageHeader title="CR Messages" subtitle="Messages the class representatives sent you" />
      {error && <Alert tone="error" className="mb-5">{error}</Alert>}
      {loading ? (
        <SkeletonCard />
      ) : items.length === 0 ? (
        <EmptyState icon={Inbox} title="No messages" text="No CR has written to you yet." />
      ) : (
        <div className="flex flex-col gap-4">
          {items.map((item) => (
            <Card key={item.id} hover={false} className="animate-fade-in-up">
              <div className="flex flex-wrap items-center gap-2">
                <p className="font-medium text-brand-900">{item.sender_name}</p>
                <StatusBadge status="yours" label="CR" />
                {item.sender_batch && (
                  <span className="text-xs text-brand-500">
                    {item.sender_batch}
                    {item.sender_section ? ` · section ${item.sender_section}` : ''}
                  </span>
                )}
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
                <Button variant="secondary" size="sm" onClick={() => setAnswering(item)}>
                  <Reply size={15} />
                  Reply
                </Button>
              </div>
            </Card>
          ))}
        </div>
      )}
      <Modal
        open={Boolean(answering)}
        onClose={close}
        title={answering ? `Reply to ${answering.sender_name}` : 'Reply'}
        footer={
          <>
            <Button variant="secondary" onClick={close}>
              Cancel
            </Button>
            <Button form="reply-form" type="submit" loading={sending}>
              Send the reply
            </Button>
          </>
        }
      >
        <form id="reply-form" onSubmit={send} className="flex flex-col gap-4">
          {formError && <Alert tone="error">{formError}</Alert>}
          <Textarea
            label="Reply"
            value={content}
            onChange={(event) => setContent(event.target.value)}
            rows={5}
          />
          <PdfInput file={file} onChange={setFile} />
        </form>
      </Modal>
    </>
  )
}

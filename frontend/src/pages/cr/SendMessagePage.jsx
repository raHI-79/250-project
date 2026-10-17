import { useEffect, useMemo, useState } from 'react'
import { useLocation } from 'react-router-dom'
import { FileText, Send } from 'lucide-react'
import {
  Alert,
  Button,
  Card,
  EmptyState,
  Field,
  PageHeader,
  Select,
  SkeletonCard,
  StatusBadge,
  Textarea,
  useToast,
} from '../../components/ui'
import PdfInput from '../../components/forms/PdfInput'
import { useAuth } from '../../context/auth'
import api from '../../lib/api'
import { useResource } from '../../lib/hooks'
import { formatDateTime } from '../../lib/time'

const TARGETS = [
  { value: 'own_batch', label: 'My batch' },
  { value: 'batch', label: 'Another batch' },
  { value: 'teacher', label: 'A teacher' },
]

const targetLabel = (item) => {
  if (item.target_type === 'batch') return `Batch ${item.target_batch}`
  if (item.target_type === 'section') return `${item.target_batch} section ${item.target_section}`
  if (item.target_type === 'teacher') return 'Teacher'
  return 'Direct'
}

export default function SendMessagePage() {
  const toast = useToast()
  const { user } = useAuth()
  const location = useLocation()
  const replyTo = location.state?.teacherName || ''
  const [target, setTarget] = useState(replyTo ? 'teacher' : 'own_batch')
  const [batch, setBatch] = useState('')
  const [teacherUserId, setTeacherUserId] = useState('')
  const [content, setContent] = useState('')
  const [file, setFile] = useState(null)
  const [error, setError] = useState('')
  const [sending, setSending] = useState(false)
  const batches = useResource('/api/lookups/batches')
  const teachers = useResource('/api/lookups/teachers')
  const sent = useResource('/api/cr/messages/sent')
  const teacherOptions = useMemo(() => teachers.data?.items || [], [teachers.data])
  useEffect(() => {
    if (!replyTo || teacherUserId) return
    const match = teacherOptions.find(
      (entry) => entry.name.toLowerCase() === replyTo.toLowerCase(),
    )
    if (match) setTeacherUserId(String(match.user_id))
  }, [replyTo, teacherUserId, teacherOptions])
  const otherBatches = (batches.data?.items || []).filter((entry) => entry !== user?.batch)
  const onSubmit = async (event) => {
    event.preventDefault()
    setError('')
    if (!content.trim() && !file) {
      setError('Write a message or attach a PDF.')
      return
    }
    const payload = new FormData()
    payload.set('target', target)
    payload.set('content', content.trim())
    if (target === 'batch') payload.set('batch', batch)
    if (target === 'teacher') payload.set('teacher_user_id', teacherUserId)
    if (file) payload.set('file', file)
    setSending(true)
    try {
      await api.post('/api/cr/messages', payload)
      toast.success('Message sent')
      setContent('')
      setFile(null)
      sent.reload()
    } catch (failure) {
      setError(failure.message)
    } finally {
      setSending(false)
    }
  }
  const items = sent.data?.items || []
  return (
    <>
      <PageHeader
        title="Send Message"
        subtitle="Write to your batch, another batch or one of the teachers"
      />
      <Card hover={false} className="mb-6">
        <form onSubmit={onSubmit} className="flex flex-col gap-4">
          {error && <Alert tone="error">{error}</Alert>}
          {replyTo && <Alert tone="info">Replying to {replyTo}.</Alert>}
          <Field label="Send to">
            <div className="flex flex-wrap gap-4">
              {TARGETS.map((entry) => (
                <label
                  key={entry.value}
                  className="inline-flex items-center gap-2 text-sm text-brand-800"
                >
                  <input
                    type="radio"
                    name="target"
                    value={entry.value}
                    checked={target === entry.value}
                    onChange={() => setTarget(entry.value)}
                    className="size-4 accent-brand-600"
                  />
                  {entry.label}
                </label>
              ))}
            </div>
          </Field>
          {target === 'batch' && (
            <Select
              label="Batch"
              value={batch}
              onChange={(event) => setBatch(event.target.value)}
              className="animate-fade-in"
              required
            >
              <option value="">Choose a batch</option>
              {otherBatches.map((entry) => (
                <option key={entry} value={entry}>
                  {entry}
                </option>
              ))}
            </Select>
          )}
          {target === 'teacher' && (
            <Select
              label="Teacher"
              value={teacherUserId}
              onChange={(event) => setTeacherUserId(event.target.value)}
              className="animate-fade-in"
              required
            >
              <option value="">Choose a teacher</option>
              {teacherOptions.map((entry) => (
                <option key={entry.user_id} value={String(entry.user_id)}>
                  {entry.name}
                </option>
              ))}
            </Select>
          )}
          <Textarea
            label="Message"
            value={content}
            onChange={(event) => setContent(event.target.value)}
            rows={5}
            placeholder="What do you want them to know?"
          />
          <PdfInput file={file} onChange={setFile} />
          <Button type="submit" loading={sending} className="w-full sm:w-auto sm:self-end">
            <Send size={15} />
            Send the message
          </Button>
        </form>
      </Card>
      <p className="font-display mb-3 text-lg font-semibold text-brand-800">Sent messages</p>
      {sent.error && <Alert tone="error" className="mb-5">{sent.error}</Alert>}
      {sent.loading ? (
        <SkeletonCard />
      ) : items.length === 0 ? (
        <EmptyState icon={Send} title="Nothing sent yet" text="Your sent messages appear here." />
      ) : (
        <div className="flex flex-col gap-4">
          {items.map((item) => (
            <Card key={item.id} hover={false} className="animate-fade-in-up">
              <div className="flex flex-wrap items-center gap-2">
                <StatusBadge status="unavailable" label={targetLabel(item)} />
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

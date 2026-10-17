import { useState } from 'react'
import { FileText, Megaphone } from 'lucide-react'
import {
  Alert,
  Button,
  Card,
  EmptyState,
  Field,
  Input,
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
  { value: 'own', label: 'My batch' },
  { value: 'batch', label: 'Another batch' },
]

export default function CRNoticesPage() {
  const toast = useToast()
  const { user } = useAuth()
  const [title, setTitle] = useState('')
  const [description, setDescription] = useState('')
  const [target, setTarget] = useState('own')
  const [batch, setBatch] = useState('')
  const [file, setFile] = useState(null)
  const [error, setError] = useState('')
  const [sending, setSending] = useState(false)
  const batches = useResource('/api/lookups/batches')
  const notices = useResource('/api/cr/notices')
  const otherBatches = (batches.data?.items || []).filter((entry) => entry !== user?.batch)
  const onSubmit = async (event) => {
    event.preventDefault()
    setError('')
    const payload = new FormData()
    payload.set('title', title.trim())
    payload.set('description', description.trim())
    payload.set('target', target === 'own' ? 'own' : batch)
    if (file) payload.set('file', file)
    setSending(true)
    try {
      await api.post('/api/cr/notices', payload)
      toast.success('Notice posted')
      setTitle('')
      setDescription('')
      setFile(null)
      notices.reload()
    } catch (failure) {
      setError(failure.message)
    } finally {
      setSending(false)
    }
  }
  const items = notices.data?.items || []
  return (
    <>
      <PageHeader title="Notices" subtitle="Post a notice for your batch or another batch" />
      <Card hover={false} className="mb-6">
        <form onSubmit={onSubmit} className="flex flex-col gap-4">
          {error && <Alert tone="error">{error}</Alert>}
          <Input
            label="Title"
            value={title}
            onChange={(event) => setTitle(event.target.value)}
            required
          />
          <Textarea
            label="Description"
            value={description}
            onChange={(event) => setDescription(event.target.value)}
            rows={4}
          />
          <Field label="Target">
            <div className="flex flex-wrap gap-4">
              {TARGETS.map((entry) => (
                <label
                  key={entry.value}
                  className="inline-flex items-center gap-2 text-sm text-brand-800"
                >
                  <input
                    type="radio"
                    name="notice-target"
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
          <PdfInput file={file} onChange={setFile} />
          <Button type="submit" loading={sending} className="w-full sm:w-auto sm:self-end">
            Post the notice
          </Button>
        </form>
      </Card>
      <p className="font-display mb-3 text-lg font-semibold text-brand-800">My notices</p>
      {notices.error && <Alert tone="error" className="mb-5">{notices.error}</Alert>}
      {notices.loading ? (
        <SkeletonCard />
      ) : items.length === 0 ? (
        <EmptyState
          icon={Megaphone}
          title="No notices yet"
          text="The notices you post appear here."
        />
      ) : (
        <div className="flex flex-col gap-4">
          {items.map((item) => (
            <Card key={item.id} hover={false} className="animate-fade-in-up">
              <div className="flex flex-wrap items-center gap-2">
                <p className="font-display text-base font-semibold text-brand-900">{item.title}</p>
                <StatusBadge status="unavailable" label={item.batch} />
                <span className="ml-auto text-xs text-brand-500">
                  {formatDateTime(item.created_at)}
                </span>
              </div>
              {item.description && (
                <p className="mt-2 text-sm whitespace-pre-line text-brand-700">
                  {item.description}
                </p>
              )}
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

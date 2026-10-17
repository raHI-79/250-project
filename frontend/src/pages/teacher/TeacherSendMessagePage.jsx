import { useState } from 'react'
import { FileText, Send, X } from 'lucide-react'
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
import api from '../../lib/api'
import { useDebounced, useResource } from '../../lib/hooks'
import { formatDateTime } from '../../lib/time'

const TARGETS = [
  { value: 'cr', label: 'A CR' },
  { value: 'batch', label: 'A batch' },
  { value: 'student', label: 'One student' },
]

const targetLabel = (item) => {
  if (item.target_type === 'batch') return `Batch ${item.target_batch}`
  if (item.target_type === 'section') return `${item.target_batch} section ${item.target_section}`
  return 'Direct'
}

export default function TeacherSendMessagePage() {
  const toast = useToast()
  const [target, setTarget] = useState('cr')
  const [crBatch, setCrBatch] = useState('')
  const [crUserId, setCrUserId] = useState('')
  const [batch, setBatch] = useState('')
  const [section, setSection] = useState('')
  const [search, setSearch] = useState('')
  const [student, setStudent] = useState(null)
  const [content, setContent] = useState('')
  const [file, setFile] = useState(null)
  const [error, setError] = useState('')
  const [sending, setSending] = useState(false)
  const settled = useDebounced(search)
  const batches = useResource('/api/lookups/batches')
  const crs = useResource('/api/teacher/lookups/crs', crBatch ? { batch: crBatch } : null)
  const sections = useResource('/api/lookups/sections', batch ? { batch } : null)
  const found = useResource(
    target === 'student' && settled.trim() ? '/api/teacher/lookups/students' : null,
    { search: settled.trim() },
  )
  const sent = useResource('/api/teacher/messages/sent')
  const onSubmit = async (event) => {
    event.preventDefault()
    setError('')
    if (!content.trim() && !file) {
      setError('Write a message or attach a PDF.')
      return
    }
    if (target === 'student' && !student) {
      setError('Search for a student and pick one.')
      return
    }
    const payload = new FormData()
    payload.set('target', target)
    payload.set('content', content.trim())
    if (target === 'cr') payload.set('target_user_id', crUserId)
    if (target === 'student') payload.set('target_user_id', String(student.user_id))
    if (target === 'batch') {
      payload.set('batch', batch)
      if (section) payload.set('section', section)
    }
    if (file) payload.set('file', file)
    setSending(true)
    try {
      await api.post('/api/teacher/messages', payload)
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
  const results = found.data?.items || []
  return (
    <>
      <PageHeader
        title="Send Message"
        subtitle="Write to a class representative, a batch or one student"
      />
      <Card hover={false} className="mb-6">
        <form onSubmit={onSubmit} className="flex flex-col gap-4">
          {error && <Alert tone="error">{error}</Alert>}
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
          {target === 'cr' && (
            <div className="animate-fade-in grid gap-4 sm:grid-cols-2">
              <Select
                label="Batch"
                value={crBatch}
                onChange={(event) => {
                  setCrBatch(event.target.value)
                  setCrUserId('')
                }}
              >
                <option value="">Every batch</option>
                {(batches.data?.items || []).map((entry) => (
                  <option key={entry} value={entry}>
                    {entry}
                  </option>
                ))}
              </Select>
              <Select
                label="CR"
                value={crUserId}
                onChange={(event) => setCrUserId(event.target.value)}
                required
              >
                <option value="">Choose a CR</option>
                {(crs.data?.items || []).map((entry) => (
                  <option key={entry.user_id} value={String(entry.user_id)}>
                    {entry.full_name} · {entry.batch} {entry.section}
                  </option>
                ))}
              </Select>
            </div>
          )}
          {target === 'batch' && (
            <div className="animate-fade-in grid gap-4 sm:grid-cols-2">
              <Select
                label="Batch"
                value={batch}
                onChange={(event) => {
                  setBatch(event.target.value)
                  setSection('')
                }}
                required
              >
                <option value="">Choose a batch</option>
                {(batches.data?.items || []).map((entry) => (
                  <option key={entry} value={entry}>
                    {entry}
                  </option>
                ))}
              </Select>
              <Select
                label="Section"
                value={section}
                onChange={(event) => setSection(event.target.value)}
              >
                <option value="">Whole batch</option>
                {(sections.data?.items || []).map((entry) => (
                  <option key={entry} value={entry}>
                    Section {entry}
                  </option>
                ))}
              </Select>
            </div>
          )}
          {target === 'student' && (
            <div className="animate-fade-in flex flex-col gap-3">
              <Input
                label="Find a student"
                value={search}
                onChange={(event) => setSearch(event.target.value)}
                placeholder="Name or registration number"
              />
              {student ? (
                <div className="flex items-center justify-between gap-3 rounded-lg border border-brand-200 bg-white px-3 py-2 text-sm">
                  <span className="text-brand-800">
                    {student.full_name} · {student.reg_no} · {student.batch} {student.section}
                  </span>
                  <Button variant="ghost" size="sm" onClick={() => setStudent(null)}>
                    <X size={15} />
                  </Button>
                </div>
              ) : (
                settled.trim() && (
                  <div className="rounded-lg border border-brand-100">
                    {found.loading ? (
                      <p className="px-3 py-2 text-sm text-brand-500">Searching…</p>
                    ) : results.length === 0 ? (
                      <p className="px-3 py-2 text-sm text-brand-500">No student matches.</p>
                    ) : (
                      <ul className="flex max-h-56 flex-col overflow-y-auto">
                        {results.map((entry) => (
                          <li key={entry.user_id}>
                            <button
                              type="button"
                              onClick={() => setStudent(entry)}
                              className="w-full border-b border-brand-50 px-3 py-2 text-left text-sm text-brand-800 transition-colors last:border-b-0 hover:bg-brand-50"
                            >
                              {entry.full_name} · {entry.reg_no} · {entry.batch} {entry.section}
                            </button>
                          </li>
                        ))}
                      </ul>
                    )}
                  </div>
                )
              )}
            </div>
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

import { useState } from 'react'
import { Megaphone } from 'lucide-react'
import {
  Alert,
  Button,
  Card,
  EmptyState,
  Input,
  PageHeader,
  Select,
  SkeletonCard,
  StatusBadge,
  Textarea,
  useConfirm,
  useToast,
} from '../../components/ui'
import AttachmentView from '../../components/AttachmentView'
import FileInput from '../../components/forms/FileInput'
import LinkInput from '../../components/forms/LinkInput'
import api from '../../lib/api'
import { appendIf } from '../../lib/account'
import { linkError } from '../../lib/files'
import { useHighlight, useResource } from '../../lib/hooks'
import cx from '../../lib/cx'
import { formatDateTime } from '../../lib/time'

export default function TeacherNoticesPage() {
  const toast = useToast()
  const confirm = useConfirm()
  const [title, setTitle] = useState('')
  const [description, setDescription] = useState('')
  const [batch, setBatch] = useState('')
  const [section, setSection] = useState('')
  const [link, setLink] = useState('')
  const [file, setFile] = useState(null)
  const [error, setError] = useState('')
  const [saving, setSaving] = useState(false)
  const [busy, setBusy] = useState(0)
  const batches = useResource('/api/lookups/batches')
  const sections = useResource(batch ? '/api/lookups/sections' : null, { batch })
  const notices = useResource('/api/teacher/notices')
  const highlight = useHighlight(Boolean(notices.data))
  const publish = async (event) => {
    event.preventDefault()
    const problem = linkError(link)
    if (problem) {
      setError(problem)
      return
    }
    setError('')
    const payload = new FormData()
    payload.set('title', title.trim())
    payload.set('batch', batch)
    appendIf(payload, 'description', description.trim())
    appendIf(payload, 'section', section)
    appendIf(payload, 'link', link.trim())
    if (file) payload.set('file', file)
    setSaving(true)
    try {
      await api.post('/api/teacher/notices', payload)
      toast.success('Notice posted')
      setTitle('')
      setDescription('')
      setLink('')
      setFile(null)
      notices.reload()
    } catch (failure) {
      setError(failure.message)
    } finally {
      setSaving(false)
    }
  }
  const remove = async (item) => {
    const ok = await confirm({
      title: 'Delete this notice?',
      message: `"${item.title}" and its attachment will be removed for everyone.`,
      confirmLabel: 'Delete',
    })
    if (!ok) return
    setBusy(item.id)
    try {
      await api.delete(`/api/teacher/notices/${item.id}`)
      toast.success('Notice deleted')
      notices.reload()
    } catch (failure) {
      toast.error(failure.message)
    } finally {
      setBusy(0)
    }
  }
  const items = notices.data?.items || []
  return (
    <>
      <PageHeader title="Notices" subtitle="Post a notice for a batch or for one section" />
      <Card hover={false} className="mb-6">
        <form onSubmit={publish} className="flex flex-col gap-4">
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
          <div className="grid gap-4 sm:grid-cols-2">
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
              disabled={!batch}
            >
              <option value="">Whole batch</option>
              {(sections.data?.items || []).map((entry) => (
                <option key={entry} value={entry}>
                  Section {entry}
                </option>
              ))}
            </Select>
          </div>
          <LinkInput value={link} onChange={setLink} />
          <FileInput file={file} onChange={setFile} />
          <Button type="submit" loading={saving} className="w-full sm:w-auto sm:self-end">
            Post the notice
          </Button>
        </form>
      </Card>
      <p className="font-display mb-3 text-lg font-semibold text-brand-800">My notices</p>
      {notices.error && (
        <Alert tone="error" className="mb-5">
          {notices.error}
        </Alert>
      )}
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
            <Card
              key={item.id}
              ref={highlight.register(item.id)}
              hover={false}
              className={cx(
                'animate-fade-in-up',
                highlight.highlighted(item.id) && 'border-accent-300 bg-accent-50/60',
              )}
            >
              <div className="flex flex-wrap items-center gap-2">
                <p className="font-display text-base font-semibold text-brand-900">{item.title}</p>
                <StatusBadge status="unavailable" label={item.batch} />
                {item.section && (
                  <StatusBadge status="unavailable" label={`Section ${item.section}`} />
                )}
                <span className="ml-auto text-xs text-brand-500">
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
              <div className="mt-4 flex justify-end">
                <Button
                  size="sm"
                  variant="danger"
                  loading={busy === item.id}
                  onClick={() => remove(item)}
                >
                  Delete
                </Button>
              </div>
            </Card>
          ))}
        </div>
      )}
    </>
  )
}

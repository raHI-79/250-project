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
import { appendIf } from '../../lib/account'
import api from '../../lib/api'
import { linkError } from '../../lib/files'
import { useHighlight, useResource } from '../../lib/hooks'
import cx from '../../lib/cx'
import { formatDateTime } from '../../lib/time'

export default function AdminPostNoticesPage() {
  const toast = useToast()
  const confirm = useConfirm()
  const [title, setTitle] = useState('')
  const [description, setDescription] = useState('')
  const [batch, setBatch] = useState('')
  const [link, setLink] = useState('')
  const [file, setFile] = useState(null)
  const [error, setError] = useState('')
  const [saving, setSaving] = useState(false)
  const [busy, setBusy] = useState(0)
  const batches = useResource('/api/lookups/batches')
  const notices = useResource('/api/admin/notices')
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
    appendIf(payload, 'link', link.trim())
    if (file) payload.set('file', file)
    setSaving(true)
    try {
      await api.post('/api/admin/notices', payload)
      toast.success('Notice published')
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
      await api.delete(`/api/admin/notices/${item.id}`)
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
      <PageHeader eyebrow="Notice board" title="Notices" subtitle="Publish a notice for one batch or for everyone" />
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
          <Select label="Batch" value={batch} onChange={(event) => setBatch(event.target.value)}>
            <option value="">All batches</option>
            {(batches.data?.items || []).map((entry) => (
              <option key={entry} value={entry}>
                {entry}
              </option>
            ))}
          </Select>
          <LinkInput value={link} onChange={setLink} />
          <FileInput file={file} onChange={setFile} />
          <Button type="submit" loading={saving} className="w-full sm:w-auto sm:self-end">
            Publish the notice
          </Button>
        </form>
      </Card>
      <p className="font-display mb-3 text-lg font-semibold text-ink">Published notices</p>
      {notices.error && <Alert tone="error" className="mb-5">{notices.error}</Alert>}
      {notices.loading ? (
        <SkeletonCard />
      ) : items.length === 0 ? (
        <EmptyState
          icon={Megaphone}
          title="No notices"
          text="The notices you publish appear here."
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
                <p className="font-display text-lg font-bold text-ink">{item.title}</p>
                <StatusBadge
                  status={item.batch === 'ALL' ? 'approved' : 'unavailable'}
                  label={item.batch === 'ALL' ? 'All batches' : item.batch}
                />
                <span className="ml-auto text-xs text-muted">
                  {formatDateTime(item.created_at)}
                </span>
              </div>
              {item.description && (
                <p className="mt-2 text-sm whitespace-pre-line text-ink-soft">
                  {item.description}
                </p>
              )}
              <div className="mt-3 flex flex-wrap items-center justify-between gap-3">
                <AttachmentView
                  fileUrl={item.file_url}
                  fileName={item.file_name}
                  fileMime={item.file_mime}
                  fileSize={item.file_size}
                  linkUrl={item.link_url}
                />
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

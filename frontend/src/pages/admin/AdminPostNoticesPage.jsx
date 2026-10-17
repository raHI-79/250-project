import { useState } from 'react'
import { FileText, Megaphone } from 'lucide-react'
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
import PdfInput from '../../components/forms/PdfInput'
import api from '../../lib/api'
import { useResource } from '../../lib/hooks'
import { formatDateTime } from '../../lib/time'

export default function AdminPostNoticesPage() {
  const toast = useToast()
  const confirm = useConfirm()
  const [title, setTitle] = useState('')
  const [description, setDescription] = useState('')
  const [batch, setBatch] = useState('')
  const [file, setFile] = useState(null)
  const [error, setError] = useState('')
  const [saving, setSaving] = useState(false)
  const [busy, setBusy] = useState(0)
  const batches = useResource('/api/lookups/batches')
  const notices = useResource('/api/admin/notices')
  const publish = async (event) => {
    event.preventDefault()
    setError('')
    const payload = new FormData()
    payload.set('title', title.trim())
    payload.set('description', description.trim())
    payload.set('batch', batch)
    if (file) payload.set('file', file)
    setSaving(true)
    try {
      await api.post('/api/admin/notices', payload)
      toast.success('Notice published')
      setTitle('')
      setDescription('')
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
      message: `"${item.title}" and its PDF will be removed for everyone.`,
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
      <PageHeader title="Notices" subtitle="Publish a notice for one batch or for everyone" />
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
          <PdfInput file={file} onChange={setFile} />
          <Button type="submit" loading={saving} className="w-full sm:w-auto sm:self-end">
            Publish the notice
          </Button>
        </form>
      </Card>
      <p className="font-display mb-3 text-lg font-semibold text-brand-800">Published notices</p>
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
            <Card key={item.id} hover={false} className="animate-fade-in-up">
              <div className="flex flex-wrap items-center gap-2">
                <p className="font-display text-base font-semibold text-brand-900">{item.title}</p>
                <StatusBadge
                  status={item.batch === 'ALL' ? 'approved' : 'unavailable'}
                  label={item.batch === 'ALL' ? 'All batches' : item.batch}
                />
                <span className="ml-auto text-xs text-brand-500">
                  {formatDateTime(item.created_at)}
                </span>
              </div>
              {item.description && (
                <p className="mt-2 text-sm whitespace-pre-line text-brand-700">
                  {item.description}
                </p>
              )}
              <div className="mt-3 flex flex-wrap items-center justify-between gap-3">
                {item.file_url ? (
                  <a
                    href={item.file_url}
                    target="_blank"
                    rel="noreferrer"
                    className="inline-flex items-center gap-1.5 text-sm text-brand-600 hover:underline"
                  >
                    <FileText size={15} />
                    Open the PDF
                  </a>
                ) : (
                  <span className="text-xs text-brand-400">No attachment</span>
                )}
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

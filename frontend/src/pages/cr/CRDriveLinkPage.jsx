import { useEffect, useState } from 'react'
import { ExternalLink, Link2 } from 'lucide-react'
import {
  Alert,
  Button,
  Card,
  EmptyState,
  Input,
  PageHeader,
  SkeletonCard,
  useConfirm,
  useToast,
} from '../../components/ui'
import api from '../../lib/api'
import { useResource } from '../../lib/hooks'
import { relativeTime } from '../../lib/time'

export default function CRDriveLinkPage() {
  const toast = useToast()
  const confirm = useConfirm()
  const current = useResource('/api/cr/drive-link')
  const [link, setLink] = useState('')
  const [error, setError] = useState('')
  const [saving, setSaving] = useState(false)
  const [removing, setRemoving] = useState(false)
  const saved = current.data
  useEffect(() => {
    setLink(saved?.link || '')
  }, [saved?.link])
  const onSubmit = async (event) => {
    event.preventDefault()
    setError('')
    setSaving(true)
    try {
      await api.put('/api/cr/drive-link', { link: link.trim() })
      toast.success('Drive link saved')
      current.reload()
    } catch (failure) {
      setError(failure.message)
    } finally {
      setSaving(false)
    }
  }
  const remove = async () => {
    const ok = await confirm({
      title: 'Remove the Drive link?',
      message: `${saved?.title} will have no Drive folder until you add one again.`,
      confirmLabel: 'Remove',
    })
    if (!ok) return
    setRemoving(true)
    try {
      await api.delete('/api/cr/drive-link')
      toast.success('Drive link removed')
      setLink('')
      current.reload()
    } catch (failure) {
      toast.error(failure.message)
    } finally {
      setRemoving(false)
    }
  }
  return (
    <>
      <PageHeader
        title="Batch Drive link"
        subtitle="One Drive folder for your batch, shown to every student on the materials page"
      />
      {current.error && (
        <Alert tone="error" className="mb-5">
          {current.error}
        </Alert>
      )}
      {current.loading ? (
        <SkeletonCard className="mb-6" />
      ) : saved?.link ? (
        <Card hover={false} className="mb-6 flex flex-col gap-3">
          <div className="flex flex-wrap items-start justify-between gap-2">
            <p className="font-display text-2xl font-semibold text-brand-900">{saved.title}</p>
            <Button variant="danger" size="sm" loading={removing} onClick={remove}>
              Remove
            </Button>
          </div>
          <p className="text-xs text-brand-500">Updated {relativeTime(saved.updated_at)}</p>
          <a
            href={saved.link}
            target="_blank"
            rel="noreferrer"
            className="inline-flex w-fit items-center gap-2 rounded-lg bg-brand-600 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-brand-700"
          >
            <ExternalLink size={16} />
            Open Drive
          </a>
        </Card>
      ) : (
        <EmptyState
          icon={Link2}
          title="No Drive link yet"
          text="Add the folder your batch shares its materials in."
          className="mb-6"
        />
      )}
      <Card hover={false}>
        <form onSubmit={onSubmit} className="flex flex-col gap-4">
          {error && <Alert tone="error">{error}</Alert>}
          <Input label="Title" value={saved?.title || ''} readOnly disabled hint="Your batch code" />
          <Input
            label="Drive link"
            type="url"
            value={link}
            onChange={(event) => setLink(event.target.value)}
            placeholder="https://drive.google.com/drive/folders/..."
            required
          />
          <Button type="submit" loading={saving} className="w-full sm:w-auto sm:self-end">
            {saved?.link ? 'Save the link' : 'Add the link'}
          </Button>
        </form>
      </Card>
    </>
  )
}

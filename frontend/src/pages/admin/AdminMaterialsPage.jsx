import { useState } from 'react'
import { ExternalLink, FolderOpen, Plus } from 'lucide-react'
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
  useToast,
} from '../../components/ui'
import api from '../../lib/api'
import { useResource } from '../../lib/hooks'
import { formatDateTime } from '../../lib/time'

export default function AdminMaterialsPage() {
  const toast = useToast()
  const [batch, setBatch] = useState('ALL')
  const [title, setTitle] = useState('')
  const [link, setLink] = useState('')
  const [error, setError] = useState('')
  const [saving, setSaving] = useState(false)
  const [filter, setFilter] = useState('')
  const batches = useResource('/api/lookups/batches')
  const materials = useResource('/api/admin/materials', filter ? { batch: filter } : null)
  const save = async (event) => {
    event.preventDefault()
    setError('')
    setSaving(true)
    try {
      await api.post('/api/admin/materials', {
        batch,
        title: title.trim(),
        link: link.trim(),
      })
      toast.success('Material added')
      setTitle('')
      setLink('')
      materials.reload()
    } catch (failure) {
      setError(failure.message)
    } finally {
      setSaving(false)
    }
  }
  const items = materials.data?.items || []
  return (
    <>
      <PageHeader title="Materials" subtitle="Drive links shared with the students" />
      <Card hover={false} className="mb-6">
        <form onSubmit={save} className="flex flex-col gap-4">
          {error && <Alert tone="error">{error}</Alert>}
          <div className="grid gap-4 sm:grid-cols-2">
            <Select label="Batch" value={batch} onChange={(event) => setBatch(event.target.value)}>
              <option value="ALL">ALL</option>
              {(batches.data?.items || []).map((entry) => (
                <option key={entry} value={entry}>
                  {entry}
                </option>
              ))}
            </Select>
            <Input
              label="Title"
              value={title}
              onChange={(event) => setTitle(event.target.value)}
              required
            />
          </div>
          <div className="grid gap-4 sm:grid-cols-[1fr_auto] sm:items-end">
            <Input
              label="Drive link"
              type="url"
              value={link}
              onChange={(event) => setLink(event.target.value)}
              placeholder="https://drive.google.com/..."
              required
            />
            <Button type="submit" loading={saving}>
              <Plus size={15} />
              Add
            </Button>
          </div>
        </form>
      </Card>
      <Card hover={false} className="mb-5">
        <Select
          label="Filter by batch"
          value={filter}
          onChange={(event) => setFilter(event.target.value)}
          className="max-w-xs"
        >
          <option value="">Every batch</option>
          <option value="ALL">ALL</option>
          {(batches.data?.items || []).map((entry) => (
            <option key={entry} value={entry}>
              {entry}
            </option>
          ))}
        </Select>
      </Card>
      {materials.error && <Alert tone="error" className="mb-5">{materials.error}</Alert>}
      {materials.loading ? (
        <div className="grid gap-4 sm:grid-cols-2">
          <SkeletonCard />
          <SkeletonCard />
        </div>
      ) : items.length === 0 ? (
        <EmptyState
          icon={FolderOpen}
          title="No materials"
          text="Nothing matches this filter yet."
        />
      ) : (
        <div className="grid gap-4 sm:grid-cols-2">
          {items.map((item) => (
            <Card key={item.id} hover={false} className="animate-fade-in-up flex flex-col gap-3">
              <div className="flex flex-wrap items-start justify-between gap-2">
                <p className="font-display text-base font-semibold text-brand-900">{item.title}</p>
                <StatusBadge
                  status={item.batch === 'ALL' ? 'approved' : 'unavailable'}
                  label={item.batch}
                />
              </div>
              <p className="text-xs text-brand-500">
                {item.created_by_name} · {formatDateTime(item.created_at)}
              </p>
              <Button
                variant="secondary"
                size="sm"
                className="self-start"
                onClick={() => window.open(item.link, '_blank', 'noreferrer')}
              >
                <ExternalLink size={15} />
                Open the link
              </Button>
            </Card>
          ))}
        </div>
      )}
    </>
  )
}

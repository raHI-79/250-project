import { useState } from 'react'
import { ExternalLink, FolderOpen } from 'lucide-react'
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
  useConfirm,
  useToast,
} from '../../components/ui'
import { useAuth } from '../../context/auth'
import api from '../../lib/api'
import { useResource } from '../../lib/hooks'
import { formatDateTime } from '../../lib/time'

const VISIBILITIES = [
  { value: 'mine', label: 'My batch only' },
  { value: 'batch', label: 'A specific other batch' },
  { value: 'all', label: 'All batches' },
]

export default function CRMaterialsPage() {
  const toast = useToast()
  const confirm = useConfirm()
  const { user } = useAuth()
  const [title, setTitle] = useState('')
  const [link, setLink] = useState('')
  const [visibility, setVisibility] = useState('mine')
  const [batch, setBatch] = useState('')
  const [error, setError] = useState('')
  const [sending, setSending] = useState(false)
  const [removing, setRemoving] = useState(0)
  const batches = useResource('/api/lookups/batches')
  const materials = useResource('/api/cr/materials')
  const otherBatches = (batches.data?.items || []).filter((entry) => entry !== user?.batch)
  const onSubmit = async (event) => {
    event.preventDefault()
    setError('')
    setSending(true)
    try {
      await api.post('/api/cr/materials', {
        title: title.trim(),
        link: link.trim(),
        visibility,
        batch: visibility === 'batch' ? batch : '',
      })
      toast.success('Material added')
      setTitle('')
      setLink('')
      materials.reload()
    } catch (failure) {
      setError(failure.message)
    } finally {
      setSending(false)
    }
  }
  const remove = async (item) => {
    const ok = await confirm({
      title: 'Remove this material?',
      message: `"${item.title}" will no longer be visible to the students.`,
      confirmLabel: 'Remove',
    })
    if (!ok) return
    setRemoving(item.id)
    try {
      await api.delete(`/api/cr/materials/${item.id}`)
      toast.success('Material removed')
      materials.reload()
    } catch (failure) {
      toast.error(failure.message)
    } finally {
      setRemoving(0)
    }
  }
  const items = materials.data?.items || []
  return (
    <>
      <PageHeader title="Materials" subtitle="Share Drive links with your batch or every batch" />
      <Card hover={false} className="mb-6">
        <form onSubmit={onSubmit} className="flex flex-col gap-4">
          {error && <Alert tone="error">{error}</Alert>}
          <Input
            label="Title"
            value={title}
            onChange={(event) => setTitle(event.target.value)}
            required
          />
          <Input
            label="Drive link"
            type="url"
            value={link}
            onChange={(event) => setLink(event.target.value)}
            placeholder="https://drive.google.com/..."
            required
          />
          <Select
            label="Visibility"
            value={visibility}
            onChange={(event) => setVisibility(event.target.value)}
          >
            {VISIBILITIES.map((entry) => (
              <option key={entry.value} value={entry.value}>
                {entry.label}
              </option>
            ))}
          </Select>
          {visibility === 'batch' && (
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
          <Button type="submit" loading={sending} className="w-full sm:w-auto sm:self-end">
            Add the material
          </Button>
        </form>
      </Card>
      <p className="font-display mb-3 text-lg font-semibold text-brand-800">My materials</p>
      {materials.error && <Alert tone="error" className="mb-5">{materials.error}</Alert>}
      {materials.loading ? (
        <div className="grid gap-4 sm:grid-cols-2">
          <SkeletonCard />
          <SkeletonCard />
        </div>
      ) : items.length === 0 ? (
        <EmptyState
          icon={FolderOpen}
          title="No materials yet"
          text="The links you add appear here."
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
              <a
                href={item.link}
                target="_blank"
                rel="noreferrer"
                className="inline-flex items-center gap-1.5 text-sm text-brand-600 hover:underline"
              >
                <ExternalLink size={15} />
                Open the link
              </a>
              <div className="flex items-center justify-between gap-3">
                <p className="text-xs text-brand-500">{formatDateTime(item.created_at)}</p>
                <Button
                  variant="danger"
                  size="sm"
                  loading={removing === item.id}
                  onClick={() => remove(item)}
                >
                  Remove
                </Button>
              </div>
            </Card>
          ))}
        </div>
      )}
    </>
  )
}

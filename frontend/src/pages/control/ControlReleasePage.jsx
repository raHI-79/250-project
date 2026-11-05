import { useState } from 'react'
import { Send, Smartphone } from 'lucide-react'
import {
  Alert,
  Button,
  Card,
  CardHeader,
  PageHeader,
  SkeletonLines,
  useToast,
} from '../../components/ui'
import api from '../../lib/api'
import { useResource } from '../../lib/hooks'

export default function ControlReleasePage() {
  const toast = useToast()
  const { data, loading, error, reload } = useResource('/api/super-admin/release')
  const [busy, setBusy] = useState(false)
  const announce = async () => {
    setBusy(true)
    try {
      const result = await api.post('/api/super-admin/release/announce')
      toast(result.message)
      await reload()
    } catch (failure) {
      toast(failure.message, 'error')
    } finally {
      setBusy(false)
    }
  }
  return (
    <>
      <PageHeader
        eyebrow="Control room"
        title="Android app"
        subtitle="The release people download and the version they must have"
      />
      {error && (
        <Alert tone="error" className="mb-5">
          {error}
        </Alert>
      )}
      {loading ? (
        <SkeletonLines lines={4} />
      ) : (
        <div className="grid gap-4 lg:grid-cols-2">
          <Card hover={false}>
            <CardHeader title="Current release" subtitle="Read from the GitHub releases." />
            <dl className="mt-4 grid grid-cols-2 gap-3 text-sm">
              <div>
                <dt className="text-xs text-muted">Version</dt>
                <dd className="font-display text-section font-bold text-ink">
                  {data?.version || 'Unknown'}
                </dd>
              </div>
              <div>
                <dt className="text-xs text-muted">Downloads</dt>
                <dd className="font-display text-section font-bold text-ink">
                  {data?.downloads ?? 0}
                </dd>
              </div>
              <div className="col-span-2">
                <dt className="text-xs text-muted">Minimum supported</dt>
                <dd className="text-ink">
                  {data?.min_version || 'No floor — every version may sign in'}
                </dd>
              </div>
            </dl>
            <Button className="mt-4" onClick={announce} loading={busy}>
              <Send size={16} aria-hidden="true" />
              Send the update notification again
            </Button>
          </Card>
          <Card hover={false}>
            <CardHeader title="Share link" subtitle="Anyone with this link can install the app." />
            <p className="mt-4 wrap-long rounded-tile bg-wash p-3 font-mono text-sm text-ink">
              {data?.share_link}
            </p>
            <Button
              variant="secondary"
              className="mt-3"
              onClick={() => {
                navigator.clipboard?.writeText(data?.share_link || '')
                toast('Link copied.')
              }}
            >
              <Smartphone size={16} aria-hidden="true" />
              Copy the link
            </Button>
            {data?.notes && (
              <p className="mt-4 wrap-long text-sm leading-relaxed text-ink-soft">{data.notes}</p>
            )}
          </Card>
        </div>
      )}
    </>
  )
}

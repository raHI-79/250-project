import { useState } from 'react'
import { Smartphone } from 'lucide-react'
import {
  Alert,
  Button,
  Card,
  EmptyState,
  Field,
  PageHeader,
  PullToRefresh,
  SkeletonCard,
  Textarea,
  useToast,
} from '../../components/ui'
import api from '../../lib/api'
import { useHighlight, useResource } from '../../lib/hooks'
import cx from '../../lib/cx'
import { formatDateTime } from '../../lib/time'
import { sectionLabel } from '../../lib/attendance'

function RequestCard({ request, busy, onDecide, flash }) {
  const [note, setNote] = useState('')
  const blocked = request.needs_note && !note.trim()
  return (
    <Card hover={false} className={cx('animate-fade-in-up', flash && 'animate-flash border-gold-300')}>
      <div className="flex flex-wrap items-start justify-between gap-2">
        <div className="min-w-0">
          <p className="font-display wrap-long text-lg font-bold text-ink">
            {request.student_name}
          </p>
          <p className="mt-0.5 text-xs text-muted">
            {request.reg_no} · {request.batch} · {sectionLabel(request.section)} ·{' '}
            {formatDateTime(request.created_at)}
          </p>
        </div>
        <span className="shrink-0 text-xs text-muted">
          {request.changes} change{request.changes === 1 ? '' : 's'} this semester
        </span>
      </div>
      <p className="mt-3 rounded-tile bg-wash px-3 py-2 text-sm text-ink-soft">{request.reason}</p>
      {request.needs_note && (
        <Alert tone="warning" className="mt-3">
          This student already used both phone changes of the semester, so add a note to approve.
        </Alert>
      )}
      <Field label="Note" htmlFor={`note-${request.id}`} className="mt-3">
        <Textarea
          id={`note-${request.id}`}
          rows={2}
          value={note}
          onChange={(event) => setNote(event.target.value)}
          placeholder="What you checked before deciding"
        />
      </Field>
      <div className="mt-3 flex flex-wrap gap-2">
        <Button
          loading={busy === 'approve'}
          disabled={blocked}
          onClick={() => onDecide(request.id, 'approve', note.trim())}
        >
          Approve this phone
        </Button>
        <Button
          variant="danger"
          loading={busy === 'reject'}
          onClick={() => onDecide(request.id, 'reject', note.trim())}
        >
          Refuse
        </Button>
      </div>
    </Card>
  )
}

export default function TeacherPhoneRequestsPage() {
  const toast = useToast()
  const { data, loading, error, reload } = useResource('/api/teacher/phone-requests')
  const highlight = useHighlight(Boolean(data))
  const [busy, setBusy] = useState({})
  const items = data?.items || []
  const decide = async (requestID, action, note) => {
    setBusy({ [requestID]: action })
    try {
      const result = await api.post(`/api/teacher/phone-requests/${requestID}/${action}`, { note })
      toast.success(result?.message || 'The student was told.')
      reload()
    } catch (failure) {
      toast.error(failure.message)
    } finally {
      setBusy({})
    }
  }
  return (
    <PullToRefresh onRefresh={reload}>
      <PageHeader
        eyebrow="Your students"
        title="Phone requests"
        subtitle="Students of the batches you teach asking to move their attendance to another phone. Any one of their teachers can decide."
      />
      {error && (
        <Alert tone="error" className="mb-5">
          {error}
        </Alert>
      )}
      {loading ? (
        <SkeletonCard />
      ) : items.length === 0 ? (
        <EmptyState
          icon={Smartphone}
          title="Nothing waiting"
          text="When a student signs in from a new phone and asks for it, the request shows up here with their reason."
        />
      ) : (
        <div className="flex flex-col gap-4">
          {items.map((request) => (
            <div key={request.id} ref={highlight.register(request.id)}>
              <RequestCard
                request={request}
                busy={busy[request.id]}
                onDecide={decide}
                flash={highlight.highlighted(request.id)}
              />
            </div>
          ))}
        </div>
      )}
    </PullToRefresh>
  )
}

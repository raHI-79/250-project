import { useState } from 'react'
import { ClipboardCheck } from 'lucide-react'
import {
  Alert,
  Button,
  Card,
  EmptyState,
  PageHeader,
  SkeletonCard,
  StatusBadge,
  useConfirm,
  useToast,
} from '../../components/ui'
import api from '../../lib/api'
import { useResource } from '../../lib/hooks'
import { formatDate, formatDateTime, formatTimeRange } from '../../lib/time'

const TYPE_LABELS = { class: 'Class', exam: 'Exam', extra: 'Extra class' }

const sectionLabel = (section) => (section === 'ALL' ? 'Both sections' : `Section ${section}`)

export default function ApprovalRequestsPage() {
  const toast = useToast()
  const confirm = useConfirm()
  const { data, loading, error, reload } = useResource('/api/teacher/requests')
  const [busy, setBusy] = useState(0)
  const [decided, setDecided] = useState([])
  const items = (data?.items || []).filter((item) => !decided.includes(item.id))
  const decide = async (item, action) => {
    if (action === 'reject') {
      const ok = await confirm({
        title: 'Reject this request?',
        message: `${item.course_code} on ${formatDate(item.event_date)} will not be scheduled and ${item.requested_by} is notified.`,
        confirmLabel: 'Reject',
      })
      if (!ok) return
    }
    setBusy(item.id)
    try {
      await api.post(`/api/teacher/requests/${item.id}/${action}`)
      toast.success(action === 'approve' ? 'Request approved' : 'Request rejected')
      setDecided((current) => [...current, item.id])
      reload()
    } catch (failure) {
      toast.error(failure.message)
    } finally {
      setBusy(0)
    }
  }
  return (
    <>
      <PageHeader
        title="Approval Requests"
        subtitle="Class, exam and extra class requests waiting for your decision"
      />
      {error && <Alert tone="error" className="mb-5">{error}</Alert>}
      {loading ? (
        <SkeletonCard />
      ) : items.length === 0 ? (
        <EmptyState
          icon={ClipboardCheck}
          title="Nothing to decide"
          text="No request is waiting for you."
        />
      ) : (
        <div className="flex flex-col gap-4">
          {items.map((item) => (
            <Card key={item.id} hover={false} className="animate-fade-in-up">
              <div className="flex flex-wrap items-center gap-2">
                <p className="font-display text-base font-semibold text-brand-900">
                  {item.course_code}
                  {item.course_title ? ` ${item.course_title}` : ''}
                </p>
                <StatusBadge
                  status="pending"
                  label={TYPE_LABELS[item.event_type] || item.event_type}
                />
                <span className="ml-auto text-xs text-brand-500">
                  Asked {formatDateTime(item.created_at)}
                </span>
              </div>
              <div className="mt-3 grid gap-2 text-sm text-brand-700 sm:grid-cols-2">
                <p>
                  <span className="text-brand-500">Batch </span>
                  {item.batch} · {sectionLabel(item.section)}
                </p>
                <p>
                  <span className="text-brand-500">Room </span>
                  {item.room}
                </p>
                <p>
                  <span className="text-brand-500">Date </span>
                  {formatDate(item.event_date)}
                </p>
                <p className="tabular-nums">
                  <span className="text-brand-500">Time </span>
                  {formatTimeRange(item.start_time, item.end_time)}
                </p>
                <p className="sm:col-span-2">
                  <span className="text-brand-500">Requested by </span>
                  {item.requested_by} ({item.requested_by_reg_no})
                </p>
              </div>
              <div className="mt-4 flex flex-wrap justify-end gap-2">
                <Button loading={busy === item.id} onClick={() => decide(item, 'approve')}>
                  Approve
                </Button>
                <Button
                  variant="danger"
                  loading={busy === item.id}
                  onClick={() => decide(item, 'reject')}
                >
                  Reject
                </Button>
              </div>
            </Card>
          ))}
        </div>
      )}
    </>
  )
}

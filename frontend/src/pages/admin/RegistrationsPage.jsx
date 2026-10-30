import { useState } from 'react'
import { UserCheck } from 'lucide-react'
import {
  Alert,
  Avatar,
  Button,
  DataTable,
  EmptyState,
  Field,
  PageHeader,
  Sheet,
  SkeletonTable,
  Textarea,
  useToast,
} from '../../components/ui'
import api from '../../lib/api'
import { useResource } from '../../lib/hooks'
import { formatDate } from '../../lib/time'

const HEADINGS = [
  'Name',
  'Reg No',
  'Batch',
  'Section',
  'Email',
  'Phone',
  'Blood Group',
  'Date of Birth',
  'Registered',
  '',
]

function RejectSheet({ item, reasons, min, busy, onClose, onReject }) {
  const [reason, setReason] = useState('')
  const trimmed = reason.trim()
  const short = trimmed.length > 0 && trimmed.length < min
  return (
    <Sheet
      open={Boolean(item)}
      onClose={onClose}
      title="Reject this registration?"
      description={item ? `${item.full_name} (${item.reg_no}) will not be able to sign in.` : ''}
      footer={
        <div className="flex gap-2">
          <Button variant="secondary" className="flex-1" onClick={onClose}>
            Keep it pending
          </Button>
          <Button
            variant="danger"
            className="flex-1"
            loading={busy}
            disabled={trimmed.length < min}
            onClick={() => onReject(trimmed)}
          >
            Reject and email
          </Button>
        </div>
      }
    >
      <p className="text-sm text-muted">
        The student is emailed this reason, so write it the way you want them to read it.
      </p>
      <div className="mt-3 flex flex-wrap gap-2">
        {reasons.map((quick) => (
          <button
            key={quick}
            type="button"
            onClick={() => setReason(quick)}
            className="press glass-chip min-h-10 rounded-full border px-3 text-sm font-medium text-muted hover:border-brand-300 hover:text-ink"
          >
            {quick}
          </button>
        ))}
      </div>
      <Field
        label="Reason"
        htmlFor="reject-reason"
        className="mt-4"
        error={short ? `Write at least ${min} characters.` : ''}
        required
      >
        <Textarea
          id="reject-reason"
          rows={3}
          value={reason}
          onChange={(event) => setReason(event.target.value)}
          placeholder="Registration number does not match"
        />
      </Field>
    </Sheet>
  )
}

export default function RegistrationsPage() {
  const toast = useToast()
  const { data, loading, error, reload } = useResource('/api/admin/registrations')
  const picks = useResource('/api/admin/rejection-reasons')
  const [busy, setBusy] = useState(0)
  const [decided, setDecided] = useState([])
  const [rejecting, setRejecting] = useState(null)
  const items = (data?.items || []).filter((item) => !decided.includes(item.id))
  const send = async (item, action, reason) => {
    setBusy(item.id)
    try {
      await api.post(
        `/api/admin/registrations/${item.id}/${action}`,
        action === 'reject' ? { reason } : undefined,
      )
      toast.success(
        action === 'approve'
          ? 'Registration approved. The student was emailed a welcome.'
          : 'Registration rejected. The student was emailed the reason.',
      )
      setDecided((current) => [...current, item.id])
      setRejecting(null)
      reload()
    } catch (failure) {
      toast.error(failure.message)
    } finally {
      setBusy(0)
    }
  }
  const decide = (item, action) => {
    if (action === 'reject') {
      setRejecting(item)
      return
    }
    send(item, action)
  }
  return (
    <>
      <PageHeader
        eyebrow="Student records" title="Registrations"
        subtitle="Student accounts waiting for your approval, oldest first"
      />
      {error && <Alert tone="error" className="mb-5">{error}</Alert>}
      <RejectSheet
        item={rejecting}
        reasons={picks.data?.items || []}
        min={picks.data?.min || 5}
        busy={busy === rejecting?.id}
        onClose={() => setRejecting(null)}
        onReject={(reason) => send(rejecting, 'reject', reason)}
      />
      {loading ? (
        <SkeletonTable columns={10} />
      ) : items.length === 0 ? (
        <EmptyState
          icon={UserCheck}
          title="Nothing to review"
          text="Every registration has been decided."
        />
      ) : (
        <DataTable className="animate-fade-in">
          <thead>
            <tr className="border-b border-line bg-wash">
              {HEADINGS.map((heading, index) => (
                <th key={heading || index} className="px-4 py-3 font-semibold text-ink-soft">
                  {heading}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {items.map((item) => (
              <tr key={item.id} className="border-b border-line last:border-b-0">
                <td className="px-4 py-3">
                  <span className="flex items-center gap-2.5">
                    <Avatar src={item.photo_url} name={item.full_name} size="sm" />
                    <span className="font-medium text-ink">{item.full_name}</span>
                  </span>
                </td>
                <td className="px-4 py-3 tabular-nums text-ink-soft">{item.reg_no}</td>
                <td className="px-4 py-3 text-ink-soft">{item.batch}</td>
                <td className="px-4 py-3 text-ink-soft">{item.section}</td>
                <td className="px-4 py-3 text-ink-soft">{item.email}</td>
                <td className="px-4 py-3 text-ink-soft">{item.phone}</td>
                <td className="px-4 py-3 text-ink-soft">{item.blood_group}</td>
                <td className="px-4 py-3 whitespace-nowrap text-ink-soft">
                  {item.date_of_birth ? formatDate(item.date_of_birth) : '—'}
                </td>
                <td className="px-4 py-3 whitespace-nowrap text-ink-soft">
                  {formatDate(item.created_at)}
                </td>
                <td className="px-4 py-3">
                  <div className="flex flex-wrap justify-end gap-2">
                    <Button
                      size="sm"
                      loading={busy === item.id}
                      onClick={() => decide(item, 'approve')}
                    >
                      Approve
                    </Button>
                    <Button
                      size="sm"
                      variant="danger"
                      loading={busy === item.id}
                      onClick={() => decide(item, 'reject')}
                    >
                      Reject
                    </Button>
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </DataTable>
      )}
    </>
  )
}

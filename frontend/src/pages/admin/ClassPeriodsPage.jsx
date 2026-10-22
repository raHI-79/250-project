import { useState } from 'react'
import { ArrowDown, ArrowUp, Clock, Pencil, Plus, Trash2 } from 'lucide-react'
import {
  Alert,
  Button,
  Card,
  DataTable,
  EmptyState,
  Input,
  PageHeader,
  SkeletonTable,
  useConfirm,
  useToast,
} from '../../components/ui'
import api from '../../lib/api'
import { useResource } from '../../lib/hooks'
import { formatTimeRange } from '../../lib/time'

const EMPTY = { label: '', start_time: '', end_time: '', is_break: false }

export default function ClassPeriodsPage() {
  const toast = useToast()
  const confirm = useConfirm()
  const [form, setForm] = useState(EMPTY)
  const [editing, setEditing] = useState(0)
  const [draft, setDraft] = useState(EMPTY)
  const [error, setError] = useState('')
  const [saving, setSaving] = useState(false)
  const [busy, setBusy] = useState(0)
  const { data, loading, error: listError, reload } = useResource('/api/admin/periods')
  const items = data?.items || []
  const set = (field) => (event) => setForm((current) => ({ ...current, [field]: event.target.value }))
  const setDraftField = (field) => (event) =>
    setDraft((current) => ({ ...current, [field]: event.target.value }))
  const add = async (event) => {
    event.preventDefault()
    setError('')
    setSaving(true)
    try {
      await api.post('/api/admin/periods', { ...form, label: form.label.trim() })
      toast.success('Class period added')
      setForm(EMPTY)
      reload()
    } catch (failure) {
      setError(failure.message)
    } finally {
      setSaving(false)
    }
  }
  const startEdit = (period) => {
    setEditing(period.id)
    setDraft({
      label: period.label,
      start_time: period.start_time,
      end_time: period.end_time,
      is_break: period.is_break,
    })
  }
  const saveEdit = async (period) => {
    setBusy(period.id)
    try {
      await api.patch(`/api/admin/periods/${period.id}`, { ...draft, label: draft.label.trim() })
      toast.success('Class period updated')
      setEditing(0)
      reload()
    } catch (failure) {
      toast.error(failure.message)
    } finally {
      setBusy(0)
    }
  }
  const remove = async (period) => {
    const ok = await confirm({
      title: 'Delete this class period?',
      message: `${period.label} (${formatTimeRange(period.start_time, period.end_time)}) leaves the day.`,
      confirmLabel: 'Delete the period',
    })
    if (!ok) return
    setBusy(period.id)
    try {
      await api.delete(`/api/admin/periods/${period.id}`)
      toast.success('Class period deleted')
      reload()
    } catch (failure) {
      toast.error(failure.message)
    } finally {
      setBusy(0)
    }
  }
  const move = async (index, step) => {
    const target = index + step
    if (target < 0 || target >= items.length) return
    const ids = items.map((item) => item.id)
    ;[ids[index], ids[target]] = [ids[target], ids[index]]
    setBusy(items[index].id)
    try {
      await api.post('/api/admin/periods/reorder', { ids })
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
        eyebrow="The teaching day"
        title="Class periods"
        subtitle="The periods every routine entry, class and exam has to sit in"
      />
      <Card hover={false} className="mb-5">
        <form onSubmit={add} className="flex flex-col gap-4">
          {error && <Alert tone="error">{error}</Alert>}
          <div className="grid gap-4 sm:grid-cols-[2fr_1fr_1fr_auto] sm:items-end">
            <Input
              label="Label"
              value={form.label}
              onChange={set('label')}
              placeholder="Period 10"
              required
            />
            <Input label="Start" type="time" value={form.start_time} onChange={set('start_time')} required />
            <Input label="End" type="time" value={form.end_time} onChange={set('end_time')} required />
            <Button type="submit" loading={saving}>
              <Plus size={15} />
              Add
            </Button>
          </div>
          <label className="flex items-center gap-2 text-sm text-ink-soft">
            <input
              type="checkbox"
              checked={form.is_break}
              onChange={(event) => setForm((current) => ({ ...current, is_break: event.target.checked }))}
              className="size-4 rounded border-line-strong"
            />
            This period is a break, so nothing may be scheduled in it
          </label>
        </form>
      </Card>
      {listError && (
        <Alert tone="error" className="mb-5">
          {listError}
        </Alert>
      )}
      {loading ? (
        <SkeletonTable columns={4} />
      ) : items.length === 0 ? (
        <EmptyState icon={Clock} title="No class periods" text="Add the first period above." />
      ) : (
        <DataTable className="animate-fade-in">
          <thead>
            <tr className="border-b border-line bg-wash">
              <th className="px-4 py-3 font-semibold text-ink-soft">Period</th>
              <th className="px-4 py-3 font-semibold text-ink-soft">Time</th>
              <th className="px-4 py-3 font-semibold text-ink-soft">Kind</th>
              <th className="px-4 py-3 text-right font-semibold text-ink-soft">Actions</th>
            </tr>
          </thead>
          <tbody>
            {items.map((item, index) => (
              <tr key={item.id} className="border-b border-line last:border-b-0">
                {editing === item.id ? (
                  <td colSpan={4} className="px-4 py-3">
                    <div className="grid gap-3 sm:grid-cols-[2fr_1fr_1fr_auto] sm:items-end">
                      <Input label="Label" value={draft.label} onChange={setDraftField('label')} />
                      <Input
                        label="Start"
                        type="time"
                        value={draft.start_time}
                        onChange={setDraftField('start_time')}
                      />
                      <Input
                        label="End"
                        type="time"
                        value={draft.end_time}
                        onChange={setDraftField('end_time')}
                      />
                      <div className="flex gap-2">
                        <Button size="sm" loading={busy === item.id} onClick={() => saveEdit(item)}>
                          Save
                        </Button>
                        <Button size="sm" variant="ghost" onClick={() => setEditing(0)}>
                          Cancel
                        </Button>
                      </div>
                    </div>
                    <label className="mt-3 flex items-center gap-2 text-sm text-ink-soft">
                      <input
                        type="checkbox"
                        checked={draft.is_break}
                        onChange={(event) =>
                          setDraft((current) => ({ ...current, is_break: event.target.checked }))
                        }
                        className="size-4 rounded border-line-strong"
                      />
                      This period is a break
                    </label>
                  </td>
                ) : (
                  <>
                    <td className="px-4 py-3 font-medium text-ink">{item.label}</td>
                    <td className="px-4 py-3 tabular-nums text-ink-soft">
                      {formatTimeRange(item.start_time, item.end_time)}
                    </td>
                    <td className="px-4 py-3 text-ink-soft">{item.is_break ? 'Break' : 'Teaching'}</td>
                    <td className="px-4 py-3">
                      <div className="flex flex-wrap justify-end gap-2">
                        <Button
                          variant="ghost"
                          size="sm"
                          aria-label={`Move ${item.label} earlier`}
                          disabled={index === 0 || busy === item.id}
                          onClick={() => move(index, -1)}
                        >
                          <ArrowUp size={14} />
                        </Button>
                        <Button
                          variant="ghost"
                          size="sm"
                          aria-label={`Move ${item.label} later`}
                          disabled={index === items.length - 1 || busy === item.id}
                          onClick={() => move(index, 1)}
                        >
                          <ArrowDown size={14} />
                        </Button>
                        <Button variant="secondary" size="sm" onClick={() => startEdit(item)}>
                          <Pencil size={14} />
                          Edit
                        </Button>
                        <Button
                          variant="danger"
                          size="sm"
                          loading={busy === item.id}
                          onClick={() => remove(item)}
                        >
                          <Trash2 size={14} />
                          Delete
                        </Button>
                      </div>
                    </td>
                  </>
                )}
              </tr>
            ))}
          </tbody>
        </DataTable>
      )}
    </>
  )
}

import { useEffect, useMemo, useState } from 'react'
import { Eye, ListChecks, Send } from 'lucide-react'
import ItemBadges from '../../../components/schedule/ItemBadges'
import PreviewModal from '../../../components/schedule/PreviewModal'
import SlotFields from '../../../components/schedule/SlotFields'
import TimeRangeFields from '../../../components/forms/TimeRangeFields'
import {
  Alert,
  Button,
  Card,
  EmptyState,
  SkeletonCard,
  Textarea,
  useConfirm,
  useToast,
} from '../../../components/ui'
import api from '../../../lib/api'
import { usePeriods } from '../../../lib/hooks'
import {
  MODE_ROOM,
  ORIGIN_TEACHER_EXAM,
  courseLine,
  periodRangeError,
  placeOf,
  sectionShort,
  slotError,
} from '../../../lib/schedule'
import { formatDate, formatTimeRange } from '../../../lib/time'
import cx from '../../../lib/cx'

export default function FinalTab({ final, highlight, onPublished }) {
  const toast = useToast()
  const periods = usePeriods()
  const confirm = useConfirm()
  const [picked, setPicked] = useState([])
  const [editing, setEditing] = useState(null)
  const [draft, setDraft] = useState(null)
  const [busy, setBusy] = useState(0)
  const [previewOpen, setPreviewOpen] = useState(false)
  const [publishing, setPublishing] = useState(false)
  const days = useMemo(() => final.data?.days || [], [final.data])
  const items = useMemo(() => days.flatMap((day) => day.items), [days])
  useEffect(() => {
    setPicked((current) => current.filter((id) => items.some((item) => item.id === id)))
  }, [items])
  const toggle = (id) =>
    setPicked((current) =>
      current.includes(id) ? current.filter((entry) => entry !== id) : [...current, id],
    )
  const pickAll = () => setPicked(items.map((item) => item.id))
  const openEdit = (item) => {
    setEditing(item.id)
    setDraft({
      start: item.start_time || '',
      end: item.end_time || '',
      mode: item.mode || MODE_ROOM,
      room: item.room || '',
      meeting_link: item.meeting_link || '',
      note: item.note || '',
    })
  }
  const saveEdit = async (item) => {
    const timeProblem = periodRangeError(periods, draft.start, draft.end)
    const found = timeProblem || slotError(draft.mode, draft.room, draft.meeting_link)
    if (found) {
      toast.error(found)
      return
    }
    setBusy(item.id)
    try {
      await api.patch(`/api/cr/items/${item.id}`, {
        start_time: draft.start,
        end_time: draft.end,
        mode: draft.mode,
        room: draft.room,
        meeting_link: draft.meeting_link,
        note: draft.note,
      })
      toast.success('Item updated')
      setEditing(null)
      final.reload()
    } catch (failure) {
      toast.error(failure.message)
    } finally {
      setBusy(0)
    }
  }
  const removeFromFinal = async (item) => {
    setBusy(item.id)
    try {
      await api.delete(`/api/cr/items/${item.id}/final`)
      toast.success('Removed from the final routine')
      final.reload()
    } catch (failure) {
      toast.error(failure.message)
    } finally {
      setBusy(0)
    }
  }
  const publish = async () => {
    const ok = await confirm({
      title: `Publish ${picked.length} item${picked.length === 1 ? '' : 's'}?`,
      message: 'Your batch reads the published routine straight away.',
      confirmLabel: 'Publish',
    })
    if (!ok) return
    setPublishing(true)
    try {
      const result = await api.post('/api/cr/final/publish', { item_ids: picked })
      toast.success(`Published ${result?.published || picked.length} items`)
      setPreviewOpen(false)
      setPicked([])
      final.reload()
      onPublished()
    } catch (failure) {
      toast.error(failure.message)
    } finally {
      setPublishing(false)
    }
  }
  const previewDays = days
    .map((day) => ({ ...day, items: day.items.filter((item) => picked.includes(item.id)) }))
    .filter((day) => day.items.length > 0)
  if (final.loading) return <SkeletonCard />
  if (final.error) return <Alert tone="error">{final.error}</Alert>
  if (items.length === 0) {
    return (
      <EmptyState
        icon={ListChecks}
        title="The final routine is empty"
        text="Add items from the upcoming routine, then publish them for your batch."
      />
    )
  }
  return (
    <>
      <Card hover={false} className="mb-4 flex flex-wrap items-center gap-3">
        <p className="text-sm text-brand-600">
          {picked.length} of {items.length} picked
          {final.data?.needs_time > 0 && ` · ${final.data.needs_time} still need a time`}
          {final.data?.changed_since_publish > 0 &&
            ` · ${final.data.changed_since_publish} changed since the last publish`}
        </p>
        <div className="ml-auto flex flex-wrap gap-2">
          <Button variant="secondary" size="sm" onClick={pickAll}>
            Pick every item
          </Button>
          <Button
            variant="secondary"
            size="sm"
            disabled={picked.length === 0}
            onClick={() => setPreviewOpen(true)}
          >
            <Eye size={15} />
            Preview
          </Button>
          <Button size="sm" disabled={picked.length === 0} loading={publishing} onClick={publish}>
            <Send size={15} />
            Publish All
          </Button>
        </div>
      </Card>
      <div className="flex flex-col gap-4">
        {days.map((day) => (
          <Card key={day.date} hover={false} className="animate-fade-in-up">
            <p className="font-display text-lg font-bold text-ink">
              {formatDate(day.date)} · {day.weekday}
            </p>
            <ul className="mt-2 flex flex-col gap-3">
              {day.items.map((item) => (
                <li
                  key={item.id}
                  ref={highlight.register(item.id)}
                  className={cx(
                    'rounded-md border border-line p-3 transition-colors',
                    highlight.highlighted(item.id) && 'border-accent-300 bg-accent-50/60',
                  )}
                >
                  <div className="flex flex-wrap items-start gap-3">
                    <input
                      type="checkbox"
                      checked={picked.includes(item.id)}
                      onChange={() => toggle(item.id)}
                      aria-label={`Pick ${courseLine(item)}`}
                      className="mt-1 size-4 rounded border-line-strong text-brand-600"
                    />
                    <div className="min-w-0 flex-1">
                      <div className="flex flex-wrap items-center gap-x-3 gap-y-1.5">
                        <span className="font-medium tabular-nums text-brand-600">
                          {formatTimeRange(item.start_time, item.end_time) || 'No time yet'}
                        </span>
                        <span className="font-medium text-ink">{courseLine(item)}</span>
                        <span className="text-muted">
                          {sectionShort(item.section)}
                          {placeOf(item) ? ` · ${placeOf(item)}` : ''}
                        </span>
                        <ItemBadges item={item} showOrigin showFinal />
                      </div>
                      {item.note && <p className="mt-1 text-xs text-muted">{item.note}</p>}
                      {editing === item.id ? (
                        <div className="mt-3 flex flex-col gap-4 rounded-md border border-line bg-wash/40 p-3">
                          <TimeRangeFields
                            periods={periods}
                            start={draft.start}
                            end={draft.end}
                            onStartChange={(value) =>
                              setDraft((current) => ({ ...current, start: value }))
                            }
                            onEndChange={(value) =>
                              setDraft((current) => ({ ...current, end: value }))
                            }
                          />
                          <SlotFields value={draft} onChange={setDraft} />
                          <Textarea
                            label="Extra information"
                            value={draft.note}
                            onChange={(event) =>
                              setDraft((current) => ({ ...current, note: event.target.value }))
                            }
                            rows={2}
                          />
                          {item.origin === ORIGIN_TEACHER_EXAM && (
                            <Alert tone="info">
                              The teacher owns the date, the syllabus and the instructions of an
                              exam. You set the time, the place and the extra information.
                            </Alert>
                          )}
                          <div className="flex flex-wrap gap-2">
                            <Button size="sm" loading={busy === item.id} onClick={() => saveEdit(item)}>
                              Save
                            </Button>
                            <Button size="sm" variant="secondary" onClick={() => setEditing(null)}>
                              Cancel
                            </Button>
                          </div>
                        </div>
                      ) : (
                        <div className="mt-2.5 flex flex-wrap gap-2">
                          <Button size="sm" variant="secondary" onClick={() => openEdit(item)}>
                            Edit
                          </Button>
                          <Button
                            size="sm"
                            variant="secondary"
                            loading={busy === item.id}
                            onClick={() => removeFromFinal(item)}
                          >
                            Remove from final
                          </Button>
                        </div>
                      )}
                    </div>
                  </div>
                </li>
              ))}
            </ul>
          </Card>
        ))}
      </div>
      <PreviewModal
        open={previewOpen}
        days={previewDays}
        count={picked.length}
        busy={publishing}
        onClose={() => setPreviewOpen(false)}
        onPublish={publish}
      />
    </>
  )
}

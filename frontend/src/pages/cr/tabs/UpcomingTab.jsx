import { useState } from 'react'
import { CalendarClock, Link2, Pencil, Trash2 } from 'lucide-react'
import ItemBadges from '../../../components/schedule/ItemBadges'
import { CancelModal, LinkModal, RescheduleModal } from '../../../components/schedule/ScheduleModals'
import { Alert, Button, Card, EmptyState, SkeletonCard, useConfirm, useToast } from '../../../components/ui'
import api from '../../../lib/api'
import {
  ACTION_ADD_TO_FINAL,
  ACTION_CANCEL,
  ACTION_DELETE,
  ACTION_EDIT,
  ACTION_REMOVE_FROM_FINAL,
  ACTION_REQUEST_RESCHEDULE,
  ACTION_SET_SLOT,
  ACTION_SET_TIME,
  allows,
  courseLine,
  isOnline,
  needsLink,
  placeOf,
  sectionShort,
} from '../../../lib/schedule'
import { formatDate, formatTimeRange } from '../../../lib/time'
import cx from '../../../lib/cx'

const byDate = (items) => {
  const days = []
  const index = new Map()
  items.forEach((item) => {
    if (!index.has(item.item_date)) {
      index.set(item.item_date, days.length)
      days.push({ date: item.item_date, items: [] })
    }
    days[index.get(item.item_date)].items.push(item)
  })
  return days
}

export default function UpcomingTab({ upcoming, highlight }) {
  const toast = useToast()
  const confirm = useConfirm()
  const [busy, setBusy] = useState(0)
  const [cancelling, setCancelling] = useState(null)
  const [moving, setMoving] = useState(null)
  const [linking, setLinking] = useState(null)
  const items = upcoming.data?.items || []
  const run = async (id, action, message) => {
    setBusy(id)
    try {
      await action()
      toast.success(message)
      upcoming.reload()
    } catch (failure) {
      toast.error(failure.message)
    } finally {
      setBusy(0)
    }
  }
  const setFinal = (item, inFinal) =>
    run(
      item.id,
      () =>
        inFinal
          ? api.post(`/api/cr/items/${item.id}/final`)
          : api.delete(`/api/cr/items/${item.id}/final`),
      inFinal ? 'Added to the final routine' : 'Removed from the final routine',
    )
  const remove = async (item) => {
    const ok = await confirm({
      title: 'Delete this item?',
      message: `${courseLine(item)} on ${formatDate(item.item_date)} disappears for everybody.`,
      confirmLabel: 'Delete',
    })
    if (!ok) return
    run(item.id, () => api.delete(`/api/cr/items/${item.id}`), 'Deleted')
  }
  const askReschedule = async (values) => {
    await api.post(`/api/cr/items/${moving.id}/request-reschedule`, {
      date: values.date,
      start: values.start,
      end: values.end,
      mode: values.mode,
      room: values.room,
      meeting_link: values.meeting_link,
    })
    toast.success('The teacher was asked for the new slot')
    setMoving(null)
    upcoming.reload()
  }
  const setSlot = async (values) => {
    await api.patch(`/api/cr/items/${moving.id}`, {
      item_date: values.date,
      start_time: values.start,
      end_time: values.end,
      mode: values.mode,
      room: values.room,
      meeting_link: values.meeting_link,
    })
    toast.success('The new slot is saved')
    setMoving(null)
    upcoming.reload()
  }
  const cancelItem = async (reason) => {
    await api.post(`/api/cr/items/${cancelling.id}/cancel`, { reason })
    toast.success('Cancelled')
    setCancelling(null)
    upcoming.reload()
  }
  const saveLink = async (link) => {
    await api.patch(`/api/schedule/items/${linking.id}/link`, { meeting_link: link })
    toast.success('Meeting link updated')
    setLinking(null)
    upcoming.reload()
  }
  const [asking, setAsking] = useState(false)
  const openMove = (item, ask) => {
    setAsking(ask)
    setMoving(item)
  }
  if (upcoming.loading) return <SkeletonCard />
  if (upcoming.error) return <Alert tone="error">{upcoming.error}</Alert>
  if (items.length === 0) {
    return (
      <EmptyState
        icon={CalendarClock}
        title="Nothing ahead"
        text="Items show up here as soon as the routine for the coming days is generated."
      />
    )
  }
  return (
    <>
      <div className="flex flex-col gap-4">
        {byDate(items).map((day) => (
          <Card key={day.date} hover={false} className="animate-fade-in-up">
            <p className="font-display text-lg font-bold text-ink">
              {formatDate(day.date)}
            </p>
            <ul className="mt-2 flex flex-col gap-3">
              {day.items.map((item) => (
                <li
                  key={item.id}
                  ref={highlight.register(item.id)}
                  className={cx(
                    'rounded-xl border border-line p-3 transition-colors',
                    highlight.highlighted(item.id) && 'border-accent-300 bg-accent-50/60',
                  )}
                >
                  <div className="flex flex-wrap items-center gap-x-3 gap-y-1.5">
                    <span className="font-medium tabular-nums text-brand-600">
                      {formatTimeRange(item.start_time, item.end_time) || 'No time yet'}
                    </span>
                    <span className="font-medium text-ink">{courseLine(item)}</span>
                    <span className="text-muted">
                      {sectionShort(item.section)}
                      {placeOf(item) ? ` · ${placeOf(item)}` : ''}
                    </span>
                    <ItemBadges item={item} />
                  </div>
                  {item.teacher_message && (
                    <Alert tone="info" title="The teacher wrote" className="mt-2">
                      {item.teacher_message}
                    </Alert>
                  )}
                  {item.proposed && (
                    <p className="mt-2 text-xs text-muted">
                      Proposed: {formatDate(item.proposed.item_date)}{' '}
                      {formatTimeRange(item.proposed.start_time, item.proposed.end_time)}
                      {item.proposed.room ? ` · ${item.proposed.room}` : ''}
                    </p>
                  )}
                  {item.cancel_reason && (
                    <p className="mt-2 text-xs text-muted">Reason: {item.cancel_reason}</p>
                  )}
                  <div className="mt-2.5 flex flex-wrap gap-2">
                    {allows(item, ACTION_ADD_TO_FINAL) && (
                      <Button
                        size="sm"
                        loading={busy === item.id}
                        onClick={() => setFinal(item, true)}
                      >
                        Add to final
                      </Button>
                    )}
                    {allows(item, ACTION_REMOVE_FROM_FINAL) && (
                      <Button
                        size="sm"
                        variant="secondary"
                        loading={busy === item.id}
                        onClick={() => setFinal(item, false)}
                      >
                        Remove from final
                      </Button>
                    )}
                    {allows(item, ACTION_REQUEST_RESCHEDULE) && (
                      <Button size="sm" variant="secondary" onClick={() => openMove(item, true)}>
                        Request reschedule
                      </Button>
                    )}
                    {allows(item, ACTION_SET_SLOT) && (
                      <Button size="sm" onClick={() => openMove(item, false)}>
                        Set new slot
                      </Button>
                    )}
                    {(allows(item, ACTION_EDIT) || allows(item, ACTION_SET_TIME)) && (
                      <Button size="sm" variant="secondary" onClick={() => openMove(item, false)}>
                        <Pencil size={14} />
                        {allows(item, ACTION_SET_TIME) ? 'Set time' : 'Edit'}
                      </Button>
                    )}
                    {isOnline(item) && item.status !== 'cancelled' && (
                      <Button size="sm" variant="secondary" onClick={() => setLinking(item)}>
                        <Link2 size={14} />
                        {needsLink(item) ? 'Add link' : 'Edit link'}
                      </Button>
                    )}
                    {allows(item, ACTION_CANCEL) && (
                      <Button size="sm" variant="danger" onClick={() => setCancelling(item)}>
                        Cancel
                      </Button>
                    )}
                    {allows(item, ACTION_DELETE) && (
                      <Button
                        size="sm"
                        variant="danger"
                        loading={busy === item.id}
                        onClick={() => remove(item)}
                      >
                        <Trash2 size={14} />
                        Delete
                      </Button>
                    )}
                  </div>
                </li>
              ))}
            </ul>
          </Card>
        ))}
      </div>
      <RescheduleModal
        item={moving}
        title={asking ? 'Ask the teacher for another slot' : 'Set the new slot'}
        taken={items.filter((entry) => entry.id !== moving?.id && entry.status !== 'cancelled')}
        onClose={() => setMoving(null)}
        onConfirm={asking ? askReschedule : setSlot}
      />
      <CancelModal item={cancelling} onClose={() => setCancelling(null)} onConfirm={cancelItem} />
      <LinkModal item={linking} onClose={() => setLinking(null)} onConfirm={saveLink} />
    </>
  )
}

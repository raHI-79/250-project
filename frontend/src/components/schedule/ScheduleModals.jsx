import { useEffect, useState } from 'react'
import { Alert, Button, Input, Modal, Textarea } from '../ui'
import SlotFields from './SlotFields'
import TimeRangeFields from '../forms/TimeRangeFields'
import { MODE_ROOM, courseLine, periodRangeError, slotError } from '../../lib/schedule'
import { usePeriods } from '../../lib/hooks'
import { formatDate, formatTimeRange, today } from '../../lib/time'

const emptySlot = { mode: MODE_ROOM, room: '', meeting_link: '' }

export function CancelModal({ item, onClose, onConfirm, busy }) {
  const [reason, setReason] = useState('')
  const [problem, setProblem] = useState('')
  useEffect(() => {
    setReason('')
    setProblem('')
  }, [item])
  const submit = async () => {
    setProblem('')
    try {
      await onConfirm(reason.trim())
    } catch (failure) {
      setProblem(failure.message)
    }
  }
  return (
    <Modal
      open={Boolean(item)}
      onClose={onClose}
      title="Cancel this class?"
      footer={
        <>
          <Button variant="secondary" onClick={onClose}>
            Keep it
          </Button>
          <Button variant="danger" loading={busy} onClick={submit}>
            Cancel the class
          </Button>
        </>
      }
    >
      <div className="flex flex-col gap-4">
        {problem && <Alert tone="error">{problem}</Alert>}
        <p className="text-sm text-brand-600">
          {item ? `${courseLine(item)} on ${formatDate(item.item_date)}` : ''}
        </p>
        <Textarea
          label="Reason"
          value={reason}
          onChange={(event) => setReason(event.target.value)}
          rows={3}
          hint="Optional, everybody who sees the class reads it"
        />
      </div>
    </Modal>
  )
}

export function MessageModal({ item, onClose, onConfirm, busy }) {
  const [message, setMessage] = useState('')
  const [problem, setProblem] = useState('')
  useEffect(() => {
    setMessage('')
    setProblem('')
  }, [item])
  const submit = async () => {
    setProblem('')
    if (!message.trim()) {
      setProblem('Write a short message for the class representative.')
      return
    }
    try {
      await onConfirm(message.trim())
    } catch (failure) {
      setProblem(failure.message)
    }
  }
  return (
    <Modal
      open={Boolean(item)}
      onClose={onClose}
      title="Ask for another slot"
      footer={
        <>
          <Button variant="secondary" onClick={onClose}>
            Cancel
          </Button>
          <Button loading={busy} onClick={submit}>
            Send the request
          </Button>
        </>
      }
    >
      <div className="flex flex-col gap-4">
        {problem && <Alert tone="error">{problem}</Alert>}
        <Alert tone="info">
          The class representative picks the new slot and the class moves without another approval.
        </Alert>
        <Textarea
          label="Message"
          value={message}
          onChange={(event) => setMessage(event.target.value)}
          rows={3}
          placeholder="I am at a conference that morning."
          required
        />
      </div>
    </Modal>
  )
}

export function ConfirmSlotModal({ item, onClose, onConfirm, busy }) {
  const [slot, setSlot] = useState(emptySlot)
  const [problem, setProblem] = useState('')
  useEffect(() => {
    if (!item) return
    setSlot({
      mode: item.mode || MODE_ROOM,
      room: item.room || '',
      meeting_link: item.meeting_link || '',
    })
    setProblem('')
  }, [item])
  const submit = async () => {
    const found = slotError(slot.mode, slot.room, slot.meeting_link)
    setProblem(found)
    if (found) return
    try {
      await onConfirm(slot)
    } catch (failure) {
      setProblem(failure.message)
    }
  }
  return (
    <Modal
      open={Boolean(item)}
      onClose={onClose}
      title="Confirm this class"
      footer={
        <>
          <Button variant="secondary" onClick={onClose}>
            Cancel
          </Button>
          <Button loading={busy} onClick={submit}>
            Confirm
          </Button>
        </>
      }
    >
      <div className="flex flex-col gap-4">
        {problem && <Alert tone="error">{problem}</Alert>}
        {item && (
          <p className="text-sm text-brand-600">
            {courseLine(item)} · {item.batch} {item.section} ·{' '}
            {formatTimeRange(item.start_time, item.end_time)}
          </p>
        )}
        <SlotFields value={slot} onChange={setSlot} />
      </div>
    </Modal>
  )
}

export function LinkModal({ item, onClose, onConfirm, busy }) {
  const [link, setLink] = useState('')
  const [problem, setProblem] = useState('')
  useEffect(() => {
    setLink(item?.meeting_link || '')
    setProblem('')
  }, [item])
  const submit = async () => {
    if (!/^https?:\/\/\S+$/i.test(link.trim())) {
      setProblem('The meeting link must start with http:// or https://.')
      return
    }
    setProblem('')
    try {
      await onConfirm(link.trim())
    } catch (failure) {
      setProblem(failure.message)
    }
  }
  return (
    <Modal
      open={Boolean(item)}
      onClose={onClose}
      title="Meeting link"
      footer={
        <>
          <Button variant="secondary" onClick={onClose}>
            Cancel
          </Button>
          <Button loading={busy} onClick={submit}>
            Save the link
          </Button>
        </>
      }
    >
      <div className="flex flex-col gap-4">
        {problem && <Alert tone="error">{problem}</Alert>}
        <Alert tone="info">
          Students of a published class see the new link straight away.
        </Alert>
        <Input
          label="Meeting link"
          type="url"
          value={link}
          onChange={(event) => setLink(event.target.value)}
          placeholder="https://meet.example.com/your-class"
          required
        />
      </div>
    </Modal>
  )
}

export function RescheduleModal({ item, title, taken, onClose, onConfirm, busy }) {
  const periods = usePeriods()
  const [form, setForm] = useState({ date: today(), start: '', end: '' })
  const [slot, setSlot] = useState(emptySlot)
  const [problem, setProblem] = useState('')
  useEffect(() => {
    if (!item) return
    setForm({
      date: item.item_date || today(),
      start: item.start_time || '',
      end: item.end_time || '',
    })
    setSlot({
      mode: item.mode || MODE_ROOM,
      room: item.room || '',
      meeting_link: item.meeting_link || '',
    })
    setProblem('')
  }, [item])
  const timeProblem = periodRangeError(periods, form.start, form.end)
  const clashes = (taken || []).filter(
    (entry) =>
      entry.item_date === form.date &&
      form.start &&
      form.end &&
      entry.start_time < form.end &&
      form.start < entry.end_time,
  )
  const submit = async () => {
    const found = timeProblem || slotError(slot.mode, slot.room, slot.meeting_link)
    setProblem(found)
    if (found) return
    try {
      await onConfirm({ ...form, ...slot })
    } catch (failure) {
      setProblem(failure.message)
    }
  }
  return (
    <Modal
      open={Boolean(item)}
      onClose={onClose}
      title={title}
      className="max-w-xl"
      footer={
        <>
          <Button variant="secondary" onClick={onClose}>
            Cancel
          </Button>
          <Button loading={busy} onClick={submit}>
            Save the slot
          </Button>
        </>
      }
    >
      <div className="flex flex-col gap-4">
        {problem && <Alert tone="error">{problem}</Alert>}
        {item?.teacher_message && (
          <Alert tone="info" title="The teacher wrote">
            {item.teacher_message}
          </Alert>
        )}
        <Input
          label="Date"
          type="date"
          value={form.date}
          min={today()}
          onChange={(event) => setForm((current) => ({ ...current, date: event.target.value }))}
          required
        />
        <TimeRangeFields
          periods={periods}
          start={form.start}
          end={form.end}
          onStartChange={(value) => setForm((current) => ({ ...current, start: value }))}
          onEndChange={(value) => setForm((current) => ({ ...current, end: value }))}
          error={timeProblem}
        />
        <SlotFields value={slot} onChange={setSlot} />
        {clashes.length > 0 ? (
          <Alert tone="warning" title="Your batch is busy then">
            <ul className="flex flex-col gap-0.5">
              {clashes.map((entry) => (
                <li key={`${entry.start_time}-${entry.course_code}`}>
                  {formatTimeRange(entry.start_time, entry.end_time)} · {entry.course_code}
                </li>
              ))}
            </ul>
          </Alert>
        ) : (
          form.start &&
          form.end &&
          !timeProblem && (
            <Alert tone="success">
              Nothing else of your batch sits in {formatTimeRange(form.start, form.end)} that day.
            </Alert>
          )
        )}
      </div>
    </Modal>
  )
}

import { useEffect, useState } from 'react'
import { CalendarPlus } from 'lucide-react'
import SlotFields from '../../../components/schedule/SlotFields'
import TimeRangeFields from '../../../components/forms/TimeRangeFields'
import { Alert, Button, Card, Input, Select, SkeletonLines, Textarea, useToast } from '../../../components/ui'
import api from '../../../lib/api'
import { MODE_ROOM, SECTION_ALL, sectionLabel, slotError } from '../../../lib/schedule'
import { timeRangeError, today } from '../../../lib/time'

const EMPTY = {
  kind: 'class',
  course_id: '',
  section: '',
  item_date: today(),
  start: '',
  end: '',
  syllabus: '',
  instructions: '',
  note: '',
}

const weekdayOf = (date) => new Date(`${date}T00:00:00`).getDay()

const slotFor = (course, date) => {
  if (!course || !date) return null
  const day = weekdayOf(date)
  return (course.slots || []).find((entry) => entry.day_of_week === day) || null
}

export default function CreateTab({ courses, ownSection, prefill, onCreated }) {
  const toast = useToast()
  const [form, setForm] = useState({ ...EMPTY, section: ownSection || '' })
  const [slot, setSlot] = useState({ mode: MODE_ROOM, room: '', meeting_link: '' })
  const [error, setError] = useState('')
  const [saving, setSaving] = useState(false)
  useEffect(() => {
    if (!prefill) return
    setForm((current) => ({
      ...current,
      item_date: prefill.date || current.item_date,
      start: prefill.start || current.start,
      end: prefill.end || current.end,
    }))
    setSlot((current) => ({ ...current, mode: MODE_ROOM, room: prefill.room || current.room }))
  }, [prefill])
  const items = courses.data?.items || []
  const course = items.find((entry) => String(entry.id) === form.course_id) || null
  const routineSlot = slotFor(course, form.item_date)
  const set = (key) => (event) => setForm((current) => ({ ...current, [key]: event.target.value }))
  const timeProblem = timeRangeError(form.start, form.end)
  const useRoutineSlot = () => {
    if (!routineSlot) return
    setForm((current) => ({
      ...current,
      start: routineSlot.start_time,
      end: routineSlot.end_time,
    }))
    setSlot((current) => ({ ...current, mode: MODE_ROOM, room: routineSlot.room }))
  }
  const submit = async (event) => {
    event.preventDefault()
    setError('')
    if (!form.course_id) {
      setError('Pick a course first.')
      return
    }
    const found = timeProblem || slotError(slot.mode, slot.room, slot.meeting_link)
    if (found) {
      setError(found)
      return
    }
    setSaving(true)
    try {
      await api.post('/api/cr/items', {
        kind: form.kind,
        course_id: Number(form.course_id),
        section: form.section || ownSection,
        item_date: form.item_date,
        start_time: form.start,
        end_time: form.end,
        mode: slot.mode,
        room: slot.room,
        meeting_link: slot.meeting_link,
        syllabus: form.kind === 'exam' ? form.syllabus : '',
        instructions: form.kind === 'exam' ? form.instructions : '',
        note: form.note,
      })
      toast.success(form.kind === 'exam' ? 'Exam added' : 'Class added')
      setForm({ ...EMPTY, section: ownSection || '', item_date: form.item_date })
      setSlot({ mode: MODE_ROOM, room: '', meeting_link: '' })
      onCreated()
    } catch (failure) {
      setError(failure.message)
    } finally {
      setSaving(false)
    }
  }
  if (courses.loading) return <SkeletonLines lines={6} />
  return (
    <Card hover={false} className="animate-fade-in">
      <form onSubmit={submit} className="flex flex-col gap-4">
        {error && <Alert tone="error">{error}</Alert>}
        <Alert tone="info">
          What you add here needs no approval. It reaches your batch once you put it in the final
          routine and publish.
        </Alert>
        <div className="grid gap-4 sm:grid-cols-2">
          <Select label="Kind" value={form.kind} onChange={set('kind')}>
            <option value="class">Class</option>
            <option value="exam">Exam</option>
          </Select>
          <Select label="Course" value={form.course_id} onChange={set('course_id')} required>
            <option value="">Choose a course</option>
            {items.map((entry) => (
              <option key={entry.id} value={entry.id}>
                {entry.code} {entry.title}
              </option>
            ))}
          </Select>
          <Select label="Section" value={form.section} onChange={set('section')}>
            {ownSection && <option value={ownSection}>{sectionLabel(ownSection)}</option>}
            <option value={SECTION_ALL}>{sectionLabel(SECTION_ALL)}</option>
          </Select>
          <Input
            label="Date"
            type="date"
            value={form.item_date}
            min={today()}
            onChange={set('item_date')}
            required
          />
        </div>
        {course?.teacher?.name && (
          <p className="text-xs text-muted">
            {course.code} is taught by {course.teacher.name}.
            {routineSlot && (
              <Button variant="ghost" size="sm" onClick={useRoutineSlot} className="ml-1 align-baseline">
                Use the routine slot
              </Button>
            )}
          </p>
        )}
        <TimeRangeFields
          start={form.start}
          end={form.end}
          onStartChange={(value) => setForm((current) => ({ ...current, start: value }))}
          onEndChange={(value) => setForm((current) => ({ ...current, end: value }))}
          error={timeProblem}
        />
        <SlotFields value={slot} onChange={setSlot} />
        {form.kind === 'exam' && (
          <>
            <Textarea
              label="Syllabus"
              value={form.syllabus}
              onChange={set('syllabus')}
              rows={3}
              hint="What the exam covers"
            />
            <Textarea
              label="Instructions"
              value={form.instructions}
              onChange={set('instructions')}
              rows={2}
              hint="What to bring, where to sit"
            />
          </>
        )}
        <Textarea
          label="Extra information"
          value={form.note}
          onChange={set('note')}
          rows={2}
          hint="Shown to your batch with the class"
        />
        <Button type="submit" loading={saving} className="self-start">
          <CalendarPlus size={15} />
          Add to the batch schedule
        </Button>
      </form>
    </Card>
  )
}

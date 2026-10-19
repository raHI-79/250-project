import { Fragment, useEffect, useState } from 'react'
import { CalendarCheck, ClipboardList } from 'lucide-react'
import DateChips from '../../components/DateChips'
import NoteFields from '../../components/forms/NoteFields'
import NoteLine from '../../components/NoteLine'
import TimeRangeFields from '../../components/forms/TimeRangeFields'
import {
  Alert,
  Button,
  Card,
  DataTable,
  EmptyState,
  Input,
  PageHeader,
  Select,
  SkeletonLines,
  SkeletonTable,
  StatusBadge,
  useToast,
} from '../../components/ui'
import api from '../../lib/api'
import { useResource } from '../../lib/hooks'
import { noteError } from '../../lib/options'
import { formatDate, formatTimeRange, overlaps, timeRangeError, tomorrow } from '../../lib/time'

const EMPTY = { eventType: 'class', courseId: '', teacherId: '', start: '', end: '', room: '', note: '', audience: '' }

const KIND_LABELS = { class: 'Class', lab: 'Lab', exam: 'Exam', extra: 'Extra class' }

const REQUEST_HEADINGS = ['Type', 'Course', 'Date', 'Time', 'Room', 'Status']

const weekdayOf = (date) => new Date(`${date}T00:00:00`).getDay()

const slotFor = (course, date) => {
  if (!course || !date) return null
  const day = weekdayOf(date)
  return (course.slots || []).find((slot) => slot.day_of_week === day) || null
}

const sameRoom = (left, right) =>
  left.trim() !== '' && left.trim().toLowerCase() === right.trim().toLowerCase()

export default function CreateClassPage() {
  const toast = useToast()
  const [date, setDate] = useState(tomorrow())
  const [form, setForm] = useState(EMPTY)
  const [filled, setFilled] = useState(false)
  const [error, setError] = useState('')
  const [sending, setSending] = useState(false)
  const courses = useResource('/api/cr/courses')
  const teachers = useResource('/api/lookups/teachers')
  const day = useResource('/api/cr/day', { date })
  const requests = useResource('/api/cr/requests')
  const bookings = useResource('/api/cr/room-availability', { date })
  const set = (key) => (event) => setForm((current) => ({ ...current, [key]: event.target.value }))
  const setValue = (key) => (value) => setForm((current) => ({ ...current, [key]: value }))
  const course = (courses.data?.items || []).find((entry) => String(entry.id) === form.courseId)
  const slot = slotFor(course, date)
  useEffect(() => {
    if (!course) {
      setFilled(false)
      return
    }
    const match = slotFor(course, date)
    setFilled(Boolean(match))
    setForm((current) => ({
      ...current,
      teacherId: String(course.teacher.id),
      start: match ? match.start_time : current.start,
      end: match ? match.end_time : current.end,
      room: match ? match.room : current.room,
    }))
  }, [course, date])
  const timeError = timeRangeError(form.start, form.end)
  const clashes = (bookings.data?.items || []).filter(
    (item) =>
      Boolean(form.room && form.start && form.end) &&
      sameRoom(item.room, form.room) &&
      overlaps(form.start, form.end, item.start_time, item.end_time),
  )
  const onSubmit = async (event) => {
    event.preventDefault()
    const noteProblem = noteError(form.note, form.audience)
    if (noteProblem) {
      setError(noteProblem)
      return
    }
    setError('')
    if (timeError) return
    setSending(true)
    try {
      await api.post('/api/cr/class-requests', {
        event_type: form.eventType,
        course_id: Number(form.courseId),
        teacher_id: Number(form.teacherId),
        event_date: date,
        start_time: form.start,
        end_time: form.end,
        room: form.room.trim(),
        note: form.note.trim(),
        note_audience: form.note.trim() ? form.audience : '',
      })
      toast.success('Request sent to the teacher for approval')
      setForm(EMPTY)
      setFilled(false)
      day.reload()
      requests.reload()
      bookings.reload()
    } catch (failure) {
      setError(failure.message)
    } finally {
      setSending(false)
    }
  }
  const onThatDay = day.data?.items || []
  const mine = requests.data?.items || []
  return (
    <>
      <PageHeader
        title="Create class / exam"
        subtitle="Ask a teacher to approve a class or an exam for your batch and section"
      />
      <DateChips value={date} onChange={setDate} label="Another day" min={tomorrow()} />
      <div className="grid gap-5 lg:grid-cols-2">
        <div className="flex flex-col gap-5">
          <Card hover={false}>
            <p className="font-display text-base font-semibold text-brand-800">
              Already on {formatDate(date)}
            </p>
            {day.error && (
              <Alert tone="error" className="mt-3">
                {day.error}
              </Alert>
            )}
            {day.loading ? (
              <SkeletonLines lines={4} className="mt-4" />
            ) : onThatDay.length === 0 ? (
              <EmptyState
                icon={CalendarCheck}
                title="The day is free"
                text="Your batch has nothing on this date yet."
                className="mt-4 border-0 py-8"
              />
            ) : (
              <ul className="mt-3 flex flex-col">
                {onThatDay.map((item, index) => (
                  <li
                    key={`${item.source}-${item.start_time}-${item.course_code}-${index}`}
                    className="flex flex-wrap items-center gap-x-3 gap-y-1 border-b border-brand-50 py-2 text-sm last:border-b-0"
                  >
                    <span className="font-medium tabular-nums text-brand-600">
                      {formatTimeRange(item.start_time, item.end_time)}
                    </span>
                    <span className="font-medium text-brand-900">{item.course_code}</span>
                    <span className="text-brand-500">
                      {item.section === 'ALL' ? 'Both sections' : `Section ${item.section}`}
                      {item.room ? ` @ ${item.room}` : ''}
                    </span>
                    <span className="text-xs text-brand-400">
                      {item.source === 'fixed' ? 'Fixed routine' : KIND_LABELS[item.kind] || item.kind}
                    </span>
                    <StatusBadge status={item.status} className="ml-auto" />
                  </li>
                ))}
              </ul>
            )}
          </Card>
          <Card hover={false}>
            <p className="font-display text-base font-semibold text-brand-800">My requests</p>
            {requests.error && (
              <Alert tone="error" className="mt-3">
                {requests.error}
              </Alert>
            )}
            {requests.loading ? (
              <SkeletonTable columns={6} className="mt-4" />
            ) : mine.length === 0 ? (
              <EmptyState
                icon={ClipboardList}
                title="No requests"
                text="What you ask for shows up here with its status."
                className="mt-4 border-0 py-8"
              />
            ) : (
              <DataTable className="mt-3">
                <thead>
                  <tr className="border-b border-brand-100 bg-brand-50/60">
                    {REQUEST_HEADINGS.map((heading) => (
                      <th key={heading} className="px-3 py-2.5 font-semibold text-brand-700">
                        {heading}
                      </th>
                    ))}
                  </tr>
                </thead>
                <tbody>
                  {mine.map((item) => (
                    <Fragment key={item.id}>
                      <tr className="border-b border-brand-50 last:border-b-0">
                        <td className="px-3 py-2.5 text-brand-700">
                          {KIND_LABELS[item.event_type] || item.event_type}
                        </td>
                        <td className="px-3 py-2.5 font-medium text-brand-900">{item.course_code}</td>
                        <td className="px-3 py-2.5 whitespace-nowrap text-brand-700">
                          {formatDate(item.event_date)}
                        </td>
                        <td className="px-3 py-2.5 whitespace-nowrap tabular-nums text-brand-600">
                          {formatTimeRange(item.start_time, item.end_time)}
                        </td>
                        <td className="px-3 py-2.5 text-brand-700">{item.room}</td>
                        <td className="px-3 py-2.5">
                          <StatusBadge status={item.status} />
                        </td>
                      </tr>
                      {item.note && (
                        <tr className="border-b border-brand-50 last:border-b-0">
                          <td colSpan={REQUEST_HEADINGS.length} className="px-3 pb-2.5">
                            <NoteLine text={item.note} audience={item.note_audience} />
                          </td>
                        </tr>
                      )}
                    </Fragment>
                  ))}
                </tbody>
              </DataTable>
            )}
          </Card>
        </div>
        <Card hover={false}>
          <form onSubmit={onSubmit} className="flex flex-col gap-4">
            {error && <Alert tone="error">{error}</Alert>}
            {clashes.length > 0 && (
              <Alert tone="warning" title="This room is taken">
                {clashes.length} booking{clashes.length === 1 ? '' : 's'} overlap room {form.room} at
                that time. Pick another room or another time before you send the request.
              </Alert>
            )}
            <Select label="Type" value={form.eventType} onChange={set('eventType')} required>
              <option value="class">Class</option>
              <option value="exam">Exam</option>
            </Select>
            <Select
              label="Course"
              value={form.courseId}
              onChange={set('courseId')}
              hint="The running courses of your batch"
              required
            >
              <option value="">Choose a course</option>
              {(courses.data?.items || []).map((entry) => (
                <option key={entry.id} value={String(entry.id)}>
                  {entry.code} {entry.title}
                </option>
              ))}
            </Select>
            <Select label="Teacher" value={form.teacherId} onChange={set('teacherId')} required>
              <option value="">Choose a teacher</option>
              {(teachers.data?.items || []).map((entry) => (
                <option key={entry.id} value={String(entry.id)}>
                  {entry.name}
                </option>
              ))}
            </Select>
            {filled && slot && (
              <p className="text-xs text-brand-500">
                Filled from the fixed routine ({slot.day} {formatTimeRange(slot.start_time, slot.end_time)}{' '}
                @ {slot.room}). Change anything you need.
              </p>
            )}
            <TimeRangeFields
              start={form.start}
              end={form.end}
              onStartChange={setValue('start')}
              onEndChange={setValue('end')}
              error={timeError}
            />
            <Input label="Room" value={form.room} onChange={set('room')} placeholder="1003" required />
            <NoteFields
              note={form.note}
              audience={form.audience}
              onNoteChange={setValue('note')}
              onAudienceChange={setValue('audience')}
              error={noteError(form.note, form.audience)}
            />
            <Button type="submit" loading={sending} className="w-full">
              Send the request
            </Button>
          </form>
        </Card>
      </div>
    </>
  )
}

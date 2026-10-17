import { useCallback, useState } from 'react'
import { Alert, Button, Card, Input, PageHeader, Select, useToast } from '../../components/ui'
import RoomAvailabilityPanel from '../../components/forms/RoomAvailabilityPanel'
import TimeRangeFields from '../../components/forms/TimeRangeFields'
import { useAuth } from '../../context/auth'
import api from '../../lib/api'
import { useResource } from '../../lib/hooks'
import { timeRangeError, tomorrow } from '../../lib/time'

const EMPTY = { courseId: '', teacherId: '', date: '', start: '', end: '', room: '' }

export default function ExtraClassPage() {
  const toast = useToast()
  const { user } = useAuth()
  const [form, setForm] = useState(EMPTY)
  const [error, setError] = useState('')
  const [sending, setSending] = useState(false)
  const [status, setStatus] = useState({ loaded: false, conflicts: 0 })
  const courses = useResource('/api/lookups/courses')
  const teachers = useResource('/api/lookups/teachers')
  const set = (key) => (event) => setForm((current) => ({ ...current, [key]: event.target.value }))
  const setValue = (key) => (value) => setForm((current) => ({ ...current, [key]: value }))
  const onStatus = useCallback((next) => setStatus(next), [])
  const timeError = timeRangeError(form.start, form.end)
  const onSubmit = async (event) => {
    event.preventDefault()
    setError('')
    if (timeError) return
    setSending(true)
    try {
      await api.post('/api/cr/class-requests', {
        event_type: 'extra',
        course_id: Number(form.courseId),
        teacher_id: Number(form.teacherId),
        event_date: form.date,
        start_time: form.start,
        end_time: form.end,
        room: form.room.trim(),
      })
      toast.success('Request sent to the teacher for approval')
      setForm(EMPTY)
    } catch (failure) {
      setError(failure.message)
    } finally {
      setSending(false)
    }
  }
  return (
    <>
      <PageHeader
        title="Extra Class"
        subtitle="Request an extra class outside the fixed routine, once the room is free"
      />
      <div className="grid gap-5 lg:grid-cols-2">
        <Card hover={false}>
          <form onSubmit={onSubmit} className="flex flex-col gap-4">
            {error && <Alert tone="error">{error}</Alert>}
            {status.conflicts > 0 && (
              <Alert tone="warning">
                The room you chose is already booked at that time. Pick another room or another
                time.
              </Alert>
            )}
            <div className="grid gap-4 sm:grid-cols-2">
              <Input label="Batch" value={user?.batch || ''} readOnly disabled />
              <Input label="Section" value={user?.section || ''} readOnly disabled />
            </div>
            <Select label="Course" value={form.courseId} onChange={set('courseId')} required>
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
            <Input
              label="Date"
              type="date"
              value={form.date}
              min={tomorrow()}
              onChange={set('date')}
              hint="Tomorrow or later"
              required
            />
            <TimeRangeFields
              start={form.start}
              end={form.end}
              onStartChange={setValue('start')}
              onEndChange={setValue('end')}
              error={timeError}
            />
            <Input
              label="Room"
              value={form.room}
              onChange={set('room')}
              placeholder="1003"
              required
            />
            <Button type="submit" loading={sending} disabled={!status.loaded} className="w-full">
              Send the request
            </Button>
            {!status.loaded && (
              <p className="text-center text-xs text-brand-500">
                Choose a date and wait for the room availability to load.
              </p>
            )}
          </form>
        </Card>
        <RoomAvailabilityPanel
          date={form.date}
          room={form.room}
          start={form.start}
          end={form.end}
          onStatus={onStatus}
        />
      </div>
    </>
  )
}

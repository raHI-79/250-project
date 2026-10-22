import { useState } from 'react'
import { CalendarDays, Plus } from 'lucide-react'
import {
  Alert,
  Button,
  Card,
  EmptyState,
  Input,
  PageHeader,
  Select,
  SkeletonCard,
  StatusBadge,
  useConfirm,
  useToast,
} from '../../components/ui'
import TimeRangeFields from '../../components/forms/TimeRangeFields'
import api from '../../lib/api'
import { useResource } from '../../lib/hooks'
import { WEEKDAYS, formatTimeRange, timeRangeError } from '../../lib/time'

const SECTIONS = ['A', 'B', 'C', 'D']

const TYPE_LABELS = { class: 'Class', lab: 'Lab' }

const EMPTY = {
  batch: '',
  section: '',
  courseId: '',
  teacherId: '',
  day: '0',
  start: '',
  end: '',
  room: '',
  type: 'class',
}

const groupRoutines = (items) => {
  const batches = []
  items.forEach((item) => {
    let batch = batches.find((entry) => entry.batch === item.batch)
    if (!batch) {
      batch = { batch: item.batch, sections: [] }
      batches.push(batch)
    }
    let section = batch.sections.find((entry) => entry.section === item.section)
    if (!section) {
      section = { section: item.section, days: [] }
      batch.sections.push(section)
    }
    let day = section.days.find((entry) => entry.index === item.day_of_week)
    if (!day) {
      day = { index: item.day_of_week, items: [] }
      section.days.push(day)
    }
    day.items.push(item)
  })
  batches.forEach((batch) =>
    batch.sections.forEach((section) => {
      section.days.sort((left, right) => left.index - right.index)
      section.days.forEach((day) =>
        day.items.sort((left, right) => left.start_time.localeCompare(right.start_time)),
      )
    }),
  )
  return batches
}

export default function AdminRoutinesPage() {
  const toast = useToast()
  const confirm = useConfirm()
  const [form, setForm] = useState(EMPTY)
  const [error, setError] = useState('')
  const [saving, setSaving] = useState(false)
  const [busy, setBusy] = useState(0)
  const [batch, setBatch] = useState('')
  const [section, setSection] = useState('')
  const batches = useResource('/api/lookups/batches')
  const courses = useResource('/api/lookups/courses')
  const teachers = useResource('/api/lookups/teachers')
  const query = {}
  if (batch) query.batch = batch
  if (section) query.section = section
  const routines = useResource('/api/admin/routines', query)
  const set = (key) => (event) => setForm((current) => ({ ...current, [key]: event.target.value }))
  const setValue = (key) => (value) => setForm((current) => ({ ...current, [key]: value }))
  const timeError = timeRangeError(form.start, form.end)
  const save = async (event) => {
    event.preventDefault()
    setError('')
    if (timeError) return
    setSaving(true)
    try {
      await api.post('/api/admin/routines', {
        batch: form.batch.trim(),
        section: form.section,
        course_id: Number(form.courseId),
        teacher_id: Number(form.teacherId),
        day_of_week: Number(form.day),
        start_time: form.start,
        end_time: form.end,
        room: form.room.trim(),
        type: form.type,
      })
      toast.success('Routine entry added')
      setForm((current) => ({ ...current, courseId: '', start: '', end: '', room: '' }))
      routines.reload()
    } catch (failure) {
      setError(failure.message)
    } finally {
      setSaving(false)
    }
  }
  const remove = async (item) => {
    const ok = await confirm({
      title: 'Delete this entry?',
      message: `${item.course_code} on ${WEEKDAYS[item.day_of_week]} at ${formatTimeRange(item.start_time, item.end_time)} will be removed from the routine.`,
      confirmLabel: 'Delete',
    })
    if (!ok) return
    setBusy(item.id)
    try {
      await api.delete(`/api/admin/routines/${item.id}`)
      toast.success('Routine entry deleted')
      routines.reload()
    } catch (failure) {
      toast.error(failure.message)
    } finally {
      setBusy(0)
    }
  }
  const items = routines.data?.items || []
  const groups = groupRoutines(items)
  return (
    <>
      <PageHeader eyebrow="Weekly routine" title="Routines" subtitle="The fixed weekly routine of every batch" />
      <Card hover={false} className="mb-6">
        <form onSubmit={save} className="flex flex-col gap-4">
          {error && <Alert tone="error">{error}</Alert>}
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
            <Select label="Batch" value={form.batch} onChange={set('batch')} required>
              <option value="">Choose a batch</option>
              {(batches.data?.items || []).map((entry) => (
                <option key={entry} value={entry}>
                  {entry}
                </option>
              ))}
            </Select>
            <Select label="Section" value={form.section} onChange={set('section')} required>
              <option value="">Choose a section</option>
              {SECTIONS.map((entry) => (
                <option key={entry} value={entry}>
                  Section {entry}
                </option>
              ))}
              <option value="ALL">Both sections</option>
            </Select>
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
            <Select label="Day" value={form.day} onChange={set('day')} required>
              {WEEKDAYS.map((name, index) => (
                <option key={name} value={String(index)}>
                  {name}
                </option>
              ))}
            </Select>
            <Select label="Type" value={form.type} onChange={set('type')}>
              <option value="class">Class</option>
              <option value="lab">Lab</option>
            </Select>
          </div>
          <TimeRangeFields
            start={form.start}
            end={form.end}
            onStartChange={setValue('start')}
            onEndChange={setValue('end')}
            error={timeError}
          />
          <div className="grid gap-4 sm:grid-cols-[1fr_auto] sm:items-end">
            <Input
              label="Room"
              value={form.room}
              onChange={set('room')}
              placeholder="1003"
              required
            />
            <Button type="submit" loading={saving}>
              <Plus size={15} />
              Add the entry
            </Button>
          </div>
        </form>
      </Card>
      <Card hover={false} className="mb-5">
        <div className="grid gap-4 sm:grid-cols-2">
          <Select label="Batch" value={batch} onChange={(event) => setBatch(event.target.value)}>
            <option value="">Every batch</option>
            {(batches.data?.items || []).map((entry) => (
              <option key={entry} value={entry}>
                {entry}
              </option>
            ))}
          </Select>
          <Select
            label="Section"
            value={section}
            onChange={(event) => setSection(event.target.value)}
          >
            <option value="">Every section</option>
            {SECTIONS.map((entry) => (
              <option key={entry} value={entry}>
                Section {entry}
              </option>
            ))}
            <option value="ALL">Both sections only</option>
          </Select>
        </div>
      </Card>
      {routines.error && <Alert tone="error" className="mb-5">{routines.error}</Alert>}
      {routines.loading ? (
        <SkeletonCard />
      ) : items.length === 0 ? (
        <EmptyState
          icon={CalendarDays}
          title="No routine entries"
          text="Nothing matches these filters yet."
        />
      ) : (
        <div className="flex flex-col gap-5">
          {groups.map((group) => (
            <Card key={group.batch} hover={false} className="animate-fade-in-up">
              <p className="font-display text-lg font-bold text-ink">{group.batch}</p>
              {group.sections.map((entry) => (
                <div key={entry.section} className="mt-3">
                  <p className="text-xs font-semibold tracking-wide text-muted uppercase">
                    {entry.section === 'ALL' ? 'Both sections' : `Section ${entry.section}`}
                  </p>
                  {entry.days.map((day) => (
                    <div key={day.index} className="mt-2">
                      <p className="text-sm font-medium text-ink-soft">{WEEKDAYS[day.index]}</p>
                      <ul className="flex flex-col">
                        {day.items.map((item) => (
                          <li
                            key={item.id}
                            className="flex flex-wrap items-center gap-x-3 gap-y-1 border-b border-line py-2 text-sm last:border-b-0"
                          >
                            <span className="font-medium tabular-nums text-brand-600">
                              {formatTimeRange(item.start_time, item.end_time)}
                            </span>
                            <span className="font-medium text-ink">
                              {item.course_code}
                              {item.room ? ` @ ${item.room}` : ''}
                            </span>
                            <StatusBadge
                              status="unavailable"
                              label={TYPE_LABELS[item.type] || item.type}
                            />
                            <span className="text-muted">{item.teacher_name}</span>
                            <Button
                              size="sm"
                              variant="danger"
                              loading={busy === item.id}
                              onClick={() => remove(item)}
                              className="ml-auto"
                            >
                              Delete
                            </Button>
                          </li>
                        ))}
                      </ul>
                    </div>
                  ))}
                </div>
              ))}
            </Card>
          ))}
        </div>
      )}
    </>
  )
}

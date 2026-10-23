import { useState } from 'react'
import { CalendarDays, Plus } from 'lucide-react'
import {
  Alert,
  Button,
  Card,
  EmptyState,
  Input,
  Modal,
  PageHeader,
  Select,
  SkeletonCard,
  StatusBadge,
  useConfirm,
  useToast,
} from '../../components/ui'
import TimeRangeFields from '../../components/forms/TimeRangeFields'
import api from '../../lib/api'
import { usePeriods, useResource } from '../../lib/hooks'
import RoutineWeekGrid from '../../components/RoutineWeekGrid'
import { SECTIONS, SEMESTERS, semesterLabel, semesterOptions } from '../../lib/options'
import { periodRangeError } from '../../lib/schedule'
import { WEEKDAYS, formatTimeRange } from '../../lib/time'

const TYPE_LABELS = { class: 'Class', lab: 'Lab' }

const EMPTY = {
  batch: '',
  semester: '',
  section: '',
  courseId: '',
  teacherId: '',
  day: '0',
  start: '',
  end: '',
  room: '',
  type: 'class',
}

export default function AdminRoutinesPage() {
  const toast = useToast()
  const confirm = useConfirm()
  const [form, setForm] = useState(EMPTY)
  const [error, setError] = useState('')
  const [saving, setSaving] = useState(false)
  const [busy, setBusy] = useState(0)
  const [batch, setBatch] = useState('')
  const [semester, setSemester] = useState('')
  const [running, setRunning] = useState('')
  const [movingBatch, setMovingBatch] = useState(false)
  const batches = useResource('/api/lookups/batches')
  const semesterList = useResource('/api/admin/batches/running-semester')
  const courses = useResource('/api/lookups/courses')
  const teachers = useResource('/api/lookups/teachers')
  const query = {}
  if (batch) query.batch = batch
  if (semester) query.semester = semester
  const routines = useResource('/api/admin/routines', query)
  const set = (key) => (event) => setForm((current) => ({ ...current, [key]: event.target.value }))
  const setValue = (key) => (value) => setForm((current) => ({ ...current, [key]: value }))
  const periods = usePeriods()
  const timeError = periodRangeError(periods, form.start, form.end)
  const save = async (event) => {
    event.preventDefault()
    setError('')
    if (timeError) return
    setSaving(true)
    try {
      await api.post('/api/admin/routines', {
        batch: form.batch.trim(),
        semester: form.semester,
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
  const runningOf = (wanted) =>
    (semesterList.data?.items || []).find((entry) => entry.batch === wanted)?.semester || ''
  const runningHint = runningOf(batch)
    ? `${semesterLabel(runningOf(batch))} is running now`
    : 'No running semester set for this batch'
  const openRunning = () => {
    setRunning(runningOf(batch) || SEMESTERS[0])
    setMovingBatch(true)
  }
  const saveRunning = async () => {
    setMovingBatch(false)
    try {
      await api.put(`/api/admin/batches/${batch}/running-semester`, { semester: running })
      toast.success(`${batch} now runs ${semesterLabel(running)}`)
      semesterList.reload()
      routines.reload()
    } catch (failure) {
      toast.error(failure.message)
    }
  }
  const moveEntry = async (item, wanted) => {
    if (!wanted || wanted === item.semester) return
    setBusy(item.id)
    try {
      await api.patch(`/api/admin/routines/${item.id}`, { semester: wanted })
      toast.success(`Moved to ${semesterLabel(wanted)}`)
      routines.reload()
    } catch (failure) {
      toast.error(failure.message)
    } finally {
      setBusy(0)
    }
  }
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
            <Select label="Semester" value={form.semester} onChange={set('semester')} required>
              <option value="">Choose a semester</option>
              {SEMESTERS.map((entry) => (
                <option key={entry} value={entry}>
                  {semesterLabel(entry)}
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
            periods={periods}
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
        <div className="grid gap-4 sm:grid-cols-[1fr_1fr_auto] sm:items-end">
          <Select label="Batch" value={batch} onChange={(event) => setBatch(event.target.value)}>
            <option value="">Every batch</option>
            {(batches.data?.items || []).map((entry) => (
              <option key={entry} value={entry}>
                {entry}
              </option>
            ))}
          </Select>
          <Select
            label="Semester"
            value={semester}
            onChange={(event) => setSemester(event.target.value)}
            hint={batch ? runningHint : 'Pick a batch to see its running semester'}
          >
            <option value="">Every semester</option>
            {semesterOptions(runningOf(batch)).map((entry) => (
              <option key={entry} value={entry}>
                {semesterLabel(entry)}
                {entry === runningOf(batch) ? ' · running now' : ''}
              </option>
            ))}
          </Select>
          <Button variant="secondary" onClick={openRunning} disabled={!batch}>
            Running semester
          </Button>
        </div>
      </Card>
      {routines.error && (
        <Alert tone="error" className="mb-5">
          {routines.error}
        </Alert>
      )}
      {routines.loading ? (
        <SkeletonCard />
      ) : items.length === 0 ? (
        <EmptyState
          icon={CalendarDays}
          title="No routine entries"
          text="Nothing matches these filters yet."
        />
      ) : batch && semester ? (
        <div className="grid gap-4 lg:grid-cols-2">
          {SECTIONS.map((entry) => (
            <RoutineWeekGrid key={entry} items={items} periods={periods} section={entry} />
          ))}
        </div>
      ) : (
        <Card hover={false} className="animate-fade-in-up overflow-x-auto">
          <p className="font-display text-lg font-bold text-ink">Routine entries</p>
          <p className="mt-1 text-sm text-muted">
            Pick one batch and one semester to read sections A and B side by side.
          </p>
          <ul className="mt-3 flex flex-col">
            {items.map((item) => (
              <li
                key={item.id}
                className="flex flex-wrap items-center gap-x-3 gap-y-1 border-b border-line py-2 text-sm last:border-b-0"
              >
                <span className="font-semibold text-ink">{item.batch}</span>
                <span className="text-muted">
                  {item.section === 'ALL' ? 'Both sections' : `Section ${item.section}`}
                </span>
                <span className="text-ink-soft">{WEEKDAYS[item.day_of_week]}</span>
                <span className="font-medium tabular-nums text-brand-600">
                  {formatTimeRange(item.start_time, item.end_time)}
                </span>
                <span className="font-medium text-ink">
                  {item.course_code}
                  {item.room ? ` @ ${item.room}` : ''}
                </span>
                <StatusBadge status="unavailable" label={TYPE_LABELS[item.type] || item.type} />
                <span className="text-muted">{item.teacher_name}</span>
                <div className="ml-auto flex items-center gap-2">
                  <select
                    value={item.semester}
                    disabled={busy === item.id}
                    onChange={(event) => moveEntry(item, event.target.value)}
                    aria-label={`Semester of ${item.course_code}`}
                    className="rounded-xl border border-line-strong bg-surface px-2 py-1 text-xs text-ink"
                  >
                    {SEMESTERS.map((entry) => (
                      <option key={entry} value={entry}>
                        {semesterLabel(entry)}
                      </option>
                    ))}
                  </select>
                  <Button
                    size="sm"
                    variant="danger"
                    loading={busy === item.id}
                    onClick={() => remove(item)}
                  >
                    Delete
                  </Button>
                </div>
              </li>
            ))}
          </ul>
        </Card>
      )}
      <Modal
        open={movingBatch}
        onClose={() => setMovingBatch(false)}
        title={`Running semester of ${batch}`}
        footer={
          <>
            <Button variant="secondary" onClick={() => setMovingBatch(false)}>
              Cancel
            </Button>
            <Button onClick={saveRunning}>Save</Button>
          </>
        }
      >
        <p className="text-sm text-muted">
          Only the running semester creates dated classes, fills the course helper and counts as a
          drop course.
        </p>
        <Select
          label="Semester"
          value={running}
          onChange={(event) => setRunning(event.target.value)}
          className="mt-4"
        >
          {SEMESTERS.map((entry) => (
            <option key={entry} value={entry}>
              {semesterLabel(entry)}
            </option>
          ))}
        </Select>
      </Modal>
    </>
  )
}

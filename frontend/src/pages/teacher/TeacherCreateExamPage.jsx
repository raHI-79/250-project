import { useEffect, useState } from 'react'
import { ClipboardList, Plus, Trash2 } from 'lucide-react'
import ItemBadges from '../../components/schedule/ItemBadges'
import { CancelModal } from '../../components/schedule/ScheduleModals'
import {
  Alert,
  Button,
  Card,
  EmptyState,
  Input,
  PageHeader,
  Select,
  SkeletonCard,
  Textarea,
  useConfirm,
  useToast,
} from '../../components/ui'
import api from '../../lib/api'
import { useHighlight, useResource } from '../../lib/hooks'
import { SECTION_ALL, courseLine, placeOf, sectionLabel, sectionShort } from '../../lib/schedule'
import { formatDate, formatTimeRange, today } from '../../lib/time'
import cx from '../../lib/cx'

const EMPTY = {
  course_id: '',
  batch: '',
  section: SECTION_ALL,
  item_date: today(),
  syllabus: '',
  instructions: '',
}

export default function TeacherCreateExamPage() {
  const toast = useToast()
  const confirm = useConfirm()
  const [form, setForm] = useState(EMPTY)
  const [editing, setEditing] = useState(0)
  const [error, setError] = useState('')
  const [saving, setSaving] = useState(false)
  const [busy, setBusy] = useState(0)
  const [cancelling, setCancelling] = useState(null)
  const courses = useResource('/api/lookups/courses')
  const batches = useResource('/api/lookups/batches')
  const sections = useResource(form.batch ? '/api/lookups/sections' : null, { batch: form.batch })
  const exams = useResource('/api/teacher/exams')
  const highlight = useHighlight(Boolean(exams.data))
  const set = (key) => (event) => setForm((current) => ({ ...current, [key]: event.target.value }))
  useEffect(() => {
    const list = batches.data?.items || []
    if (!form.batch && list.length > 0) setForm((current) => ({ ...current, batch: list[0] }))
  }, [batches.data, form.batch])
  const reset = () => {
    setForm((current) => ({ ...EMPTY, batch: current.batch }))
    setEditing(0)
    setError('')
  }
  const submit = async (event) => {
    event.preventDefault()
    setError('')
    if (!form.course_id) {
      setError('Pick a course first.')
      return
    }
    setSaving(true)
    try {
      if (editing) {
        await api.patch(`/api/teacher/exams/${editing}`, {
          item_date: form.item_date,
          syllabus: form.syllabus,
          instructions: form.instructions,
        })
        toast.success('Exam updated')
      } else {
        await api.post('/api/teacher/exams', {
          course_id: Number(form.course_id),
          batch: form.batch,
          section: form.section,
          item_date: form.item_date,
          syllabus: form.syllabus,
          instructions: form.instructions,
        })
        toast.success('Exam added')
      }
      reset()
      exams.reload()
    } catch (failure) {
      setError(failure.message)
    } finally {
      setSaving(false)
    }
  }
  const startEdit = (item) => {
    setEditing(item.id)
    setForm({
      course_id: String(item.course_id),
      batch: item.batch,
      section: item.section,
      item_date: item.item_date,
      syllabus: item.syllabus || '',
      instructions: item.instructions || '',
    })
    setError('')
  }
  const cancelExam = async (reason) => {
    await api.post(`/api/teacher/exams/${cancelling.id}/cancel`, { reason })
    toast.success('Exam cancelled')
    setCancelling(null)
    exams.reload()
  }
  const remove = async (item) => {
    const ok = await confirm({
      title: 'Delete this exam?',
      message: `${courseLine(item)} on ${formatDate(item.item_date)} disappears for everybody.`,
      confirmLabel: 'Delete',
    })
    if (!ok) return
    setBusy(item.id)
    try {
      await api.delete(`/api/teacher/exams/${item.id}`)
      toast.success('Exam deleted')
      exams.reload()
    } catch (failure) {
      toast.error(failure.message)
    } finally {
      setBusy(0)
    }
  }
  const items = exams.data?.items || []
  return (
    <>
      <PageHeader
        title="Create exam"
        subtitle="File an exam with its syllabus; the class representative sets the time and the room"
      />
      <Card hover={false} className="mb-5">
        <form onSubmit={submit} className="flex flex-col gap-4">
          {error && <Alert tone="error">{error}</Alert>}
          <Alert tone="info">
            Students read the date, the syllabus and the instructions at once. The time and the room
            arrive when the batch publishes its routine.
          </Alert>
          <div className="grid gap-4 sm:grid-cols-2">
            <Select label="Course" value={form.course_id} onChange={set('course_id')} disabled={Boolean(editing)} required>
              <option value="">Choose a course</option>
              {(courses.data?.items || []).map((course) => (
                <option key={course.id} value={course.id}>
                  {course.code} {course.title}
                </option>
              ))}
            </Select>
            <Select label="Batch" value={form.batch} onChange={set('batch')} disabled={Boolean(editing)} required>
              {(batches.data?.items || []).map((batch) => (
                <option key={batch} value={batch}>
                  {batch}
                </option>
              ))}
            </Select>
            <Select label="Section" value={form.section} onChange={set('section')} disabled={Boolean(editing)}>
              <option value={SECTION_ALL}>{sectionLabel(SECTION_ALL)}</option>
              {(sections.data?.items || []).map((section) => (
                <option key={section} value={section}>
                  {sectionLabel(section)}
                </option>
              ))}
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
          <div className="flex gap-2">
            <Button type="submit" loading={saving}>
              <Plus size={15} />
              {editing ? 'Save the exam' : 'Add the exam'}
            </Button>
            {editing > 0 && (
              <Button variant="secondary" onClick={reset}>
                Cancel the edit
              </Button>
            )}
          </div>
        </form>
      </Card>
      {exams.error && (
        <Alert tone="error" className="mb-5">
          {exams.error}
        </Alert>
      )}
      {exams.loading ? (
        <SkeletonCard />
      ) : items.length === 0 ? (
        <EmptyState
          icon={ClipboardList}
          title="No exams yet"
          text="The exams you file show up here with their state."
        />
      ) : (
        <ul className="flex flex-col gap-3">
          {items.map((item) => (
            <li key={item.id} ref={highlight.register(item.id)}>
              <Card
                hover={false}
                className={cx(
                  'animate-fade-in-up',
                  highlight.highlighted(item.id) && 'border-accent-300 bg-accent-50/60',
                )}
              >
                <div className="flex flex-wrap items-center gap-x-3 gap-y-1.5">
                  <span className="text-sm font-medium text-brand-700">
                    {formatDate(item.item_date)}
                  </span>
                  <span className="font-medium tabular-nums text-brand-600">
                    {formatTimeRange(item.start_time, item.end_time) || 'Time to be announced'}
                  </span>
                  <span className="font-medium text-brand-900">{courseLine(item)}</span>
                  <span className="text-brand-500">
                    {item.batch} {sectionShort(item.section)}
                    {placeOf(item) ? ` · ${placeOf(item)}` : ''}
                  </span>
                  <ItemBadges item={item} />
                </div>
                {item.syllabus && (
                  <p className="mt-2 text-sm text-brand-600">Syllabus: {item.syllabus}</p>
                )}
                {item.instructions && (
                  <p className="mt-1 text-sm text-brand-600">{item.instructions}</p>
                )}
                <div className="mt-3 flex flex-wrap gap-2">
                  <Button size="sm" variant="secondary" onClick={() => startEdit(item)}>
                    Edit
                  </Button>
                  {item.status !== 'cancelled' && (
                    <Button size="sm" variant="danger" onClick={() => setCancelling(item)}>
                      Cancel
                    </Button>
                  )}
                  <Button
                    size="sm"
                    variant="danger"
                    loading={busy === item.id}
                    onClick={() => remove(item)}
                  >
                    <Trash2 size={14} />
                    Delete
                  </Button>
                </div>
              </Card>
            </li>
          ))}
        </ul>
      )}
      <CancelModal item={cancelling} onClose={() => setCancelling(null)} onConfirm={cancelExam} />
    </>
  )
}

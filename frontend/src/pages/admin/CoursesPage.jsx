import { useState } from 'react'
import { BookOpen, Plus, Trash2 } from 'lucide-react'
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

const MIN_CREDIT = 0.5

const MAX_CREDIT = 6

const CREDIT_STEP = 0.25

export default function CoursesPage() {
  const toast = useToast()
  const confirm = useConfirm()
  const [removing, setRemoving] = useState(0)
  const [code, setCode] = useState('')
  const [title, setTitle] = useState('')
  const [credit, setCredit] = useState('3')
  const [error, setError] = useState('')
  const [saving, setSaving] = useState(false)
  const { data, loading, error: listError, reload } = useResource('/api/admin/courses')
  const items = data?.items || []
  const save = async (event) => {
    event.preventDefault()
    setError('')
    setSaving(true)
    try {
      await api.post('/api/admin/courses', {
        code: code.trim(),
        title: title.trim(),
        credit: Number(credit),
      })
      toast.success('Course added')
      setCode('')
      setTitle('')
      setCredit('3')
      reload()
    } catch (failure) {
      setError(failure.message)
    } finally {
      setSaving(false)
    }
  }
  const remove = async (course) => {
    const ok = await confirm({
      title: 'Delete this course?',
      message: `${course.code} ${course.title} is removed from the catalogue. A course can only go when no routine entry, class, exam or drop course uses it.`,
      confirmLabel: 'Delete the course',
    })
    if (!ok) return
    setRemoving(course.id)
    try {
      await api.delete(`/api/admin/courses/${course.id}`)
      toast.success('Course deleted')
      reload()
    } catch (failure) {
      if (failure.status === 409) {
        await confirm({
          title: 'This course is still in use',
          message: failure.message,
          confirmLabel: 'Close',
          cancelLabel: null,
          tone: 'primary',
        })
        return
      }
      toast.error(failure.message)
    } finally {
      setRemoving(0)
    }
  }
  return (
    <>
      <PageHeader eyebrow="Course catalogue" title="Courses" subtitle="The course catalogue of the department" />
      <Card hover={false} className="mb-5">
        <form onSubmit={save} className="flex flex-col gap-4">
          {error && <Alert tone="error">{error}</Alert>}
          <div className="grid gap-4 sm:grid-cols-[1fr_2fr_auto_auto] sm:items-end">
            <Input
              label="Code"
              value={code}
              onChange={(event) => setCode(event.target.value)}
              placeholder="CSE 2243 7637D"
              required
            />
            <Input
              label="Title"
              value={title}
              onChange={(event) => setTitle(event.target.value)}
              placeholder="Digital Logic Design"
              required
            />
            <Input
              label="Credit"
              type="number"
              value={credit}
              onChange={(event) => setCredit(event.target.value)}
              min={MIN_CREDIT}
              max={MAX_CREDIT}
              step={CREDIT_STEP}
              hint="0.5 to 6.0"
              required
            />
            <Button type="submit" loading={saving}>
              <Plus size={15} />
              Add
            </Button>
          </div>
        </form>
      </Card>
      {listError && <Alert tone="error" className="mb-5">{listError}</Alert>}
      {loading ? (
        <SkeletonTable columns={4} />
      ) : items.length === 0 ? (
        <EmptyState icon={BookOpen} title="No courses" text="Add the first course above." />
      ) : (
        <DataTable className="animate-fade-in">
          <thead>
            <tr className="border-b border-line bg-wash">
              <th className="px-4 py-3 font-semibold text-ink-soft">Code</th>
              <th className="px-4 py-3 font-semibold text-ink-soft">Title</th>
              <th className="px-4 py-3 font-semibold text-ink-soft">Credit</th>
              <th className="px-4 py-3 text-right font-semibold text-ink-soft">Action</th>
            </tr>
          </thead>
          <tbody>
            {items.map((item) => (
              <tr key={item.id} className="border-b border-line last:border-b-0">
                <td className="px-4 py-3 font-medium text-ink">{item.code}</td>
                <td className="px-4 py-3 text-ink-soft">{item.title}</td>
                <td className="px-4 py-3 tabular-nums text-ink-soft">{item.credit?.toFixed(2)}</td>
                <td className="px-4 py-3">
                  <div className="flex flex-wrap justify-end gap-2">
                    <Button
                      variant="danger"
                      size="sm"
                      loading={removing === item.id}
                      onClick={() => remove(item)}
                    >
                      <Trash2 size={14} />
                      Delete
                    </Button>
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </DataTable>
      )}
    </>
  )
}

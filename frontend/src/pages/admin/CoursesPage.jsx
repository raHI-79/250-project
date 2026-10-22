import { useState } from 'react'
import { BookOpen, Plus } from 'lucide-react'
import {
  Alert,
  Button,
  Card,
  DataTable,
  EmptyState,
  Input,
  PageHeader,
  SkeletonTable,
  useToast,
} from '../../components/ui'
import api from '../../lib/api'
import { useResource } from '../../lib/hooks'

export default function CoursesPage() {
  const toast = useToast()
  const [code, setCode] = useState('')
  const [title, setTitle] = useState('')
  const [error, setError] = useState('')
  const [saving, setSaving] = useState(false)
  const { data, loading, error: listError, reload } = useResource('/api/admin/courses')
  const items = data?.items || []
  const save = async (event) => {
    event.preventDefault()
    setError('')
    setSaving(true)
    try {
      await api.post('/api/admin/courses', { code: code.trim(), title: title.trim() })
      toast.success('Course added')
      setCode('')
      setTitle('')
      reload()
    } catch (failure) {
      setError(failure.message)
    } finally {
      setSaving(false)
    }
  }
  return (
    <>
      <PageHeader eyebrow="Course catalogue" title="Courses" subtitle="The course catalogue of the department" />
      <Card hover={false} className="mb-5">
        <form onSubmit={save} className="flex flex-col gap-4">
          {error && <Alert tone="error">{error}</Alert>}
          <div className="grid gap-4 sm:grid-cols-[1fr_2fr_auto] sm:items-end">
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
            <Button type="submit" loading={saving}>
              <Plus size={15} />
              Add
            </Button>
          </div>
        </form>
      </Card>
      {listError && <Alert tone="error" className="mb-5">{listError}</Alert>}
      {loading ? (
        <SkeletonTable columns={2} />
      ) : items.length === 0 ? (
        <EmptyState icon={BookOpen} title="No courses" text="Add the first course above." />
      ) : (
        <DataTable className="animate-fade-in">
          <thead>
            <tr className="border-b border-line bg-wash">
              <th className="px-4 py-3 font-semibold text-ink-soft">Code</th>
              <th className="px-4 py-3 font-semibold text-ink-soft">Title</th>
            </tr>
          </thead>
          <tbody>
            {items.map((item) => (
              <tr key={item.id} className="border-b border-line last:border-b-0">
                <td className="px-4 py-3 font-medium text-ink">{item.code}</td>
                <td className="px-4 py-3 text-ink-soft">{item.title}</td>
              </tr>
            ))}
          </tbody>
        </DataTable>
      )}
    </>
  )
}

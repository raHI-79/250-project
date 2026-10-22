import { useState } from 'react'
import { BookOpen } from 'lucide-react'
import { Alert, Card, EmptyState, PageHeader, Select, SkeletonCard } from '../../components/ui'
import RoutineByDay from '../../components/RoutineByDay'
import { useAuth } from '../../context/auth'
import { useResource } from '../../lib/hooks'
import { WEEKDAYS } from '../../lib/time'

export default function FixedRoutinePage() {
  const { user } = useAuth()
  const [batch, setBatch] = useState(user?.batch || '')
  const [section, setSection] = useState(user?.section || '')
  const [day, setDay] = useState('')
  const [course, setCourse] = useState('')
  const batches = useResource('/api/lookups/batches')
  const sections = useResource('/api/lookups/sections', batch ? { batch } : null)
  const courses = useResource('/api/lookups/courses')
  const query = {}
  if (batch) query.batch = batch
  if (section) query.section = section
  if (day) query.day = day
  if (course) query.course = course
  const { data, loading, error } = useResource('/api/students/routine', query)
  const items = data?.items || []
  return (
    <>
      <PageHeader
        eyebrow="Weekly routine" title="Fixed Routine"
        subtitle="The weekly routine of this semester, for any batch and section"
      />
      <Card hover={false} className="mb-5">
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
          <Select label="Batch" value={batch} onChange={(event) => setBatch(event.target.value)}>
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
            {(sections.data?.items || []).map((entry) => (
              <option key={entry} value={entry}>
                Section {entry}
              </option>
            ))}
            <option value="ALL">Both sections only</option>
          </Select>
          <Select label="Day" value={day} onChange={(event) => setDay(event.target.value)}>
            <option value="">Every day</option>
            {WEEKDAYS.map((name, index) => (
              <option key={name} value={String(index)}>
                {name}
              </option>
            ))}
          </Select>
          <Select
            label="Course"
            value={course}
            onChange={(event) => setCourse(event.target.value)}
          >
            <option value="">Every course</option>
            {(courses.data?.items || []).map((entry) => (
              <option key={entry.id} value={entry.code}>
                {entry.code} {entry.title}
              </option>
            ))}
          </Select>
        </div>
      </Card>
      {error && <Alert tone="error" className="mb-5">{error}</Alert>}
      {loading ? (
        <SkeletonCard />
      ) : items.length === 0 ? (
        <EmptyState
          icon={BookOpen}
          title="No routine entries"
          text="Nothing matches these filters yet."
        />
      ) : (
        <RoutineByDay items={items} />
      )}
    </>
  )
}

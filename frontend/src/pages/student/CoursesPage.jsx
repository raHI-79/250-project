import { useState } from 'react'
import { Link } from 'react-router-dom'
import { BookOpen } from 'lucide-react'
import {
  Alert,
  Card,
  EmptyState,
  PageHeader,
  Select,
  SkeletonCard,
  StatusBadge,
} from '../../components/ui'
import { useAuth } from '../../context/auth'
import { useResource } from '../../lib/hooks'
import { paths } from '../../lib/paths'
import { semesterLabel } from '../../lib/options'
import { BADGE_COURSES, useSectionSeen } from '../../lib/badges'

const batchOptions = (batches, own) => {
  const all = batches || []
  if (!own) return all
  return [own, ...all.filter((entry) => entry !== own)]
}

const weeklyLine = (course) => {
  const parts = []
  if (course.classes_per_week > 0) {
    parts.push(`${course.classes_per_week} class${course.classes_per_week === 1 ? '' : 'es'} a week`)
  }
  if (course.labs_per_week > 0) {
    parts.push(`${course.labs_per_week} lab${course.labs_per_week === 1 ? '' : 's'} a week`)
  }
  return parts.join(' · ')
}

export default function CoursesPage() {
  useSectionSeen(BADGE_COURSES)
  const { user } = useAuth()
  const [batch, setBatch] = useState(user?.batch || '')
  const batches = useResource('/api/lookups/batches')
  const { data, loading, error } = useResource('/api/students/courses', batch ? { batch } : {})
  const items = data?.items || []
  const total = data?.total_credits || 0
  return (
    <>
      <PageHeader
        eyebrow="This semester"
        title="Courses"
        subtitle={
          data
            ? `${data.batch} · ${semesterLabel(data.semester)}`
            : 'The courses running right now'
        }
      />
      <Card hover={false} className="mb-5">
        <div className="grid gap-4 sm:grid-cols-2">
          <Select label="Batch" value={batch} onChange={(event) => setBatch(event.target.value)}>
            {batchOptions(batches.data?.items, user?.batch).map((entry) => (
              <option key={entry} value={entry}>
                {entry}
              </option>
            ))}
          </Select>
          <div className="flex items-end">
            <p className="text-sm font-semibold text-ink">
              {items.length} course{items.length === 1 ? '' : 's'} · {total.toFixed(2)} credits
            </p>
          </div>
        </div>
      </Card>
      {error && (
        <Alert tone="error" className="mb-5">
          {error}
        </Alert>
      )}
      {loading ? (
        <SkeletonCard />
      ) : items.length === 0 ? (
        <EmptyState
          icon={BookOpen}
          title="No courses yet"
          text="The office has not entered a routine for this batch's running semester."
        />
      ) : (
        <div className="grid gap-4 sm:grid-cols-2">
          {items.map((course) => (
            <Card key={course.course_id} hover={false} className="animate-fade-in-up">
              <div className="flex flex-wrap items-start justify-between gap-2">
                <div className="min-w-0">
                  <p className="font-display text-lg font-bold text-ink">{course.code}</p>
                  <p className="text-sm text-ink-soft">{course.title}</p>
                </div>
                <StatusBadge status="approved" label={`${course.credit.toFixed(2)} credits`} />
              </div>
              <p className="mt-3 text-sm text-muted">
                {course.teachers.map((teacher, index) => (
                  <span key={teacher.user_id || index}>
                    {index > 0 ? ', ' : ''}
                    {teacher.user_id ? (
                      <Link
                        to={paths.person(teacher.user_id)}
                        className="font-medium text-brand-600 hover:underline"
                      >
                        {teacher.name}
                      </Link>
                    ) : (
                      <span className="font-medium text-ink-soft">{teacher.name}</span>
                    )}
                    {teacher.designation ? ` · ${teacher.designation}` : ''}
                  </span>
                ))}
              </p>
              <p className="mt-2 text-xs text-faint">
                {course.sections}
                {weeklyLine(course) ? ` · ${weeklyLine(course)}` : ''}
              </p>
            </Card>
          ))}
        </div>
      )}
    </>
  )
}

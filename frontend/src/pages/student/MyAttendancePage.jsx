import { useMemo, useState } from 'react'
import { Check, PieChart, X } from 'lucide-react'
import {
  Alert,
  Card,
  EmptyState,
  Field,
  PageHeader,
  PullToRefresh,
  Select,
  SkeletonCard,
} from '../../components/ui'
import cx from '../../lib/cx'
import { useResource } from '../../lib/hooks'
import { useSearchParams } from 'react-router-dom'
import { formatClock, formatDate } from '../../lib/time'
import { belowPass } from '../../lib/attendance'

const monthKey = (date) => String(date || '').slice(0, 7)

const monthLabel = (key) => {
  const [year, month] = String(key || '').split('-')
  if (!year || !month) return ''
  const date = new Date(Number(year), Number(month) - 1, 1)
  return date.toLocaleDateString('en-GB', { month: 'long', year: 'numeric' })
}

const methodLabel = (method) => {
  if (method === 'app') return 'Bluetooth'
  if (method === 'code') return 'Code'
  if (method === 'manual') return 'Set by the teacher'
  return 'Not marked'
}

function CourseCard({ course, active, onPick }) {
  return (
    <button
      type="button"
      onClick={() => onPick(active ? '' : String(course.course_id))}
      className={cx(
        'press rounded-card border px-4 py-3 text-left transition-colors',
        active
          ? 'border-brand-600 bg-brand-50'
          : 'border-line-strong bg-surface hover:bg-wash',
      )}
    >
      <p className="truncate text-sm font-semibold text-ink">{course.course_code}</p>
      <p className="mt-0.5 truncate text-xs text-muted">{course.course_title}</p>
      <p
        className={cx(
          'font-display mt-2 text-title font-bold',
          belowPass(course.percentage) ? 'text-danger-600' : 'text-ink',
        )}
      >
        {course.percentage} %
      </p>
      <p className="text-xs text-muted">
        {course.present} of {course.held} class{course.held === 1 ? '' : 'es'}
      </p>
    </button>
  )
}

export default function MyAttendancePage() {
  const [params, setParams] = useSearchParams()
  const { data, loading, error, reload } = useResource('/api/students/attendance/mine')
  const [month, setMonth] = useState('')
  const course = params.get('course') || ''
  const courses = useMemo(() => data?.courses || [], [data])
  const classes = useMemo(() => data?.classes || [], [data])
  const months = useMemo(() => {
    const keys = new Set()
    classes.forEach((entry) => keys.add(monthKey(entry.date)))
    return Array.from(keys).sort().reverse()
  }, [classes])
  const shown = useMemo(
    () =>
      classes.filter((entry) => {
        if (course && String(entry.course_id) !== course) return false
        if (month && monthKey(entry.date) !== month) return false
        return true
      }),
    [classes, course, month],
  )
  const pickCourse = (value) => {
    setParams(
      (current) => {
        const copy = new URLSearchParams(current)
        if (value) copy.set('course', value)
        else copy.delete('course')
        return copy
      },
      { replace: true },
    )
  }
  const low = courses.filter((entry) => belowPass(entry.percentage) && entry.held >= 3)
  return (
    <PullToRefresh onRefresh={reload}>
      <PageHeader
        eyebrow="Your classes"
        title="My attendance"
        subtitle="Your share of every course, and every class with the time, the teacher and any note."
      />
      {error && (
        <Alert tone="error" className="mb-5">
          {error}
        </Alert>
      )}
      {low.length > 0 && (
        <Alert tone="warning" className="mb-5" title="Below 50 %">
          {low.map((entry) => entry.course_code).join(', ')} need more classes before the exams.
        </Alert>
      )}
      {loading ? (
        <SkeletonCard />
      ) : courses.length === 0 ? (
        <EmptyState
          icon={PieChart}
          title="No attendance yet"
          text="Once your teachers start taking attendance by Bluetooth, every class shows up here."
        />
      ) : (
        <div className="flex flex-col gap-5">
          <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
            {courses.map((entry) => (
              <CourseCard
                key={entry.course_id}
                course={entry}
                active={course === String(entry.course_id)}
                onPick={pickCourse}
              />
            ))}
          </div>
          <Card hover={false}>
            <div className="flex flex-wrap items-end gap-3">
              <Field label="Course" htmlFor="attendance-course" className="min-w-44 flex-1">
                <Select
                  id="attendance-course"
                  value={course}
                  onChange={(event) => pickCourse(event.target.value)}
                >
                  <option value="">Every course</option>
                  {courses.map((entry) => (
                    <option key={entry.course_id} value={entry.course_id}>
                      {entry.course_code}
                    </option>
                  ))}
                </Select>
              </Field>
              <Field label="Month" htmlFor="attendance-month" className="min-w-44 flex-1">
                <Select
                  id="attendance-month"
                  value={month}
                  onChange={(event) => setMonth(event.target.value)}
                >
                  <option value="">Every month</option>
                  {months.map((key) => (
                    <option key={key} value={key}>
                      {monthLabel(key)}
                    </option>
                  ))}
                </Select>
              </Field>
            </div>
            {shown.length === 0 ? (
              <p className="mt-4 text-sm text-muted">No class matches these filters.</p>
            ) : (
              <ul className="mt-4 flex flex-col divide-y divide-line">
                {shown.map((entry) => {
                  const present = entry.status === 'present'
                  return (
                    <li key={entry.session_id} className="flex items-center gap-3 py-3">
                      <span
                        className={cx(
                          'grid size-9 shrink-0 place-items-center rounded-full',
                          present
                            ? 'bg-success-50 text-success-600'
                            : 'bg-danger-50 text-danger-600',
                        )}
                      >
                        {present ? <Check size={17} /> : <X size={17} />}
                      </span>
                      <span className="min-w-0 flex-1">
                        <span className="block truncate text-sm font-semibold text-ink">
                          {entry.course_code} · {formatDate(entry.date)}
                        </span>
                        <span className="mt-0.5 block truncate text-xs text-muted">
                          {entry.teacher_name} · {methodLabel(entry.method)}
                          {entry.marked_at ? ` · ${formatClock(entry.marked_at)}` : ''}
                        </span>
                        {entry.note && (
                          <span className="mt-0.5 block text-xs text-ink-soft">{entry.note}</span>
                        )}
                      </span>
                      <span
                        className={cx(
                          'shrink-0 text-xs font-bold uppercase',
                          present ? 'text-success-700' : 'text-danger-600',
                        )}
                      >
                        {present ? 'Present' : 'Absent'}
                      </span>
                    </li>
                  )
                })}
              </ul>
            )}
          </Card>
        </div>
      )}
    </PullToRefresh>
  )
}

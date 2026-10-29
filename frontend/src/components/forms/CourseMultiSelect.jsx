import { useEffect, useState } from 'react'
import { Loader2, Search, X } from 'lucide-react'
import { Field, Input } from '../ui'
import api from '../../lib/api'
import { useDebounced } from '../../lib/hooks'

export default function CourseMultiSelect({
  value = [],
  onChange,
  onRemove,
  label = 'My drop courses',
  hint = 'Search the running courses of the other batches, then pick the ones you are retaking',
  error = '',
  className,
}) {
  const [search, setSearch] = useState('')
  const [options, setOptions] = useState([])
  const [loading, setLoading] = useState(false)
  const [problem, setProblem] = useState('')
  const settled = useDebounced(search)
  useEffect(() => {
    const term = settled.trim()
    if (!term) {
      setOptions([])
      setLoading(false)
      return undefined
    }
    const controller = new AbortController()
    let active = true
    setLoading(true)
    setProblem('')
    api
      .get('/api/students/drop-course-options', {
        query: { search: term },
        signal: controller.signal,
      })
      .then((data) => {
        if (active) setOptions(data?.items || [])
      })
      .catch((failure) => {
        if (!active || failure?.name === 'AbortError') return
        setProblem(failure.message)
      })
      .finally(() => {
        if (active) setLoading(false)
      })
    return () => {
      active = false
      controller.abort()
    }
  }, [settled])
  const chosen = new Set(value.map((course) => course.id))
  const add = (course) => {
    if (chosen.has(course.id)) return
    onChange([...value, course])
    setSearch('')
    setOptions([])
  }
  const remove = (course) => onChange(value.filter((entry) => entry.id !== course.id))
  const drop = async (course) => {
    if (onRemove && !(await onRemove(course))) return
    remove(course)
  }
  const available = options.filter((course) => !chosen.has(course.id))
  return (
    <Field label={label} error={error || problem} hint={hint} className={className}>
      {value.length > 0 && (
        <div className="flex flex-wrap gap-2">
          {value.map((course) => (
            <span
              key={course.id}
              className="inline-flex items-center gap-1.5 rounded-full border border-line-strong bg-wash py-1 pr-1 pl-3 text-xs font-medium text-ink-soft"
            >
              <span>
                {course.title || course.code}
                {course.title ? ` · ${course.code}` : ''}
              </span>
              <button
                type="button"
                onClick={() => drop(course)}
                aria-label={`Remove ${course.code}`}
                className="rounded-full p-0.5 text-muted transition-colors hover:bg-brand-200 hover:text-ink"
              >
                <X size={13} />
              </button>
            </span>
          ))}
        </div>
      )}
      <div className="relative">
        <Input
          value={search}
          onChange={(event) => setSearch(event.target.value)}
          placeholder="CSE 2243 7637D or a title"
          aria-label="Search drop courses"
        />
        <span className="pointer-events-none absolute top-2.5 right-3 text-faint">
          {loading ? <Loader2 size={15} className="animate-spin" /> : <Search size={15} />}
        </span>
        {available.length > 0 && (
          <ul className="animate-fade-in absolute inset-x-0 top-full z-20 mt-1 max-h-56 overflow-y-auto rounded-xl border border-line-strong bg-surface py-1 shadow-lg">
            {available.map((course) => (
              <li key={course.id}>
                <button
                  type="button"
                  onClick={() => add(course)}
                  className="flex w-full flex-col items-start px-3 py-2 text-left transition-colors hover:bg-wash"
                >
                  <span className="text-sm font-medium text-ink">
                    {course.title || course.code}
                  </span>
                  <span className="text-xs text-muted">{course.code}</span>
                </button>
              </li>
            ))}
          </ul>
        )}
        {!loading && settled.trim() && available.length === 0 && options.length === 0 && (
          <p className="mt-1 text-xs text-faint">
            No course in a fixed routine matches this search.
          </p>
        )}
      </div>
    </Field>
  )
}

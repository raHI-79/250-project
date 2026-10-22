import { CalendarDays } from 'lucide-react'
import { Alert, Card, EmptyState, PageHeader, SkeletonCard } from '../../components/ui'
import { useResource } from '../../lib/hooks'
import { WEEKDAYS, formatTimeRange } from '../../lib/time'
import { kindLabel, sectionShort } from '../../lib/schedule'

export default function TeacherFixedRoutinePage() {
  const { data, loading, error } = useResource('/api/teacher/fixed-routine')
  const items = data?.items || []
  const days = WEEKDAYS.map((name, index) => ({ name, index })).filter((day) =>
    items.some((item) => item.day_of_week === day.index),
  )
  return (
    <>
      <PageHeader
        title="Fixed weekly routine"
        subtitle="The slots the department gave you, week after week"
      />
      {error && (
        <Alert tone="error" className="mb-5">
          {error}
        </Alert>
      )}
      {loading ? (
        <SkeletonCard />
      ) : items.length === 0 ? (
        <EmptyState
          icon={CalendarDays}
          title="No fixed slots"
          text="The administration has not put you in the weekly routine yet."
        />
      ) : (
        <div className="grid gap-4 lg:grid-cols-2">
          {days.map((day) => (
            <Card key={day.name} hover={false} className="animate-fade-in-up">
              <p className="font-display text-base font-semibold text-brand-800">{day.name}</p>
              <ul className="mt-2 flex flex-col">
                {items
                  .filter((item) => item.day_of_week === day.index)
                  .map((item) => (
                    <li
                      key={item.id}
                      className="flex flex-wrap items-center gap-x-3 gap-y-1 border-b border-brand-50 py-2 text-sm last:border-b-0"
                    >
                      <span className="w-32 shrink-0 font-medium tabular-nums text-brand-600">
                        {formatTimeRange(item.start_time, item.end_time)}
                      </span>
                      <span className="font-medium text-brand-900">
                        {item.course_code}
                        {item.course_title ? ` ${item.course_title}` : ''}
                      </span>
                      <span className="text-brand-500">
                        {item.batch} {sectionShort(item.section)}
                        {item.room ? ` · ${item.room}` : ''}
                      </span>
                      {item.type !== 'class' && (
                        <span className="text-xs text-brand-400">{kindLabel(item.type)}</span>
                      )}
                    </li>
                  ))}
              </ul>
            </Card>
          ))}
        </div>
      )}
    </>
  )
}

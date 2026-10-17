import { CalendarDays } from 'lucide-react'
import {
  Alert,
  Card,
  EmptyState,
  PageHeader,
  SkeletonCard,
  StatusBadge,
} from '../../components/ui'
import { useResource } from '../../lib/hooks'
import { formatDate, formatTimeRange } from '../../lib/time'

const KIND_LABELS = { lab: 'Lab', exam: 'Exam', extra: 'Extra class' }

const sectionLabel = (section) => (section === 'ALL' ? 'Both sections' : `Section ${section}`)

function EntryLine({ item }) {
  const kind = KIND_LABELS[item.kind]
  return (
    <li className="flex flex-wrap items-center gap-x-3 gap-y-1 border-b border-brand-50 py-2 text-sm last:border-b-0">
      <span className="w-36 shrink-0 font-medium text-brand-700">
        {item.source === 'dated' ? formatDate(item.event_date) : item.day}
      </span>
      <span className="font-medium tabular-nums text-brand-600">
        {formatTimeRange(item.start_time, item.end_time)}
      </span>
      <span className="font-medium text-brand-900">
        {item.course_code}
        {item.course_title ? ` ${item.course_title}` : ''}
        {kind ? ` ${kind}` : ''}
        {item.room ? ` @ ${item.room}` : ''}
      </span>
      <StatusBadge status="unavailable" label={sectionLabel(item.section)} />
      {item.teacher_name && <span className="text-brand-500">{item.teacher_name}</span>}
    </li>
  )
}

function BatchCard({ group }) {
  const fixed = group.items.filter((item) => item.source === 'fixed')
  const dated = group.items.filter((item) => item.source !== 'fixed')
  return (
    <Card hover={false} className="animate-fade-in-up">
      <p className="font-display text-base font-semibold text-brand-800">{group.batch}</p>
      {fixed.length > 0 && (
        <div className="mt-3">
          <p className="text-xs font-semibold tracking-wide text-brand-500 uppercase">
            Fixed routine
          </p>
          <ul className="mt-1 flex flex-col">
            {fixed.map((item, index) => (
              <EntryLine key={`fixed-${item.day_of_week}-${item.start_time}-${index}`} item={item} />
            ))}
          </ul>
        </div>
      )}
      {dated.length > 0 && (
        <div className="mt-4">
          <p className="text-xs font-semibold tracking-wide text-brand-500 uppercase">
            Classes and exams ahead
          </p>
          <ul className="mt-1 flex flex-col">
            {dated.map((item, index) => (
              <EntryLine key={`dated-${item.event_date}-${item.start_time}-${index}`} item={item} />
            ))}
          </ul>
        </div>
      )}
    </Card>
  )
}

export default function DropCourseRoutinePage() {
  const { data, loading, error } = useResource('/api/students/drop-course-routine')
  const groups = data?.groups || []
  return (
    <>
      <PageHeader
        title="Drop Course Routine"
        subtitle={
          data?.course_code
            ? `Where ${data.course_code} runs in the other batches`
            : 'Your drop course in the other batches'
        }
      />
      {error && <Alert tone="error" className="mb-5">{error}</Alert>}
      {loading ? (
        <SkeletonCard />
      ) : !data?.has_drop_course ? (
        <EmptyState
          icon={CalendarDays}
          title="No drop course"
          text="You did not register a drop course, so only your own batch routine applies to you."
        />
      ) : groups.length === 0 ? (
        <EmptyState
          icon={CalendarDays}
          title="Nothing scheduled"
          text={`No other batch has ${data.course_code} in the routine or in the approved events yet.`}
        />
      ) : (
        <div className="flex flex-col gap-4">
          {groups.map((group) => (
            <BatchCard key={group.batch} group={group} />
          ))}
        </div>
      )}
    </>
  )
}

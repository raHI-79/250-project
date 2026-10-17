import { Card, StatusBadge } from './ui'
import { WEEKDAYS, formatTimeRange } from '../lib/time'

const TYPE_LABELS = { class: 'Class', lab: 'Lab', theory: 'Theory', exam: 'Exam' }

const sectionLabel = (section) => (section === 'ALL' ? 'Both sections' : `Section ${section}`)

export default function RoutineByDay({ items }) {
  const days = WEEKDAYS.map((name, index) => ({ name, index })).filter((entry) =>
    items.some((item) => item.day_of_week === entry.index),
  )
  return (
    <div className="flex flex-col gap-4">
      {days.map((entry) => (
        <Card key={entry.name} hover={false} className="animate-fade-in-up">
          <p className="font-display text-base font-semibold text-brand-800">{entry.name}</p>
          <ul className="mt-2 flex flex-col">
            {items
              .filter((item) => item.day_of_week === entry.index)
              .map((item) => (
                <li
                  key={item.id}
                  className="flex flex-wrap items-center gap-x-3 gap-y-1 border-b border-brand-50 py-2 text-sm last:border-b-0"
                >
                  <span className="font-medium tabular-nums text-brand-600">
                    {formatTimeRange(item.start_time, item.end_time)}
                  </span>
                  <span className="font-medium text-brand-900">
                    {item.course_code}
                    {item.course_title ? ` ${item.course_title}` : ''}
                    {TYPE_LABELS[item.type] ? ` ${TYPE_LABELS[item.type]}` : ''}
                    {item.room ? ` @ ${item.room}` : ''}
                  </span>
                  <StatusBadge status="unavailable" label={sectionLabel(item.section)} />
                  {item.teacher_name && <span className="text-brand-500">{item.teacher_name}</span>}
                </li>
              ))}
          </ul>
        </Card>
      ))}
    </div>
  )
}

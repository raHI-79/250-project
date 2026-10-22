import { Card, StatusBadge } from './ui'
import TimelineItem from './schedule/TimelineItem'
import { WEEKDAYS } from '../lib/time'

const TYPE_LABELS = { class: 'Class', lab: 'Lab', theory: 'Theory', exam: 'Exam' }

const sectionLabel = (section) => (section === 'ALL' ? 'Both sections' : `Section ${section}`)

export default function RoutineByDay({ items }) {
  const days = WEEKDAYS.map((name, index) => ({ name, index })).filter((entry) =>
    items.some((item) => item.day_of_week === entry.index),
  )
  return (
    <div className="flex flex-col gap-4">
      {days.map((entry) => (
        <Card key={entry.name} hover={false} className="animate-fade-in-up p-0">
          <p className="border-b border-line px-4 py-3 text-xs font-semibold tracking-[0.1em] text-brand-600 uppercase sm:px-5">
            {entry.name}
          </p>
          <ul className="flex flex-col px-4 sm:px-5">
            {items
              .filter((item) => item.day_of_week === entry.index)
              .map((item) => (
                <TimelineItem key={item.id} start={item.start_time} end={item.end_time}>
                  <div className="flex flex-wrap items-center gap-x-2.5 gap-y-1.5">
                    <span className="text-sm font-semibold text-ink">
                      {item.course_code}
                      {item.course_title ? ` ${item.course_title}` : ''}
                    </span>
                    {TYPE_LABELS[item.type] && item.type !== 'class' && (
                      <StatusBadge status="unavailable" label={TYPE_LABELS[item.type]} />
                    )}
                    <StatusBadge status="unavailable" label={sectionLabel(item.section)} />
                  </div>
                  <p className="mt-1 text-xs text-muted">
                    {item.room ? `Room ${item.room}` : ''}
                    {item.teacher_name && item.room ? ' · ' : ''}
                    {item.teacher_name}
                  </p>
                </TimelineItem>
              ))}
          </ul>
        </Card>
      ))}
    </div>
  )
}

import { Card, EmptyState } from './ui'
import { CalendarDays } from 'lucide-react'
import { SECTION_ALL } from '../lib/schedule'
import { WEEKDAYS, formatTimeRange, overlaps } from '../lib/time'
import cx from '../lib/cx'

const inSection = (item, section) => item.section === section || item.section === SECTION_ALL

const entryAt = (items, section, day, period) =>
  items.find(
    (item) =>
      inSection(item, section) &&
      item.day_of_week === day &&
      overlaps(item.start_time, item.end_time, period.start_time, period.end_time),
  ) || null

export default function RoutineWeekGrid({ items, periods, section, className }) {
  const teaching = (periods || []).filter((period) => !period.is_break)
  const mine = (items || []).filter((item) => inSection(item, section))
  const days = WEEKDAYS.map((name, index) => ({ name, index })).filter((day) =>
    mine.some((item) => item.day_of_week === day.index),
  )
  if (teaching.length === 0 || days.length === 0) {
    return (
      <EmptyState
        icon={CalendarDays}
        title={`Nothing for section ${section}`}
        text="No class sits in this semester for this section yet."
        className={className}
      />
    )
  }
  return (
    <Card hover={false} className={cx('animate-fade-in-up overflow-x-auto', className)}>
      <p className="font-display text-lg font-bold text-ink">Section {section}</p>
      <table className="mt-3 w-full min-w-[34rem] border-separate border-spacing-1 text-xs">
        <thead>
          <tr>
            <th className="w-20 text-left font-semibold text-muted">Day</th>
            {teaching.map((period) => (
              <th key={period.id} className="font-medium whitespace-nowrap text-ink-soft">
                <span className="block">{period.label}</span>
                <span className="block tabular-nums text-faint">{period.start_time}</span>
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {days.map((day) => (
            <tr key={day.index}>
              <th className="text-left font-medium whitespace-nowrap text-ink-soft">
                {day.name.slice(0, 3)}
              </th>
              {teaching.map((period) => {
                const entry = entryAt(mine, section, day.index, period)
                return (
                  <td key={period.id} className="p-0 align-top">
                    <div
                      title={
                        entry
                          ? `${entry.course_code} ${entry.course_title} · ${entry.teacher_name} · ${
                              entry.room ? `Room ${entry.room}` : 'No room'
                            } · ${formatTimeRange(entry.start_time, entry.end_time)}`
                          : `Free · ${formatTimeRange(period.start_time, period.end_time)}`
                      }
                      className={cx(
                        'flex h-14 flex-col justify-center gap-0.5 rounded border px-1.5 text-center',
                        entry
                          ? 'border-brand-200 bg-brand-50 text-ink'
                          : 'border-line bg-wash text-faint',
                      )}
                    >
                      {entry ? (
                        <>
                          <span className="truncate font-semibold">{entry.course_code}</span>
                          <span className="truncate text-faint">
                            {entry.room ? entry.room : '—'}
                            {entry.type === 'lab' ? ' · Lab' : ''}
                          </span>
                        </>
                      ) : (
                        <span>·</span>
                      )}
                    </div>
                  </td>
                )
              })}
            </tr>
          ))}
        </tbody>
      </table>
    </Card>
  )
}

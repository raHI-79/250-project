import { Card, EmptyState } from '../ui'
import { DoorClosed } from 'lucide-react'
import { formatTimeRange, overlaps } from '../../lib/time'
import { sectionShort } from '../../lib/schedule'
import cx from '../../lib/cx'

const bookingIn = (slots, period) =>
  (slots || []).find((slot) =>
    slot.start_time && overlaps(slot.start_time, slot.end_time, period.start_time, period.end_time),
  ) || null

const byDate = (rooms) => {
  const dates = new Map()
  rooms.forEach((room) => {
    ;(room.days || []).forEach((day) => {
      if (!dates.has(day.date)) dates.set(day.date, { date: day.date, weekday: day.weekday, rows: [] })
      dates.get(day.date).rows.push({ room: room.room, slots: day.slots })
    })
  })
  return [...dates.values()].sort((left, right) => left.date.localeCompare(right.date))
}

export default function AvailabilityGrid({ rooms, periods, onPick }) {
  const days = byDate(rooms || [])
  if (days.length === 0 || !periods || periods.length === 0) {
    return (
      <EmptyState
        icon={DoorClosed}
        title="No managed rooms"
        text="A room shows up here once a fixed routine uses it and the class periods are set."
      />
    )
  }
  return (
    <div className="flex flex-col gap-4">
      {days.map((day) => (
        <Card key={day.date} hover={false} className="animate-fade-in-up overflow-x-auto">
          <p className="font-display text-lg font-bold text-ink">
            {day.weekday} · {day.date}
          </p>
          <table className="mt-3 w-full min-w-[46rem] border-separate border-spacing-1 text-xs">
            <thead>
              <tr>
                <th className="w-20 text-left font-semibold text-muted">Room</th>
                {periods.map((period) => (
                  <th
                    key={period.id}
                    className={cx(
                      'font-medium whitespace-nowrap',
                      period.is_break ? 'text-faint' : 'text-ink-soft',
                    )}
                  >
                    <span className="block">{period.label}</span>
                    <span className="block tabular-nums text-faint">{period.start_time}</span>
                  </th>
                ))}
              </tr>
            </thead>
            <tbody>
              {day.rows.map((row) => (
                <tr key={row.room}>
                  <th className="text-left font-semibold whitespace-nowrap text-ink">{row.room}</th>
                  {periods.map((period) => {
                    const booking = period.is_break ? null : bookingIn(row.slots, period)
                    const range = formatTimeRange(period.start_time, period.end_time)
                    const title = period.is_break
                      ? `${period.label} · ${range}`
                      : booking
                        ? `${booking.batch} ${sectionShort(booking.section)} · ${booking.course_code} · ${booking.status} · ${formatTimeRange(booking.start_time, booking.end_time)}`
                        : `Free · ${range}`
                    return (
                      <td key={period.id} className="p-0">
                        <button
                          type="button"
                          title={title}
                          aria-label={`Room ${row.room} on ${day.date}, ${period.label}: ${title}`}
                          disabled={period.is_break || Boolean(booking)}
                          onClick={() =>
                            onPick?.({
                              room: row.room,
                              date: day.date,
                              start: period.start_time,
                              end: period.end_time,
                            })
                          }
                          className={cx(
                            'h-8 w-full rounded border transition-colors',
                            period.is_break
                              ? 'cursor-not-allowed border-line bg-wash bg-[repeating-linear-gradient(45deg,transparent,transparent_4px,rgba(0,0,0,0.06)_4px,rgba(0,0,0,0.06)_8px)]'
                              : booking
                                ? 'cursor-not-allowed border-accent-300 bg-accent-100'
                                : 'border-success-100 bg-success-50 hover:border-brand-500 hover:bg-brand-100',
                          )}
                        />
                      </td>
                    )
                  })}
                </tr>
              ))}
            </tbody>
          </table>
          <div className="mt-3 flex flex-wrap items-center gap-4 text-xs text-muted">
            <span className="inline-flex items-center gap-1.5">
              <span className="size-3 rounded border border-success-100 bg-success-50" />
              Free, click to fill the form
            </span>
            <span className="inline-flex items-center gap-1.5">
              <span className="size-3 rounded border border-accent-300 bg-accent-100" />
              Booked, hover for the details
            </span>
            <span className="inline-flex items-center gap-1.5">
              <span className="size-3 rounded border border-line bg-wash" />
              Break
            </span>
          </div>
        </Card>
      ))}
    </div>
  )
}

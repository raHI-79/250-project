import { Card, EmptyState } from '../ui'
import { DoorClosed } from 'lucide-react'
import { formatTimeRange } from '../../lib/time'
import { sectionShort } from '../../lib/schedule'
import cx from '../../lib/cx'

const HOURS = ['08:00', '09:00', '10:00', '11:00', '12:00', '13:00', '14:00', '15:00', '16:00']

const END_OF = (start) => `${String(Number(start.slice(0, 2)) + 1).padStart(2, '0')}:00`

const bookingAt = (slots, start) => {
  const end = END_OF(start)
  return (slots || []).find((slot) => slot.start_time < end && start < slot.end_time) || null
}

export default function AvailabilityGrid({ rooms, onPick }) {
  if (!rooms || rooms.length === 0) {
    return (
      <EmptyState
        icon={DoorClosed}
        title="No managed rooms"
        text="A room shows up here once a fixed routine uses it."
      />
    )
  }
  return (
    <div className="flex flex-col gap-4">
      {rooms.map((room) => (
        <Card key={room.room} hover={false} className="animate-fade-in-up overflow-x-auto">
          <p className="font-display text-base font-semibold text-brand-800">Room {room.room}</p>
          <table className="mt-3 w-full min-w-[42rem] border-separate border-spacing-1 text-xs">
            <thead>
              <tr>
                <th className="w-24 text-left font-semibold text-brand-500">Day</th>
                {HOURS.map((hour) => (
                  <th key={hour} className="font-medium text-brand-400">
                    {hour.slice(0, 2)}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody>
              {room.days.map((day) => (
                <tr key={day.date}>
                  <th className="text-left font-medium whitespace-nowrap text-brand-700">
                    <span className="block">{day.weekday?.slice(0, 3)}</span>
                    <span className="block tabular-nums text-brand-400">
                      {day.date.slice(8, 10)}.{day.date.slice(5, 7)}
                    </span>
                  </th>
                  {HOURS.map((hour) => {
                    const booking = bookingAt(day.slots, hour)
                    const title = booking
                      ? `${booking.batch} ${sectionShort(booking.section)} · ${booking.course_code} · ${booking.status} · ${formatTimeRange(booking.start_time, booking.end_time)}`
                      : `Free · ${formatTimeRange(hour, END_OF(hour))}`
                    return (
                      <td key={hour} className="p-0">
                        <button
                          type="button"
                          title={title}
                          aria-label={`Room ${room.room} on ${day.date} at ${hour}: ${title}`}
                          disabled={Boolean(booking)}
                          onClick={() => onPick?.({ room: room.room, date: day.date, start: hour })}
                          className={cx(
                            'h-7 w-full rounded border transition-colors',
                            booking
                              ? 'cursor-not-allowed border-accent-200 bg-accent-100'
                              : 'border-brand-200 bg-brand-50 hover:border-brand-500 hover:bg-brand-100',
                          )}
                        />
                      </td>
                    )
                  })}
                </tr>
              ))}
            </tbody>
          </table>
          <div className="mt-3 flex flex-wrap items-center gap-4 text-xs text-brand-500">
            <span className="inline-flex items-center gap-1.5">
              <span className="size-3 rounded border border-brand-200 bg-brand-50" />
              Free, click to fill the form
            </span>
            <span className="inline-flex items-center gap-1.5">
              <span className="size-3 rounded border border-accent-200 bg-accent-100" />
              Booked, hover for the details
            </span>
          </div>
        </Card>
      ))}
    </div>
  )
}

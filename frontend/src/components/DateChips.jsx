import { Card, Input } from './ui'
import { WEEKDAYS, shiftDate, today } from '../lib/time'
import cx from '../lib/cx'

const CHIP_DAYS = 7

const chipLabel = (value, index) => {
  if (index === 0) return 'Today'
  if (index === 1) return 'Tomorrow'
  return WEEKDAYS[new Date(`${value}T00:00:00`).getDay()].slice(0, 3)
}

const chipDate = (value) => value.slice(8, 10) + '.' + value.slice(5, 7)

export default function DateChips({ value, onChange, label = 'Pick a day', min, className }) {
  const start = today()
  const days = Array.from({ length: CHIP_DAYS }, (_, index) => shiftDate(start, index))
  const chips = min ? days.filter((day) => day >= min) : days
  return (
    <Card hover={false} className={cx('mb-5 flex flex-col gap-4', className)}>
      <div className="flex flex-wrap gap-2">
        {chips.map((day) => (
          <button
            key={day}
            type="button"
            onClick={() => onChange(day)}
            className={cx(
              'flex min-w-18 flex-col items-center rounded-lg border px-3 py-2 text-sm transition-colors',
              value === day
                ? 'border-brand-600 bg-brand-600 text-white'
                : 'border-brand-200 bg-white text-brand-700 hover:border-brand-300 hover:bg-brand-50',
            )}
          >
            <span className="font-medium">{chipLabel(day, days.indexOf(day))}</span>
            <span
              className={cx('text-xs tabular-nums', value === day ? 'text-white/80' : 'text-brand-400')}
            >
              {chipDate(day)}
            </span>
          </button>
        ))}
      </div>
      <Input
        label={label}
        type="date"
        value={value}
        min={min}
        onChange={(event) => onChange(event.target.value)}
        className="max-w-xs"
      />
    </Card>
  )
}

import { Input } from './ui'
import { WEEKDAYS, shiftDate, today } from '../lib/time'
import cx from '../lib/cx'

const CHIP_DAYS = 7

const chipLabel = (value, index) => {
  if (index === 0) return 'Today'
  if (index === 1) return 'Tomorrow'
  return WEEKDAYS[new Date(`${value}T00:00:00`).getDay()].slice(0, 3)
}

const chipDate = (value) => `${value.slice(8, 10)}.${value.slice(5, 7)}`

export default function DateChips({ value, onChange, label = 'Pick a day', min, className }) {
  const start = today()
  const days = Array.from({ length: CHIP_DAYS }, (_, index) => shiftDate(start, index))
  const chips = min ? days.filter((day) => day >= min) : days
  return (
    <div className={cx('glass-card mb-5 rounded-card p-4 sm:p-5', className)}>
      <div className="flex flex-wrap gap-2">
        {chips.map((day) => (
          <button
            key={day}
            type="button"
            onClick={() => onChange(day)}
            className={cx(
              'flex min-w-18 flex-col items-center border px-3 py-2 text-xs transition-colors',
              value === day
                ? 'border-brand-600 bg-brand-600 text-page'
                : 'border-line-strong bg-surface text-ink-soft hover:bg-wash',
            )}
          >
            <span className="font-semibold">{chipLabel(day, days.indexOf(day))}</span>
            <span
              className={cx(
                'font-numeral text-xs',
                value === day ? 'text-white/80' : 'text-faint',
              )}
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
        className="mt-4 max-w-xs"
      />
    </div>
  )
}

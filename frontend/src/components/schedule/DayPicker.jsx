import { DateField } from '../ui'
import Segmented from '../ui/Segmented'
import { today as todayDate, tomorrow as tomorrowDate } from '../../lib/time'
import cx from '../../lib/cx'

export const DAY_TOMORROW = 'tomorrow'

export const DAY_TODAY = 'today'

const DAY_TABS = [
  { value: DAY_TOMORROW, label: 'Tomorrow' },
  { value: DAY_TODAY, label: 'Today' },
]

export default function DayPicker({
  value,
  onChange,
  label = 'Pick a day',
  toggle = false,
  onClear,
  clearLabel = 'Show everything ahead',
  className,
}) {
  const today = todayDate()
  const tomorrow = tomorrowDate()
  const mode = value === today ? DAY_TODAY : value === tomorrow ? DAY_TOMORROW : ''
  return (
    <div className={cx('glass-card mb-5 rounded-card p-4 sm:p-5', className)}>
      <div className="flex flex-wrap items-end gap-4">
        {toggle && (
          <Segmented
            options={DAY_TABS}
            value={mode}
            onChange={(next) => onChange(next === DAY_TODAY ? today : tomorrow)}
          />
        )}
        <DateField
          label={label}
          value={value || ''}
          onChange={(next) => {
            if (next) onChange(next)
          }}
          className="w-full max-w-xs"
        />
      </div>
      <div className="mt-3 flex flex-wrap gap-4">
        {value && value !== today && (
          <button
            type="button"
            onClick={() => onChange(today)}
            className="text-sm font-semibold text-brand-600 hover:underline"
          >
            Back to today
          </button>
        )}
        {onClear && value && (
          <button
            type="button"
            onClick={onClear}
            className="text-sm font-semibold text-brand-600 hover:underline"
          >
            {clearLabel}
          </button>
        )}
      </div>
    </div>
  )
}

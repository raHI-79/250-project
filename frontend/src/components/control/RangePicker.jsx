import { DateField } from '../ui'
import cx from '../../lib/cx'

const RANGES = [
  { value: 'today', label: 'Today' },
  { value: '7d', label: '7 days' },
  { value: '30d', label: '30 days' },
  { value: 'semester', label: 'Semester' },
  { value: 'custom', label: 'Custom' },
]

export default function RangePicker({ value, onChange, from, to, onFrom, onTo, className }) {
  return (
    <div className={cx('flex flex-wrap items-end gap-2', className)}>
      <div role="tablist" aria-label="Date range" className="flex flex-wrap gap-1 rounded-input bg-white/15 p-1">
        {RANGES.map((range) => {
          const active = range.value === value
          return (
            <button
              key={range.value}
              type="button"
              role="tab"
              aria-selected={active}
              onClick={() => onChange(range.value)}
              className={cx(
                'min-h-10 rounded-[0.625rem] px-3 text-sm whitespace-nowrap transition-colors',
                active ? 'bg-white font-bold text-brand-700' : 'font-semibold text-white/85 hover:bg-white/10',
              )}
            >
              {range.label}
            </button>
          )
        })}
      </div>
      {value === 'custom' && (
        <div className="flex flex-wrap items-end gap-3">
          <DateField label="From" tone="dark" value={from} onChange={onFrom} className="w-56" />
          <DateField label="To" tone="dark" value={to} onChange={onTo} className="w-56" />
        </div>
      )}
    </div>
  )
}

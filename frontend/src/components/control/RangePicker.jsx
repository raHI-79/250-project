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
        <div className="flex flex-wrap items-end gap-2">
          <label className="flex flex-col gap-1 text-2xs font-semibold text-white/80">
            From
            <input
              type="date"
              value={from}
              onChange={(event) => onFrom(event.target.value)}
              className="h-10 rounded-input border border-white/30 bg-white/15 px-2 text-sm text-white"
            />
          </label>
          <label className="flex flex-col gap-1 text-2xs font-semibold text-white/80">
            To
            <input
              type="date"
              value={to}
              onChange={(event) => onTo(event.target.value)}
              className="h-10 rounded-input border border-white/30 bg-white/15 px-2 text-sm text-white"
            />
          </label>
        </div>
      )}
    </div>
  )
}

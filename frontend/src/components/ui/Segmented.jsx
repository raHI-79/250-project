import cx from '../../lib/cx'

export default function Segmented({ options, value, onChange, label = 'Options', className }) {
  const index = Math.max(
    0,
    options.findIndex((option) => option.value === value),
  )
  return (
    <div
      role="tablist"
      aria-label={label}
      className={cx('glass-chip relative flex gap-1 rounded-full p-1', className)}
    >
      <span
        aria-hidden="true"
        style={{
          width: `calc((100% - 0.5rem) / ${options.length})`,
          transform: `translateX(calc(${index} * 100%))`,
        }}
        className="absolute inset-y-1 left-1 rounded-full bg-surface shadow-[var(--shadow-glass)] transition-transform duration-300 ease-out motion-reduce:transition-none"
      />
      {options.map((option) => {
        const active = option.value === value
        return (
          <button
            key={String(option.value)}
            type="button"
            role="tab"
            aria-selected={active}
            onClick={() => onChange(option.value)}
            className={cx(
              'relative z-10 min-h-10 flex-1 rounded-full px-3 text-sm whitespace-nowrap transition-colors',
              active ? 'font-semibold text-brand-700' : 'font-medium text-muted hover:text-ink',
            )}
          >
            {option.label}
          </button>
        )
      })}
    </div>
  )
}

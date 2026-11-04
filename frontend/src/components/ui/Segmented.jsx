import cx from '../../lib/cx'

export default function Segmented({
  options,
  value,
  onChange,
  label = 'Options',
  tone,
  className,
}) {
  const onHeader = tone === 'header'
  const index = Math.max(
    0,
    options.findIndex((option) => option.value === value),
  )
  return (
    <div
      role="tablist"
      aria-label={label}
      className={cx(
        'relative flex gap-1 rounded-full p-1',
        onHeader ? 'bg-white/20' : 'border border-line bg-wash',
        className,
      )}
    >
      <span
        aria-hidden="true"
        style={{
          width: `calc((100% - 0.5rem) / ${options.length})`,
          transform: `translateX(calc(${index} * 100%))`,
        }}
        className={cx(
          'absolute inset-y-1 left-1 rounded-full transition-transform duration-300 ease-out motion-reduce:transition-none',
          onHeader ? 'bg-white' : 'bg-surface shadow-[var(--shadow-card)]',
        )}
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
              'relative z-10 min-h-11 flex-1 rounded-full px-3 text-sm whitespace-nowrap transition-colors',
              onHeader
                ? active
                  ? 'font-bold text-brand-700'
                  : 'font-semibold text-white'
                : active
                  ? 'font-bold text-brand-700'
                  : 'font-semibold text-muted hover:text-ink',
            )}
          >
            {option.label}
          </button>
        )
      })}
    </div>
  )
}

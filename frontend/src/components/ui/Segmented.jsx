import cx from '../../lib/cx'

export default function Segmented({ options, value, onChange, label = 'Options', className }) {
  return (
    <div
      role="tablist"
      aria-label={label}
      className={cx(
        'flex gap-1 rounded-full border border-line bg-wash p-1',
        className,
      )}
    >
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
              'min-h-10 flex-1 rounded-full px-3 text-sm whitespace-nowrap transition-colors',
              active
                ? 'bg-surface font-semibold text-brand-600 shadow-sm'
                : 'font-medium text-muted hover:text-ink',
            )}
          >
            {option.label}
          </button>
        )
      })}
    </div>
  )
}

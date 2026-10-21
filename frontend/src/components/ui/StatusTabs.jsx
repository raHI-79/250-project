import cx from '../../lib/cx'

export default function StatusTabs({ tabs, value, onChange, className }) {
  return (
    <div
      role="tablist"
      aria-label="Status filter"
      className={cx('flex flex-wrap gap-1 rounded-lg border border-brand-100 bg-brand-50/60 p-1', className)}
    >
      {tabs.map((tab) => {
        const active = tab.value === value
        return (
          <button
            key={tab.value || 'all'}
            type="button"
            role="tab"
            aria-selected={active}
            onClick={() => onChange(tab.value)}
            className={cx(
              'rounded-md px-3 py-1.5 text-sm font-medium transition-colors',
              active
                ? 'bg-white text-brand-800 shadow-sm'
                : 'text-brand-500 hover:bg-white/70 hover:text-brand-700',
            )}
          >
            {tab.label}
          </button>
        )
      })}
    </div>
  )
}

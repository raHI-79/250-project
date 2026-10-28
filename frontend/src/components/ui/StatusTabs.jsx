import cx from '../../lib/cx'

export default function StatusTabs({ tabs, value, onChange, className }) {
  return (
    <div
      role="tablist"
      aria-label="Tabs"
      className={cx(
        'no-scrollbar -mx-4 flex gap-2 overflow-x-auto px-4 sm:mx-0 sm:px-0',
        className,
      )}
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
              'press min-h-10 shrink-0 rounded-full border px-4 text-sm whitespace-nowrap',
              active
                ? 'border-brand-600 bg-brand-600 font-semibold text-page'
                : 'glass-chip font-medium text-muted hover:border-brand-300 hover:text-ink',
            )}
          >
            {tab.label}
          </button>
        )
      })}
    </div>
  )
}

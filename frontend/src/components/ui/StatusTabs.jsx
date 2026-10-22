import cx from '../../lib/cx'

export default function StatusTabs({ tabs, value, onChange, className }) {
  return (
    <div
      role="tablist"
      aria-label="Tabs"
      className={cx('flex flex-wrap border-b border-line bg-surface', className)}
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
              'tap-target border-b-2 px-4 py-3 text-sm whitespace-nowrap transition-colors',
              active
                ? 'border-brand-600 bg-brand-50 font-semibold text-brand-600'
                : 'border-transparent font-medium text-muted hover:bg-wash hover:text-ink-soft',
            )}
          >
            {tab.label}
          </button>
        )
      })}
    </div>
  )
}

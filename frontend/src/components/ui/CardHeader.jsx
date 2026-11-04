import cx from '../../lib/cx'

export default function CardHeader({ title, subtitle, meta, className, children }) {
  return (
    <div
      className={cx(
        'flex flex-wrap items-center justify-between gap-x-3 gap-y-2 px-4 pt-4 pb-3 sm:px-5',
        className,
      )}
    >
      <div className="min-w-0">
        <h2 className="font-display wrap-long text-section font-bold text-ink">{title}</h2>
        {subtitle && <p className="mt-0.5 wrap-long text-xs text-muted">{subtitle}</p>}
      </div>
      {meta}
      {children}
    </div>
  )
}

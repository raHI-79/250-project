import cx from '../../lib/cx'

export default function CardHeader({ title, meta, className, children }) {
  return (
    <div
      className={cx(
        'flex flex-wrap items-center justify-between gap-x-3 gap-y-2 px-4 pt-4 pb-3 sm:px-5',
        className,
      )}
    >
      <h2 className="font-display text-section font-bold text-ink">{title}</h2>
      {meta}
      {children}
    </div>
  )
}

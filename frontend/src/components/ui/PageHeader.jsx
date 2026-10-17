import cx from '../../lib/cx'

export default function PageHeader({ title, subtitle, actions, className }) {
  return (
    <div
      className={cx(
        'flex flex-col gap-3 pb-5 sm:flex-row sm:items-end sm:justify-between',
        className,
      )}
    >
      <div>
        <h1 className="font-display text-2xl font-semibold text-brand-900 sm:text-3xl">{title}</h1>
        {subtitle && <p className="mt-1 text-sm text-brand-500">{subtitle}</p>}
      </div>
      {actions && <div className="flex flex-wrap gap-2">{actions}</div>}
    </div>
  )
}

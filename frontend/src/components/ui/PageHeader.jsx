import cx from '../../lib/cx'

export default function PageHeader({ eyebrow, title, subtitle, actions, className }) {
  return (
    <div
      className={cx(
        'mb-5 flex flex-col gap-3 border-b border-ink pb-3 sm:flex-row sm:items-end sm:justify-between',
        className,
      )}
    >
      <div className="min-w-0">
        {eyebrow && <p className="label-caps">{eyebrow}</p>}
        <h1 className="font-display mt-1.5 text-[30px] leading-tight font-bold text-ink sm:text-[32px]">
          {title}
        </h1>
        {subtitle && <p className="mt-2 max-w-3xl text-sm text-muted">{subtitle}</p>}
      </div>
      {actions && <div className="flex flex-wrap gap-2">{actions}</div>}
    </div>
  )
}

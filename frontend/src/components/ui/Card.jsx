import cx from '../../lib/cx'

export default function Card({
  className,
  hover = true,
  rule = false,
  raised = false,
  flush = false,
  children,
  ...rest
}) {
  return (
    <div
      className={cx(
        'rounded-card border border-line bg-surface',
        !flush && 'p-4 sm:p-5',
        rule && 'border-t-[3px] border-t-brand-600',
        raised && 'bg-raised shadow-sm',
        hover && 'card-hover',
        className,
      )}
      {...rest}
    >
      {children}
    </div>
  )
}

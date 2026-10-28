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
        'glass-card rounded-card',
        !flush && 'p-4 sm:p-5',
        rule && 'border-t-[3px] border-t-brand-600',
        raised && 'shadow-[var(--shadow-glass-lift)]',
        hover && 'card-hover',
        className,
      )}
      {...rest}
    >
      {children}
    </div>
  )
}

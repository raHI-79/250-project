import cx from '../../lib/cx'

export default function Card({ className, hover = true, rule = false, children, ...rest }) {
  return (
    <div
      className={cx(
        'border border-line bg-surface p-4 sm:p-5',
        rule && 'border-t-[3px] border-t-brand-600',
        hover && 'card-hover',
        className,
      )}
      {...rest}
    >
      {children}
    </div>
  )
}

import cx from '../../lib/cx'

export default function Card({ className, hover = true, children, ...rest }) {
  return (
    <div
      className={cx(
        'rounded-xl border border-brand-100 bg-surface p-5 shadow-sm',
        hover && 'card-hover',
        className,
      )}
      {...rest}
    >
      {children}
    </div>
  )
}

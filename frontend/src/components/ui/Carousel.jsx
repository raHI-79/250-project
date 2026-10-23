import cx from '../../lib/cx'

export default function Carousel({ label, children, className }) {
  return (
    <div
      role="group"
      aria-label={label}
      className={cx(
        'no-scrollbar -mx-4 flex snap-x snap-mandatory gap-3 overflow-x-auto px-4 pb-1 sm:mx-0 sm:px-0',
        className,
      )}
    >
      {children}
    </div>
  )
}

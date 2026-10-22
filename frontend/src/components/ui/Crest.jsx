import cx from '../../lib/cx'

const SIZES = {
  sm: 'size-8 text-xs',
  md: 'size-10 text-base',
  lg: 'size-12 text-lg',
}

export default function Crest({ size = 'md', tone = 'brand', className }) {
  return (
    <span
      aria-hidden="true"
      className={cx(
        'font-display grid shrink-0 place-items-center rounded-sm border-2 font-bold',
        tone === 'light'
          ? 'border-white/80 bg-transparent text-white'
          : 'border-brand-600 bg-surface text-brand-600',
        SIZES[size] || SIZES.md,
        className,
      )}
    >
      CS
    </span>
  )
}

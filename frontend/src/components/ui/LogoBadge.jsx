import BrandMark from './BrandMark'
import cx from '../../lib/cx'
import { SUST_LOGO } from '../../lib/brand'
import { UNIVERSITY } from './IdentityBar'

const SIZES = {
  sm: 'size-9 rounded-xl p-1',
  md: 'size-11 rounded-2xl p-1.5',
  lg: 'size-16 rounded-3xl p-2.5',
  xl: 'size-24 rounded-[1.75rem] p-4',
}

const TONES = {
  glass: 'border border-white/60 bg-white/85 shadow-[var(--shadow-glass)] backdrop-blur-sm',
  light: 'border border-white/45 bg-white/90 backdrop-blur-sm',
  plain: 'bg-white',
}

export default function LogoBadge({ size = 'md', tone = 'glass', className }) {
  return (
    <span
      aria-hidden="true"
      className={cx(
        'grid shrink-0 place-items-center overflow-hidden',
        TONES[tone] || TONES.glass,
        SIZES[size] || SIZES.md,
        className,
      )}
    >
      <BrandMark src={SUST_LOGO} alt={UNIVERSITY} className="size-full" />
    </span>
  )
}

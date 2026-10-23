import { useState } from 'react'
import { SUST_LOGO } from '../../lib/brand'
import cx from '../../lib/cx'

const SIZES = {
  sm: 'size-9 text-2xs',
  md: 'size-11 text-xs',
  lg: 'size-14 text-sm',
}

export default function Crest({ size = 'md', tone = 'brand', className }) {
  const [broken, setBroken] = useState(false)
  if (!broken) {
    return (
      <img
        src={SUST_LOGO}
        alt=""
        aria-hidden="true"
        onError={() => setBroken(true)}
        className={cx('shrink-0 object-contain', SIZES[size] || SIZES.md, className)}
      />
    )
  }
  return (
    <span
      aria-hidden="true"
      className={cx(
        'font-display grid shrink-0 place-items-center rounded-xl border-2 font-bold tracking-tight',
        tone === 'light'
          ? 'border-white/80 bg-transparent text-white'
          : 'border-brand-600 bg-brand-600 text-page',
        SIZES[size] || SIZES.md,
        className,
      )}
    >
      CSE
    </span>
  )
}

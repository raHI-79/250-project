import { useState } from 'react'
import { CSE_LOGO } from '../../lib/brand'
import cx from '../../lib/cx'

const SIZES = {
  sm: 'size-8 text-xs',
  md: 'size-10 text-base',
  lg: 'size-12 text-lg',
}

export default function Crest({ size = 'md', tone = 'brand', className }) {
  const [broken, setBroken] = useState(false)
  if (!broken) {
    return (
      <img
        src={CSE_LOGO}
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

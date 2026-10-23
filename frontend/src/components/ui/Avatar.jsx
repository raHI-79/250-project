import { useEffect, useState } from 'react'
import cx from '../../lib/cx'
import { initialsOf } from '../../lib/account'

const sizes = {
  xs: 'size-7 text-xs',
  sm: 'size-8 text-xs',
  md: 'size-10 text-xs',
  lg: 'size-16 text-base',
  xl: 'size-24 text-2xl',
}

export default function Avatar({ src, name, size = 'md', className }) {
  const [broken, setBroken] = useState(false)
  useEffect(() => setBroken(false), [src])
  const showImage = Boolean(src) && !broken
  return (
    <span
      className={cx(
        'font-display inline-flex shrink-0 items-center justify-center overflow-hidden rounded-full bg-linear-to-br from-maroon-600 to-maroon-900 font-semibold text-white uppercase',
        sizes[size] || sizes.md,
        className,
      )}
    >
      {showImage ? (
        <img src={src} alt="" onError={() => setBroken(true)} className="size-full object-cover" />
      ) : (
        initialsOf(name)
      )}
    </span>
  )
}

import { useState } from 'react'
import cx from '../../lib/cx'

export default function BrandMark({ src, alt, className }) {
  const [broken, setBroken] = useState(false)
  if (broken) return null
  return (
    <img
      src={src}
      alt={alt}
      onError={() => setBroken(true)}
      className={cx('shrink-0 object-contain', className)}
    />
  )
}

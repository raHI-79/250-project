import { useState } from 'react'
import { HERO_IMAGES } from '../../lib/campus'
import cx from '../../lib/cx'

export default function CampusBackdrop({
  images = HERO_IMAGES,
  index = 0,
  still = false,
  overlay,
  className,
}) {
  const [broken, setBroken] = useState({})
  return (
    <div
      aria-hidden="true"
      className={cx('absolute inset-0 overflow-hidden bg-deep-950', className)}
    >
      {images.map((source, position) =>
        broken[source] ? null : (
          <img
            key={source}
            src={source}
            alt=""
            sizes="100vw"
            decoding="async"
            fetchPriority={position === 0 ? 'high' : 'low'}
            onError={() => setBroken((current) => ({ ...current, [source]: true }))}
            className={cx(
              'absolute inset-0 size-full object-cover transition-opacity duration-[1600ms] ease-in-out will-change-[opacity,transform] motion-reduce:transition-none',
              position === index ? 'opacity-100' : 'opacity-0',
              position === index && !still ? 'animate-ken-burns' : '',
            )}
          />
        ),
      )}
      {overlay && <div className={cx('absolute inset-0', overlay)} />}
    </div>
  )
}

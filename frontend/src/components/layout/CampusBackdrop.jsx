import { useEffect, useState } from 'react'
import { HERO_IMAGES } from '../../lib/campus'
import cx from '../../lib/cx'

export default function CampusBackdrop({
  images = HERO_IMAGES,
  rotateAfter = 0,
  overlay,
  className,
}) {
  const [index, setIndex] = useState(0)
  const [broken, setBroken] = useState({})
  useEffect(() => {
    if (!rotateAfter || images.length < 2) return undefined
    const timer = window.setInterval(() => {
      setIndex((current) => (current + 1) % images.length)
    }, rotateAfter)
    return () => window.clearInterval(timer)
  }, [images, rotateAfter])
  return (
    <div
      aria-hidden="true"
      className={cx(
        'absolute inset-0 overflow-hidden bg-linear-to-br from-maroon-900 via-maroon-700 to-maroon-600',
        className,
      )}
    >
      {images.map((source, position) =>
        broken[source] ? null : (
          <img
            key={source}
            src={source}
            alt=""
            onError={() => setBroken((current) => ({ ...current, [source]: true }))}
            className={cx(
              'absolute inset-0 size-full object-cover transition-opacity duration-1000',
              position === index ? 'opacity-100' : 'opacity-0',
            )}
          />
        ),
      )}
      {overlay && <div className={cx('absolute inset-0', overlay)} />}
    </div>
  )
}

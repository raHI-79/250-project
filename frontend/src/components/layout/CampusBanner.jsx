import { useEffect, useState } from 'react'
import cx from '../../lib/cx'

const IMAGES = [
  '/campus/banner-monument.jpg',
  '/campus/banner-building.jpg',
  '/campus/banner-entrance.jpg',
]

const ROTATE_AFTER = 6000

export default function CampusBanner({ className }) {
  const [index, setIndex] = useState(0)
  const [broken, setBroken] = useState({})
  useEffect(() => {
    const timer = window.setInterval(() => {
      setIndex((current) => (current + 1) % IMAGES.length)
    }, ROTATE_AFTER)
    return () => window.clearInterval(timer)
  }, [])
  return (
    <div
      className={cx(
        'relative mb-6 h-24 overflow-hidden rounded-xl bg-brand-800 sm:h-28',
        className,
      )}
    >
      {IMAGES.map((source, position) =>
        broken[source] ? null : (
          <img
            key={source}
            src={source}
            alt=""
            aria-hidden="true"
            onError={() => setBroken((current) => ({ ...current, [source]: true }))}
            className={cx(
              'absolute inset-0 size-full object-cover transition-opacity duration-1000',
              position === index ? 'opacity-100' : 'opacity-0',
            )}
          />
        ),
      )}
      <div className="absolute inset-0 bg-linear-to-r from-brand-950/90 via-brand-800/70 to-brand-600/40" />
      <div className="relative flex h-full items-center px-5 sm:px-7">
        <p className="font-display text-sm font-semibold text-white sm:text-base">
          Shahjalal University of Science and Technology
          <span className="px-2 text-brand-200">·</span>
          Department of CSE
        </p>
      </div>
    </div>
  )
}

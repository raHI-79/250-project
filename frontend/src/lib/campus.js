import { useCallback, useEffect, useState } from 'react'

export const CAMPUS_PHOTOS = [
  { src: '/campus/hero-oldGate.jpg', caption: 'SUST Iconic Old Gate, Kumargaon' },
  { src: '/campus/hero-building.jpg', caption: 'IICT Building, SUST campus' },
  { src: '/campus/hero-monument.jpg', caption: 'Shaheed Minar, SUST campus' },
  { src: '/campus/hero-entrance.jpg', caption: 'University main entrance, Kumargaon' },
]

export const HERO_IMAGES = CAMPUS_PHOTOS.map((photo) => photo.src)

export const HERO_BUILDING = [CAMPUS_PHOTOS[1].src]

export const HOME_PHOTO = CAMPUS_PHOTOS[1]

export const captionFor = (source) =>
  CAMPUS_PHOTOS.find((photo) => photo.src === source)?.caption || ''

const REDUCED_QUERY = '(prefers-reduced-motion: reduce)'

const prefersReduced = () => {
  try {
    return window.matchMedia(REDUCED_QUERY).matches
  } catch {
    return false
  }
}

export const useCampusSlideshow = (images = HERO_IMAGES, rotateAfter = 0) => {
  const [index, setIndex] = useState(0)
  const [still, setStill] = useState(prefersReduced)
  const count = images.length
  useEffect(() => {
    let media = null
    try {
      media = window.matchMedia(REDUCED_QUERY)
    } catch {
      return undefined
    }
    const onChange = () => setStill(media.matches)
    media.addEventListener('change', onChange)
    return () => media.removeEventListener('change', onChange)
  }, [])
  useEffect(() => {
    images.forEach((source) => {
      const probe = new Image()
      probe.decoding = 'async'
      probe.src = source
    })
  }, [images])
  useEffect(() => {
    if (still || !rotateAfter || count < 2) return undefined
    let timer = 0
    const start = () => {
      window.clearInterval(timer)
      timer = window.setInterval(() => setIndex((current) => (current + 1) % count), rotateAfter)
    }
    const onVisibility = () => {
      window.clearInterval(timer)
      if (document.visibilityState === 'visible') start()
    }
    start()
    document.addEventListener('visibilitychange', onVisibility)
    return () => {
      window.clearInterval(timer)
      document.removeEventListener('visibilitychange', onVisibility)
    }
  }, [count, rotateAfter, still])
  const show = useCallback((next) => setIndex(next), [])
  const source = images[Math.min(index, Math.max(count - 1, 0))] || ''
  return { index, show, still, caption: captionFor(source) }
}

import { Link } from 'react-router-dom'
import cx from '../../lib/cx'
import paths from '../../lib/paths'
import LogoBadge from '../ui/LogoBadge'
import { DEPARTMENT, UNIVERSITY } from '../ui/IdentityBar'
import CampusBackdrop from './CampusBackdrop'
import AppAuthScreen from './AppAuthScreen'
import { useCampusSlideshow } from '../../lib/campus'
import { isNative } from '../../lib/native'

const TONES = {
  brand: {
    overlay: 'bg-linear-to-b from-deep-950/55 via-deep-950/20 to-deep-950/70',
    card: 'glass-auth',
  },
  dark: {
    overlay: 'bg-linear-to-b from-black/70 via-deep-950/55 to-black/80',
    card: 'glass-auth',
  },
}

export default function AuthScreen({
  images,
  rotateAfter,
  overlay,
  tone = 'brand',
  note,
  wide = false,
  className,
  children,
}) {
  const onWeb = !isNative()
  const slideshow = useCampusSlideshow(images, rotateAfter)
  if (!onWeb) {
    return (
      <AppAuthScreen wide={wide} className={className}>
        {children}
      </AppAuthScreen>
    )
  }
  const skin = TONES[tone] || TONES.brand
  const slides = images || []
  return (
    <div className="relative flex min-h-dvh flex-col">
      <CampusBackdrop
        images={images}
        index={slideshow.index}
        still={slideshow.still}
        overlay={overlay || skin.overlay}
        className="fixed"
      />
      <div className="relative flex min-h-dvh flex-col px-4 pt-[max(1rem,env(safe-area-inset-top,0px))] pb-[max(1rem,env(safe-area-inset-bottom,0px))] sm:px-6">
        <Link to={paths.root()} className="flex items-center gap-2.5 self-start">
          <LogoBadge size="sm" tone="light" />
          <span className="min-w-0">
            <span className="font-display block truncate text-sm font-semibold text-white">
              {UNIVERSITY}
            </span>
            <span className="block truncate text-2xs text-white/70">{DEPARTMENT}</span>
          </span>
        </Link>
        <div className="flex flex-1 items-center justify-center py-6">
          <div
            className={cx(
              'animate-fade-in-up w-full rounded-[1.5rem] p-6 sm:p-7',
              skin.card,
              wide ? 'max-w-2xl' : 'max-w-sm',
              className,
            )}
          >
            {children}
          </div>
        </div>
        {slides.length > 1 && (
          <div className="mx-auto flex max-w-md flex-col items-center gap-2 pb-2">
            <div className="flex gap-2">
              {slides.map((source, position) => (
                <button
                  key={source}
                  type="button"
                  aria-label={`Show campus photo ${position + 1}`}
                  aria-current={position === slideshow.index}
                  onClick={() => slideshow.show(position)}
                  className={cx(
                    'h-1.5 rounded-full transition-all duration-500 motion-reduce:transition-none',
                    position === slideshow.index ? 'w-7 bg-white/85' : 'w-2.5 bg-white/35 hover:bg-white/60',
                  )}
                />
              ))}
            </div>
            <p className="text-center text-2xs text-white/65">{slideshow.caption}</p>
          </div>
        )}
        {note && (
          <p className="mx-auto max-w-md text-center text-xs leading-relaxed text-white/80">
            {note}
          </p>
        )}
      </div>
    </div>
  )
}

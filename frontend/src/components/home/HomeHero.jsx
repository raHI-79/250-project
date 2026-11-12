import { useMemo } from 'react'
import CampusBackdrop from '../layout/CampusBackdrop'
import { HOME_SLIDE_AFTER, useCampusSlideshow } from '../../lib/campus'
import cx from '../../lib/cx'
import { isSafeImage, isSafeLink, withDefaults } from '../../lib/siteContent'
import { SafeLink } from '../site/RichText'
import AndroidDownload from './AndroidDownload'

const BUTTONS = {
  light: 'bg-white text-brand-700 hover:bg-white/90',
  outline: 'border border-white/70 text-white hover:bg-white/10',
}

function HeroButton({ link, tone }) {
  if (!link?.label || !isSafeLink(link.url)) return null
  return (
    <SafeLink
      to={link.url}
      className={cx(
        'press inline-flex min-h-12 items-center rounded-xl px-5 text-sm font-semibold',
        BUTTONS[tone],
      )}
    >
      {link.label}
    </SafeLink>
  )
}

export default function HomeHero({ content }) {
  const home = withDefaults(content).home
  const slides = useMemo(
    () => home.slides.filter((slide) => isSafeImage(slide.image_url)),
    [home.slides],
  )
  const images = useMemo(() => slides.map((slide) => slide.image_url), [slides])
  const slideshow = useCampusSlideshow(images, HOME_SLIDE_AFTER)
  const current = slides[Math.min(slideshow.index, Math.max(slides.length - 1, 0))]
  return (
    <section className="relative isolate overflow-hidden bg-deep-950">
      <CampusBackdrop
        images={images}
        index={slideshow.index}
        still={slideshow.still}
        overlay="bg-linear-to-r from-deep-950/85 via-deep-950/55 to-deep-900/25"
      />
      <div className="relative mx-auto flex min-h-[26rem] max-w-6xl flex-col justify-center px-4 py-14 sm:px-6 sm:py-20 lg:min-h-[32.5rem]">
        {home.hero_eyebrow && (
          <p className="text-xs font-semibold tracking-[0.14em] text-sand uppercase">
            {home.hero_eyebrow}
          </p>
        )}
        <h1
          data-hero-title
          className="font-display mt-4 max-w-2xl wrap-long text-3xl leading-tight font-semibold text-white sm:text-4xl lg:text-5xl"
        >
          {home.hero_title}
        </h1>
        {home.hero_text && (
          <p className="mt-4 max-w-xl wrap-long text-sm leading-relaxed text-white/80 sm:text-base">
            {home.hero_text}
          </p>
        )}
        <div className="mt-7 flex flex-wrap gap-3">
          <HeroButton link={home.primary_cta} tone="light" />
          <HeroButton link={home.secondary_cta} tone="outline" />
          <AndroidDownload variant="outlineLight" />
        </div>
        <div className="mt-10 flex flex-wrap items-center gap-3">
          <div className="flex gap-2">
            {images.map((source, position) => (
              <button
                key={source}
                type="button"
                aria-label={`Show campus photo ${position + 1}`}
                aria-current={position === slideshow.index}
                data-compact=""
                onClick={() => slideshow.show(position)}
                className={cx(
                  'h-1.5 rounded-full transition-all duration-500 motion-reduce:transition-none',
                  position === slideshow.index
                    ? 'w-7 bg-white/85'
                    : 'w-2.5 bg-white/35 hover:bg-white/60',
                )}
              />
            ))}
          </div>
          <p className="text-xs text-white/55">{current?.caption || ''}</p>
        </div>
      </div>
    </section>
  )
}

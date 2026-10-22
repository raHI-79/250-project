import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { HERO_IMAGES } from '../../lib/campus'
import cx from '../../lib/cx'
import paths from '../../lib/paths'
import { Button } from '../ui'

const ROTATE_AFTER = 7000

const CAPTIONS = [
  'SUST Iconic Old Gate, Kumargaon',
  'IICT Building, SUST campus',
  'University main entrance, Kumargaon',
  'Shaheed Minar, SUST campus',
]

export default function HomeHero() {
  const [index, setIndex] = useState(0)
  const [broken, setBroken] = useState({})
  useEffect(() => {
    const timer = window.setInterval(() => {
      setIndex((current) => (current + 1) % HERO_IMAGES.length)
    }, ROTATE_AFTER)
    return () => window.clearInterval(timer)
  }, [])
  return (
    <section className="relative isolate overflow-hidden bg-linear-to-br from-brand-900 via-brand-700 to-brand-500">
      {HERO_IMAGES.map((source, position) =>
        broken[source] ? null : (
          <img
            key={source}
            src={source}
            alt=""
            aria-hidden="true"
            onError={() => setBroken((current) => ({ ...current, [source]: true }))}
            className={cx(
              'absolute inset-0 size-full object-cover transition-opacity duration-1000',
              position === index ? 'animate-ken-burns opacity-100' : 'opacity-0',
            )}
          />
        ),
      )}
      <div className="absolute inset-0 bg-linear-to-r from-brand-950/92 via-brand-900/75 to-brand-700/35" />
      <div className="relative mx-auto flex min-h-[26rem] max-w-6xl flex-col justify-center px-4 py-14 sm:px-6 sm:py-20 lg:min-h-[32.5rem]">
        <p className="text-xs font-semibold tracking-[0.14em] text-gold-200 uppercase">
          Established 1991 · Sylhet, Bangladesh
        </p>
        <h1 className="font-display mt-4 max-w-2xl text-3xl leading-tight font-semibold text-white sm:text-4xl lg:text-5xl">
          Class and exam coordination for the <em className="text-gold-200 not-italic">CSE</em>{' '}
          department
        </h1>
        <p className="mt-4 max-w-xl text-sm leading-relaxed text-white/80 sm:text-base">
          One record for the weekly routine, rescheduled classes, exams and notices. Class
          representatives draft the plan, teachers confirm it, and every student reads the published
          result.
        </p>
        <div className="mt-7 flex flex-wrap gap-3">
          <Button as={Link} to={paths.studentLogin()} size="lg" variant="light">
            Student sign in
          </Button>
          <Button as={Link} to={paths.teacherLogin()} size="lg" variant="outlineLight">
            Teacher sign in
          </Button>
        </div>
        <div className="mt-10 flex items-center gap-3">
          <div className="flex gap-2">
            {HERO_IMAGES.map((source, position) => (
              <button
                key={source}
                type="button"
                aria-label={`Show campus photo ${position + 1}`}
                aria-current={position === index}
                onClick={() => setIndex(position)}
                className={cx(
                  'h-1.5 rounded-full transition-all',
                  position === index ? 'w-7 bg-gold-200' : 'w-3 bg-white/40 hover:bg-white/70',
                )}
              />
            ))}
          </div>
          <p className="text-xs text-white/60">{CAPTIONS[index]}</p>
        </div>
      </div>
    </section>
  )
}

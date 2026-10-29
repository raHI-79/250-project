import { Search } from 'lucide-react'
import LogoBadge from '../ui/LogoBadge'
import cx from '../../lib/cx'
import { HERO_BUILDING } from '../../lib/campus'
import { useChrome } from '../../context/chrome'

const greeting = (hour) => {
  if (hour < 12) return 'Good morning'
  if (hour < 17) return 'Good afternoon'
  return 'Good evening'
}

const dateLine = (date) =>
  date.toLocaleDateString('en-GB', { weekday: 'long', day: 'numeric', month: 'long' })

export default function TodayHero({ name, meta, now = new Date(), className }) {
  const chrome = useChrome()
  const first = String(name || '')
    .trim()
    .split(/\s+/)[0]
  return (
    <section
      className={cx(
        'relative -mx-4 overflow-hidden px-4 pt-5 pb-6 sm:mx-0 sm:rounded-card sm:px-6',
        className,
      )}
    >
      <div
        aria-hidden="true"
        style={{ backgroundImage: `url(${HERO_BUILDING[0]})` }}
        className="absolute inset-0 bg-cover bg-center"
      />
      <div
        aria-hidden="true"
        className="absolute inset-0 bg-linear-to-t from-deep-950/55 via-deep-950/30 to-deep-950/15"
      />
      <div className="relative text-shadow-hero">
        <div className="flex items-center gap-2.5">
          <LogoBadge size="sm" tone="light" />
          <p className="text-2xs font-bold tracking-[0.14em] text-white/80 uppercase">SUST CSE</p>
        </div>
        <h1 className="font-display mt-4 text-title leading-tight font-bold text-white">
          {greeting(now.getHours())}
          {first ? <>,&nbsp;{first}</> : null}
        </h1>
        <p className="mt-1.5 text-sm text-white/85">
          {dateLine(now)}
          {meta ? <span className="text-white/60"> · {meta}</span> : null}
        </p>
        {chrome?.openSearch && (
          <button
            type="button"
            onClick={chrome.openSearch}
            className="mt-5 flex h-12 w-full items-center gap-2.5 rounded-xl bg-white/15 px-3.5 text-left text-sm text-white/80 backdrop-blur-sm transition-colors hover:bg-white/25"
          >
            <Search size={18} className="shrink-0" />
            Search courses, people, notices
          </button>
        )}
      </div>
    </section>
  )
}

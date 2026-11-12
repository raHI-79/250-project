import { Search } from 'lucide-react'
import Avatar from '../ui/Avatar'
import HeaderArt from '../ui/HeaderArt'
import cx from '../../lib/cx'
import { useChrome } from '../../context/chrome'
import { hourIn, longDate } from '../../lib/time'

const greeting = (hour) => {
  if (hour < 12) return 'Good morning'
  if (hour < 17) return 'Good afternoon'
  return 'Good evening'
}

const dateLine = (date) => longDate(date)

export default function TodayHero({ name, regNo, photo, meta, now = new Date(), className }) {
  const chrome = useChrome()
  const first = String(name || '')
    .trim()
    .split(/\s+/)[0]
  return (
    <section
      className={cx(
        'relative isolate -mx-4 -mt-5 rounded-b-sheet bg-header px-4 pt-5 pb-14 sm:-mx-6 sm:px-6 lg:-mx-8 lg:px-8',
        className,
      )}
    >
      <HeaderArt variant="hero" />
      <div className="flex items-center gap-3">
        <Avatar src={photo} name={name} size="lg" className="ring-2 ring-white/40" />
        <div className="min-w-0 flex-1">
          <p className="text-2xs font-extrabold tracking-[0.14em] text-white/95 uppercase">
            SUST CSE
          </p>
          <p className="font-display mt-0.5 wrap-long text-section leading-tight text-white">
            {name}
          </p>
          {regNo && (
            <p className="wrap-long text-xs font-semibold text-white/95 tabular-nums">{regNo}</p>
          )}
        </div>
      </div>
      <h1 className="font-display mt-4 text-title leading-tight text-white">
        {greeting(hourIn(now))}
        {first ? <>,&nbsp;{first}</> : null}
      </h1>
      <p className="mt-1 wrap-long text-sm text-white/95">
        {dateLine(now)}
        {meta ? <span className="text-white/95"> · {meta}</span> : null}
      </p>
      {chrome?.openSearch && (
        <button
          type="button"
          onClick={chrome.openSearch}
          className="press mt-4 flex h-12 w-full items-center gap-2.5 rounded-input bg-[rgb(4_40_66/0.3)] px-3.5 text-left text-sm font-semibold text-white ring-1 ring-white/25 transition-colors ring-inset hover:bg-[rgb(4_40_66/0.42)]"
        >
          <Search size={18} className="shrink-0" />
          Search courses, people, notices
        </button>
      )}
    </section>
  )
}
